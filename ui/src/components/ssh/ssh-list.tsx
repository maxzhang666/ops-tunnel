import { useState } from 'react'
import { useNavigate } from 'react-router'
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
import { useSSHConnections, useDeleteSSHConnection } from '@/hooks/use-ssh-connections'
import { SSHTestButton } from './ssh-test-button'
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
  const navigate = useNavigate()
  const [deleteTarget, setDeleteTarget] = useState<SSHConnection | null>(null)

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
              <TableHead>Endpoint</TableHead>
              <TableHead>Auth</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {connections.map((conn) => (
              <TableRow key={conn.id}>
                <TableCell className="font-medium">{conn.name}</TableCell>
                <TableCell className="text-muted-foreground">
                  {conn.endpoint.host}:{conn.endpoint.port}
                </TableCell>
                <TableCell><AuthBadge type={conn.auth.type} /></TableCell>
                <TableCell className="text-right">
                  <div className="flex items-center justify-end gap-1">
                    <SSHTestButton id={conn.id} />
                    <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => navigate(`/ssh/${conn.id}`)}>
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
