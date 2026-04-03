import { TunnelForm } from './tunnel-form'
import { useUpdateTunnel } from '@/hooks/use-tunnels'
import { toast } from 'sonner'
import type { Tunnel } from '@/types/api'

interface DetailConfigProps {
  tunnel: Tunnel
}

export function DetailConfig({ tunnel }: DetailConfigProps) {
  const updateMutation = useUpdateTunnel()

  return (
    <TunnelForm
      initialData={tunnel}
      submitLabel="Save Changes"
      onSubmit={async (data) => {
        await updateMutation.mutateAsync({ id: tunnel.id, data })
        toast.success('Tunnel configuration updated')
      }}
    />
  )
}
