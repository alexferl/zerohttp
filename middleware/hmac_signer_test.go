package middleware

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
)

func TestNewHMACSigner(t *testing.T) {
	signer := NewHMACSigner("test-key", "test-secret-that-is-32-bytes!")

	if signer.AccessKeyID() != "test-key" {
		t.Errorf("expected access key ID 'test-key', got %s", signer.AccessKeyID())
	}

	if signer.Algorithm() != config.HMACSHA256 {
		t.Errorf("expected default algorithm SHA256, got %s", signer.Algorithm())
	}
}

func TestNewHMACSignerWithAlgorithm(t *testing.T) {
	tests := []struct {
		name      string
		algorithm config.HMACHashAlgorithm
		secret    string
	}{
		{"SHA256", config.HMACSHA256, "test-secret-key-that-is-32-bytes-long!"},
		{"SHA384", config.HMACSHA384, "test-secret-key-that-is-48-bytes-long-for-sha384-algo!"},
		{"SHA512", config.HMACSHA512, "test-secret-key-that-is-64-bytes-long-for-the-sha512-algorithm-use!!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := NewHMACSignerWithAlgorithm("test-key", tt.secret, tt.algorithm)
			if signer.Algorithm() != tt.algorithm {
				t.Errorf("expected algorithm %s, got %s", tt.algorithm, signer.Algorithm())
			}
		})
	}
}

func TestHMACSigner_AccessKeyID(t *testing.T) {
	signer := NewHMACSigner("my-access-key", "test-secret-that-is-32-bytes!")
	if signer.AccessKeyID() != "my-access-key" {
		t.Errorf("expected AccessKeyID to be 'my-access-key', got %q", signer.AccessKeyID())
	}
}

func TestHMACSigner_Algorithm(t *testing.T) {
	tests := []struct {
		name     string
		signer   *HMACSigner
		expected config.HMACHashAlgorithm
	}{
		{
			name:     "default SHA256",
			signer:   NewHMACSigner("key", "test-secret-that-is-32-bytes!"),
			expected: config.HMACSHA256,
		},
		{
			name:     "SHA384",
			signer:   NewHMACSignerWithAlgorithm("key", "test-secret-key-that-is-48-bytes-long-for-sha384-use!", config.HMACSHA384),
			expected: config.HMACSHA384,
		},
		{
			name:     "SHA512",
			signer:   NewHMACSignerWithAlgorithm("key", "test-secret-key-that-is-64-bytes-long-for-the-sha512-algorithm-use!", config.HMACSHA512),
			expected: config.HMACSHA512,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.signer.Algorithm() != tt.expected {
				t.Errorf("expected algorithm %q, got %q", tt.expected, tt.signer.Algorithm())
			}
		})
	}
}

func TestHMACSigner_SignRequest(t *testing.T) {
	signer := NewHMACSigner("test-key", "test-secret-that-is-32-bytes!")

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check that Authorization header was added
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		t.Error("expected Authorization header to be set")
	}

	// Check that X-Timestamp header was added
	timestampHeader := req.Header.Get("X-Timestamp")
	if timestampHeader == "" {
		t.Error("expected X-Timestamp header to be set")
	}

	// Verify Authorization header format
	if !strings.HasPrefix(authHeader, "HMAC-SHA256 ") {
		t.Errorf("expected Authorization header to start with 'HMAC-SHA256 ', got: %s", authHeader)
	}
}

func TestHMACSigner_SignRequestWithTime(t *testing.T) {
	signer := NewHMACSigner("test-key", "test-secret-that-is-32-bytes!")

	fixedTime := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequestWithTime(req, fixedTime)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	timestampHeader := req.Header.Get("X-Timestamp")
	expectedTimestamp := fixedTime.Format(time.RFC3339)
	if timestampHeader != expectedTimestamp {
		t.Errorf("expected timestamp %s, got %s", expectedTimestamp, timestampHeader)
	}
}

func TestHMACSigner_SignRequestWithBody(t *testing.T) {
	signer := NewHMACSigner("test-key", "test-secret-that-is-32-bytes!")

	body := `{"test":"data"}`
	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	err := signer.SignRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify body is still readable
	if req.Body == nil {
		t.Error("expected body to be preserved")
	}

	readBody, _ := io.ReadAll(req.Body)
	if string(readBody) != body {
		t.Errorf("expected body %s, got %s", body, string(readBody))
	}
}

func TestHMACSigner_SetAllowUnsignedPayload(t *testing.T) {
	signer := NewHMACSigner("test-key", "test-secret-that-is-32-bytes!")
	signer.SetAllowUnsignedPayload(true)

	body := `{"test":"data"}`
	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(body))

	hash := signer.computeBodyHash(req)
	if hash != "UNSIGNED-PAYLOAD" {
		t.Errorf("expected UNSIGNED-PAYLOAD, got %s", hash)
	}
}

func TestHMACSigner_SetHeadersToSign(t *testing.T) {
	tests := []struct {
		name            string
		headersToSign   []string
		requestHeaders  map[string]string
		expectedHeaders []string
	}{
		{
			name:            "default headers",
			headersToSign:   nil,
			requestHeaders:  map[string]string{"Content-Type": "application/json"},
			expectedHeaders: []string{"host", "x-timestamp", "content-type"},
		},
		{
			name:            "custom headers",
			headersToSign:   []string{"host", "x-request-id", "x-correlation-id"},
			requestHeaders:  map[string]string{"X-Request-Id": "123", "X-Correlation-Id": "abc"},
			expectedHeaders: []string{"host", "x-request-id", "x-correlation-id"},
		},
		{
			name:            "custom headers with missing optional",
			headersToSign:   []string{"host", "x-request-id", "x-optional"},
			requestHeaders:  map[string]string{"X-Request-Id": "123"},
			expectedHeaders: []string{"host", "x-request-id"},
		},
		{
			name:            "host always included even without header",
			headersToSign:   []string{"host", "x-timestamp"},
			requestHeaders:  map[string]string{"X-Timestamp": "2026-03-07T12:00:00Z"},
			expectedHeaders: []string{"host", "x-timestamp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := NewHMACSigner("key", "test-secret-that-is-32-bytes!")
			if tt.headersToSign != nil {
				signer.SetHeadersToSign(tt.headersToSign)
			}

			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			for k, v := range tt.requestHeaders {
				req.Header.Set(k, v)
			}

			headers := signer.buildSignedHeadersList(req)

			if len(headers) != len(tt.expectedHeaders) {
				t.Errorf("expected %d headers, got %d: %v", len(tt.expectedHeaders), len(headers), headers)
				return
			}

			for i, h := range headers {
				if h != tt.expectedHeaders[i] {
					t.Errorf("header[%d]: expected %q, got %q", i, tt.expectedHeaders[i], h)
				}
			}
		})
	}
}

func TestHMACSigner_CustomHeadersRoundTrip(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-that-is-32-bytes!"}
	mw := HMACAuth(config.HMACAuthConfig{
		CredentialStore: func(id string) []string {
			if secret, ok := creds[id]; ok {
				return []string{secret}
			}
			return nil
		},
		RequiredHeaders: []string{"host", "x-timestamp", "x-request-id"},
	})

	var handlerCalled bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	// Create signer with custom headers
	signer := NewHMACSigner("test-key", "test-secret-that-is-32-bytes!")
	signer.SetHeadersToSign([]string{"host", "x-timestamp", "x-request-id"})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Request-Id", "test-request-123")
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

func TestHMACSigner_GenerateSignature(t *testing.T) {
	signer := NewHMACSigner("test-key", "test-secret-that-is-32-bytes!")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)

	timestamp := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	sig, err := signer.GenerateSignature(req, timestamp)
	if err != nil {
		t.Fatalf("GenerateSignature failed: %v", err)
	}

	if sig == "" {
		t.Error("expected non-empty signature")
	}

	// Signature should be base64-encoded
	if _, err := base64.StdEncoding.DecodeString(sig); err != nil {
		t.Errorf("signature is not valid base64: %v", err)
	}
}

func TestHMACSigner_PresignURL(t *testing.T) {
	signer := NewHMACSigner("test-key", "test-secret-that-is-32-bytes!")
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/data?foo=bar", nil)

	presignedURL, err := signer.PresignURL(req, 5*time.Minute)
	if err != nil {
		t.Fatalf("PresignURL failed: %v", err)
	}

	if presignedURL == "" {
		t.Fatal("expected non-empty presigned URL")
	}

	// Check that query parameters were added
	if !strings.Contains(presignedURL, "X-HMAC-Algorithm=") {
		t.Error("expected X-HMAC-Algorithm in presigned URL")
	}
	if !strings.Contains(presignedURL, "X-HMAC-Credential=") {
		t.Error("expected X-HMAC-Credential in presigned URL")
	}
	if !strings.Contains(presignedURL, "X-HMAC-SignedHeaders=") {
		t.Error("expected X-HMAC-SignedHeaders in presigned URL")
	}
	if !strings.Contains(presignedURL, "X-HMAC-Signature=") {
		t.Error("expected X-HMAC-Signature in presigned URL")
	}
}

func TestHMACSigner_PresignURLWithTime(t *testing.T) {
	signer := NewHMACSigner("test-key", "test-secret-that-is-32-bytes!")
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/data", nil)

	expiresAt := time.Date(2026, 3, 7, 13, 0, 0, 0, time.UTC)
	presignedURL, err := signer.PresignURLWithTime(req, expiresAt)
	if err != nil {
		t.Fatalf("PresignURLWithTime failed: %v", err)
	}

	if presignedURL == "" {
		t.Fatal("expected non-empty presigned URL")
	}

	// URL should contain the specific expiration time
	if !strings.Contains(presignedURL, "2026-03-07T13%3A00%3A00Z") {
		t.Error("expected expiration time in presigned URL")
	}
}

func TestHMACSigner_computeBodyHash(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		algorithm config.HMACHashAlgorithm
		secret    string
		unsigned  bool
		expected  string
	}{
		{
			name:      "SHA256 with body",
			body:      `{"test":"data"}`,
			algorithm: config.HMACSHA256,
			secret:    "test-secret-key-that-is-32-bytes-long!",
			unsigned:  false,
			expected:  "", // Will compute actual hash
		},
		{
			name:      "SHA384 with body",
			body:      `{"test":"data"}`,
			algorithm: config.HMACSHA384,
			secret:    "test-secret-key-that-is-48-bytes-long-for-sha384-algo!",
			unsigned:  false,
			expected:  "", // Will compute actual hash
		},
		{
			name:      "SHA512 with body",
			body:      `{"test":"data"}`,
			algorithm: config.HMACSHA512,
			secret:    "test-secret-key-that-is-64-bytes-long-for-the-sha512-algorithm-use!!",
			unsigned:  false,
			expected:  "", // Will compute actual hash
		},
		{
			name:      "Unsigned payload",
			body:      `{"test":"data"}`,
			algorithm: config.HMACSHA256,
			secret:    "test-secret-key-that-is-32-bytes-long!",
			unsigned:  true,
			expected:  "UNSIGNED-PAYLOAD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := NewHMACSignerWithAlgorithm("test-key", tt.secret, tt.algorithm)
			signer.SetAllowUnsignedPayload(tt.unsigned)

			req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(tt.body))

			hash := signer.computeBodyHash(req)

			if tt.unsigned {
				if hash != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, hash)
				}
			} else {
				// Hash should be hex encoded
				if len(hash) == 0 {
					t.Error("expected non-empty hash")
				}
				// Check it's valid hex
				for _, c := range hash {
					if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
						t.Errorf("hash contains invalid hex character: %c", c)
					}
				}
			}
		})
	}
}

func TestHMACSigner_buildCanonicalQueryString(t *testing.T) {
	signer := NewHMACSigner("test-key", "test-secret-key-that-is-32-bytes-long!")

	tests := []struct {
		name     string
		query    map[string][]string
		expected string
	}{
		{
			name:     "empty query",
			query:    map[string][]string{},
			expected: "",
		},
		{
			name:     "single parameter",
			query:    map[string][]string{"key": {"value"}},
			expected: "key=value",
		},
		{
			name:     "multiple parameters sorted",
			query:    map[string][]string{"b": {"2"}, "a": {"1"}},
			expected: "a=1&b=2",
		},
		{
			name:     "multiple values sorted",
			query:    map[string][]string{"key": {"c", "a", "b"}},
			expected: "key=a&key=b&key=c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := signer.buildCanonicalQueryString(tt.query)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
