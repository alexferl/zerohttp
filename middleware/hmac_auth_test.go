package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
)

func TestHMACAuth_MissingAuthorization(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHMACAuth_InvalidFormat(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set(httpx.HeaderAuthorization, "invalid-format")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHMACAuth_InvalidAlgorithm(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		Algorithm: config.HMACSHA256,
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Try with SHA512 when expecting SHA256
	signer := NewHMACSignerWithAlgorithm("test-key", "test-secret-key-that-is-64-bytes-long-for-the-sha512-algorithm-use!!", config.HMACSHA512)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	if err := signer.SignRequest(req); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for algorithm mismatch, got %d", rr.Code)
	}
}

func TestHMACAuth_ValidRequest(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
	})

	var handlerCalled bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req)
	if err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Error("handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHMACAuth_WithBody(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"test":"data"}` {
			t.Errorf("body mismatch: got %s", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	body := bytes.NewReader([]byte(`{"test":"data"}`))
	req := httptest.NewRequest("POST", "/api/test", body)
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	err := signer.SignRequest(req)
	if err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHMACAuth_InvalidSignature(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Sign with wrong secret
	signer := NewHMACSigner("test-key", "wrong-secret")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	if err := signer.SignRequest(req); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid signature, got %d", rr.Code)
	}
}

func TestHMACAuth_UnknownAccessKey(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewHMACSigner("unknown-key", "test-secret-key-that-is-32-bytes-long!")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	if err := signer.SignRequest(req); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unknown key, got %d", rr.Code)
	}
}

func TestHMACAuth_ExpiredTimestamp(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		MaxSkew: 1 * time.Minute,
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")

	// Sign with old timestamp (10 minutes ago)
	oldTime := time.Now().UTC().Add(-10 * time.Minute)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	if err := signer.SignRequestWithTime(req, oldTime); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired timestamp, got %d", rr.Code)
	}
}

func TestHMACAuth_FutureTimestamp(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		MaxSkew:        1 * time.Minute,
		ClockSkewGrace: 30 * time.Second,
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")

	// Sign with future timestamp (10 minutes from now)
	futureTime := time.Now().UTC().Add(10 * time.Minute)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	if err := signer.SignRequestWithTime(req, futureTime); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for future timestamp, got %d", rr.Code)
	}
}

func TestHMACAuth_ExcludedPath(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		ExcludedPaths: []string{"/health", "/metrics"},
	})

	var handlerCalled bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Error("handler was not called for excluded path")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for excluded path, got %d", rr.Code)
	}
}

func TestHMACAuth_SHA384(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-48-bytes-long-for-sha384-algorithm-use"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		Algorithm: config.HMACSHA384,
	})

	var handlerCalled bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewHMACSignerWithAlgorithm("test-key", "test-secret-key-that-is-48-bytes-long-for-sha384-algorithm-use", config.HMACSHA384)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req)
	if err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Error("handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHMACAuth_SHA512(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-64-bytes-long-for-the-sha512-algorithm-use!!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		Algorithm: config.HMACSHA512,
	})

	var handlerCalled bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewHMACSignerWithAlgorithm("test-key", "test-secret-key-that-is-64-bytes-long-for-the-sha512-algorithm-use!!", config.HMACSHA512)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req)
	if err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Error("handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHMACAuth_QueryParameters(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
	})

	var handlerCalled bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		// Verify query params are still accessible
		if r.URL.Query().Get("foo") != "bar" {
			t.Errorf("query param mismatch")
		}
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	req := httptest.NewRequest(http.MethodGet, "/api/test?foo=bar&baz=qux", nil)
	err := signer.SignRequest(req)
	if err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Error("handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHMACAuth_MissingRequiredHeader(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		RequiredHeaders: []string{"host", "x-timestamp", "x-request-id"},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	if err := signer.SignRequest(req); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}
	// Remove the X-Request-Id header (which was never set)
	// Actually, the signer won't sign it if it's not present,
	// so this test validates that the middleware rejects requests
	// that don't have required headers

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Should be rejected because x-request-id is missing
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing required header, got %d", rr.Code)
	}
}

func TestHMACAuth_CustomErrorHandler(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	customCalled := false
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request) {
			customCalled = true
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("custom error"))
		},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !customCalled {
		t.Error("custom error handler was not called")
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 from custom handler, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "custom error") {
		t.Errorf("custom error response not found")
	}
}

func TestHMACAuth_AllowUnsignedPayload(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		AllowUnsignedPayload: true,
	})

	var handlerCalled bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	signer.SetAllowUnsignedPayload(true)
	req := httptest.NewRequest("POST", "/api/test", strings.NewReader("body"))
	if err := signer.SignRequest(req); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Error("handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHMACAuth_PanicWithoutCredentialStore(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic without CredentialStore")
		}
	}()

	HMACAuth(config.HMACAuthConfig{})
}

func TestParseAuthorizationHeader(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantError bool
	}{
		{
			name:      "valid header",
			header:    "HMAC-SHA256 Credential=test-key/2026-03-07T12:00:00Z, SignedHeaders=host;x-timestamp, Signature=abcd1234",
			wantError: false,
		},
		{
			name:      "missing algorithm",
			header:    "Credential=test-key/2026-03-07T12:00:00Z, SignedHeaders=host, Signature=abcd",
			wantError: true,
		},
		{
			name:      "wrong algorithm prefix",
			header:    "Bearer token123",
			wantError: true,
		},
		{
			name:      "missing credential",
			header:    "HMAC-SHA256 SignedHeaders=host, Signature=abcd",
			wantError: true,
		},
		{
			name:      "invalid credential format",
			header:    "HMAC-SHA256 Credential=test-key, SignedHeaders=host, Signature=abcd",
			wantError: true,
		},
		{
			name:      "missing signature",
			header:    "HMAC-SHA256 Credential=test-key/2026-03-07T12:00:00Z, SignedHeaders=host",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseAuthorizationHeader(tt.header, "X-Timestamp")
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if parsed == nil {
					t.Error("expected parsed auth, got nil")
				}
			}
		})
	}
}

func TestValidateTimestamp(t *testing.T) {
	tests := []struct {
		name      string
		timestamp time.Time
		maxSkew   time.Duration
		grace     time.Duration
		wantError bool
	}{
		{
			name:      "valid current time",
			timestamp: time.Now().UTC(),
			maxSkew:   5 * time.Minute,
			grace:     1 * time.Minute,
			wantError: false,
		},
		{
			name:      "expired",
			timestamp: time.Now().UTC().Add(-10 * time.Minute),
			maxSkew:   5 * time.Minute,
			grace:     1 * time.Minute,
			wantError: true,
		},
		{
			name:      "future time beyond grace",
			timestamp: time.Now().UTC().Add(10 * time.Minute),
			maxSkew:   5 * time.Minute,
			grace:     1 * time.Minute,
			wantError: true,
		},
		{
			name:      "within grace period future",
			timestamp: time.Now().UTC().Add(30 * time.Second),
			maxSkew:   5 * time.Minute,
			grace:     1 * time.Minute,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTimestamp(tt.timestamp, tt.maxSkew, tt.grace)
			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestComputeHMACSignature(t *testing.T) {
	secret := "test-secret-key-that-is-32-bytes-long!"
	canonicalRequest := "GET\n/api/test\n\nhost:example.com\nx-timestamp:2026-03-07T12:00:00Z\n\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	// Test SHA256
	sig256 := computeHMACSignature(secret, canonicalRequest, config.HMACSHA256)
	if len(sig256) != 32 { // SHA256 produces 32 bytes
		t.Errorf("expected 32 bytes for SHA256, got %d", len(sig256))
	}

	// Test SHA384
	sig384 := computeHMACSignature(secret, canonicalRequest, config.HMACSHA384)
	if len(sig384) != 48 { // SHA384 produces 48 bytes
		t.Errorf("expected 48 bytes for SHA384, got %d", len(sig384))
	}

	// Test SHA512
	sig512 := computeHMACSignature(secret, canonicalRequest, config.HMACSHA512)
	if len(sig512) != 64 { // SHA512 produces 64 bytes
		t.Errorf("expected 64 bytes for SHA512, got %d", len(sig512))
	}
}

func TestBuildCanonicalQueryString(t *testing.T) {
	tests := []struct {
		name   string
		values map[string][]string
		want   string
	}{
		{
			name:   "empty",
			values: map[string][]string{},
			want:   "",
		},
		{
			name:   "single param",
			values: map[string][]string{"foo": {"bar"}},
			want:   "foo=bar",
		},
		{
			name:   "multiple params sorted",
			values: map[string][]string{"b": {"2"}, "a": {"1"}},
			want:   "a=1&b=2",
		},
		{
			name:   "multiple values sorted",
			values: map[string][]string{"foo": {"c", "a", "b"}},
			want:   "foo=a&foo=b&foo=c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a signer just to access the method
			signer := NewHMACSigner("key", "secret")
			got := signer.buildCanonicalQueryString(tt.values)
			if got != tt.want {
				t.Errorf("buildCanonicalQueryString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetHMACAccessKeyID(t *testing.T) {
	tests := []struct {
		name          string
		setupContext  func(*http.Request)
		expectedKeyID string
	}{
		{
			name: "authenticated request",
			setupContext: func(req *http.Request) {
				// Simulate what the middleware does
				ctx := context.WithValue(req.Context(), HMACAccessKeyIDContextKey, "test-access-key")
				*req = *req.WithContext(ctx)
			},
			expectedKeyID: "test-access-key",
		},
		{
			name: "unauthenticated request",
			setupContext: func(req *http.Request) {
				// No context value set
			},
			expectedKeyID: "",
		},
		{
			name: "empty key id",
			setupContext: func(req *http.Request) {
				ctx := context.WithValue(req.Context(), HMACAccessKeyIDContextKey, "")
				*req = *req.WithContext(ctx)
			},
			expectedKeyID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			tt.setupContext(req)

			accessKeyID := GetHMACAccessKeyID(req)
			if accessKeyID != tt.expectedKeyID {
				t.Errorf("expected %q, got %q", tt.expectedKeyID, accessKeyID)
			}
		})
	}
}

func TestHMACAuth_ContextPropagation(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
	})

	var receivedAccessKeyID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAccessKeyID = GetHMACAccessKeyID(r)
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	if err := signer.SignRequest(req); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if receivedAccessKeyID != "test-key" {
		t.Errorf("expected access key ID %q, got %q", "test-key", receivedAccessKeyID)
	}
}

func TestHMACAuth_AuditLogging(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	var auditEvents []struct {
		accessKeyID string
		success     bool
		errType     string
	}

	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		AuditLogger: func(accessKeyID string, timestamp time.Time, success bool, errType string) {
			auditEvents = append(auditEvents, struct {
				accessKeyID string
				success     bool
				errType     string
			}{accessKeyID, success, errType})
		},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test 1: Missing auth
	req1 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	if len(auditEvents) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(auditEvents))
	}
	if auditEvents[0].success {
		t.Error("expected failed audit event for missing auth")
	}
	if auditEvents[0].errType != "missing_auth" {
		t.Errorf("expected errType 'missing_auth', got %q", auditEvents[0].errType)
	}

	// Test 2: Successful auth
	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	req2 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	if err := signer.SignRequest(req2); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if len(auditEvents) != 2 {
		t.Fatalf("expected 2 audit events, got %d", len(auditEvents))
	}
	if !auditEvents[1].success {
		t.Error("expected successful audit event")
	}
	if auditEvents[1].errType != "" {
		t.Errorf("expected empty errType for success, got %q", auditEvents[1].errType)
	}
	if auditEvents[1].accessKeyID != "test-key" {
		t.Errorf("expected access key ID 'test-key', got %q", auditEvents[1].accessKeyID)
	}

	// Test 3: Invalid credentials
	signer3 := NewHMACSigner("unknown-key", "test-secret-key-that-is-32-bytes-long!")
	req3 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	if err := signer3.SignRequest(req3); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)

	if len(auditEvents) != 3 {
		t.Fatalf("expected 3 audit events, got %d", len(auditEvents))
	}
	if auditEvents[2].success {
		t.Error("expected failed audit event for invalid credentials")
	}
	if auditEvents[2].errType != "invalid_credentials" {
		t.Errorf("expected errType 'invalid_credentials', got %q", auditEvents[2].errType)
	}
	if auditEvents[2].accessKeyID != "unknown-key" {
		t.Errorf("expected access key ID 'unknown-key', got %q", auditEvents[2].accessKeyID)
	}
}

func TestGetHMACError(t *testing.T) {
	tests := []struct {
		name         string
		setupContext func(*http.Request)
		expectedType string
		expectedNil  bool
	}{
		{
			name: "missing_auth error",
			setupContext: func(req *http.Request) {
				ctx := context.WithValue(req.Context(), HMACErrorContextKey, errMissingAuth)
				*req = *req.WithContext(ctx)
			},
			expectedNil: false,
		},
		{
			name: "signature_mismatch error",
			setupContext: func(req *http.Request) {
				ctx := context.WithValue(req.Context(), HMACErrorContextKey, errSignatureMismatch)
				*req = *req.WithContext(ctx)
			},
			expectedNil: false,
		},
		{
			name: "no error",
			setupContext: func(req *http.Request) {
				// No error in context
			},
			expectedType: "",
			expectedNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			tt.setupContext(req)

			err := GetHMACError(req)
			if tt.expectedNil {
				if err != nil {
					t.Errorf("expected nil error, got %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Type != tt.expectedType {
				t.Errorf("expected error type %q, got %q", tt.expectedType, err.Type)
			}
		})
	}
}

func TestHMACAuth_CustomErrorHandlerWithContext(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	var receivedError *HMACAuthError

	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request) {
			// Custom handler can access the error via context
			receivedError = GetHMACError(r)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("custom: " + receivedError.Type))
		},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test with missing auth - should trigger custom handler with error
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	if receivedError == nil {
		t.Fatal("expected error in custom handler, got nil")
	}
	if receivedError.Type != errMissingAuth.Type {
		t.Errorf("expected error type %q, got %q", errMissingAuth.Type, receivedError.Type)
	}
}

func TestHMACAuth_CustomErrorHandlerWithSignatureMismatch(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	var receivedError *HMACAuthError

	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request) {
			receivedError = GetHMACError(r)
			w.WriteHeader(http.StatusForbidden) // Custom status
			_, _ = w.Write([]byte("auth failed: " + receivedError.Title))
		},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test with wrong secret - should trigger signature mismatch
	signer := NewHMACSigner("test-key", "wrong-secret")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	if err := signer.SignRequest(req); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	rr := httptest.NewRecorder()
	receivedError = nil
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 from custom handler, got %d", rr.Code)
	}
	if receivedError == nil {
		t.Fatal("expected error in custom handler, got nil")
	}
	if receivedError.Type != errSignatureMismatch.Type {
		t.Errorf("expected signature mismatch, got %q", receivedError.Type)
	}
	if !strings.Contains(rr.Body.String(), "auth failed: Signature Mismatch") {
		t.Errorf("unexpected response body: %s", rr.Body.String())
	}
}

func TestHMACAuth_KeyRotation(t *testing.T) {
	// Simulate key rotation scenario with old and new secrets
	// Secrets must be at least 32 bytes for security
	oldSecret := "old-secret-key-for-test-32bytes!"
	newSecret := "new-secret-key-for-test-32bytes!"

	// Credential store returns both secrets during rotation
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if id == "test-key" {
				// During rotation, both old and new secrets are valid
				return []string{newSecret, oldSecret}
			}
			return nil
		},
	})

	var handlerCalled bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	// Test 1: Request signed with old secret should still work
	t.Run("old secret", func(t *testing.T) {
		handlerCalled = false
		oldSigner := NewHMACSigner("test-key", oldSecret)
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		if err := oldSigner.SignRequest(req); err != nil {
			t.Fatalf("failed to sign request: %v", err)
		}

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if !handlerCalled {
			t.Error("handler was not called with old secret")
		}
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 with old secret, got %d", rr.Code)
		}
	})

	// Test 2: Request signed with new secret should work
	t.Run("new secret", func(t *testing.T) {
		handlerCalled = false
		newSigner := NewHMACSigner("test-key", newSecret)
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		if err := newSigner.SignRequest(req); err != nil {
			t.Fatalf("failed to sign request: %v", err)
		}

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if !handlerCalled {
			t.Error("handler was not called with new secret")
		}
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 with new secret, got %d", rr.Code)
		}
	})

	// Test 3: Request with wrong secret should fail
	t.Run("wrong secret", func(t *testing.T) {
		wrongSigner := NewHMACSigner("test-key", "completely-wrong-secret")
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		if err := wrongSigner.SignRequest(req); err != nil {
			t.Fatalf("failed to sign request: %v", err)
		}

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 with wrong secret, got %d", rr.Code)
		}
	})
}

func TestHMACAuth_NoSecrets(t *testing.T) {
	// Test that empty secrets slice returns invalid credentials
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			return nil // no secrets for any key
		},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewHMACSigner("any-key", "this-secret-is-32-bytes-long!!")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	if err := signer.SignRequest(req); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unknown access key, got %d", rr.Code)
	}
}

func TestHMACAuth_ShortSecretRejected(t *testing.T) {
	// Test that secrets shorter than minimum length are rejected
	// This is a security measure to prevent brute-force attacks
	creds := map[string]string{"test-key": "short-secret"} // Only 12 bytes
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Sign with short secret - should be rejected by middleware
	signer := NewHMACSigner("test-key", "short-secret")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	if err := signer.SignRequest(req); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for short secret, got %d", rr.Code)
	}
}

func TestHMACAuth_ShortSecretRejected_AllAlgorithms(t *testing.T) {
	// Test minimum secret length enforcement for all HMAC algorithms
	// The middleware filters out secrets that don't meet the minimum length
	tests := []struct {
		name         string
		algorithm    config.HMACHashAlgorithm
		storeSecrets []string // secrets returned by credential store
		validSecret  string   // secret used to sign (must meet min length)
		shouldPass   bool
	}{
		{
			name:         "SHA256 with only short secrets in store",
			algorithm:    config.HMACSHA256,
			storeSecrets: []string{"short-secret-12", "another-short-15"},
			validSecret:  "this-is-exactly-32-bytes-!!!!!!!",
			shouldPass:   false,
		},
		{
			name:         "SHA256 with valid secret in store",
			algorithm:    config.HMACSHA256,
			storeSecrets: []string{"this-is-exactly-32-bytes-!!!!!!!"},
			validSecret:  "this-is-exactly-32-bytes-!!!!!!!",
			shouldPass:   true,
		},
		{
			name:         "SHA384 with only short secrets in store",
			algorithm:    config.HMACSHA384,
			storeSecrets: []string{"short-secret-12", "this-secret-is-exactly-31-bytes"},
			validSecret:  "this-is-exactly-48-bytes-for-sha384-tests!!!!!!!",
			shouldPass:   false,
		},
		{
			name:         "SHA384 with valid secret in store",
			algorithm:    config.HMACSHA384,
			storeSecrets: []string{"this-is-exactly-48-bytes-for-sha384-tests!!!!!!!"},
			validSecret:  "this-is-exactly-48-bytes-for-sha384-tests!!!!!!!",
			shouldPass:   true,
		},
		{
			name:         "SHA512 with only short secrets in store",
			algorithm:    config.HMACSHA512,
			storeSecrets: []string{"short-secret-12", "this-secret-is-exactly-47-bytes-long-enough!!"},
			validSecret:  "this-is-exactly-64-bytes-for-sha512-tests-in-middleware-!!!!!!!!",
			shouldPass:   false,
		},
		{
			name:         "SHA512 with valid secret in store",
			algorithm:    config.HMACSHA512,
			storeSecrets: []string{"this-is-exactly-64-bytes-for-sha512-tests-in-middleware-!!!!!!!!"},
			validSecret:  "this-is-exactly-64-bytes-for-sha512-tests-in-middleware-!!!!!!!!",
			shouldPass:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := HMACAuth(config.HMACAuthConfig{
				Algorithm: tt.algorithm,
				CredentialStore: func(id string) []string {
					if id == "test-key" {
						return tt.storeSecrets
					}
					return nil
				},
			})

			handlerCalled := false
			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			}))

			signer := NewHMACSignerWithAlgorithm("test-key", tt.validSecret, tt.algorithm)
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			if err := signer.SignRequest(req); err != nil {
				t.Fatalf("failed to sign request: %v", err)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if tt.shouldPass {
				if rr.Code != http.StatusOK {
					t.Errorf("expected 200 for valid secret length, got %d", rr.Code)
				}
				if !handlerCalled {
					t.Error("handler was not called for valid secret")
				}
			} else {
				if rr.Code != http.StatusUnauthorized {
					t.Errorf("expected 401 for short secret, got %d", rr.Code)
				}
				if handlerCalled {
					t.Error("handler was called for invalid secret")
				}
			}
		})
	}
}

func TestHMACAuth_ReplayAttack_Prevented(t *testing.T) {
	// Test that replay attacks are prevented via timestamp validation
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		MaxSkew:        1 * time.Minute,
		ClockSkewGrace: 30 * time.Second,
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")

	// Test 1: Request with timestamp exactly at skew boundary (should fail)
	t.Run("at_boundary", func(t *testing.T) {
		oldTime := time.Now().UTC().Add(-2 * time.Minute)
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		if err := signer.SignRequestWithTime(req, oldTime); err != nil {
			t.Fatalf("failed to sign request: %v", err)
		}

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for replayed request, got %d", rr.Code)
		}
	})

	// Test 2: Request with timestamp just inside window (should succeed)
	t.Run("just_inside_window", func(t *testing.T) {
		recentTime := time.Now().UTC().Add(-30 * time.Second)
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		if err := signer.SignRequestWithTime(req, recentTime); err != nil {
			t.Fatalf("failed to sign request: %v", err)
		}

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 for valid request, got %d", rr.Code)
		}
	})
}

func TestHMACAuth_HeaderTampering_Detected(t *testing.T) {
	// Test that tampering with signed headers invalidates the signature
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		RequiredHeaders: []string{"host", "x-timestamp", "x-request-id"},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	signer.SetHeadersToSign([]string{"host", "x-timestamp", "x-request-id"})

	// Sign a request with specific headers
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set(httpx.HeaderXRequestId, "original-request-id")
	if err := signer.SignRequest(req); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	// Tamper with the header after signing
	req.Header.Set(httpx.HeaderXRequestId, "tampered-request-id")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for tampered header, got %d", rr.Code)
	}
}

func TestHMACAuth_BodyTampering_Detected(t *testing.T) {
	// Test that tampering with request body invalidates the signature
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")

	// Sign a request with specific body
	originalBody := []byte(`{"amount": 100, "to": "alice"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/transfer", bytes.NewReader(originalBody))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	if err := signer.SignRequest(req); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	// Tamper with the body after signing
	tamperedBody := []byte(`{"amount": 9999, "to": "attacker"}`)
	req.Body = io.NopCloser(bytes.NewReader(tamperedBody))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for tampered body, got %d", rr.Code)
	}
}

func TestHMACAuth_QueryParamTampering_Detected(t *testing.T) {
	// Test that tampering with query parameters invalidates the signature
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")

	// Sign a request with specific query params
	req := httptest.NewRequest(http.MethodGet, "/api/data?user=alice&amount=100", nil)
	if err := signer.SignRequest(req); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	// Tamper with query params after signing
	req.URL.RawQuery = "user=alice&amount=9999"

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for tampered query params, got %d", rr.Code)
	}
}

func TestHMACAuth_MiddlewareChaining(t *testing.T) {
	// Test that HMAC auth works correctly when chained with other middleware
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}

	// Simulate another middleware that runs before HMAC auth
	addHeaderMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Chain-Test", "added-by-middleware")
			next.ServeHTTP(w, r)
		})
	}

	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
	})

	var handlerCalled bool
	var chainHeaderValue string
	handler := addHeaderMiddleware(mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		chainHeaderValue = r.Header.Get("X-Chain-Test")
		w.WriteHeader(http.StatusOK)
	})))

	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	if err := signer.SignRequest(req); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Error("handler was not called")
	}
	if chainHeaderValue != "added-by-middleware" {
		t.Errorf("expected chain middleware header, got %q", chainHeaderValue)
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestParseAuthorizationHeader_InvalidCases(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		wantErr string
	}{
		{
			name:    "not HMAC algorithm",
			header:  "Bearer token123 Credential=test/2026-03-07T12:00:00Z, SignedHeaders=host, Signature=abcd",
			wantErr: "invalid format: not HMAC algorithm",
		},
		{
			name:    "invalid base64 signature",
			header:  "HMAC-SHA256 Credential=test-key/2026-03-07T12:00:00Z, SignedHeaders=host, Signature=!!!invalid!!!",
			wantErr: "invalid signature encoding",
		},
		{
			name:    "missing credential",
			header:  "HMAC-SHA256 SignedHeaders=host, Signature=YWJjZA==",
			wantErr: "missing required fields",
		},
		{
			name:    "missing signed headers",
			header:  "HMAC-SHA256 Credential=test-key/2026-03-07T12:00:00Z, Signature=YWJjZA==",
			wantErr: "missing required fields",
		},
		{
			name:    "missing signature",
			header:  "HMAC-SHA256 Credential=test-key/2026-03-07T12:00:00Z, SignedHeaders=host",
			wantErr: "missing required fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseAuthorizationHeader(tt.header, "X-Timestamp")
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestComputeHMACSignature_DefaultAlgorithm(t *testing.T) {
	// Test that unknown algorithm defaults to SHA256
	secret := "test-secret-key-that-is-32-bytes-long!"
	canonicalRequest := "GET\n/api/test\n\nhost:example.com\n\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	// Use an invalid algorithm value
	sig := computeHMACSignature(secret, canonicalRequest, config.HMACHashAlgorithm("INVALID"))
	if len(sig) != 32 { // Should default to SHA256 (32 bytes)
		t.Errorf("expected 32 bytes for default SHA256, got %d", len(sig))
	}
}

func TestComputeBodyHash_DefaultAlgorithm(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)

	// Use an invalid algorithm value - should default to SHA256
	hash, err := computeBodyHash(req, config.HMACHashAlgorithm("INVALID"), 1024*1024)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(hash) != 64 { // SHA256 hex is 64 characters
		t.Errorf("expected 64 char hex for default SHA256, got %d", len(hash))
	}
}

func TestValidateTimestamp_FutureBeyondGrace(t *testing.T) {
	// Timestamp too far in the future (beyond grace period)
	future := time.Now().UTC().Add(10 * time.Minute)
	err := validateTimestamp(future, 5*time.Minute, 1*time.Minute)
	if err == nil {
		t.Error("expected error for future timestamp beyond grace")
	}
}

func TestValidateTimestamp_FutureWithinGrace(t *testing.T) {
	// Timestamp slightly in the future (within grace period)
	future := time.Now().UTC().Add(30 * time.Second)
	err := validateTimestamp(future, 5*time.Minute, 1*time.Minute)
	if err != nil {
		t.Errorf("expected no error for future timestamp within grace, got %v", err)
	}
}

func TestHMACAuth_InvalidBase64Signature(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create a request with invalid base64 in signature
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set(httpx.HeaderAuthorization, "HMAC-SHA256 Credential=test-key/2026-03-07T12:00:00Z, SignedHeaders=host;x-timestamp, Signature=!!!invalid!!!")
	req.Header.Set(httpx.HeaderXTimestamp, time.Now().UTC().Format(time.RFC3339))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid base64 signature, got %d", rr.Code)
	}
}

func TestHMACAuth_MaxBodySize(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		MaxBodySize: 100, // 100 bytes max
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create a request with body larger than MaxBodySize
	largeBody := strings.Repeat("a", 200)
	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(largeBody))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)

	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	err := signer.SignRequest(req)
	if err != nil {
		t.Fatalf("Signer error: %v", err)
	}

	// Check the body after signing
	bodyBytes, _ := io.ReadAll(req.Body)
	t.Logf("Body size after signing: %d", len(bodyBytes))
	req.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	t.Logf("Response code: %d, Body: %s", rr.Code, rr.Body.String())

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413 for body exceeding MaxBodySize, got %d", rr.Code)
	}
}

func TestHMACAuth_MaxBodySize_AllowedWithUnsignedPayload(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		MaxBodySize:          100, // 100 bytes max
		AllowUnsignedPayload: true,
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create a request with body larger than MaxBodySize but with unsigned payload
	largeBody := strings.Repeat("a", 200)
	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(largeBody))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)

	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	signer.SetAllowUnsignedPayload(true)
	if err := signer.SignRequest(req); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 with AllowUnsignedPayload, got %d", rr.Code)
	}
}

func TestHMACAuth_PresignedURL(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		AllowPresignedURLs: true,
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create a pre-signed URL
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/data", nil)
	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	presignedURL, err := signer.PresignURL(req, 5*time.Minute)
	if err != nil {
		t.Fatalf("failed to create presigned URL: %v", err)
	}

	// Parse the presigned URL and make a request
	parsedURL, _ := url.Parse(presignedURL)
	req2 := httptest.NewRequest(http.MethodGet, parsedURL.String(), nil)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req2)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for valid presigned URL, got %d", rr.Code)
	}
}

func TestHMACAuth_PresignedURL_NotAllowed(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		AllowPresignedURLs: false, // explicitly disabled (default)
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create a request with presigned URL params but no Authorization header
	req := httptest.NewRequest(http.MethodGet, "/api/data?X-HMAC-Algorithm=HMAC-SHA256&X-HMAC-Credential=test-key/2026-03-07T12:00:00Z&X-HMAC-SignedHeaders=host;x-timestamp&X-HMAC-Signature=abc123", nil)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when presigned URLs not allowed, got %d", rr.Code)
	}
}

func TestHMACAuth_PresignedURL_MissingParams(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		AllowPresignedURLs: true,
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "missing algorithm",
			query: "X-HMAC-Credential=test-key/2026-03-07T12:00:00Z&X-HMAC-SignedHeaders=host;x-timestamp&X-HMAC-Signature=abc",
		},
		{
			name:  "missing credential",
			query: "X-HMAC-Algorithm=HMAC-SHA256&X-HMAC-SignedHeaders=host;x-timestamp&X-HMAC-Signature=abc",
		},
		{
			name:  "missing signed headers",
			query: "X-HMAC-Algorithm=HMAC-SHA256&X-HMAC-Credential=test-key/2026-03-07T12:00:00Z&X-HMAC-Signature=abc",
		},
		{
			name:  "missing signature",
			query: "X-HMAC-Algorithm=HMAC-SHA256&X-HMAC-Credential=test-key/2026-03-07T12:00:00Z&X-HMAC-SignedHeaders=host;x-timestamp",
		},
		{
			name:  "invalid credential format",
			query: "X-HMAC-Algorithm=HMAC-SHA256&X-HMAC-Credential=no-slash-here&X-HMAC-SignedHeaders=host;x-timestamp&X-HMAC-Signature=abc",
		},
		{
			name:  "invalid timestamp format",
			query: "X-HMAC-Algorithm=HMAC-SHA256&X-HMAC-Credential=test-key/invalid-timestamp&X-HMAC-SignedHeaders=host;x-timestamp&X-HMAC-Signature=abc",
		},
		{
			name:  "invalid base64 signature",
			query: "X-HMAC-Algorithm=HMAC-SHA256&X-HMAC-Credential=test-key/2026-03-07T12:00:00Z&X-HMAC-SignedHeaders=host;x-timestamp&X-HMAC-Signature=!!!invalid!!!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/data?"+tt.query, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Errorf("expected 401 for %s, got %d", tt.name, rr.Code)
			}
		})
	}
}

func TestHMACAuth_PresignedURL_InvalidSignature(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		AllowPresignedURLs: true,
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create a request with invalid presigned URL params
	req := httptest.NewRequest(http.MethodGet, "/api/data?X-HMAC-Algorithm=HMAC-SHA256&X-HMAC-Credential=test-key/"+time.Now().UTC().Format(time.RFC3339)+"&X-HMAC-SignedHeaders=host;x-timestamp&X-HMAC-Signature=invalid", nil)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid presigned URL signature, got %d", rr.Code)
	}
}

func TestValidatePresignedURLTimestamp_Expired(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		AllowPresignedURLs: true,
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create a request with expired presigned URL (1 hour ago)
	expiredTime := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	req := httptest.NewRequest(http.MethodGet, "/api/data?X-HMAC-Algorithm=HMAC-SHA256&X-HMAC-Credential=test-key/"+expiredTime+"&X-HMAC-SignedHeaders=host;x-timestamp&X-HMAC-Signature=abc123", nil)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired presigned URL, got %d", rr.Code)
	}
}

func TestHMACAuth_NilCredentialStore(t *testing.T) {
	// Test that nil CredentialStore causes panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil CredentialStore")
		}
	}()

	_ = HMACAuth(config.HMACAuthConfig{
		CredentialStore: nil,
	})
}

// TestHMACAuth_PresignedURL_NoHeaderModification verifies that the middleware
// does not modify the request headers when processing presigned URLs
func TestHMACAuth_PresignedURL_NoHeaderModification(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		AllowPresignedURLs: true,
	})

	var capturedRequest *http.Request
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		w.WriteHeader(http.StatusOK)
	}))

	// Create a pre-signed URL
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/data", nil)
	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	presignedURL, err := signer.PresignURL(req, 5*time.Minute)
	if err != nil {
		t.Fatalf("failed to create presigned URL: %v", err)
	}

	// Parse the presigned URL and make a request
	parsedURL, _ := url.Parse(presignedURL)
	req2 := httptest.NewRequest(http.MethodGet, parsedURL.String(), nil)

	// Capture the original headers before the request is processed
	originalTimestampHeader := req2.Header.Get("X-Timestamp")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req2)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for valid presigned URL, got %d", rr.Code)
	}

	// Verify that the X-Timestamp header was NOT modified by the middleware
	if capturedRequest == nil {
		t.Fatal("capturedRequest is nil")
	}
	finalTimestampHeader := capturedRequest.Header.Get("X-Timestamp")
	if finalTimestampHeader != originalTimestampHeader {
		t.Errorf("X-Timestamp header was modified by middleware: original=%q, final=%q",
			originalTimestampHeader, finalTimestampHeader)
	}
}

func TestHMACAuth_IncludedPaths(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		IncludedPaths: []string{"/api/", "/admin"},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")

	// Test allowed path - should require auth
	req1 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	if err := signer.SignRequest(req1); err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("expected 200 for allowed path with valid auth, got %d", rr1.Code)
	}

	// Test non-allowed path - should skip auth
	req2 := httptest.NewRequest(http.MethodGet, "/public", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("expected 200 for non-allowed path, got %d", rr2.Code)
	}

	// Test non-allowed path with missing auth - should still pass
	req3 := httptest.NewRequest(http.MethodGet, "/other", nil)
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusOK {
		t.Errorf("expected 200 for non-allowed path without auth, got %d", rr3.Code)
	}
}

func TestHMACAuth_BothExcludedAndIncludedPathsPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when both ExcludedPaths and IncludedPaths are set")
		}
	}()

	_ = HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string { return nil },
		ExcludedPaths:   []string{"/health"},
		IncludedPaths:   []string{"/api"},
	})
}
