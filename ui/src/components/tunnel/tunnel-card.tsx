import { useNavigate } from 'react-router'
import { Loader2, Play, Square, RotateCw, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { useSSHConnections } from '@/hooks/use-ssh-connections'
import { useStartTunnel, useStopTunnel, useRestartTunnel } from '@/hooks/use-tunnels'
import { cn } from '@/lib/utils'
import type { Tunnel, TunnelStatus } from '@/types/api'

const statusColors: Record<string, string> = {
  running: 'bg-green-500',
  stopped: 'bg-gray-400',
  error: 'bg-red-500',
  degraded: 'bg-yellow-500',
  starting: 'bg-blue-500',
  stopping: 'bg-gray-400',
}

const modeStyles: Record<string, { bg: string; text: string; label: string }> = {
  local: { bg: 'bg-blue-50', text: 'text-blue-700', label: 'Local' },
  remote: { bg: 'bg-pink-50', text: 'text-pink-700', label: 'Remote' },
  dynamic: { bg: 'bg-amber-50', text: 'text-amber-700', label: 'Dynamic' },
}

function formatUptime(since: string): string {
  const ms = Date.now() - new Date(since).getTime()
  const mins = Math.floor(ms / 60000)
  if (mins < 60) return `${mins}m`
  const hours = Math.floor(mins / 60)
  return `${hours}h ${mins % 60}m`
}

interface TunnelCardProps {
  tunnel: Tunnel
  status?: TunnelStatus
  onDelete: (tunnel: Tunnel) => void
}

export function TunnelCard({ tunnel, status, onDelete }: TunnelCardProps) {
  const navigate = useNavigate()
  const { data: sshConns } = useSSHConnections()
  const startMutation = useStartTunnel()
  const stopMutation = useStopTunnel()
  const restartMutation = useRestartTunnel()

  const state = status?.state ?? 'stopped'
  const mode = modeStyles[tunnel.mode] ?? modeStyles.local
  const isTransitioning = state === 'starting' || state === 'stopping'
  const isBusy = startMutation.isPending || stopMutation.isPending || restartMutation.isPending

  const chainNames = tunnel.chain
    .map((id) => sshConns?.find((c) => c.id === id)?.name ?? id)
    .join(' → ')

  const handleControl = (e: React.MouseEvent) => {
    e.stopPropagation()
    if (state === 'stopped' || state === 'error') {
      startMutation.mutate(tunnel.id)
    } else if (state === 'running' || state === 'degraded') {
      stopMutation.mutate(tunnel.id)
    }
  }

  const handleRestart = (e: React.MouseEvent) => {
    e.stopPropagation()
    restartMutation.mutate(tunnel.id)
  }

  return (
    <div
      className={cn(
        'cursor-pointer rounded-lg border bg-card p-4 transition-colors hover:bg-accent/30',
        state === 'error' && 'border-red-300'
      )}
      onClick={() => navigate(`/tunnels/${tunnel.id}/edit`)}
    >
      <div className="flex items-start justify-between">
        <div className="min-w-0 flex-1">
          <div className="mb-1.5 flex items-center gap-2">
            <span className={cn('inline-block h-2 w-2 rounded-full', statusColors[state])} />
            <span className="truncate text-[15px] font-semibold">{tunnel.name}</span>
            <Badge variant="secondary" className={cn(mode.bg, mode.text, 'text-[11px]')}>
              {mode.label}
            </Badge>
          </div>
          <div className="mb-2 text-xs text-muted-foreground">{chainNames}</div>
          <div className="flex flex-wrap gap-2">
            {tunnel.mappings.map((m) => (
              <span key={m.id} className="rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                {tunnel.mode === 'dynamic'
                  ? `${m.listen.host}:${m.listen.port} (SOCKS5)`
                  : `${m.listen.host}:${m.listen.port} → ${m.connect?.host}:${m.connect?.port}`}
              </span>
            ))}
          </div>
          {status?.lastError && (state === 'error' || state === 'degraded') && (
            <div className="mt-2 text-xs text-destructive">{status.lastError}</div>
          )}
        </div>
        <div className="ml-4 flex flex-shrink-0 items-center gap-1.5">
          <span className="mr-2 text-xs text-muted-foreground">
            {state === 'running' && status?.since ? `Running · ${formatUptime(status.since)}` : ''}
            {state === 'stopped' ? 'Stopped' : ''}
            {state === 'error' ? 'Error' : ''}
            {state === 'degraded' ? 'Degraded' : ''}
            {state === 'starting' ? 'Starting...' : ''}
            {state === 'stopping' ? 'Stopping...' : ''}
          </span>
          {state === 'error' && (
            <Button variant="outline" size="icon" className="h-8 w-8" onClick={handleRestart} disabled={isBusy} title="Restart">
              <RotateCw className="h-3.5 w-3.5" />
            </Button>
          )}
          <Button
            variant="outline" size="icon" className="h-8 w-8"
            onClick={handleControl} disabled={isBusy || isTransitioning}
            title={state === 'stopped' || state === 'error' ? 'Start' : 'Stop'}
          >
            {isBusy || isTransitioning ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : state === 'stopped' || state === 'error' ? (
              <Play className="h-3.5 w-3.5" />
            ) : (
              <Square className="h-3.5 w-3.5" />
            )}
          </Button>
          <Button
            variant="ghost" size="icon"
            className="h-8 w-8 text-destructive hover:text-destructive"
            onClick={(e) => { e.stopPropagation(); onDelete(tunnel) }}
          >
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>
    </div>
  )
}
