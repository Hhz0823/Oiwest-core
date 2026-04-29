# Oiwest Core

基于 RFC 4340 的 DCCP（Datagram Congestion Control Protocol，数据报拥塞控制协议）传输协议内核，使用 Go 语言编写，兼容 v2ray-core、Xray-core、sing-box。

## 项目概述

Oiwest Core 是一个高性能、模块化的 DCCP 协议实现，将 DCCP 作为传输层协议引入代理通信领域。通过拥塞控制、多路复用和多种混淆技术，提供安全、高效的数据传输能力。

## 功能特性

### 传输协议
- **DCCP** (RFC 4340) — 核心传输协议，支持 CCID2/CCID3/CCID4 拥塞控制
- **TCP** — 标准 TCP 传输
- **WebSocket** — 基于 WebSocket 的流式传输
- **QUIC** — 基于 QUIC 的高性能传输
- **gRPC** — 基于 gRPC 的传输支持

### 代理协议
- VMess — v2ray 标准协议
- VLESS — 轻量级无状态协议
- Trojan — Trojan 代理协议
- Shadowsocks — Shadowsocks 协议
- SOCKS5 — 标准 SOCKS5 代理
- HTTP — HTTP 代理

### 混淆与隐匿
- **XTLS** — XTLS Vision/Reality 流控混淆
- **Random Padding** — 随机填充混淆，规避流量特征检测
- **XOR Obfuscation** — 基于异或的数据混淆
- **UDP Obfuscation** — UDP 数据包混淆
- **DTLS Obfuscation** — DTLS 伪装
- **WireGuard Obfuscation** — WireGuard 协议伪装

### 加密算法
- AES-256-GCM
- ChaCha20-Poly1305
- XChaCha20-Poly1305

### 路由功能
- 域名匹配（全匹配、关键词、正则表达式）
- IP 地址匹配（CIDR / 精确匹配）
- 端口范围匹配
- 入站/出站标签匹配
- 协议类型匹配
- 负载均衡（Balancer）

### 其他
- 流量统计（上行/下行、活跃连接数）
- JSON 配置文件格式，兼容主流配置结构
- 跨平台支持：Linux / Windows / macOS（amd64 / arm64）
- 信号监听，优雅退出

## 快速开始

### 环境要求
- Go 1.22+
- Make（可选）

### 构建

```bash
# 构建当前平台
make build

# 构建所有平台
make build-all

# 构建指定平台
make build-linux-amd64
make build-linux-arm64
make build-windows-amd64
make build-darwin-amd64
make build-darwin-arm64
```

如果要直接使用 go build：

```bash
go build -o build/dccp-kernel ./cmd/dccp-kernel
```

### 运行

```bash
# 使用配置文件运行
./build/dccp-kernel -config config.json

# 测试模式（使用默认配置）
./build/dccp-kernel -test

# 调试模式
./build/dccp-kernel -debug -config config.json

# 查看版本信息
./build/dccp-kernel -version
```

### 配置文件示例

```json
{
  "log": {
    "loglevel": "warning"
  },
  "inbounds": [
    {
      "tag": "dccp-in",
      "port": 33445,
      "listen": "0.0.0.0",
      "protocol": "dccp",
      "streamSettings": {
        "network": "dccp",
        "security": "none",
        "dccpSettings": {
          "ccid": 4,
          "serviceCode": "V2RY",
          "maxPacketSize": 1500,
          "handshakeTimeout": 15000000000,
          "maxRetries": 3,
          "enableObfuscation": true,
          "obfuscationType": "random"
        }
      }
    }
  ],
  "outbounds": [
    {
      "tag": "direct",
      "protocol": "freedom"
    }
  ],
  "routing": {
    "domainStrategy": "AsIs",
    "rules": [
      {
        "type": "field",
        "domain": ["geosite:cn"],
        "outboundTag": "direct"
      }
    ]
  }
}
```

## 项目结构

```
Oiwest Core/
├── cmd/dccp-kernel/     # 主程序入口
├── common/              # 公共组件
│   ├── buf/             # 缓冲区管理
│   ├── crypto/          # 加密算法 (AES-GCM, ChaCha20-Poly1305)
│   ├── net/             # 网络地址与连接抽象
│   └── protocol/        # 协议编解码
├── config/              # 配置解析与管理
├── core/                # 核心引擎（启动、停止、生命周期）
├── features/
│   ├── multiplex/       # 多路复用
│   └── stealth/         # 隐匿与混淆（XTLS, Vision, Reality, Padding）
├── proxy/               # 代理管理器（入站/出站处理）
├── router/              # 路由引擎（域名/IP/端口匹配）
├── transport/           # 传输层抽象
│   └── dccp/            # DCCP 协议实现（拥塞控制、握手、数据包）
├── go.mod
├── go.sum
└── Makefile
```

## DCCP 协议说明

DCCP（Datagram Congestion Control Protocol）是 IETF RFC 4340 定义的传输层协议。它提供了：

- **不可靠数据传输** — 类似 UDP
- **拥塞控制** — 类似 TCP
- **连接建立与拆除** — 具有三次握手和四次挥手
- **多种拥塞控制算法** — CCID2（TCP-like）、CCID3（TFRC）、CCID4（TFRC-SP）

本项目将其引入代理通信领域，利用 DCCP 的特性在高延迟/高丢包网络环境下提供更好的传输性能。

## 兼容性

| 软件 | 兼容状态 |
|------|--------|
| v2ray-core | ✅ 兼容 |
| Xray-core | ✅ 兼容 |
| sing-box | ✅ 兼容 |

## 常用命令

```bash
make build        # 构建项目
make clean        # 清理构建产物
make test         # 运行测试
make run          # 测试模式运行
make run-debug    # 调试模式运行
make install      # 安装到 $GOPATH/bin
make lint         # 代码检查
make fmt          # 代码格式化
make vet          # 代码静态分析
make mod          # 依赖整理与验证
make deps         # 下载依赖
```

## 许可证

MIT License

---

**Warning**: 本项目仅供学习和研究目的使用，请遵守当地法律法规。
