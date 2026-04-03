import { useState } from 'react'
import { Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { SSHList } from '@/components/ssh/ssh-list'
import { SSHForm } from '@/components/ssh/ssh-form'
import { useCreateSSHConnection } from '@/hooks/use-ssh-connections'
import { toast } from 'sonner'

export default function SSHConnectionsPage() {
  const [open, setOpen] = useState(false)
  const createMutation = useCreateSSHConnection()

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">SSH Connections</h2>
        <Button onClick={() => setOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          New Connection
        </Button>
      </div>
      <SSHList />

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="flex max-h-[85vh] flex-col overflow-hidden sm:max-w-3xl">
          <DialogHeader className="shrink-0">
            <DialogTitle>New SSH Connection</DialogTitle>
          </DialogHeader>
          <div className="flex-1 overflow-y-auto p-1">
            <SSHForm
              submitLabel="Create Connection"
              onSubmit={async (data) => {
                await createMutation.mutateAsync(data)
                toast.success('SSH connection created')
                setOpen(false)
              }}
            />
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
