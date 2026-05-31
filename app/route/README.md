# route/ — Routing Engine

Rule-based routing engine for directing traffic to appropriate outbound handlers.

## Routing Rules

Rules are matched in order. First match wins.

| Field | Match Type | Example |
|---|---|---|
| `domain` | Full/Keyword/Regex | `"full:google.com"`, `"keyword:google"`, `"regexp:.*\\.google\\..*"` |
| `ip` | CIDR/Exact | `"10.0.0.0/8"`, `"192.168.1.1"` |
| `port` | Exact/Range | `"443"`, `"80-90"` |
| `network` | tcp/udp | `"tcp"` |
| `protocol` | http/tls/bittorrent | `"tls"` |
| `inboundTag` | Tag match | `"socks-in"` |
| `source` | Source IP/CIDR | `"192.168.0.0/16"` |

## Load Balancing

```json
{
  "balancers": [{
    "tag": "balancer-1",
    "selector": ["proxy-a", "proxy-b"],
    "strategy": { "type": "random" }
  }]
}
```

## Hot Reload

```go
router.ReloadConfig(newRoutingConfig)  // Thread-safe config reload
```
