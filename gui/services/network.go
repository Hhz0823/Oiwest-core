package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type InboundRule struct {
	ID       string `json:"id"`
	Tag      string `json:"tag"`
	Port     int    `json:"port"`
	Listen   string `json:"listen"`
	Protocol string `json:"protocol"`
	Enabled  bool   `json:"enabled"`
	Settings InboundSettings `json:"settings"`
}

type InboundSettings struct {
	Auth    string `json:"auth,omitempty"`
	UDP     bool   `json:"udp"`
	User    string `json:"user,omitempty"`
	Pass    string `json:"pass,omitempty"`
	Method  string `json:"method,omitempty"`
	Password string `json:"password,omitempty"`
}

type RoutingRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Domain      []string `json:"domain"`
	IP          []string `json:"ip"`
	Port        string   `json:"port"`
	Network     string   `json:"network"`
	Protocol    []string `json:"protocol"`
	InboundTag  []string `json:"inboundTag"`
	OutboundTag string   `json:"outboundTag"`
	Enabled     bool     `json:"enabled"`
	Sort        int      `json:"sort"`
}

type DNSConfig struct {
	Servers         []DNSServerItem `json:"servers"`
	Hosts           map[string]string `json:"hosts,omitempty"`
	ClientIP        string           `json:"clientIp,omitempty"`
	Tag             string           `json:"tag,omitempty"`
	QueryStrategy   string           `json:"queryStrategy"`
	DisableCache    bool             `json:"disableCache"`
	DisableFallback bool             `json:"disableFallback"`
}

type DNSServerItem struct {
	Address     string   `json:"address"`
	Port        int      `json:"port"`
	Domains     []string `json:"domains,omitempty"`
	ExpectIPs   []string `json:"expectIPs,omitempty"`
	SkipFallback bool    `json:"skipFallback"`
}

type TransparentProxyConfig struct {
	Enabled     bool   `json:"enabled"`
	Mode        string `json:"mode"`
	RedirectTCP int    `json:"redirectTcp"`
	RedirectUDP int    `json:"redirectUdp"`
	ByPassLAN   bool   `json:"bypassLan"`
}

type NetworkConfigManager struct {
	mu               sync.RWMutex
	inbounds         []InboundRule
	routingRules     []RoutingRule
	dnsConfig        DNSConfig
	transparentProxy TransparentProxyConfig
	dataPath         string
}

var networkConfigMgr *NetworkConfigManager

func GetNetworkConfigManager() *NetworkConfigManager {
	if networkConfigMgr == nil {
		networkConfigMgr = &NetworkConfigManager{
			inbounds: []InboundRule{
				{ID: "socks-in", Tag: "socks-in", Port: 10808, Listen: "127.0.0.1", Protocol: "socks", Enabled: true, Settings: InboundSettings{Auth: "noauth", UDP: true}},
				{ID: "http-in", Tag: "http-in", Port: 10809, Listen: "127.0.0.1", Protocol: "http", Enabled: true},
			},
			routingRules: []RoutingRule{
				{ID: "private", Name: "私有地址直连", Type: "field", IP: []string{"geoip:private"}, OutboundTag: "direct", Enabled: true, Sort: 1},
				{ID: "cn", Name: "国内直连", Type: "field", IP: []string{"geoip:cn"}, OutboundTag: "direct", Enabled: true, Sort: 2},
			},
			dnsConfig: DNSConfig{
				Servers:       []DNSServerItem{{Address: "localhost", Port: 53}},
				QueryStrategy: "UseIP",
			},
			transparentProxy: TransparentProxyConfig{Enabled: false, RedirectTCP: 12345, RedirectUDP: 12346, ByPassLAN: true},
		}
	}
	return networkConfigMgr
}

func (ncm *NetworkConfigManager) SetDataPath(path string) {
	ncm.dataPath = path
	ncm.loadFromDisk()
}

func (ncm *NetworkConfigManager) loadFromDisk() {
	p := ncm.configPath()
	data, err := os.ReadFile(p)
	if err != nil {
		return
	}
	var saved struct {
		Inbounds         []InboundRule           `json:"inbounds"`
		RoutingRules     []RoutingRule           `json:"routingRules"`
		DNS              DNSConfig               `json:"dns"`
		TransparentProxy TransparentProxyConfig  `json:"transparentProxy"`
	}
	if json.Unmarshal(data, &saved) == nil {
		ncm.mu.Lock()
		if len(saved.Inbounds) > 0 {
			ncm.inbounds = saved.Inbounds
		}
		if len(saved.RoutingRules) > 0 {
			ncm.routingRules = saved.RoutingRules
		}
		ncm.dnsConfig = saved.DNS
		ncm.transparentProxy = saved.TransparentProxy
		ncm.mu.Unlock()
	}
}

func (ncm *NetworkConfigManager) configPath() string {
	return filepath.Join(ncm.dataPath, "network.json")
}

func (ncm *NetworkConfigManager) save() {
	data, _ := json.MarshalIndent(map[string]interface{}{
		"inbounds":         ncm.inbounds,
		"routingRules":     ncm.routingRules,
		"dns":              ncm.dnsConfig,
		"transparentProxy": ncm.transparentProxy,
	}, "", "  ")
	if ncm.dataPath != "" {
		os.WriteFile(ncm.configPath(), data, 0644)
	}
}

func (ncm *NetworkConfigManager) GetInbounds() []InboundRule {
	ncm.mu.RLock()
	defer ncm.mu.RUnlock()
	cp := make([]InboundRule, len(ncm.inbounds))
	copy(cp, ncm.inbounds)
	return cp
}

func (ncm *NetworkConfigManager) AddInbound(rule InboundRule) error {
	ncm.mu.Lock()
	defer ncm.mu.Unlock()
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("inbound_%d", len(ncm.inbounds))
	}
	ncm.inbounds = append(ncm.inbounds, rule)
	ncm.save()
	return nil
}

func (ncm *NetworkConfigManager) UpdateInbound(rule InboundRule) error {
	ncm.mu.Lock()
	defer ncm.mu.Unlock()
	for i, r := range ncm.inbounds {
		if r.ID == rule.ID {
			ncm.inbounds[i] = rule
			ncm.save()
			return nil
		}
	}
	return fmt.Errorf("inbound %s not found", rule.ID)
}

func (ncm *NetworkConfigManager) DeleteInbound(id string) error {
	ncm.mu.Lock()
	defer ncm.mu.Unlock()
	for i, r := range ncm.inbounds {
		if r.ID == id {
			ncm.inbounds = append(ncm.inbounds[:i], ncm.inbounds[i+1:]...)
			ncm.save()
			return nil
		}
	}
	return fmt.Errorf("inbound %s not found", id)
}

func (ncm *NetworkConfigManager) ToggleInbound(id string, enabled bool) error {
	ncm.mu.Lock()
	defer ncm.mu.Unlock()
	for i, r := range ncm.inbounds {
		if r.ID == id {
			ncm.inbounds[i].Enabled = enabled
			ncm.save()
			return nil
		}
	}
	return fmt.Errorf("inbound %s not found", id)
}

func (ncm *NetworkConfigManager) GetRoutingRules() []RoutingRule {
	ncm.mu.RLock()
	defer ncm.mu.RUnlock()
	cp := make([]RoutingRule, len(ncm.routingRules))
	copy(cp, ncm.routingRules)
	return cp
}

func (ncm *NetworkConfigManager) AddRoutingRule(rule RoutingRule) error {
	ncm.mu.Lock()
	defer ncm.mu.Unlock()
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("route_%d", len(ncm.routingRules))
	}
	if rule.Sort == 0 {
		rule.Sort = len(ncm.routingRules) + 1
	}
	ncm.routingRules = append(ncm.routingRules, rule)
	ncm.save()
	return nil
}

func (ncm *NetworkConfigManager) UpdateRoutingRule(rule RoutingRule) error {
	ncm.mu.Lock()
	defer ncm.mu.Unlock()
	for i, r := range ncm.routingRules {
		if r.ID == rule.ID {
			ncm.routingRules[i] = rule
			ncm.save()
			return nil
		}
	}
	return fmt.Errorf("routing rule %s not found", rule.ID)
}

func (ncm *NetworkConfigManager) DeleteRoutingRule(id string) error {
	ncm.mu.Lock()
	defer ncm.mu.Unlock()
	for i, r := range ncm.routingRules {
		if r.ID == id {
			ncm.routingRules = append(ncm.routingRules[:i], ncm.routingRules[i+1:]...)
			ncm.save()
			return nil
		}
	}
	return fmt.Errorf("routing rule %s not found", id)
}

func (ncm *NetworkConfigManager) ReorderRoutingRules(orderedIDs []string) error {
	ncm.mu.Lock()
	defer ncm.mu.Unlock()

	idMap := make(map[string]*RoutingRule)
	for _, r := range ncm.routingRules {
		idMap[r.ID] = &r
	}

	reordered := make([]RoutingRule, 0, len(ncm.routingRules))
	for i, id := range orderedIDs {
		if r, ok := idMap[id]; ok {
			cp := *r
			cp.Sort = i + 1
			reordered = append(reordered, cp)
			delete(idMap, id)
		}
	}
	for _, r := range idMap {
		cp := *r
		cp.Sort = len(reordered) + 1
		reordered = append(reordered, cp)
	}
	ncm.routingRules = reordered
	ncm.save()
	return nil
}

func (ncm *NetworkConfigManager) GetDNSConfig() DNSConfig {
	ncm.mu.RLock()
	defer ncm.mu.RUnlock()
	return ncm.dnsConfig
}

func (ncm *NetworkConfigManager) SetDNSConfig(cfg DNSConfig) error {
	ncm.mu.Lock()
	ncm.dnsConfig = cfg
	ncm.mu.Unlock()
	ncm.save()
	return nil
}

func (ncm *NetworkConfigManager) AddDNSServer(server DNSServerItem) error {
	ncm.mu.Lock()
	ncm.dnsConfig.Servers = append(ncm.dnsConfig.Servers, server)
	ncm.mu.Unlock()
	ncm.save()
	return nil
}

func (ncm *NetworkConfigManager) RemoveDNSServer(index int) error {
	ncm.mu.Lock()
	defer ncm.mu.Unlock()
	if index < 0 || index >= len(ncm.dnsConfig.Servers) {
		return fmt.Errorf("invalid DNS server index")
	}
	ncm.dnsConfig.Servers = append(ncm.dnsConfig.Servers[:index], ncm.dnsConfig.Servers[index+1:]...)
	ncm.save()
	return nil
}

func (ncm *NetworkConfigManager) GetTransparentProxyConfig() TransparentProxyConfig {
	ncm.mu.RLock()
	defer ncm.mu.RUnlock()
	return ncm.transparentProxy
}

func (ncm *NetworkConfigManager) SetTransparentProxyConfig(cfg TransparentProxyConfig) error {
	ncm.mu.Lock()
	ncm.transparentProxy = cfg
	ncm.mu.Unlock()
	ncm.save()
	if cfg.Enabled {
		ncm.addLog("[信息] 透明代理已启用")
	} else {
		ncm.addLog("[信息] 透明代理已禁用")
	}
	return nil
}

func (ncm *NetworkConfigManager) addLog(msg string) {
	GetCoreManager().addLog(msg)
}

func (ncm *NetworkConfigManager) BuildNetworkConfigJSON() string {
	ncm.mu.RLock()
	defer ncm.mu.RUnlock()

	inboundsJSON := make([]interface{}, 0)
	for _, r := range ncm.inbounds {
		if !r.Enabled {
			continue
		}
		item := map[string]interface{}{
			"tag":      r.Tag,
			"port":     r.Port,
			"listen":   r.Listen,
			"protocol": r.Protocol,
			"settings": map[string]interface{}{
				"auth": r.Settings.Auth,
				"udp":  r.Settings.UDP,
			},
			"sniffing": map[string]interface{}{
				"enabled":      true,
				"destOverride": []string{"http", "tls"},
			},
		}
		inboundsJSON = append(inboundsJSON, item)
	}

	routingJSON := make([]interface{}, 0)
	for _, r := range ncm.routingRules {
		if !r.Enabled {
			continue
		}
		item := map[string]interface{}{
			"type":        r.Type,
			"outboundTag": r.OutboundTag,
		}
		if len(r.Domain) > 0 {
			item["domain"] = r.Domain
		}
		if len(r.IP) > 0 {
			item["ip"] = r.IP
		}
		if r.Port != "" {
			item["port"] = r.Port
		}
		if r.Network != "" {
			item["network"] = r.Network
		}
		if len(r.Protocol) > 0 {
			item["protocol"] = r.Protocol
		}
		if len(r.InboundTag) > 0 {
			item["inboundTag"] = r.InboundTag
		}
		routingJSON = append(routingJSON, item)
	}

	dnsServersJSON := make([]interface{}, 0)
	for _, s := range ncm.dnsConfig.Servers {
		item := map[string]interface{}{
			"address": s.Address,
		}
		if s.Port > 0 {
			item["port"] = s.Port
		}
		if len(s.Domains) > 0 {
			item["domains"] = s.Domains
		}
		dnsServersJSON = append(dnsServersJSON, item)
	}

	config := map[string]interface{}{
		"inbounds": inboundsJSON,
		"routing": map[string]interface{}{
			"domainStrategy": ncm.dnsConfig.QueryStrategy,
			"rules":          routingJSON,
		},
		"dns": map[string]interface{}{
			"servers":         dnsServersJSON,
			"queryStrategy":   ncm.dnsConfig.QueryStrategy,
			"disableCache":    ncm.dnsConfig.DisableCache,
			"disableFallback": ncm.dnsConfig.DisableFallback,
		},
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	return string(data)
}
