'use client'

import { Server, Cpu, HardDrive, Clock, AlertTriangle } from 'lucide-react'

interface ServiceCardProps {
  service: {
    name: string
    status: string
    cpu_usage: number
    memory_usage: number
    latency_p95: number
    error_rate: number
    last_update: string
  }
}

export function ServiceCard({ service }: ServiceCardProps) {
  const statusColors = {
    healthy: 'bg-green-500',
    warning: 'bg-yellow-500',
    unhealthy: 'bg-red-500',
    unknown: 'bg-gray-400',
  }

  const statusBgColors = {
    healthy: 'bg-green-50 border-green-200',
    warning: 'bg-yellow-50 border-yellow-200',
    unhealthy: 'bg-red-50 border-red-200',
    unknown: 'bg-gray-50 border-gray-200',
  }

  return (
    <div className={`bg-white rounded-xl shadow-sm border p-6 ${statusBgColors[service.status as keyof typeof statusBgColors] || statusBgColors.unknown}`}>
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 bg-primary-100 rounded-lg flex items-center justify-center">
            <Server className="w-5 h-5 text-primary-600" />
          </div>
          <div>
            <h3 className="font-semibold text-gray-900 capitalize">{service.name}</h3>
            <div className="flex items-center gap-2">
              <div className={`w-2 h-2 rounded-full ${statusColors[service.status as keyof typeof statusColors] || statusColors.unknown}`}></div>
              <span className="text-sm text-gray-500 capitalize">{service.status}</span>
            </div>
          </div>
        </div>
      </div>

      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2 text-sm text-gray-600">
            <Cpu className="w-4 h-4" />
            CPU
          </div>
          <div className="flex items-center gap-2">
            <div className="w-20 h-2 bg-gray-200 rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full transition-all ${
                  service.cpu_usage > 80 ? 'bg-red-500' :
                  service.cpu_usage > 60 ? 'bg-yellow-500' : 'bg-green-500'
                }`}
                style={{ width: `${service.cpu_usage}%` }}
              />
            </div>
            <span className="text-sm font-medium text-gray-900 w-12 text-right">
              {service.cpu_usage.toFixed(1)}%
            </span>
          </div>
        </div>

        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2 text-sm text-gray-600">
            <HardDrive className="w-4 h-4" />
            Memory
          </div>
          <div className="flex items-center gap-2">
            <div className="w-20 h-2 bg-gray-200 rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full transition-all ${
                  service.memory_usage > 80 ? 'bg-red-500' :
                  service.memory_usage > 60 ? 'bg-yellow-500' : 'bg-green-500'
                }`}
                style={{ width: `${service.memory_usage}%` }}
              />
            </div>
            <span className="text-sm font-medium text-gray-900 w-12 text-right">
              {service.memory_usage.toFixed(1)}%
            </span>
          </div>
        </div>

        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2 text-sm text-gray-600">
            <Clock className="w-4 h-4" />
            Latency P95
          </div>
          <span className="text-sm font-medium text-gray-900">
            {service.latency_p95.toFixed(0)}ms
          </span>
        </div>

        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2 text-sm text-gray-600">
            <AlertTriangle className="w-4 h-4" />
            Error Rate
          </div>
          <span className={`text-sm font-medium ${
            service.error_rate > 5 ? 'text-red-600' :
            service.error_rate > 1 ? 'text-yellow-600' : 'text-green-600'
          }`}>
            {service.error_rate.toFixed(2)}%
          </span>
        </div>
      </div>

      {service.last_update && (
        <p className="text-xs text-gray-400 mt-4">
          Last update: {new Date(service.last_update).toLocaleTimeString()}
        </p>
      )}
    </div>
  )
}
