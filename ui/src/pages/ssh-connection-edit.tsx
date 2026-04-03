import { useNavigate, useParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import { SSHForm } from '@/components/ssh/ssh-form'
import { useSSHConnection, useUpdateSSHConnection } from '@/hooks/use-ssh-connections'
import { toast } from 'sonner'

export default function SSHConnectionEditPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: conn, isLoading } = useSSHConnection(id!)
  const updateMutation = useUpdateSSHConnection()

  if (isLoading) {
    return <div className="py-8 text-center text-muted-foreground">{t('common.loading')}</div>
  }

  if (!conn) {
    return <div className="py-8 text-center text-muted-foreground">{t('ssh.connectionNotFound')}</div>
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <h2 className="text-2xl font-bold">{t('tunnel.editTitle', { name: conn.name })}</h2>
      <SSHForm
        initialData={conn}
        submitLabel={t('common.saveChanges')}
        onSubmit={async (data) => {
          await updateMutation.mutateAsync({ id: id!, data })
          toast.success(t('ssh.connectionUpdated'))
          navigate('/ssh')
        }}
      />
    </div>
  )
}
