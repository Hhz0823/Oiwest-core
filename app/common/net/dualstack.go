package net

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"sync"
	
	"time"
)

type IPPreference int

const (
	PreferIPv4 IPPreference = 0
	PreferIPv6 IPPreference = 1
	PreferDual IPPreference = 2
	IPv4Only   IPPreference = 3
	IPv6Only   IPPreference = 4
)

type DualStackConfig struct {
	Preference     IPPreference `json:"preference"`
	IPv4Bind       string       `json:"ipv4Bind"`
	IPv6Bind       string       `json:"ipv6Bind"`
	MultiLine      bool         `json:"multiLine"`
	Failover       bool         `json:"failover"`
	FailoverTimeout time.Duration `json:"failoverTimeout"`
	ProbeInterval  time.Duration `json:"probeInterval"`
	ProbeTimeout   time.Duration `json:"probeTimeout"`
	DualStackMode  string       `json:"dualStackMode"`
	Strategy       string       `json:"strategy"`
}

func DefaultDualStackConfig() *DualStackConfig {
	return &DualStackConfig{
		Preference:      PreferDual,
		MultiLine:       true,
		Failover:        true,
		FailoverTimeout: 5 * time.Second,
		ProbeInterval:   30 * time.Second,
		ProbeTimeout:    2 * time.Second,
		DualStackMode:   "prefer_ipv4",
		Strategy:        "latency",
	}
}

type DualStackDialer struct {
	config      *DualStackConfig
	ipv4Pool    *MultiLinePool
	ipv6Pool    *MultiLinePool
	mu          sync.RWMutex
}

type MultiLinePool struct {
	addresses []MultiLineAddr
	stats     map[string]*LineStats
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

type MultiLineAddr struct {
	IP      net.IP
	Port    int
	Weight  int
	Latency time.Duration
	Alive   bool
	Tag     string
}

type LineStats struct {
	SuccessCount    int64
	FailureCount    int64
	TotalLatency    time.Duration
	LastLatency     time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	LastChecked     time.Time
	Bandwidth       float64
	ConsecutiveFails int64
	mu              sync.Mutex
}

func NewDualStackDialer(config *DualStackConfig) *DualStackDialer {
	ctx, cancel := context.WithCancel(context.Background())
	dialer := &DualStackDialer{
		config: config,
		ipv4Pool: &MultiLinePool{
			stats:  make(map[string]*LineStats),
			ctx:    ctx,
			cancel: cancel,
		},
		ipv6Pool: &MultiLinePool{
			stats:  make(map[string]*LineStats),
			ctx:    ctx,
			cancel: cancel,
		},
	}
	return dialer
}

func (d *DualStackDialer) AddAddress(ip net.IP, port int, tag string, weight int) {
	addr := MultiLineAddr{
		IP:     ip,
		Port:   port,
		Weight: weight,
		Alive:  true,
		Tag:    tag,
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if ip.To4() != nil {
		d.ipv4Pool.addresses = append(d.ipv4Pool.addresses, addr)
		d.ipv4Pool.stats[tag] = &LineStats{
			MinLatency: time.Hour,
			MaxLatency: 0,
		}
	} else {
		d.ipv6Pool.addresses = append(d.ipv6Pool.addresses, addr)
		d.ipv6Pool.stats[tag] = &LineStats{
			MinLatency: time.Hour,
			MaxLatency: 0,
		}
	}
}

func (d *DualStackDialer) RemoveAddress(tag string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ipv4Pool.removeByTag(tag)
	d.ipv6Pool.removeByTag(tag)
	delete(d.ipv4Pool.stats, tag)
	delete(d.ipv6Pool.stats, tag)
}

func (p *MultiLinePool) removeByTag(tag string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, addr := range p.addresses {
		if addr.Tag == tag {
			p.addresses = append(p.addresses[:i], p.addresses[i+1:]...)
			return
		}
	}
}

func (d *DualStackDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	port := 0
	fmt.Sscanf(portStr, "%d", &port)

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}

	var ipAddrs []net.IP
	for _, ipAddr := range ips {
		ipAddrs = append(ipAddrs, ipAddr.IP)
	}

	return d.DialIPContext(ctx, network, ipAddrs, port)
}

func (d *DualStackDialer) DialIPContext(ctx context.Context, network string, ips []net.IP, port int) (net.Conn, error) {
	var ipv4Addrs, ipv6Addrs []net.IP
	for _, ip := range ips {
		if ip.To4() != nil {
			ipv4Addrs = append(ipv4Addrs, ip)
		} else {
			ipv6Addrs = append(ipv6Addrs, ip)
		}
	}

	switch d.config.Preference {
	case IPv4Only:
		return d.dialFirst(ctx, network, ipv4Addrs, port)
	case IPv6Only:
		return d.dialFirst(ctx, network, ipv6Addrs, port)
	case PreferIPv4:
		if len(ipv4Addrs) > 0 {
			conn, err := d.dialFirst(ctx, network, ipv4Addrs, port)
			if err == nil || !d.config.Failover {
				return conn, err
			}
		}
		return d.dialFirst(ctx, network, ipv6Addrs, port)
	case PreferIPv6:
		if len(ipv6Addrs) > 0 {
			conn, err := d.dialFirst(ctx, network, ipv6Addrs, port)
			if err == nil || !d.config.Failover {
				return conn, err
			}
		}
		return d.dialFirst(ctx, network, ipv4Addrs, port)
	default:
		return d.dialParallel(ctx, network, ipv4Addrs, ipv6Addrs, port)
	}
}

func (d *DualStackDialer) dialFirst(ctx context.Context, network string, ips []net.IP, port int) (net.Conn, error) {
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses available")
	}

	var lastErr error
	for _, ip := range ips {
		target := fmt.Sprintf("%s:%d", ip.String(), port)
		conn, err := (&net.Dialer{}).DialContext(ctx, network, target)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (d *DualStackDialer) dialStrategy(ctx context.Context, network string, ips []net.IP, port int) (net.Conn, error) {
	switch d.config.Strategy {
	case "latency":
		return d.dialByLatency(ctx, network, ips, port)
	case "random":
		return d.dialRandom(ctx, network, ips, port)
	case "roundrobin":
		return d.dialFirst(ctx, network, ips, port)
	default:
		return d.dialFirst(ctx, network, ips, port)
	}
}

func (d *DualStackDialer) dialByLatency(ctx context.Context, network string, ips []net.IP, port int) (net.Conn, error) {
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses available")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		conn net.Conn
		err  error
	}
	resultCh := make(chan result, len(ips))

	for _, ip := range ips {
		ip := ip
		go func() {
			target := fmt.Sprintf("%s:%d", ip.String(), port)
			conn, err := (&net.Dialer{}).DialContext(ctx, network, target)
			resultCh <- result{conn: conn, err: err}
		}()
	}

	var firstConn net.Conn
	var firstErr error
	for range ips {
		select {
		case r := <-resultCh:
			if r.err == nil {
				if firstConn == nil {
					firstConn = r.conn
					cancel()
				} else {
					r.conn.Close()
				}
			} else if firstErr == nil {
				firstErr = r.err
			}
		case <-ctx.Done():
			if firstConn != nil {
				return firstConn, nil
			}
			return nil, ctx.Err()
		}
	}
	if firstConn != nil {
		return firstConn, nil
	}
	return nil, firstErr
}

func (d *DualStackDialer) dialRandom(ctx context.Context, network string, ips []net.IP, port int) (net.Conn, error) {
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses available")
	}
	idx := rand.Intn(len(ips))
	target := fmt.Sprintf("%s:%d", ips[idx].String(), port)
	return (&net.Dialer{}).DialContext(ctx, network, target)
}

type dialResult struct {
	conn net.Conn
	err  error
	tag  string
}

func (d *DualStackDialer) dialParallel(ctx context.Context, network string, ipv4Addrs, ipv6Addrs []net.IP, port int) (net.Conn, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	hasIPv4 := len(ipv4Addrs) > 0 || len(d.ipv4Pool.addresses) > 0
	hasIPv6 := len(ipv6Addrs) > 0 || len(d.ipv6Pool.addresses) > 0
	if !hasIPv4 && !hasIPv6 {
		return nil, fmt.Errorf("no addresses available")
	}

	resultCh := make(chan dialResult, len(ipv4Addrs)+len(ipv6Addrs)+len(d.ipv4Pool.addresses)+len(d.ipv6Pool.addresses))

	for _, ip := range ipv4Addrs {
		ip := ip
		go func() {
			target := fmt.Sprintf("%s:%d", ip.String(), port)
			conn, err := (&net.Dialer{}).DialContext(ctx, network, target)
			resultCh <- dialResult{conn: conn, err: err}
		}()
	}
	for _, ip := range ipv6Addrs {
		ip := ip
		go func() {
			target := fmt.Sprintf("%s:%d", ip.String(), port)
			conn, err := (&net.Dialer{}).DialContext(ctx, network, target)
			resultCh <- dialResult{conn: conn, err: err}
		}()
	}

	d.mu.RLock()
	if hasIPv4 {
		for _, addr := range d.ipv4Pool.addresses {
			if !addr.Alive {
				continue
			}
			addr := addr
			go func() {
				target := fmt.Sprintf("%s:%d", addr.IP.String(), addr.Port)
				conn, err := (&net.Dialer{}).DialContext(ctx, network, target)
				resultCh <- dialResult{conn: conn, err: err, tag: addr.Tag}
			}()
		}
	}
	if hasIPv6 {
		for _, addr := range d.ipv6Pool.addresses {
			if !addr.Alive {
				continue
			}
			addr := addr
			go func() {
				target := fmt.Sprintf("%s:%d", addr.IP.String(), addr.Port)
				conn, err := (&net.Dialer{}).DialContext(ctx, network, target)
				resultCh <- dialResult{conn: conn, err: err, tag: addr.Tag}
			}()
		}
	}
	d.mu.RUnlock()

	var firstConn net.Conn
	var firstErr error
	count := 0
	d.mu.RLock()
	count += len(d.ipv4Pool.addresses) + len(d.ipv6Pool.addresses)
	d.mu.RUnlock()
	count += len(ipv4Addrs) + len(ipv6Addrs)

	for i := 0; i < count; i++ {
		select {
		case r := <-resultCh:
			if r.err == nil {
				if firstConn == nil {
					firstConn = r.conn
					cancel()
				} else {
					r.conn.Close()
				}
			} else if firstErr == nil {
				firstErr = r.err
			}
		case <-ctx.Done():
			if firstConn != nil {
				return firstConn, nil
			}
			return nil, firstErr
		}
	}
	if firstConn != nil {
		return firstConn, nil
	}
	return nil, firstErr
}

func (d *DualStackDialer) ProbeAll() {
	d.mu.RLock()
	ipv4Addrs := make([]MultiLineAddr, len(d.ipv4Pool.addresses))
	copy(ipv4Addrs, d.ipv4Pool.addresses)
	ipv6Addrs := make([]MultiLineAddr, len(d.ipv6Pool.addresses))
	copy(ipv6Addrs, d.ipv6Pool.addresses)
	d.mu.RUnlock()

	for _, addr := range ipv4Addrs {
		d.probeAddr(addr, d.ipv4Pool)
	}
	for _, addr := range ipv6Addrs {
		d.probeAddr(addr, d.ipv6Pool)
	}
}

func (d *DualStackDialer) probeAddr(addr MultiLineAddr, pool *MultiLinePool) {
	// Use the actual address port instead of hardcoded 80
	probePort := addr.Port
	if probePort == 0 {
		probePort = 443 // Default to HTTPS port if no port specified
	}
	start := time.Now()
	target := net.JoinHostPort(addr.IP.String(), fmt.Sprintf("%d", probePort))
	conn, err := (&net.Dialer{Timeout: d.config.ProbeTimeout}).Dial("tcp", target)
	latency := time.Since(start)

	pool.mu.Lock()
	stats, exists := pool.stats[addr.Tag]
	if !exists {
		stats = &LineStats{MinLatency: time.Hour}
		pool.stats[addr.Tag] = stats
	}
	stats.LastChecked = time.Now()
	stats.LastLatency = latency

	if err != nil {
		stats.FailureCount++
		stats.ConsecutiveFails++
		if stats.ConsecutiveFails >= 3 {
			for i, a := range pool.addresses {
				if a.Tag == addr.Tag {
					pool.addresses[i].Alive = false
					break
				}
			}
		}
	} else {
		conn.Close()
		stats.SuccessCount++
		stats.ConsecutiveFails = 0
		stats.TotalLatency += latency
		if latency < stats.MinLatency {
			stats.MinLatency = latency
		}
		if latency > stats.MaxLatency {
			stats.MaxLatency = latency
		}
		for i, a := range pool.addresses {
			if a.Tag == addr.Tag {
				pool.addresses[i].Alive = true
				pool.addresses[i].Latency = latency
				break
			}
		}
	}
	pool.mu.Unlock()
}

func (d *DualStackDialer) GetStats(tag string) *LineStats {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if stats, ok := d.ipv4Pool.stats[tag]; ok {
		return stats
	}
	if stats, ok := d.ipv6Pool.stats[tag]; ok {
		return stats
	}
	return nil
}

func (d *DualStackDialer) Close() {
	if d.ipv4Pool.cancel != nil {
		d.ipv4Pool.cancel()
	}
	if d.ipv6Pool.cancel != nil {
		d.ipv6Pool.cancel()
	}
}

func ResolveWithPreference(host string, preference IPPreference) ([]net.IP, error) {
	ips, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
	if err != nil {
		return nil, err
	}
	var result []net.IP
	switch preference {
	case IPv4Only:
		for _, ip := range ips {
			if ip.IP.To4() != nil {
				result = append(result, ip.IP)
			}
		}
	case IPv6Only:
		for _, ip := range ips {
			if ip.IP.To4() == nil {
				result = append(result, ip.IP)
			}
		}
	default:
		for _, ip := range ips {
			result = append(result, ip.IP)
		}
	}
	return result, nil
}

type MultiLineManager struct {
	config    *DualStackConfig
	dialer    *DualStackDialer
	lines     map[string]*MultiLineAddr
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewMultiLineManager(config *DualStackConfig) *MultiLineManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &MultiLineManager{
		config: config,
		dialer: NewDualStackDialer(config),
		lines:  make(map[string]*MultiLineAddr),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (m *MultiLineManager) AddLine(tag string, ip net.IP, port int, weight int) {
	addr := &MultiLineAddr{
		IP:     ip,
		Port:   port,
		Weight: weight,
		Alive:  true,
		Tag:    tag,
	}
	m.mu.Lock()
	m.lines[tag] = addr
	m.mu.Unlock()
	m.dialer.AddAddress(ip, port, tag, weight)
}

func (m *MultiLineManager) RemoveLine(tag string) {
	m.mu.Lock()
	delete(m.lines, tag)
	m.mu.Unlock()
	m.dialer.RemoveAddress(tag)
}

func (m *MultiLineManager) GetDialer() *DualStackDialer {
	return m.dialer
}

func (m *MultiLineManager) StartProbing(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		m.dialer.ProbeAll()
		for {
			select {
			case <-ticker.C:
				m.dialer.ProbeAll()
			case <-m.ctx.Done():
				return
			}
		}
	}()
}

func (m *MultiLineManager) Stop() {
	m.cancel()
	m.dialer.Close()
}


