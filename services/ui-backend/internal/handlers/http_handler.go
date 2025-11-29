// Package handlers provides HTTP and WebSocket handlers.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/microservices-platform/pkg/shared/jwt"
	"github.com/microservices-platform/pkg/shared/logging"
	"github.com/microservices-platform/pkg/shared/models"
	"github.com/microservices-platform/services/ui-backend/internal/store"
)

// Handler handles HTTP requests.
type Handler struct {
	store     *store.RedisStore
	jwt       *jwt.TokenService
	logger    *logging.Logger
	validator *validator.Validate
}

// NewHandler creates a new Handler.
func NewHandler(s *store.RedisStore, jwtService *jwt.TokenService, logger *logging.Logger) *Handler {
	return &Handler{
		store:     s,
		jwt:       jwtService,
		logger:    logger,
		validator: validator.New(),
	}
}

// Response represents a generic API response.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Meta contains pagination metadata.
type Meta struct {
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Pages int `json:"pages"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, Response{Success: false, Error: message})
}

// GetServices returns all monitored services.
func (h *Handler) GetServices(w http.ResponseWriter, r *http.Request) {
	services := []map[string]interface{}{
		{"name": "auth", "display_name": "Auth Service", "status": "healthy"},
		{"name": "orders", "display_name": "Orders Service", "status": "healthy"},
		{"name": "payments", "display_name": "Payments Service", "status": "healthy"},
		{"name": "notification", "display_name": "Notification Service", "status": "healthy"},
	}
	writeJSON(w, http.StatusOK, Response{Success: true, Data: services})
}

// GetServiceMetrics returns metrics for a service.
func (h *Handler) GetServiceMetrics(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "service")
	if serviceName == "" {
		writeError(w, http.StatusBadRequest, "service name required")
		return
	}

	windowStr := r.URL.Query().Get("window")
	window := 5 * time.Minute
	if windowStr != "" {
		if d, err := time.ParseDuration(windowStr); err == nil {
			window = d
		}
	}

	ctx := r.Context()
	metrics, err := h.store.GetMetrics(ctx, models.ServiceName(serviceName), window)
	if err != nil {
		h.logger.Error("failed to get metrics", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get metrics")
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: metrics})
}

// GetLatestMetrics returns the latest metric for each service.
func (h *Handler) GetLatestMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	services := []models.ServiceName{
		models.ServiceNameAuth,
		models.ServiceNameOrders,
		models.ServiceNamePayments,
		models.ServiceNameNotification,
	}

	result := make(map[string]*models.ServiceMetric)
	for _, service := range services {
		metric, err := h.store.GetLatestMetric(ctx, service)
		if err != nil {
			h.logger.Warn("failed to get latest metric",
				zap.String("service", string(service)),
				zap.Error(err),
			)
			continue
		}
		if metric != nil {
			result[string(service)] = metric
		}
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: result})
}

// GetAlerts returns alerts.
func (h *Handler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	serviceName := r.URL.Query().Get("service")
	severity := r.URL.Query().Get("severity")

	alerts, total, err := h.store.GetAlerts(ctx, serviceName, severity, page, limit)
	if err != nil {
		h.logger.Error("failed to get alerts", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get alerts")
		return
	}

	pages := total / limit
	if total%limit > 0 {
		pages++
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    alerts,
		Meta: &Meta{
			Total: total,
			Page:  page,
			Limit: limit,
			Pages: pages,
		},
	})
}

// GetAlert returns a single alert.
func (h *Handler) GetAlert(w http.ResponseWriter, r *http.Request) {
	alertID := chi.URLParam(r, "id")
	if alertID == "" {
		writeError(w, http.StatusBadRequest, "alert ID required")
		return
	}

	ctx := r.Context()
	alert, err := h.store.GetAlert(ctx, alertID)
	if err != nil {
		h.logger.Error("failed to get alert", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get alert")
		return
	}

	if alert == nil {
		writeError(w, http.StatusNotFound, "alert not found")
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: alert})
}

// AcknowledgeAlert acknowledges an alert.
func (h *Handler) AcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	alertID := chi.URLParam(r, "id")
	if alertID == "" {
		writeError(w, http.StatusBadRequest, "alert ID required")
		return
	}

	userID, _ := r.Context().Value("user_id").(string)

	ctx := r.Context()
	if err := h.store.AcknowledgeAlert(ctx, alertID, userID); err != nil {
		h.logger.Error("failed to acknowledge alert", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to acknowledge alert")
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true})
}

// GetRules returns threshold rules.
func (h *Handler) GetRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rules, err := h.store.GetRules(ctx)
	if err != nil {
		h.logger.Error("failed to get rules", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get rules")
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: rules})
}

// CreateRuleRequest represents a request to create a rule.
type CreateRuleRequest struct {
	ServiceName string  `json:"service_name" validate:"required"`
	MetricType  string  `json:"metric_type" validate:"required"`
	Threshold   float64 `json:"threshold" validate:"required"`
	Operator    string  `json:"operator" validate:"required"`
	Severity    string  `json:"severity" validate:"required"`
	Enabled     bool    `json:"enabled"`
	Cooldown    int     `json:"cooldown_seconds"`
}

// CreateRule creates a new threshold rule.
func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	rule := &models.ThresholdRule{
		ServiceName:     models.ServiceName(req.ServiceName),
		MetricType:      models.MetricType(req.MetricType),
		Threshold:       req.Threshold,
		Operator:        req.Operator,
		Severity:        models.AlertSeverity(req.Severity),
		Enabled:         req.Enabled,
		CooldownSeconds: req.Cooldown,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	ctx := r.Context()
	if err := h.store.CreateRule(ctx, rule); err != nil {
		h.logger.Error("failed to create rule", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to create rule")
		return
	}

	writeJSON(w, http.StatusCreated, Response{Success: true, Data: rule})
}

// UpdateRule updates a threshold rule.
func (h *Handler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	ruleID := chi.URLParam(r, "id")
	if ruleID == "" {
		writeError(w, http.StatusBadRequest, "rule ID required")
		return
	}

	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rule := &models.ThresholdRule{
		ID:              ruleID,
		ServiceName:     models.ServiceName(req.ServiceName),
		MetricType:      models.MetricType(req.MetricType),
		Threshold:       req.Threshold,
		Operator:        req.Operator,
		Severity:        models.AlertSeverity(req.Severity),
		Enabled:         req.Enabled,
		CooldownSeconds: req.Cooldown,
		UpdatedAt:       time.Now(),
	}

	ctx := r.Context()
	if err := h.store.UpdateRule(ctx, rule); err != nil {
		h.logger.Error("failed to update rule", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to update rule")
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: rule})
}

// DeleteRule deletes a threshold rule.
func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	ruleID := chi.URLParam(r, "id")
	if ruleID == "" {
		writeError(w, http.StatusBadRequest, "rule ID required")
		return
	}

	ctx := r.Context()
	if err := h.store.DeleteRule(ctx, ruleID); err != nil {
		h.logger.Error("failed to delete rule", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to delete rule")
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true})
}

// GetDashboardStats returns dashboard statistics.
func (h *Handler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stats, err := h.store.GetDashboardStats(ctx)
	if err != nil {
		h.logger.Error("failed to get dashboard stats", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get dashboard stats")
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: stats})
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents a login response.
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserInfo  `json:"user"`
}

// UserInfo represents user information.
type UserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// Login handles user authentication.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Simple authentication for demo
	if req.Username != "admin" || req.Password != "admin" {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	user := &models.User{
		ID:       "user-1",
		Username: req.Username,
		Role:     "admin",
	}

	token, err := h.jwt.Generate(user)
	if err != nil {
		h.logger.Error("failed to generate token", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data: LoginResponse{
			Token:     token,
			ExpiresAt: time.Now().Add(24 * time.Hour),
			User: UserInfo{
				ID:       user.ID,
				Username: user.Username,
				Role:     user.Role,
			},
		},
	})
}

// AuthMiddleware validates JWT tokens.
func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenString string

		// First check Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				tokenString = authHeader[7:]
			} else {
				writeError(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}
		}

		// For WebSocket connections, also check query parameter
		if tokenString == "" {
			tokenString = r.URL.Query().Get("token")
		}

		if tokenString == "" {
			writeError(w, http.StatusUnauthorized, "authorization required")
			return
		}

		claims, err := h.jwt.Validate(tokenString)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "username", claims.Username)
		ctx = context.WithValue(ctx, "role", claims.Role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RefreshToken refreshes a JWT token.
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || len(authHeader) < 8 {
		writeError(w, http.StatusUnauthorized, "authorization header required")
		return
	}

	tokenString := authHeader[7:]
	newToken, err := h.jwt.Refresh(tokenString)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "failed to refresh token")
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"token":      newToken,
			"expires_at": time.Now().Add(24 * time.Hour),
		},
	})
}
