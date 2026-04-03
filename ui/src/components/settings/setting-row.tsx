import { cn } from '@/lib/utils'

interface SettingRowProps {
  label: string
  description?: string
  children: React.ReactNode
  className?: string
}

export function SettingRow({ label, description, children, className }: SettingRowProps) {
  return (
    <div className={cn('flex items-center justify-between px-4 py-3', className)}>
      <div className="space-y-0.5">
        <div className="text-sm font-medium">{label}</div>
        {description && (
          <div className="text-xs text-muted-foreground">{description}</div>
        )}
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  )
}

interface SettingSectionProps {
  title: string
  children: React.ReactNode
}

export function SettingSection({ title, children }: SettingSectionProps) {
  return (
    <div className="space-y-2">
      <h3 className="px-1 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
        {title}
      </h3>
      <div className="divide-y rounded-lg border bg-card">
        {children}
      </div>
    </div>
  )
}
