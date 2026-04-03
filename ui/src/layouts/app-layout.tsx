import { Outlet } from 'react-router'
import { Sidebar } from '@/components/sidebar'
import { CloseDialog } from '@/components/close-dialog'
import { Toaster } from '@/components/ui/sonner'
import { ErrorBoundary } from '@/components/error-boundary'

const isDesktop = 'wails' in window

export function AppLayout() {
  if (isDesktop) {
    return (
      <div className="flex h-screen">
        <Sidebar />
        <main className="flex-1 overflow-y-auto overscroll-none bg-gradient-to-br from-background via-muted/20 to-muted/40 p-8">
          <ErrorBoundary>
            <Outlet />
          </ErrorBoundary>
        </main>
        <Toaster />
        <CloseDialog />
      </div>
    )
  }

  return (
    <div className="flex h-screen items-center justify-center bg-muted/40">
      <div className="flex h-full w-full max-h-[832px] max-w-[1060px] rounded-xl border shadow-xl">
        <Sidebar />
        <main className="flex-1 overflow-auto rounded-r-xl bg-gradient-to-br from-background via-muted/20 to-muted/40 p-8">
          <ErrorBoundary>
            <Outlet />
          </ErrorBoundary>
        </main>
      </div>
      <Toaster />
    </div>
  )
}
