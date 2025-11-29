'use client'

import { useState } from 'react'
import { Plus, Edit, Trash2, Save, X, AlertTriangle, Info, AlertCircle, ToggleLeft, ToggleRight } from 'lucide-react'

interface ThresholdRule {
  id: string
  name: string
  description: string
  service: string
  metric: string
  operator: 'gt' | 'lt' | 'eq' | 'gte' | 'lte'
  threshold: number
  severity: 'critical' | 'warning' | 'info'
  enabled: boolean
  cooldown: number
  created_at: string
  updated_at: string
}

interface RulesPanelProps {
  rules: ThresholdRule[]
  services: string[]
  onCreateRule: (rule: Omit<ThresholdRule, 'id' | 'created_at' | 'updated_at'>) => void
  onUpdateRule: (id: string, rule: Partial<ThresholdRule>) => void
  onDeleteRule: (id: string) => void
  onToggleRule: (id: string, enabled: boolean) => void
}

const operatorLabels: Record<string, string> = {
  gt: '>',
  lt: '<',
  eq: '=',
  gte: '≥',
  lte: '≤',
}

const severityConfig = {
  critical: {
    icon: AlertTriangle,
    bg: 'bg-red-100',
    text: 'text-red-700',
    border: 'border-red-200',
  },
  warning: {
    icon: AlertCircle,
    bg: 'bg-yellow-100',
    text: 'text-yellow-700',
    border: 'border-yellow-200',
  },
  info: {
    icon: Info,
    bg: 'bg-blue-100',
    text: 'text-blue-700',
    border: 'border-blue-200',
  },
}

const metricOptions = [
  { value: 'cpu', label: 'CPU Usage' },
  { value: 'memory', label: 'Memory Usage' },
  { value: 'latency', label: 'Latency' },
  { value: 'error_rate', label: 'Error Rate' },
  { value: 'request_count', label: 'Request Count' },
]

export function RulesPanel({
  rules,
  services,
  onCreateRule,
  onUpdateRule,
  onDeleteRule,
  onToggleRule,
}: RulesPanelProps) {
  const [editingId, setEditingId] = useState<string | null>(null)
  const [isCreating, setIsCreating] = useState(false)
  const [formData, setFormData] = useState<Partial<ThresholdRule>>({})

  const handleCreate = () => {
    setIsCreating(true)
    setFormData({
      name: '',
      description: '',
      service: services[0] || '',
      metric: 'cpu',
      operator: 'gt',
      threshold: 80,
      severity: 'warning',
      enabled: true,
      cooldown: 300,
    })
  }

  const handleEdit = (rule: ThresholdRule) => {
    setEditingId(rule.id)
    setFormData(rule)
  }

  const handleSave = () => {
    if (isCreating) {
      onCreateRule(formData as Omit<ThresholdRule, 'id' | 'created_at' | 'updated_at'>)
      setIsCreating(false)
    } else if (editingId) {
      onUpdateRule(editingId, formData)
      setEditingId(null)
    }
    setFormData({})
  }

  const handleCancel = () => {
    setIsCreating(false)
    setEditingId(null)
    setFormData({})
  }

  const RuleForm = () => (
    <div className="bg-gray-50 rounded-lg p-4 border border-gray-200 space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Rule Name
          </label>
          <input
            type="text"
            value={formData.name || ''}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-primary"
            placeholder="e.g., High CPU Alert"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Service
          </label>
          <select
            value={formData.service || ''}
            onChange={(e) => setFormData({ ...formData, service: e.target.value })}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-primary"
          >
            <option value="">All Services</option>
            {services.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Description
        </label>
        <input
          type="text"
          value={formData.description || ''}
          onChange={(e) => setFormData({ ...formData, description: e.target.value })}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-primary"
          placeholder="Describe when this alert should fire"
        />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Metric
          </label>
          <select
            value={formData.metric || 'cpu'}
            onChange={(e) => setFormData({ ...formData, metric: e.target.value })}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-primary"
          >
            {metricOptions.map((m) => (
              <option key={m.value} value={m.value}>
                {m.label}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Operator
          </label>
          <select
            value={formData.operator || 'gt'}
            onChange={(e) =>
              setFormData({ ...formData, operator: e.target.value as ThresholdRule['operator'] })
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-primary"
          >
            <option value="gt">&gt; Greater than</option>
            <option value="gte">≥ Greater or equal</option>
            <option value="lt">&lt; Less than</option>
            <option value="lte">≤ Less or equal</option>
            <option value="eq">= Equal to</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Threshold
          </label>
          <input
            type="number"
            value={formData.threshold || 0}
            onChange={(e) =>
              setFormData({ ...formData, threshold: parseFloat(e.target.value) })
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-primary"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Severity
          </label>
          <select
            value={formData.severity || 'warning'}
            onChange={(e) =>
              setFormData({ ...formData, severity: e.target.value as ThresholdRule['severity'] })
            }
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-primary"
          >
            <option value="critical">Critical</option>
            <option value="warning">Warning</option>
            <option value="info">Info</option>
          </select>
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Cooldown (seconds)
        </label>
        <input
          type="number"
          value={formData.cooldown || 300}
          onChange={(e) => setFormData({ ...formData, cooldown: parseInt(e.target.value) })}
          className="w-32 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-primary"
        />
        <p className="text-xs text-gray-500 mt-1">
          Minimum time between repeated alerts for this rule
        </p>
      </div>

      <div className="flex justify-end gap-2 pt-2">
        <button
          onClick={handleCancel}
          className="px-4 py-2 text-gray-600 hover:text-gray-800 border border-gray-300 rounded-lg hover:bg-gray-50 flex items-center gap-2"
        >
          <X className="w-4 h-4" />
          Cancel
        </button>
        <button
          onClick={handleSave}
          className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 flex items-center gap-2"
        >
          <Save className="w-4 h-4" />
          Save Rule
        </button>
      </div>
    </div>
  )

  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-200">
      <div className="p-4 border-b border-gray-200 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-900">Threshold Rules</h2>
        {!isCreating && (
          <button
            onClick={handleCreate}
            className="px-3 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 flex items-center gap-2 text-sm"
          >
            <Plus className="w-4 h-4" />
            Add Rule
          </button>
        )}
      </div>

      <div className="p-4 space-y-4">
        {isCreating && <RuleForm />}

        {rules.length === 0 && !isCreating ? (
          <div className="text-center py-8 text-gray-500">
            <AlertCircle className="w-12 h-12 mx-auto mb-3 text-gray-300" />
            <p className="font-medium">No rules configured</p>
            <p className="text-sm mt-1">Create your first threshold rule to start monitoring</p>
          </div>
        ) : (
          <div className="space-y-3">
            {rules.map((rule) => {
              const config = severityConfig[rule.severity]
              const Icon = config.icon

              if (editingId === rule.id) {
                return <RuleForm key={rule.id} />
              }

              return (
                <div
                  key={rule.id}
                  className={`p-4 rounded-lg border ${
                    rule.enabled ? 'bg-white' : 'bg-gray-50 opacity-60'
                  } ${config.border}`}
                >
                  <div className="flex items-start justify-between">
                    <div className="flex items-start gap-3">
                      <div className={`p-2 rounded-lg ${config.bg}`}>
                        <Icon className={`w-4 h-4 ${config.text}`} />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <h3 className="font-medium text-gray-900">{rule.name}</h3>
                          <span
                            className={`text-xs px-2 py-0.5 rounded ${config.bg} ${config.text}`}
                          >
                            {rule.severity}
                          </span>
                          {!rule.enabled && (
                            <span className="text-xs px-2 py-0.5 rounded bg-gray-100 text-gray-500">
                              Disabled
                            </span>
                          )}
                        </div>
                        <p className="text-sm text-gray-500 mt-1">{rule.description}</p>
                        <div className="flex items-center gap-4 mt-2 text-sm text-gray-600">
                          <span>
                            <strong>{rule.service || 'All'}</strong> ·{' '}
                            {metricOptions.find((m) => m.value === rule.metric)?.label}
                          </span>
                          <span className="font-mono bg-gray-100 px-2 py-0.5 rounded">
                            {operatorLabels[rule.operator]} {rule.threshold}
                          </span>
                          <span className="text-xs text-gray-400">
                            Cooldown: {rule.cooldown}s
                          </span>
                        </div>
                      </div>
                    </div>

                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => onToggleRule(rule.id, !rule.enabled)}
                        className={`p-2 rounded-lg transition-colors ${
                          rule.enabled
                            ? 'text-green-600 hover:bg-green-50'
                            : 'text-gray-400 hover:bg-gray-100'
                        }`}
                        title={rule.enabled ? 'Disable rule' : 'Enable rule'}
                      >
                        {rule.enabled ? (
                          <ToggleRight className="w-5 h-5" />
                        ) : (
                          <ToggleLeft className="w-5 h-5" />
                        )}
                      </button>
                      <button
                        onClick={() => handleEdit(rule)}
                        className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg"
                      >
                        <Edit className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => onDeleteRule(rule.id)}
                        className="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
