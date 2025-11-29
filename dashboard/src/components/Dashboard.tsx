'use client'

import { useState, useEffect, useCallback } from 'react'
import {
  Activity,
  AlertTriangle,
  Bell,
  Cpu,
  HardDrive,
  LogOut,
  Menu,
  RefreshCw,
  Server,
  Settings,
  X,
  Zap
} from 'lucide-react'
import { ServiceCard } from './ServiceCard'
import { AlertList } from './AlertList'
import { MetricsChart } from './MetricsChart'
import { RulesPanel } from './RulesPanel'
import { StatsCards } from './StatsCards'
import { useWebSocket } from '@/hooks/useWebSocket'

interface DashboardProps {
  token: string
  onLogout: () => void
}

interface DashboardStats {
  total_services: number
  healthy_services: number
  total_alerts: number
  critical_alerts: number
  warning_alerts: number
  active_rules: number
  service_stats: Record<string, ServiceStats>
  recent_alerts: Alert[]
}

interface ServiceStats {
  name: string
  status: string
  cpu_usage: number
  memory_usage: number
  latency_p95: number
  error_rate: number
  last_update: string
}

interface Alert {
  id: string
  service_name: string
  severity: string
  title: string
  message: string
  timestamp: string
  acknowledged: boolean
}

export function Dashboard({ token, onLogout }: DashboardProps) {
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [activeTab, setActiveTab] = useState<'overview' | 'alerts' | 'rules'>('overview')
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const { isConnected, lastMessage } = useWebSocket(token)

  const fetchStats = useCallback(async () => {
    try {
      const response = await fetch(`${process.env.API_URL}/api/dashboard/stats`, {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      })
      const data = await response.json()
      if (data.success) {
        setStats(data.data)
      }
    } catch (err) {
      setError('Failed to fetch dashboard data')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => {
    fetchStats()
    const interval = setInterval(fetchStats, 30000)
    return () => clearInterval(interval)
  }, [fetchStats])

  useEffect(() => {
    if (lastMessage?.type === 'metric' || lastMessage?.type === 'alert') {
      fetchStats()
    }
  }, [lastMessage, fetchStats])

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-100">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading dashboard...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-100">
      {/* Header */}
      <header className="bg-white shadow-sm border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4">
              <button
                onClick={() => setSidebarOpen(!sidebarOpen)}
                className="lg:hidden p-2 rounded-lg hover:bg-gray-100"
              >
                <Menu className="w-6 h-6" />
              </button>
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 bg-primary-600 rounded-lg flex items-center justify-center">
                  <Activity className="w-6 h-6 text-white" />
                </div>
                <div>
                  <h1 className="text-xl font-bold text-gray-900">Microservices Monitor</h1>
                  <p className="text-sm text-gray-500">Real-time system health</p>
                </div>
              </div>
            </div>

            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`}></div>
                <span className="text-sm text-gray-600">
                  {isConnected ? 'Connected' : 'Disconnected'}
                </span>
              </div>

              <button
                onClick={fetchStats}
                className="p-2 rounded-lg hover:bg-gray-100 transition-colors"
                title="Refresh"
              >
                <RefreshCw className="w-5 h-5 text-gray-600" />
              </button>

              <button
                onClick={onLogout}
                className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
              >
                <LogOut className="w-4 h-4" />
                Logout
              </button>
            </div>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Navigation Tabs */}
        <div className="flex gap-2 mb-8">
          <button
            onClick={() => setActiveTab('overview')}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg font-medium transition-colors ${
              activeTab === 'overview'
                ? 'bg-primary-600 text-white'
                : 'bg-white text-gray-700 hover:bg-gray-50'
            }`}
          >
            <Activity className="w-4 h-4" />
            Overview
          </button>
          <button
            onClick={() => setActiveTab('alerts')}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg font-medium transition-colors ${
              activeTab === 'alerts'
                ? 'bg-primary-600 text-white'
                : 'bg-white text-gray-700 hover:bg-gray-50'
            }`}
          >
            <Bell className="w-4 h-4" />
            Alerts
            {stats?.critical_alerts ? (
              <span className="px-2 py-0.5 text-xs bg-red-500 text-white rounded-full">
                {stats.critical_alerts}
              </span>
            ) : null}
          </button>
          <button
            onClick={() => setActiveTab('rules')}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg font-medium transition-colors ${
              activeTab === 'rules'
                ? 'bg-primary-600 text-white'
                : 'bg-white text-gray-700 hover:bg-gray-50'
            }`}
          >
            <Settings className="w-4 h-4" />
            Rules
          </button>
        </div>

        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg flex items-center gap-3">
            <AlertTriangle className="w-5 h-5 text-red-500" />
            <p className="text-red-700">{error}</p>
            <button onClick={() => setError('')} className="ml-auto">
              <X className="w-4 h-4 text-red-500" />
            </button>
          </div>
        )}

        {activeTab === 'overview' && stats && (
          <div className="space-y-8">
            {/* Stats Cards */}
            <StatsCards stats={stats} />

            {/* Service Cards */}
            <div>
              <h2 className="text-lg font-semibold text-gray-900 mb-4">Services</h2>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                {Object.values(stats.service_stats || {}).map((service) => (
                  <ServiceCard key={service.name} service={service} />
                ))}
                {Object.keys(stats.service_stats || {}).length === 0 && (
                  <>
                    <ServiceCard service={{ name: 'auth', status: 'unknown', cpu_usage: 0, memory_usage: 0, latency_p95: 0, error_rate: 0, last_update: '' }} />
                    <ServiceCard service={{ name: 'orders', status: 'unknown', cpu_usage: 0, memory_usage: 0, latency_p95: 0, error_rate: 0, last_update: '' }} />
                    <ServiceCard service={{ name: 'payments', status: 'unknown', cpu_usage: 0, memory_usage: 0, latency_p95: 0, error_rate: 0, last_update: '' }} />
                    <ServiceCard service={{ name: 'notification', status: 'unknown', cpu_usage: 0, memory_usage: 0, latency_p95: 0, error_rate: 0, last_update: '' }} />
                  </>
                )}
              </div>
            </div>

            {/* Recent Alerts */}
            {stats.recent_alerts && stats.recent_alerts.length > 0 && (
              <div>
                <h2 className="text-lg font-semibold text-gray-900 mb-4">Recent Alerts</h2>
                <div className="bg-white rounded-xl shadow-sm border border-gray-200">
                  {stats.recent_alerts.map((alert) => (
                    <div
                      key={alert.id}
                      className={`p-4 border-b border-gray-100 last:border-b-0 flex items-center gap-4 ${
                        alert.severity === 'critical' ? 'bg-red-50' :
                        alert.severity === 'warning' ? 'bg-yellow-50' : ''
                      }`}
                    >
                      <AlertTriangle className={`w-5 h-5 ${
                        alert.severity === 'critical' ? 'text-red-500' :
                        alert.severity === 'warning' ? 'text-yellow-500' : 'text-blue-500'
                      }`} />
                      <div className="flex-1">
                        <p className="font-medium text-gray-900">{alert.title}</p>
                        <p className="text-sm text-gray-500">{alert.service_name}</p>
                      </div>
                      <span className="text-sm text-gray-400">
                        {new Date(alert.timestamp).toLocaleTimeString()}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {activeTab === 'alerts' && (
          <AlertList token={token} />
        )}

        {activeTab === 'rules' && (
          <RulesPanel token={token} />
        )}
      </div>
    </div>
  )
}
