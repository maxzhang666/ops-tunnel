import { useCallback, useState } from 'react'
import { useNavigate } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useQueryClient, useQueries } from '@tanstack/react-query'
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts'
import { useTunnels, TUNNEL_KEYS } from '@/hooks/use-tunnels'
import { useRealtimeTraffic } from '@/hooks/use-traffic'
import { useStats } from '@/hooks/use-stats'
import { useWsEvent } from '@/hooks/use-ws-events'
import { api } from '@/lib/api'
import { cn } from '@/lib/utils'
import type { Tunnel, TunnelStatus } from '@/types/api'

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

function formatRate(bytesPerSec: number): string {
  return `${formatBytes(bytesPerSec)}/s`
}

function formatUptime(since: string): string {
  const ms = Date.now() - new Date(since).getTime()
  const mins = Math.floor(ms / 60000)
  if (mins < 60) return `${mins}m`
  const hours = Math.floor(mins / 60)
  return `${hours}h ${mins % 60}m`
}

const statusDotColors: Record<string, string> = {
  running: 'bg-green-500',
  stopped: 'bg-gray-400',
  error: 'bg-red-500',
  degraded: 'bg-yellow-500',
  starting: 'bg-blue-500',
  stopping: 'bg-gray-400',
}

const modeStyles: Record<string, { bg: string; text: string }> = {
  local: { bg: 'bg-blue-50', text: 'text-blue-700' },
  remote: { bg: 'bg-pink-50', text: 'text-pink-700' },
  dynamic: { bg: 'bg-amber-50', text: 'text-amber-700' },
}

const chartRanges = [
  { key: '1m', seconds: 60 },
  { key: '5m', seconds: 300 },
  { key: '10m', seconds: 600 },
] as const

type ChartRange = (typeof chartRanges)[number]['key']

export default function DashboardPage() {
  const { t } = useTranslation()
  const { data: tunnels } = useTunnels()
  const { data: realtimeData } = useRealtimeTraffic()
  const { data: stats } = useStats()
  const queryClient = useQueryClient()
  const [chartRange, setChartRange] = useState<ChartRange>('5m')

  const statusQueries = useQueries({
    queries: (tunnels ?? []).map((tunnel) => ({
      queryKey: TUNNEL_KEYS.status(tunnel.id),
      queryFn: () => api.get<TunnelStatus>(`/tunnels/${tunnel.id}/status`),
      refetchInterval: 3000,
    })),
  })

  const statusMap = new Map<string, TunnelStatus>()
  statusQueries.forEach((q) => {
    if (q.data) statusMap.set(q.data.id, q.data)
  })

  useWsEvent(useCallback((event) => {
    if (event.type === 'tunnel.stateChanged') {
      queryClient.invalidateQueries({ queryKey: TUNNEL_KEYS.allStatuses })
    }
  }, [queryClient]))

  // Aggregate stats
  let running = 0, stopped = 0, errors = 0, totalIn = 0, totalOut = 0, activeConns = 0
  tunnels?.forEach((tunnel) => {
    const s = statusMap.get(tunnel.id)
    const state = s?.state ?? 'stopped'
    if (state === 'running') running++
    else if (state === 'error' || state === 'degraded') errors++
    else stopped++
    totalIn += s?.bytesIn ?? 0
    totalOut += s?.bytesOut ?? 0
    s?.mappings?.forEach((m) => { activeConns += m.activeConns ?? 0 })
  })

  // Latest realtime speed from samples
  const allSamples = realtimeData?.samples ?? []
  const latest = allSamples.length > 0 ? allSamples[allSamples.length - 1] : null
  const uploadSpeed = latest?.bytesOut ?? 0
  const downloadSpeed = latest?.bytesIn ?? 0

  // Filter samples by selected time range
  const rangeSeconds = chartRanges.find((r) => r.key === chartRange)?.seconds ?? 300
  const cutoff = Date.now() - rangeSeconds * 1000
  const visibleSamples = allSamples.filter((s) => new Date(s.ts).getTime() >= cutoff)

  const chartData = visibleSamples.map((s) => ({
    time: new Date(s.ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }),
    in: s.bytesIn,
    out: s.bytesOut,
  }))

  return (
    <div className="space-y-2">
      <h2 className="text-2xl font-bold">{t('dashboard.title')}</h2>

      {/* 3x3 stat grid */}
      <div className="grid grid-cols-3 gap-2">
        <StatCard label={t('dashboard.running')} value={running} color="text-green-600" />
        <StatCard label={t('dashboard.stopped')} value={stopped} color="text-muted-foreground" />
        <StatCard label={t('dashboard.errors')} value={errors} color="text-red-600" />
        <StatCard label={t('dashboard.uploadSpeed')} value={formatRate(uploadSpeed)} color="text-emerald-600" />
        <StatCard label={t('dashboard.downloadSpeed')} value={formatRate(downloadSpeed)} color="text-blue-600" />
        <StatCard label={t('dashboard.activeConns')} value={activeConns} />
        <StatCard label={t('dashboard.uploadTotal')} value={formatBytes(totalOut)} />
        <StatCard label={t('dashboard.downloadTotal')} value={formatBytes(totalIn)} />
        <StatCard label={t('dashboard.memoryUsage')} value={formatBytes(stats?.memAlloc ?? 0)} />
      </div>

      {/* Bandwidth chart */}
      <div className="rounded-lg border bg-card px-3 pt-3 pb-1">
        <div className="mb-2 flex items-center justify-between">
          <h3 className="text-sm font-medium">{t('dashboard.bandwidth')}</h3>
          <div className="flex gap-1">
            {chartRanges.map((r) => (
              <button
                key={r.key}
                onClick={() => setChartRange(r.key)}
                className={cn(
                  'rounded px-2 py-0.5 text-xs transition-colors',
                  chartRange === r.key
                    ? 'bg-primary text-primary-foreground'
                    : 'text-muted-foreground hover:bg-accent',
                )}
              >
                {t(`dashboard.range${r.key}`)}
              </button>
            ))}
          </div>
        </div>
        <div className="h-44">
          {chartData.length > 0 ? (
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData} margin={{ top: 4, right: 4, bottom: 0, left: 0 }}>
                <defs>
                  <linearGradient id="colorIn" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="colorOut" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#22c55e" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#22c55e" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <XAxis dataKey="time" tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                <YAxis tick={{ fontSize: 10 }} tickFormatter={(v: number) => formatRate(v)} width={70} />
                <Tooltip
                  formatter={(value, name) => [formatRate(Number(value ?? 0)), name === 'in' ? '↓ Download' : '↑ Upload']}
                  labelStyle={{ fontSize: 11 }}
                  contentStyle={{ fontSize: 11 }}
                />
                <Area type="monotone" dataKey="in" stroke="#3b82f6" fill="url(#colorIn)" strokeWidth={1.5} isAnimationActive={false} />
                <Area type="monotone" dataKey="out" stroke="#22c55e" fill="url(#colorOut)" strokeWidth={1.5} isAnimationActive={false} />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
              {t('dashboard.noTrafficData')}
            </div>
          )}
        </div>
      </div>

      {/* Tunnel grid */}
      <div>
        <h3 className="mb-2 text-sm font-medium">{t('dashboard.tunnels')}</h3>
        <div className="grid grid-cols-2 gap-2">
          {tunnels?.map((tunnel) => (
            <TunnelMiniCard key={tunnel.id} tunnel={tunnel} status={statusMap.get(tunnel.id)} />
          ))}
          {(!tunnels || tunnels.length === 0) && (
            <div className="col-span-2 py-8 text-center text-sm text-muted-foreground">
              {t('dashboard.noTunnels')}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function StatCard({ label, value, color }: { label: string; value: string | number; color?: string }) {
  return (
    <div className="rounded-lg border bg-card px-3 py-2">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className={cn('text-lg font-bold', color)}>{value}</div>
    </div>
  )
}

function TunnelMiniCard({ tunnel, status }: { tunnel: Tunnel; status?: TunnelStatus }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const state = status?.state ?? 'stopped'
  const mode = modeStyles[tunnel.mode] ?? modeStyles.local
  const activeConns = status?.mappings?.reduce((sum, m) => sum + (m.activeConns ?? 0), 0) ?? 0

  return (
    <div
      className="cursor-pointer rounded-lg border bg-card p-3 transition-colors hover:bg-accent/30"
      onClick={() => navigate(`/tunnels/${tunnel.id}`)}
    >
      <div className="mb-1.5 flex items-center gap-2">
        <span className={cn('inline-block h-2 w-2 rounded-full', statusDotColors[state])} />
        <span className="text-sm font-semibold">{tunnel.name}</span>
        <span className={cn('rounded px-1.5 py-0.5 text-[10px] font-medium', mode.bg, mode.text)}>
          {tunnel.mode.toUpperCase()}
        </span>
        {state === 'running' && status?.since && (
          <span className="ml-auto text-[11px] text-muted-foreground">{formatUptime(status.since)}</span>
        )}
      </div>
      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <span>
          {status && (status.bytesIn > 0 || status.bytesOut > 0)
            ? `↓ ${formatBytes(status.bytesIn)}  ↑ ${formatBytes(status.bytesOut)}`
            : '—'}
        </span>
        {activeConns > 0 && <span>{t('dashboard.conns', { count: activeConns })}</span>}
      </div>
    </div>
  )
}
