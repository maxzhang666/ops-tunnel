import type { TunnelEvent } from '@/types/api'

type EventCallback = (event: TunnelEvent) => void

export class EventSocket {
  private ws: WebSocket | null = null
  private listeners = new Set<EventCallback>()
  private reconnectDelay = 1000
  private closed = false
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private wsUrl: string | null = null

  /** Set a direct WebSocket URL (used in desktop mode to bypass Wails asset server). */
  setUrl(url: string): void {
    this.wsUrl = url
  }

  connect(): void {
    this.closed = false
    const url = this.wsUrl ?? `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}/ws`

    this.ws = new WebSocket(url)

    this.ws.onopen = () => {
      this.reconnectDelay = 1000
    }

    this.ws.onmessage = (e) => {
      try {
        const event: TunnelEvent = JSON.parse(e.data)
        this.listeners.forEach((cb) => cb(event))
      } catch {
        // ignore non-JSON messages
      }
    }

    this.ws.onclose = () => {
      if (this.closed) return
      this.reconnectTimer = setTimeout(() => {
        this.connect()
      }, this.reconnectDelay)
      this.reconnectDelay = Math.min(this.reconnectDelay * 2, 8000)
    }

    this.ws.onerror = () => {
      this.ws?.close()
    }
  }

  onEvent(cb: EventCallback): () => void {
    this.listeners.add(cb)
    return () => this.listeners.delete(cb)
  }

  close(): void {
    this.closed = true
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer)
    this.ws?.close()
  }
}

export const eventSocket = new EventSocket()
