import { useNavigate, useParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import { TunnelForm } from '@/components/tunnel/tunnel-form'
import { useTunnel, useUpdateTunnel } from '@/hooks/use-tunnels'
import { toast } from 'sonner'

export default function TunnelEditPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: tunnel, isLoading } = useTunnel(id!)
  const updateMutation = useUpdateTunnel()

  if (isLoading) {
    return <div className="py-8 text-center text-muted-foreground">{t('common.loading')}</div>
  }

  if (!tunnel) {
    return <div className="py-8 text-center text-muted-foreground">{t('tunnel.tunnelNotFound')}</div>
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <h2 className="text-2xl font-bold">{t('tunnel.editTitle', { name: tunnel.name })}</h2>
      <TunnelForm
        initialData={tunnel}
        submitLabel={t('common.saveChanges')}
        onSubmit={async (data) => {
          await updateMutation.mutateAsync({ id: id!, data })
          toast.success(t('tunnel.tunnelUpdated'))
          navigate('/tunnels')
        }}
      />
    </div>
  )
}
