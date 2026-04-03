import { useNavigate } from 'react-router'
import { Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { SSHList } from '@/components/ssh/ssh-list'

export default function SSHConnectionsPage() {
  const navigate = useNavigate()

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">SSH Connections</h2>
        <Button onClick={() => navigate('/ssh/new')}>
          <Plus className="mr-2 h-4 w-4" />
          New Connection
        </Button>
      </div>
      <SSHList />
    </div>
  )
}
