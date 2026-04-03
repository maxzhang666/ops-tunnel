import { useEffect } from 'react'
import { eventSocket } from '@/lib/ws'
import type { TunnelEvent } from '@/types/api'

export function useWsEvent(callback: (event: TunnelEvent) => void) {
  useEffect(() => {
    return eventSocket.onEvent(callback)
  }, [callback])
}
