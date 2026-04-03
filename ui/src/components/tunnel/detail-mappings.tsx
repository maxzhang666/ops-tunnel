import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Copy, Check } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import type { Tunnel, TunnelStatus } from '@/types/api'

const stateColors: Record<string, string> = {
  listening: 'text-green-600',
  stopped: 'text-muted-foreground',
  error: 'text-destructive',
}

function CopyButton({ text }: { text: string }) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)

  const handleCopy = () => {
    navigator.clipboard.writeText(text)
    toast.success(t('mapping.copied', { text }))
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <Button variant="ghost" size="icon" className="h-7 w-7" onClick={handleCopy}>
      {copied ? <Check className="h-3.5 w-3.5 text-green-600" /> : <Copy className="h-3.5 w-3.5" />}
    </Button>
  )
}

interface DetailMappingsProps {
  tunnel: Tunnel
  status?: TunnelStatus
}

export function DetailMappings({ tunnel, status }: DetailMappingsProps) {
  const { t } = useTranslation()
  return (
    <div className="rounded-lg border bg-card">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('mapping.listen')}</TableHead>
            <TableHead>{t('mapping.connect')}</TableHead>
            <TableHead>{t('mapping.state')}</TableHead>
            <TableHead>{t('mapping.detail')}</TableHead>
            <TableHead className="w-12">{t('mapping.copy')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {tunnel.mappings.map((mapping) => {
            const ms = status?.mappings?.find((m) => m.mappingId === mapping.id)
            const mState = ms?.state ?? 'stopped'
            const listen = ms?.listen ?? `${mapping.listen.host}:${mapping.listen.port}`
            const connect = tunnel.mode === 'dynamic'
              ? t('mapping.socks5')
              : mapping.connect ? `${mapping.connect.host}:${mapping.connect.port}` : '\u2014'

            return (
              <TableRow key={mapping.id}>
                <TableCell className="font-mono text-xs">{listen}</TableCell>
                <TableCell className="font-mono text-xs">{connect}</TableCell>
                <TableCell>
                  <span className={cn('text-xs', stateColors[mState] ?? 'text-muted-foreground')}>
                    {'\u25CF'} {mState}
                  </span>
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">{ms?.detail ?? ''}</TableCell>
                <TableCell><CopyButton text={listen} /></TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </div>
  )
}
