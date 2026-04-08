import { lazy, Suspense } from 'react'
import { createBrowserRouter, Navigate, useLocation } from 'react-router'
import { AppLayout } from '@/layouts/app-layout'
import i18n from '@/lib/i18n'
import { useAuthQuery, AuthContext } from '@/hooks/use-auth'

const DashboardPage = lazy(() => import('@/pages/dashboard'))
const SSHConnectionsPage = lazy(() => import('@/pages/ssh-connections'))
const SSHConnectionNewPage = lazy(() => import('@/pages/ssh-connection-new'))
const SSHConnectionEditPage = lazy(() => import('@/pages/ssh-connection-edit'))
const TunnelsPage = lazy(() => import('@/pages/tunnels'))
const TunnelNewPage = lazy(() => import('@/pages/tunnel-new'))
const TunnelEditPage = lazy(() => import('@/pages/tunnel-edit'))
const TunnelDetailPage = lazy(() => import('@/pages/tunnel-detail'))
const SettingsPage = lazy(() => import('@/pages/settings'))
const NotFoundPage = lazy(() => import('@/pages/not-found'))
const LoginPage = lazy(() => import('@/pages/login'))

function AuthGuard({ children }: { children: React.ReactNode }) {
  const { data, isLoading } = useAuthQuery()
  const location = useLocation()

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="text-muted-foreground">{i18n.t('common.loading')}</div>
      </div>
    )
  }

  const required = data?.required ?? false
  const authenticated = data?.authenticated ?? false

  if (required && !authenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  return (
    <AuthContext.Provider value={{ required, authenticated, checking: isLoading }}>
      {children}
    </AuthContext.Provider>
  )
}

function LazyPage({ children }: { children: React.ReactNode }) {
  return (
    <Suspense
      fallback={
        <div className="flex items-center justify-center py-20">
          <div className="text-muted-foreground">{i18n.t('common.loading')}</div>
        </div>
      }
    >
      {children}
    </Suspense>
  )
}

export const router = createBrowserRouter([
  {
    path: 'login',
    element: <LazyPage><LoginPage /></LazyPage>,
  },
  {
    element: <AuthGuard><AppLayout /></AuthGuard>,
    children: [
      { index: true, element: <Navigate to="/dashboard" replace /> },
      { path: 'dashboard', element: <LazyPage><DashboardPage /></LazyPage> },
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
