// API client for the UI Backend

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8007'

interface ApiResponse<T> {
  success: boolean
  data?: T
  error?: string
  meta?: {
    total: number
    page: number
    limit: number
    pages: number
  }
}

class ApiClient {
  private baseUrl: string
  private token: string | null = null

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('token')
    }
  }

  setToken(token: string) {
    this.token = token
    if (typeof window !== 'undefined') {
      localStorage.setItem('token', token)
    }
  }

  clearToken() {
    this.token = null
    if (typeof window !== 'undefined') {
      localStorage.removeItem('token')
    }
  }

  getToken(): string | null {
    return this.token
  }

  private async request<T>(
    method: string,
    endpoint: string,
    body?: object
  ): Promise<ApiResponse<T>> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    }

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`
    }

    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    })

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || 'Request failed')
    }

    return data
  }

  // Authentication
  async login(username: string, password: string) {
    const response = await this.request<{
      token: string
      expires_at: string
      user: {
        id: string
        username: string
        role: string
      }
    }>('POST', '/api/auth/login', { username, password })

    if (response.success && response.data) {
      this.setToken(response.data.token)
    }

    return response
  }

  async refreshToken() {
    return this.request<{ token: string; expires_at: string }>(
      'POST',
      '/api/auth/refresh'
    )
  }

  // Services
  async getServices() {
    return this.request<
      Array<{
        name: string
        display_name: string
        status: string
      }>
    >('GET', '/api/services')
  }

  async getServiceMetrics(service: string, window?: string) {
    const params = window ? `?window=${window}` : ''
    return this.request<object[]>('GET', `/api/services/${service}/metrics${params}`)
  }

  async getLatestMetrics() {
    return this.request<
      Record<
        string,
        {
          service_name: string
          cpu_usage: number
          memory_usage: number
          latency_p95: number
          error_rate: number
          request_count: number
          timestamp: string
        }
      >
    >('GET', '/api/metrics/latest')
  }

  // Alerts
  async getAlerts(params?: {
    page?: number
    limit?: number
    service?: string
    severity?: string
  }) {
    const queryParams = new URLSearchParams()
    if (params?.page) queryParams.set('page', params.page.toString())
    if (params?.limit) queryParams.set('limit', params.limit.toString())
    if (params?.service) queryParams.set('service', params.service)
    if (params?.severity) queryParams.set('severity', params.severity)

    const query = queryParams.toString()
    return this.request<
      Array<{
        id: string
        service_name: string
        type: string
        severity: 'critical' | 'warning' | 'info'
        message: string
        value: number
        threshold: number
        timestamp: string
        acknowledged: boolean
        acknowledged_by?: string
        acknowledged_at?: string
      }>
    >('GET', `/api/alerts${query ? `?${query}` : ''}`)
  }

  async getAlert(id: string) {
    return this.request<object>('GET', `/api/alerts/${id}`)
  }

  async acknowledgeAlert(id: string) {
    return this.request<void>('POST', `/api/alerts/${id}/acknowledge`)
  }

  // Rules
  async getRules() {
    return this.request<
      Array<{
        id: string
        name: string
        description: string
        service_name: string
        metric_type: string
        threshold: number
        operator: string
        severity: 'critical' | 'warning' | 'info'
        enabled: boolean
        cooldown_seconds: number
        created_at: string
        updated_at: string
      }>
    >('GET', '/api/rules')
  }

  async createRule(rule: {
    name: string
    description?: string
    service_name: string
    metric_type: string
    threshold: number
    operator: string
    severity: string
    enabled: boolean
    cooldown_seconds?: number
  }) {
    return this.request<object>('POST', '/api/rules', rule)
  }

  async updateRule(
    id: string,
    rule: {
      name?: string
      description?: string
      service_name?: string
      metric_type?: string
      threshold?: number
      operator?: string
      severity?: string
      enabled?: boolean
      cooldown_seconds?: number
    }
  ) {
    return this.request<object>('PUT', `/api/rules/${id}`, rule)
  }

  async deleteRule(id: string) {
    return this.request<void>('DELETE', `/api/rules/${id}`)
  }

  // Dashboard Stats
  async getDashboardStats() {
    return this.request<{
      total_services: number
      healthy_services: number
      total_alerts: number
      critical_alerts: number
      warning_alerts: number
      active_rules: number
      service_stats: Record<
        string,
        {
          name: string
          status: string
          cpu_usage: number
          memory_usage: number
          latency_p95: number
          error_rate: number
          last_update: string
        }
      >
      recent_alerts: object[]
    }>('GET', '/api/dashboard/stats')
  }

  // Health
  async health() {
    return this.request<{ status: string; service: string }>('GET', '/api/health')
  }
}

export const api = new ApiClient(API_URL)
export default api
