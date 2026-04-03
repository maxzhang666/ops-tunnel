import { useState } from 'react'
import { Pencil, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table'
import {
  Dialog, DialogContent, DialogDescription, DialogFooter,
  DialogHeader, DialogTitle,
} from '@/components/ui/dialog'
import { useSSHConnections, useDeleteSSHConnection, useUpdateSSHConnection } from '@/hooks/use-ssh-connections'
import { SSHTestButton } from './ssh-test-button'
import { SSHForm } from './ssh-form'
import { toast } from 'sonner'
import type { SSHConnection } from '@/types/api'

function AuthBadge({ type }: { type: string }) {
  switch (type) {
    case 'privateKey':
      return <Badge variant="secondary" className="bg-blue-50 text-blue-700">Key</Badge>
    case 'password':
      return <Badge variant="secondary" className="bg-amber-50 text-amber-700">Password</Badge>
    default:
      return <Badge variant="outline">None</Badge>
  }
}

export function SSHList() {
  const { data: connections, isLoading } = useSSHConnections()
  const deleteMutation = useDeleteSSHConnection()
  const updateMutation = useUpdateSSHConnection()
  const [deleteTarget, setDeleteTarget] = useState<SSHConnection | null>(null)
  const [editTarget, setEditTarget] = useState<SSHConnection | null>(null)

  const handleDelete = () => {
    if (!deleteTarget) return
    deleteMutation.mutate(deleteTarget.id, {
      onSuccess: () => {
        toast.success(`Deleted "${deleteTarget.name}"`)
        setDeleteTarget(null)
      },
      onError: (err) => {
        toast.error(`Delete failed: ${err.message}`)
      },
    })
  }

  if (isLoading) {
    return <div className="py-8 text-center text-muted-foreground">Loading...</div>
  }

  if (!connections?.length) {
    return (
      <div className="py-12 text-center text-muted-foreground">
        No SSH connections yet. Create one to get started.
      </div>
    )
  }

  return (
    <>
      <div className="rounded-lg border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Host</TableHead>
              <TableHead>Auth</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {connections.map((conn) => (
              <TableRow key={conn.id}>
                <TableCell className="font-medium">{conn.name}</TableCell>
                <TableCell>
                  <div className="text-sm">{conn.auth.username}@{conn.endpoint.host}</div>
                  <div className="text-xs text-muted-foreground">Port {conn.endpoint.port}</div>
                </TableCell>
                <TableCell>
                  <div><AuthBadge type={conn.auth.type} /></div>
                  {conn.auth.type === 'privateKey' && conn.auth.privateKey?.filePath && (
                    <div className="mt-0.5 max-w-[150px] truncate text-xs text-muted-foreground">
                      {conn.auth.privateKey.filePath.split('/').pop()}
                    </div>
                  )}
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex items-center justify-end gap-1">
                    <SSHTestButton id={conn.id} />
                    <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => setEditTarget(conn)}>
                      <Pencil className="h-3.5 w-3.5" />
                    </Button>
                    <Button variant="ghost" size="icon" className="h-8 w-8 text-destructive hover:text-destructive" onClick={() => setDeleteTarget(conn)}>
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      {/* Edit dialog */}
      <Dialog open={!!editTarget} onOpenChange={() => setEditTarget(null)}>
        <DialogContent className="flex max-h-[85vh] flex-col overflow-hidden sm:max-w-3xl">
          <DialogHeader className="shrink-0">
            <DialogTitle>Edit SSH Connection</DialogTitle>
          </DialogHeader>
          <div className="flex-1 overflow-y-auto p-1">
            {editTarget && (
              <SSHForm
                initialData={editTarget}
                submitLabel="Save Changes"
                onSubmit={async (data) => {
                  await updateMutation.mutateAsync({ id: editTarget.id, data })
                  toast.success('SSH connection updated')
                  setEditTarget(null)
                }}
              />
            )}
          </div>
        </DialogContent>
      </Dialog>

      {/* Delete dialog */}
      <Dialog open={!!deleteTarget} onOpenChange={() => setDeleteTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete SSH Connection</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deleteTarget?.name}&quot;? This action cannot be undone.
              If this connection is referenced by any tunnel, deletion will fail.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>Cancel</Button>
            <Button variant="destructive" onClick={handleDelete} disabled={deleteMutation.isPending}>
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
