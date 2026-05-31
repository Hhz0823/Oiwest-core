# common/ - Shared Libraries

Foundational libraries used across the entire codebase.

## buf/ — Buffer Pool

High-performance buffer pool with tiered pooling for network I/O.

**Key design:**
- Default buffer: **32KB** (optimized for proxy traffic)
- 4-tier `sync.Pool`: 4K / 8K / 16K / 32K
- `New()` — get pooled buffer, `Release()` — return to pool
- `NewWithSize(n)` — get buffer of specific size from appropriate tier

```go
b := buf.New()        // Get 32KB buffer from pool
defer b.Release()     // Return to pool
b.Write(data)
```

## crypto/ — Cryptographic Primitives

AEAD ciphers, HKDF key derivation, session counters.

| Cipher | Key Size |
|---|---|
| AES-128-GCM | 16 bytes |
| AES-256-GCM | 32 bytes |
| ChaCha20-Poly1305 | 32 bytes |
| XChaCha20-Poly1305 | 32 bytes |

## net/ — Networking

- **Address** — IPv4/IPv6/Domain address abstraction
- **Connection** — `StatConnection` with atomic byte counters
- **DualStackDialer** — IPv4/IPv6 parallel dialing with strategy (latency/random/roundrobin)
- **MultiLineManager** — Multi-line probing and failover

## platform/ — Platform Detection

Build-tag separated platform support:

| File | Platform |
|---|---|
| `detect_windows.go` / `detect_linux.go` / `detect_darwin.go` / `detect_android.go` | OS detection |
| `paths_*.go` | Config/data/cache directory paths |
| `signal_*.go` | OS signal handling (SIGINT/SIGTERM) |
| `net_*.go` | Socket options (SO_REUSEADDR, TCP_FASTOPEN) |

## tls/ — Certificate Management

- Auto-generate RSA/ECDSA/Ed25519 certificates
- CA signing support
- Auto-renewal before expiry
- Thread-safe certificate cache

## protocol/ — Protocol Sniffing

Detects HTTP, TLS (with SNI extraction), and DNS protocols from initial bytes.
