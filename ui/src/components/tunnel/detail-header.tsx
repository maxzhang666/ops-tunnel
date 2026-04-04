import { useState } from 'react'
import { useNavigate } from 'react-router'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, Loader2, Pencil, Play, Square, RotateCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { TunnelForm } from './tunnel-form'
import { useStartTunnel, useStopTunnel, useRestartTunnel, useUpdateTunnel } from '@/hooks/use-tunnels'
import { toast } from 'sonner'
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

const stateColors: Record<string, string> = {
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

const modeStyles: Record<string, { bg: string; text: string }> = {
  local: { bg: 'bg-blue-50', text: 'text-blue-700' },
  remote: { bg: 'bg-pink-50', text: 'text-pink-700' },
  dynamic: { bg: 'bg-amber-50', text: 'text-amber-700' },
}

const modeLabels: Record<string, string> = {
  local: 'tunnel.modeLabelLocal',
  remote: 'tunnel.modeLabelRemote',
  dynamic: 'tunnel.modeLabelDynamic',
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
  const { t } = useTranslation()
  const navigate = useNavigate()
  const startMutation = useStartTunnel()
  const stopMutation = useStopTunnel()
  const restartMutation = useRestartTunnel()
  const updateMutation = useUpdateTunnel()
  const [editOpen, setEditOpen] = useState(false)

  const state = status?.state ?? 'stopped'
  const mode = modeStyles[tunnel.mode] ?? modeStyles.local
  const stColor = stateColors[state] ?? stateColors.stopped
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
          <Badge variant="secondary" className={cn(mode.bg, mode.text)}>{t(modeLabels[tunnel.mode] ?? 'tunnel.modeLabelLocal')}</Badge>
          <span className={cn('text-sm', stColor)}>
            {state === 'running' && status?.since
              ? t('tunnel.runningUptime', { uptime: formatUptime(status.since) })
              : t(stateKeys[state] ?? 'tunnel.stateStopped')}
          </span>
        </div>
        <div className="flex gap-2">
          {(state === 'running' || state === 'degraded') && (
            <Button variant="outline" size="sm" onClick={() => stopMutation.mutate(tunnel.id)} disabled={isBusy}>
              {stopMutation.isPending ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <Square className="mr-1 h-3.5 w-3.5" />}
              {t('common.stop')}
            </Button>
          )}
          {(state === 'stopped' || state === 'error') && (
            <Button variant="outline" size="sm" onClick={() => startMutation.mutate(tunnel.id)} disabled={isBusy}>
              {startMutation.isPending ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <Play className="mr-1 h-3.5 w-3.5" />}
              {t('common.start')}
            </Button>
          )}
          <Button variant="outline" size="sm" onClick={() => restartMutation.mutate(tunnel.id)} disabled={isBusy || isTransitioning}>
            {restartMutation.isPending ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <RotateCw className="mr-1 h-3.5 w-3.5" />}
            {t('common.restart')}
          </Button>
          <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
            <Pencil className="mr-1 h-3.5 w-3.5" />
            {t('common.edit')}
          </Button>
        </div>
      </div>
      {status?.lastError && (state === 'error' || state === 'degraded') && (
        <div className="ml-11 mt-2 text-sm text-destructive">{status.lastError}</div>
      )}

      <Dialog open={editOpen} onOpenChange={setEditOpen} dismissible={false}>
        <DialogContent className="flex max-h-[85vh] flex-col overflow-hidden sm:max-w-3xl">
          <DialogHeader className="shrink-0">
            <DialogTitle>{t('tunnel.editTunnel')}</DialogTitle>
          </DialogHeader>
          <TunnelForm
            initialData={tunnel}
            submitLabel={t('common.saveChanges')}
            onSubmit={async (data) => {
              await updateMutation.mutateAsync({ id: tunnel.id, data })
              toast.success(t('tunnel.tunnelConfigUpdated'))
              setEditOpen(false)
            }}
            onCancel={() => setEditOpen(false)}
          />
        </DialogContent>
      </Dialog>
    </div>
  )
}
