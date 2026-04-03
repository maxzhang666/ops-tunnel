import { useNavigate } from 'react-router'
import { useTranslation } from 'react-i18next'
import { TunnelForm } from '@/components/tunnel/tunnel-form'
import { useCreateTunnel } from '@/hooks/use-tunnels'
import { toast } from 'sonner'

export default function TunnelNewPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const createMutation = useCreateTunnel()

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <h2 className="text-2xl font-bold">{t('tunnel.newTunnel')}</h2>
      <TunnelForm
        submitLabel={t('tunnel.createTunnel')}
        onSubmit={async (data) => {
          await createMutation.mutateAsync(data)
          toast.success(t('tunnel.tunnelCreated'))
          navigate('/tunnels')
        }}
      />
    </div>
  )
}
