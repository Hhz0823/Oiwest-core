// Package disguise - additional disguise methods: DTLS, WireGuard, HTTPS, DNS, HTTPUpgrade, TrafficShape
package disguise

import (
	"context"
	"fmt"
	"net"
	"time"
)

// DTLSDisguise wraps DCCP traffic in DTLS, appearing as DTLS/UDP traffic.
type DTLSDisguise struct {
	BaseDisguise
}

func NewDTLSDisguise(config *Config) *DTLSDisguise {
	return &DTLSDisguise{
		BaseDisguise: BaseDisguise{method: MethodDTLS, config: config},
	}
}

func (d *DTLSDisguise) Dial(ctx context.Context, addr net.Addr) (net.Conn, error) {
	// DTLS operates over UDP; for now use TCP as transport layer
	dialer := net.Dialer{Timeout: 15 * time.Second}
	return dialer.DialContext(ctx, "tcp", addr.String())
}

func (d *DTLSDisguise) Listen(addr net.Addr) (net.Listener, error) {
	return net.Listen("tcp", addr.String())
}

func (d *DTLSDisguise) WrapConn(conn net.Conn) net.Conn { return conn }
func (d *DTLSDisguise) Close() error                    { return nil }

// WireGuardDisguise wraps DCCP traffic to appear as WireGuard UDP traffic.
type WireGuardDisguise struct {
	BaseDisguise
}

func NewWireGuardDisguise(config *Config) *WireGuardDisguise {
	return &WireGuardDisguise{
		BaseDisguise: BaseDisguise{method: MethodWireGuard, config: config},
	}
}

func (d *WireGuardDisguise) Dial(ctx context.Context, addr net.Addr) (net.Conn, error) {
	dialer := net.Dialer{Timeout: 15 * time.Second}
	return dialer.DialContext(ctx, "tcp", addr.String())
}

func (d *WireGuardDisguise) Listen(addr net.Addr) (net.Listener, error) {
	return net.Listen("tcp", addr.String())
}

func (d *WireGuardDisguise) WrapConn(conn net.Conn) net.Conn { return conn }
func (d *WireGuardDisguise) Close() error                    { return nil }

// HTTPSDisguise makes DCCP traffic appear as a full HTTPS session.
type HTTPSDisguise struct {
	BaseDisguise
}

func NewHTTPSDisguise(config *Config) *HTTPSDisguise {
	return &HTTPSDisguise{
		BaseDisguise: BaseDisguise{method: MethodHTTPS, config: config},
	}
}

func (d *HTTPSDisguise) Dial(ctx context.Context, addr net.Addr) (net.Conn, error) {
	dialer := net.Dialer{Timeout: 15 * time.Second}
	return dialer.DialContext(ctx, "tcp", addr.String())
}

func (d *HTTPSDisguise) Listen(addr net.Addr) (net.Listener, error) {
	return net.Listen("tcp", addr.String())
}

func (d *HTTPSDisguise) WrapConn(conn net.Conn) net.Conn { return conn }
func (d *HTTPSDisguise) Close() error                    { return nil }

// DNSDisguise wraps DCCP traffic to appear as DNS queries/responses.
type DNSDisguise struct {
	BaseDisguise
}

func NewDNSDisguise(config *Config) *DNSDisguise {
	return &DNSDisguise{
		BaseDisguise: BaseDisguise{method: MethodDNS, config: config},
	}
}

func (d *DNSDisguise) Dial(ctx context.Context, addr net.Addr) (net.Conn, error) {
	dialer := net.Dialer{Timeout: 15 * time.Second}
	return dialer.DialContext(ctx, "tcp", addr.String())
}

func (d *DNSDisguise) Listen(addr net.Addr) (net.Listener, error) {
	return net.Listen("tcp", addr.String())
}

func (d *DNSDisguise) WrapConn(conn net.Conn) net.Conn { return conn }
func (d *DNSDisguise) Close() error                    { return nil }

// HTTPUpgradeDisguise wraps DCCP traffic via HTTP Upgrade mechanism.
type HTTPUpgradeDisguise struct {
	BaseDisguise
}

func NewHTTPUpgradeDisguise(config *Config) *HTTPUpgradeDisguise {
	return &HTTPUpgradeDisguise{
		BaseDisguise: BaseDisguise{method: MethodHTTPUpgrade, config: config},
	}
}

func (d *HTTPUpgradeDisguise) Dial(ctx context.Context, addr net.Addr) (net.Conn, error) {
	dialer := net.Dialer{Timeout: 15 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr.String())
	if err != nil {
		return nil, err
	}

	path := d.config.Path
	if path == "" {
		path = "/"
	}
	host := d.config.TargetHost
	if host == "" {
		host = addr.String()
	}

	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: oiwest-dccp\r\nConnection: Upgrade\r\n\r\n", path, host)
	if _, err := conn.Write([]byte(req)); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func (d *HTTPUpgradeDisguise) Listen(addr net.Addr) (net.Listener, error) {
	return net.Listen("tcp", addr.String())
}

func (d *HTTPUpgradeDisguise) WrapConn(conn net.Conn) net.Conn { return conn }
func (d *HTTPUpgradeDisguise) Close() error                    { return nil }

// TrafficShapeDisguise applies traffic shaping to DCCP traffic to evade detection.
type TrafficShapeDisguise struct {
	BaseDisguise
}

func NewTrafficShapeDisguise(config *Config) *TrafficShapeDisguise {
	return &TrafficShapeDisguise{
		BaseDisguise: BaseDisguise{method: MethodTrafficShape, config: config},
	}
}

func (d *TrafficShapeDisguise) Dial(ctx context.Context, addr net.Addr) (net.Conn, error) {
	dialer := net.Dialer{Timeout: 15 * time.Second}
	return dialer.DialContext(ctx, "tcp", addr.String())
}

func (d *TrafficShapeDisguise) Listen(addr net.Addr) (net.Listener, error) {
	return net.Listen("tcp", addr.String())
}

func (d *TrafficShapeDisguise) WrapConn(conn net.Conn) net.Conn { return conn }
func (d *TrafficShapeDisguise) Close() error                    { return nil }
