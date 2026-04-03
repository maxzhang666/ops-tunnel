import type { VersionInfo } from '@/types/api'
import { openExternal } from '@/lib/utils'
import { SettingRow, SettingSection } from './setting-row'

interface Props {
  version?: VersionInfo
  isLoading: boolean
}

export function AboutSection({ version, isLoading }: Props) {
  const isDev = version?.version === 'dev'
  const hasUpdate = version?.latest && !isDev && version.latest.version !== version.version

  return (
    <SettingSection title="About">
      <SettingRow label="Version">
        <span className="text-sm text-muted-foreground">{version?.version ?? '—'}</span>
      </SettingRow>

      <SettingRow label="GitHub">
        <button
          type="button"
          onClick={() => openExternal('https://github.com/maxzhang666/ops-tunnel')}
          className="text-sm text-primary hover:underline"
        >
          maxzhang666/ops-tunnel
        </button>
      </SettingRow>

      <div className="px-4 py-3">
        {isLoading ? (
          <div className="text-sm text-muted-foreground">Checking for updates...</div>
        ) : hasUpdate ? (
          <div className="flex items-center justify-between">
            <div>
              <div className="text-sm font-semibold">Update Available</div>
              <div className="text-xs text-muted-foreground">
                v{version!.latest!.version} — {new Date(version!.latest!.publishedAt).toLocaleDateString()}
              </div>
            </div>
            <button
              type="button"
              onClick={() => openExternal(version!.latest!.url)}
              className="rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90"
            >
              Download
            </button>
          </div>
        ) : isDev && version?.latest ? (
          <div className="text-sm text-muted-foreground">
            Development build — latest release: v{version.latest.version}
          </div>
        ) : (
          <div className="text-sm text-muted-foreground">
            {version?.latest ? 'Up to date' : 'Unable to check for updates'}
          </div>
        )}
      </div>
    </SettingSection>
  )
}
