package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Hhz0823/oiwest-core/config"
	"github.com/Hhz0823/oiwest-core/proxy"
	"github.com/Hhz0823/oiwest-core/router"
	"github.com/Hhz0823/oiwest-core/transport"
	"github.com/Hhz0823/oiwest-core/transport/dccp"
)

const (
	Version    = "1.0.0"
	CoreName   = "Oiwest Core"
	APIVersion = 1
)

type Core struct {
	config      *config.Config
	proxyMgr    *proxy.ProxyManager
	router      *router.Router
	ctx         context.Context
	cancel      context.CancelFunc
	running     bool
	mu          sync.Mutex
	startTime   time.Time
	stats       *CoreStats
}

type CoreStats struct {
	UplinkBytes   int64
	DownlinkBytes int64
	ActiveConns   int64
	TotalConns    int64
	mu            sync.Mutex
}

func NewCore(cfg *config.Config) *Core {
	ctx, cancel := context.WithCancel(context.Background())
	return &Core{
		config:    cfg,
		ctx:       ctx,
		cancel:    cancel,
		startTime: time.Now(),
		stats:     &CoreStats{},
	}
}

func (c *Core) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return errors.New("core already running")
	}

	log.Printf("[%s] Starting %s v%s", CoreName, CoreName, Version)
	log.Printf("[%s] DCCP Protocol Kernel initializing...", CoreName)

	c.proxyMgr = proxy.NewProxyManager(c.config)

	if c.config.Routing != nil {
		c.router = router.NewRouter(c.config.Routing)
		log.Printf("[%s] Router initialized with %d rules", CoreName, len(c.config.Routing.Rules))
	} else {
		c.router = router.NewRouter(&config.RoutingConfig{})
	}

	c.registerDefaultOutbounds()

	if err := c.proxyMgr.Start(); err != nil {
		return fmt.Errorf("failed to start proxy manager: %w", err)
	}

	c.running = true
	log.Printf("[%s] Core started successfully", CoreName)
	log.Printf("[%s] DCCP Protocol ready on configured ports", CoreName)

	return nil
}

func (c *Core) registerDefaultOutbounds() {
	directHandler := &DirectOutboundHandler{
		tag: "direct",
	}
	c.proxyMgr.RegisterOutbound(directHandler)

	freedomHandler := &FreedomOutboundHandler{
		tag: "freedom",
	}
	c.proxyMgr.RegisterOutbound(freedomHandler)

	blackholeHandler := &BlackholeOutboundHandler{
		tag: "blackhole",
	}
	c.proxyMgr.RegisterOutbound(blackholeHandler)

	for _, outbound := range c.config.Outbounds {
		handler, err := c.createOutboundHandler(&outbound)
		if err != nil {
			log.Printf("[%s] Warning: failed to create outbound handler '%s': %v", CoreName, outbound.Tag, err)
			continue
		}
		c.proxyMgr.RegisterOutbound(handler)
		log.Printf("[%s] Outbound '%s' registered (protocol: %s)", CoreName, outbound.Tag, outbound.Protocol)
	}
}

func (c *Core) createOutboundHandler(outbound *config.OutboundConfig) (proxy.OutboundHandler, error) {
	switch outbound.Protocol {
	case "dccp":
		return &DCCPOutboundProxyHandler{
			tag:            outbound.Tag,
			streamSettings: outbound.StreamSettings,
		}, nil
	case "freedom":
		return &FreedomOutboundHandler{
			tag: outbound.Tag,
		}, nil
	case "blackhole":
		return &BlackholeOutboundHandler{
			tag: outbound.Tag,
		}, nil
	default:
		return &DirectOutboundHandler{
			tag: outbound.Tag,
		}, nil
	}
}

func (c *Core) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	log.Printf("[%s] Shutting down...", CoreName)
	c.cancel()

	if c.proxyMgr != nil {
		c.proxyMgr.Close()
	}

	c.running = false
	log.Printf("[%s] Core stopped", CoreName)
	return nil
}

func (c *Core) WaitForSignal() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("[%s] Received signal: %v", CoreName, sig)
	c.Stop()
}

func (c *Core) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

func (c *Core) Config() *config.Config {
	return c.config
}

func (c *Core) Router() *router.Router {
	return c.router
}

func (c *Core) Uptime() time.Duration {
	return time.Since(c.startTime)
}

func (c *Core) Stats() *CoreStats {
	return c.stats
}

type DirectOutboundHandler struct {
	tag string
}

func (h *DirectOutboundHandler) Tag() string { return h.tag }

func (h *DirectOutboundHandler) Process(ctx context.Context, link *proxy.Link) error {
	return errors.New("direct outbound: no target specified")
}

type FreedomOutboundHandler struct {
	tag string
}

func (h *FreedomOutboundHandler) Tag() string { return h.tag }

func (h *FreedomOutboundHandler) Process(ctx context.Context, link *proxy.Link) error {
	go io.Copy(io.Discard, link.Reader)
	return nil
}

type BlackholeOutboundHandler struct {
	tag string
}

func (h *BlackholeOutboundHandler) Tag() string { return h.tag }

func (h *BlackholeOutboundHandler) Process(ctx context.Context, link *proxy.Link) error {
	go io.Copy(io.Discard, link.Reader)
	return nil
}

type DCCPOutboundProxyHandler struct {
	tag            string
	streamSettings *transport.StreamSettings
}

func (h *DCCPOutboundProxyHandler) Tag() string { return h.tag }

func (h *DCCPOutboundProxyHandler) Process(ctx context.Context, link *proxy.Link) error {
	if h.streamSettings == nil {
		h.streamSettings = &transport.StreamSettings{
			Network: transport.TransportDCCP,
			DCCPSettings: &transport.DCCPSettings{
				CCID:             dccp.CCID4,
				ServiceCode:      "V2RY",
				MaxPacketSize:    1500,
				HandshakeTimeout: 15 * time.Second,
				MaxRetries:       3,
			},
		}
	}

	targetAddr := &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 33445,
	}

	dccpTransport := transport.NewDCCPTransport(h.streamSettings)
	if err := dccpTransport.Dial(ctx, targetAddr); err != nil {
		return fmt.Errorf("dccp dial failed: %w", err)
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

func (c *Core) LoadConfig(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cfg, err := config.ParseConfig(data)
	if err != nil {
		return err
	}

	c.config = cfg

	if c.config.Routing != nil && c.router != nil {
		c.router.ReloadConfig(c.config.Routing)
	}

	return nil
}

func (c *Core) ExportConfig() ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return json.MarshalIndent(c.config, "", "  ")
}

func (c *Core) String() string {
	return fmt.Sprintf("%s v%s (uptime: %s)", CoreName, Version, c.Uptime().Round(time.Second))
}
