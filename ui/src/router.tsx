import { createBrowserRouter, Navigate } from 'react-router'
import { AppLayout } from '@/layouts/app-layout'
import SSHConnectionsPage from '@/pages/ssh-connections'
import TunnelsPlaceholder from '@/pages/tunnels-placeholder'

export const router = createBrowserRouter([
  {
    element: <AppLayout />,
    children: [
      { index: true, element: <Navigate to="/ssh" replace /> },
      { path: 'ssh', element: <SSHConnectionsPage /> },
      { path: 'tunnels', element: <TunnelsPlaceholder /> },
    ],
  },
])
