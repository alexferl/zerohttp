package realip

import (
	"net"
	"net/http"
	"strings"

	"github.com/alexferl/zerohttp/httpx"
)

// IPExtractor defines a function to extract the real client IP.
type IPExtractor func(*http.Request) string

// Config allows customization of real IP extraction
type Config struct {
	// IPExtractor function to extract real client IP (defaults to DefaultIPExtractor)
	IPExtractor IPExtractor
}

// DefaultConfig contains the default values for real IP configuration.
var DefaultConfig = Config{
	IPExtractor: DefaultIPExtractor,
}

// DefaultIPExtractor extracts the real client IP from various headers
func DefaultIPExtractor(r *http.Request) string {
	// Check X-Forwarded-For header first (most common)
	if xff := r.Header.Get(httpx.HeaderXForwardedFor); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header (used by Nginx and some proxies)
	if xri := r.Header.Get(httpx.HeaderXRealIP); xri != "" {
		return xri
	}

	// Check X-Forwarded header (less common)
	if xf := r.Header.Get(httpx.HeaderXForwarded); xf != "" {
		return xf
	}

	// Check Forwarded header (RFC 7239 standard)
	if forwarded := r.Header.Get(httpx.HeaderForwarded); forwarded != "" {
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
