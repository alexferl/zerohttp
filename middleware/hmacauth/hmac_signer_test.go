package hmacauth

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestNewHMACSigner(t *testing.T) {
	signer := NewSigner("test-key", "test-secret-that-is-32-bytes-long!!")

	zhtest.AssertEqual(t, "test-key", signer.AccessKeyID())
	zhtest.AssertEqual(t, SHA256, signer.Algorithm())
}

func TestNewHMACSignerWithAlgorithm(t *testing.T) {
	tests := []struct {
		name      string
		algorithm HashAlgorithm
		secret    string
	}{
		{"SHA256", SHA256, "test-secret-key-that-is-32-bytes-long!"},
		{"SHA384", SHA384, "test-secret-key-that-is-48-bytes-long-for-sha384-algo!"},
		{"SHA512", SHA512, "test-secret-key-that-is-64-bytes-long-for-the-sha512-algorithm-use!!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := NewSignerWithAlgorithm("test-key", tt.secret, tt.algorithm)
			zhtest.AssertEqual(t, tt.algorithm, signer.Algorithm())
		})
	}
}

func TestNewHMACSignerWithAlgorithm_ShortSecretPanics(t *testing.T) {
	tests := []struct {
		name         string
		algorithm    HashAlgorithm
		secret       string
		expectedSize int
	}{
		{"SHA256 too short", SHA256, "short-secret", 32},
		{"SHA384 too short", SHA384, "test-secret-key-that-is-32-bytes-long!", 48},
		{"SHA512 too short", SHA512, "test-secret-key-that-is-48-bytes-long-for-sha384-algo!", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zhtest.AssertPanic(t, func() {
				_ = NewSignerWithAlgorithm("test-key", tt.secret, tt.algorithm)
			})
		})
	}
}

func TestHMACSigner_AccessKeyID(t *testing.T) {
	signer := NewSigner("my-access-key", "test-secret-that-is-32-bytes!")
	zhtest.AssertEqual(t, "my-access-key", signer.AccessKeyID())
}

func TestHMACSigner_Algorithm(t *testing.T) {
	tests := []struct {
		name     string
		signer   *Signer
		expected HashAlgorithm
	}{
		{
			name:     "default SHA256",
			signer:   NewSigner("key", "test-secret-that-is-32-bytes!"),
			expected: SHA256,
		},
		{
			name:     "SHA384",
			signer:   NewSignerWithAlgorithm("key", "test-secret-key-that-is-48-bytes-long-for-sha384-use!", SHA384),
			expected: SHA384,
		},
		{
			name:     "SHA512",
			signer:   NewSignerWithAlgorithm("key", "test-secret-key-that-is-64-bytes-long-for-the-sha512-algorithm-use!", SHA512),
			expected: SHA512,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zhtest.AssertEqual(t, tt.expected, tt.signer.Algorithm())
		})
	}
}

func TestHMACSigner_SignRequest(t *testing.T) {
	signer := NewSigner("test-key", "test-secret-that-is-32-bytes-long!!")

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	// Check that Authorization header was added
	zhtest.AssertNotEmpty(t, req.Header.Get(httpx.HeaderAuthorization))

	// Check that X-Timestamp header was added
	zhtest.AssertNotEmpty(t, req.Header.Get("X-Timestamp"))

	// Verify Authorization header format
	zhtest.AssertTrue(t, strings.HasPrefix(req.Header.Get(httpx.HeaderAuthorization), "HMAC-SHA256 "))
}

func TestHMACSigner_SignRequestWithTime(t *testing.T) {
	signer := NewSigner("test-key", "test-secret-that-is-32-bytes-long!!")

	fixedTime := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	err := signer.SignRequestWithTime(req, fixedTime)
	zhtest.AssertNoError(t, err)

	expectedTimestamp := fixedTime.Format(time.RFC3339)
	zhtest.AssertEqual(t, expectedTimestamp, req.Header.Get("X-Timestamp"))
}

func TestHMACSigner_SignRequestWithBody(t *testing.T) {
	signer := NewSigner("test-key", "test-secret-that-is-32-bytes-long!!")

	body := `{"test":"data"}`
	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)

	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	// Verify body is still readable
	zhtest.AssertNotNil(t, req.Body)

	readBody, _ := io.ReadAll(req.Body)
	zhtest.AssertEqual(t, body, string(readBody))
}

func TestHMACSigner_SetAllowUnsignedPayload(t *testing.T) {
	signer := NewSigner("test-key", "test-secret-that-is-32-bytes-long!!")
	signer.SetAllowUnsignedPayload(true)

	body := `{"test":"data"}`
	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(body))

	hash := signer.computeBodyHash(req)
	zhtest.AssertEqual(t, "UNSIGNED-PAYLOAD", hash)
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
			signer := NewSigner("key", "test-secret-that-is-32-bytes!")
			if tt.headersToSign != nil {
				signer.SetHeadersToSign(tt.headersToSign)
			}

			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			for k, v := range tt.requestHeaders {
				req.Header.Set(k, v)
			}

			headers := signer.buildSignedHeadersList(req)

			zhtest.AssertEqual(t, len(tt.expectedHeaders), len(headers))
			for i, h := range headers {
				zhtest.AssertEqual(t, tt.expectedHeaders[i], h)
			}
		})
	}
}

func TestHMACSigner_CustomHeadersRoundTrip(t *testing.T) {
	creds := map[string]string{"test-key": "test-secret-that-is-32-bytes-long!!"}
	mw := New(Config{
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
	signer := NewSigner("test-key", "test-secret-that-is-32-bytes-long!!")
	signer.SetHeadersToSign([]string{"host", "x-timestamp", "x-request-id"})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set(httpx.HeaderXRequestId, "test-request-123")
	err := signer.SignRequest(req)
	zhtest.AssertNoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertTrue(t, handlerCalled)
	zhtest.AssertEqual(t, http.StatusOK, rr.Code)
}

func TestHMACSigner_GenerateSignature(t *testing.T) {
	signer := NewSigner("test-key", "test-secret-that-is-32-bytes-long!!")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)

	timestamp := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	sig, err := signer.GenerateSignature(req, timestamp)
	zhtest.AssertNoError(t, err)

	zhtest.AssertNotEmpty(t, sig)

	// Signature should be base64-encoded
	_, err = base64.StdEncoding.DecodeString(sig)
	zhtest.AssertNoError(t, err)
}

func TestHMACSigner_PresignURL(t *testing.T) {
	signer := NewSigner("test-key", "test-secret-that-is-32-bytes-long!!")
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/data?foo=bar", nil)

	presignedURL, err := signer.PresignURL(req, 5*time.Minute)
	zhtest.AssertNoError(t, err)

	zhtest.AssertNotEmpty(t, presignedURL)

	// Check that query parameters were added
	zhtest.AssertTrue(t, strings.Contains(presignedURL, "X-HMAC-Algorithm="))
	zhtest.AssertTrue(t, strings.Contains(presignedURL, "X-HMAC-Credential="))
	zhtest.AssertTrue(t, strings.Contains(presignedURL, "X-HMAC-SignedHeaders="))
	zhtest.AssertTrue(t, strings.Contains(presignedURL, "X-HMAC-Signature="))
}

func TestHMACSigner_PresignURLWithTime(t *testing.T) {
	signer := NewSigner("test-key", "test-secret-that-is-32-bytes-long!!")
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/data", nil)

	expiresAt := time.Date(2026, 3, 7, 13, 0, 0, 0, time.UTC)
	presignedURL, err := signer.PresignURLWithTime(req, expiresAt)
	zhtest.AssertNoError(t, err)

	zhtest.AssertNotEmpty(t, presignedURL)

	// URL should contain the specific expiration time
	zhtest.AssertTrue(t, strings.Contains(presignedURL, "2026-03-07T13%3A00%3A00Z"))
}

func TestHMACSigner_computeBodyHash(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		algorithm HashAlgorithm
		secret    string
		unsigned  bool
		expected  string
	}{
		{
			name:      "SHA256 with body",
			body:      `{"test":"data"}`,
			algorithm: SHA256,
			secret:    "test-secret-key-that-is-32-bytes-long!",
			unsigned:  false,
			expected:  "", // Will compute actual hash
		},
		{
			name:      "SHA384 with body",
			body:      `{"test":"data"}`,
			algorithm: SHA384,
			secret:    "test-secret-key-that-is-48-bytes-long-for-sha384-algo!",
			unsigned:  false,
			expected:  "", // Will compute actual hash
		},
		{
			name:      "SHA512 with body",
			body:      `{"test":"data"}`,
			algorithm: SHA512,
			secret:    "test-secret-key-that-is-64-bytes-long-for-the-sha512-algorithm-use!!",
			unsigned:  false,
			expected:  "", // Will compute actual hash
		},
		{
			name:      "Unsigned payload",
			body:      `{"test":"data"}`,
			algorithm: SHA256,
			secret:    "test-secret-key-that-is-32-bytes-long!",
			unsigned:  true,
			expected:  "UNSIGNED-PAYLOAD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := NewSignerWithAlgorithm("test-key", tt.secret, tt.algorithm)
			signer.SetAllowUnsignedPayload(tt.unsigned)

			req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(tt.body))

			hash := signer.computeBodyHash(req)

			if tt.unsigned {
				zhtest.AssertEqual(t, tt.expected, hash)
			} else {
				// Hash should be hex encoded
				zhtest.AssertNotEmpty(t, hash)
				// Check it's valid hex
				for _, c := range hash {
					zhtest.AssertTrue(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'))
				}
			}
		})
	}
}

func TestHMACSigner_buildCanonicalQueryString(t *testing.T) {
	signer := NewSigner("test-key", "test-secret-key-that-is-32-bytes-long!")

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
			zhtest.AssertEqual(t, tt.expected, result)
		})
	}
}
