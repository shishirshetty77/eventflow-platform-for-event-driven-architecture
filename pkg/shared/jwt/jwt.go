// Package jwt provides JWT token utilities.
package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/microservices-platform/pkg/shared/models"
)

// Config holds JWT configuration.
type Config struct {
	SecretKey     string        `json:"secret_key"`
	Issuer        string        `json:"issuer"`
	TokenExpiry   time.Duration `json:"token_expiry"`
	RefreshExpiry time.Duration `json:"refresh_expiry"`
}

// DefaultConfig returns default JWT configuration.
func DefaultConfig() *Config {
	return &Config{
		SecretKey:     "your-secret-key-change-in-production",
		Issuer:        "microservices-platform",
		TokenExpiry:   24 * time.Hour,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
}

// Manager handles JWT operations.
type Manager struct {
	config *Config
}

// NewManager creates a new JWT manager.
func NewManager(cfg *Config) *Manager {
	return &Manager{config: cfg}
}

// Claims represents JWT claims.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken generates a new JWT token for a user.
func (m *Manager) GenerateToken(user *models.User) (string, int64, error) {
	expiresAt := time.Now().Add(m.config.TokenExpiry)

	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    m.config.Issuer,
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(m.config.SecretKey))
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, expiresAt.Unix(), nil
}

// ValidateToken validates a JWT token and returns the claims.
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.SecretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// RefreshToken generates a new token from a valid token.
func (m *Manager) RefreshToken(tokenString string) (string, int64, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return "", 0, err
	}

	// Create a new user from claims to generate new token
	user := &models.User{
		ID:    claims.UserID,
		Email: claims.Email,
		Role:  claims.Role,
	}

	return m.GenerateToken(user)
}

// GetTokenClaims extracts claims from a token without full validation.
// Useful for getting user info from expired tokens.
func (m *Manager) GetTokenClaims(tokenString string) (*models.TokenClaims, error) {
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return &models.TokenClaims{
		UserID: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
	}, nil
}
