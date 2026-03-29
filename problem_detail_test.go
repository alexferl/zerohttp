package zerohttp

import (
	"net/http"
	"testing"

	"github.com/alexferl/zerohttp/internal/problem"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestNewProblemDetail(t *testing.T) {
	t.Run("creates problem detail with status and detail", func(t *testing.T) {
		pd := NewProblemDetail(http.StatusNotFound, "Resource not found")

		zhtest.AssertEqual(t, http.StatusNotFound, pd.Status)
		zhtest.AssertEqual(t, "Not Found", pd.Title)
		zhtest.AssertEqual(t, "Resource not found", pd.Detail)
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
			zhtest.AssertEqual(t, tt.status, pd.Status)
			zhtest.AssertEqual(t, tt.want, pd.Title)
		}
	})

	t.Run("problem detail has extensions map", func(t *testing.T) {
		pd := NewProblemDetail(http.StatusBadRequest, "Bad request")

		zhtest.AssertNotNil(t, pd.Extensions)
	})
}

func TestNewValidationProblemDetail(t *testing.T) {
	t.Run("creates validation problem detail with errors", func(t *testing.T) {
		errors := []problem.ValidationError{
			{Detail: "Field is required", Field: "name"},
			{Detail: "Invalid email", Field: "email"},
		}

		pd := NewValidationProblemDetail("Validation failed", errors)

		zhtest.AssertEqual(t, http.StatusUnprocessableEntity, pd.Status)
		zhtest.AssertEqual(t, "Unprocessable Entity", pd.Title)
		zhtest.AssertEqual(t, "Validation failed", pd.Detail)
	})

	t.Run("has errors extension", func(t *testing.T) {
		errors := []problem.ValidationError{
			{Detail: "Field is required", Field: "name"},
		}

		pd := NewValidationProblemDetail("Validation failed", errors)

		zhtest.AssertNotNil(t, pd.Extensions)

		extensions, ok := pd.Extensions["errors"]
		zhtest.AssertTrue(t, ok)
		zhtest.AssertNotNil(t, extensions)
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

		zhtest.AssertEqual(t, http.StatusUnprocessableEntity, pd.Status)
	})
}
