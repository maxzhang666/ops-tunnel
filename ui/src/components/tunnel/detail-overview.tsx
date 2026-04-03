import { useTranslation } from 'react-i18next'
import { useSSHConnections } from '@/hooks/use-ssh-connections'
import { LogViewer } from './log-viewer'
import { cn } from '@/lib/utils'
import type { Tunnel, TunnelStatus } from '@/types/api'

const stateColorMap: Record<string, string> = {
  running: 'text-green-600',
  stopped: 'text-muted-foreground',
  error: 'text-destructive',
  degraded: 'text-yellow-600',
  starting: 'text-blue-600',
  stopping: 'text-muted-foreground',
}

const stateKeys: Record<string, string> = {
  running: 'tunnel.stateRunning',
  stopped: 'tunnel.stateStopped',
  error: 'tunnel.stateError',
  degraded: 'tunnel.stateDegraded',
  starting: 'tunnel.stateStarting',
  stopping: 'tunnel.stateStopping',
}

function formatUptime(since?: string): string {
  if (!since) return '\u2014'
  const ms = Date.now() - new Date(since).getTime()
  const mins = Math.floor(ms / 60000)
  if (mins < 60) return `${mins}m`
  const hours = Math.floor(mins / 60)
  return `${hours}h ${mins % 60}m`
}

interface DetailOverviewProps {
  tunnel: Tunnel
  status?: TunnelStatus
}

export function DetailOverview({ tunnel, status }: DetailOverviewProps) {
  const { t } = useTranslation()
  const { data: sshConns } = useSSHConnections()
  const state = status?.state ?? 'stopped'
  const stColor = stateColorMap[state] ?? stateColorMap.stopped
  const activeMappings = status?.mappings?.filter((m) => m.state === 'listening').length ?? 0

  return (
    <div className="flex h-full flex-col gap-4">
      <div className="rounded-lg border bg-card p-4">
        <h4 className="mb-3 text-sm font-medium">{t('overview.sshChain')}</h4>
        <div className="flex flex-wrap items-center gap-2">
          {tunnel.chain.map((connId, i) => {
            const conn = sshConns?.find((c) => c.id === connId)
            const hopStatus = status?.chain?.[i]
            const connected = hopStatus?.state === 'connected'
            return (
              <div key={connId} className="flex items-center gap-2">
                {i > 0 && <span className="text-muted-foreground">{'\u2192'}</span>}
                <div className={cn(
                  'rounded-md border px-3 py-2 text-center',
                  connected ? 'border-green-200 bg-green-50' : 'border-gray-200 bg-gray-50'
                )}>
                  <div className="text-xs font-semibold">{conn?.name ?? connId}</div>
                  <div className="text-[10px] text-muted-foreground">
                    {conn ? `${conn.endpoint.host}:${conn.endpoint.port}` : ''}
                  </div>
                  <div className={cn('text-[10px]', connected ? 'text-green-600' : 'text-muted-foreground')}>
                    {'\u25CF'} {hopStatus?.state ?? t('tunnel.disconnected')}
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      </div>

      <div className="grid grid-cols-3 gap-3">
        <div className="rounded-lg border bg-card p-3">
          <div className="text-xs text-muted-foreground">{t('overview.state')}</div>
          <div className={cn('text-base font-semibold', stColor)}>{t(stateKeys[state] ?? 'tunnel.stateStopped')}</div>
        </div>
        <div className="rounded-lg border bg-card p-3">
          <div className="text-xs text-muted-foreground">{t('overview.mappings')}</div>
          <div className="text-base font-semibold">
            {state === 'running' ? t('mapping.active', { count: activeMappings }) : t('mapping.configured', { count: tunnel.mappings.length })}
          </div>
        </div>
        <div className="rounded-lg border bg-card p-3">
          <div className="text-xs text-muted-foreground">{t('overview.uptime')}</div>
          <div className="text-base font-semibold">
            {state === 'running' ? formatUptime(status?.since) : '\u2014'}
          </div>
        </div>
      </div>

      <div className="min-h-0 flex-1">
        <LogViewer tunnelId={tunnel.id} />
      </div>
    </div>
  )
}
