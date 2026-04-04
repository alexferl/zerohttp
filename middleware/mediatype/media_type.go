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

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !mwutil.ShouldProcessMiddleware(r.URL.Path, c.IncludedPaths, c.ExcludedPaths) {
				next.ServeHTTP(w, r)
				return
			}

			accept := r.Header.Get(httpx.HeaderAccept)
			if accept != "" && accept != "*/*" {
				if !matchAcceptHeader(accept, allowedPatterns) {
					detail := problem.NewDetail(http.StatusNotAcceptable, "Not Acceptable")
					detail.Detail = "The requested media type is not supported"
					w.Header().Set(httpx.HeaderAccept, strings.Join(c.AllowedTypes, ", "))
					_ = detail.RenderAuto(w, r)
					return
				}
			}

			if c.ValidateContentType && r.ContentLength > 0 {
				contentType := r.Header.Get(httpx.HeaderContentType)
				contentType, _, _ = strings.Cut(contentType, ";")
				contentType = strings.TrimSpace(strings.ToLower(contentType))

				if contentType != "" && !matchMediaType(contentType, allowedPatterns) {
					detail := problem.NewDetail(http.StatusUnsupportedMediaType, "Unsupported Media Type")
					detail.Detail = "The request body content type is not supported"
					w.Header().Set(httpx.HeaderAccept, strings.Join(c.AllowedTypes, ", "))
					_ = detail.RenderAuto(w, r)
					return
				}
			}

			// Wrap response writer to set default content type if needed
			if c.DefaultType != "" {
				w = &responseWriter{
					ResponseWriter: w,
					defaultType:    c.DefaultType,
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter wraps http.ResponseWriter to set a default Content-Type
type responseWriter struct {
	http.ResponseWriter
	defaultType string
	wroteHeader bool
}

func (w *responseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		if w.Header().Get(httpx.HeaderContentType) == "" {
			w.Header().Set(httpx.HeaderContentType, w.defaultType)
		}
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// normalizePatterns prepares patterns for matching
type pattern struct {
	mediaType   string
	suffix      string
	hasWildcard bool
}

func normalizePatterns(patterns []string) []pattern {
	result := make([]pattern, 0, len(patterns))
	for _, p := range patterns {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}

		mediaType, suffix, hasSuffix := strings.Cut(p, "+")
		if !hasSuffix {
			suffix = ""
			mediaType = p
		}

		result = append(result, pattern{
			mediaType:   mediaType,
			suffix:      suffix,
			hasWildcard: strings.Contains(p, "*"),
		})
	}
	return result
}

// matchAcceptHeader checks if any of the accepted types match allowed patterns
func matchAcceptHeader(accept string, allowed []pattern) bool {
	// Parse Accept header (can contain multiple types with q-values)
	// e.g., "application/json, application/xml;q=0.9, */*;q=0.1"
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
func matchMediaType(mediaType string, allowed []pattern) bool {
	// Extract suffix from media type if present
	baseType, suffix, hasSuffix := strings.Cut(mediaType, "+")
	if !hasSuffix {
		suffix = ""
	}

	for _, p := range allowed {
		if matchPattern(baseType, suffix, p) {
			return true
		}
	}
	return false
}

// matchPattern checks if a media type matches a specific pattern
func matchPattern(baseType, suffix string, p pattern) bool {
	// If pattern has a suffix requirement, check it
	if p.suffix != "" {
		// If media type has no suffix, check if the last part of baseType matches the suffix
		// e.g., "application/json" with suffix "json" should match "*+json"
		if suffix == "" {
			// Check if baseType ends with "/" + p.suffix (e.g., "/json")
			if strings.HasSuffix(baseType, "/"+p.suffix) {
				// Now match the base type part before the suffix
				baseWithoutSuffix := baseType[:len(baseType)-len(p.suffix)-1]
				if p.hasWildcard {
					return matchWildcard(baseWithoutSuffix, p.mediaType)
				}
				return baseWithoutSuffix == p.mediaType
			}
			// Also try matching full type against pattern (for cases like "json" matching "*+json")
			if p.hasWildcard {
				fullPattern := p.mediaType + "+" + p.suffix
				if matchWildcard(baseType, fullPattern) {
					return true
				}
			}
			return false
		}
		if suffix != p.suffix {
			return false
		}
	}

	// Match the base type (with wildcard support)
	if p.hasWildcard {
		return matchWildcard(baseType, p.mediaType)
	}
	return baseType == p.mediaType
}

// matchWildcard matches a media type against a pattern with wildcards
func matchWildcard(mediaType, pattern string) bool {
	// Simple wildcard matching - * matches any sequence
	parts := strings.Split(pattern, "*")

	// No wildcards (shouldn't happen due to hasWildcard check)
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
