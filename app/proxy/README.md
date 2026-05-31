# proxy/ — Protocol Implementations

All proxy protocol inbound/outbound handlers.

## Architecture

- `proxy.go` — ProxyManager, handler interfaces, core handler implementations
- `registry.go` — Protocol factory registry, Dokodemo/Loopback/DNS handlers
- `constructors.go` — Constructor functions for all handler types

## Inbound Protocols

| Protocol | File | Description |
|---|---|---|
| VMess | `vmess/vmess.go` | VMess with AEAD encryption |
| VLESS | `vless/vless.go` | VLESS with XTLS flow support |
| Trojan | `trojan/trojan.go` | Trojan with SHA-224 password hash |
| Shadowsocks | `shadowsocks/ss.go` | SS with AES-GCM/ChaCha20 |
| SOCKS5 | `proxy.go` | SOCKS5 proxy |
| HTTP | `proxy.go` | HTTP CONNECT proxy |
| Dokodemo | `registry.go` | Transparent proxy |
| Loopback | `registry.go` | Internal loopback |
| DCCP | `proxy.go` | DCCP inbound |

## Outbound Protocols

| Protocol | File | Description |
|---|---|---|
| Freedom | `proxy.go` | Direct connection |
| Blackhole | `proxy.go` | Drop all traffic |
| Direct | `proxy.go` | Direct outbound |
| DNS | `registry.go` | DNS forwarding |
| DCCP | `proxy.go` | DCCP outbound |

## Connection Management

- **Connection semaphore**: 4096 max concurrent connections per inbound
- **TCP KeepAlive**: 30-second interval
- **Graceful rejection**: Connections beyond limit are closed immediately

## adapters/ — Interface Adapters

Defines abstract interfaces (`Inbound`, `OutboundHandler`, `NetDialer`, `InboundManager`, `OutboundManager`) for the proxy system.
