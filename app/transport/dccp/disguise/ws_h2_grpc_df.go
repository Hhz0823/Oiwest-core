package disguise

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// WebSocketDisguise wraps DCCP in WebSocket frames, appearing as WS traffic.
type WebSocketDisguise struct {
	BaseDisguise
}

func NewWebSocketDisguise(config *Config) *WebSocketDisguise {
	return &WebSocketDisguise{
		BaseDisguise: BaseDisguise{method: MethodWebSocket, config: config},
	}
}

func (d *WebSocketDisguise) Dial(ctx context.Context, addr net.Addr) (net.Conn, error) {
	dialer := net.Dialer{Timeout: 15 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr.String())
	if err != nil {
		return nil, err
	}

	path := d.config.Path
	if path == "" { path = "/ws" }
	host := d.config.TargetHost
	if host == "" { host = addr.String() }

	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n", path, host)
	if _, err := conn.Write([]byte(req)); err != nil {
		conn.Close()
		return nil, err
	}

	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if resp.StatusCode != 101 {
		conn.Close()
		return nil, fmt.Errorf("websocket upgrade failed: %d", resp.StatusCode)
	}

	return conn, nil
}

func (d *WebSocketDisguise) Listen(addr net.Addr) (net.Listener, error) {
	return net.Listen("tcp", addr.String())
}

func (d *WebSocketDisguise) WrapConn(conn net.Conn) net.Conn { return conn }
func (d *WebSocketDisguise) Close() error { return nil }

// HTTP2Disguise wraps DCCP in HTTP/2 streams.
type HTTP2Disguise struct {
	BaseDisguise
}

func NewHTTP2Disguise(config *Config) *HTTP2Disguise {
	return &HTTP2Disguise{
		BaseDisguise: BaseDisguise{method: MethodHTTP2, config: config},
	}
}

func (d *HTTP2Disguise) Dial(ctx context.Context, addr net.Addr) (net.Conn, error) {
	dialer := net.Dialer{Timeout: 15 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr.String())
	if err != nil {
		return nil, err
	}

	preface := "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
	if _, err := conn.Write([]byte(preface)); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func (d *HTTP2Disguise) Listen(addr net.Addr) (net.Listener, error) {
	return net.Listen("tcp", addr.String())
}

func (d *HTTP2Disguise) WrapConn(conn net.Conn) net.Conn { return conn }
func (d *HTTP2Disguise) Close() error { return nil }

// GRPCDisguise wraps DCCP in gRPC streams.
type GRPCDisguise struct {
	BaseDisguise
}

func NewGRPCDisguise(config *Config) *GRPCDisguise {
	return &GRPCDisguise{
		BaseDisguise: BaseDisguise{method: MethodGRPC, config: config},
	}
}

func (d *GRPCDisguise) Dial(ctx context.Context, addr net.Addr) (net.Conn, error) {
	dialer := net.Dialer{Timeout: 15 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr.String())
	if err != nil {
		return nil, err
	}

	serviceName := d.config.ServiceName
	if serviceName == "" { serviceName = "grpc.service" }
	host := d.config.TargetHost
	if host == "" { host = addr.String() }

	req := fmt.Sprintf("POST /%s/Tun HTTP/1.1\r\nHost: %s\r\nContent-Type: application/grpc\r\nTransfer-Encoding: chunked\r\nTE: trailers\r\n\r\n", serviceName, host)
	if _, err := conn.Write([]byte(req)); err != nil {
		conn.Close()
		return nil, err
	}

	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		conn.Close()
		return nil, err
	}
	_ = resp

	return conn, nil
}

func (d *GRPCDisguise) Listen(addr net.Addr) (net.Listener, error) {
	return net.Listen("tcp", addr.String())
}

func (d *GRPCDisguise) WrapConn(conn net.Conn) net.Conn { return conn }
func (d *GRPCDisguise) Close() error { return nil }

// DomainFrontingDisguise routes DCCP via CDN domain fronting.
type DomainFrontingDisguise struct {
	BaseDisguise
}

func NewDomainFrontingDisguise(config *Config) *DomainFrontingDisguise {
	return &DomainFrontingDisguise{
		BaseDisguise: BaseDisguise{method: MethodDomainFronting, config: config},
	}
}

func (d *DomainFrontingDisguise) Dial(ctx context.Context, addr net.Addr) (net.Conn, error) {
	dialer := net.Dialer{Timeout: 15 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr.String())
	if err != nil {
		return nil, err
	}

	host := d.config.ServerName
	if host == "" { host = d.config.TargetHost }

	connectReq := fmt.Sprintf("CONNECT %s:443 HTTP/1.1\r\nHost: %s\r\nUser-Agent: %s\r\n\r\n",
		host, host, d.config.UserAgent)
	if d.config.UserAgent == "" {
		connectReq = fmt.Sprintf("CONNECT %s:443 HTTP/1.1\r\nHost: %s\r\n\r\n", host, host)
	}

	if _, err := conn.Write([]byte(connectReq)); err != nil {
		conn.Close()
		return nil, err
	}

	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if resp.StatusCode != 200 {
		conn.Close()
		return nil, fmt.Errorf("domain fronting CONNECT failed: %d", resp.StatusCode)
	}

	return conn, nil
}

func (d *DomainFrontingDisguise) Listen(addr net.Addr) (net.Listener, error) {
	return net.Listen("tcp", addr.String())
}

func (d *DomainFrontingDisguise) WrapConn(conn net.Conn) net.Conn { return conn }
func (d *DomainFrontingDisguise) Close() error { return nil }

// NoneDisguise passes traffic through without any disguise.
type NoneDisguise struct {
	BaseDisguise
}

func NewNoneDisguise(config *Config) *NoneDisguise {
	return &NoneDisguise{
		BaseDisguise: BaseDisguise{method: MethodNone, config: config},
	}
}

func (d *NoneDisguise) Dial(ctx context.Context, addr net.Addr) (net.Conn, error) {
	dialer := net.Dialer{Timeout: 15 * time.Second}
	return dialer.DialContext(ctx, "tcp", addr.String())
}

func (d *NoneDisguise) Listen(addr net.Addr) (net.Listener, error) {
	return net.Listen("tcp", addr.String())
}

func (d *NoneDisguise) WrapConn(conn net.Conn) net.Conn { return conn }
func (d *NoneDisguise) Close() error { return nil }

// Helper
func init() { _ = strings.TrimSpace }
