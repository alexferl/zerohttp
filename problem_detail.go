package zerohttp

import (
	"github.com/alexferl/zerohttp/internal/problem"
)

// ProblemDetail is an alias to problem.Detail for backward compatibility.
// It represents an RFC 9457 Problem Details response.
type ProblemDetail = problem.Detail

// ValidationError is an alias to problem.ValidationError for backward compatibility.
// It represents a single validation error with optional field location information.
type ValidationError = problem.ValidationError

// NewProblemDetail creates a new ProblemDetail with the given status code and detail message.
// This is a convenience wrapper around problem.NewDetail.
func NewProblemDetail(statusCode int, detail string) *ProblemDetail {
	return problem.NewDetail(statusCode, detail)
}

// NewValidationProblemDetail creates a problem detail for validation errors (HTTP 422).
// This is a convenience wrapper around problem.NewValidationDetail.
func NewValidationProblemDetail[T any](detail string, errors []T) *ProblemDetail {
	return problem.NewValidationDetail(detail, errors)
}
