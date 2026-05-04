import { useState } from 'react'
import { useAppStore } from '../stores/appStore'
import type { ServerNode, ServerProtocol } from '../types'
import {
  Plus, Trash2, Edit3, Link, Copy, Check, Zap,
  Globe, Shield, X, Search, Gauge, Network, Info
} from 'lucide-react'

const protocolLabels: Record<ServerProtocol, string> = {
  vmess: 'VMess', vless: 'VLESS', trojan: 'Trojan', shadowsocks: 'Shadowsocks', socks: 'SOCKS', http: 'HTTP',
}
const protocolColors: Record<ServerProtocol, string> = {
  vmess: 'text-blue-400 bg-blue-400/10', vless: 'text-purple-400 bg-purple-400/10',
  trojan: 'text-amber-400 bg-amber-400/10', shadowsocks: 'text-emerald-400 bg-emerald-400/10',
  socks: 'text-slate-400 bg-slate-400/10', http: 'text-cyan-400 bg-cyan-400/10',
}

const emptyNode: ServerNode = {
  id: '', name: '', group: '默认分组', protocol: 'vmess',
  address: '', port: 443, uuid: '', password: '',
  security: 'auto', flow: '', network: 'tcp', path: '', host: '',
  tls: true, sni: '', fingerprint: 'chrome',
  publicKey: '', shortId: '', spiderX: '',
  allowInsecure: false, latency: 0, upload: 0, download: 0,
  createdAt: '', updatedAt: '',
}

export default function ServerList() {
  const nodes = useAppStore((s) => s.nodes)
  const groups = useAppStore((s) => s.groups)
  const activeNodeID = useAppStore((s) => s.activeNodeID)
  const nodeLatencies = useAppStore((s) => s.nodeLatencies)
  const addNode = useAppStore((s) => s.addNode)
  const updateNode = useAppStore((s) => s.updateNode)
  const deleteNode = useAppStore((s) => s.deleteNode)
  const selectNode = useAppStore((s) => s.selectNode)
  const importFromLink = useAppStore((s) => s.importFromLink)
  const testNodeLatency = useAppStore((s) => s.testNodeLatency)
  const testAllLatency = useAppStore((s) => s.testAllLatency)
  const getNodeIPs = useAppStore((s) => s.getNodeIPs)

  const [showEditor, setShowEditor] = useState(false)
  const [editingNode, setEditingNode] = useState<ServerNode>(emptyNode)
  const [isNew, setIsNew] = useState(true)
  const [importLink, setImportLink] = useState('')
  const [showImport, setShowImport] = useState(false)
  const [selectedGroup, setSelectedGroup] = useState('全部')
  const [searchTerm, setSearchTerm] = useState('')
  const [copiedID, setCopiedID] = useState('')
  const [testing, setTesting] = useState(false)
  const [ipsModal, setIpsModal] = useState<string[] | null>(null)
  const [ipsLoading, setIpsLoading] = useState(false)

  const filteredNodes = nodes.filter((n) => {
    if (selectedGroup !== '全部' && n.group !== selectedGroup) return false
    if (searchTerm) {
      const t = searchTerm.toLowerCase()
      return n.name.toLowerCase().includes(t) || n.address.toLowerCase().includes(t)
    }
    return true
  })

  const openAdd = () => { setEditingNode({ ...emptyNode }); setIsNew(true); setShowEditor(true) }
  const openEdit = (node: ServerNode) => { setEditingNode({ ...node }); setIsNew(false); setShowEditor(true) }

  const handleSave = async () => {
    if (!editingNode.name.trim() || !editingNode.address.trim()) return
    if (isNew) await addNode(editingNode); else await updateNode(editingNode)
    setShowEditor(false)
  }

  const handleDelete = async (id: string) => { await deleteNode(id) }
  const handleSelect = async (id: string) => { await selectNode(id) }

  const handleImport = async () => {
    if (!importLink.trim()) return
    try { await importFromLink(importLink.trim()); setImportLink(''); setShowImport(false) }
    catch { alert('导入失败，请检查链接格式') }
  }

  const copyLink = async (node: ServerNode) => {
    const link = `vmess://${btoa(JSON.stringify({ v: '2', ps: node.name, add: node.address, port: node.port, id: node.uuid, aid: 0, net: node.network, type: 'none', host: node.host, path: node.path, tls: node.tls ? 'tls' : '' }))}`
    await navigator.clipboard.writeText(link); setCopiedID(node.id); setTimeout(() => setCopiedID(''), 2000)
  }

  const handleTestLatency = async (id: string) => { setTesting(true); await testNodeLatency(id); setTesting(false) }
  const handleTestAll = async () => { setTesting(true); await testAllLatency(); setTesting(false) }

  const handleShowIPs = async (id: string) => {
    setIpsLoading(true); setIpsModal([])
    const ips = await getNodeIPs(id)
    setIpsModal(ips || []); setIpsLoading(false)
  }

  const getNodeLatency = (node: ServerNode) => {
    const lr = nodeLatencies[node.id]
    if (lr) return lr
    if (node.latency > 0) return { success: true, latency: node.latency }
    return null
  }

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between animate-fade-in-up">
        <div>
          <h2 className="text-xl font-semibold text-white">服务器</h2>
          <p className="text-slate-400 text-sm mt-0.5">{nodes.length} 个节点</p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => setShowImport(!showImport)} className="btn-secondary flex items-center gap-2"><Link size={16} /> 导入链接</button>
          <button onClick={handleTestAll} disabled={testing} className="btn-secondary flex items-center gap-2"><Gauge size={16} /> {testing ? '测试中...' : '测试全部'}</button>
          <button onClick={openAdd} className="btn-primary flex items-center gap-2"><Plus size={16} /> 添加节点</button>
        </div>
      </div>

      {showImport && (
        <div className="card flex gap-2 animate-fade-in-scale">
          <input type="text" value={importLink} onChange={(e) => setImportLink(e.target.value)}
            placeholder="粘贴 vmess:// 分享链接..." className="input-field flex-1"
            onKeyDown={(e) => e.key === 'Enter' && handleImport()} />
          <button onClick={handleImport} className="btn-primary">导入</button>
          <button onClick={() => setShowImport(false)} className="btn-ghost"><X size={16} /></button>
        </div>
      )}

      <div className="flex items-center gap-3">
        <div className="relative flex-1 max-w-xs">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
          <input type="text" value={searchTerm} onChange={(e) => setSearchTerm(e.target.value)} placeholder="搜索节点..." className="input-field pl-9" />
        </div>
        <div className="flex gap-1.5">
          {['全部', ...groups.map((g) => g.name)].map((g) => (
            <button key={g} onClick={() => setSelectedGroup(g)}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all duration-150 ${selectedGroup === g ? 'bg-primary-600/20 text-primary-400 border border-primary-500/30' : 'bg-surface-800 text-slate-400 border border-surface-700 hover:text-white'}`}>
              {g}
            </button>
          ))}
        </div>
      </div>

      <div className="space-y-2">
        {filteredNodes.length === 0 ? (
          <div className="card text-center py-12 animate-fade-in-scale">
            <Globe size={40} className="text-slate-600 mx-auto mb-3" />
            <p className="text-slate-400 text-sm">暂无服务器节点</p>
            <p className="text-slate-600 text-xs mt-1 mb-4">点击"添加节点"来创建第一个节点</p>
            <button onClick={openAdd} className="btn-primary inline-flex items-center gap-2 mx-auto"><Plus size={16} /> 添加节点</button>
          </div>
        ) : (
          filteredNodes.map((node, idx) => {
            const lr = getNodeLatency(node)
            const latencyColor = !lr?.success ? 'text-slate-500' : lr.latency < 0 ? 'text-red-400' : lr.latency < 100 ? 'text-green-400' : lr.latency < 300 ? 'text-yellow-400' : 'text-red-400'

            return (
              <div key={node.id}
                className={`card flex items-center gap-4 transition-all duration-200 animate-fade-in-up ${activeNodeID === node.id ? 'border-primary-500/50 bg-primary-500/5' : ''}`}
                style={{ animationDelay: `${idx * 0.03}s` }}>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-white text-sm font-medium truncate">{node.name}</span>
                    <span className={`badge ${protocolColors[node.protocol]}`}>{protocolLabels[node.protocol]}</span>
                    {activeNodeID === node.id && <span className="badge bg-green-500/20 text-green-400">使用中</span>}
                  </div>
                  <div className="flex items-center gap-3 mt-1 text-xs text-slate-500">
                    <span className="font-mono">{node.address}:{node.port}</span>
                    {node.network !== 'tcp' && <span className="uppercase">{node.network}</span>}
                    {node.tls && <span className="text-green-500/80 flex items-center gap-1"><Shield size={10} />TLS</span>}
                    <button onClick={() => handleTestLatency(node.id)} className="flex items-center gap-1 text-slate-500 hover:text-primary-400 transition-colors" title="测试延迟">
                      <Gauge size={11} />
                      <span className={latencyColor}>
                        {testing && !lr ? '...' : lr?.success ? `${lr.latency}ms` : lr ? '超时' : '-'}
                      </span>
                    </button>
                  </div>
                </div>

                <div className="flex items-center gap-1">
                  {activeNodeID !== node.id && (
                    <button onClick={() => handleSelect(node.id)} className="btn-ghost text-green-400 hover:text-green-300" title="设为活跃节点"><Zap size={14} /></button>
                  )}
                  <button onClick={() => handleShowIPs(node.id)} className="btn-ghost" title="查看IP"><Network size={14} /></button>
                  <button onClick={() => copyLink(node)} className="btn-ghost" title="复制链接">
                    {copiedID === node.id ? <Check size={14} className="text-green-400" /> : <Copy size={14} />}
                  </button>
                  <button onClick={() => openEdit(node)} className="btn-ghost" title="编辑"><Edit3 size={14} /></button>
                  <button onClick={() => handleDelete(node.id)} className="btn-ghost text-red-400 hover:text-red-300" title="删除"><Trash2 size={14} /></button>
                </div>
              </div>
            )
          })
        )}
      </div>

      {showEditor && (
        <NodeEditor node={editingNode} setNode={setEditingNode} onSave={handleSave} onClose={() => setShowEditor(false)} isNew={isNew} groups={groups} />
      )}

      {ipsModal !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 animate-fade-in" onClick={() => setIpsModal(null)}>
          <div className="bg-surface-900 border border-surface-700 rounded-xl w-96 shadow-2xl animate-fade-in-scale" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between p-4 border-b border-surface-700">
              <h3 className="text-white font-semibold flex items-center gap-2"><Network size={16} /> IP 地址信息</h3>
              <button onClick={() => setIpsModal(null)} className="btn-ghost"><X size={16} /></button>
            </div>
            <div className="p-4 max-h-60 overflow-y-auto">
              {ipsLoading ? (
                <div className="text-center py-6"><div className="w-6 h-6 border-2 border-primary-500 border-t-transparent rounded-full animate-spin mx-auto" /><p className="text-slate-400 text-xs mt-2">解析中...</p></div>
              ) : ipsModal.length === 0 ? (
                <p className="text-slate-500 text-sm text-center py-4">无解析结果</p>
              ) : (
                <div className="space-y-1">
                  {ipsModal.map((ip, i) => (
                    <div key={i} className="flex items-center gap-2 px-3 py-2 bg-surface-800 rounded-lg">
                      <span className="text-slate-500 text-xs w-8">{i + 1}.</span>
                      <span className="text-white text-sm font-mono">{ip}</span>
                      <button onClick={() => { navigator.clipboard.writeText(ip); }} className="btn-ghost ml-auto" title="复制"><Copy size={12} /></button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function NodeEditor({ node, setNode, onSave, onClose, isNew, groups }: {
  node: ServerNode; setNode: (n: ServerNode) => void; onSave: () => void; onClose: () => void; isNew: boolean; groups: { name: string; count: number }[]
}) {
  const [tab, setTab] = useState<'basic' | 'stream' | 'advanced'>('basic')
  const update = (partial: Partial<ServerNode>) => setNode({ ...node, ...partial })

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 animate-fade-in" onClick={onClose}>
      <div className="bg-surface-900 border border-surface-700 rounded-xl w-[560px] max-h-[80vh] overflow-hidden shadow-2xl animate-fade-in-scale" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between p-4 border-b border-surface-700">
          <h3 className="text-white font-semibold">{isNew ? '添加节点' : '编辑节点'}</h3>
          <button onClick={onClose} className="btn-ghost"><X size={16} /></button>
        </div>
        <div className="flex border-b border-surface-700">
          {(['basic', 'stream', 'advanced'] as const).map((t) => (
            <button key={t} onClick={() => setTab(t)} className={`px-4 py-2 text-sm font-medium transition-all ${tab === t ? 'text-primary-400 border-b-2 border-primary-500' : 'text-slate-400 hover:text-white'}`}>
              {{ basic: '基本', stream: '传输', advanced: '高级' }[t]}
            </button>
          ))}
        </div>
        <div className="p-4 space-y-3 max-h-[50vh] overflow-y-auto">
          {tab === 'basic' && (
            <>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-slate-400 text-xs mb-1 block">名称 *</label><input value={node.name} onChange={(e) => update({ name: e.target.value })} placeholder="节点名称" className="input-field" /></div>
                <div><label className="text-slate-400 text-xs mb-1 block">分组</label><select value={node.group} onChange={(e) => update({ group: e.target.value })} className="input-field"><option>默认分组</option>{groups.filter(g => g.name !== '默认分组').map(g => <option key={g.name}>{g.name} ({g.count})</option>)}</select></div>
              </div>
              <div><label className="text-slate-400 text-xs mb-1 block">协议</label><select value={node.protocol} onChange={(e) => update({ protocol: e.target.value as ServerProtocol })} className="input-field"><option value="vmess">VMess</option><option value="vless">VLESS</option><option value="trojan">Trojan</option><option value="shadowsocks">Shadowsocks</option><option value="socks">SOCKS</option><option value="http">HTTP</option></select></div>
              <div className="grid grid-cols-3 gap-3">
                <div className="col-span-2"><label className="text-slate-400 text-xs mb-1 block">地址 *</label><input value={node.address} onChange={(e) => update({ address: e.target.value })} placeholder="服务器地址" className="input-field" /></div>
                <div><label className="text-slate-400 text-xs mb-1 block">端口 *</label><input type="number" value={node.port} onChange={(e) => update({ port: parseInt(e.target.value) || 0 })} className="input-field" /></div>
              </div>
              {(node.protocol === 'vmess' || node.protocol === 'vless') && <div><label className="text-slate-400 text-xs mb-1 block">UUID</label><input value={node.uuid || ''} onChange={(e) => update({ uuid: e.target.value })} placeholder="UUID" className="input-field font-mono" /></div>}
              {(node.protocol === 'trojan' || node.protocol === 'shadowsocks') && <div><label className="text-slate-400 text-xs mb-1 block">密码</label><input value={node.password || ''} onChange={(e) => update({ password: e.target.value })} placeholder="密码" className="input-field" /></div>}
            </>
          )}
          {tab === 'stream' && (
            <>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-slate-400 text-xs mb-1 block">传输方式</label><select value={node.network || 'tcp'} onChange={(e) => update({ network: e.target.value })} className="input-field"><option value="tcp">TCP</option><option value="ws">WebSocket</option><option value="grpc">gRPC</option><option value="quic">QUIC</option><option value="dccp">DCCP</option></select></div>
                <div><label className="text-slate-400 text-xs mb-1 block">安全</label><select value={node.security || 'auto'} onChange={(e) => update({ security: e.target.value })} className="input-field"><option value="none">无</option><option value="auto">自动</option><option value="aes-128-gcm">AES-128-GCM</option><option value="chacha20-poly1305">ChaCha20-Poly1305</option></select></div>
              </div>
              <div className="flex items-center gap-3">
                <label className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={node.tls} onChange={(e) => update({ tls: e.target.checked })} className="w-4 h-4 accent-primary-500" /><span className="text-sm text-slate-300">启用 TLS</span></label>
                <label className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={node.allowInsecure} onChange={(e) => update({ allowInsecure: e.target.checked })} className="w-4 h-4 accent-primary-500" /><span className="text-sm text-slate-300">跳过证书验证</span></label>
              </div>
              {node.tls && (
                <div className="grid grid-cols-2 gap-3">
                  <div><label className="text-slate-400 text-xs mb-1 block">SNI</label><input value={node.sni || ''} onChange={(e) => update({ sni: e.target.value })} placeholder="服务器名称" className="input-field" /></div>
                  <div><label className="text-slate-400 text-xs mb-1 block">Fingerprint</label><select value={node.fingerprint || 'chrome'} onChange={(e) => update({ fingerprint: e.target.value })} className="input-field"><option value="chrome">Chrome</option><option value="firefox">Firefox</option><option value="safari">Safari</option><option value="random">Random</option><option value="randomized">Randomized</option><option value="none">None</option></select></div>
                </div>
              )}
              {node.network === 'ws' && <div className="grid grid-cols-2 gap-3"><div><label className="text-slate-400 text-xs mb-1 block">路径</label><input value={node.path || ''} onChange={(e) => update({ path: e.target.value })} placeholder="/path" className="input-field" /></div><div><label className="text-slate-400 text-xs mb-1 block">Host</label><input value={node.host || ''} onChange={(e) => update({ host: e.target.value })} className="input-field" /></div></div>}
              {node.network === 'grpc' && <div><label className="text-slate-400 text-xs mb-1 block">Service Name</label><input value={node.path || ''} onChange={(e) => update({ path: e.target.value })} className="input-field" /></div>}
            </>
          )}
          {tab === 'advanced' && node.protocol === 'vless' && (
            <div><label className="text-slate-400 text-xs mb-1 block">Flow</label><select value={node.flow || ''} onChange={(e) => update({ flow: e.target.value })} className="input-field"><option value="">无</option><option value="xtls-rprx-vision">XTLS Vision</option><option value="xtls-rprx-vision-udp443">Vision + UDP443</option></select></div>
          )}
        </div>
        <div className="flex justify-end gap-2 p-4 border-t border-surface-700">
          <button onClick={onClose} className="btn-secondary">取消</button>
          <button onClick={onSave} className="btn-primary">{isNew ? '添加' : '保存'}</button>
        </div>
      </div>
    </div>
  )
}
