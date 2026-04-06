import { useCallback, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { FolderOpen } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { SSHTestButton } from './ssh-test-button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent } from '@/components/ui/card'
import { ApiError } from '@/lib/api'
import { translateValidationErrors } from '@/lib/api-errors'
import type { SSHConnection, AuthType, PrivateKeySource, HostKeyVerifyMode } from '@/types/api'

const isDesktop = typeof window !== 'undefined' && 'go' in window

interface SSHFormProps {
  initialData?: SSHConnection
  onSubmit: (data: Partial<SSHConnection>) => Promise<void>
  submitLabel: string
  onCancel?: () => void
}

export function SSHForm({ initialData, onSubmit, submitLabel, onCancel }: SSHFormProps) {
  const { t } = useTranslation()
  const [name, setName] = useState(initialData?.name ?? '')
  const [host, setHost] = useState(initialData?.endpoint?.host ?? '')
  const [port, setPort] = useState(initialData?.endpoint?.port ?? 22)
  const [authType, setAuthType] = useState<AuthType>(initialData?.auth?.type ?? 'password')
  const [username, setUsername] = useState(initialData?.auth?.username ?? '')
  const [password, setPassword] = useState('')
  const [keySource, setKeySource] = useState<PrivateKeySource>(
    initialData?.auth?.privateKey?.source ?? 'file'
  )
  const [keyPem, setKeyPem] = useState('')
  const [keyFilePath, setKeyFilePath] = useState(initialData?.auth?.privateKey?.filePath ?? '')
  const [passphrase, setPassphrase] = useState('')
  const [hostKeyMode, setHostKeyMode] = useState<HostKeyVerifyMode>(
    initialData?.hostKeyVerification?.mode ?? 'acceptNew'
  )
  const [dialTimeout, setDialTimeout] = useState(initialData?.dialTimeoutMs ?? 10000)
  const [kaInterval, setKaInterval] = useState(initialData?.keepAlive?.intervalMs ?? 15000)
  const [kaMaxMissed, setKaMaxMissed] = useState(initialData?.keepAlive?.maxMissed ?? 3)
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const buildFormData = useCallback((): Partial<SSHConnection> => ({
    name,
    endpoint: { host, port },
    auth: {
      type: authType,
      username,
      ...(authType === 'password' ? { password } : {}),
      ...(authType === 'privateKey' ? {
        privateKey: {
          source: keySource,
          ...(keySource === 'inline' ? { keyPem } : { filePath: keyFilePath }),
          ...(passphrase ? { passphrase } : {}),
        },
      } : {}),
    },
    hostKeyVerification: { mode: hostKeyMode },
    dialTimeoutMs: dialTimeout,
    keepAlive: { intervalMs: kaInterval, maxMissed: kaMaxMissed },
  }), [name, host, port, authType, username, password, keySource, keyPem, keyFilePath, passphrase, hostKeyMode, dialTimeout, kaInterval, kaMaxMissed])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      await onSubmit(buildFormData())
    } catch (err) {
      if (err instanceof ApiError) {
        const details = err.body.details
          ? translateValidationErrors(err.body.details).map((d) => d.message).join(', ')
          : undefined
        setError(details || err.body.error)
      } else {
        setError(String(err))
      }
    } finally {
      setSubmitting(false)
    }
  }

  const handlePickFile = async () => {
    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const path = await (window as any).go.main.App.PickFile(t('ssh.selectKeyFile'))
      if (path) setKeyFilePath(path)
    } catch {
      // user cancelled
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col">
      <div className="flex-1 space-y-4 overflow-y-auto p-1">
      {error && (
        <div className="rounded-md bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      )}

      <Card>
        <CardContent className="space-y-4 pt-5">
          {/* Name + Username */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="name">{t('common.name')}</Label>
              <Input id="name" value={name} onChange={(e) => setName(e.target.value)} required />
            </div>
            {authType !== 'none' && (
              <div className="space-y-2">
                <Label htmlFor="username">{t('common.username')}</Label>
                <Input id="username" value={username} onChange={(e) => setUsername(e.target.value)} />
              </div>
            )}
          </div>

          {/* Host + Port */}
          <div className="grid grid-cols-4 gap-4">
            <div className="col-span-3 space-y-2">
              <Label htmlFor="host">{t('common.host')}</Label>
              <Input id="host" value={host} onChange={(e) => setHost(e.target.value)} required />
            </div>
            <div className="space-y-2">
              <Label htmlFor="port">{t('common.port')}</Label>
              <Input id="port" type="number" value={port} onChange={(e) => setPort(Number(e.target.value))} />
            </div>
          </div>

          {/* Auth type */}
          <div className="space-y-2">
            <Label>{t('ssh.authType')}</Label>
            <select
              className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-xs"
              value={authType}
              onChange={(e) => setAuthType(e.target.value as AuthType)}
            >
              <option value="password">{t('ssh.authPassword')}</option>
              <option value="privateKey">{t('ssh.authPrivateKey')}</option>
              <option value="none">{t('ssh.authNone')}</option>
            </select>
          </div>

          {/* Password auth */}
          {authType === 'password' && (
            <div className="space-y-2">
              <Label htmlFor="password">{t('common.password')}</Label>
              <Input
                id="password" type="password" value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder={initialData ? t('ssh.passwordUnchanged') : ''}
              />
            </div>
          )}

          {/* Private key auth */}
          {authType === 'privateKey' && (
            <>
              <div className="space-y-2">
                <Label>{t('ssh.keySource')}</Label>
                <select
                  className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-xs"
                  value={keySource}
                  onChange={(e) => setKeySource(e.target.value as PrivateKeySource)}
                >
                  <option value="file">{t('ssh.keySourceFile')}</option>
                  <option value="inline">{t('ssh.keySourceInline')}</option>
                </select>
              </div>
              {keySource === 'file' ? (
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="keyFile">{t('ssh.keyFilePath')}</Label>
                    <div className="flex gap-2">
                      <Input id="keyFile" value={keyFilePath} onChange={(e) => setKeyFilePath(e.target.value)} placeholder={t('ssh.keyFilePlaceholder')} className="flex-1" />
                      {isDesktop && (
                        <Button type="button" variant="outline" size="icon" className="shrink-0" onClick={handlePickFile}>
                          <FolderOpen className="h-4 w-4" />
                        </Button>
                      )}
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="passphrase">{t('ssh.passphrase')}</Label>
                    <Input id="passphrase" type="password" value={passphrase} onChange={(e) => setPassphrase(e.target.value)} />
                  </div>
                </div>
              ) : (
                <>
                  <div className="space-y-2">
                    <Label htmlFor="keyPem">{t('ssh.privateKeyPem')}</Label>
                    <Textarea id="keyPem" value={keyPem} onChange={(e) => setKeyPem(e.target.value)} rows={6} className="font-mono text-xs" placeholder={t('ssh.privateKeyPlaceholder')} />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="passphrase">{t('ssh.passphrase')}</Label>
                    <Input id="passphrase" type="password" value={passphrase} onChange={(e) => setPassphrase(e.target.value)} />
                  </div>
                </>
              )}
            </>
          )}
        </CardContent>
      </Card>

      <div>
        <Button type="button" variant="ghost" size="sm" onClick={() => setShowAdvanced(!showAdvanced)}>
          {showAdvanced ? `\u25be ${t('common.hideAdvanced')}` : `\u25b8 ${t('common.showAdvanced')}`}
        </Button>
      </div>

      {showAdvanced && (
        <Card>
          <CardContent className="space-y-4 pt-5">
            <div className="space-y-2">
              <Label>{t('ssh.hostKeyVerification')}</Label>
              <select
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-xs"
                value={hostKeyMode}
                onChange={(e) => setHostKeyMode(e.target.value as HostKeyVerifyMode)}
              >
                <option value="acceptNew">{t('ssh.hostKeyAcceptNew')}</option>
                <option value="strict">{t('ssh.hostKeyStrict')}</option>
                <option value="insecure">{t('ssh.hostKeyInsecure')}</option>
              </select>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <div className="space-y-2">
                <Label htmlFor="dialTimeout">{t('ssh.dialTimeout')}</Label>
                <Input id="dialTimeout" type="number" value={dialTimeout} onChange={(e) => setDialTimeout(Number(e.target.value))} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="kaInterval">{t('ssh.keepAliveInterval')}</Label>
                <Input id="kaInterval" type="number" value={kaInterval} onChange={(e) => setKaInterval(Number(e.target.value))} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="kaMaxMissed">{t('ssh.maxMissed')}</Label>
                <Input id="kaMaxMissed" type="number" value={kaMaxMissed} onChange={(e) => setKaMaxMissed(Number(e.target.value))} />
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      </div>
      <div className="shrink-0 border-t bg-background px-1 py-3">
        <div className="flex items-center gap-3">
          {initialData?.id ? (
            <SSHTestButton id={initialData.id} />
          ) : (
            <SSHTestButton getData={buildFormData} />
          )}
          <div className="ml-auto flex gap-3">
            {onCancel && (
              <Button type="button" variant="outline" onClick={onCancel}>
                {t('common.cancel')}
              </Button>
            )}
            <Button type="submit" disabled={submitting}>
              {submitting ? t('common.saving') : submitLabel}
            </Button>
          </div>
        </div>
      </div>
    </form>
  )
}
