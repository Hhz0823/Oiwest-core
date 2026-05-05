import { useEffect } from 'react'
import { useAppStore } from '../stores/appStore'
import {
  Play, Square, RefreshCw, Shield, ShieldOff,
  ArrowUp, ArrowDown, Activity, HardDrive, Globe,
  Cpu, AlertTriangle, MemoryStick, Wifi, Copy, Check, Tag
} from 'lucide-react'
import { useState } from 'react'

function fmtBytes(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  let i = -1, val = bytes
  do { val /= 1024; i++ } while (val >= 1024 && i < 3)
  return val.toFixed(1) + ' ' + ['KB','MB','GB','TB'][i]
}
function fmtSpeed(bps: number): string { return fmtBytes(bps) + '/s' }

export default function Dashboard() {
  const coreStatus = useAppStore((s) => s.coreStatus)
  const coreUptime = useAppStore((s) => s.coreUptime)
  const trafficStats = useAppStore((s) => s.trafficStats)
  const proxyEnabled = useAppStore((s) => s.proxyEnabled)
  const nodes = useAppStore((s) => s.nodes)
  const activeNodeID = useAppStore((s) => s.activeNodeID)
  const kernelStatus = useAppStore((s) => s.kernelStatus)
  const deviceInfo = useAppStore((s) => s.deviceInfo)
  const systemUsage = useAppStore((s) => s.systemUsage)
  const appVersion = useAppStore((s) => s.appVersion)
  const startCore = useAppStore((s) => s.startCore)
  const stopCore = useAppStore((s) => s.stopCore)
  const restartCore = useAppStore((s) => s.restartCore)
  const toggleProxy = useAppStore((s) => s.toggleProxy)
  const refreshSystemUsage = useAppStore((s) => s.refreshSystemUsage)
  const refreshPublicIP = useAppStore((s) => s.refreshPublicIP)

  const [copiedIPv4, setCopiedIPv4] = useState(false)
  const [copiedIPv6, setCopiedIPv6] = useState(false)

  useEffect(() => {
    const interval = setInterval(() => refreshSystemUsage(), 3000)
    return () => clearInterval(interval)
  }, [])

  const activeNode = nodes.find((n) => n.id === activeNodeID)
  const isRunning = coreStatus === 'running'
  const isStarting = coreStatus === 'starting'
  const cpu = systemUsage?.cpuPercent ?? 0
  const mem = systemUsage?.memoryPercent ?? 0
  const memUsed = systemUsage?.memoryUsed ?? 0
  const memTotal = systemUsage?.memoryTotal ?? 0
  const publicIPv4 = systemUsage?.publicIp ?? '-'
  const publicIPv6 = systemUsage?.publicIpv6 ?? ''
  const totalMem = deviceInfo?.totalMemory ?? 0

  const copyIPv4 = async () => {
    if (publicIPv4 === '-' || publicIPv4 === 'N/A') return
    await navigator.clipboard.writeText(publicIPv4)
    setCopiedIPv4(true); setTimeout(() => setCopiedIPv4(false), 1500)
  }
  const copyIPv6 = async () => {
    if (!publicIPv6) return
    await navigator.clipboard.writeText(publicIPv6)
    setCopiedIPv6(true); setTimeout(() => setCopiedIPv6(false), 1500)
  }

  const cpuColor = cpu > 80 ? 'var(--red)' : cpu > 50 ? 'var(--yellow)' : 'var(--green)'
  const memColor = mem > 85 ? 'var(--red)' : mem > 60 ? 'var(--yellow)' : 'var(--green)'

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between animate-fade-in-up">
        <div className="flex items-center gap-4">
          <div>
            <h2 className="text-xl font-semibold" style={{ color: 'var(--text-primary)' }}>仪表盘</h2>
            <p className="text-sm mt-0.5" style={{ color: 'var(--text-secondary)' }}>网络状态、系统监控与流量统计</p>
          </div>
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg" style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}>
            <Tag size={13} style={{ color: 'var(--accent)' }} />
            <span className="text-sm font-mono font-semibold" style={{ color: 'var(--accent)' }}>v{appVersion}</span>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {kernelStatus?.installed === false && (
            <div className="flex items-center gap-2 px-3 py-2 rounded-lg text-xs animate-pulse-red" style={{ color: 'var(--yellow)', background: 'rgba(234,179,8,0.1)', border: '1px solid rgba(234,179,8,0.3)' }}>
              <AlertTriangle size={14} /> 内核未安装
            </div>
          )}
          <button onClick={toggleProxy}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 active:scale-95 ${proxyEnabled ? 'text-green-400 border border-green-500/30' : ''}`}
            style={proxyEnabled ? { background: 'rgba(34,197,94,0.15)' } : { background: 'var(--bg-tertiary)', color: 'var(--text-secondary)', border: '1px solid var(--border)' }}>
            {proxyEnabled ? <Shield size={16} /> : <ShieldOff size={16} />}
            {proxyEnabled ? '代理已开启' : '代理已关闭'}
          </button>
          {isRunning ? (
            <button onClick={restartCore} disabled={isStarting} className="btn-primary flex items-center gap-2">
              <RefreshCw size={16} className={isStarting ? 'animate-spin' : ''} /> {isStarting ? '重启中...' : '重启内核'}
            </button>
          ) : (
            <button onClick={startCore} disabled={isStarting} className="btn-primary flex items-center gap-2">
              <Play size={16} /> {isStarting ? '启动中...' : '启动内核'}
            </button>
          )}
          {isRunning && (
            <button onClick={stopCore} className="btn-danger flex items-center gap-2"><Square size={16} /> 停止</button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-6 gap-3">
        {[
          { label: '上行', value: fmtSpeed(trafficStats.uploadSpeed), icon: ArrowUp, color: '#60a5fa' },
          { label: '下行', value: fmtSpeed(trafficStats.downloadSpeed), icon: ArrowDown, color: '#34d399' },
          { label: '连接', value: String(trafficStats.activeConns), icon: Activity, color: '#a78bfa' },
          { label: 'CPU', value: `${cpu.toFixed(0)}%`, icon: Cpu, color: cpuColor },
          { label: '内存', value: `${mem.toFixed(0)}%`, icon: MemoryStick, color: memColor },
          { label: '运行', value: coreUptime, icon: HardDrive, color: '#fbbf24' },
        ].map((c, i) => (
          <div key={c.label} className="glass-card animate-fade-in-up px-3 py-3" style={{ animationDelay: `${i * 0.05}s` }}>
            <div className="flex items-center gap-1.5 mb-1.5" style={{ color: 'var(--text-secondary)', fontSize: '11px' }}>
              <c.icon size={13} style={{ color: c.color }} /> {c.label}
            </div>
            <p className="font-mono font-semibold" style={{ color: c.color, fontSize: '15px' }}>{c.value}</p>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-2 gap-4 animate-fade-in-up" style={{ animationDelay: '0.3s' }}>
        <div className="glass-card space-y-3">
          <h3 className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>公网 IP 地址</h3>
          <div className="space-y-3">
            <div className="flex items-center gap-3 px-3 py-2.5 rounded-lg" style={{ background: 'var(--bg-secondary)' }}>
              <span className="text-xs font-semibold shrink-0 px-1.5 py-0.5 rounded" style={{ color: '#60a5fa', background: 'rgba(96,165,250,0.12)' }}>IPv4</span>
              <span className="text-sm font-mono flex-1 truncate" style={{ color: 'var(--text-primary)' }}>{publicIPv4}</span>
              <button onClick={async (e) => { e.stopPropagation(); await copyIPv4() }} className="btn-ghost px-1 py-0" title="复制 IPv4">
                {copiedIPv4 ? <Check size={12} style={{ color: 'var(--green)' }} /> : <Copy size={12} />}
              </button>
            </div>
            <div className="flex items-center gap-3 px-3 py-2.5 rounded-lg" style={{ background: 'var(--bg-secondary)' }}>
              <span className="text-xs font-semibold shrink-0 px-1.5 py-0.5 rounded" style={{ color: '#34d399', background: 'rgba(52,211,153,0.12)' }}>IPv6</span>
              <span className="text-sm font-mono flex-1 truncate" style={{ color: publicIPv6 ? 'var(--text-primary)' : 'var(--text-muted)' }}>
                {publicIPv6 || '未检测到 IPv6'}
              </span>
              {publicIPv6 && (
                <button onClick={async (e) => { e.stopPropagation(); await copyIPv6() }} className="btn-ghost px-1 py-0" title="复制 IPv6">
                  {copiedIPv6 ? <Check size={12} style={{ color: 'var(--green)' }} /> : <Copy size={12} />}
                </button>
              )}
            </div>
          </div>
          <div className="flex justify-end">
            <button onClick={refreshPublicIP} className="btn-ghost px-2 py-1 text-xs flex items-center gap-1">
              <RefreshCw size={11} /> 刷新 IP
            </button>
          </div>
        </div>

        <div className="glass-card space-y-3">
          <h3 className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>系统资源</h3>
          <div className="space-y-3">
            <div>
              <div className="flex justify-between text-xs mb-1"><span style={{ color: 'var(--text-secondary)' }}>CPU</span><span style={{ color: cpuColor }} className="font-mono">{cpu.toFixed(1)}%</span></div>
              <div className="h-2 rounded-full overflow-hidden" style={{ background: 'var(--bg-tertiary)' }}>
                <div className="h-full rounded-full transition-all duration-700 ease-out" style={{ width: `${Math.min(cpu, 100)}%`, background: cpuColor }} />
              </div>
            </div>
            <div>
              <div className="flex justify-between text-xs mb-1"><span style={{ color: 'var(--text-secondary)' }}>内存</span><span style={{ color: memColor }} className="font-mono">{fmtBytes(memUsed)} / {fmtBytes(memTotal)}</span></div>
              <div className="h-2 rounded-full overflow-hidden" style={{ background: 'var(--bg-tertiary)' }}>
                <div className="h-full rounded-full transition-all duration-700 ease-out" style={{ width: `${Math.min(mem, 100)}%`, background: memColor }} />
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-4 animate-fade-in-up" style={{ animationDelay: '0.38s' }}>
        <div className="glass-card space-y-3">
          <h3 className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>流量统计</h3>
          <div className="space-y-2">
            {[
              { label: '总上传', value: fmtBytes(trafficStats.totalUpload), color: '#60a5fa' },
              { label: '总下载', value: fmtBytes(trafficStats.totalDownload), color: '#34d399' },
              { label: '发送包', value: trafficStats.packetsSent.toLocaleString(), color: 'var(--text-secondary)' },
              { label: '接收包', value: trafficStats.packetsRecv.toLocaleString(), color: 'var(--text-secondary)' },
            ].map((r) => (
              <div key={r.label} className="flex justify-between text-xs">
                <span style={{ color: 'var(--text-secondary)' }}>{r.label}</span>
                <span className="font-mono" style={{ color: r.color }}>{r.value}</span>
              </div>
            ))}
          </div>
          {trafficStats.totalDownload + trafficStats.totalUpload > 0 && (
            <div className="h-2 rounded-full overflow-hidden flex" style={{ background: 'var(--bg-tertiary)' }}>
              <div className="h-full transition-all duration-500" style={{ width: `${(trafficStats.totalUpload/(trafficStats.totalUpload+trafficStats.totalDownload))*100}%`, background: '#60a5fa' }} />
              <div className="h-full transition-all duration-500" style={{ width: `${(trafficStats.totalDownload/(trafficStats.totalUpload+trafficStats.totalDownload))*100}%`, background: '#34d399' }} />
            </div>
          )}
        </div>

        <div className="glass-card space-y-3">
          <h3 className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>设备信息</h3>
          <div className="space-y-1.5">
            {[
              ['主机名', deviceInfo?.hostname || '-', true],
              ['OS', deviceInfo ? `${deviceInfo.os} / ${deviceInfo.arch}` : '-', true],
              ['CPU 核心', deviceInfo?.cpuCores ? `${deviceInfo.cpuCores} 核` : '-', false],
              ['总内存', totalMem > 0 ? fmtBytes(totalMem) : '-', false],
              ['Go 版本', deviceInfo?.goVersion || '-', false],
            ].map(([label, value, mono]) => (
              <div key={label as string} className="flex justify-between text-xs">
                <span style={{ color: 'var(--text-secondary)' }}>{label}</span>
                <span className={mono ? 'font-mono' : ''} style={{ color: 'var(--text-primary)' }}>{value as string}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="glass-card space-y-3">
          <h3 className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>连接信息</h3>
          {activeNode ? (
            <div className="space-y-1.5">
              {[
                ['节点', activeNode.name],
                ['协议', activeNode.protocol.toUpperCase()],
                ['地址', `${activeNode.address}:${activeNode.port}`],
                ['传输', `${activeNode.network || 'tcp'}${activeNode.tls ? '+TLS' : ''}`],
                ['延迟', activeNode.latency > 0 ? `${activeNode.latency}ms` : 'N/A'],
                ['状态', isRunning ? '运行中' : '已停止'],
              ].map(([l, v]) => (
                <div key={l} className="flex justify-between text-xs">
                  <span style={{ color: 'var(--text-secondary)' }}>{l}</span>
                  <span className="font-mono" style={{
                    color: l === '延迟' ? (activeNode.latency > 0 ? (activeNode.latency < 100 ? 'var(--green)' : activeNode.latency < 300 ? 'var(--yellow)' : 'var(--red)') : 'var(--text-muted)')
                      : l === '状态' ? (isRunning ? 'var(--green)' : 'var(--red)') : 'var(--text-primary)',
                  }}>{v}</span>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-4">
              <Globe size={28} style={{ color: 'var(--text-muted)', opacity: 0.5 }} className="mx-auto mb-2" />
              <p className="text-sm" style={{ color: 'var(--text-secondary)' }}>未选择节点</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
