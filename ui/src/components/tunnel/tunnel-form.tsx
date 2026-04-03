import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ChainSelector } from './chain-selector'
import { MappingEditor } from './mapping-editor'
import { ApiError } from '@/lib/api'
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
      {error && <div className="rounded-md bg-destructive/10 px-4 py-3 text-sm text-destructive">{error}</div>}

      <Card>
        <CardHeader><CardTitle>Basic</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Name</Label>
            <Input id="name" value={name} onChange={(e) => setName(e.target.value)} required />
          </div>
          <div className="space-y-2">
            <Label>Mode</Label>
            <select className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-xs" value={mode} onChange={(e) => handleModeChange(e.target.value as TunnelMode)}>
              <option value="local">Local (-L)</option>
              <option value="remote">Remote (-R)</option>
              <option value="dynamic">Dynamic SOCKS5 (-D)</option>
            </select>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>SSH Chain</CardTitle></CardHeader>
        <CardContent><ChainSelector value={chain} onChange={setChain} /></CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>Port Mappings</CardTitle></CardHeader>
        <CardContent><MappingEditor mode={mode} value={mappings} onChange={setMappings} /></CardContent>
      </Card>

      <div>
        <Button type="button" variant="ghost" size="sm" onClick={() => setShowAdvanced(!showAdvanced)}>
          {showAdvanced ? '▾ Hide Advanced' : '▸ Show Advanced'}
        </Button>
      </div>

      {showAdvanced && (
        <Card>
          <CardHeader><CardTitle>Policy</CardTitle></CardHeader>
          <CardContent className="space-y-4">
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={autoRestart} onChange={(e) => setAutoRestart(e.target.checked)} className="h-4 w-4 rounded border-gray-300" />
              <span className="text-sm">Auto Restart on Failure</span>
            </label>
            <div className="grid grid-cols-3 gap-3">
              <div className="space-y-1">
                <Label className="text-xs">Backoff Min (ms)</Label>
                <Input type="number" value={backoffMin} onChange={(e) => setBackoffMin(Number(e.target.value))} />
              </div>
              <div className="space-y-1">
                <Label className="text-xs">Backoff Max (ms)</Label>
                <Input type="number" value={backoffMax} onChange={(e) => setBackoffMax(Number(e.target.value))} />
              </div>
              <div className="space-y-1">
                <Label className="text-xs">Backoff Factor</Label>
                <Input type="number" step="0.1" value={backoffFactor} onChange={(e) => setBackoffFactor(Number(e.target.value))} />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label className="text-xs">Max Restarts Per Hour</Label>
                <Input type="number" value={maxRestarts} onChange={(e) => setMaxRestarts(Number(e.target.value))} />
              </div>
              <div className="space-y-1">
                <Label className="text-xs">Graceful Stop Timeout (ms)</Label>
                <Input type="number" value={gracefulTimeout} onChange={(e) => setGracefulTimeout(Number(e.target.value))} />
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <Separator />
      <div className="flex justify-end gap-3">
        <Button type="submit" disabled={submitting}>{submitting ? 'Saving...' : submitLabel}</Button>
      </div>
    </form>
  )
}
