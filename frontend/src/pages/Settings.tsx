import { useState, useEffect } from 'react'
import { useAppStore } from '../stores/appStore'
import { useSettingsStore, useT, themeNames, type Theme } from '../stores/settingsStore'
import { langNames, langFlags, type Lang } from '../i18n'
import type { ProxySettings } from '../types'
import { Shield, Server, Info, Palette, Languages } from 'lucide-react'

export default function Settings() {
  const proxySettings = useAppStore((s) => s.proxySettings)
  const setProxySettings = useAppStore((s) => s.setProxySettings)
  const kernelStatus = useAppStore((s) => s.kernelStatus)
  const appVersion = useAppStore((s) => s.appVersion)
  const theme = useSettingsStore((s) => s.theme)
  const setTheme = useSettingsStore((s) => s.setTheme)
  const lang = useSettingsStore((s) => s.lang)
  const setLang = useSettingsStore((s) => s.setLang)
  const { t } = useT()

  const [local, setLocal] = useState<ProxySettings>(proxySettings)
  const [saved, setSaved] = useState(false)

  useEffect(() => { setLocal(proxySettings) }, [proxySettings])

  const handleSave = async () => { await setProxySettings(local); setSaved(true); setTimeout(() => setSaved(false), 2000) }
  const update = (partial: Partial<ProxySettings>) => setLocal({ ...local, ...partial })

  return (
    <div className="p-6 space-y-6">
      <div className="page-section">
        <h2 className="text-xl font-semibold" style={{ color: 'var(--text-primary)' }}>{t('settings.title')}</h2>
        <p className="text-sm mt-0.5" style={{ color: 'var(--text-secondary)' }}>{t('settings.subtitle')}</p>
      </div>

      <div className="card space-y-4 page-section" style={{ animationDelay: '0.05s' }}>
        <div className="flex items-center gap-3 pb-3 border-b" style={{ borderColor: 'var(--border)' }}>
          <Palette size={20} style={{ color: 'var(--accent)' }} />
          <div><h3 className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>主题 / 语言</h3><p className="text-xs" style={{ color: 'var(--text-muted)' }}>自定义外观和语言</p></div>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div className="custom-select">
            <label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>主题</label>
            <select value={theme} onChange={(e) => setTheme(e.target.value as Theme)} className="input-field">
              {Object.entries(themeNames).map(([k, v]) => (
                <option key={k} value={k}>{v}</option>
              ))}
            </select>
          </div>
          <div className="custom-select">
            <label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>语言</label>
            <select value={lang} onChange={(e) => setLang(e.target.value as Lang)} className="input-field">
              {Object.entries(langNames).map(([k, v]) => (
                <option key={k} value={k}>{langFlags[k as Lang]} {v}</option>
              ))}
            </select>
          </div>
        </div>
      </div>

      <div className="card space-y-4 page-section" style={{ animationDelay: '0.1s' }}>
        <div className="flex items-center gap-3 pb-3 border-b" style={{ borderColor: 'var(--border)' }}>
          <Shield size={20} style={{ color: 'var(--accent)' }} />
          <div><h3 className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>{t('settings.proxy')}</h3><p className="text-xs" style={{ color: 'var(--text-muted)' }}>{t('settings.proxyDesc')}</p></div>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div className="custom-select"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>{t('settings.proxyMode')}</label><select value={local.mode} onChange={(e) => update({ mode: e.target.value as ProxySettings['mode'] })} className="input-field"><option value="global">{t('settings.global')}</option><option value="pac">{t('settings.pac')}</option><option value="none">{t('settings.none')}</option></select></div>
        </div>
        <div className="grid grid-cols-3 gap-4">
          <div className="col-span-2"><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>{t('settings.proxyAddr')}</label><input value={local.proxyHost} onChange={(e) => update({ proxyHost: e.target.value })} className="input-field" /></div>
          <div><label className="text-xs mb-1 block" style={{ color: 'var(--text-secondary)' }}>{t('settings.proxyPort')}</label><input type="number" value={local.proxyPort} onChange={(e) => update({ proxyPort: parseInt(e.target.value) || 10808 })} className="input-field" /></div>
        </div>
        <label className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={local.bypassLocal} onChange={(e) => update({ bypassLocal: e.target.checked })} className="w-4 h-4" style={{ accentColor: 'var(--accent)' }} /><span className="text-sm" style={{ color: 'var(--text-secondary)' }}>{t('settings.bypassLocal')}</span></label>
        <div className="pt-3 border-t flex items-center gap-3" style={{ borderColor: 'var(--border)' }}>
          <button onClick={handleSave} className="btn-primary">{saved ? t('settings.saved') : t('settings.save')}</button>
          <p className="text-xs" style={{ color: 'var(--text-muted)' }}>{t('settings.saveHint')}</p>
        </div>
      </div>

      <div className="card space-y-4 page-section" style={{ animationDelay: '0.15s' }}>
        <div className="flex items-center gap-3 pb-3 border-b" style={{ borderColor: 'var(--border)' }}>
          <Server size={20} style={{ color: 'var(--accent)' }} />
          <div><h3 className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>{t('settings.kernelInfo')}</h3><p className="text-xs" style={{ color: 'var(--text-muted)' }}>{t('settings.kernelDesc')}</p></div>
        </div>
        <div className="space-y-2">
          <div className="flex justify-between text-sm"><span style={{ color: 'var(--text-secondary)' }}>{t('settings.installStatus')}</span>
            {kernelStatus?.installed !== false
              ? <span style={{ color: 'var(--green)' }} className="flex items-center gap-1"><span className="w-1.5 h-1.5 rounded-full" style={{ background: 'var(--green)' }} />{t('settings.installed')}</span>
              : <span style={{ color: 'var(--red)' }} className="flex items-center gap-1"><span className="w-1.5 h-1.5 rounded-full" style={{ background: 'var(--red)' }} />{t('settings.notInstalled')}</span>}
          </div>
          {kernelStatus?.path && <div className="flex justify-between text-sm"><span style={{ color: 'var(--text-secondary)' }}>{t('settings.kernelPath')}</span><span className="font-mono text-xs truncate max-w-[300px]" style={{ color: 'var(--text-secondary)' }}>{kernelStatus.path}</span></div>}
        </div>
      </div>

      <div className="card space-y-4 page-section" style={{ animationDelay: '0.2s' }}>
        <div className="flex items-center gap-3 pb-3 border-b" style={{ borderColor: 'var(--border)' }}>
          <Info size={20} style={{ color: 'var(--accent)' }} />
          <div><h3 className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>{t('settings.sysInfo')}</h3><p className="text-xs" style={{ color: 'var(--text-muted)' }}>{t('settings.sysDesc')}</p></div>
        </div>
        <div className="space-y-2">
          {[
            [t('settings.appVersion'), appVersion],
            [t('settings.coreProtocol'), 'DCCP (RFC 4340)'],
            [t('settings.transportSupport'), 'TCP / WS / gRPC / QUIC / DCCP'],
            [t('settings.proxyProtocols'), 'VMess / VLESS / Trojan / Shadowsocks / SOCKS / HTTP'],
            [t('settings.stealthTech'), 'XTLS Vision / Reality / Random Padding'],
            [t('settings.compatible'), 'v2ray-core / Xray-core / sing-box'],
          ].map(([label, value]) => (
            <div key={label} className="flex justify-between text-sm">
              <span style={{ color: 'var(--text-secondary)' }}>{label}</span>
              <span style={{ color: 'var(--text-primary)' }}>{value}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
