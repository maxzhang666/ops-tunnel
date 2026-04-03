import { useNavigate } from 'react-router'
import { ArrowLeft, Loader2, Play, Square, RotateCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
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

const stateText: Record<string, { label: string; color: string }> = {
  running: { label: 'Running', color: 'text-green-600' },
  stopped: { label: 'Stopped', color: 'text-muted-foreground' },
  error: { label: 'Error', color: 'text-destructive' },
  degraded: { label: 'Degraded', color: 'text-yellow-600' },
  starting: { label: 'Starting...', color: 'text-blue-600' },
  stopping: { label: 'Stopping...', color: 'text-muted-foreground' },
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

interface DetailHeaderProps {
  tunnel: Tunnel
  status?: TunnelStatus
}

export function DetailHeader({ tunnel, status }: DetailHeaderProps) {
  const navigate = useNavigate()
  const startMutation = useStartTunnel()
  const stopMutation = useStopTunnel()
  const restartMutation = useRestartTunnel()

  const state = status?.state ?? 'stopped'
  const mode = modeStyles[tunnel.mode] ?? modeStyles.local
  const st = stateText[state] ?? stateText.stopped
  const isTransitioning = state === 'starting' || state === 'stopping'
  const isBusy = startMutation.isPending || stopMutation.isPending || restartMutation.isPending

  return (
    <div className="mb-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => navigate('/tunnels')}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <span className={cn('inline-block h-2.5 w-2.5 rounded-full', statusColors[state])} />
          <h2 className="text-xl font-bold">{tunnel.name}</h2>
          <Badge variant="secondary" className={cn(mode.bg, mode.text)}>{mode.label}</Badge>
          <span className={cn('text-sm', st.color)}>
            {st.label}
            {state === 'running' && status?.since ? ` · ${formatUptime(status.since)}` : ''}
          </span>
        </div>
        <div className="flex gap-2">
          {(state === 'running' || state === 'degraded') && (
            <Button variant="outline" size="sm" onClick={() => stopMutation.mutate(tunnel.id)} disabled={isBusy}>
              {stopMutation.isPending ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <Square className="mr-1 h-3.5 w-3.5" />}
              Stop
            </Button>
          )}
          {(state === 'stopped' || state === 'error') && (
            <Button variant="outline" size="sm" onClick={() => startMutation.mutate(tunnel.id)} disabled={isBusy}>
              {startMutation.isPending ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <Play className="mr-1 h-3.5 w-3.5" />}
              Start
            </Button>
          )}
          <Button variant="outline" size="sm" onClick={() => restartMutation.mutate(tunnel.id)} disabled={isBusy || isTransitioning}>
            {restartMutation.isPending ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <RotateCw className="mr-1 h-3.5 w-3.5" />}
            Restart
          </Button>
        </div>
      </div>
      {status?.lastError && (state === 'error' || state === 'degraded') && (
        <div className="ml-11 mt-2 text-sm text-destructive">{status.lastError}</div>
      )}
    </div>
  )
}
