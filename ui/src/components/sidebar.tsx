import { useTranslation } from 'react-i18next'
import { NavLink } from 'react-router'
import { Cable, LayoutDashboard, Link2, LogOut, Settings } from 'lucide-react'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'
import { useVersion } from '@/hooks/use-settings'
import { useAuth, useLogout } from '@/hooks/use-auth'

const navItems = [
  { to: '/dashboard', labelKey: 'sidebar.dashboard', icon: LayoutDashboard },
  { to: '/ssh', labelKey: 'sidebar.sshConnections', icon: Link2 },
  { to: '/tunnels', labelKey: 'sidebar.tunnels', icon: Cable },
]

export function Sidebar() {
  const { t } = useTranslation()
  const { data: version } = useVersion()
  const auth = useAuth()
  const logout = useLogout()
  const hasUpdate = version?.latest && version.latest.version !== version.version && version.version !== 'dev'

  return (
    <aside className="flex h-full w-56 shrink-0 flex-col rounded-l-xl border-r bg-background">
      <div className="flex items-center gap-3 px-4 py-5">
        <img src="/favicon.svg" alt="OpsTunnel" className="h-8 w-8 shrink-0 rounded-lg" />
        <div className="min-w-0">
          <h1 className="text-lg font-bold leading-tight">{t('sidebar.title')}</h1>
          <p className="text-xs text-muted-foreground">{t('sidebar.subtitle')}</p>
        </div>
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
            {t(item.labelKey)}
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
          {t('sidebar.settings')}
          {hasUpdate && (
            <span className="ml-auto h-2 w-2 rounded-full bg-destructive" />
          )}
        </NavLink>
      </div>
      {auth.required && (
        <>
          <Separator />
          <div className="px-2 py-3">
            <button
              onClick={logout}
              className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent/50 hover:text-foreground"
            >
              <LogOut className="h-4 w-4" />
              {t('auth.logout')}
            </button>
          </div>
        </>
      )}
    </aside>
  )
}
