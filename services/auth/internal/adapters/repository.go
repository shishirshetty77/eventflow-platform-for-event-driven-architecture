// Package adapters provides implementations of ports interfaces.
package adapters

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/microservices-platform/pkg/shared/models"
	"github.com/microservices-platform/pkg/shared/utils"
	"github.com/microservices-platform/services/auth/internal/ports"
)

// InMemoryUserRepository is an in-memory implementation of UserRepository.
// In production, this would be replaced with a database-backed implementation.
type InMemoryUserRepository struct {
	mu    sync.RWMutex
	users map[string]*models.User
}

// NewInMemoryUserRepository creates a new InMemoryUserRepository with demo users.
func NewInMemoryUserRepository() ports.UserRepository {
	repo := &InMemoryUserRepository{
		users: make(map[string]*models.User),
	}

	// Create demo users
	repo.createDemoUsers()

	return repo
}

// createDemoUsers creates demo users for testing.
func (r *InMemoryUserRepository) createDemoUsers() {
	now := time.Now().UTC()

	// Hash passwords for demo users
	adminPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123!"), bcrypt.DefaultCost)
	operatorPassword, _ := bcrypt.GenerateFromPassword([]byte("operator123!"), bcrypt.DefaultCost)
	viewerPassword, _ := bcrypt.GenerateFromPassword([]byte("viewer123!"), bcrypt.DefaultCost)

	demoUsers := []*models.User{
		{
			ID:        uuid.New().String(),
			Email:     "admin@example.com",
			Password:  string(adminPassword),
			Name:      "Admin User",
			Role:      "admin",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New().String(),
			Email:     "operator@example.com",
			Password:  string(operatorPassword),
			Name:      "Operator User",
			Role:      "operator",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New().String(),
			Email:     "viewer@example.com",
			Password:  string(viewerPassword),
			Name:      "Viewer User",
			Role:      "viewer",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	for _, user := range demoUsers {
		r.users[user.ID] = user
	}
}

// Create creates a new user.
func (r *InMemoryUserRepository) Create(ctx context.Context, user *models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate email
	for _, u := range r.users {
		if u.Email == user.Email {
			return utils.ErrConflict("user with this email already exists")
		}
	}

	r.users[user.ID] = user
	return nil
}

// GetByID retrieves a user by ID.
func (r *InMemoryUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[id]
	if !ok {
		return nil, utils.ErrNotFound("user")
	}

	// Return a copy to prevent mutation
	userCopy := *user
	return &userCopy, nil
}

// GetByEmail retrieves a user by email.
func (r *InMemoryUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.Email == email {
			// Return a copy to prevent mutation
			userCopy := *user
			return &userCopy, nil
		}
	}

	return nil, utils.ErrNotFound("user")
}

// Update updates a user.
func (r *InMemoryUserRepository) Update(ctx context.Context, user *models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[user.ID]; !ok {
		return utils.ErrNotFound("user")
	}

	user.UpdatedAt = time.Now().UTC()
	r.users[user.ID] = user
	return nil
}

// Delete deletes a user.
func (r *InMemoryUserRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[id]; !ok {
		return utils.ErrNotFound("user")
	}

	delete(r.users, id)
	return nil
}

// List lists all users with pagination.
func (r *InMemoryUserRepository) List(ctx context.Context, page, pageSize int) ([]*models.User, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Convert map to slice
	allUsers := make([]*models.User, 0, len(r.users))
	for _, user := range r.users {
		userCopy := *user
		userCopy.Password = "" // Don't expose passwords
		allUsers = append(allUsers, &userCopy)
	}

	total := int64(len(allUsers))

	// Calculate pagination
	start := (page - 1) * pageSize
	if start >= len(allUsers) {
		return []*models.User{}, total, nil
	}

	end := start + pageSize
	if end > len(allUsers) {
		end = len(allUsers)
	}

	return allUsers[start:end], total, nil
}
