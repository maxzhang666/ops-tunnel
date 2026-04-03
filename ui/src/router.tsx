import { createBrowserRouter, Navigate } from 'react-router'
import { AppLayout } from '@/layouts/app-layout'
import SSHConnectionsPage from '@/pages/ssh-connections'
import SSHConnectionNewPage from '@/pages/ssh-connection-new'
import SSHConnectionEditPage from '@/pages/ssh-connection-edit'
import TunnelsPage from '@/pages/tunnels'

export const router = createBrowserRouter([
  {
    element: <AppLayout />,
    children: [
      { index: true, element: <Navigate to="/ssh" replace /> },
      { path: 'ssh', element: <SSHConnectionsPage /> },
      { path: 'ssh/new', element: <SSHConnectionNewPage /> },
      { path: 'ssh/:id', element: <SSHConnectionEditPage /> },
      { path: 'tunnels', element: <TunnelsPage /> },
    ],
  },
])
