# Contributing to Oiwest Core

Thank you for your interest in contributing to Oiwest Core! This document provides guidelines and information for contributors.

## Getting Started

### Prerequisites

- Go 1.22.0 or later
- Git
- (Optional) Node.js 18+ for GUI development
- (Optional) Wails v2 for GUI builds

### Development Setup

```bash
# Clone the repository
git clone https://github.com/Hhz0823/oiwest-core.git
cd oiwest-core

# Download dependencies
go mod download

# Build CLI
go build -o build/oiwest-core ./app/cmd/cli

# Build Daemon
go build -o build/oiwest-daemon ./app/cmd/daemon

# Run tests
go test ./app/...

# Vet all packages
go vet ./app/...
```

## Project Structure

```
oiwest-core/
├── app/
│   ├── cmd/
│   │   ├── cli/          # CLI entry point
│   │   └── daemon/       # Daemon entry point (headless mode)
│   ├── common/
│   │   ├── buf/          # Buffer pool with tiered pooling
│   │   ├── crypto/       # AEAD ciphers, HKDF, session counters
│   │   ├── net/          # DualStack dialer, connection types
│   │   ├── platform/     # OS/distro detection, paths, signals
│   │   ├── protocol/     # Protocol sniffing (HTTP, TLS, DNS)
│   │   ├── serial/       # Serialization utilities
│   │   ├── signal/       # OS signal handling
│   │   └── tls/          # Certificate generation and management
│   ├── config/           # Configuration parsing
│   ├── core/             # Core engine, WorkerPool, ParallelExecutor
│   ├── features/
│   │   ├── multiplex/    # MuxSession frame-level multiplexing
│   │   └── stealth/      # Obfuscation (XTLS, Vision, Reality)
│   ├── proxy/
│   │   ├── adapters/     # Interface adapters
│   │   ├── blackhole/    # Blackhole outbound
│   │   ├── dccp/         # DCCP inbound/outbound
│   │   ├── dns/          # DNS outbound
│   │   ├── freedom/      # Freedom (direct) outbound
│   │   ├── http/         # HTTP proxy inbound
│   │   ├── shadowsocks/  # Shadowsocks protocol
│   │   ├── socks/        # SOCKS5 proxy inbound
│   │   ├── trojan/       # Trojan protocol
│   │   ├── vless/        # VLESS protocol
│   │   ├── vmess/        # VMess protocol
│   │   ├── proxy.go      # ProxyManager, handler interfaces
│   │   ├── registry.go   # Protocol registry, Dokodemo, Loopback
│   │   └── constructors.go # Constructor functions
│   ├── route/            # Routing engine (domain, IP, port matching)
│   └── transport/
│       ├── bbr/          # BBR congestion control variants
│       ├── dccp/         # DCCP protocol implementation
│       │   └── disguise/ # DCCP traffic disguise methods
│       ├── config.go     # Transport configuration types
│       ├── grpc.go       # gRPC transport
│       ├── mkcp.go       # mKCP transport
│       ├── quic.go       # QUIC transport
│       ├── transport.go  # DCCP transport with split locks
│       ├── websocket.go  # WebSocket transport
│       └── xhttp.go      # XHTTP transport
├── gui/                  # Wails v2 GUI application
├── O-ui/                 # Web-based management UI
├── Makefile              # Build targets for all platforms
├── build-all.ps1         # PowerShell build script
└── go.mod
```

## Coding Guidelines

### Go Style

- Follow standard Go conventions and `gofmt` formatting
- Use `golangci-lint` for linting
- Keep functions focused and under 50 lines when possible
- Document all exported types and functions

### Concurrency

- Use `sync.RWMutex` for read-heavy workloads (see BBR, DualStack)
- Use `atomic` operations for simple counters (see StatCounter, WorkerPool)
- Separate read/write locks for I/O operations (see DCCP Transport)
- Always use `sync.Once` for one-time initialization

### Memory Management

- Use `sync.Pool` for frequently allocated buffers (see buf package)
- Pre-allocate reusable buffers where possible (see MuxSession.writeBuf)
- Default buffer size for network I/O: 32KB

### Error Handling

- Return errors with context using `fmt.Errorf("context: %w", err)`
- Log warnings for non-fatal errors
- Use sentinel errors for known error types

### Platform Support

- Use build tags for platform-specific code
- Test on at least Linux, Windows, and macOS
- Consider resource constraints on ARM/embedded devices

## Submitting Changes

### Pull Request Process

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes following the coding guidelines
4. Run tests: `go test ./app/...`
5. Run vet: `go vet ./app/...`
6. Commit with descriptive messages
7. Push and create a Pull Request

### Commit Messages

Follow Conventional Commits:

```
type(scope): description

[optional body]

[optional footer]
```

Types: `feat`, `fix`, `perf`, `refactor`, `docs`, `test`, `chore`

Examples:
```
feat(transport): add HTTP/3 QUIC support
perf(buffer): increase default buffer size to 32KB
fix(dualstack): use actual port in probe instead of hardcoded 80
```

### Code Review

- All changes require review before merging
- Address review comments promptly
- Keep PRs focused on a single change

## Reporting Issues

- Use GitHub Issues for bug reports and feature requests
- Include Go version, OS, and architecture
- Provide minimal reproduction steps
- Include relevant log output

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
