# EventFlow API Reference

Complete API documentation for all EventFlow services.

---

## Table of Contents

1. [Overview](#overview)
2. [Authentication](#authentication)
3. [UI Backend API](#ui-backend-api)
4. [Auth Service API](#auth-service-api)
5. [Orders Service API](#orders-service-api)
6. [Payments Service API](#payments-service-api)
7. [WebSocket Events](#websocket-events)
8. [Error Handling](#error-handling)

---

## Overview

### API Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         API GATEWAY PATTERN                             │
└─────────────────────────────────────────────────────────────────────────┘

                         ┌──────────────────────┐
                         │      Dashboard       │
                         │    (Next.js App)     │
                         └──────────┬───────────┘
                                    │
                         ┌──────────▼───────────┐
                         │     UI-Backend       │
                         │    (Port 8007)       │
                         │                      │
                         │  ┌────────────────┐  │
                         │  │  REST Routes   │  │
                         │  │  /api/*        │  │
                         │  └────────────────┘  │
                         │                      │
                         │  ┌────────────────┐  │
                         │  │  WebSocket     │  │
                         │  │  /ws           │  │
                         │  └────────────────┘  │
                         └──────────┬───────────┘
                                    │
           ┌────────────────────────┼────────────────────────┐
           │                        │                        │
    ┌──────▼──────┐          ┌──────▼──────┐          ┌──────▼──────┐
    │    Redis    │          │   Kafka     │          │  Services   │
    │  (Metrics)  │          │  (Events)   │          │  (Business) │
    └─────────────┘          └─────────────┘          └─────────────┘
```

### Base URLs

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          BASE URLS                                      │
├─────────────────────────────────────────────────────────────────────────┤
│  Environment    │  Base URL                                             │
├─────────────────┼───────────────────────────────────────────────────────┤
│  Local Dev      │  http://localhost:8007                                │
│  EC2            │  http://<EC2_PUBLIC_IP>:8007                          │
│  Production     │  https://api.yourdomain.com                           │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Authentication

### Login Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      AUTHENTICATION FLOW                                │
└─────────────────────────────────────────────────────────────────────────┘

  ┌──────────┐     POST /api/login      ┌────────────┐
  │  Client  │ ───────────────────────► │ UI-Backend │
  │          │   {email, password}      │            │
  └──────────┘                          └─────┬──────┘
       │                                      │
       │                                      ▼
       │                              ┌────────────┐
       │                              │Auth Service│
       │                              │  :8001     │
       │                              └─────┬──────┘
       │                                    │
       │       200 OK {token, user}         │
       │ ◄─────────────────────────────────-┘
       │
       │     Authorization: Bearer <token>
       │ ─────────────────────────────────►  Protected Routes
       ▼
```

### Login Endpoint

```http
POST /api/login
Content-Type: application/json

{
  "email": "admin@eventflow.io",
  "password": "admin123"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "user_123",
    "email": "admin@eventflow.io",
    "name": "Admin User",
    "role": "admin"
  }
}
```

### Default Credentials

```
┌─────────────────────────────────────────────────────────────┐
│  Email              │  Password   │  Role                   │
├─────────────────────┼─────────────┼─────────────────────────┤
│  admin@eventflow.io │  admin123   │  Administrator          │
└─────────────────────────────────────────────────────────────┘
```

---

## UI Backend API

The UI Backend serves as the API gateway for the dashboard.

### Health Check

```http
GET /api/health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "services": {
    "redis": "connected",
    "kafka": "connected"
  }
}
```

### Get All Metrics

```
┌─────────────────────────────────────────────────────────────┐
│                    METRICS ENDPOINT                         │
└─────────────────────────────────────────────────────────────┘

GET /api/metrics
           │
           ▼
    ┌──────────────┐
    │    Redis     │
    │   HGETALL    │
    │  metrics:*   │
    └──────────────┘
           │
           ▼
    Response: Array of ServiceMetrics
```

```http
GET /api/metrics
```

**Response:**
```json
{
  "services": [
    {
      "name": "auth-service",
      "status": "healthy",
      "metrics": {
        "cpu": 45.2,
        "memory": 128.5,
        "latency": 15.3,
        "errorRate": 0.1,
        "requestCount": 1523
      },
      "timestamp": "2024-01-15T10:30:00Z"
    },
    {
      "name": "orders-service",
      "status": "healthy",
      "metrics": {
        "cpu": 62.8,
        "memory": 256.7,
        "latency": 45.2,
        "errorRate": 0.5,
        "requestCount": 3421
      },
      "timestamp": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### Get Service Metrics

```http
GET /api/metrics/{serviceName}
```

**Parameters:**
| Name | Type | Description |
|------|------|-------------|
| serviceName | string | Service identifier (e.g., "auth-service") |

**Response:**
```json
{
  "name": "auth-service",
  "status": "healthy",
  "metrics": {
    "cpu": 45.2,
    "memory": 128.5,
    "latency": 15.3,
    "errorRate": 0.1,
    "requestCount": 1523
  },
  "history": [
    {"timestamp": "2024-01-15T10:29:00Z", "cpu": 44.1, "memory": 127.2},
    {"timestamp": "2024-01-15T10:30:00Z", "cpu": 45.2, "memory": 128.5}
  ]
}
```

### Get Active Alerts

```
┌─────────────────────────────────────────────────────────────┐
│                    ALERTS FLOW                              │
└─────────────────────────────────────────────────────────────┘

GET /api/alerts
         │
         ▼
  ┌─────────────┐     ┌─────────────┐
  │    Redis    │◄────│Alert Engine │
  │  alerts:*   │     │  Consumer   │
  └─────────────┘     └─────────────┘
         │
         ▼
  Response: Array of Alert objects
```

```http
GET /api/alerts
```

**Response:**
```json
{
  "alerts": [
    {
      "id": "alert_001",
      "severity": "critical",
      "service": "payments-service",
      "metric": "errorRate",
      "value": 15.5,
      "threshold": 10.0,
      "message": "Error rate exceeded threshold",
      "timestamp": "2024-01-15T10:30:00Z",
      "acknowledged": false
    }
  ],
  "total": 1,
  "critical": 1,
  "warning": 0
}
```

### Acknowledge Alert

```http
POST /api/alerts/{alertId}/acknowledge
```

**Response:**
```json
{
  "success": true,
  "alertId": "alert_001",
  "acknowledgedAt": "2024-01-15T10:35:00Z"
}
```

### Get System Overview

```http
GET /api/overview
```

**Response:**
```json
{
  "totalServices": 7,
  "healthyServices": 6,
  "unhealthyServices": 1,
  "activeAlerts": 3,
  "criticalAlerts": 1,
  "averageLatency": 35.4,
  "totalRequests": 15234,
  "errorRate": 0.8,
  "uptime": "99.95%"
}
```

---

## Auth Service API

### Service Info

```
┌─────────────────────────────────────────────────────────────┐
│  Base URL    │  http://localhost:8001 (internal)            │
│  Protocol    │  REST/JSON                                   │
│  Auth        │  JWT Bearer Token                            │
└─────────────────────────────────────────────────────────────┘
```

### Endpoints

#### Register User

```http
POST /api/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword",
  "name": "John Doe"
}
```

#### Login

```http
POST /api/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword"
}
```

#### Validate Token

```http
POST /api/auth/validate
Authorization: Bearer <token>
```

#### Health Check

```http
GET /health
```

---

## Orders Service API

### Service Info

```
┌─────────────────────────────────────────────────────────────┐
│  Base URL    │  http://localhost:8002 (internal)            │
│  Protocol    │  REST/JSON                                   │
│  Auth        │  JWT Bearer Token                            │
└─────────────────────────────────────────────────────────────┘
```

### Endpoints

#### Create Order

```http
POST /api/orders
Authorization: Bearer <token>
Content-Type: application/json

{
  "customerId": "cust_123",
  "items": [
    {"productId": "prod_001", "quantity": 2, "price": 29.99}
  ],
  "shippingAddress": {
    "street": "123 Main St",
    "city": "New York",
    "zipCode": "10001"
  }
}
```

#### Get Order

```http
GET /api/orders/{orderId}
Authorization: Bearer <token>
```

#### List Orders

```http
GET /api/orders?page=1&limit=10
Authorization: Bearer <token>
```

#### Update Order Status

```http
PATCH /api/orders/{orderId}/status
Authorization: Bearer <token>
Content-Type: application/json

{
  "status": "shipped"
}
```

---

## Payments Service API

### Service Info

```
┌─────────────────────────────────────────────────────────────┐
│  Base URL    │  http://localhost:8003 (internal)            │
│  Protocol    │  REST/JSON                                   │
│  Auth        │  JWT Bearer Token                            │
└─────────────────────────────────────────────────────────────┘
```

### Endpoints

#### Process Payment

```http
POST /api/payments
Authorization: Bearer <token>
Content-Type: application/json

{
  "orderId": "order_123",
  "amount": 59.98,
  "currency": "USD",
  "method": "credit_card",
  "cardDetails": {
    "last4": "4242",
    "expMonth": 12,
    "expYear": 2025
  }
}
```

#### Get Payment Status

```http
GET /api/payments/{paymentId}
Authorization: Bearer <token>
```

#### Refund Payment

```http
POST /api/payments/{paymentId}/refund
Authorization: Bearer <token>
Content-Type: application/json

{
  "amount": 59.98,
  "reason": "Customer request"
}
```

---

## WebSocket Events

### Connection

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      WEBSOCKET CONNECTION                               │
└─────────────────────────────────────────────────────────────────────────┘

Client                                           Server
  │                                                │
  │  ws://localhost:8007/ws                        │
  │ ─────────────────────────────────────────────► │
  │                                                │
  │  Connection Established                        │
  │ ◄───────────────────────────────────────────── │
  │                                                │
  │  {"type": "metrics", "data": {...}}            │
  │ ◄───────────────────────────────────────────── │
  │                    (every 5s)                  │
  │                                                │
  │  {"type": "alert", "data": {...}}              │
  │ ◄───────────────────────────────────────────── │
  │                 (on new alert)                 │
  │                                                │
```

### Event Types

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      WEBSOCKET EVENTS                                   │
├─────────────────────────────────────────────────────────────────────────┤
│  Event Type      │  Direction    │  Description                        │
├──────────────────┼───────────────┼─────────────────────────────────────┤
│  metrics         │  Server→Client│  Real-time service metrics          │
│  alert           │  Server→Client│  New alert notification             │
│  alert_resolved  │  Server→Client│  Alert has been resolved            │
│  service_status  │  Server→Client│  Service health status change       │
│  ping            │  Client→Server│  Keep-alive ping                    │
│  pong            │  Server→Client│  Keep-alive response                │
└─────────────────────────────────────────────────────────────────────────┘
```

### Metrics Event

```json
{
  "type": "metrics",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "services": [
      {
        "name": "auth-service",
        "cpu": 45.2,
        "memory": 128.5,
        "latency": 15.3,
        "errorRate": 0.1
      }
    ]
  }
}
```

### Alert Event

```json
{
  "type": "alert",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "id": "alert_001",
    "severity": "critical",
    "service": "payments-service",
    "metric": "errorRate",
    "value": 15.5,
    "threshold": 10.0,
    "message": "Error rate exceeded threshold"
  }
}
```

### JavaScript Client Example

```javascript
const ws = new WebSocket('ws://localhost:8007/ws');

ws.onopen = () => {
  console.log('Connected to EventFlow');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  
  switch (message.type) {
    case 'metrics':
      updateDashboardMetrics(message.data);
      break;
    case 'alert':
      showAlertNotification(message.data);
      break;
    case 'alert_resolved':
      removeAlert(message.data.id);
      break;
  }
};

ws.onclose = () => {
  console.log('Disconnected');
  // Implement reconnection logic
};
```

---

## Error Handling

### Error Response Format

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message",
    "details": {}
  },
  "timestamp": "2024-01-15T10:30:00Z",
  "requestId": "req_abc123"
}
```

### Error Codes

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         ERROR CODES                                     │
├─────────────────────────────────────────────────────────────────────────┤
│  HTTP Code │  Error Code          │  Description                       │
├────────────┼──────────────────────┼────────────────────────────────────┤
│  400       │  BAD_REQUEST         │  Invalid request format            │
│  401       │  UNAUTHORIZED        │  Missing or invalid auth token     │
│  403       │  FORBIDDEN           │  Insufficient permissions          │
│  404       │  NOT_FOUND           │  Resource not found                │
│  409       │  CONFLICT            │  Resource already exists           │
│  422       │  VALIDATION_ERROR    │  Request validation failed         │
│  429       │  RATE_LIMITED        │  Too many requests                 │
│  500       │  INTERNAL_ERROR      │  Server error                      │
│  502       │  BAD_GATEWAY         │  Upstream service error            │
│  503       │  SERVICE_UNAVAILABLE │  Service temporarily unavailable   │
└─────────────────────────────────────────────────────────────────────────┘
```

### Rate Limiting

```
┌─────────────────────────────────────────────────────────────┐
│                    RATE LIMITS                              │
├─────────────────────────────────────────────────────────────┤
│  Endpoint Type    │  Limit          │  Window               │
├───────────────────┼─────────────────┼───────────────────────┤
│  Authentication   │  10 requests    │  per minute           │
│  Read Operations  │  100 requests   │  per minute           │
│  Write Operations │  30 requests    │  per minute           │
│  WebSocket        │  1 connection   │  per client           │
└─────────────────────────────────────────────────────────────┘

Headers returned:
  X-RateLimit-Limit: 100
  X-RateLimit-Remaining: 95
  X-RateLimit-Reset: 1705315800
```

---

## Related Documentation

- [Architecture Overview](./ARCHITECTURE.md)
- [Deployment Guide](./DEPLOYMENT.md)
- [Load Simulation](./LOAD_SIMULATION.md)
- [Troubleshooting](./TROUBLESHOOTING.md)
