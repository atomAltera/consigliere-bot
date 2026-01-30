package bot

import (
	"errors"
	"testing"
)

func TestUserError(t *testing.T) {
	t.Run("without cause", func(t *testing.T) {
		err := UserErrorf("user did something wrong")
		if err.Message != "user did something wrong" {
			t.Errorf("expected message 'user did something wrong', got '%s'", err.Message)
		}
		if err.Cause != nil {
			t.Error("expected no cause")
		}
		if err.Error() != "user did something wrong" {
			t.Errorf("expected Error() to return message, got '%s'", err.Error())
		}
	})

	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("database connection failed")
		err := WrapUserError("Failed to save", cause)
		if err.Message != "Failed to save" {
			t.Errorf("expected message 'Failed to save', got '%s'", err.Message)
		}
		if err.Cause != cause {
			t.Error("expected cause to be set")
		}
		if err.Error() != "Failed to save: database connection failed" {
			t.Errorf("unexpected Error() result: %s", err.Error())
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		cause := errors.New("original error")
		err := WrapUserError("wrapper", cause)
		if !errors.Is(err, cause) {
			t.Error("expected errors.Is to match cause")
		}
	})
}

func TestIsUserError(t *testing.T) {
	t.Run("with UserError", func(t *testing.T) {
		err := UserErrorf("test")
		if !IsUserError(err) {
			t.Error("expected IsUserError to return true")
		}
	})

	t.Run("with regular error", func(t *testing.T) {
		err := errors.New("regular error")
		if IsUserError(err) {
			t.Error("expected IsUserError to return false")
		}
	})
}

func TestGetUserMessage(t *testing.T) {
	t.Run("with UserError", func(t *testing.T) {
		err := UserErrorf("friendly message")
		msg := GetUserMessage(err)
		if msg != "friendly message" {
			t.Errorf("expected 'friendly message', got '%s'", msg)
		}
	})

	t.Run("with wrapped UserError", func(t *testing.T) {
		err := WrapUserError("friendly message", errors.New("internal"))
		msg := GetUserMessage(err)
		if msg != "friendly message" {
			t.Errorf("expected 'friendly message', got '%s'", msg)
		}
	})

	t.Run("with regular error", func(t *testing.T) {
		err := errors.New("database error")
		msg := GetUserMessage(err)
		if msg != "An internal error occurred. Please try again later." {
			t.Errorf("expected generic message, got '%s'", msg)
		}
	})
}

func TestShouldLog(t *testing.T) {
	t.Run("UserError without cause - don't log", func(t *testing.T) {
		err := UserErrorf("user mistake")
		if ShouldLog(err) {
			t.Error("expected ShouldLog to return false for UserError without cause")
		}
	})

	t.Run("UserError with cause - log", func(t *testing.T) {
		err := WrapUserError("failed", errors.New("db error"))
		if !ShouldLog(err) {
			t.Error("expected ShouldLog to return true for UserError with cause")
		}
	})

	t.Run("regular error - log", func(t *testing.T) {
		err := errors.New("some error")
		if !ShouldLog(err) {
			t.Error("expected ShouldLog to return true for regular error")
		}
	})
}
