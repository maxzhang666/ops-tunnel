import { Outlet } from 'react-router'
import { Sidebar } from '@/components/sidebar'
import { Toaster } from '@/components/ui/sonner'

export function AppLayout() {
  return (
    <div className="flex h-screen">
      <Sidebar />
      <main className="flex-1 overflow-auto bg-muted/30 p-6">
        <Outlet />
      </main>
      <Toaster />
    </div>
  )
}
