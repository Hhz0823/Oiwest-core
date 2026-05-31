# core/ — Core Engine

The central engine that orchestrates all components.

## core.go — Engine Core

**Core struct** manages:
- `ProxyManager` — inbound/outbound proxy handlers
- `Router` — routing engine
- `WorkerPool` — task pool
- `BBRCongestionControl` — congestion control
- `DualStackDialer` — IPv4/IPv6 dialing
- `CertificateManager` — TLS certificates
- `ParallelExecutor` — concurrent task execution

### Lifecycle

```go
core := core.NewCore(cfg)    // Initialize all components
core.Start()                  // Start proxy manager, register protocols
core.WaitForSignal()          // Block until SIGINT/SIGTERM
core.Stop()                   // Graceful shutdown
```

### Protocol Registration

`registerAllProtocols()` registers all inbound/outbound factories:
- Inbound: VMess, VLESS, Trojan, Shadowsocks, SOCKS, HTTP, Dokodemo, Loopback, DCCP
- Outbound: Freedom, Blackhole, Direct, DNS, VMess, VLESS, Trojan, Shadowsocks, SOCKS, DCCP

## worker.go — Worker Pool

Dynamic worker pool with auto-scaling.

| Feature | Description |
|---|---|
| Idle Timeout | Workers exit after 30s idle (min 25% alive) |
| Auto Scale-Up | Queue >75% → spawn workers (max 4x CPU) |
| Task Scheduler | Priority-based task scheduling |
| ParallelExecutor | Semaphore-bounded concurrent execution |

```go
pool := NewWorkerPool(config)
pool.Submit(func(ctx context.Context) error { ... })
pool.Shutdown()
```
