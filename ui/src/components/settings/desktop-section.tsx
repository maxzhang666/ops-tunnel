import { useTranslation } from 'react-i18next'
import type { Settings } from '@/types/api'
import { SettingRow, SettingSection } from './setting-row'

interface Props {
  settings: Settings
  onUpdate: (patch: Record<string, unknown>) => void
}

export function DesktopSection({ settings, onUpdate }: Props) {
  const { t } = useTranslation()

  const CLOSE_ACTIONS = [
    { value: 'ask', label: t('settings.closeAsk') },
    { value: 'minimize', label: t('settings.closeMinimize') },
    { value: 'quit', label: t('settings.closeQuit') },
  ]

  return (
    <SettingSection title={t('settings.desktop')}>
      <SettingRow label={t('settings.closeAction')} description={t('settings.closeActionDesc')}>
        <select
          className="rounded-md border bg-background px-3 py-1.5 text-sm"
          value={settings.desktop.closeAction}
          onChange={(e) => onUpdate({ desktop: { closeAction: e.target.value } })}
        >
          {CLOSE_ACTIONS.map((a) => (
            <option key={a.value} value={a.value}>{a.label}</option>
          ))}
        </select>
      </SettingRow>
    </SettingSection>
  )
}
