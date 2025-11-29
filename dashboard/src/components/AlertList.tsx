'use client'

import { useState } from 'react'
import { AlertTriangle, AlertCircle, Info, CheckCircle, Clock, RefreshCw } from 'lucide-react'

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
  acknowledged_by?: string
  acknowledged_at?: string
}

interface AlertListProps {
  alerts: Alert[]
  onAcknowledge: (alertId: string) => void
  onRefresh: () => void
}

export function AlertList({ alerts, onAcknowledge, onRefresh }: AlertListProps) {
  const [filter, setFilter] = useState<'all' | 'critical' | 'warning' | 'info'>('all')
  const [showAcknowledged, setShowAcknowledged] = useState(false)

  const severityConfig = {
    critical: {
      icon: AlertTriangle,
      bg: 'bg-red-50',
      border: 'border-red-200',
      text: 'text-red-700',
      badge: 'bg-red-100 text-red-800',
    },
    warning: {
      icon: AlertCircle,
      bg: 'bg-yellow-50',
      border: 'border-yellow-200',
      text: 'text-yellow-700',
      badge: 'bg-yellow-100 text-yellow-800',
    },
    info: {
      icon: Info,
      bg: 'bg-blue-50',
      border: 'border-blue-200',
      text: 'text-blue-700',
      badge: 'bg-blue-100 text-blue-800',
    },
  }

  const filteredAlerts = alerts.filter((alert) => {
    if (filter !== 'all' && alert.severity !== filter) return false
    if (!showAcknowledged && alert.acknowledged) return false
    return true
  })

  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    
    if (diff < 60000) return 'Just now'
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`
    return date.toLocaleDateString()
  }

  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-200">
      <div className="p-4 border-b border-gray-200">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-gray-900">Alerts</h2>
          <button
            onClick={onRefresh}
            className="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
        </div>

        <div className="flex flex-wrap gap-2 items-center">
          <div className="flex gap-1 bg-gray-100 p-1 rounded-lg">
            {(['all', 'critical', 'warning', 'info'] as const).map((sev) => (
              <button
                key={sev}
                onClick={() => setFilter(sev)}
                className={`px-3 py-1 text-sm rounded-md transition-colors ${
                  filter === sev
                    ? 'bg-white text-gray-900 shadow-sm'
                    : 'text-gray-600 hover:text-gray-900'
                }`}
              >
                {sev.charAt(0).toUpperCase() + sev.slice(1)}
              </button>
            ))}
          </div>

          <label className="flex items-center gap-2 text-sm text-gray-600 ml-auto">
            <input
              type="checkbox"
              checked={showAcknowledged}
              onChange={(e) => setShowAcknowledged(e.target.checked)}
              className="w-4 h-4 rounded border-gray-300 text-primary focus:ring-primary"
            />
            Show acknowledged
          </label>
        </div>
      </div>

      <div className="divide-y divide-gray-100 max-h-[600px] overflow-y-auto">
        {filteredAlerts.length === 0 ? (
          <div className="p-8 text-center text-gray-500">
            <CheckCircle className="w-12 h-12 mx-auto mb-3 text-green-400" />
            <p className="font-medium">No alerts to display</p>
            <p className="text-sm mt-1">All systems operating normally</p>
          </div>
        ) : (
          filteredAlerts.map((alert) => {
            const config = severityConfig[alert.severity]
            const Icon = config.icon

            return (
              <div
                key={alert.id}
                className={`p-4 ${config.bg} ${alert.acknowledged ? 'opacity-60' : ''}`}
              >
                <div className="flex items-start gap-3">
                  <div className={`p-2 rounded-lg ${config.badge}`}>
                    <Icon className="w-4 h-4" />
                  </div>

                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className={`text-xs font-medium px-2 py-0.5 rounded ${config.badge}`}>
                        {alert.severity.toUpperCase()}
                      </span>
                      <span className="text-xs text-gray-500 font-medium">
                        {alert.service}
                      </span>
                      <span className="text-xs text-gray-400">•</span>
                      <span className="text-xs text-gray-500">{alert.type}</span>
                    </div>

                    <p className={`font-medium ${config.text} mb-1`}>{alert.message}</p>

                    <div className="flex items-center gap-4 text-xs text-gray-500">
                      <span className="flex items-center gap-1">
                        <Clock className="w-3 h-3" />
                        {formatTime(alert.timestamp)}
                      </span>
                      <span>
                        Value: <strong>{alert.value.toFixed(2)}</strong> / Threshold:{' '}
                        <strong>{alert.threshold.toFixed(2)}</strong>
                      </span>
                    </div>

                    {alert.acknowledged && alert.acknowledged_by && (
                      <p className="text-xs text-gray-400 mt-2">
                        Acknowledged by {alert.acknowledged_by} at{' '}
                        {new Date(alert.acknowledged_at!).toLocaleString()}
                      </p>
                    )}
                  </div>

                  {!alert.acknowledged && (
                    <button
                      onClick={() => onAcknowledge(alert.id)}
                      className="px-3 py-1 text-sm bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors whitespace-nowrap"
                    >
                      Acknowledge
                    </button>
                  )}
                </div>
              </div>
            )
          })
        )}
      </div>

      {filteredAlerts.length > 0 && (
        <div className="p-3 border-t border-gray-200 text-center text-sm text-gray-500">
          Showing {filteredAlerts.length} of {alerts.length} alerts
        </div>
      )}
    </div>
  )
}
