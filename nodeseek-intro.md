## 开源项目分享：Oiwest Core — 基于 DCCP 协议的新一代传输内核，附带 Windows GUI

---

### 前言

最近在折腾代理底层传输的时候，发现大部分工具都绑死在 TCP 上。高延迟、高丢包的环境下，TCP 的拥塞控制其实并不理想——重传机制在恶劣网络中反而成了瓶颈。

于是花了不少时间研究 RFC 4340，把 **DCCP（数据报拥塞控制协议）** 完整移植到了代理场景中，做成了这个项目 **Oiwest Core**。目前已经开源，欢迎大家来玩。

---

### 项目地址

**GitHub**: [https://github.com/Hhz0823/oiwest-core](https://github.com/Hhz0823/oiwest-core)

---

### 核心思路：为什么是 DCCP？

DCCP 结合了 UDP 的**无连接开销**和 TCP 的**拥塞控制**，有点像两者的结合体：

- 像 UDP 一样传输数据，没有 TCP 的队头阻塞
- 像 TCP 一样拥塞控制，不会把网络打爆
- 支持多种 CCID（拥塞控制算法）：CCID2（类 TCP）、CCID3（TFRC 速率控制）、CCID4（TFRC-SP）

在实际测试中，**高延迟+高丢包**的场景下，DCCP 的传输表现明显优于传统 TCP 代理，尤其适合跨境这种网络质量不太稳定的链路。

---

### 功能特性一览

**传输协议**
- DCCP（核心特色）、TCP、WebSocket、QUIC、gRPC

**代理协议**
- VMess、VLESS（支持 XTLS Vision Flow）、Trojan、Shadowsocks、SOCKS5、HTTP

**流量混淆**
- XTLS Vision / Reality、Random Padding、XOR / UDP 混淆、DTLS / WireGuard 伪装
- 支持 Chrome / Firefox / Safari / iOS / Android / Edge / Random 指纹模拟

**加密**
- AES-256-GCM、ChaCha20-Poly1305、XChaCha20-Poly1305

**路由**
- 域名（全匹配/关键词/正则）、IP（CIDR）、端口范围、负载均衡

**兼容性**
- 与 v2ray-core / Xray-core / sing-box 配置文件兼容，可无缝替换

---

### 附带 Windows GUI

很多人不太喜欢折腾命令行配置，所以顺便用 **Wails v2 + React 18 + TypeScript** 写了一个桌面 GUI（打包成单个 exe），功能包括：

- 节点管理：添加/编辑/删除，分组管理，支持 vmess:// 导入导出
- 核心控制：一键启停核心引擎，实时查看运行状态
- 系统代理：全局 / PAC / 直连三种模式，自动配置 Windows 代理
- 流量监控：实时上下行速度、总流量、数据包统计
- 实时日志：彩色分级日志，自动滚动

GUI 会自动管理核心进程和数据持久化，配置保存在 `~/.oiwest/` 目录下。

---

### 快速上手

```bash
# 构建 CLI 核心
make build

# 运行
./build/oiwest-core -config config.json

# 开发模式运行 GUI
wails dev

# 构建 GUI 桌面程序
wails build
```

多平台交叉编译：

```bash
make build-linux-amd64      # Linux
make build-windows-amd64    # Windows
make build-darwin-arm64     # macOS
```

---

### 配置文件示例

```json
{
  "inbounds": [
    {
      "port": 33445,
      "protocol": "dccp",
      "streamSettings": {
        "network": "dccp",
        "dccpSettings": {
          "ccid": 4,
          "serviceCode": "V2RY",
          "maxPacketSize": 1500,
          "enableObfuscation": true
        }
      }
    }
  ],
  "outbounds": [
    {
      "protocol": "vmess",
      "settings": {}
    }
  ],
  "routing": {
    "rules": [
      {"type": "field", "domain": ["geosite:cn"], "outboundTag": "direct"}
    ]
  }
}
```

---

### 一点感想

做这个项目的初衷其实很简单——现有工具在传输层的创新基本停滞了，大家都在应用层卷协议和混淆。DCCP 作为一个存在了十几年的 RFC 标准协议，始终没有被代理工具充分利用，有点可惜。

项目目前还在早期阶段，肯定有不少需要完善的地方。如果你有兴趣，欢迎提 Issue 或者 PR，一起把这个传输层的事做好。

最后，**如果觉得项目对你有帮助，麻烦在 GitHub 上点个 Star**，感谢支持 🙏

---

> ⚠️ 本项目仅供学习和研究使用，请遵守当地法律法规。
