// Package utils provides shared utility functions.
package utils

import (
	"fmt"
)

// Error codes for the microservices platform.
const (
	ErrCodeValidation   = "VALIDATION_ERROR"
	ErrCodeNotFound     = "NOT_FOUND"
	ErrCodeUnauthorized = "UNAUTHORIZED"
	ErrCodeForbidden    = "FORBIDDEN"
	ErrCodeConflict     = "CONFLICT"
	ErrCodeInternal     = "INTERNAL_ERROR"
	ErrCodeBadRequest   = "BAD_REQUEST"
	ErrCodeTimeout      = "TIMEOUT"
	ErrCodeUnavailable  = "SERVICE_UNAVAILABLE"
	ErrCodeRateLimit    = "RATE_LIMIT_EXCEEDED"
	ErrCodeKafka        = "KAFKA_ERROR"
	ErrCodeRedis        = "REDIS_ERROR"
	ErrCodeDatabase     = "DATABASE_ERROR"
	ErrCodeExternalAPI  = "EXTERNAL_API_ERROR"
)

// AppError represents an application error with code and context.
type AppError struct {
	Code    string
	Message string
	Err     error
	Details map[string]interface{}
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithDetails adds details to the error.
func (e *AppError) WithDetails(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// NewError creates a new AppError.
func NewError(code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// WrapError wraps an existing error with code and message.
func WrapError(err error, code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Convenience error constructors.

// ErrValidation creates a validation error.
func ErrValidation(message string) *AppError {
	return NewError(ErrCodeValidation, message)
}

// ErrNotFound creates a not found error.
func ErrNotFound(resource string) *AppError {
	return NewError(ErrCodeNotFound, fmt.Sprintf("%s not found", resource))
}

// ErrUnauthorized creates an unauthorized error.
func ErrUnauthorized(message string) *AppError {
	if message == "" {
		message = "unauthorized"
	}
	return NewError(ErrCodeUnauthorized, message)
}

// ErrForbidden creates a forbidden error.
func ErrForbidden(message string) *AppError {
	if message == "" {
		message = "access denied"
	}
	return NewError(ErrCodeForbidden, message)
}

// ErrConflict creates a conflict error.
func ErrConflict(message string) *AppError {
	return NewError(ErrCodeConflict, message)
}

// ErrInternal creates an internal error.
func ErrInternal(message string) *AppError {
	if message == "" {
		message = "internal server error"
	}
	return NewError(ErrCodeInternal, message)
}

// ErrBadRequest creates a bad request error.
func ErrBadRequest(message string) *AppError {
	return NewError(ErrCodeBadRequest, message)
}

// ErrTimeout creates a timeout error.
func ErrTimeout(message string) *AppError {
	if message == "" {
		message = "operation timed out"
	}
	return NewError(ErrCodeTimeout, message)
}

// ErrUnavailable creates a service unavailable error.
func ErrUnavailable(message string) *AppError {
	if message == "" {
		message = "service unavailable"
	}
	return NewError(ErrCodeUnavailable, message)
}

// ErrRateLimit creates a rate limit error.
func ErrRateLimit(message string) *AppError {
	if message == "" {
		message = "rate limit exceeded"
	}
	return NewError(ErrCodeRateLimit, message)
}

// ErrKafka creates a Kafka error.
func ErrKafka(err error, message string) *AppError {
	return WrapError(err, ErrCodeKafka, message)
}

// ErrRedis creates a Redis error.
func ErrRedis(err error, message string) *AppError {
	return WrapError(err, ErrCodeRedis, message)
}

// ErrDatabase creates a database error.
func ErrDatabase(err error, message string) *AppError {
	return WrapError(err, ErrCodeDatabase, message)
}

// ErrExternalAPI creates an external API error.
func ErrExternalAPI(err error, message string) *AppError {
	return WrapError(err, ErrCodeExternalAPI, message)
}

// IsAppError checks if an error is an AppError with the given code.
func IsAppError(err error, code string) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == code
	}
	return false
}

// GetErrorCode extracts the error code from an error.
func GetErrorCode(err error) string {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return ErrCodeInternal
}

// GetHTTPStatus maps error codes to HTTP status codes.
func GetHTTPStatus(err error) int {
	code := GetErrorCode(err)
	switch code {
	case ErrCodeValidation, ErrCodeBadRequest:
		return 400
	case ErrCodeUnauthorized:
		return 401
	case ErrCodeForbidden:
		return 403
	case ErrCodeNotFound:
		return 404
	case ErrCodeConflict:
		return 409
	case ErrCodeRateLimit:
		return 429
	case ErrCodeTimeout:
		return 504
	case ErrCodeUnavailable:
		return 503
	default:
		return 500
	}
}
