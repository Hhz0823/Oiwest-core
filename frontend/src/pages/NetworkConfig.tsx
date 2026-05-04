import { useState } from 'react'
import { useAppStore } from '../stores/appStore'
import { useT } from '../stores/settingsStore'
import type { InboundRule, RoutingRule, DNSConfig, DNSServerItem, TransparentProxyConfig } from '../types'
import { Plus, Trash2, Edit3, X, Globe, ArrowUp, ArrowDown, Server, Radio } from 'lucide-react'

export default function NetworkConfig() {
  const [tab, setTab] = useState<'inbound' | 'routing' | 'dns' | 'tproxy'>('inbound')
  const { t } = useT()

  const tabs = [
    { id: 'inbound' as const, label: t('network.inbound'), icon: Server },
    { id: 'routing' as const, label: t('network.routing'), icon: ArrowUp },
    { id: 'dns' as const, label: t('network.dns'), icon: Globe },
    { id: 'tproxy' as const, label: t('network.tproxy'), icon: Radio },
  ]

  return (
    <div className="p-6 space-y-4">
      <div className="page-section">
        <h2 className="text-xl font-semibold" style={{ color: 'var(--text-primary)' }}>{t('network.title')}</h2>
        <p className="text-sm mt-0.5" style={{ color: 'var(--text-secondary)' }}>{t('network.subtitle')}</p>
      </div>
      <div className="flex gap-1.5 flex-wrap">
        {tabs.map(({ id, label, icon: Icon }) => (
          <button key={id} onClick={() => setTab(id)} className={`tab-btn ${tab === id ? 'active' : ''}`}>
            <Icon size={14} /> {label}
          </button>
        ))}
      </div>
      <div className="tab-content" key={tab}>
        {tab === 'inbound' && <InboundPanel />}
        {tab === 'routing' && <RoutingPanel />}
        {tab === 'dns' && <DNSPanel />}
        {tab === 'tproxy' && <TProxyPanel />}
      </div>
    </div>
  )
}

function InboundPanel() {
  const inbounds = useAppStore((s) => s.inbounds) || []
  const addInbound = useAppStore((s) => s.addInbound)
  const updateInbound = useAppStore((s) => s.updateInbound)
  const deleteInbound = useAppStore((s) => s.deleteInbound)
  const { t } = useT()
  const [editing, setEditing] = useState<InboundRule | null>(null)
  const [isNew, setIsNew] = useState(false)

  const empty: InboundRule = { id: '', tag: 'custom-in', port: 10810, listen: '127.0.0.1', protocol: 'socks', enabled: true, settings: { auth: 'noauth', udp: true, user: '', pass: '', method: '', password: '' } }
  const save = async () => { if (!editing) return; if (isNew) await addInbound(editing); else await updateInbound(editing); setEditing(null) }

  return (
    <div className="space-y-2">
      <div className="flex justify-end">
        <button onClick={() => { setEditing({ ...empty }); setIsNew(true) }} className="btn-primary flex items-center gap-2"><Plus size={16} /> {t('network.addInbound')}</button>
      </div>
      {inbounds.map((r, i) => (
        <div key={r.id} className="card flex items-center gap-4" style={{ animationDelay: `${i * 0.04}s` }}>
          <div className="flex-1">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>{r.tag}</span>
              <span className="badge uppercase" style={{ color: 'var(--accent)', background: 'color-mix(in srgb, var(--accent) 12%, transparent)' }}>{r.protocol}</span>
              {r.enabled ? <span className="badge" style={{ color: 'var(--green)', background: 'rgba(34,197,94,0.15)' }}>{t('network.enabled')}</span> : <span className="badge" style={{ color: 'var(--red)', background: 'rgba(239,68,68,0.15)' }}>{t('network.disabled')}</span>}
            </div>
            <p className="text-xs mt-0.5" style={{ color: 'var(--text-muted)' }}>{r.listen}:{r.port} · UDP: {r.settings.udp ? t('network.enabled') : t('network.disabled')}</p>
          </div>
          <button onClick={() => { setEditing({ ...r }); setIsNew(false) }} className="btn-ghost"><Edit3 size={14} /></button>
          <button onClick={() => deleteInbound(r.id)} className="btn-ghost" style={{ color: 'var(--red)' }}><Trash2 size={14} /></button>
        </div>
      ))}
      {editing && (
        <Modal title={isNew ? t('network.addInboundTitle') : t('network.editInbound')} onClose={() => setEditing(null)}
          footer={<><button onClick={() => setEditing(null)} className="btn-secondary">{t('servers.cancel')}</button><button onClick={save} className="btn-primary">{t('servers.save')}</button></>}>
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>{t('network.inboundTag')}</label><input value={editing.tag} onChange={(e) => setEditing({ ...editing, tag: e.target.value })} className="input-field" /></div>
              <div className="custom-select"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>{t('servers.protocol')}</label><select value={editing.protocol} onChange={(e) => setEditing({ ...editing, protocol: e.target.value })} className="input-field"><option value="socks">SOCKS</option><option value="http">HTTP</option><option value="vmess">VMess</option><option value="vless">VLESS</option></select></div>
            </div>
            <div className="grid grid-cols-3 gap-3">
              <div className="col-span-2"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>{t('network.inboundListen')}</label><input value={editing.listen} onChange={(e) => setEditing({ ...editing, listen: e.target.value })} className="input-field" /></div>
              <div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>{t('servers.port')}</label><input type="number" value={editing.port} onChange={(e) => setEditing({ ...editing, port: parseInt(e.target.value) || 0 })} className="input-field" /></div>
            </div>
            <label className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={editing.settings.udp} onChange={(e) => setEditing({ ...editing, settings: { ...editing.settings, udp: e.target.checked } })} className="w-4 h-4" style={{ accentColor: 'var(--accent)' }} /><span className="text-sm" style={{ color: 'var(--text-secondary)' }}>{t('network.udpSupport')}</span></label>
          </div>
        </Modal>
      )}
    </div>
  )
}

function RoutingPanel() {
  const routingRules = useAppStore((s) => s.routingRules) || []
  const addRoutingRule = useAppStore((s) => s.addRoutingRule)
  const updateRoutingRule = useAppStore((s) => s.updateRoutingRule)
  const deleteRoutingRule = useAppStore((s) => s.deleteRoutingRule)
  const reorderRoutingRules = useAppStore((s) => s.reorderRoutingRules)
  const { t } = useT()
  const [editing, setEditing] = useState<RoutingRule | null>(null)
  const [isNew, setIsNew] = useState(false)

  const empty: RoutingRule = { id: '', name: '', type: 'field', domain: [], ip: [], port: '', network: '', protocol: [], inboundTag: [], outboundTag: 'proxy', enabled: true, sort: 99 }
  const save = async () => { if (!editing || !editing.name.trim()) return; if (isNew) await addRoutingRule(editing); else await updateRoutingRule(editing); setEditing(null) }

  const moveUp = async (idx: number) => {
    if (idx === 0 || !Array.isArray(routingRules)) return
    const ids = routingRules.map(r => r.id); [ids[idx - 1], ids[idx]] = [ids[idx], ids[idx - 1]]; await reorderRoutingRules(ids)
  }
  const moveDown = async (idx: number) => {
    if (!Array.isArray(routingRules) || idx >= routingRules.length - 1) return
    const ids = routingRules.map(r => r.id); [ids[idx], ids[idx + 1]] = [ids[idx + 1], ids[idx]]; await reorderRoutingRules(ids)
  }

  const setDomains = (d: string) => setEditing(editing ? { ...editing, domain: d ? d.split(',').map(s => s.trim()).filter(Boolean) : [] } : editing)
  const setIPs = (d: string) => setEditing(editing ? { ...editing, ip: d ? d.split(',').map(s => s.trim()).filter(Boolean) : [] } : editing)

  return (
    <div className="space-y-2">
      <div className="flex justify-end">
        <button onClick={() => { setEditing({ ...empty }); setIsNew(true) }} className="btn-primary flex items-center gap-2"><Plus size={16} /> {t('network.addRule')}</button>
      </div>
      {!Array.isArray(routingRules) || routingRules.length === 0 ? (
        <div className="card text-center py-8">
          <ArrowUp size={32} className="mx-auto mb-2" style={{ color: 'var(--text-muted)' }} />
          <p className="text-sm" style={{ color: 'var(--text-secondary)' }}>{t('network.noRoutes')}</p>
          <p className="text-xs mt-1" style={{ color: 'var(--text-muted)' }}>{t('network.noRoutesHint')}</p>
        </div>
      ) : (
        routingRules.map((r, i) => (
          <div key={r.id || i} className="card flex items-center gap-3">
            <div className="flex flex-col gap-0.5">
              <button onClick={() => moveUp(i)} disabled={i === 0} className="btn-ghost px-1 py-0 disabled:opacity-20"><ArrowUp size={12} /></button>
              <button onClick={() => moveDown(i)} disabled={i === routingRules.length - 1} className="btn-ghost px-1 py-0 disabled:opacity-20"><ArrowDown size={12} /></button>
            </div>
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>{r.name}</span>
                <span className="badge" style={{ color: 'var(--text-secondary)', background: 'var(--bg-tertiary)' }}>{r.outboundTag}</span>
                {r.enabled ? <span className="badge" style={{ color: 'var(--green)', background: 'rgba(34,197,94,0.15)' }}>{t('network.enabled')}</span> : <span className="badge" style={{ color: 'var(--red)', background: 'rgba(239,68,68,0.15)' }}>{t('network.disabled')}</span>}
              </div>
              <p className="text-xs mt-0.5" style={{ color: 'var(--text-muted)' }}>
                {(Array.isArray(r.domain) && r.domain.length > 0) && <span>域名: {r.domain.join(', ')} </span>}
                {(Array.isArray(r.ip) && r.ip.length > 0) && <span>IP: {r.ip.join(', ')} </span>}
                {r.port && <span>端口: {r.port} </span>}
                {r.network && <span>网络: {r.network}</span>}
              </p>
            </div>
            <button onClick={() => { setEditing({ ...r, domain: Array.isArray(r.domain) ? [...r.domain] : [], ip: Array.isArray(r.ip) ? [...r.ip] : [] }); setIsNew(false) }} className="btn-ghost"><Edit3 size={14} /></button>
            <button onClick={() => deleteRoutingRule(r.id)} className="btn-ghost" style={{ color: 'var(--red)' }}><Trash2 size={14} /></button>
          </div>
        ))
      )}
      {editing && (
        <Modal title={isNew ? t('network.addRouteTitle') : t('network.editRouteTitle')} onClose={() => setEditing(null)}
          footer={<><button onClick={() => setEditing(null)} className="btn-secondary">{t('servers.cancel')}</button><button onClick={save} className="btn-primary">{t('servers.save')}</button></>}>
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>{t('network.ruleName')}</label><input value={editing.name} onChange={(e) => setEditing({ ...editing, name: e.target.value })} className="input-field" /></div>
              <div className="custom-select"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>{t('network.outboundTag')}</label><select value={editing.outboundTag} onChange={(e) => setEditing({ ...editing, outboundTag: e.target.value })} className="input-field"><option value="proxy">proxy</option><option value="direct">direct</option><option value="block">block</option></select></div>
            </div>
            <div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>{t('network.domains')}</label><input value={(editing.domain || []).join(', ')} onChange={(e) => setDomains(e.target.value)} placeholder="geosite:cn, example.com" className="input-field" /></div>
            <div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>{t('network.ips')}</label><input value={(editing.ip || []).join(', ')} onChange={(e) => setIPs(e.target.value)} placeholder="geoip:cn, 1.2.3.4" className="input-field" /></div>
            <div className="grid grid-cols-2 gap-3">
              <div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>{t('network.rulePort')}</label><input value={editing.port} onChange={(e) => setEditing({ ...editing, port: e.target.value })} placeholder="443" className="input-field" /></div>
              <div className="custom-select"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>{t('network.ruleNetwork')}</label><select value={editing.network} onChange={(e) => setEditing({ ...editing, network: e.target.value })} className="input-field"><option value="">-</option><option value="tcp">TCP</option><option value="udp">UDP</option></select></div>
            </div>
          </div>
        </Modal>
      )}
    </div>
  )
}

function DNSPanel() {
  const dnsConfig = useAppStore((s) => s.dnsConfig)
  const setDNSConfig = useAppStore((s) => s.setDNSConfig)
  const addDNSServer = useAppStore((s) => s.addDNSServer)
  const removeDNSServer = useAppStore((s) => s.removeDNSServer)
  const { t } = useT()
  const [newAddr, setNewAddr] = useState('')
  const [newPort, setNewPort] = useState(53)

  const handleAdd = async () => { if (!newAddr.trim()) return; await addDNSServer({ address: newAddr.trim(), port: newPort, domains: [], expectIPs: [], skipFallback: false }); setNewAddr(''); setNewPort(53) }
  const update = (partial: Partial<DNSConfig>) => { if (!dnsConfig) return; setDNSConfig({ ...dnsConfig, ...partial }) }
  if (!dnsConfig) return <div className="card text-center py-8"><p style={{ color: 'var(--text-secondary)' }}>Loading...</p></div>

  return (
    <div className="space-y-4">
      <div className="card space-y-3">
        <div className="flex items-center gap-2 pb-2 border-b" style={{ borderColor: 'var(--border)' }}><Globe size={16} style={{ color: 'var(--accent)' }} /><h3 className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>{t('network.dnsServers')}</h3></div>
        <div className="space-y-2">
          {(dnsConfig.servers || []).map((s, i) => (
            <div key={i} className="flex items-center gap-3 rounded-lg px-3 py-2" style={{ background: 'var(--bg-secondary)' }}>
              <span className="text-xs w-6" style={{ color: 'var(--text-muted)' }}>#{i + 1}</span>
              <span className="text-sm font-mono flex-1" style={{ color: 'var(--text-primary)' }}>{s.address}{s.port > 0 ? `:${s.port}` : ''}</span>
              <button onClick={() => removeDNSServer(i)} className="btn-ghost" style={{ color: 'var(--red)' }}><Trash2 size={14} /></button>
            </div>
          ))}
        </div>
        <div className="flex gap-2">
          <input value={newAddr} onChange={(e) => setNewAddr(e.target.value)} placeholder={t('network.dnsAddr')} className="input-field flex-1" onKeyDown={(e) => e.key === 'Enter' && handleAdd()} />
          <input type="number" value={newPort} onChange={(e) => setNewPort(parseInt(e.target.value) || 53)} className="input-field w-24" />
          <button onClick={handleAdd} className="btn-primary"><Plus size={16} /></button>
        </div>
      </div>
      <div className="card space-y-3">
        <h3 className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>{t('network.dnsOptions')}</h3>
        <div className="grid grid-cols-2 gap-4">
          <div className="custom-select"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>{t('network.queryStrategy')}</label><select value={dnsConfig.queryStrategy} onChange={(e) => update({ queryStrategy: e.target.value })} className="input-field"><option value="UseIP">UseIP</option><option value="UseIPv4">UseIPv4</option><option value="UseIPv6">UseIPv6</option></select></div>
          <div className="flex flex-col justify-end gap-3">
            <label className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={dnsConfig.disableCache} onChange={(e) => update({ disableCache: e.target.checked })} className="w-4 h-4" style={{ accentColor: 'var(--accent)' }} /><span className="text-sm" style={{ color: 'var(--text-secondary)' }}>{t('network.disableCache')}</span></label>
            <label className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={dnsConfig.disableFallback} onChange={(e) => update({ disableFallback: e.target.checked })} className="w-4 h-4" style={{ accentColor: 'var(--accent)' }} /><span className="text-sm" style={{ color: 'var(--text-secondary)' }}>{t('network.disableFallback')}</span></label>
          </div>
        </div>
        <div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>{t('network.clientIp')}</label><input value={dnsConfig.clientIp || ''} onChange={(e) => update({ clientIp: e.target.value })} placeholder={t('network.clientIpHint')} className="input-field" /></div>
      </div>
    </div>
  )
}

function TProxyPanel() {
  const tpConfig = useAppStore((s) => s.transparentProxy)
  const setTPConfig = useAppStore((s) => s.setTransparentProxyConfig)
  const { t } = useT()
  if (!tpConfig) return <div className="card text-center py-8"><p style={{ color: 'var(--text-secondary)' }}>Loading...</p></div>
  const update = (p: Partial<TransparentProxyConfig>) => setTPConfig({ ...tpConfig, ...p })

  return (
    <div className="card space-y-4">
      <div className="flex items-center gap-3 pb-3 border-b" style={{ borderColor: 'var(--border)' }}>
        <Radio size={20} style={{ color: 'var(--accent)' }} />
        <div><h3 className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>{t('network.tproxyTitle')}</h3><p className="text-xs" style={{ color: 'var(--text-muted)' }}>{t('network.tproxyDesc')}</p></div>
      </div>
      <label className="flex items-center gap-3 cursor-pointer">
        <input type="checkbox" checked={tpConfig.enabled} onChange={(e) => update({ enabled: e.target.checked })} className="w-5 h-5" style={{ accentColor: 'var(--accent)' }} />
        <div><span className="text-sm font-medium" style={{ color: 'var(--text-secondary)' }}>{tpConfig.enabled ? t('network.tproxyEnabled') : t('network.tproxyDisabled')}</span><p className="text-xs" style={{ color: 'var(--text-muted)' }}>{t('network.tproxyHint')}</p></div>
      </label>
      {tpConfig.enabled && (
        <div className="grid grid-cols-2 gap-4 tab-content">
          <div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>{t('network.tcpRedirect')}</label><input type="number" value={tpConfig.redirectTcp} onChange={(e) => update({ redirectTcp: parseInt(e.target.value) || 0 })} className="input-field" /></div>
          <div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>{t('network.udpRedirect')}</label><input type="number" value={tpConfig.redirectUdp} onChange={(e) => update({ redirectUdp: parseInt(e.target.value) || 0 })} className="input-field" /></div>
          <label className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={tpConfig.bypassLan} onChange={(e) => update({ bypassLan: e.target.checked })} className="w-4 h-4" style={{ accentColor: 'var(--accent)' }} /><span className="text-sm" style={{ color: 'var(--text-secondary)' }}>{t('network.bypassLan')}</span></label>
        </div>
      )}
    </div>
  )
}

function Modal({ title, onClose, footer, children }: { title: string; onClose: () => void; footer?: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 animate-fade-in" onClick={onClose}>
      <div className="rounded-xl w-[500px] max-h-[80vh] overflow-hidden shadow-2xl animate-fade-in-scale" style={{ background: 'var(--sidebar-bg)', border: '1px solid var(--border)' }} onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between p-4 border-b" style={{ borderColor: 'var(--border)' }}>
          <h3 className="font-semibold" style={{ color: 'var(--text-primary)' }}>{title}</h3>
          <button onClick={onClose} className="btn-ghost"><X size={16} /></button>
        </div>
        <div className="p-4 max-h-[55vh] overflow-y-auto">{children}</div>
        {footer && <div className="flex justify-end gap-2 p-4 border-t" style={{ borderColor: 'var(--border)' }}>{footer}</div>}
      </div>
    </div>
  )
}
