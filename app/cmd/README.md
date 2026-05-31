# cmd/ - Entry Points

Contains the two main entry points for Oiwest Core.

## cli/

CLI command-line interface (`oiwest-core`).

```bash
# Run with config file
oiwest-core -config config.json

# Run with default config
oiwest-core -test

# Show version
oiwest-core -version

# Debug mode
oiwest-core -debug -config config.json
```

**Flags:**
| Flag | Default | Description |
|---|---|---|
| `-config` | `config.json` | Path to config file |
| `-version` | `false` | Show version info |
| `-test` | `false` | Use default config |
| `-debug` | `false` | Enable debug logging |

## daemon/

Headless daemon mode (`oiwest-daemon`).

- **Windows**: `daemon_windows.go` — uses platform paths (APPDATA)
- **Linux/macOS/Android**: `daemon_unix.go` — uses XDG paths

**Features:**
- Auto-detects platform (OS, Arch, Distro)
- Auto-creates config directory and default config
- Writes PID file for service management
- Graceful shutdown on SIGINT/SIGTERM

```bash
# Run daemon
oiwest-daemon

# Check if running (Linux)
cat ~/.local/share/oiwest-core/oiwest-core.pid
```
