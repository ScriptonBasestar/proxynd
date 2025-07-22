package configs

import (
	"fmt"
	"log"
	"strings"

	"github.com/go-playground/validator/v10"
)

// validator is the shared validator instance
var validate *validator.Validate

func init() {
	var err error
	validate, err = createValidator()
	if err != nil {
		log.Fatalf("Failed to initialize validator: %v", err)
	}
}

// createValidator creates and configures a new validator instance
func createValidator() (*validator.Validate, error) {
	v := validator.New()

	// Register custom validators
	if err := v.RegisterValidation("duration", validateDuration); err != nil {
		return nil, fmt.Errorf("failed to register duration validator: %w", err)
	}
	if err := v.RegisterValidation("url", validateURL); err != nil {
		return nil, fmt.Errorf("failed to register url validator: %w", err)
	}
	if err := v.RegisterValidation("path", validatePath); err != nil {
		return nil, fmt.Errorf("failed to register path validator: %w", err)
	}
	if err := v.RegisterValidation("port", validatePort); err != nil {
		return nil, fmt.Errorf("failed to register port validator: %w", err)
	}

	return v, nil
}

// ValidateStruct validates a struct using struct tags
func ValidateStruct(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		return formatValidationError(err)
	}
	return nil
}

// formatValidationError formats validation errors into readable messages
func formatValidationError(err error) error {
	if err == nil {
		return nil
	}

	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	var messages []string
	for _, e := range validationErrs {
		field := e.Field()
		tag := e.Tag()
		param := e.Param()

		switch tag {
		case "required":
			messages = append(messages, fmt.Sprintf("%s is required", field))
		case "min":
			messages = append(messages, fmt.Sprintf("%s must be at least %s", field, param))
		case "max":
			messages = append(messages, fmt.Sprintf("%s must be at most %s", field, param))
		case "url":
			messages = append(messages, fmt.Sprintf("%s must be a valid URL", field))
		case "duration":
			messages = append(messages, fmt.Sprintf("%s must be a valid duration", field))
		case "path":
			messages = append(messages, fmt.Sprintf("%s must be a valid path", field))
		case "port":
			messages = append(messages, fmt.Sprintf("%s must be a valid port (1-65535)", field))
		case "oneof":
			messages = append(messages, fmt.Sprintf("%s must be one of: %s", field, param))
		default:
			messages = append(messages, fmt.Sprintf("%s failed validation: %s", field, tag))
		}
	}

	return fmt.Errorf("validation failed: %s", strings.Join(messages, "; "))
}

// Custom validators

// validateDuration validates duration strings
func validateDuration(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Allow empty, use required tag if needed
	}

	// Simple duration validation - check for common patterns
	// Format: number + unit (s, m, h, d)
	if len(value) < 2 {
		return false
	}

	// Check if it ends with valid unit
	validUnits := []string{"ns", "us", "µs", "ms", "s", "m", "h", "d"}
	for _, unit := range validUnits {
		if strings.HasSuffix(value, unit) {
			// Check if the rest is a number
			numPart := strings.TrimSuffix(value, unit)
			if numPart == "" {
				return false
			}
			// Simple numeric check
			for _, ch := range numPart {
				if ch < '0' || ch > '9' {
					if ch != '.' {
						return false
					}
				}
			}
			return true
		}
	}

	return false
}

// validateURL validates URL strings
func validateURL(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Allow empty, use required tag if needed
	}

	// Basic URL validation
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		return false
	}

	// Check for basic structure
	parts := strings.Split(value, "://")
	if len(parts) != 2 {
		return false
	}

	// Check if there's something after the protocol
	if len(parts[1]) < 3 { // at least "a.b"
		return false
	}

	return true
}

// validatePath validates file system paths
func validatePath(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Allow empty, use required tag if needed
	}

	// Path should not contain null bytes
	if strings.Contains(value, "\x00") {
		return false
	}

	// Basic path validation - just ensure it's not obviously invalid
	// More complex validation would require OS-specific checks
	return true
}

// validatePort validates port numbers
func validatePort(fl validator.FieldLevel) bool {
	value := fl.Field().Int()
	return value >= 1 && value <= 65535
}

// ValidationError represents a validation error with detailed information
type ValidationError struct {
	Field   string
	Message string
	Value   interface{}
}

// Error performs error operation
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s (value: %v)",
		e.Field, e.Message, e.Value)
}

// Validatable interface for configs that implement their own validation
type Validatable interface {
	Validate() error
}
