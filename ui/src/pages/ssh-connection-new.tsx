import { useNavigate } from 'react-router'
import { SSHForm } from '@/components/ssh/ssh-form'
import { useCreateSSHConnection } from '@/hooks/use-ssh-connections'
import { toast } from 'sonner'

export default function SSHConnectionNewPage() {
  const navigate = useNavigate()
  const createMutation = useCreateSSHConnection()

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <h2 className="text-2xl font-bold">New SSH Connection</h2>
      <SSHForm
        submitLabel="Create Connection"
        onSubmit={async (data) => {
          await createMutation.mutateAsync(data)
          toast.success('SSH connection created')
          navigate('/ssh')
        }}
      />
    </div>
  )
}
