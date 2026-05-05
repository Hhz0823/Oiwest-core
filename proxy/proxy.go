package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	netaddr "github.com/Hhz0823/oiwest-core/common/net"
	"github.com/Hhz0823/oiwest-core/config"
	"github.com/Hhz0823/oiwest-core/transport"
)

var (
	ErrProxyNotFound    = errors.New("proxy: outbound not found")
	ErrInboundNotFound  = errors.New("proxy: inbound not found")
	ErrUnsupportedProxy = errors.New("proxy: unsupported proxy protocol")
)

type OutboundHandler interface {
	Tag() string
	Process(ctx context.Context, link *Link) error
}

type InboundHandler interface {
	Tag() string
	Network() []string
	Process(ctx context.Context, conn net.Conn, handler InboundConnHandler) error
}

type InboundConnHandler func(ctx context.Context, conn net.Conn)

type Link struct {
	Reader io.Reader
	Writer io.Writer
}

type HandlerConfig struct {
	Tag            string
	Port           uint16
	Listen         string
	Settings       json.RawMessage
	StreamSettings *transport.StreamSettings
}

type ProxyManager struct {
	inbounds  map[string]InboundHandler
	outbounds map[string]OutboundHandler
	config    *config.Config
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewProxyManager(cfg *config.Config) *ProxyManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ProxyManager{
		inbounds:  make(map[string]InboundHandler),
		outbounds: make(map[string]OutboundHandler),
		config:    cfg,
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (pm *ProxyManager) RegisterInbound(handler InboundHandler) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.inbounds[handler.Tag()] = handler
}

func (pm *ProxyManager) RegisterOutbound(handler OutboundHandler) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.outbounds[handler.Tag()] = handler
}

func (pm *ProxyManager) GetOutbound(tag string) (OutboundHandler, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	h, ok := pm.outbounds[tag]
	if !ok {
		return nil, ErrProxyNotFound
	}
	return h, nil
}

func (pm *ProxyManager) GetInbound(tag string) (InboundHandler, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	h, ok := pm.inbounds[tag]
	if !ok {
		return nil, ErrInboundNotFound
	}
	return h, nil
}

func (pm *ProxyManager) Start() error {
	for _, inbound := range pm.config.Inbounds {
		handler, err := pm.createInboundHandler(&inbound)
		if err != nil {
			return err
		}
		go pm.startInbound(handler, &inbound)
	}
	return nil
}

func (pm *ProxyManager) createInboundHandler(inbound *config.InboundConfig) (InboundHandler, error) {
	factory, err := GetInboundFactory(ProtocolType(inbound.Protocol))
	if err == nil {
		return factory(pm.ctx, inbound, pm)
	}
	switch inbound.Protocol {
	case "dccp":
		return &DCCPInboundHandler{
			tag:      inbound.Tag,
			port:     inbound.Port,
			listen:   inbound.Listen,
			settings: inbound.StreamSettings,
			manager:  pm,
		}, nil
	case "socks":
		return &SOCKSInboundHandler{
			tag:     inbound.Tag,
			port:    inbound.Port,
			listen:  inbound.Listen,
			manager: pm,
		}, nil
	case "http":
		return &HTTPInboundHandler{
			tag:     inbound.Tag,
			port:    inbound.Port,
			listen:  inbound.Listen,
			manager: pm,
		}, nil
	case "vmess":
		return &VmessInboundHandler{
			tag:            inbound.Tag,
			port:           inbound.Port,
			listen:         inbound.Listen,
			settings:       inbound.Settings,
			streamSettings: inbound.StreamSettings,
			manager:        pm,
		}, nil
	case "vless":
		return &VlessInboundHandler{
			tag:            inbound.Tag,
			port:           inbound.Port,
			listen:         inbound.Listen,
			settings:       inbound.Settings,
			streamSettings: inbound.StreamSettings,
			manager:        pm,
		}, nil
	case "trojan":
		return &TrojanInboundHandler{
			tag:            inbound.Tag,
			port:           inbound.Port,
			listen:         inbound.Listen,
			settings:       inbound.Settings,
			streamSettings: inbound.StreamSettings,
			manager:        pm,
		}, nil
	case "shadowsocks":
		return &ShadowsocksInboundHandler{
			tag:            inbound.Tag,
			port:           inbound.Port,
			listen:         inbound.Listen,
			settings:       inbound.Settings,
			streamSettings: inbound.StreamSettings,
			manager:        pm,
		}, nil
	default:
		return nil, ErrUnsupportedProxy
	}
}

func (pm *ProxyManager) startInbound(handler InboundHandler, inbound *config.InboundConfig) {
	pm.RegisterInbound(handler)

	addr := &net.TCPAddr{
		IP:   net.ParseIP(inbound.Listen),
		Port: int(inbound.Port),
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return
	}
	defer listener.Close()

	for {
		select {
		case <-pm.ctx.Done():
			return
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		go handler.Process(pm.ctx, conn, pm.dispatchRequest)
	}
}

func (pm *ProxyManager) dispatchRequest(ctx context.Context, conn net.Conn) {
	outbound, err := pm.selectOutbound(conn)
	if err != nil {
		conn.Close()
		return
	}

	link := &Link{
		Reader: conn,
		Writer: conn,
	}

	outbound.Process(ctx, link)
}

func (pm *ProxyManager) selectOutbound(conn net.Conn) (OutboundHandler, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, outbound := range pm.outbounds {
		return outbound, nil
	}

	return nil, ErrProxyNotFound
}

func (pm *ProxyManager) Close() error {
	pm.cancel()
	return nil
}

type DCCPInboundHandler struct {
	tag      string
	port     uint16
	listen   string
	settings *transport.StreamSettings
	manager  *ProxyManager
}

func NewDCCPInboundHandler(tag string, port uint16, listen string, settings *transport.StreamSettings, mgr *ProxyManager) *DCCPInboundHandler {
	return &DCCPInboundHandler{tag: tag, port: port, listen: listen, settings: settings, manager: mgr}
}

func NewSOCKSInboundHandler(tag string, port uint16, listen string, mgr *ProxyManager) *SOCKSInboundHandler {
	return &SOCKSInboundHandler{tag: tag, port: port, listen: listen, manager: mgr}
}

func NewHTTPInboundHandler(tag string, port uint16, listen string, mgr *ProxyManager) *HTTPInboundHandler {
	return &HTTPInboundHandler{tag: tag, port: port, listen: listen, manager: mgr}
}

func NewVmessInboundHandler(tag string, port uint16, listen string, settings json.RawMessage, streamSettings *transport.StreamSettings, mgr *ProxyManager) *VmessInboundHandler {
	return &VmessInboundHandler{tag: tag, port: port, listen: listen, settings: settings, streamSettings: streamSettings, manager: mgr}
}

func NewVlessInboundHandler(tag string, port uint16, listen string, settings json.RawMessage, streamSettings *transport.StreamSettings, mgr *ProxyManager) *VlessInboundHandler {
	return &VlessInboundHandler{tag: tag, port: port, listen: listen, settings: settings, streamSettings: streamSettings, manager: mgr}
}

func NewTrojanInboundHandler(tag string, port uint16, listen string, settings json.RawMessage, streamSettings *transport.StreamSettings, mgr *ProxyManager) *TrojanInboundHandler {
	return &TrojanInboundHandler{tag: tag, port: port, listen: listen, settings: settings, streamSettings: streamSettings, manager: mgr}
}

func NewShadowsocksInboundHandler(tag string, port uint16, listen string, settings json.RawMessage, streamSettings *transport.StreamSettings, mgr *ProxyManager) *ShadowsocksInboundHandler {
	return &ShadowsocksInboundHandler{tag: tag, port: port, listen: listen, settings: settings, streamSettings: streamSettings, manager: mgr}
}

func NewDCCPOutboundHandler(tag string, target netaddr.Address, streamSettings *transport.StreamSettings) *DCCPOutboundHandler {
	return &DCCPOutboundHandler{tag: tag, target: target, streamSettings: streamSettings}
}

func NewDirectOutboundHandler(tag string) *DirectOutboundHandler {
	return &DirectOutboundHandler{tag: tag}
}

func NewFreedomOutboundHandler(tag string) *FreedomOutboundHandler {
	return &FreedomOutboundHandler{tag: tag}
}

func NewBlackholeOutboundHandler(tag string) *BlackholeOutboundHandler {
	return &BlackholeOutboundHandler{tag: tag}
}

func NewDNSOutboundHandler(tag string, network string, address string, port uint16) *DNSOutboundHandler {
	return &DNSOutboundHandler{tag: tag, network: network, address: address, port: port}
}

func NewSocksOutboundHandler(tag string) *SocksOutboundHandler {
	return &SocksOutboundHandler{tag: tag}
}

func (h *DCCPInboundHandler) Tag() string       { return h.tag }
func (h *DCCPInboundHandler) Network() []string { return []string{"dccp"} }

func (h *DCCPInboundHandler) Process(ctx context.Context, conn net.Conn, handler InboundConnHandler) error {
	if h.settings != nil && h.settings.Network == transport.TransportDCCP {
		dccpTransport := transport.NewDCCPTransport(h.settings)
		handler(ctx, dccpTransport)
		return nil
	}
	handler(ctx, conn)
	return nil
}

type SOCKSInboundHandler struct {
	tag     string
	port    uint16
	listen  string
	manager *ProxyManager
}

func (h *SOCKSInboundHandler) Tag() string       { return h.tag }
func (h *SOCKSInboundHandler) Network() []string { return []string{"tcp"} }
func (h *SOCKSInboundHandler) Process(ctx context.Context, conn net.Conn, handler InboundConnHandler) error {
	handler(ctx, conn)
	return nil
}

type HTTPInboundHandler struct {
	tag     string
	port    uint16
	listen  string
	manager *ProxyManager
}

func (h *HTTPInboundHandler) Tag() string       { return h.tag }
func (h *HTTPInboundHandler) Network() []string { return []string{"tcp"} }
func (h *HTTPInboundHandler) Process(ctx context.Context, conn net.Conn, handler InboundConnHandler) error {
	handler(ctx, conn)
	return nil
}

type VmessInboundHandler struct {
	tag            string
	port           uint16
	listen         string
	settings       json.RawMessage
	streamSettings *transport.StreamSettings
	manager        *ProxyManager
}

func (h *VmessInboundHandler) Tag() string       { return h.tag }
func (h *VmessInboundHandler) Network() []string { return []string{"tcp", "dccp"} }
func (h *VmessInboundHandler) Process(ctx context.Context, conn net.Conn, handler InboundConnHandler) error {
	if h.streamSettings != nil && h.streamSettings.Network == transport.TransportDCCP {
		dccpTransport := transport.NewDCCPTransport(h.streamSettings)
		handler(ctx, dccpTransport)
		return nil
	}
	handler(ctx, conn)
	return nil
}

type VlessInboundHandler struct {
	tag            string
	port           uint16
	listen         string
	settings       json.RawMessage
	streamSettings *transport.StreamSettings
	manager        *ProxyManager
}

func (h *VlessInboundHandler) Tag() string       { return h.tag }
func (h *VlessInboundHandler) Network() []string { return []string{"tcp", "dccp", "quic"} }
func (h *VlessInboundHandler) Process(ctx context.Context, conn net.Conn, handler InboundConnHandler) error {
	if h.streamSettings != nil && h.streamSettings.Network == transport.TransportDCCP {
		dccpTransport := transport.NewDCCPTransport(h.streamSettings)
		handler(ctx, dccpTransport)
		return nil
	}
	handler(ctx, conn)
	return nil
}

type TrojanInboundHandler struct {
	tag            string
	port           uint16
	listen         string
	settings       json.RawMessage
	streamSettings *transport.StreamSettings
	manager        *ProxyManager
}

func (h *TrojanInboundHandler) Tag() string       { return h.tag }
func (h *TrojanInboundHandler) Network() []string { return []string{"tcp", "ws", "h2", "grpc", "dccp"} }
func (h *TrojanInboundHandler) Process(ctx context.Context, conn net.Conn, handler InboundConnHandler) error {
	if h.streamSettings != nil && h.streamSettings.Network == transport.TransportDCCP {
		dccpTransport := transport.NewDCCPTransport(h.streamSettings)
		handler(ctx, dccpTransport)
		return nil
	}
	handler(ctx, conn)
	return nil
}

type ShadowsocksInboundHandler struct {
	tag            string
	port           uint16
	listen         string
	settings       json.RawMessage
	streamSettings *transport.StreamSettings
	manager        *ProxyManager
}

func (h *ShadowsocksInboundHandler) Tag() string       { return h.tag }
func (h *ShadowsocksInboundHandler) Network() []string { return []string{"tcp", "udp", "dccp"} }
func (h *ShadowsocksInboundHandler) Process(ctx context.Context, conn net.Conn, handler InboundConnHandler) error {
	if h.streamSettings != nil && h.streamSettings.Network == transport.TransportDCCP {
		dccpTransport := transport.NewDCCPTransport(h.streamSettings)
		handler(ctx, dccpTransport)
		return nil
	}
	handler(ctx, conn)
	return nil
}

type DCCPOutboundHandler struct {
	tag            string
	target         netaddr.Address
	streamSettings *transport.StreamSettings
}

func (h *DCCPOutboundHandler) Tag() string { return h.tag }

func (h *DCCPOutboundHandler) Process(ctx context.Context, link *Link) error {
	targetAddr := &net.TCPAddr{
		IP:   h.target.IP,
		Port: int(h.target.Port),
	}

	if h.streamSettings != nil && h.streamSettings.Network == transport.TransportDCCP {
		dccpTransport := transport.NewDCCPTransport(h.streamSettings)
		if err := dccpTransport.Dial(ctx, targetAddr); err != nil {
			return err
		}
		defer dccpTransport.Close()
		errCh := make(chan error, 2)
		go func() {
			_, err := io.Copy(dccpTransport, link.Reader)
			errCh <- err
		}()
		go func() {
			_, err := io.Copy(link.Writer, dccpTransport)
			errCh <- err
		}()
		select {
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	conn, err := net.DialTCP("tcp", nil, targetAddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	errCh := make(chan error, 2)
	go func() {
		_, err := io.Copy(conn, link.Reader)
		errCh <- err
	}()
	go func() {
		_, err := io.Copy(link.Writer, conn)
		errCh <- err
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

type VMessOutboundHandler struct {
	tag            string
	address        string
	port           uint16
	uuid           string
	security       int
	streamSettings *transport.StreamSettings
}

func (h *VMessOutboundHandler) Tag() string { return h.tag }
func (h *VMessOutboundHandler) Process(ctx context.Context, link *Link) error {
	return nil
}

type VLESSOutboundHandler struct {
	tag            string
	address        string
	port           uint16
	uuid           string
	flow           string
	streamSettings *transport.StreamSettings
}

func (h *VLESSOutboundHandler) Tag() string { return h.tag }
func (h *VLESSOutboundHandler) Process(ctx context.Context, link *Link) error {
	return nil
}

type TrojanOutboundHandler struct {
	tag            string
	address        string
	port           uint16
	password       string
	streamSettings *transport.StreamSettings
}

func (h *TrojanOutboundHandler) Tag() string { return h.tag }
func (h *TrojanOutboundHandler) Process(ctx context.Context, link *Link) error {
	return nil
}

type ShadowsocksOutboundHandler struct {
	tag      string
	address  string
	port     uint16
	method   string
	password string
}

func (h *ShadowsocksOutboundHandler) Tag() string { return h.tag }
func (h *ShadowsocksOutboundHandler) Process(ctx context.Context, link *Link) error {
	return nil
}

type SocksOutboundHandler struct {
	tag     string
	address string
	port    uint16
	user    string
	pass    string
}

func (h *SocksOutboundHandler) Tag() string { return h.tag }
func (h *SocksOutboundHandler) Process(ctx context.Context, link *Link) error {
	return nil
}

type DirectOutboundHandler struct {
	tag string
}

func (h *DirectOutboundHandler) Tag() string { return h.tag }
func (h *DirectOutboundHandler) Process(ctx context.Context, link *Link) error {
	return errors.New("direct outbound: no target specified")
}

type FreedomOutboundHandler struct {
	tag string
}

func (h *FreedomOutboundHandler) Tag() string { return h.tag }
func (h *FreedomOutboundHandler) Process(ctx context.Context, link *Link) error {
	go io.Copy(io.Discard, link.Reader)
	return nil
}

type BlackholeOutboundHandler struct {
	tag string
}

func (h *BlackholeOutboundHandler) Tag() string { return h.tag }
func (h *BlackholeOutboundHandler) Process(ctx context.Context, link *Link) error {
	go io.Copy(io.Discard, link.Reader)
	return nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func init() {
	_ = time.Now()
}
