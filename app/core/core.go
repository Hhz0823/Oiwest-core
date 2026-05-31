package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/Hhz0823/oiwest-core/app/common/net"
	"github.com/Hhz0823/oiwest-core/app/common/platform"
	"github.com/Hhz0823/oiwest-core/app/common/tls"
	"github.com/Hhz0823/oiwest-core/app/config"
	"github.com/Hhz0823/oiwest-core/app/proxy"
	"github.com/Hhz0823/oiwest-core/app/route"
	"github.com/Hhz0823/oiwest-core/app/transport/bbr"
)

const (
	Version    = "2.1.0"
	CoreName   = "Oiwest Core"
	APIVersion = 2
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
	workerPool  *WorkerPool
	bbrCtrl     bbr.BBRCongestionControl
	dualStack   *net.DualStackDialer
	multiLine   *net.MultiLineManager
	certMgr     *tls.CertificateManager
	executor    *ParallelExecutor
}

type CoreStats struct {
	UplinkBytes   int64
	DownlinkBytes int64
	ActiveConns   int64
	TotalConns    int64
	TasksQueued   int64
	TasksDone     int64
	mu            sync.Mutex
}

func NewCore(cfg *config.Config) *Core {
	ctx, cancel := context.WithCancel(context.Background())
	core := &Core{
		config:    cfg,
		ctx:       ctx,
		cancel:    cancel,
		startTime: time.Now(),
		stats:     &CoreStats{},
	}

	if cfg.WorkerPool != nil {
		wpc := DefaultWorkerPoolConfig()
		if cfg.WorkerPool.NumWorkers > 0 {
			wpc.NumWorkers = cfg.WorkerPool.NumWorkers
		}
		if cfg.WorkerPool.QueueSize > 0 {
			wpc.QueueSize = cfg.WorkerPool.QueueSize
		}
		core.workerPool = NewWorkerPool(wpc)
	}

	if cfg.BBR != nil && cfg.BBR.Enabled {
		bbrCfg := cfg.BBR.Settings
		if bbrCfg == nil {
			bbrCfg = bbr.DefaultBBRConfig(cfg.BBR.Algorithm)
		}
		core.bbrCtrl = bbr.GetBBRFactory().Create(cfg.BBR.Algorithm, bbrCfg)
		log.Printf("[%s] BBR congestion control initialized: %s", CoreName, core.bbrCtrl.Name())
	}

	if cfg.DualStack != nil && cfg.DualStack.Enabled {
		dsCfg := net.DefaultDualStackConfig()
		if cfg.DualStack.Config != nil {
			dsCfg = cfg.DualStack.Config
		}
		switch cfg.DualStack.Preference {
		case "ipv4":
			dsCfg.Preference = net.IPv4Only
		case "ipv6":
			dsCfg.Preference = net.IPv6Only
		case "dual":
			dsCfg.Preference = net.PreferDual
		case "prefer_ipv4":
			dsCfg.Preference = net.PreferIPv4
		case "prefer_ipv6":
			dsCfg.Preference = net.PreferIPv6
		}
		dsCfg.MultiLine = cfg.DualStack.MultiLine
		dsCfg.Failover = cfg.DualStack.Failover
		dsCfg.Strategy = cfg.DualStack.Strategy
		core.dualStack = net.NewDualStackDialer(dsCfg)
		core.multiLine = net.NewMultiLineManager(dsCfg)
		log.Printf("[%s] Dual-stack dialer initialized: %s", CoreName, cfg.DualStack.Preference)
	}

	if cfg.Certificate != nil && cfg.Certificate.Enabled {
		certCfg := cfg.Certificate.Config
		if certCfg == nil {
			certCfg = tls.DefaultCertificateConfig()
		}
		core.certMgr = tls.NewCertificateManager(certCfg)
		if cfg.Certificate.AutoGenerate {
			core.autoGenerateCertificates()
		}
	}

	core.executor = NewParallelExecutor(runtime.NumCPU() * 2)

	core.registerAllProtocols()

	return core
}

func (c *Core) autoGenerateCertificates() {
	commonName := "localhost"
	if c.config.Certificate.Config != nil && c.config.Certificate.Config.CommonName != "" {
		commonName = c.config.Certificate.Config.CommonName
	}
	go func() {
		_, err := c.certMgr.GetCertificate(commonName)
		if err != nil {
			log.Printf("[%s] Auto-generate certificate for '%s': %v", CoreName, commonName, err)
		} else {
			log.Printf("[%s] TLS certificate auto-generated for '%s'", CoreName, commonName)
		}
	}()
}

func (c *Core) registerAllProtocols() {
	proxy.RegisterInboundProtocol("vmess", func(ctx context.Context, cfg *config.InboundConfig, mgr *proxy.ProxyManager) (proxy.InboundHandler, error) {
		return createVMessInbound(cfg, mgr)
	})
	proxy.RegisterInboundProtocol("vless", func(ctx context.Context, cfg *config.InboundConfig, mgr *proxy.ProxyManager) (proxy.InboundHandler, error) {
		return createVLESSInbound(cfg, mgr)
	})
	proxy.RegisterInboundProtocol("trojan", func(ctx context.Context, cfg *config.InboundConfig, mgr *proxy.ProxyManager) (proxy.InboundHandler, error) {
		return createTrojanInbound(cfg, mgr)
	})
	proxy.RegisterInboundProtocol("shadowsocks", func(ctx context.Context, cfg *config.InboundConfig, mgr *proxy.ProxyManager) (proxy.InboundHandler, error) {
		return createShadowsocksInbound(cfg, mgr)
	})
	proxy.RegisterInboundProtocol("socks", func(ctx context.Context, cfg *config.InboundConfig, mgr *proxy.ProxyManager) (proxy.InboundHandler, error) {
		return proxy.NewSOCKSInboundHandler(cfg.Tag, cfg.Port, cfg.Listen, mgr), nil
	})
	proxy.RegisterInboundProtocol("http", func(ctx context.Context, cfg *config.InboundConfig, mgr *proxy.ProxyManager) (proxy.InboundHandler, error) {
		return proxy.NewHTTPInboundHandler(cfg.Tag, cfg.Port, cfg.Listen, mgr), nil
	})
	proxy.RegisterInboundProtocol("dokodemo-door", proxy.CreateDokodemoInbound)
	proxy.RegisterInboundProtocol("loopback", proxy.CreateLoopbackInbound)
	proxy.RegisterInboundProtocol("dccp", func(ctx context.Context, cfg *config.InboundConfig, mgr *proxy.ProxyManager) (proxy.InboundHandler, error) {
		return proxy.NewDCCPInboundHandler(cfg.Tag, cfg.Port, cfg.Listen, cfg.StreamSettings, mgr), nil
	})

	proxy.RegisterOutboundProtocol("freedom", func(ctx context.Context, cfg *config.OutboundConfig) (proxy.OutboundHandler, error) {
		return proxy.NewFreedomOutboundHandler(cfg.Tag), nil
	})
	proxy.RegisterOutboundProtocol("blackhole", func(ctx context.Context, cfg *config.OutboundConfig) (proxy.OutboundHandler, error) {
		return proxy.NewBlackholeOutboundHandler(cfg.Tag), nil
	})
	proxy.RegisterOutboundProtocol("direct", func(ctx context.Context, cfg *config.OutboundConfig) (proxy.OutboundHandler, error) {
		return proxy.NewDirectOutboundHandler(cfg.Tag), nil
	})
	proxy.RegisterOutboundProtocol("dns", proxy.CreateDNSOutbound)
	proxy.RegisterOutboundProtocol("vmess", func(ctx context.Context, cfg *config.OutboundConfig) (proxy.OutboundHandler, error) {
		return createVMessOutbound(cfg)
	})
	proxy.RegisterOutboundProtocol("vless", func(ctx context.Context, cfg *config.OutboundConfig) (proxy.OutboundHandler, error) {
		return createVLESSOutbound(cfg)
	})
	proxy.RegisterOutboundProtocol("trojan", func(ctx context.Context, cfg *config.OutboundConfig) (proxy.OutboundHandler, error) {
		return createTrojanOutbound(cfg)
	})
	proxy.RegisterOutboundProtocol("shadowsocks", func(ctx context.Context, cfg *config.OutboundConfig) (proxy.OutboundHandler, error) {
		return createShadowsocksOutbound(cfg)
	})
	proxy.RegisterOutboundProtocol("socks", func(ctx context.Context, cfg *config.OutboundConfig) (proxy.OutboundHandler, error) {
		return proxy.NewSocksOutboundHandler(cfg.Tag), nil
	})
	proxy.RegisterOutboundProtocol("dccp", func(ctx context.Context, cfg *config.OutboundConfig) (proxy.OutboundHandler, error) {
		return proxy.NewDCCPOutboundHandler(cfg.Tag, net.Address{}, cfg.StreamSettings), nil
	})
}

func createVMessInbound(cfg *config.InboundConfig, mgr *proxy.ProxyManager) (proxy.InboundHandler, error) {
	return proxy.NewVmessInboundHandler(cfg.Tag, cfg.Port, cfg.Listen, cfg.Settings, cfg.StreamSettings, mgr), nil
}

func createVLESSInbound(cfg *config.InboundConfig, mgr *proxy.ProxyManager) (proxy.InboundHandler, error) {
	return proxy.NewVlessInboundHandler(cfg.Tag, cfg.Port, cfg.Listen, cfg.Settings, cfg.StreamSettings, mgr), nil
}

func createTrojanInbound(cfg *config.InboundConfig, mgr *proxy.ProxyManager) (proxy.InboundHandler, error) {
	return proxy.NewTrojanInboundHandler(cfg.Tag, cfg.Port, cfg.Listen, cfg.Settings, cfg.StreamSettings, mgr), nil
}

func createShadowsocksInbound(cfg *config.InboundConfig, mgr *proxy.ProxyManager) (proxy.InboundHandler, error) {
	return proxy.NewShadowsocksInboundHandler(cfg.Tag, cfg.Port, cfg.Listen, cfg.Settings, cfg.StreamSettings, mgr), nil
}

func createVMessOutbound(cfg *config.OutboundConfig) (proxy.OutboundHandler, error) {
	return &proxy.VMessOutboundHandler{}, nil
}

func createVLESSOutbound(cfg *config.OutboundConfig) (proxy.OutboundHandler, error) {
	return &proxy.VLESSOutboundHandler{}, nil
}

func createTrojanOutbound(cfg *config.OutboundConfig) (proxy.OutboundHandler, error) {
	return &proxy.TrojanOutboundHandler{}, nil
}

func createShadowsocksOutbound(cfg *config.OutboundConfig) (proxy.OutboundHandler, error) {
	return &proxy.ShadowsocksOutboundHandler{}, nil
}

func (c *Core) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return errors.New("core already running")
	}

	log.Printf("[%s] Starting %s v%s", CoreName, CoreName, Version)
	log.Printf("[%s] Go version: %s | OS/Arch: %s/%s", CoreName, runtime.Version(), runtime.GOOS, runtime.GOARCH)

	c.proxyMgr = proxy.NewProxyManager(c.config)

	if c.config.Routing != nil {
		c.router = router.NewRouter(c.config.Routing)
		log.Printf("[%s] Router initialized with %d rules", CoreName, len(c.config.Routing.Rules))
	} else {
		c.router = router.NewRouter(&config.RoutingConfig{})
	}

	c.registerDefaultOutbounds()

	if c.workerPool != nil {
		log.Printf("[%s] Worker pool: %d workers", CoreName, c.workerPool.NumWorkers())
	}

	if err := c.proxyMgr.Start(); err != nil {
		return fmt.Errorf("failed to start proxy manager: %w", err)
	}

	c.startBackgroundTasks()

	c.running = true
	log.Printf("[%s] Core v%s started successfully", CoreName, Version)
	log.Printf("[%s] Protocols: VMess, VLESS, Trojan, Shadowsocks, SOCKS, HTTP, Dokodemo, Loopback, DNS, DCCP", CoreName)
	log.Printf("[%s] Transports: TCP, mKCP, WebSocket, HTTP/2, QUIC, gRPC, XHTTP, DCCP", CoreName)
	log.Printf("[%s] Features: WorkerPool, BBR, DualStack, TLS, Stealth", CoreName)

	return nil
}

func (c *Core) startBackgroundTasks() {
	if c.workerPool != nil {
		c.executor.Execute(func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					c.stats.mu.Lock()
					c.stats.TasksQueued = int64(c.workerPool.Pending())
					c.stats.TasksDone = c.workerPool.Completed()
					c.stats.mu.Unlock()
				case <-c.ctx.Done():
					return
				}
			}
		})
	}
	if c.multiLine != nil && c.config.DualStack.MultiLine {
		c.multiLine.StartProbing(30 * time.Second)
	}
	if c.bbrCtrl != nil {
		c.executor.Execute(func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					c.bbrCtrl.OnRTTUpdate(time.Second)
				case <-c.ctx.Done():
					return
				}
			}
		})
	}
}

func (c *Core) registerDefaultOutbounds() {
	directHandler := proxy.NewDirectOutboundHandler("direct")
	c.proxyMgr.RegisterOutbound(directHandler)

	freedomHandler := proxy.NewFreedomOutboundHandler("freedom")
	c.proxyMgr.RegisterOutbound(freedomHandler)

	blackholeHandler := proxy.NewBlackholeOutboundHandler("blackhole")
	c.proxyMgr.RegisterOutbound(blackholeHandler)

	dnsHandler := proxy.NewDNSOutboundHandler("dns", "udp", "8.8.8.8", 53)
	c.proxyMgr.RegisterOutbound(dnsHandler)

	for _, outbound := range c.config.Outbounds {
		handler, err := c.createOutboundHandler(&outbound)
		if err != nil {
			log.Printf("[%s] Warning: failed to create outbound '%s': %v", CoreName, outbound.Tag, err)
			continue
		}
		c.proxyMgr.RegisterOutbound(handler)
		log.Printf("[%s] Outbound '%s' registered (protocol: %s)", CoreName, outbound.Tag, outbound.Protocol)
	}
}

func (c *Core) createOutboundHandler(outbound *config.OutboundConfig) (proxy.OutboundHandler, error) {
	factory, err := proxy.GetOutboundFactory(proxy.ProtocolType(outbound.Protocol))
	if err == nil {
		return factory(c.ctx, outbound)
	}
	switch outbound.Protocol {
	case "dccp":
		return proxy.NewDCCPOutboundHandler(outbound.Tag, net.Address{}, outbound.StreamSettings), nil
	case "freedom":
		return proxy.NewFreedomOutboundHandler(outbound.Tag), nil
	case "blackhole":
		return proxy.NewBlackholeOutboundHandler(outbound.Tag), nil
	case "dns":
		return proxy.NewDNSOutboundHandler(outbound.Tag, "udp", "8.8.8.8", 53), nil
	default:
		return proxy.NewDirectOutboundHandler(outbound.Tag), nil
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
	if c.workerPool != nil {
		c.workerPool.Shutdown()
	}
	if c.multiLine != nil {
		c.multiLine.Stop()
	}
	if c.dualStack != nil {
		c.dualStack.Close()
	}

	c.running = false
	log.Printf("[%s] Core stopped", CoreName)
	return nil
}

func (c *Core) WaitForSignal() {
	sig := platform.WaitForSignal()
	log.Printf("[%s] Received signal: %v", CoreName, sig)
	c.Stop()
}

func (c *Core) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

func (c *Core) Config() *config.Config { return c.config }
func (c *Core) Router() *router.Router { return c.router }
func (c *Core) Uptime() time.Duration  { return time.Since(c.startTime) }
func (c *Core) Stats() *CoreStats      { return c.stats }
func (c *Core) WorkerPool() *WorkerPool { return c.workerPool }
func (c *Core) BBR() bbr.BBRCongestionControl { return c.bbrCtrl }
func (c *Core) DualStack() *net.DualStackDialer { return c.dualStack }
func (c *Core) CertManager() *tls.CertificateManager { return c.certMgr }

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

