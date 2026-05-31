# config/ — Configuration

JSON-based configuration parser compatible with v2ray/Xray format.

## Config Structure

```json
{
  "log": { "loglevel": "warning" },
  "inbounds": [...],
  "outbounds": [...],
  "routing": { "rules": [...] },
  "workerPool": { "numWorkers": 8, "queueSize": 1000 },
  "bbr": { "enabled": true, "algorithm": "bbr" },
  "dualStack": { "enabled": true, "preference": "dual" },
  "certificate": { "enabled": true, "autoGenerate": true }
}
```

## Key Types

| Type | Description |
|---|---|
| `Config` | Root configuration |
| `InboundConfig` | Inbound proxy (tag, port, protocol, settings, streamSettings) |
| `OutboundConfig` | Outbound proxy (tag, protocol, settings, streamSettings) |
| `RoutingConfig` | Routing rules and balancers |
| `WorkerPoolConfig` | Worker pool tuning |
| `BBRGlobalConfig` | BBR congestion control settings |
| `DualStackConfig` | IPv4/IPv6 dual-stack settings |

## Functions

- `ParseConfig(data []byte) (*Config, error)` — parse JSON config
- `DefaultConfig() *Config` — default DCCP config on port 33445
- `Config.ToJSON() ([]byte, error)` — serialize to JSON
