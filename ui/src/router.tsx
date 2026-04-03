import { lazy, Suspense } from 'react'
import { createBrowserRouter, Navigate } from 'react-router'
import { AppLayout } from '@/layouts/app-layout'

const SSHConnectionsPage = lazy(() => import('@/pages/ssh-connections'))
const SSHConnectionNewPage = lazy(() => import('@/pages/ssh-connection-new'))
const SSHConnectionEditPage = lazy(() => import('@/pages/ssh-connection-edit'))
const TunnelsPage = lazy(() => import('@/pages/tunnels'))
const TunnelNewPage = lazy(() => import('@/pages/tunnel-new'))
const TunnelEditPage = lazy(() => import('@/pages/tunnel-edit'))
const TunnelDetailPage = lazy(() => import('@/pages/tunnel-detail'))
const SettingsPage = lazy(() => import('@/pages/settings'))
const NotFoundPage = lazy(() => import('@/pages/not-found'))

function LazyPage({ children }: { children: React.ReactNode }) {
  return (
    <Suspense
      fallback={
        <div className="flex items-center justify-center py-20">
          <div className="text-muted-foreground">Loading...</div>
        </div>
      }
    >
      {children}
    </Suspense>
  )
}

export const router = createBrowserRouter([
  {
    element: <AppLayout />,
    children: [
      { index: true, element: <Navigate to="/ssh" replace /> },
      { path: 'ssh', element: <LazyPage><SSHConnectionsPage /></LazyPage> },
      { path: 'ssh/new', element: <LazyPage><SSHConnectionNewPage /></LazyPage> },
      { path: 'ssh/:id', element: <LazyPage><SSHConnectionEditPage /></LazyPage> },
      { path: 'tunnels', element: <LazyPage><TunnelsPage /></LazyPage> },
      { path: 'tunnels/new', element: <LazyPage><TunnelNewPage /></LazyPage> },
      { path: 'tunnels/:id', element: <LazyPage><TunnelDetailPage /></LazyPage> },
      { path: 'tunnels/:id/edit', element: <LazyPage><TunnelEditPage /></LazyPage> },
      { path: 'settings', element: <LazyPage><SettingsPage /></LazyPage> },
      { path: '*', element: <LazyPage><NotFoundPage /></LazyPage> },
    ],
  },
])
