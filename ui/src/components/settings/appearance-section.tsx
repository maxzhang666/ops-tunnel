import { useTranslation } from 'react-i18next'
import type { Settings } from '@/types/api'
import { SettingRow, SettingSection } from './setting-row'

interface Props {
  settings: Settings
  onUpdate: (patch: Record<string, unknown>) => void
}

export function AppearanceSection({ settings, onUpdate }: Props) {
  const { t } = useTranslation()

  const THEMES = [
    { value: 'light', label: t('settings.themeLight') },
    { value: 'dark', label: t('settings.themeDark') },
    { value: 'system', label: t('settings.themeSystem') },
  ]

  return (
    <SettingSection title={t('settings.appearance')}>
      <SettingRow label={t('settings.theme')}>
        <div className="flex gap-1 rounded-md border bg-muted p-0.5">
          {THEMES.map((theme) => (
            <button
              key={theme.value}
              onClick={() => onUpdate({ appearance: { theme: theme.value } })}
              className={`rounded px-3 py-1 text-xs font-medium transition-colors ${
                settings.appearance.theme === theme.value
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              {theme.label}
            </button>
          ))}
        </div>
      </SettingRow>
    </SettingSection>
  )
}
