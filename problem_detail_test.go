package zerohttp

import (
	"net/http"
	"testing"

	"github.com/alexferl/zerohttp/internal/problem"
)

func TestNewProblemDetail(t *testing.T) {
	t.Run("creates problem detail with status and detail", func(t *testing.T) {
		pd := NewProblemDetail(http.StatusNotFound, "Resource not found")

		if pd.Status != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, pd.Status)
		}
		if pd.Title != "Not Found" {
			t.Errorf("expected title 'Not Found', got '%s'", pd.Title)
		}
		if pd.Detail != "Resource not found" {
			t.Errorf("expected detail 'Resource not found', got '%s'", pd.Detail)
		}
	})

	t.Run("creates problem detail with different status codes", func(t *testing.T) {
		tests := []struct {
			status int
			want   string
		}{
			{http.StatusBadRequest, "Bad Request"},
			{http.StatusUnauthorized, "Unauthorized"},
			{http.StatusForbidden, "Forbidden"},
			{http.StatusInternalServerError, "Internal Server Error"},
			{http.StatusServiceUnavailable, "Service Unavailable"},
		}

		for _, tt := range tests {
			pd := NewProblemDetail(tt.status, "test detail")
			if pd.Status != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, pd.Status)
			}
			if pd.Title != tt.want {
				t.Errorf("expected title '%s', got '%s'", tt.want, pd.Title)
			}
		}
	})

	t.Run("problem detail has extensions map", func(t *testing.T) {
		pd := NewProblemDetail(http.StatusBadRequest, "Bad request")

		if pd.Extensions == nil {
			t.Error("expected Extensions to be initialized")
		}
	})
}

func TestNewValidationProblemDetail(t *testing.T) {
	t.Run("creates validation problem detail with errors", func(t *testing.T) {
		errors := []problem.ValidationError{
			{Detail: "Field is required", Field: "name"},
			{Detail: "Invalid email", Field: "email"},
		}

		pd := NewValidationProblemDetail("Validation failed", errors)

		if pd.Status != http.StatusUnprocessableEntity {
			t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, pd.Status)
		}
		if pd.Title != "Unprocessable Entity" {
			t.Errorf("expected title 'Unprocessable Entity', got '%s'", pd.Title)
		}
		if pd.Detail != "Validation failed" {
			t.Errorf("expected detail 'Validation failed', got '%s'", pd.Detail)
		}
	})

	t.Run("has errors extension", func(t *testing.T) {
		errors := []problem.ValidationError{
			{Detail: "Field is required", Field: "name"},
		}

		pd := NewValidationProblemDetail("Validation failed", errors)

		if pd.Extensions == nil {
			t.Fatal("expected Extensions to be initialized")
		}

		extensions, ok := pd.Extensions["errors"]
		if !ok {
			t.Error("expected 'errors' key in Extensions")
		}

		if extensions == nil {
			t.Error("expected errors to not be nil")
		}
	})

	t.Run("works with custom error types", func(t *testing.T) {
		type CustomError struct {
			Field   string `json:"field"`
			Message string `json:"message"`
			Code    int    `json:"code"`
		}

		errors := []CustomError{
			{Field: "username", Message: "Too short", Code: 1001},
			{Field: "password", Message: "Too weak", Code: 1002},
		}

		pd := NewValidationProblemDetail("Custom validation failed", errors)

		if pd.Status != http.StatusUnprocessableEntity {
			t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, pd.Status)
		}
	})
}
