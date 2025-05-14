package domain

import (
	"errors"
	"fmt"
)

// Standard domain errors - these are sentinel error values that can be compared directly
var (
	// ErrUserNotFound indicates that the requested user email was not found in the database
	ErrUserNotFound = errors.New("user email not found in database")

	// ErrDatabaseUnavailable indicates a temporary issue with accessing the database
	ErrDatabaseUnavailable = errors.New("database is temporarily unavailable, try later")

	// ErrInvalidEmail indicates the provided email has invalid format
	ErrInvalidEmail = errors.New("invalid email format")

	// ErrRateLimitExceeded indicates a user has made too many requests
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrAlreadyRedeemed indicates a user has already redeemed their cocktail
	ErrAlreadyRedeemed = errors.New("cocktail already redeemed")

	// ErrInvalidCredentials indicates invalid authentication credentials
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrInternalServer indicates a generic internal server error
	ErrInternalServer = errors.New("internal server error")
)

// DatabaseError provides additional context for database related errors
type DatabaseError struct {
	OrigErr  error  // Original error from database driver
	Op       string // Operation being performed
	Database string // Database system being used
	Message  string // Human-readable message
}

// Error implements the error interface
func (e *DatabaseError) Error() string {
	if e.OrigErr != nil {
		return fmt.Sprintf("%s: %s (db: %s, op: %s)", e.Message, e.OrigErr.Error(), e.Database, e.Op)
	}
	return fmt.Sprintf("%s (db: %s, op: %s)", e.Message, e.Database, e.Op)
}

// Unwrap returns the original error for compatibility with errors.Is/As
func (e *DatabaseError) Unwrap() error {
	return e.OrigErr
}

// NewDatabaseError creates a new database error
func NewDatabaseError(origErr error, database, op, msg string) *DatabaseError {
	return &DatabaseError{
		OrigErr:  origErr,
		Database: database,
		Op:       op,
		Message:  msg,
	}
}

// ValidationError represents errors related to input validation
type ValidationError struct {
	Field   string // Field that failed validation
	Value   string // Value that was invalid (may be omitted for privacy)
	Message string // Human-readable error message
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("validation failed for %s: %s (value: %s)", e.Field, e.Message, e.Value)
	}
	return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string, value ...string) *ValidationError {
	e := &ValidationError{
		Field:   field,
		Message: message,
	}
	if len(value) > 0 {
		e.Value = value[0]
	}
	return e
}

// IsNotFound returns true if the error indicates a not-found condition
func IsNotFound(err error) bool {
	return errors.Is(err, ErrUserNotFound)
}

// IsUnavailable returns true if the error indicates the database is unavailable
func IsUnavailable(err error) bool {
	return errors.Is(err, ErrDatabaseUnavailable)
}

// IsRateLimited returns true if the error indicates rate limiting
func IsRateLimited(err error) bool {
	return errors.Is(err, ErrRateLimitExceeded)
}

// IsValidationError returns true if the error is a validation error
func IsValidationError(err error) bool {
	var validErr *ValidationError
	return errors.As(err, &validErr)
}

// IsDatabaseError returns true if the error is a database error
func IsDatabaseError(err error) bool {
	var dbErr *DatabaseError
	return errors.As(err, &dbErr)
}

// IsAlreadyRedeemed returns true if the error indicates already redeemed status
func IsAlreadyRedeemed(err error) bool {
	return errors.Is(err, ErrAlreadyRedeemed)
}