// Package core contains the business logic for the auth service.
package core

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/microservices-platform/pkg/shared/jwt"
	"github.com/microservices-platform/pkg/shared/logging"
	"github.com/microservices-platform/pkg/shared/models"
	"github.com/microservices-platform/pkg/shared/tracing"
	"github.com/microservices-platform/pkg/shared/utils"
	"github.com/microservices-platform/services/auth/internal/ports"
)

// AuthServiceImpl implements the AuthService interface.
type AuthServiceImpl struct {
	userRepo   ports.UserRepository
	jwtManager *jwt.Manager
	logger     *logging.Logger
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(userRepo ports.UserRepository, jwtManager *jwt.Manager, logger *logging.Logger) ports.AuthService {
	return &AuthServiceImpl{
		userRepo:   userRepo,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

// Login authenticates a user and returns a JWT token.
func (s *AuthServiceImpl) Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error) {
	ctx, span := tracing.StartSpanFromContext(ctx, "AuthService.Login")
	defer span.End()

	// Validate request
	if err := models.Validate(req); err != nil {
		s.logger.WithContext(ctx).Warn("login validation failed",
			zap.String("email", req.Email),
			zap.Error(err),
		)
		return nil, utils.ErrValidation(err.Error())
	}

	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if utils.IsAppError(err, utils.ErrCodeNotFound) {
			s.logger.WithContext(ctx).Warn("user not found during login",
				zap.String("email", req.Email),
			)
			return nil, utils.ErrUnauthorized("invalid credentials")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		s.logger.WithContext(ctx).Warn("invalid password during login",
			zap.String("email", req.Email),
		)
		return nil, utils.ErrUnauthorized("invalid credentials")
	}

	// Generate JWT token
	token, expiresAt, err := s.jwtManager.GenerateToken(user)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to generate token",
			zap.String("user_id", user.ID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	s.logger.WithContext(ctx).Info("user logged in successfully",
		zap.String("user_id", user.ID),
		zap.String("email", user.Email),
	)

	// Clear password before returning
	user.Password = ""

	return &models.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	}, nil
}

// Register registers a new user.
func (s *AuthServiceImpl) Register(ctx context.Context, email, password, name, role string) (*models.User, error) {
	ctx, span := tracing.StartSpanFromContext(ctx, "AuthService.Register")
	defer span.End()

	// Validate role
	validRoles := []string{"admin", "operator", "viewer"}
	if !utils.Contains(validRoles, role) {
		return nil, utils.ErrValidation("invalid role, must be one of: admin, operator, viewer")
	}

	// Check if user already exists
	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, utils.ErrConflict("user with this email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	now := time.Now().UTC()
	user := &models.User{
		ID:        uuid.New().String(),
		Email:     email,
		Password:  string(hashedPassword),
		Name:      name,
		Role:      role,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Validate user
	if err := models.Validate(user); err != nil {
		return nil, utils.ErrValidation(err.Error())
	}

	// Save user
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.WithContext(ctx).Info("user registered successfully",
		zap.String("user_id", user.ID),
		zap.String("email", user.Email),
		zap.String("role", user.Role),
	)

	// Clear password before returning
	user.Password = ""

	return user, nil
}

// ValidateToken validates a JWT token and returns the claims.
func (s *AuthServiceImpl) ValidateToken(ctx context.Context, token string) (*models.TokenClaims, error) {
	ctx, span := tracing.StartSpanFromContext(ctx, "AuthService.ValidateToken")
	defer span.End()

	claims, err := s.jwtManager.ValidateToken(token)
	if err != nil {
		s.logger.WithContext(ctx).Warn("token validation failed",
			zap.Error(err),
		)
		return nil, utils.ErrUnauthorized("invalid token")
	}

	return &models.TokenClaims{
		UserID: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
	}, nil
}

// RefreshToken refreshes a JWT token.
func (s *AuthServiceImpl) RefreshToken(ctx context.Context, token string) (*models.LoginResponse, error) {
	ctx, span := tracing.StartSpanFromContext(ctx, "AuthService.RefreshToken")
	defer span.End()

	newToken, expiresAt, err := s.jwtManager.RefreshToken(token)
	if err != nil {
		s.logger.WithContext(ctx).Warn("token refresh failed",
			zap.Error(err),
		)
		return nil, utils.ErrUnauthorized("invalid or expired token")
	}

	// Get user from claims
	claims, _ := s.jwtManager.GetTokenClaims(newToken)
	user, _ := s.userRepo.GetByID(ctx, claims.UserID)
	if user != nil {
		user.Password = ""
	}

	s.logger.WithContext(ctx).Info("token refreshed successfully",
		zap.String("user_id", claims.UserID),
	)

	return &models.LoginResponse{
		Token:     newToken,
		ExpiresAt: expiresAt,
		User:      user,
	}, nil
}
