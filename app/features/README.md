# features/ — Advanced Features

## multiplex/ — MuxSession

Frame-level multiplexing over a single connection.

**Features:**
- Up to 128 concurrent streams per session
- Frame types: Data, Close, Reset, Ping, Pong, GoAway
- Pre-allocated write buffer for zero-alloc frame writing
- Reusable data buffer in read loop
- KeepAlive ping/pong

**Frame format:**
```
[StreamID:4][Flags:1][Reserved:1][Length:2][Payload:N]
```

```go
session := multiplex.NewMuxSession(conn, nil)
stream, _ := session.OpenStream()
stream.Write(data)
```

## stealth/ — Obfuscation & Anti-Detection

| Method | Description |
|---|---|
| `random` | XOR-based data obfuscation |
| `random_padding` | Random padding to mask traffic patterns |
| `xtls` | XTLS with flow control and splitting |
| `vision` | XTLS Vision obfuscation |
| `reality` | Reality protocol with shortID authentication |

**Additional files:**
- `reality.go` — Reality handshake and authentication
- `vision.go` — Vision flow control
- `xtls.go` — XTLS state machine
