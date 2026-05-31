import { useState } from 'react'
import { useAppStore } from '../stores/appStore'
import type { ServerNode, ServerProtocol, TLSKeyPair } from '../types'
import {
  Plus, Trash2, Edit3, Link, Copy, Check, Zap,
  Globe, Shield, X, Search, Gauge, Network, Info
} from 'lucide-react'

const protocolLabels: Record<string, string> = {
  vmess: 'VMess', vless: 'VLESS', trojan: 'Trojan', shadowsocks: 'SS',
  socks: 'SOCKS', http: 'HTTP', dns: 'DNS', freedom: 'Freedom',
  blackhole: 'Blackhole', wireguard: 'WireGuard',
}
const protocolColors: Record<string, string> = {
  vmess: 'text-blue-400 bg-blue-400/10', vless: 'text-purple-400 bg-purple-400/10',
  trojan: 'text-amber-400 bg-amber-400/10', shadowsocks: 'text-emerald-400 bg-emerald-400/10',
  socks: 'text-slate-400 bg-slate-400/10', http: 'text-cyan-400 bg-cyan-400/10',
  dns: 'text-indigo-400 bg-indigo-400/10', freedom: 'text-teal-400 bg-teal-400/10',
  blackhole: 'text-red-400 bg-red-400/10', wireguard: 'text-pink-400 bg-pink-400/10',
}

const invoke = (method: string, ...args: any[]) => {
  const fn = method.split('.').reduce((obj: any, key) => obj?.[key], window.go?.main)
  if (!fn) return Promise.reject(new Error('Not ready'))
  return fn(...args)
}

const emptyNode: ServerNode = {
  id: '', name: '', group: '默认分组', protocol: 'vmess',
  address: '', port: 443, address6: '', port6: 443, ipv6: false, multiLine: false,
  uuid: '', password: '', security: 'auto', flow: '', network: 'tcp', path: '', host: '',
  tls: true, sni: '', fingerprint: 'chrome',
  publicKey: '', shortId: '', spiderX: '', allowInsecure: false,
  bbrType: 'default', tlsCertFile: '', tlsKeyFile: '',
  mkcpHeader: 'none', mkcpSeed: '', xhttpMode: 'auto', xhttpPath: '',
  quicSecurity: 'none', quicKey: '', headerType: 'none',
  h2Path: '/', h2Host: '', dsUseDomain: false,
  wireguardPriv: '', wireguardPub: '', wireguardPeers: '', ssMethod: 'aes-256-gcm',
  latency: 0, upload: 0, download: 0, createdAt: '', updatedAt: '',
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
    if (searchTerm) { const t = searchTerm.toLowerCase(); return n.name.toLowerCase().includes(t) || n.address.toLowerCase().includes(t) }
    return true
  })

  const openAdd = () => { setEditingNode({ ...emptyNode }); setIsNew(true); setShowEditor(true) }
  const openEdit = (node: ServerNode) => { setEditingNode({ ...node }); setIsNew(false); setShowEditor(true) }
  const handleSave = async () => {
    if (!editingNode.name.trim() || !editingNode.address.trim()) return
    if (isNew) await addNode(editingNode); else await updateNode(editingNode)
    setShowEditor(false)
  }
  const handleImport = async () => {
    if (!importLink.trim()) return
    try { await importFromLink(importLink.trim()); setImportLink(''); setShowImport(false) } catch { alert('导入失败') }
  }
  const copyLink = async (node: ServerNode) => {
    const link = `vmess://${btoa(JSON.stringify({ v: '2', ps: node.name, add: node.address, port: node.port, id: node.uuid, aid: 0, net: node.network, type: 'none', host: node.host, path: node.path, tls: node.tls ? 'tls' : '' }))}`
    await navigator.clipboard.writeText(link); setCopiedID(node.id); setTimeout(() => setCopiedID(''), 2000)
  }
  const handleTestLatency = async (id: string) => { setTesting(true); await testNodeLatency(id); setTesting(false) }
  const handleTestAll = async () => { setTesting(true); await testAllLatency(); setTesting(false) }
  const handleShowIPs = async (id: string) => { setIpsLoading(true); setIpsModal([]); const ips = await getNodeIPs(id); setIpsModal(ips || []); setIpsLoading(false) }
  const getNodeLatency = (node: ServerNode) => {
    const lr = nodeLatencies[node.id]
    if (lr) return lr
    if (node.latency > 0) return { success: true, latency: node.latency }
    return null
  }

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between animate-fade-in-up">
        <div><h2 className="text-xl font-semibold" style={{ color: 'var(--text-primary)' }}>服务器</h2><p className="text-sm mt-0.5" style={{ color: 'var(--text-secondary)' }}>{nodes.length} 个节点</p></div>
        <div className="flex items-center gap-2">
          <button onClick={() => setShowImport(!showImport)} className="btn-secondary flex items-center gap-2"><Link size={16} /> 导入</button>
          <button onClick={handleTestAll} disabled={testing} className="btn-secondary flex items-center gap-2"><Gauge size={16} /> {testing ? '测试中...' : '测试全部'}</button>
          <button onClick={openAdd} className="btn-primary flex items-center gap-2"><Plus size={16} /> 添加</button>
        </div>
      </div>
      {showImport && (
        <div className="glass-card flex gap-2 animate-fade-in-scale">
          <input value={importLink} onChange={(e) => setImportLink(e.target.value)} placeholder="粘贴 vmess:// 分享链接..." className="input-field flex-1" onKeyDown={(e) => e.key === 'Enter' && handleImport()} />
          <button onClick={handleImport} className="btn-primary">导入</button>
          <button onClick={() => setShowImport(false)} className="btn-ghost"><X size={16} /></button>
        </div>
      )}
      <div className="flex items-center gap-3">
        <div className="relative flex-1 max-w-xs"><Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2" style={{ color: 'var(--text-muted)' }} /><input value={searchTerm} onChange={(e) => setSearchTerm(e.target.value)} placeholder="搜索节点..." className="input-field pl-9" /></div>
        <div className="flex gap-1.5">{['全部', ...groups.map((g) => g.name)].map((g) => <button key={g} onClick={() => setSelectedGroup(g)} className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${selectedGroup === g ? 'tab-btn active' : 'tab-btn'}`}>{g}</button>)}</div>
      </div>
      <div className="space-y-2">
        {filteredNodes.length === 0 ? (
          <div className="card text-center py-12"><Globe size={40} className="mx-auto mb-3" style={{ color: 'var(--text-muted)' }} /><p style={{ color: 'var(--text-secondary)' }} className="text-sm">暂无节点</p><button onClick={openAdd} className="btn-primary inline-flex items-center gap-2 mx-auto mt-4"><Plus size={16} /> 添加节点</button></div>
        ) : filteredNodes.map((node, idx) => {
          const lr = getNodeLatency(node)
          const latColor = !lr?.success ? 'var(--text-muted)' : lr.latency < 0 ? 'var(--red)' : lr.latency < 100 ? 'var(--green)' : lr.latency < 300 ? 'var(--yellow)' : 'var(--red)'
          return (
            <div key={node.id} className={`glass-card flex items-center gap-4 ${activeNodeID === node.id ? 'border-primary-500/50' : ''}`} style={{ animationDelay: `${idx * 0.03}s`, ...(activeNodeID === node.id ? { borderColor: 'var(--accent)' } : {}) }}>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2"><span className="text-sm font-medium truncate" style={{ color: 'var(--text-primary)' }}>{node.name}</span><span className={`badge ${protocolColors[node.protocol] || ''}`}>{protocolLabels[node.protocol] || node.protocol}</span>{activeNodeID === node.id && <span className="badge" style={{ color: 'var(--green)', background: 'rgba(34,197,94,0.15)' }}>使用中</span>}{node.ipv6 && <span className="badge" style={{ color: '#60a5fa', background: 'rgba(96,165,250,0.12)' }}>IPv6</span>}{node.multiLine && <span className="badge" style={{ color: '#f472b6', background: 'rgba(244,114,182,0.12)' }}>双栈</span>}</div>
                <div className="flex items-center gap-3 mt-1 text-xs" style={{ color: 'var(--text-muted)' }}><span className="font-mono">{node.address}:{node.port}</span>{node.network !== 'tcp' && <span className="uppercase">{node.network}</span>}{node.tls && <span className="flex items-center gap-1"><Shield size={10} />TLS</span>}<button onClick={() => handleTestLatency(node.id)} className="flex items-center gap-1 hover:text-primary-400 transition-colors"><Gauge size={11} /><span style={{ color: latColor }}>{testing && !lr ? '...' : lr?.success ? `${lr.latency}ms` : lr ? '超时' : '-'}</span></button></div>
              </div>
              <div className="flex items-center gap-1">
                {activeNodeID !== node.id && <button onClick={() => selectNode(node.id)} className="btn-ghost" style={{ color: 'var(--green)' }}><Zap size={14} /></button>}
                <button onClick={() => handleShowIPs(node.id)} className="btn-ghost"><Network size={14} /></button>
                <button onClick={() => copyLink(node)} className="btn-ghost">{copiedID === node.id ? <Check size={14} style={{ color: 'var(--green)' }} /> : <Copy size={14} />}</button>
                <button onClick={() => openEdit(node)} className="btn-ghost"><Edit3 size={14} /></button>
                <button onClick={() => deleteNode(node.id)} className="btn-ghost" style={{ color: 'var(--red)' }}><Trash2 size={14} /></button>
              </div>
            </div>
          )
        })}
      </div>
      {showEditor && <NodeEditor node={editingNode} setNode={setEditingNode} onSave={handleSave} onClose={() => setShowEditor(false)} isNew={isNew} groups={groups} />}
      {ipsModal !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 animate-fade-in" onClick={() => setIpsModal(null)}>
          <div className="glass rounded-xl w-96 shadow-2xl animate-fade-in-scale" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between p-4 border-b" style={{ borderColor: 'var(--border)' }}><h3 className="font-semibold flex items-center gap-2" style={{ color: 'var(--text-primary)' }}><Network size={16} /> IP 信息</h3><button onClick={() => setIpsModal(null)} className="btn-ghost"><X size={16} /></button></div>
            <div className="p-4 max-h-60 overflow-y-auto">{ipsLoading ? <div className="text-center py-6"><div className="w-6 h-6 border-2 border-t-transparent rounded-full animate-spin mx-auto" style={{ borderColor: 'var(--accent)', borderTopColor: 'transparent' }} /></div> : ipsModal.map((ip, i) => <div key={i} className="flex items-center gap-2 px-3 py-2 rounded-lg" style={{ background: 'var(--bg-secondary)' }}><span className="text-xs w-8" style={{ color: 'var(--text-muted)' }}>{i + 1}.</span><span className="text-sm font-mono" style={{ color: 'var(--text-primary)' }}>{ip}</span></div>)}</div>
          </div>
        </div>
      )}
    </div>
  )
}

function NodeEditor({ node, setNode, onSave, onClose, isNew, groups }: {
  node: ServerNode; setNode: (n: ServerNode) => void; onSave: () => void; onClose: () => void; isNew: boolean; groups: { name: string; count: number }[]
}) {
  const [tab, setTab] = useState<'basic' | 'stream' | 'tls' | 'ipv6' | 'bbr'>('basic')
  const [tlsLoading, setTlsLoading] = useState(false)
  const update = (partial: Partial<ServerNode>) => setNode({ ...node, ...partial })

  const tabs = [
    { id: 'basic' as const, label: '基本' },
    { id: 'stream' as const, label: '传输' },
    { id: 'tls' as const, label: 'TLS' },
    { id: 'ipv6' as const, label: 'IPv6' },
    { id: 'bbr' as const, label: 'BBR' },
  ]

  const generateTLS = async () => {
    setTlsLoading(true)
    try {
      const result = await invoke('App.GenerateTLS', node.sni || node.address || 'oiwest.local') as TLSKeyPair
      if (result && result.certFile) update({ tlsCertFile: result.certFile, tlsKeyFile: result.keyFile })
    } catch (e) { alert('TLS 证书生成失败') }
    setTlsLoading(false)
  }
  const generateReality = async () => {
    try {
      const keys = await invoke('App.GenerateRealityKeys') as Record<string, string>
      if (keys?.publicKey) update({ publicKey: keys.publicKey })
    } catch { alert('Reality 密钥生成失败') }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 animate-fade-in" onClick={onClose}>
      <div className="glass rounded-xl w-[640px] max-h-[85vh] overflow-hidden shadow-2xl animate-fade-in-scale" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between p-4 border-b" style={{ borderColor: 'var(--border)' }}>
          <h3 className="font-semibold" style={{ color: 'var(--text-primary)' }}>{isNew ? '添加节点' : '编辑节点'}</h3>
          <button onClick={onClose} className="btn-ghost"><X size={16} /></button>
        </div>
        <div className="flex border-b gap-0 overflow-x-auto" style={{ borderColor: 'var(--border)' }}>
          {tabs.map(({ id, label }) => (
            <button key={id} onClick={() => setTab(id)} className={`px-4 py-2 text-sm font-medium transition-all whitespace-nowrap shrink-0 ${tab === id ? 'border-b-2' : ''}`}
              style={{ color: tab === id ? 'var(--accent)' : 'var(--text-secondary)', borderColor: tab === id ? 'var(--accent)' : 'transparent' }}>
              {label}
            </button>
          ))}
        </div>
        <div className="p-4 space-y-3 max-h-[55vh] overflow-y-auto">
          {tab === 'basic' && (
            <>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>名称 *</label><input value={node.name} onChange={(e) => update({ name: e.target.value })} className="input-field" /></div>
                <div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>分组</label><select value={node.group} onChange={(e) => update({ group: e.target.value })} className="input-field custom-select"><option>默认分组</option>{groups.filter(g => g.name !== '默认分组').map(g => <option key={g.name}>{g.name} ({g.count})</option>)}</select></div>
              </div>
              <div className="custom-select"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>协议</label><select value={node.protocol} onChange={(e) => update({ protocol: e.target.value as ServerProtocol })} className="input-field"><option value="vmess">VMess</option><option value="vless">VLESS</option><option value="trojan">Trojan</option><option value="shadowsocks">Shadowsocks</option><option value="socks">SOCKS</option><option value="http">HTTP</option><option value="dns">DNS</option><option value="freedom">Freedom</option><option value="blackhole">Blackhole</option><option value="wireguard">WireGuard</option></select></div>
              <div className="grid grid-cols-3 gap-3">
                <div className="col-span-2"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>地址 *</label><input value={node.address} onChange={(e) => update({ address: e.target.value })} className="input-field" /></div>
                <div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>端口 *</label><input type="number" value={node.port} onChange={(e) => update({ port: parseInt(e.target.value) || 0 })} className="input-field" /></div>
              </div>
              {(node.protocol === 'vmess' || node.protocol === 'vless') && <div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>UUID</label><input value={node.uuid || ''} onChange={(e) => update({ uuid: e.target.value })} className="input-field font-mono" /></div>}
              {(node.protocol === 'trojan' || node.protocol === 'shadowsocks') && <div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>密码</label><input value={node.password || ''} onChange={(e) => update({ password: e.target.value })} className="input-field" /></div>}
              {node.protocol === 'shadowsocks' && <div className="custom-select"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>加密方法</label><select value={node.ssMethod || 'aes-256-gcm'} onChange={(e) => update({ ssMethod: e.target.value })} className="input-field"><option value="aes-256-gcm">AES-256-GCM</option><option value="chacha20-ietf-poly1305">ChaCha20-IETF-Poly1305</option><option value="2022-blake3-aes-256-gcm">2022-Blake3-AES-256-GCM</option><option value="none">None</option></select></div>}
              {node.protocol === 'wireguard' && <><div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>私钥</label><input value={node.wireguardPriv || ''} onChange={(e) => update({ wireguardPriv: e.target.value })} className="input-field font-mono text-xs" /></div><div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>对方公钥</label><input value={node.wireguardPub || ''} onChange={(e) => update({ wireguardPub: e.target.value })} className="input-field font-mono text-xs" /></div></>}
            </>
          )}
          {tab === 'stream' && (
            <>
              <div className="grid grid-cols-2 gap-3">
                <div className="custom-select"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>传输方式</label><select value={node.network || 'tcp'} onChange={(e) => update({ network: e.target.value })} className="input-field">
                  <option value="tcp">TCP</option><option value="ws">WebSocket</option><option value="grpc">gRPC</option><option value="quic">QUIC</option><option value="dccp">DCCP</option><option value="mkcp">mKCP</option><option value="h2">HTTP/2</option><option value="xhttp">XHTTP</option>
                </select></div>
                <div className="custom-select"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>安全</label><select value={node.security || 'auto'} onChange={(e) => update({ security: e.target.value })} className="input-field"><option value="none">无</option><option value="auto">自动</option><option value="aes-128-gcm">AES-128-GCM</option><option value="chacha20-poly1305">ChaCha20-Poly1305</option></select></div>
              </div>
              {node.network === 'ws' && <div className="grid grid-cols-2 gap-3"><div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>路径</label><input value={node.path || ''} onChange={(e) => update({ path: e.target.value })} className="input-field" /></div><div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>Host</label><input value={node.host || ''} onChange={(e) => update({ host: e.target.value })} className="input-field" /></div></div>}
              {node.network === 'grpc' && <div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>Service Name</label><input value={node.path || ''} onChange={(e) => update({ path: e.target.value })} className="input-field" /></div>}
              {node.network === 'quic' && <div className="grid grid-cols-2 gap-3"><div className="custom-select"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>QUIC 安全</label><select value={node.quicSecurity || 'none'} onChange={(e) => update({ quicSecurity: e.target.value })} className="input-field"><option value="none">None</option><option value="aes-128-gcm">AES-128-GCM</option><option value="chacha20-poly1305">ChaCha20-Poly1305</option></select></div><div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>Key</label><input value={node.quicKey || ''} onChange={(e) => update({ quicKey: e.target.value })} className="input-field" /></div></div>}
              {node.network === 'mkcp' && <div className="grid grid-cols-2 gap-3"><div className="custom-select"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>mKCP Header</label><select value={node.mkcpHeader || 'none'} onChange={(e) => update({ mkcpHeader: e.target.value })} className="input-field"><option value="none">None</option><option value="srtp">SRTP</option><option value="utp">uTP</option><option value="wechat-video">微信视频</option><option value="dtls">DTLS</option><option value="wireguard">WireGuard</option></select></div><div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>Seed</label><input value={node.mkcpSeed || ''} onChange={(e) => update({ mkcpSeed: e.target.value })} className="input-field" /></div></div>}
              {node.network === 'h2' && <div className="grid grid-cols-2 gap-3"><div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>H2 Path</label><input value={node.h2Path || '/'} onChange={(e) => update({ h2Path: e.target.value })} className="input-field" /></div><div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>H2 Host</label><input value={node.h2Host || ''} onChange={(e) => update({ h2Host: e.target.value })} className="input-field" /></div></div>}
              {node.network === 'xhttp' && <div className="grid grid-cols-2 gap-3"><div className="custom-select"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>XHTTP Mode</label><select value={node.xhttpMode || 'auto'} onChange={(e) => update({ xhttpMode: e.target.value })} className="input-field"><option value="auto">Auto</option><option value="packet-up">Packet-Up</option><option value="stream-up">Stream-Up</option><option value="stream-one">Stream-One</option></select></div><div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>XHTTP Path</label><input value={node.xhttpPath || ''} onChange={(e) => update({ xhttpPath: e.target.value })} className="input-field" /></div></div>}
              {node.network === 'tcp' && <div className="grid grid-cols-2 gap-3"><div className="custom-select"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>TCP Header</label><select value={node.headerType || 'none'} onChange={(e) => update({ headerType: e.target.value })} className="input-field"><option value="none">None</option><option value="http">HTTP</option></select></div><label className="flex items-center gap-2 cursor-pointer mt-5"><input type="checkbox" checked={node.dsUseDomain || false} onChange={(e) => update({ dsUseDomain: e.target.checked })} className="w-4 h-4" style={{ accentColor: 'var(--accent)' }} /><span className="text-sm" style={{ color: 'var(--text-secondary)' }}>域名策略</span></label></div>}
            </>
          )}
          {tab === 'tls' && (
            <>
              <div className="flex items-center gap-3 mb-3">
                <label className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={node.tls} onChange={(e) => update({ tls: e.target.checked })} className="w-4 h-4" style={{ accentColor: 'var(--accent)' }} /><span className="text-sm" style={{ color: 'var(--text-secondary)' }}>启用 TLS</span></label>
                <label className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={node.allowInsecure} onChange={(e) => update({ allowInsecure: e.target.checked })} className="w-4 h-4" style={{ accentColor: 'var(--accent)' }} /><span className="text-sm" style={{ color: 'var(--text-secondary)' }}>跳过证书验证</span></label>
              </div>
              {node.tls && <><div className="grid grid-cols-2 gap-3"><div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>SNI</label><input value={node.sni || ''} onChange={(e) => update({ sni: e.target.value })} className="input-field" /></div><div className="custom-select"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>Fingerprint</label><select value={node.fingerprint || 'chrome'} onChange={(e) => update({ fingerprint: e.target.value })} className="input-field"><option value="chrome">Chrome</option><option value="firefox">Firefox</option><option value="safari">Safari</option><option value="edge">Edge</option><option value="random">Random</option><option value="randomized">Randomized</option><option value="none">None</option></select></div></div>
                <div className="bg-surface-800/50 rounded-lg p-3 space-y-2">
                  <p className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>TLS 证书</p>
                  <div className="grid grid-cols-2 gap-3"><div><label className="text-xs mb-1 block" style={{ color: 'var(--text-muted)' }}>证书文件</label><input value={node.tlsCertFile || ''} onChange={(e) => update({ tlsCertFile: e.target.value })} className="input-field text-xs" placeholder="自动生成" /></div><div><label className="text-xs mb-1 block" style={{ color: 'var(--text-muted)' }}>密钥文件</label><input value={node.tlsKeyFile || ''} onChange={(e) => update({ tlsKeyFile: e.target.value })} className="input-field text-xs" placeholder="自动生成" /></div></div>
                  <button onClick={generateTLS} disabled={tlsLoading} className="btn-secondary text-xs flex items-center gap-2">{tlsLoading ? '生成中...' : '自动生成 TLS 证书'}</button>
                </div></>}
              <div className="bg-surface-800/50 rounded-lg p-3 space-y-2 mt-3">
                <p className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>Reality 配置</p>
                <div className="grid grid-cols-2 gap-3"><div><label className="text-xs mb-1 block" style={{ color: 'var(--text-muted)' }}>Public Key</label><input value={node.publicKey || ''} onChange={(e) => update({ publicKey: e.target.value })} className="input-field font-mono text-xs" /></div><div><label className="text-xs mb-1 block" style={{ color: 'var(--text-muted)' }}>Short ID</label><input value={node.shortId || ''} onChange={(e) => update({ shortId: e.target.value })} className="input-field text-xs" /></div></div>
                <div className="grid grid-cols-2 gap-3"><div><label className="text-xs mb-1 block" style={{ color: 'var(--text-muted)' }}>SpiderX</label><input value={node.spiderX || ''} onChange={(e) => update({ spiderX: e.target.value })} className="input-field text-xs" /></div></div>
                <button onClick={generateReality} className="btn-secondary text-xs flex items-center gap-2">生成 Reality 密钥对</button>
              </div>
            </>
          )}
          {tab === 'ipv6' && (
            <>
              <label className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={node.ipv6} onChange={(e) => update({ ipv6: e.target.checked })} className="w-4 h-4" style={{ accentColor: 'var(--accent)' }} /><span className="text-sm" style={{ color: 'var(--text-secondary)' }}>启用 IPv6 支持</span></label>
              {node.ipv6 && <>
                <div className="grid grid-cols-3 gap-3">
                  <div className="col-span-2"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>IPv6 地址</label><input value={node.address6 || ''} onChange={(e) => update({ address6: e.target.value })} className="input-field" /></div>
                  <div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>IPv6 端口</label><input type="number" value={node.port6 || 443} onChange={(e) => update({ port6: parseInt(e.target.value) || 443 })} className="input-field" /></div>
                </div>
                <label className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={node.multiLine} onChange={(e) => update({ multiLine: e.target.checked })} className="w-4 h-4" style={{ accentColor: 'var(--accent)' }} /><span className="text-sm" style={{ color: 'var(--text-secondary)' }}>多线路传输（同时使用 IPv4 + IPv6）</span></label>
              </>}
            </>
          )}
          {tab === 'bbr' && (
            <>
              <div className="custom-select"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>BBR 拥塞控制</label><select value={node.bbrType || 'default'} onChange={(e) => update({ bbrType: e.target.value })} className="input-field"><option value="default">默认（不启用）</option><option value="bbr">BBR v1</option><option value="bbr2">BBR v2</option><option value="bbr3">BBR v3</option><option value="bbr_tso">BBR TSO</option><option value="bbr_pacing">BBR Pacing</option></select></div>
              <p className="text-xs" style={{ color: 'var(--text-muted)' }}>BBR (Bottleneck Bandwidth and Round-trip) 拥塞控制算法可显著提升高延迟/高丢包网络环境下的传输性能。推荐在高延迟跨境线路中使用 BBR v3。</p>
              {node.protocol === 'vless' && <div className="custom-select mt-3"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>VLESS Flow</label><select value={node.flow || ''} onChange={(e) => update({ flow: e.target.value })} className="input-field"><option value="">无</option><option value="xtls-rprx-vision">XTLS Vision</option><option value="xtls-rprx-vision-udp443">Vision + UDP443</option></select></div>}
            </>
          )}
        </div>
        <div className="flex justify-end gap-2 p-4 border-t" style={{ borderColor: 'var(--border)' }}>
          <button onClick={onClose} className="btn-secondary">取消</button>
          <button onClick={onSave} className="btn-primary">{isNew ? '添加' : '保存'}</button>
        </div>
      </div>
    </div>
  )
}
