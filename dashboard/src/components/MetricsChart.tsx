'use client'

import { useMemo } from 'react'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js'
import { Line } from 'react-chartjs-2'

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
)

interface MetricDataPoint {
  timestamp: string
  cpu: number
  memory: number
  latency: number
  error_rate: number
  request_count: number
}

interface MetricsChartProps {
  data: MetricDataPoint[]
  metric: 'cpu' | 'memory' | 'latency' | 'error_rate' | 'request_count'
  title: string
  color: string
  unit: string
}

export function MetricsChart({ data, metric, title, color, unit }: MetricsChartProps) {
  const chartData = useMemo(() => {
    const labels = data.map((d) => {
      const date = new Date(d.timestamp)
      return date.toLocaleTimeString('en-US', {
        hour: '2-digit',
        minute: '2-digit',
      })
    })

    const values = data.map((d) => d[metric])

    return {
      labels,
      datasets: [
        {
          label: title,
          data: values,
          borderColor: color,
          backgroundColor: `${color}20`,
          fill: true,
          tension: 0.4,
          pointRadius: 0,
          pointHoverRadius: 4,
          borderWidth: 2,
        },
      ],
    }
  }, [data, metric, title, color])

  const options = useMemo(
    () => ({
      responsive: true,
      maintainAspectRatio: false,
      interaction: {
        mode: 'index' as const,
        intersect: false,
      },
      plugins: {
        legend: {
          display: false,
        },
        tooltip: {
          backgroundColor: 'rgba(17, 24, 39, 0.9)',
          titleColor: '#fff',
          bodyColor: '#fff',
          padding: 12,
          displayColors: false,
          callbacks: {
            label: (context: { parsed: { y: number } }) => {
              return `${context.parsed.y.toFixed(2)} ${unit}`
            },
          },
        },
      },
      scales: {
        x: {
          grid: {
            display: false,
          },
          ticks: {
            color: '#9CA3AF',
            maxTicksLimit: 6,
          },
        },
        y: {
          grid: {
            color: '#F3F4F6',
          },
          ticks: {
            color: '#9CA3AF',
            callback: (value: number) => `${value}${unit}`,
          },
          min: 0,
        },
      },
    }),
    [unit]
  )

  return (
    <div className="bg-white rounded-xl p-4 border border-gray-200">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-medium text-gray-700">{title}</h3>
        {data.length > 0 && (
          <span className="text-2xl font-bold" style={{ color }}>
            {data[data.length - 1][metric].toFixed(2)}
            <span className="text-sm font-normal text-gray-500 ml-1">{unit}</span>
          </span>
        )}
      </div>
      <div className="h-48">
        {data.length > 0 ? (
          <Line data={chartData} options={options as object} />
        ) : (
          <div className="h-full flex items-center justify-center text-gray-400">
            No data available
          </div>
        )}
      </div>
    </div>
  )
}

interface MetricsGridProps {
  data: MetricDataPoint[]
}

export function MetricsGrid({ data }: MetricsGridProps) {
  const metrics = [
    { key: 'cpu' as const, title: 'CPU Usage', color: '#3B82F6', unit: '%' },
    { key: 'memory' as const, title: 'Memory Usage', color: '#8B5CF6', unit: '%' },
    { key: 'latency' as const, title: 'Latency', color: '#F59E0B', unit: 'ms' },
    { key: 'error_rate' as const, title: 'Error Rate', color: '#EF4444', unit: '%' },
  ]

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      {metrics.map((m) => (
        <MetricsChart
          key={m.key}
          data={data}
          metric={m.key}
          title={m.title}
          color={m.color}
          unit={m.unit}
        />
      ))}
    </div>
  )
}
