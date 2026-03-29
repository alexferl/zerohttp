package hmacauth

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

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestHMACAuth_MissingAuthorization(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestHMACAuth_InvalidFormat(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestHMACAuth_InvalidAlgorithm(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		Algorithm: SHA256,
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Try with SHA512 when expecting SHA256
	signer := NewSignerWithAlgorithm("test-key", "test-secret-key-that-is-64-bytes-long-for-the-sha512-algorithm-use!!", SHA512)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestHMACAuth_ValidRequest(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertTrue(t, handlerCalled)
	zhtest.AssertEqual(t, http.StatusOK, rr.Code)
}

func TestHMACAuth_WithBody(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		zhtest.AssertEqual(t, `{"test":"data"}`, string(body))
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	body := bytes.NewReader([]byte(`{"test":"data"}`))
	req := httptest.NewRequest("POST", "/api/test", body)
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusOK, rr.Code)
}

func TestHMACAuth_InvalidSignature(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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
	signer := NewSigner("test-key", "wrong-secret")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestHMACAuth_UnknownAccessKey(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	signer := NewSigner("unknown-key", "test-secret-key-that-is-32-bytes-long!")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestHMACAuth_ExpiredTimestamp(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")

	// Sign with old timestamp (10 minutes ago)
	oldTime := time.Now().UTC().Add(-10 * time.Minute)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequestWithTime(req, oldTime)
	zhtest.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestHMACAuth_FutureTimestamp(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")

	// Sign with future timestamp (10 minutes from now)
	futureTime := time.Now().UTC().Add(10 * time.Minute)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequestWithTime(req, futureTime)
	zhtest.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestHMACAuth_ExcludedPath(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	zhtest.AssertTrue(t, handlerCalled)
	zhtest.AssertEqual(t, http.StatusOK, rr.Code)
}

func TestHMACAuth_SHA384(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-48-bytes-long-for-sha384-algorithm-use"}
	mw := New(Config{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		Algorithm: SHA384,
	})

	var handlerCalled bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewSignerWithAlgorithm("test-key", "test-secret-key-that-is-48-bytes-long-for-sha384-algorithm-use", SHA384)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertTrue(t, handlerCalled)
	zhtest.AssertEqual(t, http.StatusOK, rr.Code)
}

func TestHMACAuth_SHA512(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-64-bytes-long-for-the-sha512-algorithm-use!!"}
	mw := New(Config{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		Algorithm: SHA512,
	})

	var handlerCalled bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewSignerWithAlgorithm("test-key", "test-secret-key-that-is-64-bytes-long-for-the-sha512-algorithm-use!!", SHA512)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertTrue(t, handlerCalled)
	zhtest.AssertEqual(t, http.StatusOK, rr.Code)
}

func TestHMACAuth_QueryParameters(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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
		zhtest.AssertEqual(t, "bar", r.URL.Query().Get("foo"))
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	req := httptest.NewRequest(http.MethodGet, "/api/test?foo=bar&baz=qux", nil)
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertTrue(t, handlerCalled)
	zhtest.AssertEqual(t, http.StatusOK, rr.Code)
}

func TestHMACAuth_MissingRequiredHeader(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)
	// Remove the X-Request-Id header (which was never set)
	// Actually, the signer won't sign it if it's not present,
	// so this test validates that the middleware rejects requests
	// that don't have required headers

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Should be rejected because x-request-id is missing
	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestHMACAuth_CustomErrorHandler(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	customCalled := false
	mw := New(Config{
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

	zhtest.AssertTrue(t, customCalled)
	zhtest.AssertEqual(t, http.StatusForbidden, rr.Code)
	zhtest.AssertContains(t, rr.Body.String(), "custom error")
}

func TestHMACAuth_AllowUnsignedPayload(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	signer.SetAllowUnsignedPayload(true)
	req := httptest.NewRequest("POST", "/api/test", strings.NewReader("body"))
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertTrue(t, handlerCalled)
	zhtest.AssertEqual(t, http.StatusOK, rr.Code)
}

func TestHMACAuth_PanicWithoutCredentialStore(t *testing.T) {
	zhtest.AssertPanic(t, func() {
		New(Config{})
	})
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
				zhtest.AssertError(t, err)
			} else {
				zhtest.AssertNoError(t, err)
				zhtest.AssertNotNil(t, parsed)
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
			if tt.wantError {
				zhtest.AssertError(t, err)
			} else {
				zhtest.AssertNoError(t, err)
			}
		})
	}
}

func TestComputeHMACSignature(t *testing.T) {
	secret := "test-secret-key-that-is-32-bytes-long!"
	canonicalRequest := "GET\n/api/test\n\nhost:example.com\nx-timestamp:2026-03-07T12:00:00Z\n\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	// Test SHA256
	sig256 := computeHMACSignature(secret, canonicalRequest, SHA256)
	zhtest.AssertEqual(t, 32, len(sig256)) // SHA256 produces 32 bytes

	// Test SHA384
	sig384 := computeHMACSignature(secret, canonicalRequest, SHA384)
	zhtest.AssertEqual(t, 48, len(sig384)) // SHA384 produces 48 bytes

	// Test SHA512
	sig512 := computeHMACSignature(secret, canonicalRequest, SHA512)
	zhtest.AssertEqual(t, 64, len(sig512)) // SHA512 produces 64 bytes
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
			signer := NewSigner("key", "secret")
			got := signer.buildCanonicalQueryString(tt.values)
			zhtest.AssertEqual(t, tt.want, got)
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
				ctx := context.WithValue(req.Context(), AccessKeyIDContextKey, "test-access-key")
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
				ctx := context.WithValue(req.Context(), AccessKeyIDContextKey, "")
				*req = *req.WithContext(ctx)
			},
			expectedKeyID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			tt.setupContext(req)

			accessKeyID := GetAccessKeyID(req)
			zhtest.AssertEqual(t, tt.expectedKeyID, accessKeyID)
		})
	}
}

func TestHMACAuth_ContextPropagation(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
	})

	var receivedAccessKeyID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAccessKeyID = GetAccessKeyID(r)
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusOK, rr.Code)
	zhtest.AssertEqual(t, "test-key", receivedAccessKeyID)
}

func TestHMACAuth_AuditLogging(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	var auditEvents []struct {
		accessKeyID string
		success     bool
		errType     string
	}

	mw := New(Config{
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

	zhtest.AssertEqual(t, 1, len(auditEvents))
	zhtest.AssertFalse(t, auditEvents[0].success)
	zhtest.AssertEqual(t, "missing_auth", auditEvents[0].errType)

	// Test 2: Successful auth
	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	req2 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req2)
	zhtest.AssertNoError(t, err)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	zhtest.AssertEqual(t, 2, len(auditEvents))
	zhtest.AssertTrue(t, auditEvents[1].success)
	zhtest.AssertEqual(t, "", auditEvents[1].errType)
	zhtest.AssertEqual(t, "test-key", auditEvents[1].accessKeyID)

	// Test 3: Invalid credentials
	signer3 := NewSigner("unknown-key", "test-secret-key-that-is-32-bytes-long!")
	req3 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err = signer3.SignRequest(req3)
	zhtest.AssertNoError(t, err)
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)

	zhtest.AssertEqual(t, 3, len(auditEvents))
	zhtest.AssertFalse(t, auditEvents[2].success)
	zhtest.AssertEqual(t, "invalid_credentials", auditEvents[2].errType)
	zhtest.AssertEqual(t, "unknown-key", auditEvents[2].accessKeyID)
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
				ctx := context.WithValue(req.Context(), ErrorContextKey, errMissingAuth)
				*req = *req.WithContext(ctx)
			},
			expectedNil: false,
		},
		{
			name: "signature_mismatch error",
			setupContext: func(req *http.Request) {
				ctx := context.WithValue(req.Context(), ErrorContextKey, errSignatureMismatch)
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

			err := GetError(req)
			if tt.expectedNil {
				zhtest.AssertNil(t, err)
				return
			}

			zhtest.AssertNotNil(t, err)
			zhtest.AssertEqual(t, tt.expectedType, err.Type)
		})
	}
}

func TestHMACAuth_CustomErrorHandlerWithContext(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	var receivedError *AuthError

	mw := New(Config{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request) {
			// Custom handler can access the error via context
			receivedError = GetError(r)
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

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
	zhtest.AssertNotNil(t, receivedError)
	zhtest.AssertEqual(t, errMissingAuth.Type, receivedError.Type)
}

func TestHMACAuth_CustomErrorHandlerWithSignatureMismatch(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	var receivedError *AuthError

	mw := New(Config{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request) {
			receivedError = GetError(r)
			w.WriteHeader(http.StatusForbidden) // Custom status
			_, _ = w.Write([]byte("auth failed: " + receivedError.Title))
		},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test with wrong secret - should trigger signature mismatch
	signer := NewSigner("test-key", "wrong-secret")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	receivedError = nil
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusForbidden, rr.Code)
	zhtest.AssertNotNil(t, receivedError)
	zhtest.AssertEqual(t, errSignatureMismatch.Type, receivedError.Type)
	zhtest.AssertContains(t, rr.Body.String(), "auth failed: Signature Mismatch")
}

func TestHMACAuth_KeyRotation(t *testing.T) {
	// Simulate key rotation scenario with old and new secrets
	// Secrets must be at least 32 bytes for security
	oldSecret := "old-secret-key-for-test-32bytes!"
	newSecret := "new-secret-key-for-test-32bytes!"

	// Credential store returns both secrets during rotation
	mw := New(Config{
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
		oldSigner := NewSigner("test-key", oldSecret)
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		err := oldSigner.SignRequest(req)
		zhtest.AssertNoError(t, err)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		zhtest.AssertTrue(t, handlerCalled)
		zhtest.AssertEqual(t, http.StatusOK, rr.Code)
	})

	// Test 2: Request signed with new secret should work
	t.Run("new secret", func(t *testing.T) {
		handlerCalled = false
		newSigner := NewSigner("test-key", newSecret)
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		err := newSigner.SignRequest(req)
		zhtest.AssertNoError(t, err)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		zhtest.AssertTrue(t, handlerCalled)
		zhtest.AssertEqual(t, http.StatusOK, rr.Code)
	})

	// Test 3: Request with wrong secret should fail
	t.Run("wrong secret", func(t *testing.T) {
		wrongSigner := NewSigner("test-key", "completely-wrong-secret")
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		err := wrongSigner.SignRequest(req)
		zhtest.AssertNoError(t, err)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestHMACAuth_NoSecrets(t *testing.T) {
	// Test that empty secrets slice returns invalid credentials
	mw := New(Config{
		CredentialStore: func(id string) []string {
			return nil // no secrets for any key
		},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	signer := NewSigner("any-key", "this-secret-is-32-bytes-long!!")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestHMACAuth_ShortSecretRejected(t *testing.T) {
	// Test that secrets shorter than minimum length are rejected
	// This is a security measure to prevent brute-force attacks
	creds := map[string]string{"test-key": "short-secret"} // Only 12 bytes
	mw := New(Config{
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
	signer := NewSigner("test-key", "short-secret")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestHMACAuth_ShortSecretRejected_AllAlgorithms(t *testing.T) {
	// Test minimum secret length enforcement for all HMAC algorithms
	// The middleware filters out secrets that don't meet the minimum length
	tests := []struct {
		name         string
		algorithm    HashAlgorithm
		storeSecrets []string // secrets returned by credential store
		validSecret  string   // secret used to sign (must meet min length)
		shouldPass   bool
	}{
		{
			name:         "SHA256 with only short secrets in store",
			algorithm:    SHA256,
			storeSecrets: []string{"short-secret-12", "another-short-15"},
			validSecret:  "this-is-exactly-32-bytes-!!!!!!!",
			shouldPass:   false,
		},
		{
			name:         "SHA256 with valid secret in store",
			algorithm:    SHA256,
			storeSecrets: []string{"this-is-exactly-32-bytes-!!!!!!!"},
			validSecret:  "this-is-exactly-32-bytes-!!!!!!!",
			shouldPass:   true,
		},
		{
			name:         "SHA384 with only short secrets in store",
			algorithm:    SHA384,
			storeSecrets: []string{"short-secret-12", "this-secret-is-exactly-31-bytes"},
			validSecret:  "this-is-exactly-48-bytes-for-sha384-tests!!!!!!!",
			shouldPass:   false,
		},
		{
			name:         "SHA384 with valid secret in store",
			algorithm:    SHA384,
			storeSecrets: []string{"this-is-exactly-48-bytes-for-sha384-tests!!!!!!!"},
			validSecret:  "this-is-exactly-48-bytes-for-sha384-tests!!!!!!!",
			shouldPass:   true,
		},
		{
			name:         "SHA512 with only short secrets in store",
			algorithm:    SHA512,
			storeSecrets: []string{"short-secret-12", "this-secret-is-exactly-47-bytes-long-enough!!"},
			validSecret:  "this-is-exactly-64-bytes-for-sha512-tests-in-middleware-!!!!!!!!",
			shouldPass:   false,
		},
		{
			name:         "SHA512 with valid secret in store",
			algorithm:    SHA512,
			storeSecrets: []string{"this-is-exactly-64-bytes-for-sha512-tests-in-middleware-!!!!!!!!"},
			validSecret:  "this-is-exactly-64-bytes-for-sha512-tests-in-middleware-!!!!!!!!",
			shouldPass:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := New(Config{
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

			signer := NewSignerWithAlgorithm("test-key", tt.validSecret, tt.algorithm)
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			err := signer.SignRequest(req)
			zhtest.AssertNoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if tt.shouldPass {
				zhtest.AssertEqual(t, http.StatusOK, rr.Code)
				zhtest.AssertTrue(t, handlerCalled)
			} else {
				zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
				zhtest.AssertFalse(t, handlerCalled)
			}
		})
	}
}

func TestHMACAuth_ReplayAttack_Prevented(t *testing.T) {
	// Test that replay attacks are prevented via timestamp validation
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")

	// Test 1: Request with timestamp exactly at skew boundary (should fail)
	t.Run("at_boundary", func(t *testing.T) {
		oldTime := time.Now().UTC().Add(-2 * time.Minute)
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		err := signer.SignRequestWithTime(req, oldTime)
		zhtest.AssertNoError(t, err)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
	})

	// Test 2: Request with timestamp just inside window (should succeed)
	t.Run("just_inside_window", func(t *testing.T) {
		recentTime := time.Now().UTC().Add(-30 * time.Second)
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		err := signer.SignRequestWithTime(req, recentTime)
		zhtest.AssertNoError(t, err)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		zhtest.AssertEqual(t, http.StatusOK, rr.Code)
	})
}

func TestHMACAuth_HeaderTampering_Detected(t *testing.T) {
	// Test that tampering with signed headers invalidates the signature
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	signer.SetHeadersToSign([]string{"host", "x-timestamp", "x-request-id"})

	// Sign a request with specific headers
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set(httpx.HeaderXRequestId, "original-request-id")
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	// Tamper with the header after signing
	req.Header.Set(httpx.HeaderXRequestId, "tampered-request-id")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestHMACAuth_BodyTampering_Detected(t *testing.T) {
	// Test that tampering with request body invalidates the signature
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")

	// Sign a request with specific body
	originalBody := []byte(`{"amount": 100, "to": "alice"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/transfer", bytes.NewReader(originalBody))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	// Tamper with the body after signing
	tamperedBody := []byte(`{"amount": 9999, "to": "attacker"}`)
	req.Body = io.NopCloser(bytes.NewReader(tamperedBody))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestHMACAuth_QueryParamTampering_Detected(t *testing.T) {
	// Test that tampering with query parameters invalidates the signature
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")

	// Sign a request with specific query params
	req := httptest.NewRequest(http.MethodGet, "/api/data?user=alice&amount=100", nil)
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	// Tamper with query params after signing
	req.URL.RawQuery = "user=alice&amount=9999"

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
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

	mw := New(Config{
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

	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertTrue(t, handlerCalled)
	zhtest.AssertEqual(t, "added-by-middleware", chainHeaderValue)
	zhtest.AssertEqual(t, http.StatusOK, rr.Code)
}

func TestParseAuthorizationHeader_InvalidCases(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{
			name:   "not HMAC algorithm",
			header: "Bearer token123 Credential=test/2026-03-07T12:00:00Z, SignedHeaders=host, Signature=abcd",
		},
		{
			name:   "invalid base64 signature",
			header: "HMAC-SHA256 Credential=test-key/2026-03-07T12:00:00Z, SignedHeaders=host, Signature=!!!invalid!!!",
		},
		{
			name:   "missing credential",
			header: "HMAC-SHA256 SignedHeaders=host, Signature=YWJjZA==",
		},
		{
			name:   "missing signed headers",
			header: "HMAC-SHA256 Credential=test-key/2026-03-07T12:00:00Z, Signature=YWJjZA==",
		},
		{
			name:   "missing signature",
			header: "HMAC-SHA256 Credential=test-key/2026-03-07T12:00:00Z, SignedHeaders=host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseAuthorizationHeader(tt.header, "X-Timestamp")
			zhtest.AssertError(t, err)
		})
	}
}

func TestComputeHMACSignature_DefaultAlgorithm(t *testing.T) {
	// Test that unknown algorithm defaults to SHA256
	secret := "test-secret-key-that-is-32-bytes-long!"
	canonicalRequest := "GET\n/api/test\n\nhost:example.com\n\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	// Use an invalid algorithm value
	sig := computeHMACSignature(secret, canonicalRequest, HashAlgorithm("INVALID"))
	zhtest.AssertEqual(t, 32, len(sig)) // Should default to SHA256 (32 bytes)
}

func TestComputeBodyHash_DefaultAlgorithm(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)

	// Use an invalid algorithm value - should default to SHA256
	hash, err := computeBodyHash(req, HashAlgorithm("INVALID"), 1024*1024)
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, 64, len(hash)) // SHA256 hex is 64 characters
}

func TestValidateTimestamp_FutureBeyondGrace(t *testing.T) {
	// Timestamp too far in the future (beyond grace period)
	future := time.Now().UTC().Add(10 * time.Minute)
	err := validateTimestamp(future, 5*time.Minute, 1*time.Minute)
	zhtest.AssertError(t, err)
}

func TestValidateTimestamp_FutureWithinGrace(t *testing.T) {
	// Timestamp slightly in the future (within grace period)
	future := time.Now().UTC().Add(30 * time.Second)
	err := validateTimestamp(future, 5*time.Minute, 1*time.Minute)
	zhtest.AssertNoError(t, err)
}

func TestHMACAuth_InvalidBase64Signature(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestHMACAuth_MaxBodySize(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	// Check the body after signing
	bodyBytes, _ := io.ReadAll(req.Body)
	t.Logf("Body size after signing: %d", len(bodyBytes))
	req.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	t.Logf("Response code: %d, Body: %s", rr.Code, rr.Body.String())

	zhtest.AssertEqual(t, http.StatusRequestEntityTooLarge, rr.Code)
}

func TestHMACAuth_MaxBodySize_AllowedWithUnsignedPayload(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	signer.SetAllowUnsignedPayload(true)
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusOK, rr.Code)
}

func TestHMACAuth_PresignedURL(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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
	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	presignedURL, err := signer.PresignURL(req, 5*time.Minute)
	zhtest.AssertNoError(t, err)

	// Parse the presigned URL and make a request
	parsedURL, _ := url.Parse(presignedURL)
	req2 := httptest.NewRequest(http.MethodGet, parsedURL.String(), nil)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req2)

	zhtest.AssertEqual(t, http.StatusOK, rr.Code)
}

func TestHMACAuth_PresignedURL_NotAllowed(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestHMACAuth_PresignedURL_MissingParams(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

			zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
		})
	}
}

func TestHMACAuth_PresignedURL_InvalidSignature(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestValidatePresignedURLTimestamp_Expired(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestHMACAuth_NilCredentialStore(t *testing.T) {
	// Test that nil CredentialStore causes panic
	zhtest.AssertPanic(t, func() {
		_ = New(Config{
			CredentialStore: nil,
		})
	})
}

// TestHMACAuth_PresignedURL_NoHeaderModification verifies that the middleware
// does not modify the request headers when processing presigned URLs
func TestHMACAuth_PresignedURL_NoHeaderModification(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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
	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")
	presignedURL, err := signer.PresignURL(req, 5*time.Minute)
	zhtest.AssertNoError(t, err)

	// Parse the presigned URL and make a request
	parsedURL, _ := url.Parse(presignedURL)
	req2 := httptest.NewRequest(http.MethodGet, parsedURL.String(), nil)

	// Capture the original headers before the request is processed
	originalTimestampHeader := req2.Header.Get("X-Timestamp")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req2)

	zhtest.AssertEqual(t, http.StatusOK, rr.Code)

	// Verify that the X-Timestamp header was NOT modified by the middleware
	zhtest.AssertNotNil(t, capturedRequest)
	finalTimestampHeader := capturedRequest.Header.Get("X-Timestamp")
	zhtest.AssertEqual(t, originalTimestampHeader, finalTimestampHeader)
}

func TestHMACAuth_IncludedPaths(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-key-that-is-32-bytes-long!"}
	mw := New(Config{
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

	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")

	// Test allowed path - should require auth
	req1 := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req1)
	zhtest.AssertNoError(t, err)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	zhtest.AssertEqual(t, http.StatusOK, rr1.Code)

	// Test non-allowed path - should skip auth
	req2 := httptest.NewRequest(http.MethodGet, "/public", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	zhtest.AssertEqual(t, http.StatusOK, rr2.Code)

	// Test non-allowed path with missing auth - should still pass
	req3 := httptest.NewRequest(http.MethodGet, "/other", nil)
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)

	zhtest.AssertEqual(t, http.StatusOK, rr3.Code)
}

func TestHMACAuth_BothExcludedAndIncludedPathsPanics(t *testing.T) {
	zhtest.AssertPanic(t, func() {
		_ = New(Config{
			CredentialStore: func(id string) []string { return nil },
			ExcludedPaths:   []string{"/health"},
			IncludedPaths:   []string{"/api"},
		})
	})
}
