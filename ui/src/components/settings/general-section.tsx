import type { Settings, VersionInfo } from '@/types/api'
import { SettingRow, SettingSection } from './setting-row'

interface Props {
  settings: Settings
  version?: VersionInfo
  onUpdate: (patch: Record<string, unknown>) => void
}

const LOG_LEVELS = [
  { value: 'debug', label: 'Debug' },
  { value: 'info', label: 'Info' },
  { value: 'warn', label: 'Warn' },
  { value: 'error', label: 'Error' },
]

export function GeneralSection({ settings, version, onUpdate }: Props) {
  return (
    <SettingSection title="General">
      <SettingRow label="Log Level">
        <select
          className="rounded-md border bg-background px-3 py-1.5 text-sm"
          value={settings.general.logLevel}
          onChange={(e) => onUpdate({ general: { logLevel: e.target.value } })}
        >
          {LOG_LEVELS.map((l) => (
            <option key={l.value} value={l.value}>{l.label}</option>
          ))}
        </select>
      </SettingRow>

      <SettingRow label="Language" description="Coming soon">
        <select
          className="rounded-md border bg-muted px-3 py-1.5 text-sm text-muted-foreground"
          value="en"
          disabled
        >
          <option value="en">English</option>
        </select>
      </SettingRow>

      {version?.mode === 'desktop' && (
        <SettingRow label="Auto Start" description="Launch on system startup">
          <button
            type="button"
            role="switch"
            aria-checked={settings.general.autoStart}
            onClick={() => onUpdate({ general: { autoStart: !settings.general.autoStart } })}
            className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full transition-colors ${
              settings.general.autoStart ? 'bg-primary' : 'bg-input'
            }`}
          >
            <span
              className={`pointer-events-none block h-4 w-4 rounded-full bg-background shadow-sm ring-0 transition-transform ${
                settings.general.autoStart ? 'translate-x-4' : 'translate-x-0.5'
              }`}
            />
          </button>
        </SettingRow>
      )}
    </SettingSection>
  )
}
