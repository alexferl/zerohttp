package mediatype

import (
	"net/http"
	"strings"

	"github.com/alexferl/zerohttp/httpx"
	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/internal/mwutil"
	"github.com/alexferl/zerohttp/internal/problem"
)

// New creates a media type middleware with the provided configuration.
// It validates the Accept header against allowed patterns and responds with
// 406 Not Acceptable if no match is found.
// When ValidateContentType is true, it also validates the Content-Type header
// and responds with 415 Unsupported Media Type on mismatch.
func New(cfg ...Config) func(http.Handler) http.Handler {
	c := DefaultConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	mwutil.ValidatePathConfig(c.ExcludedPaths, c.IncludedPaths, "MediaType")

	// If no allowed types configured, skip validation entirely
	if len(c.AllowedTypes) == 0 {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	allowedPatterns := normalizePatterns(c.AllowedTypes)

	// Determine the effective media type at construction time.
	// ResponseTypeValue takes precedence, falls back to DefaultType.
	effectiveType := c.ResponseTypeValue
	if effectiveType == "" {
		effectiveType = c.DefaultType
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !mwutil.ShouldProcessMiddleware(r.URL.Path, c.IncludedPaths, c.ExcludedPaths) {
				next.ServeHTTP(w, r)
				return
			}

			// Set response header early so it's present even in error responses.
			if c.ResponseTypeHeader != "" && effectiveType != "" {
				w.Header().Set(c.ResponseTypeHeader, effectiveType)
			}

			accept := r.Header.Get(httpx.HeaderAccept)
			if accept != "" && accept != "*/*" {
				if !matchAcceptHeader(accept, allowedPatterns) {
					detail := problem.NewDetail(http.StatusNotAcceptable, "Not Acceptable")
					detail.Detail = "The requested media type is not supported"
					// NOTE: Accept header in response is non-standard here but commonly used
					// to indicate what Content-Types the server accepts for the request body.
					w.Header().Set(httpx.HeaderAccept, strings.Join(c.AllowedTypes, ", "))
					_ = detail.RenderAuto(w, r)
					return
				}
			}

			// ValidateContentType check: ContentLength != 0 covers both explicit sizes (> 0)
			// and unknown chunked bodies (-1). Go sets ContentLength = -1 for chunked encoding.
			if c.ValidateContentType && r.ContentLength != 0 && r.Body != nil && r.Body != http.NoBody {
				contentType := r.Header.Get(httpx.HeaderContentType)
				contentType, _, _ = strings.Cut(contentType, ";")
				contentType = strings.TrimSpace(strings.ToLower(contentType))

				if contentType != "" && !matchMediaType(contentType, allowedPatterns) {
					detail := problem.NewDetail(http.StatusUnsupportedMediaType, "Unsupported Media Type")
					detail.Detail = "The request body content type is not supported"
					// NOTE: Accept header in response is non-standard here but commonly used
					// to indicate what Content-Types the server accepts for the request body.
					w.Header().Set(httpx.HeaderAccept, strings.Join(c.AllowedTypes, ", "))
					_ = detail.RenderAuto(w, r)
					return
				}
			}

			// Normalize Accept so downstream handlers always receive a concrete media type.
			if c.DefaultType != "" {
				accept := r.Header.Get(httpx.HeaderAccept)
				if accept == "" || accept == "*/*" {
					r.Header.Set(httpx.HeaderAccept, c.DefaultType)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// normalizePatterns prepares patterns for matching
type pattern struct {
	mediaType            string
	suffix               string
	hasMediaTypeWildcard bool
	hasSuffixWildcard    bool
}

func normalizePatterns(patterns []string) []pattern {
	result := make([]pattern, 0, len(patterns))
	for _, p := range patterns {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}

		mediaType, suffix, _ := strings.Cut(p, "+")

		result = append(result, pattern{
			mediaType:            mediaType,
			suffix:               suffix,
			hasMediaTypeWildcard: strings.Contains(mediaType, "*"),
			hasSuffixWildcard:    strings.Contains(suffix, "*"),
		})
	}
	return result
}

// matchAcceptHeader checks if any of the accepted types match allowed patterns
func matchAcceptHeader(accept string, allowed []pattern) bool {
	// Parse Accept header (can contain multiple types with q-values)
	// e.g., "application/json, application/xml;q=0.9, */*;q=0.1"
	// NOTE: q-values are stripped and not used for priority ordering.
	// This middleware only validates presence of an acceptable type, not preference.
	// If format selection is added in the future, q-values must be respected per RFC 7231 §5.3.
	for _, part := range strings.Split(accept, ",") {
		part = strings.TrimSpace(part)
		// Remove q-value if present
		if idx := strings.Index(part, ";"); idx != -1 {
			part = strings.TrimSpace(part[:idx])
		}

		if part == "" {
			continue
		}

		// */* matches everything
		if part == "*/*" {
			return true
		}

		if matchMediaType(strings.ToLower(part), allowed) {
			return true
		}
	}
	return false
}

// matchMediaType checks if a media type matches any of the allowed patterns
// Supports bidirectional wildcard matching per RFC 7231 §5.3.2:
// - Server pattern wildcards (e.g., application/*) match client types
// - Client wildcard types (e.g., application/*) match server allowed types
func matchMediaType(mediaType string, allowed []pattern) bool {
	// Extract suffix from media type if present
	baseType, suffix, _ := strings.Cut(mediaType, "+")

	for _, p := range allowed {
		if matchPattern(baseType, suffix, p) {
			return true
		}
	}

	// Bidirectional wildcard: client's wildcard covers a server-allowed type
	// e.g., client sends "Accept: application/*" should match "application/json"
	if strings.Contains(mediaType, "*") {
		for _, p := range allowed {
			// If client specified a +suffix, the allowed type must also carry that suffix
			if suffix != "" && p.suffix != suffix {
				continue
			}
			if matchWildcard(strings.ToLower(p.mediaType), baseType) {
				return true
			}
		}
	}

	return false
}

// matchPattern checks if a media type matches a specific pattern
func matchPattern(baseType, suffix string, p pattern) bool {
	// Match suffix (with wildcard support if present)
	if p.hasSuffixWildcard {
		// Pattern like "application/vnd+*" requires input to HAVE a suffix
		if suffix == "" {
			return false
		}
		if !matchWildcard(suffix, p.suffix) {
			return false
		}
	} else if p.suffix != suffix {
		// Both must agree on suffix: either both empty, or both the same value
		// Only matches literal +suffixes per RFC 6838 §4.2.8
		return false
	}

	// Match the base type (with wildcard support)
	if p.hasMediaTypeWildcard {
		return matchWildcard(baseType, p.mediaType)
	}
	return baseType == p.mediaType
}

// matchWildcard matches a media type against a pattern with wildcards.
// NOTE: Only single-wildcard patterns per segment are fully supported.
// Patterns with multiple wildcards in the same segment (e.g., "application/x*x*x")
// use first-match greedy logic and may produce unexpected results.
func matchWildcard(mediaType, pattern string) bool {
	// Simple wildcard matching - * matches any sequence
	parts := strings.Split(pattern, "*")

	// No wildcards (shouldn't happen due to hasMediaTypeWildcard/hasSuffixWildcard check)
	if len(parts) == 1 {
		return mediaType == pattern
	}

	// Check prefix
	if !strings.HasPrefix(mediaType, parts[0]) {
		return false
	}

	// Move past the prefix
	mediaType = mediaType[len(parts[0]):]

	// Match remaining parts
	for i := 1; i < len(parts); i++ {
		part := parts[i]
		if part == "" {
			// Trailing wildcard
			if i == len(parts)-1 {
				return true
			}
			continue
		}

		idx := strings.Index(mediaType, part)
		if idx == -1 {
			return false
		}
		mediaType = mediaType[idx+len(part):]
	}

	// If we've consumed the entire media type, it's a match
	return mediaType == ""
}
