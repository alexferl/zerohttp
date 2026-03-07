package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/alexferl/zerohttp/config"
)

// HMACAuthError represents an HMAC authentication error with RFC 9457 Problem Details
type HMACAuthError struct {
	Type   string
	Title  string
	Status int
	Detail string
}

// Common HMAC auth errors
var (
	errMissingAuth = &HMACAuthError{
		Type:   "urn:ietf:rfc:9457:hmac-auth:missing-header",
		Title:  "Missing Authorization Header",
		Status: http.StatusUnauthorized,
		Detail: "Request is missing the Authorization header",
	}
	errInvalidFormat = &HMACAuthError{
		Type:   "urn:ietf:rfc:9457:hmac-auth:invalid-format",
		Title:  "Invalid Authorization Format",
		Status: http.StatusUnauthorized,
		Detail: "Authorization header does not match expected format",
	}
	errInvalidCredentials = &HMACAuthError{
		Type:   "urn:ietf:rfc:9457:hmac-auth:invalid-credentials",
		Title:  "Invalid Credentials",
		Status: http.StatusUnauthorized,
		Detail: "The access key ID was not found or signature is invalid",
	}
	errRequestExpired = &HMACAuthError{
		Type:   "urn:ietf:rfc:9457:hmac-auth:request-expired",
		Title:  "Request Expired",
		Status: http.StatusUnauthorized,
		Detail: "Request timestamp is outside the valid time window",
	}
	errSignatureMismatch = &HMACAuthError{
		Type:   "urn:ietf:rfc:9457:hmac-auth:signature-mismatch",
		Title:  "Signature Mismatch",
		Status: http.StatusUnauthorized,
		Detail: "The provided signature does not match the computed signature",
	}
	errMissingHeader = &HMACAuthError{
		Type:   "urn:ietf:rfc:9457:hmac-auth:missing-required-header",
		Title:  "Missing Required Header",
		Status: http.StatusUnauthorized,
		Detail: "Request is missing a required header for signature verification",
	}
)

// HMACAuthContextKey is the context key type for HMAC auth info
type HMACAuthContextKey string

const (
	// HMACAccessKeyIDContextKey holds the verified access key ID in the request context
	HMACAccessKeyIDContextKey HMACAuthContextKey = "hmac_access_key_id"
	// HMACErrorContextKey holds the HMACAuthError in the request context (only set on auth failures)
	HMACErrorContextKey HMACAuthContextKey = "hmac_error"
)

// GetHMACAccessKeyID retrieves the verified access key ID from the request context.
// Returns empty string if the request was not authenticated with HMAC.
func GetHMACAccessKeyID(r *http.Request) string {
	if accessKeyID, ok := r.Context().Value(HMACAccessKeyIDContextKey).(string); ok {
		return accessKeyID
	}
	return ""
}

// GetHMACError retrieves the HMAC authentication error from the request context.
// Returns nil if there was no authentication error (e.g., success or not an HMAC-authenticated request).
func GetHMACError(r *http.Request) *HMACAuthError {
	if err, ok := r.Context().Value(HMACErrorContextKey).(*HMACAuthError); ok {
		return err
	}
	return nil
}

// parsedAuth represents the parsed Authorization header
type parsedAuth struct {
	Algorithm   string
	AccessKeyID string
	Timestamp   time.Time
	Headers     []string
	Signature   []byte
	IsPresigned bool // true if auth came from URL query params (presigned URL)
}

// HMACAuth creates HMAC request signing authentication middleware
func HMACAuth(cfg ...config.HMACAuthConfig) func(http.Handler) http.Handler {
	c := config.DefaultHMACAuthConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}

	if c.CredentialStore == nil {
		panic("HMACAuth: CredentialStore is required")
	}

	if c.Algorithm == "" {
		c.Algorithm = config.HMACSHA256
	}
	if c.MaxSkew == 0 {
		c.MaxSkew = config.DefaultHMACAuthConfig.MaxSkew
	}
	if c.ClockSkewGrace == 0 {
		c.ClockSkewGrace = config.DefaultHMACAuthConfig.ClockSkewGrace
	}
	if c.AuthHeaderName == "" {
		c.AuthHeaderName = config.DefaultHMACAuthConfig.AuthHeaderName
	}
	if c.TimestampHeader == "" {
		c.TimestampHeader = config.DefaultHMACAuthConfig.TimestampHeader
	}
	if c.RequiredHeaders == nil {
		c.RequiredHeaders = config.DefaultHMACAuthConfig.RequiredHeaders
	}
	if c.OptionalHeaders == nil {
		c.OptionalHeaders = config.DefaultHMACAuthConfig.OptionalHeaders
	}
	if c.ExemptPaths == nil {
		c.ExemptPaths = config.DefaultHMACAuthConfig.ExemptPaths
	}
	if c.MaxBodySize == 0 {
		c.MaxBodySize = config.DefaultHMACAuthConfig.MaxBodySize
	}

	errorHandler := c.ErrorHandler
	if errorHandler == nil {
		errorHandler = defaultHMACErrorHandler
	}

	auditLogger := c.AuditLogger

	requiredHeaders := make([]string, len(c.RequiredHeaders))
	for i, h := range c.RequiredHeaders {
		requiredHeaders[i] = strings.ToLower(strings.TrimSpace(h))
	}
	optionalHeaders := make([]string, len(c.OptionalHeaders))
	for i, h := range c.OptionalHeaders {
		optionalHeaders[i] = strings.ToLower(strings.TrimSpace(h))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, exemptPath := range c.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			var parsed *parsedAuth
			var err error

			authHeader := r.Header.Get(c.AuthHeaderName)
			if authHeader == "" {
				// Check for pre-signed URL parameters if enabled
				if c.AllowPresignedURLs {
					parsed, err = parsePresignedURLParams(r)
					if err != nil {
						if auditLogger != nil {
							auditLogger("", time.Time{}, false, "invalid_presigned_url")
						}
						handleHMACError(w, r, errInvalidFormat, errorHandler)
						return
					}
				} else {
					if auditLogger != nil {
						auditLogger("", time.Time{}, false, "missing_auth")
					}
					handleHMACError(w, r, errMissingAuth, errorHandler)
					return
				}
			} else {
				parsed, err = parseAuthorizationHeader(authHeader, c.TimestampHeader)
			}
			if err != nil {
				if auditLogger != nil {
					auditLogger("", time.Time{}, false, "invalid_format")
				}
				handleHMACError(w, r, errInvalidFormat, errorHandler)
				return
			}

			expectedAlgo := "HMAC-" + string(c.Algorithm)
			if !strings.EqualFold(parsed.Algorithm, expectedAlgo) {
				if auditLogger != nil {
					auditLogger(parsed.AccessKeyID, parsed.Timestamp, false, "algorithm_mismatch")
				}
				handleHMACError(w, r, errInvalidFormat, errorHandler)
				return
			}

			for _, header := range requiredHeaders {
				if header == "host" {
					// Host is special - comes from r.Host
					if r.Host == "" {
						if auditLogger != nil {
							auditLogger(parsed.AccessKeyID, parsed.Timestamp, false, "missing_header")
						}
						handleHMACError(w, r, &HMACAuthError{
							Type:   errMissingHeader.Type,
							Title:  errMissingHeader.Title,
							Status: errMissingHeader.Status,
							Detail: "Missing required header: host",
						}, errorHandler)
						return
					}
					continue
				}
				if r.Header.Get(header) == "" {
					if auditLogger != nil {
						auditLogger(parsed.AccessKeyID, parsed.Timestamp, false, "missing_header")
					}
					handleHMACError(w, r, &HMACAuthError{
						Type:   errMissingHeader.Type,
						Title:  errMissingHeader.Title,
						Status: errMissingHeader.Status,
						Detail: "Missing required header: " + header,
					}, errorHandler)
					return
				}
			}

			// Validate timestamp - presigned URLs use the timestamp as expiration time
			if parsed.IsPresigned {
				if err := validatePresignedURLTimestamp(parsed.Timestamp, c.ClockSkewGrace); err != nil {
					if auditLogger != nil {
						auditLogger(parsed.AccessKeyID, parsed.Timestamp, false, "request_expired")
					}
					handleHMACError(w, r, errRequestExpired, errorHandler)
					return
				}
			} else {
				if err := validateTimestamp(parsed.Timestamp, c.MaxSkew, c.ClockSkewGrace); err != nil {
					if auditLogger != nil {
						auditLogger(parsed.AccessKeyID, parsed.Timestamp, false, "request_expired")
					}
					handleHMACError(w, r, errRequestExpired, errorHandler)
					return
				}
			}

			secretKeys := c.CredentialStore(parsed.AccessKeyID)
			if len(secretKeys) == 0 {
				if auditLogger != nil {
					auditLogger(parsed.AccessKeyID, parsed.Timestamp, false, "invalid_credentials")
				}
				handleHMACError(w, r, errInvalidCredentials, errorHandler)
				return
			}

			var bodyHash string
			if c.AllowUnsignedPayload {
				bodyHash = "UNSIGNED-PAYLOAD"
			} else {
				bodyHash, err = computeBodyHash(r, c.Algorithm, c.MaxBodySize)
				if err != nil {
					if auditLogger != nil {
						auditLogger(parsed.AccessKeyID, parsed.Timestamp, false, "body_too_large")
					}
					handleHMACError(w, r, &HMACAuthError{
						Type:   "urn:ietf:rfc:9457:hmac-auth:payload-too-large",
						Title:  "Request Body Too Large",
						Status: http.StatusRequestEntityTooLarge,
						Detail: "Request body exceeds maximum size for HMAC verification",
					}, errorHandler)
					return
				}
			}

			canonicalRequest := buildCanonicalRequest(r, parsed, requiredHeaders, optionalHeaders, bodyHash, c.TimestampHeader)

			// Try each secret key - supports key rotation with multiple valid keys
			authenticated := false
			for _, secretKey := range secretKeys {
				expectedSig := computeHMACSignature(secretKey, canonicalRequest, c.Algorithm)
				if subtle.ConstantTimeCompare(parsed.Signature, expectedSig) == 1 {
					authenticated = true
					break
				}
			}

			if !authenticated {
				if auditLogger != nil {
					auditLogger(parsed.AccessKeyID, parsed.Timestamp, false, "signature_mismatch")
				}
				handleHMACError(w, r, errSignatureMismatch, errorHandler)
				return
			}

			// Success - log audit event
			if auditLogger != nil {
				auditLogger(parsed.AccessKeyID, parsed.Timestamp, true, "")
			}

			// Add access key ID to context for handlers to use
			ctx := context.WithValue(r.Context(), HMACAccessKeyIDContextKey, parsed.AccessKeyID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// parseAuthorizationHeader parses the Authorization header
// Format: HMAC-<ALGORITHM> Credential=<access-key-id>/<timestamp>, SignedHeaders=<headers>, Signature=<base64>
func parseAuthorizationHeader(header, timestampHeader string) (*parsedAuth, error) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid format: missing algorithm")
	}

	algo := parts[0]
	if !strings.HasPrefix(algo, "HMAC-") {
		return nil, errors.New("invalid format: not HMAC algorithm")
	}

	rest := parts[1]
	result := &parsedAuth{
		Algorithm: algo,
	}

	// Split by comma, but handle potential spaces
	pairs := splitAuthPairs(rest)

	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch key {
		case "Credential":
			// Format: access-key-id/timestamp
			credParts := strings.SplitN(value, "/", 2)
			if len(credParts) != 2 {
				return nil, errors.New("invalid credential format")
			}
			result.AccessKeyID = credParts[0]
			ts, err := time.Parse(time.RFC3339, credParts[1])
			if err != nil {
				// Try other common formats
				ts, err = time.Parse("2006-01-02T15:04:05Z", credParts[1])
				if err != nil {
					return nil, errors.New("invalid timestamp format")
				}
			}
			result.Timestamp = ts
		case "SignedHeaders":
			result.Headers = strings.Split(value, ";")
		case "Signature":
			sig, err := base64.StdEncoding.DecodeString(value)
			if err != nil {
				return nil, errors.New("invalid signature encoding")
			}
			result.Signature = sig
		}
	}

	if result.AccessKeyID == "" || result.Signature == nil || len(result.Headers) == 0 {
		return nil, errors.New("missing required fields")
	}

	return result, nil
}

// parsePresignedURLParams parses HMAC authentication parameters from URL query string
// Format: X-HMAC-Algorithm=HMAC-SHA256, X-HMAC-Credential=<access-key-id>/<timestamp>,
// X-HMAC-SignedHeaders=<headers>, X-HMAC-Signature=<base64>
func parsePresignedURLParams(r *http.Request) (*parsedAuth, error) {
	q := r.URL.Query()

	algo := q.Get("X-HMAC-Algorithm")
	if algo == "" {
		return nil, errors.New("missing algorithm parameter")
	}

	credential := q.Get("X-HMAC-Credential")
	if credential == "" {
		return nil, errors.New("missing credential parameter")
	}

	signedHeaders := q.Get("X-HMAC-SignedHeaders")
	if signedHeaders == "" {
		return nil, errors.New("missing signed headers parameter")
	}

	signature := q.Get("X-HMAC-Signature")
	if signature == "" {
		return nil, errors.New("missing signature parameter")
	}

	// Parse credential: access-key-id/timestamp
	credParts := strings.SplitN(credential, "/", 2)
	if len(credParts) != 2 {
		return nil, errors.New("invalid credential format")
	}

	accessKeyID := credParts[0]
	ts, err := time.Parse(time.RFC3339, credParts[1])
	if err != nil {
		// Try other common formats
		ts, err = time.Parse("2006-01-02T15:04:05Z", credParts[1])
		if err != nil {
			return nil, errors.New("invalid timestamp format")
		}
	}

	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return nil, errors.New("invalid signature encoding")
	}

	// For presigned URLs, set the X-Timestamp header from the credential timestamp
	// so that required header checks pass
	r.Header.Set("X-Timestamp", ts.Format(time.RFC3339))

	return &parsedAuth{
		Algorithm:   algo,
		AccessKeyID: accessKeyID,
		Timestamp:   ts,
		Headers:     strings.Split(signedHeaders, ";"),
		Signature:   sig,
		IsPresigned: true,
	}, nil
}

// splitAuthPairs splits authorization header value by comma while respecting potential complexity
func splitAuthPairs(s string) []string {
	var parts []string
	var current strings.Builder

	for _, ch := range s {
		switch ch {
		case ',':
			parts = append(parts, current.String())
			current.Reset()
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// validatePresignedURLTimestamp checks if the presigned URL has expired
// For presigned URLs, the timestamp is the expiration time
func validatePresignedURLTimestamp(expiration time.Time, grace time.Duration) error {
	now := time.Now().UTC()
	// URL is valid if now is before expiration + grace
	if now.After(expiration.Add(grace)) {
		return errors.New("presigned URL has expired")
	}
	return nil
}

// validateTimestamp checks if the request timestamp is within the allowed skew window
func validateTimestamp(ts time.Time, maxSkew, grace time.Duration) error {
	now := time.Now().UTC()
	diff := now.Sub(ts)
	if diff < 0 {
		diff = -diff
	}

	allowedSkew := maxSkew + grace
	if diff > allowedSkew {
		return errors.New("timestamp outside valid window")
	}

	if ts.After(now.Add(grace)) {
		return errors.New("timestamp in future")
	}

	return nil
}

// computeBodyHash reads the body and computes its hash
// Returns an error if the body exceeds maxBodySize
func computeBodyHash(r *http.Request, algo config.HMACHashAlgorithm, maxBodySize int64) (string, error) {
	var h hash.Hash
	switch algo {
	case config.HMACSHA256:
		h = sha256.New()
	case config.HMACSHA384:
		h = sha512.New384()
	case config.HMACSHA512:
		h = sha512.New()
	default:
		h = sha256.New()
	}

	if r.Body != nil {
		// Limit body read to maxBodySize + 1 to detect overflow
		limitedReader := io.LimitReader(r.Body, maxBodySize+1)
		body, err := io.ReadAll(limitedReader)
		if err != nil {
			return "", err
		}

		// Check if body exceeded max size
		if int64(len(body)) > maxBodySize {
			return "", errors.New("request body too large")
		}

		h.Write(body)
		// Restore body for next handlers
		r.Body = io.NopCloser(strings.NewReader(string(body)))
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// buildCanonicalRequest creates the canonical request string for signing
func buildCanonicalRequest(
	r *http.Request,
	parsed *parsedAuth,
	requiredHeaders []string,
	optionalHeaders []string,
	bodyHash string,
	timestampHeader string,
) string {
	var b strings.Builder

	b.WriteString(strings.ToUpper(r.Method))
	b.WriteByte('\n')

	b.WriteString(url.PathEscape(r.URL.Path))
	b.WriteByte('\n')

	// For presigned URLs, exclude HMAC-related query params from canonical request
	query := r.URL.Query()
	if parsed.IsPresigned {
		query.Del("X-HMAC-Algorithm")
		query.Del("X-HMAC-Credential")
		query.Del("X-HMAC-SignedHeaders")
		query.Del("X-HMAC-Signature")
	}
	b.WriteString(buildCanonicalQueryString(query))
	b.WriteByte('\n')

	signedHeaders := buildSignedHeaders(r, parsed.Headers, timestampHeader)
	b.WriteString(signedHeaders)
	b.WriteString("\n\n") // Two newlines: end headers section + blank line

	b.WriteString(bodyHash)

	return b.String()
}

// buildCanonicalQueryString builds the canonical query string (sorted by key)
func buildCanonicalQueryString(values url.Values) string {
	if len(values) == 0 {
		return ""
	}

	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		vals := values[k]
		sort.Strings(vals)
		for _, v := range vals {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}

	return strings.Join(parts, "&")
}

// buildSignedHeaders creates the canonical headers string
func buildSignedHeaders(r *http.Request, headers []string, timestampHeader string) string {
	var parts []string

	for _, h := range headers {
		h = strings.ToLower(strings.TrimSpace(h))
		var value string

		if h == "host" {
			value = r.Host
		} else {
			value = r.Header.Get(h)
		}

		value = strings.TrimSpace(value)
		parts = append(parts, h+":"+value)
	}

	return strings.Join(parts, "\n")
}

// computeHMACSignature computes the HMAC signature
func computeHMACSignature(secretKey, canonicalRequest string, algo config.HMACHashAlgorithm) []byte {
	var h hash.Hash
	switch algo {
	case config.HMACSHA256:
		h = hmac.New(sha256.New, []byte(secretKey))
	case config.HMACSHA384:
		h = hmac.New(sha512.New384, []byte(secretKey))
	case config.HMACSHA512:
		h = hmac.New(sha512.New, []byte(secretKey))
	default:
		h = hmac.New(sha256.New, []byte(secretKey))
	}

	stringToSign := "HMAC-" + string(algo) + "\n" + canonicalRequest
	h.Write([]byte(stringToSign))
	return h.Sum(nil)
}

// handleHMACError sends an error response
func handleHMACError(w http.ResponseWriter, r *http.Request, hmacErr *HMACAuthError, handler http.HandlerFunc) {
	// Add error to context so custom handlers can access it
	ctx := context.WithValue(r.Context(), HMACErrorContextKey, hmacErr)
	r = r.WithContext(ctx)

	if handler != nil {
		handler(w, r)
		return
	}
	defaultHMACErrorHandler(w, r)
}

// defaultHMACErrorHandler is the default error handler
func defaultHMACErrorHandler(w http.ResponseWriter, r *http.Request) {
	// Get error from context
	hmacErr := GetHMACError(r)
	if hmacErr == nil {
		hmacErr = errInvalidCredentials
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(hmacErr.Status)

	// Simple JSON error (RFC 9457 style)
	response := fmt.Sprintf(`{"type":"%s","title":"%s","status":%d,"detail":"%s"}`,
		hmacErr.Type, hmacErr.Title, hmacErr.Status, hmacErr.Detail)
	_, _ = w.Write([]byte(response))
}
