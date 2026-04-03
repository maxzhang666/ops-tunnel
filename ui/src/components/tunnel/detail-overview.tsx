import { useSSHConnections } from '@/hooks/use-ssh-connections'
import { LogViewer } from './log-viewer'
import { cn } from '@/lib/utils'
import type { Tunnel, TunnelStatus } from '@/types/api'

const stateColors: Record<string, { label: string; color: string }> = {
  running: { label: 'Running', color: 'text-green-600' },
  stopped: { label: 'Stopped', color: 'text-muted-foreground' },
  error: { label: 'Error', color: 'text-destructive' },
  degraded: { label: 'Degraded', color: 'text-yellow-600' },
  starting: { label: 'Starting...', color: 'text-blue-600' },
  stopping: { label: 'Stopping...', color: 'text-muted-foreground' },
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
  const { data: sshConns } = useSSHConnections()
  const state = status?.state ?? 'stopped'
  const st = stateColors[state] ?? stateColors.stopped
  const activeMappings = status?.mappings?.filter((m) => m.state === 'listening').length ?? 0

  return (
    <div className="space-y-4">
      <div className="rounded-lg border bg-card p-4">
        <h4 className="mb-3 text-sm font-medium">SSH Chain</h4>
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
                    {'\u25CF'} {hopStatus?.state ?? 'disconnected'}
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      </div>

      <div className="grid grid-cols-3 gap-3">
        <div className="rounded-lg border bg-card p-3">
          <div className="text-xs text-muted-foreground">State</div>
          <div className={cn('text-base font-semibold', st.color)}>{st.label}</div>
        </div>
        <div className="rounded-lg border bg-card p-3">
          <div className="text-xs text-muted-foreground">Mappings</div>
          <div className="text-base font-semibold">
            {state === 'running' ? `${activeMappings} active` : `${tunnel.mappings.length} configured`}
          </div>
        </div>
        <div className="rounded-lg border bg-card p-3">
          <div className="text-xs text-muted-foreground">Uptime</div>
          <div className="text-base font-semibold">
            {state === 'running' ? formatUptime(status?.since) : '\u2014'}
          </div>
        </div>
      </div>

      <LogViewer tunnelId={tunnel.id} />
    </div>
  )
}
