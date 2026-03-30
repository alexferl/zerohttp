package jwtauth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/alexferl/zerohttp/httpx"
	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/internal/mwutil"
	"github.com/alexferl/zerohttp/internal/problem"
	"github.com/alexferl/zerohttp/metrics"
)

type (
	claimsContextKey struct{}
	errorContextKey  struct{}
	tokenContextKey  struct{}
)

var (
	// ClaimsContextKey holds the validated JWT claims
	ClaimsContextKey = claimsContextKey{}
	// ErrorContextKey holds the JWT validation error
	ErrorContextKey = errorContextKey{}
	// TokenContextKey holds the raw token string
	TokenContextKey = tokenContextKey{}
)

// AuthError represents a JWT authentication error with RFC 9457 Problem Details
type AuthError struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

// Error implements the error interface
func (e *AuthError) Error() string {
	return e.Detail
}

// Common JWT auth errors
var (
	errMissingToken = &AuthError{
		Title:  "Missing Authorization Token",
		Status: http.StatusUnauthorized,
		Detail: "Request is missing the Authorization header with Bearer token",
	}
	errInvalidToken = &AuthError{
		Title:  "Invalid Token",
		Status: http.StatusUnauthorized,
		Detail: "The provided token is invalid or has expired",
	}
	errMissingRequiredClaim = &AuthError{
		Title:  "Missing Required Claim",
		Status: http.StatusForbidden,
		Detail: "Token is missing a required claim",
	}
	errTokenGeneratorNotConfigured = &AuthError{
		Title:  "Token Generator Not Configured",
		Status: http.StatusInternalServerError,
		Detail: "Token generation is not configured",
	}
	errTokenStoreNotConfigured = &AuthError{
		Title:  "Token Store Not Configured",
		Status: http.StatusUnauthorized,
		Detail: "JWT authentication is not properly configured",
	}
)

// New creates a JWT authentication middleware with the provided configuration
func New(cfg ...Config) func(http.Handler) http.Handler {
	c := DefaultConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	if c.Extractor == nil {
		c.Extractor = extractBearerToken
	}

	mwutil.ValidatePathConfig(c.ExcludedPaths, c.IncludedPaths, "JWTAuth")

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

			if slices.Contains(c.ExcludedMethods, r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			if !mwutil.ShouldProcessMiddleware(r.URL.Path, c.IncludedPaths, c.ExcludedPaths) {
				next.ServeHTTP(w, r)
				return
			}

			if c.Store == nil {
				reg.Counter("jwt_auth_requests_total", "result").WithLabelValues("not_configured").Inc()
				handleJWTError(w, r, errTokenStoreNotConfigured, errorHandler)
				return
			}

			tokenString := c.Extractor(r)
			if tokenString == "" {
				reg.Counter("jwt_auth_requests_total", "result").WithLabelValues("missing").Inc()
				handleJWTError(w, r, errMissingToken, errorHandler)
				return
			}

			claims, err := c.Store.Validate(r.Context(), tokenString)
			if err != nil {
				reg.Counter("jwt_auth_requests_total", "result").WithLabelValues("invalid").Inc()
				handleJWTError(w, r, errInvalidToken, errorHandler)
				return
			}

			// Normalize claims to map[string]any for consistent access
			normalizedClaims := normalizeClaims(claims)

			revoked, err := c.Store.IsRevoked(r.Context(), normalizedClaims)
			if err != nil {
				reg.Counter("jwt_auth_requests_total", "result").WithLabelValues("invalid").Inc()
				handleJWTError(w, r, &AuthError{
					Title:  "Token Revocation Check Failed",
					Status: http.StatusInternalServerError,
					Detail: err.Error(),
				}, errorHandler)
				return
			}
			if revoked {
				reg.Counter("jwt_auth_requests_total", "result").WithLabelValues("invalid").Inc()
				handleJWTError(w, r, &AuthError{
					Title:  "Token Revoked",
					Status: http.StatusUnauthorized,
					Detail: "token has been revoked",
				}, errorHandler)
				return
			}

			if tokenType := getStringClaim(claims, JWTClaimType); tokenType == TokenTypeRefresh {
				reg.Counter("jwt_auth_requests_total", "result").WithLabelValues("invalid").Inc()
				handleJWTError(w, r, &AuthError{
					Title:  "Invalid Token Type",
					Status: http.StatusUnauthorized,
					Detail: "refresh token cannot be used for authentication",
				}, errorHandler)
				return
			}

			for _, claim := range c.RequiredClaims {
				if !hasClaim(claims, claim) {
					reg.Counter("jwt_auth_requests_total", "result").WithLabelValues("invalid").Inc()
					handleJWTError(w, r, &AuthError{
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
			ctx = context.WithValue(ctx, ClaimsContextKey, normalizedClaims)
			ctx = context.WithValue(ctx, TokenContextKey, tokenString)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetToken retrieves the raw JWT token string from the request context.
func GetToken(r *http.Request) string {
	if token, ok := r.Context().Value(TokenContextKey).(string); ok {
		return token
	}
	return ""
}

// GetError retrieves the JWT authentication error from the request context.
func GetError(r *http.Request) *AuthError {
	if err, ok := r.Context().Value(ErrorContextKey).(error); ok {
		var jwtErr *AuthError
		if errors.As(err, &jwtErr) {
			return jwtErr
		}
	}
	return nil
}

// Claims wraps Claims to provide convenient accessor methods.
// Use GetClaims(r) to get claims from a request.
//
// Example:
//
//	claims := jwtauth.GetClaims(r)
//	subject := claims.Subject()
//	scopes := claims.Scopes()
type Claims struct {
	claims JWTClaims
}

// GetClaims retrieves JWT claims from the request and returns a Claims wrapper.
// This is the primary way to access JWT claims in handlers.
//
// Example:
//
//	claims := jwtauth.GetClaims(r)
//	subject := claims.Subject()
//	if claims.HasScope("admin") { ... }
func GetClaims(r *http.Request) Claims {
	if claims, ok := r.Context().Value(ClaimsContextKey).(JWTClaims); ok {
		return Claims{claims: claims}
	}
	return Claims{}
}

// asMap normalizes claims to map[string]any for consistent access.
// Handles both map[string]any and HS256Claims types.
func (j Claims) asMap() (map[string]any, bool) {
	m := normalizeClaims(j.claims)
	return m, m != nil
}

// Subject returns the 'sub' claim.
func (j Claims) Subject() string {
	return getStringClaim(j.claims, JWTClaimSubject)
}

// Issuer returns the 'iss' claim.
func (j Claims) Issuer() string {
	return getStringClaim(j.claims, JWTClaimIssuer)
}

// Audience returns the 'aud' claim as a string slice.
// Returns all audiences if 'aud' is an array, or a single-element slice if it's a string.
func (j Claims) Audience() []string {
	m, ok := j.asMap()
	if !ok {
		return nil
	}

	if aud, ok := m[JWTClaimAudience]; ok {
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
func (j Claims) HasAudience(audience string) bool {
	return slices.Contains(j.Audience(), audience)
}

// JTI returns the 'jti' claim (JWT ID).
func (j Claims) JTI() string {
	return getStringClaim(j.claims, JWTClaimJWTID)
}

// Expiration returns the 'exp' claim as time.Time.
func (j Claims) Expiration() time.Time {
	m, ok := j.asMap()
	if !ok {
		return time.Time{}
	}

	if exp, ok := m[JWTClaimExpiration]; ok {
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
func (j Claims) Scopes() []string {
	m, ok := j.asMap()
	if !ok {
		return nil
	}

	if scope, ok := m[JWTClaimScope]; ok {
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
func (j Claims) HasScope(scope string) bool {
	return slices.Contains(j.Scopes(), scope)
}

// Raw returns the underlying claims.
// Use this for type assertion with third-party JWT libraries.
//
// Example with lestrrat-go/jwx:
//
//	token := jwtauth.GetClaims(r).Raw().(jwt.Token)
//	subject := token.Subject()
func (j Claims) Raw() JWTClaims {
	return j.claims
}

// GenerateAccessToken generates a new access token for the given claims.
// Automatically sets 'exp' claim based on AccessTokenTTL.
// Requires Store to be configured.
func GenerateAccessToken(r *http.Request, claims JWTClaims, cfg Config) (string, error) {
	if cfg.Store == nil {
		return "", errTokenGeneratorNotConfigured
	}

	ttl := cfg.AccessTokenTTL
	if ttl == 0 {
		ttl = DefaultConfig.AccessTokenTTL
	}

	claims = addExpirationToClaims(claims, ttl)

	return cfg.Store.Generate(r.Context(), claims, AccessToken, ttl)
}

// GenerateRefreshToken generates a new refresh token for the given claims.
// Automatically sets 'exp' claim based on RefreshTokenTTL and 'type': 'refresh'.
// Requires Store to be configured.
func GenerateRefreshToken(r *http.Request, claims JWTClaims, cfg Config) (string, error) {
	if cfg.Store == nil {
		return "", errTokenGeneratorNotConfigured
	}

	ttl := cfg.RefreshTokenTTL
	if ttl == 0 {
		ttl = DefaultConfig.RefreshTokenTTL
	}

	// Add expiration and type if claims is a map
	claims = addExpirationToClaims(claims, ttl)
	claims = addTypeToClaims(claims, TokenTypeRefresh)

	return cfg.Store.Generate(r.Context(), claims, RefreshToken, ttl)
}

// SetCookie sets the JWT token as a cookie with the configured cookie settings.
// Uses Config.Cookie for cookie attributes. MaxAge defaults to AccessTokenTTL if not set.
func SetCookie(w http.ResponseWriter, token string, cfg Config) {
	cookieCfg := cfg.Cookie
	if cookieCfg.Name == "" {
		cookieCfg.Name = DefaultCookieConfig.Name
	}
	if cookieCfg.Path == "" {
		cookieCfg.Path = DefaultCookieConfig.Path
	}

	maxAge := cookieCfg.MaxAge
	if maxAge == 0 {
		ttl := cfg.AccessTokenTTL
		if ttl == 0 {
			ttl = DefaultConfig.AccessTokenTTL
		}
		maxAge = int(ttl.Seconds())
	}

	cookie := &http.Cookie{
		Name:     cookieCfg.Name,
		Value:    token,
		Path:     cookieCfg.Path,
		Domain:   cookieCfg.Domain,
		MaxAge:   maxAge,
		Secure:   cookieCfg.Secure,
		HttpOnly: cookieCfg.HttpOnly,
		SameSite: cookieCfg.SameSite,
	}
	http.SetCookie(w, cookie)
}

// DeleteCookie deletes the JWT cookie by setting MaxAge to -1.
func DeleteCookie(w http.ResponseWriter, cfg Config) {
	cookieCfg := cfg.Cookie
	if cookieCfg.Name == "" {
		cookieCfg.Name = DefaultCookieConfig.Name
	}
	if cookieCfg.Path == "" {
		cookieCfg.Path = DefaultCookieConfig.Path
	}

	cookie := &http.Cookie{
		Name:     cookieCfg.Name,
		Value:    "",
		Path:     cookieCfg.Path,
		Domain:   cookieCfg.Domain,
		MaxAge:   -1,
		Secure:   cookieCfg.Secure,
		HttpOnly: cookieCfg.HttpOnly,
		SameSite: cookieCfg.SameSite,
	}
	http.SetCookie(w, cookie)
}

// SetRefreshCookie sets the refresh token as a cookie with RefreshPath and RefreshName.
// Uses Config.Cookie.RefreshPath for the cookie path. MaxAge defaults to RefreshTokenTTL.
func SetRefreshCookie(w http.ResponseWriter, token string, cfg Config) {
	cookieCfg := cfg.Cookie
	name := cookieCfg.RefreshName
	if name == "" {
		name = DefaultCookieConfig.RefreshName
	}
	path := cookieCfg.RefreshPath
	if path == "" {
		path = DefaultCookieConfig.RefreshPath
	}

	maxAge := cookieCfg.MaxAge
	if maxAge == 0 {
		ttl := cfg.RefreshTokenTTL
		if ttl == 0 {
			ttl = DefaultConfig.RefreshTokenTTL
		}
		maxAge = int(ttl.Seconds())
	}

	cookie := &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     path,
		Domain:   cookieCfg.Domain,
		MaxAge:   maxAge,
		Secure:   cookieCfg.Secure,
		HttpOnly: cookieCfg.HttpOnly,
		SameSite: cookieCfg.SameSite,
	}
	http.SetCookie(w, cookie)
}

// CookieExtractor returns an extractor that extracts the JWT token from a cookie.
// Use this as the Extractor in Config to enable cookie-only authentication.
func CookieExtractor(cookieName string) func(r *http.Request) string {
	return func(r *http.Request) string {
		cookie, err := r.Cookie(cookieName)
		if err == nil && cookie.Value != "" {
			return cookie.Value
		}
		return ""
	}
}

// HeaderOrCookieExtractor returns an extractor that first checks the Authorization header,
// then falls back to the specified cookie name.
func HeaderOrCookieExtractor(cookieName string) func(r *http.Request) string {
	return func(r *http.Request) string {
		// First try Authorization header
		token := extractBearerToken(r)
		if token != "" {
			return token
		}

		// Fall back to cookie
		cookie, err := r.Cookie(cookieName)
		if err == nil && cookie.Value != "" {
			return cookie.Value
		}

		return ""
	}
}

// writeJWTError writes a AuthError response
func writeJWTError(w http.ResponseWriter, r *http.Request, jwtErr *AuthError) {
	detail := problem.NewDetail(jwtErr.Status, jwtErr.Detail)
	detail.Type = jwtErr.Type
	detail.Title = jwtErr.Title
	_ = detail.RenderAuto(w, r)
}

// tokenHandlerRequest parses and validates the refresh token from the request body.
// Returns the claims if validation succeeds, or an error response if it fails.
func tokenHandlerRequest(w http.ResponseWriter, r *http.Request, cfg Config) (JWTClaims, bool) {
	if r.Method != http.MethodPost {
		detail := problem.NewDetail(http.StatusMethodNotAllowed, "Method not allowed")
		_ = detail.RenderAuto(w, r)
		return nil, false
	}

	if cfg.Store == nil {
		writeJWTError(w, r, errTokenStoreNotConfigured)
		return nil, false
	}

	refreshToken := ""

	// First try to read from request body
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
		refreshToken = req.RefreshToken
	}

	// Fall back to refresh token cookie if body didn't have it and cookies are enabled
	if refreshToken == "" && cfg.Cookie.Enabled {
		cookieName := cfg.Cookie.RefreshName
		if cookieName == "" {
			cookieName = DefaultCookieConfig.RefreshName
		}
		if cookie, err := r.Cookie(cookieName); err == nil {
			refreshToken = cookie.Value
		}
	}

	if refreshToken == "" {
		writeJWTError(w, r, &AuthError{
			Title:  "Missing Refresh Token",
			Status: http.StatusUnprocessableEntity,
			Detail: "refresh_token is required",
		})
		return nil, false
	}

	claims, err := cfg.Store.Validate(r.Context(), refreshToken)
	if err != nil {
		writeJWTError(w, r, errInvalidToken)
		return nil, false
	}

	// Normalize claims to map[string]any for consistent access
	normalizedClaims := normalizeClaims(claims)

	if tokenType := getStringClaim(normalizedClaims, JWTClaimType); tokenType != TokenTypeRefresh {
		writeJWTError(w, r, &AuthError{
			Title:  "Invalid Token Type",
			Status: http.StatusUnprocessableEntity,
			Detail: "Provided token is not a refresh token",
		})
		return nil, false
	}

	revoked, err := cfg.Store.IsRevoked(r.Context(), normalizedClaims)
	if err != nil {
		writeJWTError(w, r, &AuthError{
			Title:  "Token Revocation Check Failed",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		})
		return nil, false
	}
	if revoked {
		writeJWTError(w, r, &AuthError{
			Title:  "Token Revoked",
			Status: http.StatusUnauthorized,
			Detail: "token has been revoked",
		})
		return nil, false
	}

	if err := cfg.Store.Revoke(r.Context(), normalizedClaims); err != nil {
		writeJWTError(w, r, &AuthError{
			Title:  "Token Revocation Failed",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		})
		return nil, false
	}

	return normalizedClaims, true
}

// RefreshTokenHandler returns an http.HandlerFunc that handles token refresh.
// Accepts: { "refresh_token": "..." } or reads from cookie if Cookie.Enabled is true
// Returns: { "access_token": "...", "refresh_token": "...", "token_type": httpx.AuthSchemeBearer, "expires_in": 900 }
// Also sets the new refresh token as a cookie when Cookie.Enabled is true.
// Users mount this at their chosen path: app.POST("/auth/refresh", jwtauth.RefreshTokenHandler(cfg))
func RefreshTokenHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := tokenHandlerRequest(w, r, cfg)
		if !ok {
			return
		}

		// Extract subject to identify user - in production, fetch fresh claims from database here
		subject := extractSubject(claims)
		accessClaims := map[string]any{"sub": subject}

		accessToken, err := GenerateAccessToken(r, accessClaims, cfg)
		if err != nil {
			writeJWTError(w, r, errTokenGeneratorNotConfigured)
			return
		}

		// Use same user claims but GenerateRefreshToken will add new jti and type
		refreshToken, err := GenerateRefreshToken(r, accessClaims, cfg)
		if err != nil {
			writeJWTError(w, r, errTokenGeneratorNotConfigured)
			return
		}

		expiresIn := cfg.AccessTokenTTL
		if expiresIn == 0 {
			expiresIn = DefaultConfig.AccessTokenTTL
		}

		// Set both cookies if enabled (access token with Path: /, refresh token with RefreshPath)
		if cfg.Cookie.Enabled {
			cookieCfg := cfg.Cookie
			if cookieCfg.Name == "" {
				cookieCfg.Name = DefaultCookieConfig.Name
			}

			// Access token cookie with Path: /
			accessTTL := cfg.AccessTokenTTL
			if accessTTL == 0 {
				accessTTL = DefaultConfig.AccessTokenTTL
			}
			accessPath := cookieCfg.Path
			if accessPath == "" {
				accessPath = DefaultCookieConfig.Path
			}
			accessCookie := &http.Cookie{
				Name:     cookieCfg.Name,
				Value:    accessToken,
				Path:     accessPath,
				Domain:   cookieCfg.Domain,
				MaxAge:   int(accessTTL.Seconds()),
				Secure:   cookieCfg.Secure,
				HttpOnly: cookieCfg.HttpOnly,
				SameSite: cookieCfg.SameSite,
			}
			http.SetCookie(w, accessCookie)

			// Refresh token cookie with RefreshPath
			refreshTTL := cfg.RefreshTokenTTL
			if refreshTTL == 0 {
				refreshTTL = DefaultConfig.RefreshTokenTTL
			}
			refreshPath := cookieCfg.RefreshPath
			if refreshPath == "" {
				refreshPath = DefaultCookieConfig.RefreshPath
			}
			refreshName := cookieCfg.RefreshName
			if refreshName == "" {
				refreshName = DefaultCookieConfig.RefreshName
			}
			refreshCookie := &http.Cookie{
				Name:     refreshName,
				Value:    refreshToken,
				Path:     refreshPath,
				Domain:   cookieCfg.Domain,
				MaxAge:   int(refreshTTL.Seconds()),
				Secure:   cookieCfg.Secure,
				HttpOnly: cookieCfg.HttpOnly,
				SameSite: cookieCfg.SameSite,
			}
			http.SetCookie(w, refreshCookie)
		}

		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"token_type":    httpx.AuthSchemeBearer,
			"expires_in":    int(expiresIn.Seconds()),
		})
	}
}

// LogoutTokenHandler returns an http.HandlerFunc that handles token revocation (logout).
// Accepts: { "refresh_token": "..." } or reads from cookie if Cookie.Enabled is true
// Returns: { "message": "logged out successfully" }
// Also deletes the cookie when Cookie.Enabled is true.
// Users mount this at their chosen path: app.POST("/auth/logout", jwtauth.LogoutTokenHandler(cfg))
// Requires Store to be configured in Config.
func LogoutTokenHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := tokenHandlerRequest(w, r, cfg)
		if !ok {
			return
		}

		// Delete both cookies if enabled (access token with Path: /, refresh token with RefreshPath)
		if cfg.Cookie.Enabled {
			cookieCfg := cfg.Cookie
			if cookieCfg.Name == "" {
				cookieCfg.Name = DefaultCookieConfig.Name
			}

			// Delete access token cookie with Path: /
			accessPath := cookieCfg.Path
			if accessPath == "" {
				accessPath = DefaultCookieConfig.Path
			}
			accessCookie := &http.Cookie{
				Name:     cookieCfg.Name,
				Value:    "",
				Path:     accessPath,
				Domain:   cookieCfg.Domain,
				MaxAge:   -1,
				Secure:   cookieCfg.Secure,
				HttpOnly: cookieCfg.HttpOnly,
				SameSite: cookieCfg.SameSite,
			}
			http.SetCookie(w, accessCookie)

			// Delete refresh token cookie with RefreshPath
			refreshPath := cookieCfg.RefreshPath
			if refreshPath == "" {
				refreshPath = DefaultCookieConfig.RefreshPath
			}
			refreshName := cookieCfg.RefreshName
			if refreshName == "" {
				refreshName = DefaultCookieConfig.RefreshName
			}
			refreshCookie := &http.Cookie{
				Name:     refreshName,
				Value:    "",
				Path:     refreshPath,
				Domain:   cookieCfg.Domain,
				MaxAge:   -1,
				Secure:   cookieCfg.Secure,
				HttpOnly: cookieCfg.HttpOnly,
				SameSite: cookieCfg.SameSite,
			}
			http.SetCookie(w, refreshCookie)
		}

		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": "logged out successfully",
		})
	}
}

// extractBearerToken extracts the JWT token from the Authorization header
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get(httpx.HeaderAuthorization)
	if auth == "" {
		return ""
	}

	const prefix = httpx.AuthSchemeBearer + " "
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}

	return strings.TrimSpace(auth[len(prefix):])
}

// handleJWTError sends an error response
func handleJWTError(w http.ResponseWriter, r *http.Request, jwtErr *AuthError, handler http.HandlerFunc) {
	// Add error to context so custom handlers can access it
	ctx := context.WithValue(r.Context(), ErrorContextKey, jwtErr)
	r = r.WithContext(ctx)

	if handler != nil {
		handler(w, r)
		return
	}
	defaultJWTErrorHandler(w, r)
}

// defaultJWTErrorHandler is the default error handler
func defaultJWTErrorHandler(w http.ResponseWriter, r *http.Request) {
	jwtErr := GetError(r)
	if jwtErr == nil {
		jwtErr = errInvalidToken
	}

	detail := problem.NewDetail(jwtErr.Status, jwtErr.Detail)
	detail.Type = jwtErr.Type
	detail.Title = jwtErr.Title
	_ = detail.RenderAuto(w, r)
}

// hasClaim checks if a claim exists in the claims
func hasClaim(claims JWTClaims, key string) bool {
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
func normalizeClaims(claims JWTClaims) map[string]any {
	if claims == nil {
		return nil
	}
	switch c := claims.(type) {
	case map[string]any:
		return c
	case HS256Claims:
		return c
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
func getStringClaim(claims JWTClaims, key string) string {
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
func getMapClaim(claims JWTClaims, key string) any {
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
func addExpirationToClaims(claims JWTClaims, ttl time.Duration) JWTClaims {
	switch c := claims.(type) {
	case map[string]any:
		newClaims := deepCopyMap(c)
		newClaims[JWTClaimExpiration] = time.Now().Add(ttl).Unix()
		return newClaims
	case HS256Claims:
		newClaims := deepCopyMap(c)
		newClaims[JWTClaimExpiration] = time.Now().Add(ttl).Unix()
		return HS256Claims(newClaims)
	default:
		return claims
	}
}

// addTypeToClaims adds type claim to map claims
func addTypeToClaims(claims JWTClaims, tokenType string) JWTClaims {
	switch c := claims.(type) {
	case map[string]any:
		newClaims := deepCopyMap(c)
		newClaims[JWTClaimType] = tokenType
		return newClaims
	case HS256Claims:
		newClaims := deepCopyMap(c)
		newClaims[JWTClaimType] = tokenType
		return HS256Claims(newClaims)
	default:
		return claims
	}
}

// extractSubject extracts the subject (sub) claim from claims
// Used during token refresh to identify the user for fetching fresh claims from database
func extractSubject(claims JWTClaims) string {
	return getStringClaim(claims, JWTClaimSubject)
}
