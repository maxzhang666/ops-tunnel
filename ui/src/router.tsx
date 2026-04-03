import { createBrowserRouter, Navigate } from 'react-router'
import { AppLayout } from '@/layouts/app-layout'
import SSHConnectionsPage from '@/pages/ssh-connections'
import SSHConnectionNewPage from '@/pages/ssh-connection-new'
import SSHConnectionEditPage from '@/pages/ssh-connection-edit'
import TunnelsPage from '@/pages/tunnels'
import TunnelNewPage from '@/pages/tunnel-new'
import TunnelEditPage from '@/pages/tunnel-edit'
import TunnelDetailPage from '@/pages/tunnel-detail'

export const router = createBrowserRouter([
  {
    element: <AppLayout />,
    children: [
      { index: true, element: <Navigate to="/ssh" replace /> },
      { path: 'ssh', element: <SSHConnectionsPage /> },
      { path: 'ssh/new', element: <SSHConnectionNewPage /> },
      { path: 'ssh/:id', element: <SSHConnectionEditPage /> },
      { path: 'tunnels', element: <TunnelsPage /> },
      { path: 'tunnels/new', element: <TunnelNewPage /> },
      { path: 'tunnels/:id', element: <TunnelDetailPage /> },
      { path: 'tunnels/:id/edit', element: <TunnelEditPage /> },
    ],
  },
])
