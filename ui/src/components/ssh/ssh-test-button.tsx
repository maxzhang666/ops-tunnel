import { useState } from 'react'
import { Loader2, CheckCircle2, XCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useTestSSHConnection } from '@/hooks/use-ssh-connections'

export function SSHTestButton({ id }: { id: string }) {
  const testMutation = useTestSSHConnection()
  const [result, setResult] = useState<{ ok: boolean; msg: string } | null>(null)

  const handleTest = () => {
    setResult(null)
    testMutation.mutate(id, {
      onSuccess: (data) => {
        setResult({
          ok: data.status === 'ok',
          msg: data.status === 'ok' ? `${data.latencyMs}ms` : data.message,
        })
        setTimeout(() => setResult(null), 5000)
      },
      onError: (err) => {
        setResult({ ok: false, msg: err.message })
        setTimeout(() => setResult(null), 5000)
      },
    })
  }

  return (
    <span className="inline-flex items-center gap-1.5">
      <Button variant="outline" size="sm" onClick={handleTest} disabled={testMutation.isPending}>
        {testMutation.isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : 'Test'}
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
