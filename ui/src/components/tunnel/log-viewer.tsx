import { useState, useCallback, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useWsEvent } from '@/hooks/use-ws-events'
import { cn } from '@/lib/utils'
import type { TunnelEvent } from '@/types/api'

const MAX_ENTRIES = 200

const levelColors: Record<string, string> = {
  info: 'text-blue-400',
  warn: 'text-yellow-400',
  error: 'text-red-400',
}

interface LogViewerProps {
  tunnelId: string
}

export function LogViewer({ tunnelId }: LogViewerProps) {
  const { t } = useTranslation()
  const [entries, setEntries] = useState<TunnelEvent[]>([])
  const [levelFilter, setLevelFilter] = useState<string>('all')
  const containerRef = useRef<HTMLDivElement>(null)
  const autoScrollRef = useRef(true)

  useWsEvent(useCallback((event: TunnelEvent) => {
    if (event.tunnelId !== tunnelId) return
    setEntries((prev) => {
      const next = [...prev, event]
      return next.length > MAX_ENTRIES ? next.slice(-MAX_ENTRIES) : next
    })
  }, [tunnelId]))

  useEffect(() => {
    if (autoScrollRef.current && containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight
    }
  }, [entries])

  const handleScroll = () => {
    const el = containerRef.current
    if (!el) return
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 30
    autoScrollRef.current = atBottom
  }

  const filtered = levelFilter === 'all'
    ? entries
    : entries.filter((e) => e.level === levelFilter)

  return (
    <div className="rounded-lg border bg-card overflow-hidden">
      <div className="flex items-center justify-between border-b px-3 py-2">
        <span className="text-sm font-medium">{t('log.recentLogs')}</span>
        <select
          className="rounded border border-input bg-background px-2 py-0.5 text-xs"
          value={levelFilter}
          onChange={(e) => setLevelFilter(e.target.value)}
        >
          <option value="all">{t('log.filterAll')}</option>
          <option value="info">{t('log.filterInfo')}</option>
          <option value="warn">{t('log.filterWarn')}</option>
          <option value="error">{t('log.filterError')}</option>
        </select>
      </div>
      <div
        ref={containerRef}
        onScroll={handleScroll}
        className="h-48 overflow-y-auto bg-[#0f0f0f] p-3 font-mono text-[11px] leading-relaxed text-gray-300"
      >
        {filtered.length === 0 && (
          <div className="text-gray-600">{t('log.noEntries')}</div>
        )}
        {filtered.map((entry, i) => {
          const ts = new Date(entry.ts).toLocaleTimeString()
          const level = entry.level ?? 'info'
          return (
            <div key={i} className="mb-0.5">
              <span className="text-gray-600">{ts}</span>{' '}
              <span className={cn(levelColors[level] ?? 'text-gray-400')}>{level}</span>{' '}
              <span>{entry.message}</span>
            </div>
          )
        })}
      </div>
    </div>
  )
}
