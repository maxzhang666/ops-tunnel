import { NavLink } from 'react-router'
import { Cable, Link2, Settings } from 'lucide-react'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'

const navItems = [
  { to: '/ssh', label: 'SSH Connections', icon: Link2 },
  { to: '/tunnels', label: 'Tunnels', icon: Cable },
]

export function Sidebar() {
  return (
    <aside className="flex h-full w-56 shrink-0 flex-col rounded-l-xl border-r bg-background">
      <div className="px-4 py-5">
        <h1 className="text-lg font-bold">OpsTunnel</h1>
        <p className="text-xs text-muted-foreground">SSH Tunnel Manager</p>
      </div>
      <Separator />
      <nav className="flex-1 space-y-1 px-2 py-3">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                isActive
                  ? 'bg-accent text-accent-foreground'
                  : 'text-muted-foreground hover:bg-accent/50 hover:text-foreground'
              )
            }
          >
            <item.icon className="h-4 w-4" />
            {item.label}
          </NavLink>
        ))}
      </nav>
      <Separator />
      <div className="px-2 py-3">
        <NavLink
          to="/settings"
          className={({ isActive }) =>
            cn(
              'flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors',
              isActive
                ? 'bg-accent text-accent-foreground'
                : 'text-muted-foreground hover:bg-accent/50 hover:text-foreground'
            )
          }
        >
          <Settings className="h-4 w-4" />
          Settings
        </NavLink>
      </div>
    </aside>
  )
}
