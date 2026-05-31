import { NavLink } from 'react-router-dom'
import { useAppStore } from '../stores/appStore'
import { useSettingsStore, useT } from '../stores/settingsStore'
import { LayoutDashboard, Server, Network, ScrollText } from 'lucide-react'
import { Settings } from 'lucide-react'

const navItems = [
  { to: '/', icon: LayoutDashboard, labelKey: 'sidebar.dashboard' },
  { to: '/servers', icon: Server, labelKey: 'sidebar.servers' },
  { to: '/network', icon: Network, labelKey: 'sidebar.network' },
  { to: '/logs', icon: ScrollText, labelKey: 'sidebar.logs' },
  { to: '/settings', icon: Settings, labelKey: 'sidebar.settings' },
]

export default function Sidebar() {
  const coreStatus = useAppStore((s) => s.coreStatus)
  const kernelStatus = useAppStore((s) => s.kernelStatus)
  const appVersion = useAppStore((s) => s.appVersion)
  const { t } = useT()

  const statusText: Record<string, string> = {
    running: t('status.running'),
    stopped: t('status.stopped'),
    starting: t('status.starting'),
    error: t('status.error'),
  }

  return (
    <aside className="w-56 flex flex-col shrink-0 border-r" style={{ background: 'var(--sidebar-bg)', borderColor: 'var(--border)' }}>
      <div className="p-4 border-b" style={{ borderColor: 'var(--border)' }}>
        <div className="flex items-center gap-2.5">
          <div className="w-9 h-9 rounded-xl flex items-center justify-center animate-pulse-glow" style={{ background: 'linear-gradient(135deg, var(--accent), var(--accent-hover))' }}>
            <span className="text-white font-bold text-base">O</span>
          </div>
          <div>
            <h1 className="font-semibold text-sm" style={{ color: 'var(--text-primary)' }}>Oiwest Core</h1>
            <p className="text-xs" style={{ color: 'var(--text-muted)' }}>v{appVersion}</p>
          </div>
        </div>
      </div>

      <nav className="flex-1 p-3 space-y-1">
        {navItems.map(({ to, icon: Icon, labelKey }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/'}
            className={({ isActive }) => `sidebar-item ${isActive ? 'active' : ''}`}
          >
            <Icon size={18} />
            <span>{t(labelKey)}</span>
          </NavLink>
        ))}
      </nav>

      <div className="p-3 border-t space-y-2" style={{ borderColor: 'var(--border)' }}>
        <div className="flex items-center gap-2">
          <div className={`status-dot ${coreStatus}`} />
          <span className="text-xs" style={{ color: 'var(--text-secondary)' }}>{statusText[coreStatus] || coreStatus}</span>
        </div>
        {kernelStatus?.installed === false && (
          <div className="flex items-center gap-2 text-xs" style={{ color: 'var(--red)' }}>
            <div className="status-dot error" />
            <span>{t('status.kernelNotInstalled')}</span>
          </div>
        )}
      </div>
    </aside>
  )
}
