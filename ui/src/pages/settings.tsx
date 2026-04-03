import { useTheme } from 'next-themes'
import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useSettings, useUpdateSettings, useVersion } from '@/hooks/use-settings'
import { GeneralSection } from '@/components/settings/general-section'
import { AppearanceSection } from '@/components/settings/appearance-section'
import { DesktopSection } from '@/components/settings/desktop-section'
import { AboutSection } from '@/components/settings/about-section'

export default function SettingsPage() {
  const { t } = useTranslation()
  const { data: settings, isLoading } = useSettings()
  const { data: version, isLoading: versionLoading } = useVersion()
  const updateSettings = useUpdateSettings()
  const { setTheme } = useTheme()

  const handleUpdate = useCallback(
    (patch: Record<string, unknown>) => {
      const themePatch = (patch as { appearance?: { theme?: string } }).appearance?.theme
      if (themePatch) {
        setTheme(themePatch)
      }

      updateSettings.mutate(patch, {
        onError: () => toast.error(t('settings.saveFailed')),
      })
    },
    [updateSettings, setTheme, t],
  )

  if (isLoading || !settings) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="text-muted-foreground">{t('common.loading')}</div>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-xl space-y-6">
      <h2 className="text-xl font-bold">{t('settings.title')}</h2>

      <GeneralSection settings={settings} version={version} onUpdate={handleUpdate} />
      <AppearanceSection settings={settings} onUpdate={handleUpdate} />
      {version?.mode === 'desktop' && (
        <DesktopSection settings={settings} onUpdate={handleUpdate} />
      )}
      <AboutSection version={version} isLoading={versionLoading} />
    </div>
  )
}
