import { useNavigate } from 'react-router'
import { TunnelForm } from '@/components/tunnel/tunnel-form'
import { useCreateTunnel } from '@/hooks/use-tunnels'
import { toast } from 'sonner'

export default function TunnelNewPage() {
  const navigate = useNavigate()
  const createMutation = useCreateTunnel()

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <h2 className="text-2xl font-bold">New Tunnel</h2>
      <TunnelForm
        submitLabel="Create Tunnel"
        onSubmit={async (data) => {
          await createMutation.mutateAsync(data)
          toast.success('Tunnel created')
          navigate('/tunnels')
        }}
      />
    </div>
  )
}
