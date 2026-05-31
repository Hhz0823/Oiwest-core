# transport/ — Transport Layer

All transport protocol implementations.

## Supported Transports

| Transport | File | Protocol | Description |
|---|---|---|---|
| TCP | `config.go` | Standard TCP | HTTP header masquerading |
| mKCP | `mkcp.go` | KCP over UDP | Reliable UDP with FEC |
| WebSocket | `websocket.go` | WS/WSS | 32KB buffers, stream reader |
| HTTP/2 | (via gRPC) | H2 | Multiplexed streams |
| QUIC | `quic.go` | QUIC | 0-RTT, 8MB window, 64-ch accept |
| gRPC | `grpc.go` | gRPC | Combined header+data writes |
| XHTTP | `xhttp.go` | Custom | Frame protocol over HTTP |
| DCCP | `transport.go` | RFC 4340 | Split read/write locks, atomic state |

## transport.go — DCCP Transport (Optimized v2.1.0)

**Key optimizations:**
- **Split locks**: `readMu` + `writeMu` — reads and writes never block each other
- **Atomic closed flag**: `int32` with `atomic.LoadInt32`/`atomic.StoreInt32`
- **sync.Once close**: `markClosed()` prevents double-close panics
- **Safe ACK**: `sendAck()` snapshots ports under lock before sending

## bbr/ — BBR Congestion Control

7 BBR algorithm variants:

| Variant | Description |
|---|---|
| `bbr` | Original BBR |
| `bbrv2` | BBR v2 with ECN awareness |
| `bbrv3` | BBR v3 |
| `bbrplus` | BBR Plus |
| `bbr_ecn` | BBR with ECN |
| `bbr_adaptive` | Adaptive BBR |
| `bbr_probert` | BBR ProbeRTT |

**Optimized**: Uses `sync.RWMutex` — read-only methods (`CWND`, `PacingRate`, `MinRTT`, `BW`) use `RLock`.

## dccp/ — DCCP Protocol

| File | Description |
|---|---|
| `dccp.go` | Protocol core |
| `packet.go` | Packet encode/decode |
| `handshake.go` | 3-way handshake |
| `congestion.go` | CCID2/CCID3/CCID4 |
| `options.go` | DCCP options |
| `transport.go` | DCCP transport wrapper |

## dccp/disguise/ — Traffic Disguise

12 disguise methods making DCCP traffic appear as other protocols:

| Method | Disguise As |
|---|---|
| `none` | Raw DCCP |
| `tls` | HTTPS (TLS 1.3) |
| `websocket` | WebSocket |
| `http2` | HTTP/2 |
| `grpc` | gRPC |
| `domain_fronting` | CDN domain fronting |
| `dtls` | DTLS |
| `wireguard` | WireGuard |
| `https` | Full HTTPS session |
| `dns` | DNS traffic |
| `http_upgrade` | HTTP Upgrade |
| `traffic_shape` | Traffic shaping |

