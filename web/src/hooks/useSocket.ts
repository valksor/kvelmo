import { useEffect, useRef, useCallback } from 'react'
import { SocketClient } from '../lib/socket'

export function useSocket(
  url: string,
  onConnect?: () => void,
  onMessage?: (data: unknown) => void,
  onDisconnect?: () => void
) {
  const clientRef = useRef<SocketClient | null>(null)
  const reconnectTimer = useRef<number | null>(null)
  const attemptRef = useRef(0)

  const connect = useCallback(async () => {
    if (clientRef.current) return

    const client = new SocketClient(url)
    clientRef.current = client

    if (onMessage) {
      client.subscribe(onMessage)
    }

    try {
      await client.connect()
      attemptRef.current = 0
      onConnect?.()
    } catch {
      clientRef.current = null
      // Exponential backoff: 1s → 2s → 4s → ... capped at 30s, ±20% jitter
      const base = Math.min(1000 * Math.pow(2, attemptRef.current), 30000)
      const delay = base * (0.8 + Math.random() * 0.4)
      attemptRef.current += 1
      reconnectTimer.current = window.setTimeout(() => {
        reconnectTimer.current = null
        connect()
      }, delay)
    }
  }, [url, onConnect, onMessage])

  const disconnect = useCallback(() => {
    if (reconnectTimer.current) {
      clearTimeout(reconnectTimer.current)
      reconnectTimer.current = null
    }
    attemptRef.current = 0
    clientRef.current?.close()
    clientRef.current = null
    onDisconnect?.()
  }, [onDisconnect])

  const call = useCallback(<T,>(method: string, params?: unknown): Promise<T> => {
    if (!clientRef.current) {
      return Promise.reject(new Error('Not connected'))
    }
    return clientRef.current.call<T>(method, params)
  }, [])

  useEffect(() => {
    return () => {
      disconnect()
    }
  }, [disconnect])

  return { connect, disconnect, call }
}

export function useGlobalSocket() {
  // WebSocket URL for global socket proxy
  const url = `ws://${window.location.host}/ws/global`
  return useSocket(url)
}

export function useWorktreeSocket(worktreeId: string | null) {
  // WebSocket URL for worktree socket proxy
  const url = worktreeId
    ? `ws://${window.location.host}/ws/worktree/${worktreeId}`
    : ''

  return useSocket(url)
}
