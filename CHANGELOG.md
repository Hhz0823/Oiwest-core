# Changelog

All notable changes to Oiwest Core will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.1.0] - 2025-05-31

### Performance Optimizations
- **Buffer Pool**: Increased default buffer from 2KB to 32KB, added 4-tier pooling (4K/8K/16K/32K) reducing syscall count ~16x
- **DCCP Transport**: Separated read/write locks (`readMu`/`writeMu`) for true concurrent I/O; atomic `closed` flag eliminates race conditions
- **WorkerPool**: Added idle timeout auto scale-down (minimum 25% workers alive) and auto scale-up on 75% queue pressure (up to 4x CPU)
- **WebSocket**: Buffer sizes increased from 4KB to 32KB; `Read()` now uses `NextReader()` stream mode avoiding per-message double allocation
- **gRPC**: Frame headers use stack-allocated `[5]byte`; writes combine header+data into single syscall; pre-allocated `writeBuf` for reuse
- **MuxSession**: `WriteFrame` uses pre-allocated `writeBuf`; `readLoop` reuses data buffer across frames; accept channel 16->32
- **BBR**: `sync.Mutex` replaced with `sync.RWMutex`; all read-only accessors (`CWND`, `PacingRate`, `MinRTT`, `BW`) use `RLock`
- **QUIC**: Accept channel increased from 16 to 64 for better burst handling
- **Stealth Obfuscators**: Added `obfBufPool` for crypto operation buffer reuse

### Stability Fixes
- **StatCounter**: `ReadBytes`/`WrittenBytes` now use `atomic.AddInt64`/`atomic.LoadInt64` preventing data races
- **DCCP Transport**: `markClosed()` uses `sync.Once` preventing double-close panics; `sendAck` snapshots ports safely
- **Proxy Manager**: Added connection semaphore (4096 per inbound) with graceful rejection; TCP KeepAlive enabled (30s)
- **DualStack Probe**: Fixed hardcoded port 80 — now uses actual address port (fallback 443); IPv6 uses `net.JoinHostPort`

### Build System
- Fixed CLI entry point path (`app/cmd/cli/main.go`)
- Fixed daemon entry point path (`app/cmd/daemon/main.go`)
- Fixed `adapters.go`: replaced undefined `net.Network` with `string` type
- Implemented 6 missing DCCP disguise methods: DTLS, WireGuard, HTTPS, DNS, HTTPUpgrade, TrafficShape
- Added missing proxy constructor functions (`NewDCCPInboundHandler`, `NewFreedomOutboundHandler`, etc.)
- All packages now pass `go vet` cleanly

### Documentation
- Created CHANGELOG.md
- Created CONTRIBUTING.md
- Updated .gitignore with comprehensive patterns

## [2.0.0] - 2025-01-01

### Added
- DCCP (RFC 4340) transport protocol support
- CCID2/CCID3/CCID4 congestion control
- 11 transport protocols: TCP, mKCP, WebSocket, HTTP/2, QUIC, gRPC, XHTTP, DCCP
- 18 platform targets
- WorkerPool with configurable workers/queue/retries
- BBR congestion control variants (BBR, BBRv2, BBRv3, BBRPlus, BBR-ECN, BBR-Adaptive, BBR-ProbeRTT)
- Dual-stack IPv4/IPv6 with multi-line parallel dialing
- TLS certificate auto-generation (RSA/ECDSA/Ed25519)
- Stealth/obfuscation: XTLS, Vision, Reality, Random Padding, XOR, UDP, DTLS, WireGuard
- Multiplexing: frame-level MuxSession with 128 concurrent streams
- Wails v2 + React GUI desktop application

### Protocols
- Inbound: VMess, VLESS, Trojan, Shadowsocks, SOCKS5, HTTP, Dokodemo-door, Loopback, DCCP
- Outbound: Freedom, Blackhole, Direct, DNS, VMess, VLESS, Trojan, Shadowsocks, SOCKS, DCCP

## [1.0.0] - 2024-06-01

### Added
- Initial release
- Basic proxy functionality
- CLI interface

