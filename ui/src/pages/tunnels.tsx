import { useState } from 'react'
import { Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { TunnelList } from '@/components/tunnel/tunnel-list'
import { TunnelForm } from '@/components/tunnel/tunnel-form'
import { useCreateTunnel } from '@/hooks/use-tunnels'
import { toast } from 'sonner'

export default function TunnelsPage() {
  const [open, setOpen] = useState(false)
  const createMutation = useCreateTunnel()

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">Tunnels</h2>
        <Button onClick={() => setOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          New Tunnel
        </Button>
      </div>
      <TunnelList />

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="flex max-h-[85vh] flex-col overflow-hidden sm:max-w-3xl">
          <DialogHeader className="shrink-0">
            <DialogTitle>New Tunnel</DialogTitle>
          </DialogHeader>
          <div className="flex-1 overflow-y-auto p-1">
            <TunnelForm
              submitLabel="Create Tunnel"
              onSubmit={async (data) => {
                await createMutation.mutateAsync(data)
                toast.success('Tunnel created')
                setOpen(false)
              }}
            />
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
