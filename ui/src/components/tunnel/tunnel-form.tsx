import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ChainSelector } from './chain-selector'
import { MappingEditor } from './mapping-editor'
import { ApiError } from '@/lib/api'
import { translateValidationErrors } from '@/lib/api-errors'
import type { Tunnel, TunnelMode, Mapping } from '@/types/api'

interface TunnelFormProps {
  initialData?: Tunnel
  onSubmit: (data: Partial<Tunnel>) => Promise<void>
  submitLabel: string
}

function defaultMapping(mode: TunnelMode): Mapping {
  const base: Mapping = { id: '', listen: { host: '127.0.0.1', port: 0 } }
  if (mode === 'local' || mode === 'remote') base.connect = { host: '', port: 0 }
  if (mode === 'dynamic') base.socks5 = { auth: 'none' }
  return base
}

export function TunnelForm({ initialData, onSubmit, submitLabel }: TunnelFormProps) {
  const { t } = useTranslation()
  const [name, setName] = useState(initialData?.name ?? '')
  const [mode, setMode] = useState<TunnelMode>(initialData?.mode ?? 'local')
  const [chain, setChain] = useState<string[]>(initialData?.chain ?? [])
  const [mappings, setMappings] = useState<Mapping[]>(initialData?.mappings ?? [defaultMapping('local')])
  const [autoRestart, setAutoRestart] = useState(initialData?.policy?.autoRestart ?? true)
  const [backoffMin, setBackoffMin] = useState(initialData?.policy?.restartBackoff?.minMs ?? 500)
  const [backoffMax, setBackoffMax] = useState(initialData?.policy?.restartBackoff?.maxMs ?? 15000)
  const [backoffFactor, setBackoffFactor] = useState(initialData?.policy?.restartBackoff?.factor ?? 1.7)
  const [maxRestarts, setMaxRestarts] = useState(initialData?.policy?.maxRestartsPerHour ?? 60)
  const [gracefulTimeout, setGracefulTimeout] = useState(initialData?.policy?.gracefulStopTimeoutMs ?? 5000)
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const handleModeChange = (newMode: TunnelMode) => {
    setMode(newMode)
    setMappings([defaultMapping(newMode)])
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setSubmitting(true)

    const data: Partial<Tunnel> = {
      name, mode, chain, mappings,
      policy: {
        autoStart: false,
        autoRestart,
        restartBackoff: { minMs: backoffMin, maxMs: backoffMax, factor: backoffFactor },
        maxRestartsPerHour: maxRestarts,
        gracefulStopTimeoutMs: gracefulTimeout,
      },
    }

    try {
      await onSubmit(data)
    } catch (err) {
      if (err instanceof ApiError) {
        const translated = err.body.details ? translateValidationErrors(err.body.details) : []
        const details = translated.map((d) => d.message).join(', ')
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
      {error && <div className="rounded-md bg-destructive/10 px-4 py-3 text-sm text-destructive">{error}</div>}

      <Card>
        <CardHeader><CardTitle>{t('common.basic')}</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">{t('common.name')}</Label>
            <Input id="name" value={name} onChange={(e) => setName(e.target.value)} required />
          </div>
          <div className="space-y-2">
            <Label>{t('tunnel.mode')}</Label>
            <select className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-xs" value={mode} onChange={(e) => handleModeChange(e.target.value as TunnelMode)}>
              <option value="local">{t('tunnel.modeLocal')}</option>
              <option value="remote">{t('tunnel.modeRemote')}</option>
              <option value="dynamic">{t('tunnel.modeDynamic')}</option>
            </select>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>{t('tunnel.sshChain')}</CardTitle></CardHeader>
        <CardContent><ChainSelector value={chain} onChange={setChain} /></CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>{t('tunnel.portMappings')}</CardTitle></CardHeader>
        <CardContent><MappingEditor mode={mode} value={mappings} onChange={setMappings} /></CardContent>
      </Card>

      <div>
        <Button type="button" variant="ghost" size="sm" onClick={() => setShowAdvanced(!showAdvanced)}>
          {showAdvanced ? `▾ ${t('common.hideAdvanced')}` : `▸ ${t('common.showAdvanced')}`}
        </Button>
      </div>

      {showAdvanced && (
        <Card>
          <CardHeader><CardTitle>{t('tunnel.policy')}</CardTitle></CardHeader>
          <CardContent className="space-y-4">
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={autoRestart} onChange={(e) => setAutoRestart(e.target.checked)} className="h-4 w-4 rounded border-gray-300" />
              <span className="text-sm">{t('tunnel.autoRestart')}</span>
            </label>
            <div className="grid grid-cols-3 gap-3">
              <div className="space-y-1">
                <Label className="text-xs">{t('tunnel.backoffMin')}</Label>
                <Input type="number" value={backoffMin} onChange={(e) => setBackoffMin(Number(e.target.value))} />
              </div>
              <div className="space-y-1">
                <Label className="text-xs">{t('tunnel.backoffMax')}</Label>
                <Input type="number" value={backoffMax} onChange={(e) => setBackoffMax(Number(e.target.value))} />
              </div>
              <div className="space-y-1">
                <Label className="text-xs">{t('tunnel.backoffFactor')}</Label>
                <Input type="number" step="0.1" value={backoffFactor} onChange={(e) => setBackoffFactor(Number(e.target.value))} />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label className="text-xs">{t('tunnel.maxRestartsPerHour')}</Label>
                <Input type="number" value={maxRestarts} onChange={(e) => setMaxRestarts(Number(e.target.value))} />
              </div>
              <div className="space-y-1">
                <Label className="text-xs">{t('tunnel.gracefulStopTimeout')}</Label>
                <Input type="number" value={gracefulTimeout} onChange={(e) => setGracefulTimeout(Number(e.target.value))} />
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="sticky bottom-0 -mx-1 border-t bg-background/95 px-1 py-3 backdrop-blur">
        <div className="flex justify-end gap-3">
          <Button type="submit" disabled={submitting}>{submitting ? t('common.saving') : submitLabel}</Button>
        </div>
      </div>
    </form>
  )
}
