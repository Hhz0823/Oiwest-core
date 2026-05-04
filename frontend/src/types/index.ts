export type ServerProtocol = 'vmess' | 'vless' | 'trojan' | 'shadowsocks' | 'socks' | 'http'

export interface ServerNode {
  id: string
  name: string
  group: string
  protocol: ServerProtocol
  address: string
  port: number
  uuid?: string
  password?: string
  security?: string
  flow?: string
  network?: string
  path?: string
  host?: string
  tls: boolean
  sni?: string
  fingerprint?: string
  publicKey?: string
  shortId?: string
  spiderX?: string
  allowInsecure: boolean
  latency: number
  upload: number
  download: number
  createdAt: string
  updatedAt: string
}

export interface ServerGroup {
  name: string
  count: number
}

export type CoreStatus = 'running' | 'stopped' | 'starting' | 'error'

export interface ProxySettings {
  mode: 'none' | 'global' | 'pac'
  proxyHost: string
  proxyPort: number
  bypassLocal: boolean
  pacUrl?: string
  enabled: boolean
}

export interface TrafficStats {
  uploadSpeed: number
  downloadSpeed: number
  totalUpload: number
  totalDownload: number
  activeConns: number
  packetsSent: number
  packetsRecv: number
}

export interface LatencyResult {
  nodeId: string
  latency: number
  success: boolean
  error: string
}

export interface KernelStatus {
  installed: boolean
  path: string
  status: string
  running: boolean
}

export interface InboundRule {
  id: string
  tag: string
  port: number
  listen: string
  protocol: string
  enabled: boolean
  settings: InboundSettings
}

export interface InboundSettings {
  auth: string
  udp: boolean
  user: string
  pass: string
  method: string
  password: string
}

export interface RoutingRule {
  id: string
  name: string
  type: string
  domain: string[]
  ip: string[]
  port: string
  network: string
  protocol: string[]
  inboundTag: string[]
  outboundTag: string
  enabled: boolean
  sort: number
}

export interface DNSConfig {
  servers: DNSServerItem[]
  hosts: Record<string, string>
  clientIp: string
  tag: string
  queryStrategy: string
  disableCache: boolean
  disableFallback: boolean
}

export interface DNSServerItem {
  address: string
  port: number
  domains: string[]
  expectIPs: string[]
  skipFallback: boolean
}

export interface TransparentProxyConfig {
  enabled: boolean
  mode: string
  redirectTcp: number
  redirectUdp: number
  bypassLan: boolean
}
