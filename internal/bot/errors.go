package bot

import (
	"errors"
	"fmt"
)

// UserError represents an error that should be shown to the user.
// The message is safe to display directly.
type UserError struct {
	Message string // User-friendly message to display
	Cause   error  // Original error for logging (optional)
}

func (e *UserError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *UserError) Unwrap() error {
	return e.Cause
}

// NewUserError creates a new user-facing error with an optional underlying cause.
func NewUserError(message string, cause error) *UserError {
	return &UserError{
		Message: message,
		Cause:   cause,
	}
}

// UserErrorf creates a new user-facing error with a formatted message.
func UserErrorf(format string, args ...any) *UserError {
	return &UserError{
		Message: fmt.Sprintf(format, args...),
	}
}

// WrapUserError wraps an internal error with a user-friendly message.
// Use this when you want to log the original error but show a different message to users.
func WrapUserError(message string, cause error) *UserError {
	return &UserError{
		Message: message,
		Cause:   cause,
	}
}

// IsUserError checks if the given error is a UserError.
func IsUserError(err error) bool {
	var userErr *UserError
	return errors.As(err, &userErr)
}

// GetUserMessage extracts the user-friendly message from an error.
// If the error is a UserError, returns its Message.
// Otherwise, returns a generic internal error message.
func GetUserMessage(err error) string {
	var userErr *UserError
	if errors.As(err, &userErr) {
		return userErr.Message
	}
	return MsgInternalError
}

// GetLogError extracts the error that should be logged.
// If the error is a UserError with a Cause, returns the full error chain.
// Otherwise, returns the original error.
func GetLogError(err error) error {
	var userErr *UserError
	if errors.As(err, &userErr) && userErr.Cause != nil {
		return err // Return the full UserError which includes both message and cause
	}
	return err
}

// ShouldLog returns true if the error should be logged.
// UserErrors without a cause are user mistakes and don't need logging.
func ShouldLog(err error) bool {
	var userErr *UserError
	if errors.As(err, &userErr) {
		return userErr.Cause != nil
	}
	return true // Always log non-user errors
}
