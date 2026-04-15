import { Outlet } from 'react-router'
import { Sidebar } from '@/components/sidebar'
import { CloseDialog } from '@/components/close-dialog'
import { Toaster } from '@/components/ui/sonner'
import { ErrorBoundary } from '@/components/error-boundary'

const isDesktop = 'wails' in window

function AmbientGlow() {
  return (
    <div className="pointer-events-none absolute inset-0 overflow-hidden">
      <div className="absolute -left-[15%] -top-[20%] h-[55%] w-[55%] rounded-full bg-primary opacity-[0.06] blur-[100px] dark:opacity-[0.10]" />
      <div className="absolute -bottom-[20%] -right-[15%] h-[45%] w-[45%] rounded-full bg-secondary/30 opacity-[0.15] blur-[80px] dark:opacity-[0.20]" />
    </div>
  )
}

export function AppLayout() {
  if (isDesktop) {
    return (
      <div className="relative flex h-screen">
        <div className="cyber-bg-dots pointer-events-none absolute inset-0 z-0" />
        <Sidebar />
        <main className="relative z-10 flex-1 overflow-y-auto overscroll-none bg-gradient-to-br from-background via-muted/20 to-muted/40 p-8">
          <AmbientGlow />
          <div className="relative z-10">
            <ErrorBoundary>
              <Outlet />
            </ErrorBoundary>
          </div>
        </main>
        <Toaster />
        <CloseDialog />
      </div>
    )
  }

  return (
    <div className="relative flex h-screen items-center justify-center bg-muted/40">
      <div className="cyber-bg-dots pointer-events-none absolute inset-0 z-0" />
      <div className="relative z-10 flex h-full w-full max-h-[832px] max-w-[1060px] overflow-hidden rounded-xl border shadow-xl">
        <Sidebar />
        <main className="relative flex-1 overflow-auto rounded-r-xl bg-gradient-to-br from-background via-muted/20 to-muted/40 p-8">
          <AmbientGlow />
          <div className="relative z-10">
            <ErrorBoundary>
              <Outlet />
            </ErrorBoundary>
          </div>
        </main>
      </div>
      <Toaster />
    </div>
  )
}
