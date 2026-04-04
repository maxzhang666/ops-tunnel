import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { TunnelList } from '@/components/tunnel/tunnel-list'
import { TunnelForm } from '@/components/tunnel/tunnel-form'
import { useCreateTunnel } from '@/hooks/use-tunnels'
import { toast } from 'sonner'

export default function TunnelsPage() {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const createMutation = useCreateTunnel()

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">{t('tunnel.title')}</h2>
        <Button onClick={() => setOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          {t('tunnel.newTunnel')}
        </Button>
      </div>
      <TunnelList />

      <Dialog open={open} onOpenChange={setOpen} dismissible={false}>
        <DialogContent className="flex max-h-[85vh] flex-col overflow-hidden sm:max-w-3xl">
          <DialogHeader className="shrink-0">
            <DialogTitle>{t('tunnel.newTunnel')}</DialogTitle>
          </DialogHeader>
          <TunnelForm
            submitLabel={t('tunnel.createTunnel')}
            onCancel={() => setOpen(false)}
            onSubmit={async (data) => {
              await createMutation.mutateAsync(data)
              toast.success(t('tunnel.tunnelCreated'))
              setOpen(false)
            }}
          />
        </DialogContent>
      </Dialog>
    </div>
  )
}
