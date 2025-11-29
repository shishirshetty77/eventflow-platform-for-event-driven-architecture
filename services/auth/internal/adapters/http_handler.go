// Package adapters provides HTTP handlers for the auth service.
package adapters

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/microservices-platform/pkg/shared/logging"
	"github.com/microservices-platform/pkg/shared/metrics"
	"github.com/microservices-platform/pkg/shared/models"
	"github.com/microservices-platform/pkg/shared/utils"
	"github.com/microservices-platform/services/auth/internal/ports"
)

// HTTPHandler handles HTTP requests for the auth service.
type HTTPHandler struct {
	authService ports.AuthService
	logger      *logging.Logger
	metrics     *metrics.Metrics
}

// NewHTTPHandler creates a new HTTPHandler.
func NewHTTPHandler(authService ports.AuthService, logger *logging.Logger, m *metrics.Metrics) *HTTPHandler {
	return &HTTPHandler{
		authService: authService,
		logger:      logger,
		metrics:     m,
	}
}

// Router returns the HTTP router with all routes registered.
func (h *HTTPHandler) Router() http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/ready", h.handleReady)

	// Auth endpoints
	mux.HandleFunc("/api/v1/auth/login", h.handleLogin)
	mux.HandleFunc("/api/v1/auth/register", h.handleRegister)
	mux.HandleFunc("/api/v1/auth/validate", h.handleValidate)
	mux.HandleFunc("/api/v1/auth/refresh", h.handleRefresh)

	// Wrap with middleware
	return h.withMiddleware(mux)
}

// withMiddleware applies common middleware to all routes.
func (h *HTTPHandler) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Add request ID to context
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = utils.GenerateRequestID()
		}

		// Set response headers
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("Content-Type", "application/json")

		// Create response wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call next handler
		next.ServeHTTP(rw, r)

		// Record metrics
		duration := time.Since(start)
		h.metrics.RecordHTTPRequest(r.Method, r.URL.Path, http.StatusText(rw.statusCode), duration)

		// Log request
		h.logger.Info("http request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", rw.statusCode),
			zap.Duration("duration", duration),
			zap.String("request_id", requestID),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// handleHealth handles health check requests.
func (h *HTTPHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	h.writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"status":    "healthy",
			"service":   "auth",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// handleReady handles readiness check requests.
func (h *HTTPHandler) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	h.writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"ready": true,
		},
	})
}

// handleLogin handles login requests.
func (h *HTTPHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	response, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		status := utils.GetHTTPStatus(err)
		h.writeError(w, status, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    response,
	})
}

// handleRegister handles registration requests.
func (h *HTTPHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.authService.Register(r.Context(), req.Email, req.Password, req.Name, req.Role)
	if err != nil {
		status := utils.GetHTTPStatus(err)
		h.writeError(w, status, err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Data:    user,
	})
}

// handleValidate handles token validation requests.
func (h *HTTPHandler) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		h.writeError(w, http.StatusUnauthorized, "missing authorization header")
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		h.writeError(w, http.StatusUnauthorized, "invalid authorization header format")
		return
	}

	claims, err := h.authService.ValidateToken(r.Context(), token)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    claims,
	})
}

// handleRefresh handles token refresh requests.
func (h *HTTPHandler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		h.writeError(w, http.StatusUnauthorized, "missing authorization header")
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		h.writeError(w, http.StatusUnauthorized, "invalid authorization header format")
		return
	}

	response, err := h.authService.RefreshToken(r.Context(), token)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    response,
	})
}

// writeJSON writes a JSON response.
func (h *HTTPHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
	}
}

// writeError writes an error response.
func (h *HTTPHandler) writeError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	response := models.APIResponse{
		Success: false,
		Error: &models.APIError{
			Code:    http.StatusText(status),
			Message: message,
		},
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode error response", zap.Error(err))
	}
}
