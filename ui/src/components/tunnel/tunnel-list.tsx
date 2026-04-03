import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import {
  Dialog, DialogContent, DialogDescription, DialogFooter,
  DialogHeader, DialogTitle,
} from '@/components/ui/dialog'
import { useTunnels, useTunnelStatus, useDeleteTunnel, TUNNEL_KEYS } from '@/hooks/use-tunnels'
import { useWsEvent } from '@/hooks/use-ws-events'
import { TunnelCard } from './tunnel-card'
import { toast } from 'sonner'
import type { Tunnel } from '@/types/api'

function TunnelWithStatus({ tunnel, onDelete }: { tunnel: Tunnel; onDelete: (t: Tunnel) => void }) {
  const { data: status } = useTunnelStatus(tunnel.id)
  return <TunnelCard tunnel={tunnel} status={status} onDelete={onDelete} />
}

export function TunnelList() {
  const { t } = useTranslation()
  const { data: tunnels, isLoading } = useTunnels()
  const deleteMutation = useDeleteTunnel()
  const queryClient = useQueryClient()
  const [deleteTarget, setDeleteTarget] = useState<Tunnel | null>(null)

  useWsEvent(useCallback((event) => {
    if (event.type === 'tunnel.stateChanged') {
      queryClient.invalidateQueries({ queryKey: TUNNEL_KEYS.allStatuses })
    }
  }, [queryClient]))

  const handleDelete = () => {
    if (!deleteTarget) return
    deleteMutation.mutate(deleteTarget.id, {
      onSuccess: () => {
        toast.success(t('tunnel.deleted', { name: deleteTarget.name }))
        setDeleteTarget(null)
      },
      onError: (err) => {
        toast.error(t('tunnel.deleteFailed', { error: err.message }))
      },
    })
  }

  if (isLoading) {
    return <div className="py-8 text-center text-muted-foreground">{t('common.loading')}</div>
  }

  if (!tunnels?.length) {
    return (
      <div className="py-12 text-center text-muted-foreground">
        {t('tunnel.emptyState')}
      </div>
    )
  }

  return (
    <>
      <div className="space-y-3">
        {tunnels.map((tunnel) => (
          <TunnelWithStatus key={tunnel.id} tunnel={tunnel} onDelete={setDeleteTarget} />
        ))}
      </div>

      <Dialog open={!!deleteTarget} onOpenChange={() => setDeleteTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('tunnel.deleteTitle')}</DialogTitle>
            <DialogDescription>
              {t('tunnel.deleteConfirm', { name: deleteTarget?.name })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>{t('common.cancel')}</Button>
            <Button variant="destructive" onClick={handleDelete} disabled={deleteMutation.isPending}>
              {t('common.delete')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
