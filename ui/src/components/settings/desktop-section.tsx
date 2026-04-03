import type { Settings } from '@/types/api'
import { SettingRow, SettingSection } from './setting-row'

interface Props {
  settings: Settings
  onUpdate: (patch: Record<string, unknown>) => void
}

const CLOSE_ACTIONS = [
  { value: 'ask', label: 'Ask' },
  { value: 'minimize', label: 'Minimize' },
  { value: 'quit', label: 'Quit' },
]

export function DesktopSection({ settings, onUpdate }: Props) {
  return (
    <SettingSection title="Desktop">
      <SettingRow label="Close Action" description="What happens when you close the window">
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
