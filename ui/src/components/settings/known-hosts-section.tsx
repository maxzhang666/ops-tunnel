import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { SettingSection } from './setting-row'
import { useKnownHosts, useRevokeHostKey } from '@/hooks/use-host-keys'
import type { KnownHost } from '@/types/api'

export function KnownHostsSection() {
  const { t } = useTranslation()
  const { data, isLoading } = useKnownHosts()
  const revokeMutation = useRevokeHostKey()
  const [pendingRevoke, setPendingRevoke] = useState<KnownHost | null>(null)

  const hosts = data?.hosts ?? []

  const handleRevoke = () => {
    if (!pendingRevoke) return
    revokeMutation.mutate(pendingRevoke.hostPort, {
      onSuccess: () => {
        toast.success(t('ssh.knownHostsRevoked'))
        setPendingRevoke(null)
      },
      onError: () => toast.error(t('ssh.knownHostsRevokeFailed')),
    })
  }

  return (
    <SettingSection title={t('ssh.knownHostsTitle')}>
      {isLoading ? (
        <div className="px-4 py-3 text-sm text-muted-foreground">{t('common.loading')}</div>
      ) : hosts.length === 0 ? (
        <div className="px-4 py-3 text-sm text-muted-foreground">{t('ssh.knownHostsEmpty')}</div>
      ) : (
        hosts.map((host) => (
          <div key={host.hostPort} className="flex items-center justify-between gap-3 px-4 py-3">
            <div className="min-w-0 space-y-0.5">
              <div className="truncate text-sm font-medium">{host.hostPort}</div>
              <div className="truncate font-mono text-xs text-muted-foreground">
                {host.fingerprint || t('ssh.knownHostsUnparseable')}
              </div>
              <div className="text-xs text-muted-foreground">{host.keyType}</div>
            </div>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 shrink-0 text-destructive hover:text-destructive"
              onClick={() => setPendingRevoke(host)}
              title={t('ssh.knownHostsRevoke')}
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        ))
      )}

      <Dialog open={!!pendingRevoke} onOpenChange={(open) => !open && setPendingRevoke(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t('ssh.knownHostsRevoke')}</DialogTitle>
            <DialogDescription>
              {t('ssh.knownHostsRevokeConfirm', { hostPort: pendingRevoke?.hostPort })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setPendingRevoke(null)}>
              {t('common.cancel')}
            </Button>
            <Button
              variant="destructive"
              onClick={handleRevoke}
              disabled={revokeMutation.isPending}
            >
              {t('common.delete')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </SettingSection>
  )
}
