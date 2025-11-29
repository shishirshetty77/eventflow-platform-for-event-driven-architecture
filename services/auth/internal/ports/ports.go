// Package ports defines interfaces for the auth service.
package ports

import (
	"context"

	"github.com/microservices-platform/pkg/shared/models"
)

// UserRepository defines the interface for user storage operations.
type UserRepository interface {
	// Create creates a new user.
	Create(ctx context.Context, user *models.User) error
	// GetByID retrieves a user by ID.
	GetByID(ctx context.Context, id string) (*models.User, error)
	// GetByEmail retrieves a user by email.
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	// Update updates a user.
	Update(ctx context.Context, user *models.User) error
	// Delete deletes a user.
	Delete(ctx context.Context, id string) error
	// List lists all users with pagination.
	List(ctx context.Context, page, pageSize int) ([]*models.User, int64, error)
}

// AuthService defines the interface for authentication operations.
type AuthService interface {
	// Login authenticates a user and returns a JWT token.
	Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error)
	// Register registers a new user.
	Register(ctx context.Context, email, password, name, role string) (*models.User, error)
	// ValidateToken validates a JWT token and returns the claims.
	ValidateToken(ctx context.Context, token string) (*models.TokenClaims, error)
	// RefreshToken refreshes a JWT token.
	RefreshToken(ctx context.Context, token string) (*models.LoginResponse, error)
}

// MetricsPublisher defines the interface for publishing metrics.
type MetricsPublisher interface {
	// PublishMetric publishes a metric to Kafka.
	PublishMetric(ctx context.Context, metric *models.ServiceMetric) error
	// PublishLog publishes a log entry to Kafka.
	PublishLog(ctx context.Context, log *models.ServiceLog) error
	// Close closes the publisher.
	Close() error
}

// MetricsGenerator defines the interface for generating metrics.
type MetricsGenerator interface {
	// Start starts the metrics generator.
	Start(ctx context.Context) error
	// Stop stops the metrics generator.
	Stop() error
}
