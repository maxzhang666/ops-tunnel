import type { Settings } from '@/types/api'
import { SettingRow, SettingSection } from './setting-row'

interface Props {
  settings: Settings
  onUpdate: (patch: Record<string, unknown>) => void
}

const THEMES = [
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
  { value: 'system', label: 'System' },
]

export function AppearanceSection({ settings, onUpdate }: Props) {
  return (
    <SettingSection title="Appearance">
      <SettingRow label="Theme">
        <div className="flex gap-1 rounded-md border bg-muted p-0.5">
          {THEMES.map((t) => (
            <button
              key={t.value}
              onClick={() => onUpdate({ appearance: { theme: t.value } })}
              className={`rounded px-3 py-1 text-xs font-medium transition-colors ${
                settings.appearance.theme === t.value
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>
      </SettingRow>
    </SettingSection>
  )
}
