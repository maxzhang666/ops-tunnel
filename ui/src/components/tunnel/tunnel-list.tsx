import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import {
  Dialog, DialogContent, DialogDescription, DialogFooter,
  DialogHeader, DialogTitle,
} from '@/components/ui/dialog'
import { useTunnels, useTunnelStatus, useDeleteTunnel, useUpdateTunnel, TUNNEL_KEYS } from '@/hooks/use-tunnels'
import { useWsEvent } from '@/hooks/use-ws-events'
import { TunnelCard } from './tunnel-card'
import { TunnelForm } from './tunnel-form'
import { toast } from 'sonner'
import type { Tunnel } from '@/types/api'

function TunnelWithStatus({ tunnel, onEdit, onDelete }: { tunnel: Tunnel; onEdit: (t: Tunnel) => void; onDelete: (t: Tunnel) => void }) {
  const { data: status } = useTunnelStatus(tunnel.id)
  return <TunnelCard tunnel={tunnel} status={status} onEdit={onEdit} onDelete={onDelete} />
}

export function TunnelList() {
  const { t } = useTranslation()
  const { data: tunnels, isLoading } = useTunnels()
  const deleteMutation = useDeleteTunnel()
  const updateMutation = useUpdateTunnel()
  const queryClient = useQueryClient()
  const [editTarget, setEditTarget] = useState<Tunnel | null>(null)
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
          <TunnelWithStatus key={tunnel.id} tunnel={tunnel} onEdit={setEditTarget} onDelete={setDeleteTarget} />
        ))}
      </div>

      <Dialog open={!!editTarget} onOpenChange={() => setEditTarget(null)} dismissible={false}>
        <DialogContent className="flex max-h-[85vh] flex-col overflow-hidden sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>{t('tunnel.editTunnel')}</DialogTitle>
          </DialogHeader>
          {editTarget && (
            <TunnelForm
              initialData={editTarget}
              submitLabel={t('common.saveChanges')}
              onCancel={() => setEditTarget(null)}
              onSubmit={async (data) => {
                await updateMutation.mutateAsync({ id: editTarget.id, data })
                toast.success(t('tunnel.tunnelConfigUpdated'))
                setEditTarget(null)
              }}
            />
          )}
        </DialogContent>
      </Dialog>

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
