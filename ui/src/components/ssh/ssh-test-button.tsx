import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, CheckCircle2, XCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useTestSSHConnection, useTestSSHConnectionDirect } from '@/hooks/use-ssh-connections'
import type { SSHConnection } from '@/types/api'

interface SSHTestButtonProps {
  id?: string
  getData?: () => Partial<SSHConnection>
}

export function SSHTestButton({ id, getData }: SSHTestButtonProps) {
  const { t } = useTranslation()
  const testById = useTestSSHConnection()
  const testByData = useTestSSHConnectionDirect()
  const [result, setResult] = useState<{ ok: boolean; msg: string } | null>(null)

  const isPending = testById.isPending || testByData.isPending

  const onResult = (data: { status: string; message: string; latencyMs?: number }) => {
    setResult({
      ok: data.status === 'ok',
      msg: data.status === 'ok' ? `${data.latencyMs}ms` : data.message,
    })
    setTimeout(() => setResult(null), 5000)
  }

  const onError = (err: Error) => {
    setResult({ ok: false, msg: err.message })
    setTimeout(() => setResult(null), 5000)
  }

  const handleTest = () => {
    setResult(null)
    if (id) {
      testById.mutate(id, { onSuccess: onResult, onError })
    } else if (getData) {
      testByData.mutate(getData(), { onSuccess: onResult, onError })
    }
  }

  return (
    <span className="inline-flex items-center gap-1.5">
      <Button type="button" variant="outline" size="sm" onClick={handleTest} disabled={isPending}>
        {isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : t('common.test')}
      </Button>
      {result && (
        <span className="inline-flex items-center gap-1 text-xs">
          {result.ok ? (
            <CheckCircle2 className="h-3.5 w-3.5 text-green-600" />
          ) : (
            <XCircle className="h-3.5 w-3.5 text-destructive" />
          )}
          <span className={result.ok ? 'text-green-600' : 'text-destructive'}>{result.msg}</span>
        </span>
      )}
    </span>
  )
}
