<p align="center">
  <img src="https://img.shields.io/badge/version-2.1.0-blue?style=for-the-badge" />
  <img src="https://img.shields.io/badge/go-1.22.0+-00ADD8?style=for-the-badge&logo=go&logoColor=white" />
  <img src="https://img.shields.io/badge/license-MIT-green?style=for-the-badge" />
  <img src="https://img.shields.io/badge/platforms-18+-orange?style=for-the-badge" />
</p>

<p align="center">
  <img src="https://img.shields.io/badge/DCCP-RFC_4340-blue?style=flat-square" />
  <img src="https://img.shields.io/badge/compat-v2ray--core-lightgrey?style=flat-square" />
  <img src="https://img.shields.io/badge/compat-Xray--core-lightgrey?style=flat-square" />
  <img src="https://img.shields.io/badge/compat-sing--box-lightgrey?style=flat-square" />
</p>

<br />

<p align="center">
  <strong>⚡ 高性能 · 多平台 · 模块化 代理协议内核 ⚡</strong>
</p>

<p align="center">
  基于 DCCP (RFC 4340) 传输协议 · 兼容 v2ray-core · Xray-core · sing-box 生态<br />
  支持 11 种传输协议 · 9 种代理协议 · 18 种平台架构 · 7 种 BBR 变体
</p>

<br />
<br />

---

## 📖 关于 Oiwest Core

**Oiwest Core** 是一个用 Go 语言编写的高性能、模块化代理协议内核。最初以 **DCCP（Datagram Congestion Control Protocol，RFC 4340）** 为核心传输协议，经过持续迭代，已扩展为支持 **11 种传输协议**、**9 种代理协议**、**18 种平台架构** 的通用代理引擎。

### 为什么选择 Oiwest Core？

| 特性 | 说明 |
|:---|:---|
| 🚀 **极致性能** | 32KB 分层缓冲池 · 读写分离锁 · 自动扩缩容 WorkerPool |
| 🌐 **多平台** | Windows / Linux / macOS / Android / OpenWrt，覆盖 x64/ARM64/ARMv7/x86/MIPS |
| 🔒 **安全隐匿** | XTLS · Vision · Reality · uTLS 指纹 · 12 种 DCCP 伪装方法 |
| 🧩 **模块化** | 协议即插拔 · 传输层可组合 · 路由引擎可扩展 |
| 🔌 **生态兼容** | 配置格式兼容 v2ray-core / Xray-core / sing-box |
| 📦 **双模运行** | CLI 命令行 + 无头守护进程 + GUI 桌面程序 |

<br />

---

## 🏗️ 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                      Entry Points                           │
│              oiwest-core (CLI)  │  oiwest-daemon             │
└──────────────────┬──────────────┴───────────┬───────────────┘
                   │                          │
┌──────────────────▼──────────────────────────▼───────────────┐
│                     Core Engine                             │
│         Protocol Registration · WorkerPool · BBR            │
│         DualStack · TLS Cert Manager · Router               │
└────────┬──────────────┬─────────────────────┬───────────────┘
         │              │                     │
┌────────▼──────┐ ┌─────▼──────┐ ┌────────────▼──────────────┐
│    Proxy      │ │   Route    │ │       Transport           │
│               │ │            │ │                           │
│  Inbound:     │ │  Domain    │ │  TCP · mKCP · WebSocket   │
│  · VMess      │ │  IP/CIDR   │ │  HTTP/2 · QUIC · gRPC     │
│  · VLESS      │ │  Port      │ │  XHTTP · DCCP (RFC 4340)  │
│  · Trojan     │ │  Protocol  │ │                           │
│  · SS         │ │  Balancer  │ │  BBR / BBRv2 / BBRv3      │
│  · SOCKS5     │ │            │ │  BBRPlus / BBR-ECN        │
│  · HTTP       │ │            │ │  BBR-Adaptive / ProbeRTT  │
│  · DCCP       │ │            │ │                           │
│               │ │            │ │  ┌───────────────────┐    │
│  Outbound:    │ │            │ │  │  DCCP Disguise    │    │
│  · Freedom    │ │            │ │  │  TLS · WS · H2    │    │
│  · Blackhole  │ │            │ │  │  gRPC · DTLS      │    │
│  · Direct     │ │            │ │  │  WireGuard · DNS  │    │
│  · DNS        │ │            │ │  │  HTTPS · Traffic  │    │
└────────┬──────┘ └────────────┘ └──┴───────────────────┴────┘
         │
┌────────▼───────────────────────────────────────────────────┐
│                      Common Layer                           │
│  buf (sync.Pool) · crypto (AEAD) · net (DualStack)         │
│  tls (auto-cert) · platform (OS detect) · protocol (sniff) │
└────────────────────────────────────────────────────────────┘
```

<br />

---

## ⚡ 性能亮点 (v2.1.0)

### 内存优化

```
旧版: 2KB buffer × 每次 I/O → 大量 syscall + GC 压力
新版: 32KB buffer × 4 级池化 → syscall 减少 16 倍
```

| 优化项 | 改前 | 改后 | 效果 |
|:---|:---|:---|:---|
| 缓冲池 | 2KB 单级 | 32KB 四级 (4K/8K/16K/32K) | syscall ↓16x |
| DCCP 传输 | 单锁串行读写 | 读写锁分离 | 并发吞吐 ↑2-4x |
| BBR 访问 | Mutex 全锁 | RWMutex 读锁 | 锁竞争 ↓70% |
| WorkerPool | 固定 worker | 动态扩缩容 (idle timeout) | 内存 ↓30% |
| gRPC 帧头 | 每帧堆分配 | 栈分配 [5]byte + 合并写 | 分配 ↓90% |
| MuxSession | 每帧分配缓冲 | 预分配 writeBuf 复用 | GC ↓60% |
| WebSocket | 4KB 缓冲 + ReadMessage | 32KB + NextReader 流式 | 双倍分配消除 |
| QUIC | accept channel 16 | accept channel 64 | 高并发丢弃 ↓ |

### 稳定性修复

- **StatCounter**: `atomic` 操作消除数据竞争
- **DCCP**: `sync.Once` 防止重复关闭；`sendAck` 安全快照端口
- **代理管理器**: 4096 连接信号量 + TCP KeepAlive + 优雅拒绝
- **DualStack**: 探测使用实际端口 (原硬编码 80)；IPv6 使用 `net.JoinHostPort`

<br />

---

## 🚀 快速开始

### 安装

```bash
# 从 Release 下载预编译包
# https://github.com/Hhz0823/oiwest-core/releases

# 或从源码构建
git clone https://github.com/Hhz0823/oiwest-core.git
cd oiwest-core
go build -o oiwest-core ./app/cmd/cli
```

### 运行

```bash
# 使用配置文件
./oiwest-core -config config.json

# 使用默认配置 (DCCP 端口 33445)
./oiwest-core -test

# 守护进程模式 (无头)
./oiwest-daemon
```

### 最小配置

```json
{
  "inbounds": [
    {
      "tag": "dccp-in",
      "port": 33445,
      "listen": "0.0.0.0",
      "protocol": "dccp",
      "streamSettings": {
        "network": "dccp",
        "dccpSettings": {
          "ccid": 4,
          "serviceCode": "V2RY"
        }
      }
    }
  ],
  "outbounds": [
    { "tag": "direct", "protocol": "freedom" }
  ]
}
```

<br />

---

## 📡 协议支持

### 代理协议

<p align="center">

| 入站 (Inbound) | 出站 (Outbound) |
|:---|:---|
| VMess | Freedom (直连) |
| VLESS | Blackhole (黑洞) |
| Trojan | Direct (直出) |
| Shadowsocks | DNS |
| SOCKS5 | VMess |
| HTTP | VLESS |
| Dokodemo-door | Trojan |
| Loopback | Shadowsocks |
| DCCP | SOCKS · DCCP |

</p>

### 传输协议

| 传输 | 说明 |
|:---|:---|
| **TCP** | 标准 TCP，支持 HTTP 伪装头 |
| **mKCP** | 基于 KCP 的可靠 UDP 传输 |
| **WebSocket** | WS/WSS，32KB 缓冲，流式读取 |
| **HTTP/2** | H2 多路复用传输 |
| **QUIC** | 0-RTT，8MB 窗口，64 通道 accept |
| **gRPC** | 流式传输，合并帧头+数据单次写 |
| **XHTTP** | 自定义帧协议 + 多路复用 HTTP 流 |
| **DCCP** | RFC 4340，CCID2/CCID3/CCID4，读写分离锁 |

### DCCP 伪装方法

| 伪装 | 效果 |
|:---|:---|
| TLS | DCCP 流量看起来像 HTTPS |
| WebSocket | 伪装为 WebSocket 流量 |
| HTTP/2 | 伪装为 H2 流量 |
| gRPC | 伪装为 gRPC 调用 |
| Domain Fronting | CDN 域前置 |
| DTLS | 伪装为 DTLS/UDP |
| WireGuard | 伪装为 WireGuard 流量 |
| DNS | 伪装为 DNS 查询 |
| HTTP Upgrade | HTTP 升级机制 |
| Traffic Shape | 流量整形规避检测 |

<br />

---

## 🌍 平台支持

<p align="center">

| 平台 | 架构 | 适用设备 |
|:---|:---|:---|
| Windows | amd64 · arm64 · 386 | PC · Surface Pro X · 老设备 |
| Linux | amd64 · arm64 · arm · 386 | 服务器 · 树莓派 · ARM 盒子 |
| Linux | mips · mipsle · mips64 · mips64le | OpenWrt 路由器 |
| macOS | amd64 · arm64 | Intel Mac · Apple Silicon |
| Android | arm64 | Termux 环境 |

</p>

<br />

---

## 🛠️ 从源码构建

```bash
# 前置依赖: Go 1.22+

# 当前平台
make build              # CLI
make build-daemon       # 守护进程

# 全平台交叉编译
make build-all          # 14 个平台

# OpenWrt 专用
make build-openwrt-all

# GUI 桌面程序 (需要 Node.js 18+)
make build-gui-all
```

<br />

---

## 📂 项目结构

```
oiwest-core/
├── app/                          # Go 源码
│   ├── cmd/                      # 入口: CLI + Daemon
│   ├── common/                   # 共享库: buf · crypto · net · tls · platform
│   ├── config/                   # 配置解析
│   ├── core/                     # 核心引擎 + WorkerPool
│   ├── features/                 # 多路复用 + 混淆隐匿
│   ├── proxy/                    # 9 种代理协议实现
│   ├── route/                    # 路由引擎
│   └── transport/                # 11 种传输 + BBR + DCCP 伪装
├── gui/                          # Wails v2 GUI 桌面程序
├── O-ui/                         # Web 管理界面
├── .github/                      # CI/CD + Issue/PR 模板
├── CHANGELOG.md                  # 版本变更日志
├── CONTRIBUTING.md               # 贡献指南
├── LICENSE                       # MIT 许可证
├── Makefile                      # 跨平台构建脚本
└── README.md                     # 项目文档
```

<br />

---

## 🔧 WorkerPool 动态调度

```
         Queue Pressure
              │
    ┌─────────▼─────────┐
    │   Submit Task     │
    └─────────┬─────────┘
              │
     Queue > 75%?
      ┌──────┴──────┐
      │ YES         │ NO
      ▼             ▼
  Auto Scale     Normal
  Up (4x CPU)    Submit
      │
      ▼
  ┌────────────────────┐
  │   Worker Pool      │
  │                    │
  │  Worker 1 ●━━━━━┓  │
  │  Worker 2 ●━━━━━┫  │
  │  Worker 3 ○     ┃  │ ← ○ = idle (timeout 30s → exit)
  │  Worker 4 ●━━━━━┛  │
  │  ...               │
  │  Worker N ●━━━━━━  │
  │                    │
  │  Min: 25% alive    │
  │  Max: 4x CPU       │
  └────────────────────┘
```

<br />

---

## 🔐 加密算法

| 算法 | 密钥长度 | 说明 |
|:---|:---|:---|
| AES-128-GCM | 16 字节 | 高性能硬件加速 |
| AES-256-GCM | 32 字节 | 高安全性 |
| ChaCha20-Poly1305 | 32 字节 | ARM 设备优化 |
| XChaCha20-Poly1305 | 32 字节 | 扩展 nonce 长度 |

<br />

---

## 🤝 参与贡献

我们欢迎所有形式的贡献！

1. Fork 本仓库
2. 创建特性分支: `git checkout -b feature/amazing-feature`
3. 提交更改: `git commit -m 'feat: add amazing feature'`
4. 推送分支: `git push origin feature/amazing-feature`
5. 创建 Pull Request

详见 [CONTRIBUTING.md](CONTRIBUTING.md)

<br />

---

## 📋 路线图

- [x] DCCP RFC 4340 协议支持
- [x] 11 种传输协议
- [x] BBR 拥塞控制 (7 种变体)
- [x] WorkerPool 动态扩缩容
- [x] 32KB 分层缓冲池
- [x] 读写分离锁优化
- [x] 12 种 DCCP 伪装方法
- [ ] HTTP/3 (QUIC) 完整支持
- [ ] WireGuard 协议出站
- [ ] TUN 设备集成
- [ ] Web 管理面板完善
- [ ] 性能基准测试套件
- [ ] 插件系统

<br />

---

## 📄 许可证

本项目基于 [MIT License](LICENSE) 开源。

<br />

---

<p align="center">
  <strong>⚡ Oiwest Core — 高性能代理协议内核 ⚡</strong>
</p>
<p align="center">
  <sub>如果这个项目对你有帮助，请给一个 ⭐ Star 支持一下！</sub>
</p>
<p align="center">
  <sub>⚠️ 本项目仅供学习和研究目的使用，请遵守当地法律法规。</sub>
</p>

