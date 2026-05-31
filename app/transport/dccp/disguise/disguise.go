// Package disguise provides DCCP traffic disguise/obfuscation methods.
// These methods make DCCP traffic appear as other protocols to evade detection.
package disguise

import (
	"context"
	"net"
)

// Method represents a disguise method type.
type Method string

const (
	MethodNone           Method = "none"
	MethodTLS            Method = "tls"            // DCCP over TLS (appears as HTTPS)
	MethodWebSocket      Method = "websocket"      // DCCP over WebSocket
	MethodHTTP2          Method = "http2"          // DCCP over HTTP/2
	MethodGRPC           Method = "grpc"           // DCCP over gRPC
	MethodDomainFronting Method = "domain_fronting" // DCCP via domain fronting
	MethodDTLS           Method = "dtls"           // DCCP disguised as DTLS
	MethodWireGuard      Method = "wireguard"      // DCCP disguised as WireGuard
	MethodHTTPS          Method = "https"          // DCCP disguised as full HTTPS session
	MethodDNS            Method = "dns"            // DCCP disguised as DNS traffic
	MethodHTTPUpgrade    Method = "http_upgrade"   // DCCP via HTTP Upgrade
	MethodTrafficShape   Method = "traffic_shape"  // DCCP with traffic shaping
)

// Config holds disguise configuration.
type Config struct {
	Method      Method `json:"method"`
	TargetHost  string `json:"targetHost"`
	TargetPort  int    `json:"targetPort"`
	ServerName  string `json:"serverName"`
	UserAgent   string `json:"userAgent"`
	Path        string `json:"path"`
	PublicKey   []byte `json:"publicKey"`
	PrivateKey  []byte `json:"privateKey"`
	ShortID     string `json:"shortID"`
	ServiceName string `json:"serviceName"`
	ALPN        []string `json:"alpn"`
}

// Disguise is the interface for all DCCP disguise methods.
type Disguise interface {
	// Method returns the disguise method name.
	Method() Method

	// Dial wraps a connection with the disguise.
	Dial(ctx context.Context, addr net.Addr) (net.Conn, error)

	// Listen creates a disguised listener.
	Listen(addr net.Addr) (net.Listener, error)

	// WrapConn wraps an existing connection with the disguise.
	WrapConn(conn net.Conn) net.Conn

	// Close releases disguise resources.
	Close() error
}

// BaseDisguise provides common fields for disguise implementations.
type BaseDisguise struct {
	method Method
	config *Config
}

func (b *BaseDisguise) Method() Method { return b.method }

// NewDisguise creates a new disguise instance based on the method.
func NewDisguise(method Method, config *Config) Disguise {
	switch method {
	case MethodTLS:
		return NewTLSDisguise(config)
	case MethodWebSocket:
		return NewWebSocketDisguise(config)
	case MethodHTTP2:
		return NewHTTP2Disguise(config)
	case MethodGRPC:
		return NewGRPCDisguise(config)
	case MethodDomainFronting:
		return NewDomainFrontingDisguise(config)
	case MethodDTLS:
		return NewDTLSDisguise(config)
	case MethodWireGuard:
		return NewWireGuardDisguise(config)
	case MethodHTTPS:
		return NewHTTPSDisguise(config)
	case MethodDNS:
		return NewDNSDisguise(config)
	case MethodHTTPUpgrade:
		return NewHTTPUpgradeDisguise(config)
	case MethodTrafficShape:
		return NewTrafficShapeDisguise(config)
	default:
		return NewNoneDisguise(config)
	}
}
