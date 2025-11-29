'use client'

import { Server, AlertTriangle, Bell, CheckCircle2 } from 'lucide-react'

interface StatsCardsProps {
  stats: {
    total_services: number
    healthy_services: number
    total_alerts: number
    critical_alerts: number
    warning_alerts: number
    active_rules: number
  }
}

export function StatsCards({ stats }: StatsCardsProps) {
  const cards = [
    {
      title: 'Services',
      value: `${stats.healthy_services}/${stats.total_services}`,
      subtitle: 'Healthy',
      icon: Server,
      color: 'bg-blue-500',
      bgColor: 'bg-blue-50',
    },
    {
      title: 'Critical Alerts',
      value: stats.critical_alerts.toString(),
      subtitle: 'Active',
      icon: AlertTriangle,
      color: 'bg-red-500',
      bgColor: 'bg-red-50',
    },
    {
      title: 'Warnings',
      value: stats.warning_alerts.toString(),
      subtitle: 'Active',
      icon: Bell,
      color: 'bg-yellow-500',
      bgColor: 'bg-yellow-50',
    },
    {
      title: 'Active Rules',
      value: stats.active_rules.toString(),
      subtitle: 'Monitoring',
      icon: CheckCircle2,
      color: 'bg-green-500',
      bgColor: 'bg-green-50',
    },
  ]

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      {cards.map((card) => (
        <div
          key={card.title}
          className={`${card.bgColor} rounded-xl p-6 border border-gray-200`}
        >
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">{card.title}</p>
              <p className="text-3xl font-bold text-gray-900 mt-1">{card.value}</p>
              <p className="text-sm text-gray-500 mt-1">{card.subtitle}</p>
            </div>
            <div className={`${card.color} w-12 h-12 rounded-lg flex items-center justify-center`}>
              <card.icon className="w-6 h-6 text-white" />
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
