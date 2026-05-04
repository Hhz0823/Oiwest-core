import { create } from 'zustand'
import type {
  ServerNode, ServerGroup, CoreStatus, ProxySettings, TrafficStats,
  LatencyResult, KernelStatus, InboundRule, RoutingRule, DNSConfig,
  DNSServerItem, TransparentProxyConfig
} from '../types'

declare global {
  interface Window {
    go: {
      main: {
        App: {
          GetNodes(): Promise<ServerNode[]>
          AddNode(node: ServerNode): Promise<void>
          UpdateNode(node: ServerNode): Promise<void>
          DeleteNode(id: string): Promise<void>
          GetGroups(): Promise<ServerGroup[]>
          MoveNode(id: string, newGroup: string): Promise<void>
          ImportFromLink(link: string): Promise<ServerNode>
          TestNodeLatency(id: string): Promise<LatencyResult>
          TestAllNodesLatency(): Promise<LatencyResult[]>
          GetNodeIPs(id: string): Promise<string[]>
          GetKernelStatus(): Promise<KernelStatus>
          IsKernelInstalled(): Promise<boolean>
          StartCore(): Promise<void>
          StopCore(): Promise<void>
          RestartCore(): Promise<void>
          GetCoreStatus(): Promise<string>
          GetCoreUptime(): Promise<string>
          GetCoreLogs(count: number): Promise<string[]>
          GetFilteredLogs(category: string, count: number): Promise<string[]>
          ClearCoreLogs(): Promise<void>
          CopyLogs(logs: string[]): Promise<void>
          GetProxySettings(): Promise<ProxySettings>
          SetProxySettings(settings: ProxySettings): Promise<void>
          EnableProxy(): Promise<void>
          DisableProxy(): Promise<void>
          ToggleProxy(): Promise<void>
          IsProxyEnabled(): Promise<boolean>
          GetTrafficStats(): Promise<TrafficStats>
          ResetTrafficStats(): Promise<void>
          SelectNode(nodeID: string): Promise<void>
          GetActiveNodeID(): Promise<string>
          GetAppVersion(): Promise<string>
          GetInbounds(): Promise<InboundRule[]>
          AddInbound(rule: InboundRule): Promise<void>
          UpdateInbound(rule: InboundRule): Promise<void>
          DeleteInbound(id: string): Promise<void>
          ToggleInbound(id: string, enabled: boolean): Promise<void>
          GetRoutingRules(): Promise<RoutingRule[]>
          AddRoutingRule(rule: RoutingRule): Promise<void>
          UpdateRoutingRule(rule: RoutingRule): Promise<void>
          DeleteRoutingRule(id: string): Promise<void>
          ReorderRoutingRules(ids: string[]): Promise<void>
          GetDNSConfig(): Promise<DNSConfig>
          SetDNSConfig(cfg: DNSConfig): Promise<void>
          AddDNSServer(server: DNSServerItem): Promise<void>
          RemoveDNSServer(index: number): Promise<void>
          GetTransparentProxyConfig(): Promise<TransparentProxyConfig>
          SetTransparentProxyConfig(cfg: TransparentProxyConfig): Promise<void>
        }
      }
    }
    runtime: {
      EventsOn(event: string, callback: (...args: any[]) => void): void
      EventsOff(event: string): void
    }
  }
}

const invoke = <T>(method: string, ...args: any[]): Promise<T> => {
  const fn = method.split('.').reduce((obj: any, key) => obj?.[key], window.go?.main)
  if (!fn) return Promise.reject(new Error('Not ready'))
  return fn(...args)
}

interface AppState {
  nodes: ServerNode[]
  groups: ServerGroup[]
  coreStatus: CoreStatus
  coreUptime: string
  proxyEnabled: boolean
  proxySettings: ProxySettings
  trafficStats: TrafficStats
  activeNodeID: string
  logs: string[]
  appVersion: string
  loading: boolean
  kernelStatus: KernelStatus | null
  nodeLatencies: Record<string, LatencyResult>
  inbounds: InboundRule[]
  routingRules: RoutingRule[]
  dnsConfig: DNSConfig | null
  transparentProxy: TransparentProxyConfig | null

  loadNodes: () => Promise<void>
  loadGroups: () => Promise<void>
  addNode: (node: ServerNode) => Promise<void>
  updateNode: (node: ServerNode) => Promise<void>
  deleteNode: (id: string) => Promise<void>
  moveNode: (id: string, newGroup: string) => Promise<void>
  importFromLink: (link: string) => Promise<ServerNode>
  selectNode: (nodeID: string) => Promise<void>
  testNodeLatency: (id: string) => Promise<void>
  testAllLatency: () => Promise<void>
  getNodeIPs: (id: string) => Promise<string[]>

  refreshKernelStatus: () => Promise<void>
  startCore: () => Promise<void>
  stopCore: () => Promise<void>
  restartCore: () => Promise<void>
  refreshCoreStatus: () => Promise<void>

  enableProxy: () => Promise<void>
  disableProxy: () => Promise<void>
  toggleProxy: () => Promise<void>
  loadProxySettings: () => Promise<void>
  setProxySettings: (settings: ProxySettings) => Promise<void>

  refreshTrafficStats: () => Promise<void>
  resetTrafficStats: () => Promise<void>
  loadLogs: () => Promise<void>
  clearLogs: () => Promise<void>
  copyAllLogs: () => Promise<void>
  addLog: (msg: string) => void
  updateStats: (stats: TrafficStats) => void

  loadNetworkConfig: () => Promise<void>
  addInbound: (rule: InboundRule) => Promise<void>
  updateInbound: (rule: InboundRule) => Promise<void>
  deleteInbound: (id: string) => Promise<void>
  toggleInbound: (id: string, enabled: boolean) => Promise<void>
  addRoutingRule: (rule: RoutingRule) => Promise<void>
  updateRoutingRule: (rule: RoutingRule) => Promise<void>
  deleteRoutingRule: (id: string) => Promise<void>
  reorderRoutingRules: (ids: string[]) => Promise<void>
  setDNSConfig: (cfg: DNSConfig) => Promise<void>
  addDNSServer: (server: DNSServerItem) => Promise<void>
  removeDNSServer: (index: number) => Promise<void>
  setTransparentProxyConfig: (cfg: TransparentProxyConfig) => Promise<void>

  initApp: () => Promise<void>
}

export const useAppStore = create<AppState>((set, get) => ({
  nodes: [], groups: [],
  coreStatus: 'stopped', coreUptime: '00:00:00',
  proxyEnabled: false,
  proxySettings: { mode: 'global', proxyHost: '127.0.0.1', proxyPort: 10808, bypassLocal: true, enabled: false },
  trafficStats: { uploadSpeed: 0, downloadSpeed: 0, totalUpload: 0, totalDownload: 0, activeConns: 0, packetsSent: 0, packetsRecv: 0 },
  activeNodeID: '', logs: [], appVersion: '1.0.0', loading: false,
  kernelStatus: null, nodeLatencies: {},
  inbounds: [], routingRules: [], dnsConfig: null, transparentProxy: null,

  loadNodes: async () => {
    const nodes = await invoke<ServerNode[]>('App.GetNodes')
    set({ nodes: nodes || [] })
  },
  loadGroups: async () => {
    const groups = await invoke<ServerGroup[]>('App.GetGroups')
    set({ groups: groups || [] })
  },
  addNode: async (node) => { await invoke<void>('App.AddNode', node); get().loadNodes(); get().loadGroups() },
  updateNode: async (node) => { await invoke<void>('App.UpdateNode', node); get().loadNodes(); get().loadGroups() },
  deleteNode: async (id) => { await invoke<void>('App.DeleteNode', id); get().loadNodes(); get().loadGroups() },
  moveNode: async (id, newGroup) => { await invoke<void>('App.MoveNode', id, newGroup); get().loadNodes(); get().loadGroups() },
  importFromLink: async (link) => { const node = await invoke<ServerNode>('App.ImportFromLink', link); get().loadNodes(); get().loadGroups(); return node },
  selectNode: async (nodeID) => { await invoke<void>('App.SelectNode', nodeID); set({ activeNodeID: nodeID }); get().refreshCoreStatus() },

  testNodeLatency: async (id) => {
    const result = await invoke<LatencyResult>('App.TestNodeLatency', id)
    set((s) => ({ nodeLatencies: { ...s.nodeLatencies, [id]: result } }))
    if (result.success) get().loadNodes()
  },
  testAllLatency: async () => {
    set({ nodeLatencies: {} })
    await invoke<LatencyResult[]>('App.TestAllNodesLatency')
    get().loadNodes()
  },
  getNodeIPs: async (id) => invoke<string[]>('App.GetNodeIPs', id),

  refreshKernelStatus: async () => {
    try {
      const status = await invoke<KernelStatus>('App.GetKernelStatus')
      set({ kernelStatus: status })
    } catch { /* ignore */ }
  },

  startCore: async () => { await invoke<void>('App.StartCore'); set({ coreStatus: 'starting' }) },
  stopCore: async () => { await invoke<void>('App.StopCore'); set({ coreStatus: 'stopped' }) },
  restartCore: async () => { await invoke<void>('App.RestartCore'); set({ coreStatus: 'starting' }) },

  refreshCoreStatus: async () => {
    try {
      const status = await invoke<string>('App.GetCoreStatus')
      const uptime = await invoke<string>('App.GetCoreUptime')
      const activeID = await invoke<string>('App.GetActiveNodeID')
      set({ coreStatus: status as CoreStatus, coreUptime: uptime, activeNodeID: activeID || get().activeNodeID })
    } catch { set({ coreStatus: 'stopped' }) }
  },

  enableProxy: async () => { await invoke<void>('App.EnableProxy'); set({ proxyEnabled: true }) },
  disableProxy: async () => { await invoke<void>('App.DisableProxy'); set({ proxyEnabled: false }) },
  toggleProxy: async () => { await invoke<void>('App.ToggleProxy'); const e = await invoke<boolean>('App.IsProxyEnabled'); set({ proxyEnabled: e }) },
  loadProxySettings: async () => {
    try { const s = await invoke<ProxySettings>('App.GetProxySettings'); set({ proxySettings: s, proxyEnabled: s.enabled }) } catch {}
  },
  setProxySettings: async (settings) => { await invoke<void>('App.SetProxySettings', settings); set({ proxySettings: settings, proxyEnabled: settings.enabled }) },

  refreshTrafficStats: async () => { try { set({ trafficStats: await invoke<TrafficStats>('App.GetTrafficStats') }) } catch {} },
  resetTrafficStats: async () => { await invoke<void>('App.ResetTrafficStats'); set({ trafficStats: { uploadSpeed: 0, downloadSpeed: 0, totalUpload: 0, totalDownload: 0, activeConns: 0, packetsSent: 0, packetsRecv: 0 } }) },
  loadLogs: async () => { try { set({ logs: await invoke<string[]>('App.GetCoreLogs', 200) || [] }) } catch {} },
  clearLogs: async () => { await invoke<void>('App.ClearCoreLogs'); set({ logs: [] }) },
  copyAllLogs: async () => { await invoke<void>('App.CopyLogs', get().logs) },
  addLog: (msg) => set((s) => ({ logs: [...s.logs.slice(-199), msg] })),
  updateStats: (stats) => set({ trafficStats: stats }),

  loadNetworkConfig: async () => {
    try {
      const [inbounds, routing, dns, tp] = await Promise.all([
        invoke<InboundRule[]>('App.GetInbounds'),
        invoke<RoutingRule[]>('App.GetRoutingRules'),
        invoke<DNSConfig>('App.GetDNSConfig'),
        invoke<TransparentProxyConfig>('App.GetTransparentProxyConfig'),
      ])
      set({ inbounds: inbounds || [], routingRules: routing || [], dnsConfig: dns, transparentProxy: tp })
    } catch {}
  },
  addInbound: async (r) => { await invoke<void>('App.AddInbound', r); get().loadNetworkConfig() },
  updateInbound: async (r) => { await invoke<void>('App.UpdateInbound', r); get().loadNetworkConfig() },
  deleteInbound: async (id) => { await invoke<void>('App.DeleteInbound', id); get().loadNetworkConfig() },
  toggleInbound: async (id, enabled) => { await invoke<void>('App.ToggleInbound', id, enabled); get().loadNetworkConfig() },
  addRoutingRule: async (r) => { await invoke<void>('App.AddRoutingRule', r); get().loadNetworkConfig() },
  updateRoutingRule: async (r) => { await invoke<void>('App.UpdateRoutingRule', r); get().loadNetworkConfig() },
  deleteRoutingRule: async (id) => { await invoke<void>('App.DeleteRoutingRule', id); get().loadNetworkConfig() },
  reorderRoutingRules: async (ids) => { await invoke<void>('App.ReorderRoutingRules', ids); get().loadNetworkConfig() },
  setDNSConfig: async (cfg) => { await invoke<void>('App.SetDNSConfig', cfg); set({ dnsConfig: cfg }) },
  addDNSServer: async (s) => { await invoke<void>('App.AddDNSServer', s); get().loadNetworkConfig() },
  removeDNSServer: async (i) => { await invoke<void>('App.RemoveDNSServer', i); get().loadNetworkConfig() },
  setTransparentProxyConfig: async (cfg) => { await invoke<void>('App.SetTransparentProxyConfig', cfg); set({ transparentProxy: cfg }) },

  initApp: async () => {
    set({ loading: true })
    await get().loadNodes()
    await get().loadGroups()
    await get().loadNetworkConfig()
    await get().refreshCoreStatus()
    await get().refreshKernelStatus()
    await get().loadProxySettings()
    const v = await invoke<string>('App.GetAppVersion'); set({ appVersion: v || '1.0.0' })
    window.runtime?.EventsOn('core-log', (msg: string) => get().addLog(msg))
    window.runtime?.EventsOn('stats-update', (stats: TrafficStats) => get().updateStats(stats))
    window.runtime?.EventsOn('latency-result', (r: LatencyResult) => set((s) => ({ nodeLatencies: { ...s.nodeLatencies, [r.nodeId]: r } })))
    set({ loading: false })
  },
}))
