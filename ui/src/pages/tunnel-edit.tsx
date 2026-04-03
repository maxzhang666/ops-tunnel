import { useNavigate, useParams } from 'react-router'
import { TunnelForm } from '@/components/tunnel/tunnel-form'
import { useTunnel, useUpdateTunnel } from '@/hooks/use-tunnels'
import { toast } from 'sonner'

export default function TunnelEditPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: tunnel, isLoading } = useTunnel(id!)
  const updateMutation = useUpdateTunnel()

  if (isLoading) {
    return <div className="py-8 text-center text-muted-foreground">Loading...</div>
  }

  if (!tunnel) {
    return <div className="py-8 text-center text-muted-foreground">Tunnel not found</div>
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <h2 className="text-2xl font-bold">Edit: {tunnel.name}</h2>
      <TunnelForm
        initialData={tunnel}
        submitLabel="Save Changes"
        onSubmit={async (data) => {
          await updateMutation.mutateAsync({ id: id!, data })
          toast.success('Tunnel updated')
          navigate('/tunnels')
        }}
      />
    </div>
  )
}
