<p align="center">
  <img src="https://img.shields.io/badge/version-2.1.0-blue?style=for-the-badge&logo=data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0id2hpdGUiPjxwYXRoIGQ9Ik0xMiAyQzYuNDggMiAyIDYuNDggMiAxMnM0LjQ4IDEwIDEwIDEwIDEwLTQuNDggMTAtMTBTMTcuNTIgMiAxMiAyem0tMiAxNWwtNS01IDEuNDEtMS40MUwxMCAxNC4xN2w3LjU5LTcuNTlMMTkgOGwtOSA5eiIvPjwvc3ZnPg==" />
  <img src="https://img.shields.io/badge/go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white" />
  <img src="https://img.shields.io/badge/license-MIT-green?style=for-the-badge" />
  <img src="https://img.shields.io/badge/platforms-18+-orange?style=for-the-badge" />
</p>

<p align="center">
  <strong>高性能 · 多平台 · 模块化 代理协议内核</strong><br/>
  基于 DCCP (RFC 4340) · 兼容 v2ray / Xray / sing-box 生态
</p>

---

## 项目简介

**Oiwest Core** 是一款用 Go 编写的高性能代理协议内核，支持 **11 种传输协议**、**9 种代理协议**、**18 种平台架构**，可运行于服务器、路由器（OpenWrt）、树莓派、Android（Termux）及桌面系统。

核心特性：DCCP（RFC 4340）原生传输 · BBR 拥塞控制 · 双栈并行拨号 · 帧级多路复用 · XTLS/Vision/Reality 混淆 · WorkerPool 动态调度 · 分层缓冲池优化。

---

## 项目结构

```
app/
├── cmd/                        ← 入口
│   ├── cli/main.go             ← CLI 命令行 (oiwest-core)
│   └── daemon/                 ← 无头守护进程 (oiwest-daemon)
│       ├── daemon_windows.go
│       └── daemon_unix.go
│
├── common/                     ← 共享库
│   ├── buf/buffer.go           ← 32KB 分层缓冲池 (sync.Pool)
│   ├── crypto/crypto.go        ← AEAD: AES-GCM / ChaCha20 / HKDF
│   ├── net/
│   │   ├── addr.go             ← IPv4/IPv6/Domain 地址抽象
│   │   ├── conn.go             ← 原子统计计数连接
│   │   └── dualstack.go        ← 双栈并行拨号 + 多线路探测
│   ├── platform/ (15 files)    ← OS/架构/发行版检测 + 路径 + 信号
│   ├── protocol/protocol.go    ← HTTP/TLS/DNS 协议嗅探
│   └── tls/tlsgen.go           ← RSA/ECDSA/Ed25519 证书自动生成
│
├── config/config.go            ← JSON 配置解析 (兼容 v2ray/Xray)
│
├── core/
│   ├── core.go                 ← 核心引擎: 启动/停止/协议注册
│   └── worker.go               ← WorkerPool: 动态扩缩容 + 任务调度
│
├── features/
│   ├── multiplex/mux.go        ← 帧级多路复用 (128 并发流)
│   └── stealth/
│       ├── obfuscator.go       ← XOR / Random Padding 混淆
│       ├── xtls.go             ← XTLS 流控
│       ├── vision.go           ← Vision 混淆
│       └── reality.go          ← Reality 认证
│
├── proxy/                      ← 代理协议
│   ├── proxy.go                ← ProxyManager + 连接管理 (4096 信号量)
│   ├── registry.go             ← 协议工厂注册 + Dokodemo/Loopback/DNS
│   ├── constructors.go         ← 构造函数
│   ├── adapters/adapters.go    ← 接口适配层
│   ├── vmess/vmess.go          ← VMess + AEAD 加密
│   ├── vless/vless.go          ← VLESS + XTLS Flow
│   ├── trojan/trojan.go        ← Trojan + SHA-224 认证
│   └── shadowsocks/ss.go       ← Shadowsocks + AES-GCM/ChaCha20
│
├── route/router.go             ← 路由引擎: 域名/IP/端口/协议匹配
│
└── transport/                  ← 传输层
    ├── transport.go            ← DCCP 传输 (读写分离锁)
    ├── websocket.go            ← WebSocket (32KB 缓冲)
    ├── grpc.go                 ← gRPC (合并写优化)
    ├── quic.go                 ← QUIC (0-RTT, 64 通道)
    ├── mkcp.go                 ← mKCP (可靠 UDP)
    ├── xhttp.go                ← XHTTP
    ├── config.go               ← 传输配置类型
    ├── bbr/bbr.go              ← BBR 拥塞控制 (7 种变体)
    └── dccp/
        ├── dccp.go             ← DCCP 协议核心
        ├── packet.go           ← 数据包编解码
        ├── handshake.go        ← 三次握手
        ├── congestion.go       ← CCID2/CCID3/CCID4
        ├── options.go          ← DCCP 选项
        ├── transport.go        ← DCCP 传输封装
        └── disguise/           ← 12 种流量伪装
            ├── disguise.go     ← 伪装接口 + 工厂
            ├── tls.go          ← TLS 伪装 (HTTPS)
            ├── ws_h2_grpc_df.go ← WS/H2/gRPC/域名前置
            └── extra.go        ← DTLS/WireGuard/DNS/HTTPS/HTTPUpgrade
```

---

## 支持的协议

### 代理协议

| 入站 (Inbound) | 出站 (Outbound) |
|:---|:---|
| VMess · VLESS · Trojan · Shadowsocks | Freedom · Blackhole · Direct · DNS |
| SOCKS5 · HTTP · Dokodemo · Loopback · DCCP | VMess · VLESS · Trojan · Shadowsocks · SOCKS · DCCP |

### 传输协议

| 协议 | 文件 | 说明 |
|:---|:---|:---|
| TCP | `config.go` | 标准 TCP + HTTP 伪装头 |
| mKCP | `mkcp.go` | KCP 可靠 UDP |
| WebSocket | `websocket.go` | 32KB 缓冲 + 流式读取 |
| HTTP/2 | `grpc.go` | H2 多路复用 |
| QUIC | `quic.go` | 0-RTT · 8MB 窗口 · 64 通道 |
| gRPC | `grpc.go` | 合并帧头+数据单次写 |
| XHTTP | `xhttp.go` | 自定义帧协议 |
| DCCP | `transport.go` | RFC 4340 · 读写分离锁 · 原子状态 |

### BBR 拥塞控制

`bbr` · `bbrv2` · `bbrv3` · `bbrplus` · `bbr_ecn` · `bbr_adaptive` · `bbr_probert`

使用 `sync.RWMutex`，只读方法（CWND / PacingRate / MinRTT / BW）走 `RLock`，降低锁竞争。

### DCCP 伪装方法

`tls` · `websocket` · `http2` · `grpc` · `domain_fronting` · `dtls` · `wireguard` · `https` · `dns` · `http_upgrade` · `traffic_shape` · `none`

---

## 平台支持

| 系统 | 架构 | 适用设备 |
|:---|:---|:---|
| Windows | amd64 · arm64 · 386 | PC · Surface · 老旧设备 |
| Linux | amd64 · arm64 · arm · 386 | 服务器 · 树莓派 · ARM 盒子 |
| Linux | mips · mipsle · mips64 · mips64le | OpenWrt 路由器 |
| macOS | amd64 · arm64 | Intel Mac · Apple Silicon |
| Android | arm64 | Termux |

---

## 快速开始

```bash
# 克隆
git clone https://github.com/Hhz0823/Oiwest-core.git
cd Oiwest-core

# 构建
go build -o oiwest-core ./app/cmd/cli
go build -o oiwest-daemon ./app/cmd/daemon

# 运行
./oiwest-core -config config.json    # 使用配置文件
./oiwest-core -test                  # 默认配置 (DCCP 33445)
./oiwest-daemon                      # 守护进程模式
```

### 最小配置

```json
{
  "inbounds": [{
    "tag": "dccp-in",
    "port": 33445,
    "listen": "0.0.0.0",
    "protocol": "dccp",
    "streamSettings": {
      "network": "dccp",
      "dccpSettings": { "ccid": 4, "serviceCode": "V2RY" }
    }
  }],
  "outbounds": [{ "tag": "direct", "protocol": "freedom" }]
}
```

---

## v2.1.0 性能优化

| 优化项 | 改前 | 改后 | 效果 |
|:---|:---|:---|:---|
| 缓冲池 | 2KB 单级 | 32KB 四级池化 | syscall ↓16x |
| DCCP 传输 | 单锁串行 | 读写锁分离 | 并发 ↑2-4x |
| BBR | Mutex | RWMutex | 锁竞争 ↓70% |
| WorkerPool | 固定 worker | 动态扩缩容 | 内存 ↓30% |
| gRPC | 每帧堆分配 | 栈分配+合并写 | 分配 ↓90% |
| MuxSession | 每帧分配 | 预分配复用 | GC ↓60% |
| WebSocket | 4KB + ReadMessage | 32KB + NextReader | 双倍分配消除 |
| StatCounter | 普通 int64 | atomic 操作 | 竞态消除 |
| 代理管理器 | 无限制 | 4096 信号量 + KeepAlive | 稳定性 ↑ |
| DualStack | 硬编码端口 80 | 实际端口 + JoinHostPort | 探测准确 ↑ |

---

## 加密算法

| 算法 | 密钥 | 说明 |
|:---|:---|:---|
| AES-128-GCM | 16B | 硬件加速 |
| AES-256-GCM | 32B | 高安全性 |
| ChaCha20-Poly1305 | 32B | ARM 优化 |
| XChaCha20-Poly1305 | 32B | 扩展 nonce |

---

## 构建命令

```bash
make build                  # 当前平台 CLI
make build-daemon           # 当前平台 Daemon
make build-all              # 全部 14 平台
make build-openwrt-all      # OpenWrt 全架构
make build-android-all      # Android
make test                   # 运行测试
make vet                    # 静态检查
```

---

## 生态兼容

| 生态 | 兼容内容 |
|:---|:---|
| v2ray-core | 配置格式 · 协议 · 传输 · 路由规则 |
| Xray-core | VLESS · Trojan · XTLS · Vision · Reality |
| sing-box | 传输层 · 路由规则 |

---

## 贡献

1. Fork → 创建分支 → 提交 → PR
2. 详见 [CONTRIBUTING.md](CONTRIBUTING.md)

---

## 许可证

[MIT License](LICENSE) · Copyright (c) 2025 Oiwest

---

<p align="center">
  <sub>⚠️ 仅供学习研究使用，请遵守当地法律法规</sub>
</p>
