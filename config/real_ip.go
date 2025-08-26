package config

import (
	"net"
	"net/http"
	"strings"
)

// IPExtractor defines a function to extract the real client IP.
type IPExtractor func(*http.Request) string

// RealIPConfig allows customization of real IP extraction
type RealIPConfig struct {
	// IPExtractor function to extract real client IP (defaults to DefaultIPExtractor)
	IPExtractor IPExtractor
}

// DefaultRealIPConfig contains the default values for real IP configuration.
var DefaultRealIPConfig = RealIPConfig{
	IPExtractor: DefaultIPExtractor,
}

// DefaultIPExtractor extracts the real client IP from various headers
func DefaultIPExtractor(r *http.Request) string {
	// Check X-Forwarded-For header first (most common)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header (used by Nginx and some proxies)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Check X-Forwarded header (less common)
	if xf := r.Header.Get("X-Forwarded"); xf != "" {
		return xf
	}

	// Check Forwarded header (RFC 7239 standard)
	if forwarded := r.Header.Get("Forwarded"); forwarded != "" {
		// Parse "for=" part from Forwarded header
		parts := strings.Split(forwarded, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "for=") {
				ip := strings.TrimPrefix(part, "for=")
				ip = strings.Trim(ip, `"`)
				return ip
			}
		}
	}

	// Fallback to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// Common IP extractors

// RemoteAddrIPExtractor just uses r.RemoteAddr (no proxy headers)
func RemoteAddrIPExtractor(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// XForwardedForIPExtractor only checks X-Forwarded-For header
func XForwardedForIPExtractor(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	return RemoteAddrIPExtractor(r)
}

// XRealIPExtractor only checks X-Real-IP header (Nginx style)
func XRealIPExtractor(r *http.Request) string {
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return RemoteAddrIPExtractor(r)
}

// RealIPOption configures real IP middleware.
type RealIPOption func(*RealIPConfig)

// WithRealIPExtractor sets the function to extract the real client IP.
func WithRealIPExtractor(extractor IPExtractor) RealIPOption {
	return func(c *RealIPConfig) {
		c.IPExtractor = extractor
	}
}
