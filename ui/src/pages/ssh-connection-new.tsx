import { useNavigate } from 'react-router'
import { useTranslation } from 'react-i18next'
import { SSHForm } from '@/components/ssh/ssh-form'
import { useCreateSSHConnection } from '@/hooks/use-ssh-connections'
import { toast } from 'sonner'

export default function SSHConnectionNewPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const createMutation = useCreateSSHConnection()

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <h2 className="text-2xl font-bold">{t('ssh.newConnectionTitle')}</h2>
      <SSHForm
        submitLabel={t('ssh.createConnection')}
        onSubmit={async (data) => {
          await createMutation.mutateAsync(data)
          toast.success(t('ssh.connectionCreated'))
          navigate('/ssh')
        }}
      />
    </div>
  )
}
