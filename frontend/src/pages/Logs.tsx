import { useRef, useEffect, useState } from 'react'
import { useAppStore } from '../stores/appStore'
import { ScrollText, Trash2, Copy, Filter, FilterX, Check } from 'lucide-react'

const categories = [
  { id: 'all', label: '全部', color: 'text-slate-400' },
  { id: 'info', label: '信息', color: 'text-green-400' },
  { id: 'warn', label: '警告', color: 'text-yellow-400' },
  { id: 'error', label: '错误', color: 'text-red-400' },
  { id: 'debug', label: '调试', color: 'text-slate-500' },
]

export default function Logs() {
  const logs = useAppStore((s) => s.logs)
  const clearLogs = useAppStore((s) => s.clearLogs)
  const copyAllLogs = useAppStore((s) => s.copyAllLogs)
  const scrollRef = useRef<HTMLDivElement>(null)
  const [filter, setFilter] = useState('all')
  const [copiedLine, setCopiedLine] = useState(-1)

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [logs])

  const getLogCategory = (msg: string): string => {
    if (msg.includes('[错误]') || msg.includes('[Error]')) return 'error'
    if (msg.includes('[警告]') || msg.includes('[Warn]')) return 'warn'
    if (msg.includes('[信息]') || msg.includes('[Info]') || msg.includes('started') || msg.includes('ready')) return 'info'
    if (msg.includes('[调试]') || msg.includes('[Debug]')) return 'debug'
    return 'all'
  }

  const getLogColor = (msg: string) => {
    const cat = getLogCategory(msg)
    return { error: 'text-red-400', warn: 'text-yellow-400', info: 'text-green-400', debug: 'text-slate-500', all: 'text-slate-300' }[cat]
  }

  const getLogBg = (msg: string) => {
    const cat = getLogCategory(msg)
    return { error: 'bg-red-500/5', warn: 'bg-yellow-500/5', info: 'bg-green-500/5', debug: 'bg-transparent', all: 'bg-transparent' }[cat]
  }

  const filteredLogs = filter === 'all'
    ? logs
    : logs.filter((msg) => getLogCategory(msg) === filter)

  const copyLine = async (msg: string, idx: number) => {
    await navigator.clipboard.writeText(msg.trim())
    setCopiedLine(idx)
    setTimeout(() => setCopiedLine(-1), 1500)
  }

  const handleCopyAll = async () => {
    await copyAllLogs()
    setCopiedLine(-2)
    setTimeout(() => setCopiedLine(-1), 1500)
  }

  return (
    <div className="p-6 space-y-4 h-full flex flex-col">
      <div className="flex items-center justify-between shrink-0 animate-fade-in-up">
        <div>
          <h2 className="text-xl font-semibold text-white">日志</h2>
          <p className="text-slate-400 text-sm mt-0.5">{filteredLogs.length} / {logs.length} 条日志</p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={handleCopyAll} className="btn-secondary flex items-center gap-2">
            {copiedLine === -2 ? <Check size={16} className="text-green-400" /> : <Copy size={16} />}
            {copiedLine === -2 ? '已复制' : '复制全部'}
          </button>
          <button onClick={clearLogs} className="btn-secondary flex items-center gap-2">
            <Trash2 size={16} /> 清空
          </button>
        </div>
      </div>

      <div className="flex gap-1.5 shrink-0 animate-fade-in-up">
        {categories.map(({ id, label, color }) => (
          <button key={id} onClick={() => setFilter(id)}
            className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${filter === id ? 'bg-surface-700 text-white border border-surface-600' : `text-slate-500 hover:text-white`}`}>
            <span className={filter === id ? color : ''}>{label}</span>
            {id === 'all' && <span className="ml-1 text-slate-600">({logs.length})</span>}
            {id === 'info' && <span className="ml-1 text-slate-600">({logs.filter(m => getLogCategory(m)==='info').length})</span>}
            {id === 'error' && <span className="ml-1 text-slate-600">({logs.filter(m => getLogCategory(m)==='error').length})</span>}
          </button>
        ))}
      </div>

      <div ref={scrollRef} className="flex-1 card overflow-y-auto font-mono text-xs leading-relaxed" style={{ minHeight: 0 }}>
        {filteredLogs.length === 0 ? (
          <div className="flex items-center justify-center h-full text-slate-600">
            <div className="text-center animate-fade-in-scale">
              <ScrollText size={32} className="mx-auto mb-2 opacity-50" />
              <p>暂无日志</p>
              <p className="text-xs mt-1 opacity-70">启动核心后将显示运行日志</p>
            </div>
          </div>
        ) : (
          filteredLogs.map((msg, i) => (
            <div
              key={i}
              onClick={() => copyLine(msg, i)}
              className={`group flex items-start gap-2 px-3 py-0.5 cursor-pointer transition-colors hover:bg-white/5 ${getLogBg(msg)} ${getLogColor(msg)}`}
              title="点击复制此行"
            >
              <span className="text-slate-600 shrink-0 w-7 text-right select-none">{i + 1}</span>
              <span className="flex-1 break-all">{msg.trim() || ' '}</span>
              <span className="opacity-0 group-hover:opacity-100 transition-opacity shrink-0">
                {copiedLine === i ? <Check size={12} className="text-green-400" /> : <Copy size={12} className="text-slate-600" />}
              </span>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
