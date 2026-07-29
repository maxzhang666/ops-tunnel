import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { ShieldAlert } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { useTrustHostKey } from '@/hooks/use-host-keys'
import { translateApiError } from '@/lib/api-errors'
import { ApiError } from '@/lib/api'
import type { HostKeyMismatch } from '@/types/api'

interface HostKeyMismatchDialogProps {
  mismatch: HostKeyMismatch | null
  onOpenChange: (open: boolean) => void
  onTrusted?: () => void
}

function Fingerprint({ label, value, keyType }: { label: string; value: string; keyType?: string }) {
  const { t } = useTranslation()
  return (
    <div className="space-y-1">
      <div className="text-xs font-medium text-muted-foreground">{label}</div>
      <div className="select-all break-all rounded bg-muted px-2 py-1.5 font-mono text-xs">
        {value}
      </div>
      {keyType && (
        <div className="text-xs text-muted-foreground">
          {t('ssh.hostKeyTypeLabel')}: {keyType}
        </div>
      )}
    </div>
  )
}

export function HostKeyMismatchDialog({
  mismatch,
  onOpenChange,
  onTrusted,
}: HostKeyMismatchDialogProps) {
  const { t } = useTranslation()
  const trustMutation = useTrustHostKey()

  if (!mismatch) return null

  const isChanged = mismatch.reason === 'changed'

  const handleTrust = () => {
    trustMutation.mutate(
      { hostPort: mismatch.hostPort, fingerprint: mismatch.offeredFingerprint },
      {
        onSuccess: () => {
          toast.success(t('ssh.hostKeyTrustSuccess'))
          onOpenChange(false)
          onTrusted?.()
        },
        onError: (err) => {
          const msg = err instanceof ApiError ? translateApiError(err.body) : err.message
          toast.error(msg)
        },
      },
    )
  }

  return (
    <Dialog open onOpenChange={onOpenChange} dismissible={false}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ShieldAlert className="h-4 w-4 text-destructive" />
            {isChanged ? t('ssh.hostKeyChangedTitle') : t('ssh.hostKeyUnknownTitle')}
          </DialogTitle>
          <DialogDescription>
            {isChanged
              ? t('ssh.hostKeyChangedDesc', { hostPort: mismatch.hostPort })
              : t('ssh.hostKeyUnknownDesc', { hostPort: mismatch.hostPort })}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          {isChanged && mismatch.storedFingerprint && (
            <Fingerprint
              label={t('ssh.hostKeyTrustedLabel')}
              value={mismatch.storedFingerprint}
              keyType={mismatch.storedKeyType}
            />
          )}
          <Fingerprint
            label={t('ssh.hostKeyOfferedLabel')}
            value={mismatch.offeredFingerprint}
            keyType={mismatch.offeredKeyType}
          />
          <div className="rounded border border-amber-300 bg-amber-50 px-2 py-1.5 text-xs text-amber-900 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200">
            {t('ssh.hostKeyVerifyHint')}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </Button>
          <Button
            variant="destructive"
            onClick={handleTrust}
            disabled={trustMutation.isPending}
          >
            {isChanged ? t('ssh.hostKeyTrustAction') : t('ssh.hostKeyRegisterAction')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
