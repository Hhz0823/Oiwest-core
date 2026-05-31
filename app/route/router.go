package router

import (
	"context"
	"errors"
	"net"
	"regexp"
	"strings"
	"sync"

	"github.com/Hhz0823/oiwest-core/app/config"
)

var (
	ErrNoRoute = errors.New("router: no matching route")
)

type Router struct {
	config *config.RoutingConfig
	rules  []*CompiledRule
	mu     sync.RWMutex
}

type CompiledRule struct {
	Raw         *config.RoutingRule
	domainRegex []*regexp.Regexp
	ipNets      []*net.IPNet
	outboundTag string
	balancerTag string
}

func NewRouter(cfg *config.RoutingConfig) *Router {
	r := &Router{
		config: cfg,
	}
	r.compileRules()
	return r
}

func (r *Router) compileRules() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.rules = nil
	for _, rule := range r.config.Rules {
		cr := &CompiledRule{
			Raw:         &rule,
			outboundTag: rule.OutboundTag,
			balancerTag: rule.BalancerTag,
		}

		for _, domain := range rule.Domain {
			if strings.HasPrefix(domain, "regexp:") {
				re, err := regexp.Compile(domain[7:])
				if err == nil {
					cr.domainRegex = append(cr.domainRegex, re)
				}
			} else {
				pattern := convertDomainToRegex(domain)
				re, err := regexp.Compile(pattern)
				if err == nil {
					cr.domainRegex = append(cr.domainRegex, re)
				}
			}
		}

		for _, ip := range rule.IP {
			if strings.Contains(ip, "/") {
				_, ipNet, err := net.ParseCIDR(ip)
				if err == nil {
					cr.ipNets = append(cr.ipNets, ipNet)
				}
			} else {
				parsedIP := net.ParseIP(ip)
				if parsedIP != nil {
					mask := net.CIDRMask(128, 128)
					if parsedIP.To4() != nil {
						mask = net.CIDRMask(32, 32)
					}
					cr.ipNets = append(cr.ipNets, &net.IPNet{
						IP:   parsedIP,
						Mask: mask,
					})
				}
			}
		}

		r.rules = append(r.rules, cr)
	}
}

func convertDomainToRegex(domain string) string {
	if strings.HasPrefix(domain, "domain:") {
		domain = domain[7:]
	}
	if strings.HasPrefix(domain, "full:") {
		return "^" + regexp.QuoteMeta(domain[5:]) + "$"
	}
	if strings.HasPrefix(domain, "keyword:") {
		return regexp.QuoteMeta(domain[8:])
	}
	escaped := regexp.QuoteMeta(domain)
	escaped = strings.ReplaceAll(escaped, `\*`, ".*")
	return "^" + escaped + "$"
}

type RoutingContext struct {
	InboundTag string
	Network    string
	SourceIP   net.IP
	TargetIP   net.IP
	TargetDomain string
	Port       uint16
	SourcePort uint16
	Protocol   string
	User       string
	Attributes map[string]string
}

func (r *Router) Route(ctx context.Context, rc *RoutingContext) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, rule := range r.rules {
		if r.matchRule(rule, rc) {
			if rule.outboundTag != "" {
				return rule.outboundTag, nil
			}
			if rule.balancerTag != "" {
				return r.selectBalancer(rule.balancerTag), nil
			}
		}
	}

	for _, balancer := range r.config.Balancers {
		return r.selectBalancer(balancer.Tag), nil
	}

	return "", ErrNoRoute
}

func (r *Router) matchRule(rule *CompiledRule, rc *RoutingContext) bool {
	if len(rule.Raw.InboundTag) > 0 {
		if !containsString(rule.Raw.InboundTag, rc.InboundTag) {
			return false
		}
	}

	if rule.Raw.Network != "" && rule.Raw.Network != rc.Network {
		return false
	}

	if len(rule.Raw.Protocol) > 0 {
		if !containsString(rule.Raw.Protocol, rc.Protocol) {
			return false
		}
	}

	if len(rule.Raw.User) > 0 {
		if !containsString(rule.Raw.User, rc.User) {
			return false
		}
	}

	if rule.Raw.Port != "" {
		if !matchPort(rule.Raw.Port, rc.Port) {
			return false
		}
	}

	if rule.Raw.SourcePort != "" {
		if !matchPort(rule.Raw.SourcePort, rc.SourcePort) {
			return false
		}
	}

	if len(rule.Raw.Source) > 0 && rc.SourceIP != nil {
		matched := false
		for _, srcIP := range rule.Raw.Source {
			if matchIP(rc.SourceIP, srcIP) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	domainMatched := false
	if len(rule.Raw.Domain) > 0 && rc.TargetDomain != "" {
		for _, re := range rule.domainRegex {
			if re.MatchString(rc.TargetDomain) {
				domainMatched = true
				break
			}
		}
		if !domainMatched {
			return false
		}
	}

	ipMatched := false
	if len(rule.Raw.IP) > 0 && rc.TargetIP != nil {
		for _, ipNet := range rule.ipNets {
			if ipNet.Contains(rc.TargetIP) {
				ipMatched = true
				break
			}
		}
		if !ipMatched {
			return false
		}
	}

	if len(rule.Raw.Domain) > 0 || len(rule.Raw.IP) > 0 {
		if rc.TargetDomain != "" && domainMatched {
			return true
		}
		if rc.TargetIP != nil && ipMatched {
			return true
		}
		if !domainMatched && !ipMatched {
			return false
		}
	}

	if len(rule.Raw.Domain) == 0 && len(rule.Raw.IP) == 0 &&
		len(rule.Raw.InboundTag) == 0 && rule.Raw.Network == "" &&
		len(rule.Raw.Protocol) == 0 && rule.Raw.Port == "" &&
		len(rule.Raw.User) == 0 {
		return true
	}

	return true
}

func (r *Router) selectBalancer(tag string) string {
	for _, balancer := range r.config.Balancers {
		if balancer.Tag == tag && len(balancer.Selector) > 0 {
			return balancer.Selector[0]
		}
	}
	return "direct"
}

func matchPort(rule string, port uint16) bool {
	portStr := strings.TrimSpace(rule)
	if portStr == "" {
		return true
	}
	parts := strings.Split(portStr, "-")
	if len(parts) == 2 {
		from := parsePort(parts[0])
		to := parsePort(parts[1])
		return port >= from && port <= to
	}
	return port == parsePort(portStr)
}

func parsePort(s string) uint16 {
	var port uint16
	for _, c := range s {
		if c >= '0' && c <= '9' {
			port = port*10 + uint16(c-'0')
		}
	}
	return port
}

func matchIP(ip net.IP, pattern string) bool {
	if strings.Contains(pattern, "/") {
		_, ipNet, err := net.ParseCIDR(pattern)
		if err != nil {
			return false
		}
		return ipNet.Contains(ip)
	}
	parsedIP := net.ParseIP(pattern)
	if parsedIP == nil {
		return false
	}
	return ip.Equal(parsedIP)
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func (r *Router) ReloadConfig(cfg *config.RoutingConfig) {
	r.mu.Lock()
	r.config = cfg
	r.mu.Unlock()
	r.compileRules()
}
