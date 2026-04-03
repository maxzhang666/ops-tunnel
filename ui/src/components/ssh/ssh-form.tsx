import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { SSHTestButton } from './ssh-test-button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ApiError } from '@/lib/api'
import type { SSHConnection, AuthType, PrivateKeySource, HostKeyVerifyMode } from '@/types/api'

interface SSHFormProps {
  initialData?: SSHConnection
  onSubmit: (data: Partial<SSHConnection>) => Promise<void>
  submitLabel: string
}

export function SSHForm({ initialData, onSubmit, submitLabel }: SSHFormProps) {
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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setSubmitting(true)

    const data: Partial<SSHConnection> = {
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
    }

    try {
      await onSubmit(data)
    } catch (err) {
      if (err instanceof ApiError) {
        const details = err.body.details?.map((d) => d.message).join(', ')
        setError(details || err.body.error)
      } else {
        setError(String(err))
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {error && (
        <div className="rounded-md bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      )}

      <Card>
        <CardHeader><CardTitle>Basic</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Name</Label>
            <Input id="name" value={name} onChange={(e) => setName(e.target.value)} required />
          </div>
          <div className="grid grid-cols-3 gap-4">
            <div className="col-span-2 space-y-2">
              <Label htmlFor="host">Host</Label>
              <Input id="host" value={host} onChange={(e) => setHost(e.target.value)} required />
            </div>
            <div className="space-y-2">
              <Label htmlFor="port">Port</Label>
              <Input id="port" type="number" value={port} onChange={(e) => setPort(Number(e.target.value))} />
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>Authentication</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>Auth Type</Label>
            <select
              className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-xs"
              value={authType}
              onChange={(e) => setAuthType(e.target.value as AuthType)}
            >
              <option value="password">Password</option>
              <option value="privateKey">Private Key</option>
              <option value="none">None</option>
            </select>
          </div>

          {authType !== 'none' && (
            <div className="space-y-2">
              <Label htmlFor="username">Username</Label>
              <Input id="username" value={username} onChange={(e) => setUsername(e.target.value)} />
            </div>
          )}

          {authType === 'password' && (
            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password" type="password" value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder={initialData ? '(unchanged if empty)' : ''}
              />
            </div>
          )}

          {authType === 'privateKey' && (
            <>
              <div className="space-y-2">
                <Label>Key Source</Label>
                <select
                  className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-xs"
                  value={keySource}
                  onChange={(e) => setKeySource(e.target.value as PrivateKeySource)}
                >
                  <option value="file">File Path</option>
                  <option value="inline">Inline PEM</option>
                </select>
              </div>
              {keySource === 'file' ? (
                <div className="space-y-2">
                  <Label htmlFor="keyFile">Key File Path</Label>
                  <Input id="keyFile" value={keyFilePath} onChange={(e) => setKeyFilePath(e.target.value)} placeholder="/home/user/.ssh/id_ed25519" />
                </div>
              ) : (
                <div className="space-y-2">
                  <Label htmlFor="keyPem">Private Key PEM</Label>
                  <Textarea id="keyPem" value={keyPem} onChange={(e) => setKeyPem(e.target.value)} rows={6} className="font-mono text-xs" placeholder="-----BEGIN OPENSSH PRIVATE KEY-----" />
                </div>
              )}
              <div className="space-y-2">
                <Label htmlFor="passphrase">Passphrase (optional)</Label>
                <Input id="passphrase" type="password" value={passphrase} onChange={(e) => setPassphrase(e.target.value)} />
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <div>
        <Button type="button" variant="ghost" size="sm" onClick={() => setShowAdvanced(!showAdvanced)}>
          {showAdvanced ? '\u25be Hide Advanced' : '\u25b8 Show Advanced'}
        </Button>
      </div>

      {showAdvanced && (
        <Card>
          <CardHeader><CardTitle>Advanced</CardTitle></CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>Host Key Verification</Label>
              <select
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-xs"
                value={hostKeyMode}
                onChange={(e) => setHostKeyMode(e.target.value as HostKeyVerifyMode)}
              >
                <option value="acceptNew">Accept New</option>
                <option value="strict">Strict</option>
                <option value="insecure">Insecure (skip verification)</option>
              </select>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <div className="space-y-2">
                <Label htmlFor="dialTimeout">Dial Timeout (ms)</Label>
                <Input id="dialTimeout" type="number" value={dialTimeout} onChange={(e) => setDialTimeout(Number(e.target.value))} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="kaInterval">KeepAlive Interval (ms)</Label>
                <Input id="kaInterval" type="number" value={kaInterval} onChange={(e) => setKaInterval(Number(e.target.value))} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="kaMaxMissed">Max Missed</Label>
                <Input id="kaMaxMissed" type="number" value={kaMaxMissed} onChange={(e) => setKaMaxMissed(Number(e.target.value))} />
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="sticky bottom-0 -mx-1 border-t bg-background/95 px-1 py-3 backdrop-blur">
        <div className="flex justify-end gap-3">
          {initialData?.id && <SSHTestButton id={initialData.id} />}
          <Button type="submit" disabled={submitting}>
            {submitting ? 'Saving...' : submitLabel}
          </Button>
        </div>
      </div>
    </form>
  )
}
