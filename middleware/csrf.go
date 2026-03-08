package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/internal/problem"
)

// CSRFContextKey is the key type for CSRF token in context
type CSRFContextKey string

const (
	// csrfTokenContextKey is the context key for the CSRF token
	csrfTokenContextKey CSRFContextKey = "csrf_token"
	// defaultTokenLength is the length of the random token in bytes
	defaultTokenLength = 32
)

// CSRF returns middleware that provides CSRF protection using the double-submit cookie pattern
func CSRF(cfg ...config.CSRFConfig) func(http.Handler) http.Handler {
	c := config.DefaultCSRFConfig
	if len(cfg) > 0 {
		if cfg[0].CookieName != "" {
			c.CookieName = cfg[0].CookieName
		}
		if cfg[0].CookieMaxAge != 0 {
			c.CookieMaxAge = cfg[0].CookieMaxAge
		}
		if cfg[0].CookiePath != "" {
			c.CookiePath = cfg[0].CookiePath
		}
		if cfg[0].CookieDomain != "" {
			c.CookieDomain = cfg[0].CookieDomain
		}
		// For CookieSecure: use provided value if set, otherwise use default
		if cfg[0].CookieSecure != nil {
			c.CookieSecure = cfg[0].CookieSecure
		}
		if cfg[0].CookieSameSite != 0 {
			c.CookieSameSite = cfg[0].CookieSameSite
		}
		if cfg[0].TokenLookup != "" {
			c.TokenLookup = cfg[0].TokenLookup
		}
		if cfg[0].ErrorHandler != nil {
			c.ErrorHandler = cfg[0].ErrorHandler
		}
		if cfg[0].ExemptPaths != nil {
			c.ExemptPaths = cfg[0].ExemptPaths
		}
		if cfg[0].ExemptMethods != nil {
			c.ExemptMethods = cfg[0].ExemptMethods
		}
		if cfg[0].HMACKey != nil {
			c.HMACKey = cfg[0].HMACKey
		}
	}

	// HMAC key is required - fail fast if not provided
	if len(c.HMACKey) == 0 {
		panic("CSRF: HMACKey is required. Set a fixed key in CSRFConfig{HMACKey: []byte(\"your-secret-32-bytes!!\")} " +
			"or load from environment variables. Using a random key would invalidate all tokens on server restart.")
	}
	hmacKey := c.HMACKey

	exemptMethodMap := make(map[string]bool)
	for _, method := range c.ExemptMethods {
		exemptMethodMap[strings.ToUpper(method)] = true
	}

	lookupSource, lookupName := parseTokenLookup(c.TokenLookup)

	errorHandler := c.ErrorHandler
	if errorHandler == nil {
		errorHandler = defaultCSRFErrorHandler
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, exemptPath := range c.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			if exemptMethodMap[strings.ToUpper(r.Method)] {
				cookie, err := r.Cookie(c.CookieName)
				var token string
				if err != nil || cookie.Value == "" || !validateTokenFormat(cookie.Value) {
					token = generateToken(hmacKey)
					setCSRFCookie(w, c, token)
				} else {
					token = cookie.Value
				}

				ctx := r.Context()
				ctx = context.WithValue(ctx, csrfTokenContextKey, token)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			cookie, err := r.Cookie(c.CookieName)
			if err != nil || cookie.Value == "" {
				errorHandler(w, r)
				return
			}

			requestToken := extractToken(r, lookupSource, lookupName)
			if requestToken == "" {
				errorHandler(w, r)
				return
			}

			cookieToken := cookie.Value
			if !compareTokens(cookieToken, requestToken, hmacKey) {
				errorHandler(w, r)
				return
			}

			newToken := generateToken(hmacKey)
			setCSRFCookie(w, c, newToken)

			ctx := r.Context()
			ctx = context.WithValue(ctx, csrfTokenContextKey, newToken)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetCSRFToken retrieves the CSRF token from the request context
// Returns empty string if no token is present
func GetCSRFToken(r *http.Request) string {
	token, ok := r.Context().Value(csrfTokenContextKey).(string)
	if !ok {
		return ""
	}
	return token
}

// generateToken creates a new signed CSRF token
func generateToken(key []byte) string {
	tokenBytes := make([]byte, defaultTokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return ""
	}

	mac := hmac.New(sha256.New, key)
	mac.Write(tokenBytes)
	signature := mac.Sum(nil)

	combined := append(tokenBytes, signature...)
	return base64.RawURLEncoding.EncodeToString(combined)
}

// validateTokenFormat checks if the token has a valid format (proper base64 and signature length)
func validateTokenFormat(token string) bool {
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return false
	}

	// Must have token bytes + signature
	if len(data) < defaultTokenLength+sha256.Size {
		return false
	}

	return true
}

// compareTokens compares two tokens using constant-time comparison
func compareTokens(cookieToken, requestToken string, key []byte) bool {
	// First do a string comparison to avoid base64 decoding if tokens don't match
	if subtle.ConstantTimeCompare([]byte(cookieToken), []byte(requestToken)) != 1 {
		return false
	}

	data, err := base64.RawURLEncoding.DecodeString(cookieToken)
	if err != nil {
		return false
	}

	if len(data) < defaultTokenLength+sha256.Size {
		return false
	}

	tokenBytes := data[:defaultTokenLength]
	signature := data[defaultTokenLength:]

	mac := hmac.New(sha256.New, key)
	mac.Write(tokenBytes)
	expectedMAC := mac.Sum(nil)

	return subtle.ConstantTimeCompare(signature, expectedMAC) == 1
}

// parseTokenLookup parses the token lookup string into source and name
func parseTokenLookup(lookup string) (source, name string) {
	parts := strings.SplitN(lookup, ":", 2)
	if len(parts) != 2 {
		return "header", "X-CSRF-Token"
	}
	return strings.ToLower(parts[0]), parts[1]
}

// extractToken extracts the CSRF token from the request based on source and name
func extractToken(r *http.Request, source, name string) string {
	switch source {
	case "header":
		return r.Header.Get(name)
	case "form":
		contentType := r.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "multipart/form-data") {
			if err := r.ParseMultipartForm(32 << 20); err != nil {
				return ""
			}
			if r.MultipartForm != nil {
				values := r.MultipartForm.Value[name]
				if len(values) > 0 {
					return values[0]
				}
			}
			return ""
		}
		if err := r.ParseForm(); err != nil {
			return ""
		}
		return r.FormValue(name)
	case "query":
		return r.URL.Query().Get(name)
	default:
		return r.Header.Get(name)
	}
}

// setCSRFCookie sets the CSRF cookie on the response
func setCSRFCookie(w http.ResponseWriter, cfg config.CSRFConfig, token string) {
	// Handle CookieSecure pointer - default to true if nil
	secure := true
	if cfg.CookieSecure != nil {
		secure = *cfg.CookieSecure
	}

	cookie := &http.Cookie{
		Name:     cfg.CookieName,
		Value:    token,
		MaxAge:   cfg.CookieMaxAge,
		Domain:   cfg.CookieDomain,
		Path:     cfg.CookiePath,
		Secure:   secure,
		HttpOnly: true,
		SameSite: cfg.CookieSameSite,
	}
	http.SetCookie(w, cookie)
}

// defaultCSRFErrorHandler is the default handler for CSRF validation failures
func defaultCSRFErrorHandler(w http.ResponseWriter, r *http.Request) {
	detail := problem.NewDetail(http.StatusForbidden, "CSRF token is missing or invalid")
	_ = detail.Render(w)
}
