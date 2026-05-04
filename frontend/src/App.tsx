import { useEffect } from 'react'
import { Routes, Route } from 'react-router-dom'
import { useAppStore } from './stores/appStore'
import { useSettingsStore } from './stores/settingsStore'
import Sidebar from './components/Sidebar'
import Dashboard from './pages/Dashboard'
import ServerList from './pages/ServerList'
import NetworkConfig from './pages/NetworkConfig'
import Settings from './pages/Settings'
import Logs from './pages/Logs'

export default function App() {
  const initApp = useAppStore((s) => s.initApp)
  const loading = useAppStore((s) => s.loading)
  const initSettings = useSettingsStore((s) => s.initSettings)

  useEffect(() => {
    initSettings()
    initApp()
    const interval = setInterval(() => {
      const store = useAppStore.getState()
      store.refreshCoreStatus()
      store.refreshTrafficStats()
    }, 2000)
    return () => clearInterval(interval)
  }, [])

  if (loading) {
    return (
      <div className="h-screen flex items-center justify-center" style={{ background: 'var(--bg-primary)' }}>
        <div className="text-center animate-fade-in-scale">
          <div className="w-14 h-14 rounded-xl flex items-center justify-center mx-auto mb-5 animate-pulse-glow" style={{ background: 'linear-gradient(135deg, var(--accent), var(--accent-hover))' }}>
            <span className="text-white font-bold text-xl">O</span>
          </div>
          <p style={{ color: 'var(--text-secondary)' }} className="text-sm">Oiwest Core 启动中...</p>
          <div className="mt-4 w-40 h-1 rounded-full mx-auto overflow-hidden" style={{ background: 'var(--bg-tertiary)' }}>
            <div className="h-full rounded-full animate-[progressBar_2s_ease-in-out_infinite]" style={{ background: 'var(--accent)' }} />
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="h-screen flex" style={{ background: 'var(--bg-primary)' }}>
      <Sidebar />
      <main className="flex-1 overflow-auto">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/servers" element={<ServerList />} />
          <Route path="/network" element={<NetworkConfig />} />
          <Route path="/logs" element={<Logs />} />
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </main>
    </div>
  )
}
