import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

declare global {
  interface Window {
    runtime?: {
      EventsOn: (event: string, callback: (...args: unknown[]) => void) => () => void
    }
    go?: {
      main: {
        App: {
          DoMinimize: () => Promise<void>
          DoQuit: () => Promise<void>
        }
      }
    }
  }
}

interface CloseEvent {
  action: string
  running: number
}

export function CloseDialog() {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [event, setEvent] = useState<CloseEvent | null>(null)

  useEffect(() => {
    if (!window.runtime?.EventsOn) return
    return window.runtime.EventsOn('app:close-requested', (...args: unknown[]) => {
      const data = args[0] as CloseEvent
      if (data.action === 'quit' && data.running === 0) {
        window.go?.main.App.DoQuit()
        return
      }
      setEvent(data)
      setOpen(true)
    })
  }, [])

  const doMinimize = useCallback(() => {
    setOpen(false)
    window.go?.main.App.DoMinimize()
  }, [])

  const doQuit = useCallback(() => {
    setOpen(false)
    window.go?.main.App.DoQuit()
  }, [])

  if (!event) return null

  const showAsk = event.action === 'ask'
  const hasRunning = event.running > 0

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent showCloseButton={false}>
        <DialogHeader>
          <DialogTitle>
            {hasRunning ? t('close.runningTitle') : t('close.title')}
          </DialogTitle>
          <DialogDescription>
            {hasRunning
              ? t('close.runningMessage', { count: event.running })
              : t('close.message')}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            {t('close.cancel')}
          </Button>
          {showAsk && (
            <Button variant="secondary" onClick={doMinimize}>
              {t('close.minimize')}
            </Button>
          )}
          <Button variant="destructive" onClick={doQuit}>
            {t('close.quit')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
