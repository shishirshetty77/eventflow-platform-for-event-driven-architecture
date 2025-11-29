// Package models provides shared data models and DTOs for the microservices platform.
package models

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

// Validator is the global validator instance.
var validate *validator.Validate

func init() {
	validate = validator.New()
}

// Validate validates a struct using the validator tags.
func Validate(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return &ValidationError{Errors: validationErrors}
		}
		return fmt.Errorf("validation error: %w", err)
	}
	return nil
}

// ValidationError wraps validator.ValidationErrors for better error handling.
type ValidationError struct {
	Errors validator.ValidationErrors
}

// Error implements the error interface.
func (v *ValidationError) Error() string {
	if len(v.Errors) == 0 {
		return "validation error"
	}
	return fmt.Sprintf("validation failed: %s", v.Errors[0].Error())
}

// ValidationErrorDetail represents a single validation error detail.
type ValidationErrorDetail struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// GetDetails returns detailed information about each validation error.
func (v *ValidationError) GetDetails() []ValidationErrorDetail {
	details := make([]ValidationErrorDetail, 0, len(v.Errors))
	for _, err := range v.Errors {
		details = append(details, ValidationErrorDetail{
			Field:   err.Field(),
			Tag:     err.Tag(),
			Value:   fmt.Sprintf("%v", err.Value()),
			Message: formatValidationMessage(err),
		})
	}
	return details
}

// formatValidationMessage creates a human-readable validation message.
func formatValidationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", err.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", err.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s", err.Field(), err.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", err.Field(), err.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", err.Field(), err.Param())
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", err.Field())
	default:
		return fmt.Sprintf("%s failed validation: %s", err.Field(), err.Tag())
	}
}
