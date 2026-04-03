import type { VersionInfo } from '@/types/api'
import { SettingRow, SettingSection } from './setting-row'

interface Props {
  version?: VersionInfo
  isLoading: boolean
}

export function AboutSection({ version, isLoading }: Props) {
  const hasUpdate = version?.latest && version.latest.version !== version.version && version.version !== 'dev'

  return (
    <SettingSection title="About">
      <SettingRow label="Version">
        <span className="text-sm text-muted-foreground">{version?.version ?? '—'}</span>
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
            <a
              href={version!.latest!.url}
              target="_blank"
              rel="noopener noreferrer"
              className="rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90"
            >
              Download
            </a>
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
