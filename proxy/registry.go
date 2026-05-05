package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/Hhz0823/oiwest-core/config"
)

type ProtocolType string

const (
	ProtocolVMess       ProtocolType = "vmess"
	ProtocolVLESS       ProtocolType = "vless"
	ProtocolTrojan      ProtocolType = "trojan"
	ProtocolShadowsocks ProtocolType = "shadowsocks"
	ProtocolSOCKS       ProtocolType = "socks"
	ProtocolHTTP        ProtocolType = "http"
	ProtocolDokodemo    ProtocolType = "dokodemo-door"
	ProtocolLoopback    ProtocolType = "loopback"
	ProtocolFreedom     ProtocolType = "freedom"
	ProtocolBlackhole   ProtocolType = "blackhole"
	ProtocolDNS         ProtocolType = "dns"
	ProtocolWireGuard   ProtocolType = "wireguard"
	ProtocolDirect      ProtocolType = "direct"
	ProtocolDCCP        ProtocolType = "dccp"
)

type InboundFactory func(ctx context.Context, config *config.InboundConfig, manager *ProxyManager) (InboundHandler, error)
type OutboundFactory func(ctx context.Context, config *config.OutboundConfig) (OutboundHandler, error)

var (
	inboundRegistry  = make(map[ProtocolType]InboundFactory)
	outboundRegistry = make(map[ProtocolType]OutboundFactory)
	registryMu       sync.RWMutex
)

func RegisterInboundProtocol(protocol ProtocolType, factory InboundFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	inboundRegistry[protocol] = factory
}

func RegisterOutboundProtocol(protocol ProtocolType, factory OutboundFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	outboundRegistry[protocol] = factory
}

func GetInboundFactory(protocol ProtocolType) (InboundFactory, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	factory, ok := inboundRegistry[protocol]
	if !ok {
		return nil, fmt.Errorf("inbound protocol '%s' not registered", protocol)
	}
	return factory, nil
}

func GetOutboundFactory(protocol ProtocolType) (OutboundFactory, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	factory, ok := outboundRegistry[protocol]
	if !ok {
		return nil, fmt.Errorf("outbound protocol '%s' not registered", protocol)
	}
	return factory, nil
}

func ListInboundProtocols() []ProtocolType {
	registryMu.RLock()
	defer registryMu.RUnlock()
	protocols := make([]ProtocolType, 0, len(inboundRegistry))
	for p := range inboundRegistry {
		protocols = append(protocols, p)
	}
	return protocols
}

func ListOutboundProtocols() []ProtocolType {
	registryMu.RLock()
	defer registryMu.RUnlock()
	protocols := make([]ProtocolType, 0, len(outboundRegistry))
	for p := range outboundRegistry {
		protocols = append(protocols, p)
	}
	return protocols
}

type DokodemoInboundHandler struct {
	tag     string
	port    uint16
	listen  string
	address string
	network string
	timeout int
	manager *ProxyManager
}

func (h *DokodemoInboundHandler) Tag() string       { return h.tag }
func (h *DokodemoInboundHandler) Network() []string { return []string{"tcp", "udp"} }

func (h *DokodemoInboundHandler) Process(ctx context.Context, conn net.Conn, handler InboundConnHandler) error {
	handler(ctx, conn)
	return nil
}

type DokodemoConfig struct {
	Address       string `json:"address"`
	Port          uint16 `json:"port"`
	Network       string `json:"network"`
	Timeout       int    `json:"timeout"`
	FollowRedirect bool  `json:"followRedirect"`
}

func CreateDokodemoInbound(ctx context.Context, cfg *config.InboundConfig, manager *ProxyManager) (InboundHandler, error) {
	dokodemoCfg := &DokodemoConfig{
		Network: "tcp",
		Timeout: 300,
	}
	if cfg.Settings != nil {
		json.Unmarshal(cfg.Settings, dokodemoCfg)
	}
	return &DokodemoInboundHandler{
		tag:     cfg.Tag,
		port:    cfg.Port,
		listen:  cfg.Listen,
		address: dokodemoCfg.Address,
		network: dokodemoCfg.Network,
		timeout: dokodemoCfg.Timeout,
		manager: manager,
	}, nil
}

type LoopbackInboundHandler struct {
	tag     string
	port    uint16
	listen  string
	manager *ProxyManager
}

func (h *LoopbackInboundHandler) Tag() string       { return h.tag }
func (h *LoopbackInboundHandler) Network() []string { return []string{"tcp"} }
func (h *LoopbackInboundHandler) Process(ctx context.Context, conn net.Conn, handler InboundConnHandler) error {
	handler(ctx, conn)
	return nil
}

type LoopbackConfig struct {
	InboundTag string `json:"inboundTag"`
}

func CreateLoopbackInbound(ctx context.Context, cfg *config.InboundConfig, manager *ProxyManager) (InboundHandler, error) {
	return &LoopbackInboundHandler{
		tag:    cfg.Tag,
		port:   cfg.Port,
		listen: cfg.Listen,
		manager: manager,
	}, nil
}

type DNSOutboundHandler struct {
	tag     string
	network string
	address string
	port    uint16
}

func (h *DNSOutboundHandler) Tag() string { return h.tag }

type DNSConfig struct {
	Network string `json:"network"`
	Address string `json:"address"`
	Port    uint16 `json:"port"`
}

func CreateDNSOutbound(ctx context.Context, cfg *config.OutboundConfig) (OutboundHandler, error) {
	dnsCfg := &DNSConfig{Network: "udp"}
	if cfg.Settings != nil {
		json.Unmarshal(cfg.Settings, dnsCfg)
	}
	return &DNSOutboundHandler{
		tag:     cfg.Tag,
		network: dnsCfg.Network,
		address: dnsCfg.Address,
		port:    dnsCfg.Port,
	}, nil
}

func (h *DNSOutboundHandler) Process(ctx context.Context, link *Link) error {
	target := fmt.Sprintf("%s:%d", h.address, h.port)
	conn, err := net.DialTimeout(h.network, target, 15*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	errCh := make(chan error, 2)
	go func() { _, err := io.Copy(conn, link.Reader); errCh <- err }()
	go func() { _, err := io.Copy(link.Writer, conn); errCh <- err }()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
