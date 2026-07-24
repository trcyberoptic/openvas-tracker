import { useEffect, useRef } from 'react'
import { useAuth } from './useAuth'

export function useWebSocket(onMessage: (msg: { type: string; payload: unknown }) => void) {
  const { token } = useAuth()
  const onMessageRef = useRef(onMessage)

  useEffect(() => {
    onMessageRef.current = onMessage
  }, [onMessage])

  useEffect(() => {
    if (!token) return
    // Scoped to this effect run so a token change tears the old socket down
    // completely — including a reconnect already queued by its onclose.
    let cancelled = false
    let ws: WebSocket | null = null
    let retry: ReturnType<typeof setTimeout> | undefined

    function connect() {
      if (cancelled) return
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      ws = new WebSocket(`${protocol}//${window.location.host}/ws?token=${token}`)
      ws.onmessage = (e) => {
        try {
          onMessageRef.current(JSON.parse(e.data))
        } catch {
          // ignore malformed frames
        }
      }
      ws.onclose = () => {
        if (!cancelled) retry = setTimeout(connect, 3000)
      }
    }
    connect()

    return () => {
      cancelled = true
      clearTimeout(retry)
      ws?.close()
    }
  }, [token])
}
