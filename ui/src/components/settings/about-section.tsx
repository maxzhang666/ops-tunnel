import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Copy, Check, Download } from 'lucide-react'
import type { VersionInfo, ReleaseAsset } from '@/types/api'
import { openExternal } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { SettingRow, SettingSection } from './setting-row'

interface Props {
  version?: VersionInfo
  isLoading: boolean
}

function detectPlatformAsset(assets: ReleaseAsset[]): ReleaseAsset | undefined {
  const ua = navigator.userAgent.toLowerCase()
  const isMac = ua.includes('mac')
  const isWin = ua.includes('win')
  const isArm = ua.includes('arm') || ua.includes('aarch64')

  for (const a of assets) {
    const name = a.name.toLowerCase()
    if (isMac && isArm && name.includes('arm64') && name.endsWith('.dmg')) return a
    if (isMac && !isArm && !name.includes('arm64') && name.endsWith('.dmg')) return a
    if (isWin && name.endsWith('.zip')) return a
    if (!isMac && !isWin && name.endsWith('.tar.gz')) return a
  }
  return undefined
}

function CopyCommand({ command }: { command: string }) {
  const [copied, setCopied] = useState(false)
  const handleCopy = () => {
    navigator.clipboard.writeText(command)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }
  return (
    <div className="flex items-center gap-2 rounded-md bg-muted px-3 py-2 font-mono text-xs">
      <span className="flex-1 select-all">{command}</span>
      <button type="button" onClick={handleCopy} className="shrink-0 text-muted-foreground hover:text-foreground">
        {copied ? <Check className="h-3.5 w-3.5 text-green-500" /> : <Copy className="h-3.5 w-3.5" />}
      </button>
    </div>
  )
}

export function AboutSection({ version, isLoading }: Props) {
  const { t } = useTranslation()
  const isDev = version?.version?.startsWith('dev')
  const hasUpdate = version?.latest && !isDev && version.latest.version !== version.version
  const isDesktop = version?.mode === 'desktop'

  return (
    <SettingSection title={t('about.title')}>
      <SettingRow label={t('about.version')}>
        <span className="text-sm text-muted-foreground">{version?.version ?? '\u2014'}</span>
      </SettingRow>

      <SettingRow label={t('about.github')}>
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
          <div className="text-sm text-muted-foreground">{t('about.checkingUpdates')}</div>
        ) : hasUpdate ? (
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm font-semibold">{t('about.updateAvailable')}</div>
                <div className="text-xs text-muted-foreground">
                  v{version!.latest!.version} — {new Date(version!.latest!.publishedAt).toLocaleDateString()}
                </div>
              </div>
            </div>
            {isDesktop ? (
              <DesktopUpdateAction version={version!} />
            ) : (
              <DockerUpdateAction version={version!} />
            )}
          </div>
        ) : isDev && version?.latest ? (
          <div className="text-sm text-muted-foreground">
            {t('about.devBuild', { version: version.latest.version })}
          </div>
        ) : (
          <div className="text-sm text-muted-foreground">
            {version?.latest ? t('about.upToDate') : t('about.unableToCheck')}
          </div>
        )}
      </div>
    </SettingSection>
  )
}

function DesktopUpdateAction({ version }: { version: VersionInfo }) {
  const { t } = useTranslation()
  const asset = version.latest?.assets ? detectPlatformAsset(version.latest.assets) : undefined

  if (asset) {
    return (
      <Button size="sm" onClick={() => openExternal(asset.url)}>
        <Download className="mr-1.5 h-3.5 w-3.5" />
        {t('about.downloadAsset', { name: asset.name })}
      </Button>
    )
  }
  return (
    <Button size="sm" onClick={() => openExternal(version.latest!.url)}>
      <Download className="mr-1.5 h-3.5 w-3.5" />
      {t('about.download')}
    </Button>
  )
}

function DockerUpdateAction(_props: { version: VersionInfo }) {
  const { t } = useTranslation()
  return (
    <div className="space-y-2">
      <div className="text-xs text-muted-foreground">{t('about.dockerUpdateHint')}</div>
      <CopyCommand command={`docker compose pull && docker compose up -d`} />
    </div>
  )
}
