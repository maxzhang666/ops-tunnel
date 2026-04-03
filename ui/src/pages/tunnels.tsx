import { useNavigate } from 'react-router'
import { Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { TunnelList } from '@/components/tunnel/tunnel-list'

export default function TunnelsPage() {
  const navigate = useNavigate()

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">Tunnels</h2>
        <Button onClick={() => navigate('/tunnels/new')}>
          <Plus className="mr-2 h-4 w-4" />
          New Tunnel
        </Button>
      </div>
      <TunnelList />
    </div>
  )
}
