'use client'

import { useEffect, useRef, useState, useCallback } from 'react'

interface WebSocketMessage {
  type: 'alert' | 'metric' | 'service_status' | 'heartbeat'
  data: unknown
}

interface UseWebSocketOptions {
  url: string
  token: string
  onMessage?: (message: WebSocketMessage) => void
  onAlert?: (alert: Alert) => void
  onMetric?: (metric: ServiceMetric) => void
  onServiceStatus?: (status: ServiceStatus) => void
  reconnectInterval?: number
  maxReconnectAttempts?: number
}

interface Alert {
  id: string
  service: string
  type: string
  severity: 'critical' | 'warning' | 'info'
  message: string
  value: number
  threshold: number
  timestamp: string
  acknowledged: boolean
}

interface ServiceMetric {
  service: string
  cpu: number
  memory: number
  latency: number
  error_rate: number
  request_count: number
  timestamp: string
}

interface ServiceStatus {
  service: string
  status: 'healthy' | 'degraded' | 'down'
  last_seen: string
}

interface WebSocketState {
  isConnected: boolean
  isConnecting: boolean
  error: string | null
  reconnectAttempts: number
}

export function useWebSocket({
  url,
  token,
  onMessage,
  onAlert,
  onMetric,
  onServiceStatus,
  reconnectInterval = 3000,
  maxReconnectAttempts = 10,
}: UseWebSocketOptions) {
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const [state, setState] = useState<WebSocketState>({
    isConnected: false,
    isConnecting: false,
    error: null,
    reconnectAttempts: 0,
  })

  const cleanup = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
  }, [])

  const connect = useCallback(() => {
    cleanup()

    setState((prev) => ({ ...prev, isConnecting: true, error: null }))

    try {
      const wsUrl = `${url}?token=${encodeURIComponent(token)}`
      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => {
        console.log('[WebSocket] Connected')
        setState({
          isConnected: true,
          isConnecting: false,
          error: null,
          reconnectAttempts: 0,
        })
      }

      ws.onclose = (event) => {
        console.log('[WebSocket] Disconnected', event.code, event.reason)
        setState((prev) => ({
          ...prev,
          isConnected: false,
          isConnecting: false,
        }))

        // Attempt reconnection if not a clean close
        if (event.code !== 1000) {
          setState((prev) => {
            if (prev.reconnectAttempts < maxReconnectAttempts) {
              reconnectTimeoutRef.current = setTimeout(() => {
                connect()
              }, reconnectInterval)
              return {
                ...prev,
                reconnectAttempts: prev.reconnectAttempts + 1,
                error: `Disconnected. Reconnecting... (${prev.reconnectAttempts + 1}/${maxReconnectAttempts})`,
              }
            }
            return {
              ...prev,
              error: 'Max reconnection attempts reached. Please refresh the page.',
            }
          })
        }
      }

      ws.onerror = (error) => {
        console.error('[WebSocket] Error:', error)
        setState((prev) => ({
          ...prev,
          error: 'WebSocket connection error',
        }))
      }

      ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data)

          onMessage?.(message)

          switch (message.type) {
            case 'alert':
              onAlert?.(message.data as Alert)
              break
            case 'metric':
              onMetric?.(message.data as ServiceMetric)
              break
            case 'service_status':
              onServiceStatus?.(message.data as ServiceStatus)
              break
            case 'heartbeat':
              // Heartbeat to keep connection alive
              break
            default:
              console.warn('[WebSocket] Unknown message type:', message.type)
          }
        } catch (err) {
          console.error('[WebSocket] Failed to parse message:', err)
        }
      }
    } catch (err) {
      console.error('[WebSocket] Failed to connect:', err)
      setState((prev) => ({
        ...prev,
        isConnecting: false,
        error: 'Failed to establish WebSocket connection',
      }))
    }
  }, [url, token, onMessage, onAlert, onMetric, onServiceStatus, reconnectInterval, maxReconnectAttempts, cleanup])

  const disconnect = useCallback(() => {
    cleanup()
    setState({
      isConnected: false,
      isConnecting: false,
      error: null,
      reconnectAttempts: 0,
    })
  }, [cleanup])

  const sendMessage = useCallback((type: string, data: unknown) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type, data }))
      return true
    }
    return false
  }, [])

  // Auto-connect on mount
  useEffect(() => {
    if (token) {
      connect()
    }
    return cleanup
  }, [token, connect, cleanup])

  return {
    ...state,
    connect,
    disconnect,
    sendMessage,
  }
}

// Connection status indicator component
interface ConnectionStatusProps {
  isConnected: boolean
  isConnecting: boolean
  error: string | null
}

export function ConnectionStatus({ isConnected, isConnecting, error }: ConnectionStatusProps) {
  if (isConnecting) {
    return (
      <div className="flex items-center gap-2 text-yellow-600">
        <div className="w-2 h-2 bg-yellow-500 rounded-full animate-pulse" />
        <span className="text-sm">Connecting...</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center gap-2 text-red-600">
        <div className="w-2 h-2 bg-red-500 rounded-full" />
        <span className="text-sm">{error}</span>
      </div>
    )
  }

  return (
    <div className={`flex items-center gap-2 ${isConnected ? 'text-green-600' : 'text-gray-500'}`}>
      <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-gray-400'}`} />
      <span className="text-sm">{isConnected ? 'Connected' : 'Disconnected'}</span>
    </div>
  )
}
