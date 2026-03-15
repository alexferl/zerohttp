// Package middleware provides HTTP middleware for zerohttp.
//
// # JWT Authentication Middleware
//
// The JWTAuth middleware provides pluggable JWT authentication. Users bring their own
// JWT library by implementing the TokenStore interface.
//
// Basic usage:
//
//	app.Use(middleware.JWTAuth(config.JWTAuthConfig{
//	    TokenStore: myTokenStore,
//	    RequiredClaims: []string{"sub"},
//	}))
//
// The middleware supports:
//   - Custom token extraction (Bearer header, cookies, custom headers)
//   - Required claims validation
//   - Exempt paths and methods
//   - Custom error handling
//   - Token refresh handling
//
// For a zero-dependency option, use the built-in HS256 implementation:
//
//	cfg := config.JWTAuthConfig{
//	    TokenStore: middleware.NewHS256TokenStore(secret, opts),
//	}
//
// Security Note: The built-in HS256 implementation uses HMAC-SHA256 symmetric signing.
// For production systems requiring asymmetric keys (RS256, ES256, EdDSA), use a
// proper JWT library like golang-jwt/jwt or lestrrat-go/jwx.
package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/alexferl/zerohttp/config"
	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/internal/problem"
	"github.com/alexferl/zerohttp/metrics"
)

type (
	jwtClaimsContextKey struct{}
	jwtErrorContextKey  struct{}
	jwtTokenContextKey  struct{}
)

var (
	// JWTClaimsContextKey holds the validated JWT claims
	JWTClaimsContextKey = jwtClaimsContextKey{}
	// JWTErrorContextKey holds the JWT validation error
	JWTErrorContextKey = jwtErrorContextKey{}
	// JWTTokenContextKey holds the raw token string
	JWTTokenContextKey = jwtTokenContextKey{}
)

// JWTAuthError represents a JWT authentication error with RFC 9457 Problem Details
type JWTAuthError struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

// Error implements the error interface
func (e *JWTAuthError) Error() string {
	return e.Detail
}

// Common JWT auth errors
var (
	errMissingToken = &JWTAuthError{
		Title:  "Missing Authorization Token",
		Status: http.StatusUnauthorized,
		Detail: "Request is missing the Authorization header with Bearer token",
	}
	errInvalidToken = &JWTAuthError{
		Title:  "Invalid Token",
		Status: http.StatusUnauthorized,
		Detail: "The provided token is invalid or has expired",
	}
	errMissingRequiredClaim = &JWTAuthError{
		Title:  "Missing Required Claim",
		Status: http.StatusForbidden,
		Detail: "Token is missing a required claim",
	}
	errTokenGeneratorNotConfigured = &JWTAuthError{
		Title:  "Token Generator Not Configured",
		Status: http.StatusInternalServerError,
		Detail: "Token generation is not configured",
	}
	errTokenStoreNotConfigured = &JWTAuthError{
		Title:  "Token Store Not Configured",
		Status: http.StatusUnauthorized,
		Detail: "JWT authentication is not properly configured",
	}
)

// JWTAuth creates JWT authentication middleware
func JWTAuth(cfg ...config.JWTAuthConfig) func(http.Handler) http.Handler {
	c := config.DefaultJWTAuthConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	if c.TokenExtractor == nil {
		c.TokenExtractor = extractBearerToken
	}

	errorHandler := c.ErrorHandler
	if errorHandler == nil {
		errorHandler = defaultJWTErrorHandler
	}

	onSuccess := c.OnSuccess

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reg := metrics.SafeRegistry(metrics.GetRegistry(r.Context()))

			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			if slices.Contains(c.ExemptMethods, r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			for _, exemptPath := range c.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			if c.TokenStore == nil {
				reg.Counter("jwt_auth_requests_total", "result").WithLabelValues("not_configured").Inc()
				handleJWTError(w, r, errTokenStoreNotConfigured, errorHandler)
				return
			}

			tokenString := c.TokenExtractor(r)
			if tokenString == "" {
				reg.Counter("jwt_auth_requests_total", "result").WithLabelValues("missing").Inc()
				handleJWTError(w, r, errMissingToken, errorHandler)
				return
			}

			claims, err := c.TokenStore.Validate(r.Context(), tokenString)
			if err != nil {
				reg.Counter("jwt_auth_requests_total", "result").WithLabelValues("invalid").Inc()
				handleJWTError(w, r, errInvalidToken, errorHandler)
				return
			}

			// Normalize claims to map[string]any for consistent access
			normalizedClaims := normalizeClaims(claims)

			revoked, err := c.TokenStore.IsRevoked(r.Context(), normalizedClaims)
			if err != nil {
				reg.Counter("jwt_auth_requests_total", "result").WithLabelValues("invalid").Inc()
				handleJWTError(w, r, &JWTAuthError{
					Title:  "Token Revocation Check Failed",
					Status: http.StatusInternalServerError,
					Detail: err.Error(),
				}, errorHandler)
				return
			}
			if revoked {
				reg.Counter("jwt_auth_requests_total", "result").WithLabelValues("invalid").Inc()
				handleJWTError(w, r, &JWTAuthError{
					Title:  "Token Revoked",
					Status: http.StatusUnauthorized,
					Detail: "token has been revoked",
				}, errorHandler)
				return
			}

			if tokenType := getStringClaim(claims, config.JWTClaimType); tokenType == config.TokenTypeRefresh {
				reg.Counter("jwt_auth_requests_total", "result").WithLabelValues("invalid").Inc()
				handleJWTError(w, r, &JWTAuthError{
					Title:  "Invalid Token Type",
					Status: http.StatusUnauthorized,
					Detail: "refresh token cannot be used for authentication",
				}, errorHandler)
				return
			}

			for _, claim := range c.RequiredClaims {
				if !hasClaim(claims, claim) {
					reg.Counter("jwt_auth_requests_total", "result").WithLabelValues("invalid").Inc()
					handleJWTError(w, r, &JWTAuthError{
						Type:   errMissingRequiredClaim.Type,
						Title:  errMissingRequiredClaim.Title,
						Status: errMissingRequiredClaim.Status,
						Detail: "Token is missing required claim: " + claim,
					}, errorHandler)
					return
				}
			}

			reg.Counter("jwt_auth_requests_total", "result").WithLabelValues("valid").Inc()

			if onSuccess != nil {
				onSuccess(r, normalizedClaims)
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, JWTClaimsContextKey, normalizedClaims)
			ctx = context.WithValue(ctx, JWTTokenContextKey, tokenString)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetJWTToken retrieves the raw JWT token string from the request context.
func GetJWTToken(r *http.Request) string {
	if token, ok := r.Context().Value(JWTTokenContextKey).(string); ok {
		return token
	}
	return ""
}

// GetJWTError retrieves the JWT authentication error from the request context.
func GetJWTError(r *http.Request) *JWTAuthError {
	if err, ok := r.Context().Value(JWTErrorContextKey).(error); ok {
		var jwtErr *JWTAuthError
		if errors.As(err, &jwtErr) {
			return jwtErr
		}
	}
	return nil
}

// JWTClaims wraps JWTClaims to provide convenient accessor methods.
// Use GetJWTClaims(r) to get claims from a request.
//
// Example:
//
//	jwt := middleware.GetJWTClaims(r)
//	subject := jwt.Subject()
//	scopes := jwt.Scopes()
type JWTClaims struct {
	claims config.JWTClaims
}

// GetJWTClaims retrieves JWT claims from the request and returns a JWTClaims wrapper.
// This is the primary way to access JWT claims in handlers.
//
// Example:
//
//	jwt := middleware.GetJWTClaims(r)
//	subject := jwt.Subject()
//	if jwt.HasScope("admin") { ... }
func GetJWTClaims(r *http.Request) JWTClaims {
	if claims, ok := r.Context().Value(JWTClaimsContextKey).(config.JWTClaims); ok {
		return JWTClaims{claims: claims}
	}
	return JWTClaims{}
}

// asMap normalizes claims to map[string]any for consistent access.
// Handles both map[string]any and HS256Claims types.
func (j JWTClaims) asMap() (map[string]any, bool) {
	m := normalizeClaims(j.claims)
	return m, m != nil
}

// Subject returns the 'sub' claim.
func (j JWTClaims) Subject() string {
	return getStringClaim(j.claims, config.JWTClaimSubject)
}

// Issuer returns the 'iss' claim.
func (j JWTClaims) Issuer() string {
	return getStringClaim(j.claims, config.JWTClaimIssuer)
}

// Audience returns the 'aud' claim as a string slice.
// Returns all audiences if 'aud' is an array, or a single-element slice if it's a string.
func (j JWTClaims) Audience() []string {
	m, ok := j.asMap()
	if !ok {
		return nil
	}

	if aud, ok := m[config.JWTClaimAudience]; ok {
		switch v := aud.(type) {
		case string:
			return []string{v}
		case []string:
			return v
		case []any:
			audiences := make([]string, 0, len(v))
			for _, a := range v {
				if s, ok := a.(string); ok {
					audiences = append(audiences, s)
				}
			}
			return audiences
		}
	}
	return nil
}

// HasAudience checks if the token has a specific audience.
func (j JWTClaims) HasAudience(audience string) bool {
	return slices.Contains(j.Audience(), audience)
}

// JTI returns the 'jti' claim (JWT ID).
func (j JWTClaims) JTI() string {
	return getStringClaim(j.claims, config.JWTClaimJWTID)
}

// Expiration returns the 'exp' claim as time.Time.
func (j JWTClaims) Expiration() time.Time {
	m, ok := j.asMap()
	if !ok {
		return time.Time{}
	}

	if exp, ok := m[config.JWTClaimExpiration]; ok {
		switch v := exp.(type) {
		case float64:
			return time.Unix(int64(v), 0)
		case int64:
			return time.Unix(v, 0)
		}
	}
	return time.Time{}
}

// Scopes returns the 'scope' claim as a string slice.
func (j JWTClaims) Scopes() []string {
	m, ok := j.asMap()
	if !ok {
		return nil
	}

	if scope, ok := m[config.JWTClaimScope]; ok {
		switch v := scope.(type) {
		case string:
			return strings.Fields(v)
		case []string:
			return v
		case []any:
			scopes := make([]string, 0, len(v))
			for _, s := range v {
				if str, ok := s.(string); ok {
					scopes = append(scopes, str)
				}
			}
			return scopes
		}
	}
	return nil
}

// HasScope checks if the token has a specific scope.
func (j JWTClaims) HasScope(scope string) bool {
	return slices.Contains(j.Scopes(), scope)
}

// Raw returns the underlying claims.
// Use this for type assertion with third-party JWT libraries.
//
// Example with lestrrat-go/jwx:
//
//	token := middleware.GetJWTClaims(r).Raw().(jwt.Token)
//	subject := token.Subject()
func (j JWTClaims) Raw() config.JWTClaims {
	return j.claims
}

// GenerateAccessToken generates a new access token for the given claims.
// Automatically sets 'exp' claim based on AccessTokenTTL.
// Requires TokenStore to be configured.
func GenerateAccessToken(r *http.Request, claims config.JWTClaims, cfg config.JWTAuthConfig) (string, error) {
	if cfg.TokenStore == nil {
		return "", errTokenGeneratorNotConfigured
	}

	ttl := cfg.AccessTokenTTL
	if ttl == 0 {
		ttl = config.DefaultJWTAuthConfig.AccessTokenTTL
	}

	claims = addExpirationToClaims(claims, ttl)

	return cfg.TokenStore.Generate(r.Context(), claims, config.AccessToken, ttl)
}

// GenerateRefreshToken generates a new refresh token for the given claims.
// Automatically sets 'exp' claim based on RefreshTokenTTL and 'type': 'refresh'.
// Requires TokenStore to be configured.
func GenerateRefreshToken(r *http.Request, claims config.JWTClaims, cfg config.JWTAuthConfig) (string, error) {
	if cfg.TokenStore == nil {
		return "", errTokenGeneratorNotConfigured
	}

	ttl := cfg.RefreshTokenTTL
	if ttl == 0 {
		ttl = config.DefaultJWTAuthConfig.RefreshTokenTTL
	}

	// Add expiration and type if claims is a map
	claims = addExpirationToClaims(claims, ttl)
	claims = addTypeToClaims(claims, config.TokenTypeRefresh)

	return cfg.TokenStore.Generate(r.Context(), claims, config.RefreshToken, ttl)
}

// writeJWTError writes a JWTAuthError response
func writeJWTError(w http.ResponseWriter, r *http.Request, jwtErr *JWTAuthError) {
	detail := problem.NewDetail(jwtErr.Status, jwtErr.Detail)
	detail.Type = jwtErr.Type
	detail.Title = jwtErr.Title
	_ = detail.RenderAuto(w, r)
}

// tokenHandlerRequest parses and validates the refresh token from the request body.
// Returns the claims if validation succeeds, or an error response if it fails.
func tokenHandlerRequest(w http.ResponseWriter, r *http.Request, cfg config.JWTAuthConfig) (config.JWTClaims, bool) {
	if r.Method != http.MethodPost {
		detail := problem.NewDetail(http.StatusMethodNotAllowed, "Method not allowed")
		_ = detail.RenderAuto(w, r)
		return nil, false
	}

	if cfg.TokenStore == nil {
		writeJWTError(w, r, errTokenStoreNotConfigured)
		return nil, false
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJWTError(w, r, &JWTAuthError{
			Title:  "Invalid Request",
			Status: http.StatusBadRequest,
			Detail: "Request body must contain refresh_token",
		})
		return nil, false
	}

	if req.RefreshToken == "" {
		writeJWTError(w, r, &JWTAuthError{
			Title:  "Missing Refresh Token",
			Status: http.StatusUnprocessableEntity,
			Detail: "refresh_token is required",
		})
		return nil, false
	}

	claims, err := cfg.TokenStore.Validate(r.Context(), req.RefreshToken)
	if err != nil {
		writeJWTError(w, r, errInvalidToken)
		return nil, false
	}

	// Normalize claims to map[string]any for consistent access
	normalizedClaims := normalizeClaims(claims)

	if tokenType := getStringClaim(normalizedClaims, config.JWTClaimType); tokenType != config.TokenTypeRefresh {
		writeJWTError(w, r, &JWTAuthError{
			Title:  "Invalid Token Type",
			Status: http.StatusUnprocessableEntity,
			Detail: "Provided token is not a refresh token",
		})
		return nil, false
	}

	revoked, err := cfg.TokenStore.IsRevoked(r.Context(), normalizedClaims)
	if err != nil {
		writeJWTError(w, r, &JWTAuthError{
			Title:  "Token Revocation Check Failed",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		})
		return nil, false
	}
	if revoked {
		writeJWTError(w, r, &JWTAuthError{
			Title:  "Token Revoked",
			Status: http.StatusUnauthorized,
			Detail: "token has been revoked",
		})
		return nil, false
	}

	if err := cfg.TokenStore.Revoke(r.Context(), normalizedClaims); err != nil {
		writeJWTError(w, r, &JWTAuthError{
			Title:  "Token Revocation Failed",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		})
		return nil, false
	}

	return normalizedClaims, true
}

// RefreshTokenHandler returns an http.HandlerFunc that handles token refresh.
// Accepts: { "refresh_token": "..." }
// Returns: { "access_token": "...", "refresh_token": "...", "token_type": "Bearer", "expires_in": 900 }
// Users mount this at their chosen path: app.Post("/auth/refresh", middleware.RefreshTokenHandler(cfg))
func RefreshTokenHandler(cfg config.JWTAuthConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := tokenHandlerRequest(w, r, cfg)
		if !ok {
			return
		}

		accessToken, err := GenerateAccessToken(r, claims, cfg)
		if err != nil {
			writeJWTError(w, r, errTokenGeneratorNotConfigured)
			return
		}

		refreshToken, err := GenerateRefreshToken(r, claims, cfg)
		if err != nil {
			writeJWTError(w, r, errTokenGeneratorNotConfigured)
			return
		}

		expiresIn := cfg.AccessTokenTTL
		if expiresIn == 0 {
			expiresIn = config.DefaultJWTAuthConfig.AccessTokenTTL
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"token_type":    "Bearer",
			"expires_in":    int(expiresIn.Seconds()),
		})
	}
}

// LogoutTokenHandler returns an http.HandlerFunc that handles token revocation (logout).
// Accepts: { "refresh_token": "..." }
// Returns: { "message": "logged out successfully" }
// Users mount this at their chosen path: app.Post("/auth/logout", middleware.LogoutTokenHandler(cfg))
// Requires TokenStore to be configured in JWTAuthConfig.
func LogoutTokenHandler(cfg config.JWTAuthConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := tokenHandlerRequest(w, r, cfg)
		if !ok {
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": "logged out successfully",
		})
	}
}

// extractBearerToken extracts the JWT token from the Authorization header
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}

	return strings.TrimSpace(auth[len(prefix):])
}

// handleJWTError sends an error response
func handleJWTError(w http.ResponseWriter, r *http.Request, jwtErr *JWTAuthError, handler http.HandlerFunc) {
	// Add error to context so custom handlers can access it
	ctx := context.WithValue(r.Context(), JWTErrorContextKey, jwtErr)
	r = r.WithContext(ctx)

	if handler != nil {
		handler(w, r)
		return
	}
	defaultJWTErrorHandler(w, r)
}

// defaultJWTErrorHandler is the default error handler
func defaultJWTErrorHandler(w http.ResponseWriter, r *http.Request) {
	jwtErr := GetJWTError(r)
	if jwtErr == nil {
		jwtErr = errInvalidToken
	}

	detail := problem.NewDetail(jwtErr.Status, jwtErr.Detail)
	detail.Type = jwtErr.Type
	detail.Title = jwtErr.Title
	_ = detail.RenderAuto(w, r)
}

// hasClaim checks if a claim exists in the claims
func hasClaim(claims config.JWTClaims, key string) bool {
	switch c := claims.(type) {
	case map[string]any:
		_, ok := c[key]
		return ok
	case HS256Claims:
		_, ok := c[key]
		return ok
	default:
		// Handle other map types (e.g., jwt.MapClaims from golang-jwt)
		return getMapClaim(c, key) != nil
	}
}

// normalizeClaims converts claims to map[string]any for consistent access.
// Handles map[string]any, HS256Claims, and other map types via reflection.
// Always returns a usable map (never nil) - converts non-map types to map with "_raw" key.
func normalizeClaims(claims config.JWTClaims) map[string]any {
	if claims == nil {
		return nil
	}
	switch c := claims.(type) {
	case map[string]any:
		return c
	case HS256Claims:
		return map[string]any(c)
	default:
		// Try reflection for other map types (e.g., jwt.MapClaims)
		v := reflect.ValueOf(claims)
		if v.Kind() == reflect.Map && v.Type().Key().Kind() == reflect.String {
			m := make(map[string]any, v.Len())
			for _, key := range v.MapKeys() {
				m[key.String()] = v.MapIndex(key).Interface()
			}
			return m
		}
		// Last resort: wrap in a map
		return map[string]any{"_raw": claims}
	}
}

// getStringClaim extracts a string claim from claims
func getStringClaim(claims config.JWTClaims, key string) string {
	m := normalizeClaims(claims)
	if m != nil {
		if v, ok := m[key]; ok {
			switch s := v.(type) {
			case string:
				return s
			case []string:
				if len(s) > 0 {
					return s[0]
				}
			case []any:
				if len(s) > 0 {
					if str, ok := s[0].(string); ok {
						return str
					}
				}
			}
		}
		return ""
	}

	// Handle other map types (e.g., jwt.MapClaims from golang-jwt) via reflection
	if v := getMapClaim(claims, key); v != nil {
		return extractStringValue(v)
	}
	return ""
}

// getMapClaim extracts a value from any map-like type using reflection
func getMapClaim(claims config.JWTClaims, key string) any {
	if claims == nil {
		return nil
	}
	switch m := claims.(type) {
	case map[string]any:
		return m[key]
	default:
		v := reflect.ValueOf(claims)
		if v.Kind() == reflect.Map && v.Type().Key().Kind() == reflect.String {
			val := v.MapIndex(reflect.ValueOf(key))
			if val.IsValid() {
				return val.Interface()
			}
		}
		return nil
	}
}

// extractStringValue converts a value to string
func extractStringValue(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case []string:
		if len(s) > 0 {
			return s[0]
		}
	case []any:
		if len(s) > 0 {
			if str, ok := s[0].(string); ok {
				return str
			}
		}
	}
	return ""
}

// deepCopyMap creates a deep copy of a map[string]any
func deepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	newMap := make(map[string]any, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case map[string]any:
			newMap[k] = deepCopyMap(val)
		case []any:
			newMap[k] = deepCopySlice(val)
		default:
			newMap[k] = v
		}
	}
	return newMap
}

// deepCopySlice creates a deep copy of a []any
func deepCopySlice(s []any) []any {
	if s == nil {
		return nil
	}
	newSlice := make([]any, len(s))
	for i, v := range s {
		switch val := v.(type) {
		case map[string]any:
			newSlice[i] = deepCopyMap(val)
		case []any:
			newSlice[i] = deepCopySlice(val)
		default:
			newSlice[i] = v
		}
	}
	return newSlice
}

// addExpirationToClaims adds exp claim to map claims
func addExpirationToClaims(claims config.JWTClaims, ttl time.Duration) config.JWTClaims {
	switch c := claims.(type) {
	case map[string]any:
		newClaims := deepCopyMap(c)
		newClaims[config.JWTClaimExpiration] = time.Now().Add(ttl).Unix()
		return newClaims
	case HS256Claims:
		newClaims := deepCopyMap(c)
		newClaims[config.JWTClaimExpiration] = time.Now().Add(ttl).Unix()
		return HS256Claims(newClaims)
	default:
		return claims
	}
}

// addTypeToClaims adds type claim to map claims
func addTypeToClaims(claims config.JWTClaims, tokenType string) config.JWTClaims {
	switch c := claims.(type) {
	case map[string]any:
		newClaims := deepCopyMap(c)
		newClaims[config.JWTClaimType] = tokenType
		return newClaims
	case HS256Claims:
		newClaims := deepCopyMap(c)
		newClaims[config.JWTClaimType] = tokenType
		return HS256Claims(newClaims)
	default:
		return claims
	}
}
