// frontend/src/hooks/useWebSocket.ts
import { useEffect, useRef, useCallback } from 'react'
import { useAuth } from './useAuth'

export function useWebSocket(onMessage: (msg: { type: string; payload: unknown }) => void) {
  const { token } = useAuth()
  const wsRef = useRef<WebSocket | null>(null)

  const connect = useCallback(() => {
    if (!token) return
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws?token=${token}`)
    ws.onmessage = (e) => {
      try { onMessage(JSON.parse(e.data)) } catch {}
    }
    ws.onclose = () => { setTimeout(connect, 3000) }
    wsRef.current = ws
  }, [token, onMessage])

  useEffect(() => {
    connect()
    return () => { wsRef.current?.close() }
  }, [connect])
}
