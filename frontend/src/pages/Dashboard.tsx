import { useAppStore } from '../stores/appStore'
import {
  Play, Square, RefreshCw, Shield, ShieldOff,
  ArrowUp, ArrowDown, Activity, HardDrive, Globe,
  Cpu, AlertTriangle, CheckCircle, Download, Zap, Gauge
} from 'lucide-react'

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
  const startCore = useAppStore((s) => s.startCore)
  const stopCore = useAppStore((s) => s.stopCore)
  const restartCore = useAppStore((s) => s.restartCore)
  const toggleProxy = useAppStore((s) => s.toggleProxy)

  const activeNode = nodes.find((n) => n.id === activeNodeID)
  const isRunning = coreStatus === 'running'
  const isStarting = coreStatus === 'starting'
  const kernelInstalled = kernelStatus?.installed ?? true

  const cards = [
    { label: '上行速度', value: fmtSpeed(trafficStats.uploadSpeed), color: 'text-blue-400', icon: ArrowUp },
    { label: '下行速度', value: fmtSpeed(trafficStats.downloadSpeed), color: 'text-green-400', icon: ArrowDown },
    { label: '活跃连接', value: String(trafficStats.activeConns), color: 'text-purple-400', icon: Activity },
    { label: '运行时间', value: coreUptime, color: 'text-amber-400', icon: HardDrive },
  ]

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between animate-fade-in-up">
        <div>
          <h2 className="text-xl font-semibold text-white">仪表盘</h2>
          <p className="text-slate-400 text-sm mt-0.5">网络状态与流量监控</p>
        </div>

        <div className="flex items-center gap-2">
          {!kernelInstalled && (
            <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-yellow-500/10 border border-yellow-500/30 text-yellow-400 text-xs animate-pulse-red">
              <AlertTriangle size={14} /> 内核未安装
            </div>
          )}

          <button
            onClick={toggleProxy}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 active:scale-95 ${
              proxyEnabled ? 'bg-green-600/20 text-green-400 border border-green-500/30' : 'bg-surface-700 text-slate-300 border border-surface-600'
            }`}
          >
            {proxyEnabled ? <Shield size={16} /> : <ShieldOff size={16} />}
            {proxyEnabled ? '代理已开启' : '代理已关闭'}
          </button>

          {isRunning ? (
            <button onClick={stopCore} className="btn-danger flex items-center gap-2"><Square size={16} /> 停止</button>
          ) : (
            <button onClick={startCore} disabled={isStarting} className="btn-primary flex items-center gap-2"><Play size={16} /> {isStarting ? '启动中...' : '启动'}</button>
          )}
          <button onClick={restartCore} className="btn-secondary flex items-center gap-2"><RefreshCw size={16} /> 重启</button>
        </div>
      </div>

      <div className="grid grid-cols-4 gap-4">
        {cards.map((c, i) => (
          <div key={c.label} className={`card animate-fade-in-up`} style={{ animationDelay: `${i * 0.08}s` }}>
            <div className="flex items-center gap-2 text-slate-400 text-xs mb-2">
              <c.icon size={14} className={c.color} />
              {c.label}
            </div>
            <p className={`text-lg font-mono font-semibold ${c.color}`}>{c.value}</p>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="card space-y-3 animate-fade-in-up" style={{ animationDelay: '0.32s' }}>
          <h3 className="text-sm font-medium text-slate-300">流量统计</h3>
          <div className="space-y-2">
            {[
              { label: '总上传', value: fmtBytes(trafficStats.totalUpload), color: 'text-blue-400' },
              { label: '总下载', value: fmtBytes(trafficStats.totalDownload), color: 'text-green-400' },
              { label: '发送数据包', value: trafficStats.packetsSent.toLocaleString(), color: 'text-slate-300' },
              { label: '接收数据包', value: trafficStats.packetsRecv.toLocaleString(), color: 'text-slate-300' },
            ].map((r) => (
              <div key={r.label} className="flex justify-between text-sm">
                <span className="text-slate-500">{r.label}</span>
                <span className={`font-mono ${r.color}`}>{r.value}</span>
              </div>
            ))}
          </div>
          <div className="h-1.5 bg-surface-700 rounded-full overflow-hidden">
            {trafficStats.totalDownload + trafficStats.totalUpload > 0 ? (
              <div className="flex h-full">
                <div className="bg-blue-500 h-full transition-all duration-500" style={{ width: `${(trafficStats.totalUpload/(trafficStats.totalUpload+trafficStats.totalDownload))*100}%` }} />
                <div className="bg-green-500 h-full transition-all duration-500" style={{ width: `${(trafficStats.totalDownload/(trafficStats.totalUpload+trafficStats.totalDownload))*100}%` }} />
              </div>
            ) : <div className="bg-surface-600 h-full w-full" />}
          </div>
          <div className="flex justify-between text-xs text-slate-500"><span>蓝:上传</span><span>绿:下载</span></div>
        </div>

        <div className="card space-y-3 animate-fade-in-up" style={{ animationDelay: '0.4s' }}>
          <h3 className="text-sm font-medium text-slate-300">连接信息</h3>
          {activeNode ? (
            <div className="space-y-2">
              {[
                { label: '当前节点', value: activeNode.name },
                { label: '协议', value: activeNode.protocol.toUpperCase(), color: 'text-primary-400' },
                { label: '地址', value: `${activeNode.address}:${activeNode.port}`, mono: true },
                { label: '传输', value: `${activeNode.network || 'tcp'}${activeNode.tls ? ' + TLS' : ''}` },
                {
                  label: '延迟', value: activeNode.latency > 0 ? `${activeNode.latency}ms` : 'N/A',
                  color: activeNode.latency > 0 ? (activeNode.latency < 100 ? 'text-green-400' : activeNode.latency < 300 ? 'text-yellow-400' : 'text-red-400') : 'text-slate-500'
                },
                { label: '核心状态', value: isRunning ? '运行中' : '已停止', color: isRunning ? 'text-green-400' : 'text-red-400' },
              ].map((r) => (
                <div key={r.label} className="flex justify-between text-sm">
                  <span className="text-slate-500">{r.label}</span>
                  <span className={`${r.color || 'text-slate-300'} ${r.mono ? 'font-mono' : ''}`}>{r.value}</span>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-6">
              <Globe size={32} className="text-slate-600 mx-auto mb-2" />
              <p className="text-slate-500 text-sm">未选择节点</p>
              <p className="text-slate-600 text-xs mt-1">前往服务器页面选择一个节点</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
