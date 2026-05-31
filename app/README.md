# Oiwest Core - Application Source

This directory contains the complete Go source code for Oiwest Core proxy engine.

## Directory Structure

| Directory | Description |
|---|---|
| `cmd/` | Entry points: CLI (`oiwest-core`) and Daemon (`oiwest-daemon`) |
| `common/` | Shared libraries: buffer pool, crypto, networking, platform detection, TLS |
| `config/` | Configuration parsing and validation |
| `core/` | Core engine: startup, shutdown, protocol registration, WorkerPool |
| `features/` | Advanced features: multiplexing (MuxSession), stealth/obfuscation |
| `proxy/` | Protocol implementations: VMess, VLESS, Trojan, Shadowsocks, SOCKS, HTTP, DCCP |
| `route/` | Routing engine: domain/IP/port/protocol matching, load balancing |
| `transport/` | Transport layer: TCP, mKCP, WebSocket, HTTP/2, QUIC, gRPC, XHTTP, DCCP, BBR |

## Building

```bash
# CLI
go build -o build/oiwest-core ./app/cmd/cli

# Daemon
go build -o build/oiwest-daemon ./app/cmd/daemon

# All packages
go vet ./app/...
```

## Architecture

```
                    ┌─────────────┐
                    │  cmd/cli    │  ← CLI entry point
                    │  cmd/daemon │  ← Headless daemon
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │    core     │  ← Engine: Start/Stop, protocol registration
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
       ┌──────▼──────┐ ┌──▼───┐ ┌──────▼──────┐
       │    proxy    │ │route │ │  transport  │
       │ (protocols) │ │      │ │ (transports)│
       └──────┬──────┘ └──────┘ └──────┬──────┘
              │                        │
       ┌──────▼────────────────────────▼──────┐
       │              common                   │
       │  buf │ crypto │ net │ tls │ platform  │
       └──────────────────────────────────────┘
```
