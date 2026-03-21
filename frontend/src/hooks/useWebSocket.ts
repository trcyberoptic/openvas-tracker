import { useEffect, useRef, useCallback } from 'react'
import { useAuth } from './useAuth'

export function useWebSocket(onMessage: (msg: { type: string; payload: unknown }) => void) {
  const { token } = useAuth()
  const wsRef = useRef<WebSocket | null>(null)
  const cancelledRef = useRef(false)
  const onMessageRef = useRef(onMessage)
  onMessageRef.current = onMessage

  const connect = useCallback(() => {
    if (!token || cancelledRef.current) return
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws?token=${token}`)
    ws.onmessage = (e) => {
      try { onMessageRef.current(JSON.parse(e.data)) } catch {}
    }
    ws.onclose = () => {
      if (!cancelledRef.current) setTimeout(connect, 3000)
    }
    wsRef.current = ws
  }, [token])

  useEffect(() => {
    cancelledRef.current = false
    connect()
    return () => {
      cancelledRef.current = true
      wsRef.current?.close()
    }
  }, [connect])
}
