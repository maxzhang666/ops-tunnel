import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { SSHList } from '@/components/ssh/ssh-list'
import { SSHForm } from '@/components/ssh/ssh-form'
import { useCreateSSHConnection } from '@/hooks/use-ssh-connections'
import { toast } from 'sonner'

export default function SSHConnectionsPage() {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const createMutation = useCreateSSHConnection()

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">{t('ssh.title')}</h2>
        <Button onClick={() => setOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          {t('ssh.newConnection')}
        </Button>
      </div>
      <SSHList />

      <Dialog open={open} onOpenChange={setOpen} dismissible={false}>
        <DialogContent className="flex max-h-[85vh] flex-col overflow-hidden sm:max-w-3xl">
          <DialogHeader className="shrink-0">
            <DialogTitle>{t('ssh.newConnectionTitle')}</DialogTitle>
          </DialogHeader>
          <SSHForm
            submitLabel={t('ssh.createConnection')}
            onCancel={() => setOpen(false)}
            onSubmit={async (data) => {
              await createMutation.mutateAsync(data)
              toast.success(t('ssh.connectionCreated'))
              setOpen(false)
            }}
          />
        </DialogContent>
      </Dialog>
    </div>
  )
}
