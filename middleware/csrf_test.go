package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/zhtest"
)

var testHMACKey = []byte("test-key-for-csrf-middleware-32!!")

func TestCSRF_MissingHMACKeyPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("Expected panic for missing HMACKey, but did not panic")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "HMACKey is required") {
			t.Errorf("Expected panic message to contain 'HMACKey is required', got: %v", r)
		}
	}()

	// This should panic
	_ = CSRF()
}

func TestCSRF_ValidateTokenFormat(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		{
			name:     "valid token",
			token:    generateToken(testHMACKey),
			expected: true,
		},
		{
			name:     "invalid base64",
			token:    "!!!invalid-base64!!!",
			expected: false,
		},
		{
			name:     "too short",
			token:    base64.RawURLEncoding.EncodeToString([]byte("short")),
			expected: false,
		},
		{
			name:     "empty string",
			token:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateTokenFormat(tt.token)
			if result != tt.expected {
				t.Errorf("validateTokenFormat(%q) = %v, want %v", tt.token, result, tt.expected)
			}
		})
	}
}

func TestCSRF_DefaultValues(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Test with explicit zero values to trigger default code paths
	csrf := CSRF(config.CSRFConfig{
		HMACKey:       testHMACKey,
		CookieName:    "",
		CookieMaxAge:  0,
		CookiePath:    "",
		TokenLookup:   "",
		ExemptMethods: nil,
		ExemptPaths:   nil,
	})(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	csrf.ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).Status(http.StatusOK)

	// Verify defaults were applied by checking cookie
	cookies := rr.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			csrfCookie = c
			break
		}
	}

	if csrfCookie == nil {
		t.Fatal("Expected csrf_token cookie with default name")
	}

	if csrfCookie.Path != "/" {
		t.Errorf("Expected default path /, got %s", csrfCookie.Path)
	}

	if csrfCookie.MaxAge != 86400 {
		t.Errorf("Expected default max-age 86400, got %d", csrfCookie.MaxAge)
	}
}

func TestCSRF_TokenGeneration(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := GetCSRFToken(r)
		if token == "" {
			t.Error("Expected CSRF token in context, got empty string")
		}
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.CSRFConfig{HMACKey: testHMACKey})(handler)

	// GET request should generate token
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	csrf.ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).Status(http.StatusOK)

	// Check cookie was set
	cookies := rr.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			csrfCookie = c
			break
		}
	}

	if csrfCookie == nil {
		t.Error("Expected CSRF cookie to be set")
	} else {
		if csrfCookie.HttpOnly != true {
			t.Error("Expected cookie to be HttpOnly")
		}
		if csrfCookie.Secure != true {
			t.Error("Expected cookie to be Secure")
		}
		if csrfCookie.SameSite != http.SameSiteStrictMode {
			t.Errorf("Expected SameSite=Strict, got %v", csrfCookie.SameSite)
		}
		if csrfCookie.Path != "/" {
			t.Errorf("Expected Path=/, got %s", csrfCookie.Path)
		}
	}
}

func TestCSRF_ValidToken(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	csrf := CSRF(config.CSRFConfig{HMACKey: testHMACKey})(handler)

	// First, get a CSRF token via GET
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr1 := httptest.NewRecorder()
	csrf.ServeHTTP(rr1, req1)

	// Extract cookie
	cookies := rr1.Result().Cookies()
	var token string
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			token = c.Value
			break
		}
	}

	if token == "" {
		t.Fatal("Failed to get CSRF token")
	}

	// Now make POST request with token
	req2 := httptest.NewRequest(http.MethodPost, "/", nil)
	req2.Header.Set("X-CSRF-Token", token)
	req2.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})

	rr2 := httptest.NewRecorder()
	csrf.ServeHTTP(rr2, req2)

	zhtest.AssertWith(t, rr2).
		Status(http.StatusOK).
		Body("success")
}

func TestCSRF_InvalidToken(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.CSRFConfig{HMACKey: testHMACKey})(handler)

	// Make POST request with invalid token
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-CSRF-Token", "invalid-token")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "invalid-token"})

	rr := httptest.NewRecorder()
	csrf.ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).
		Status(http.StatusForbidden).
		BodyContains("CSRF token invalid or missing")
}

func TestCSRF_MissingToken(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.CSRFConfig{HMACKey: testHMACKey})(handler)

	// Make POST request without token
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	rr := httptest.NewRecorder()
	csrf.ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).Status(http.StatusForbidden)
}

func TestCSRF_MismatchedTokens(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.CSRFConfig{HMACKey: testHMACKey})(handler)

	// First, get a valid CSRF token via GET
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr1 := httptest.NewRecorder()
	csrf.ServeHTTP(rr1, req1)

	cookies := rr1.Result().Cookies()
	var validToken string
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			validToken = c.Value
			break
		}
	}

	// Make POST request with different token in header vs cookie
	req2 := httptest.NewRequest(http.MethodPost, "/", nil)
	req2.Header.Set("X-CSRF-Token", validToken)
	req2.AddCookie(&http.Cookie{Name: "csrf_token", Value: "different-token"})

	rr2 := httptest.NewRecorder()
	csrf.ServeHTTP(rr2, req2)

	zhtest.AssertWith(t, rr2).Status(http.StatusForbidden)
}

func TestCSRF_ExemptMethods(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.CSRFConfig{HMACKey: testHMACKey})(handler)

	exemptMethods := []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace}

	for _, method := range exemptMethods {
		req := httptest.NewRequest(method, "/", nil)
		rr := httptest.NewRecorder()

		csrf.ServeHTTP(rr, req)

		zhtest.AssertWith(t, rr).Status(http.StatusOK)
	}
}

func TestCSRF_ExemptPaths(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.CSRFConfig{
		HMACKey:     testHMACKey,
		ExemptPaths: []string{"/api/webhook", "/public/"},
	})(handler)

	// Test exact path match
	req1 := httptest.NewRequest(http.MethodPost, "/api/webhook", nil)
	rr1 := httptest.NewRecorder()
	csrf.ServeHTTP(rr1, req1)

	zhtest.AssertWith(t, rr1).Status(http.StatusOK)

	// Test prefix path match
	req2 := httptest.NewRequest(http.MethodPost, "/public/something", nil)
	rr2 := httptest.NewRecorder()
	csrf.ServeHTTP(rr2, req2)

	zhtest.AssertWith(t, rr2).Status(http.StatusOK)

	// Test non-exempt path (should require token)
	req3 := httptest.NewRequest(http.MethodPost, "/api/other", nil)
	rr3 := httptest.NewRecorder()
	csrf.ServeHTTP(rr3, req3)

	zhtest.AssertWith(t, rr3).Status(http.StatusForbidden)
}

func TestCSRF_CustomErrorHandler(t *testing.T) {
	customHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"custom csrf error"}`))
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.CSRFConfig{
		HMACKey:      testHMACKey,
		ErrorHandler: customHandler,
	})(handler)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()
	csrf.ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).
		Status(http.StatusForbidden).
		Header("Content-Type", "application/json").
		BodyContains("custom csrf error")
}

func TestCSRF_FormTokenLookup(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.CSRFConfig{
		HMACKey:     testHMACKey,
		TokenLookup: "form:csrf_token",
	})(handler)

	// Get token
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr1 := httptest.NewRecorder()
	csrf.ServeHTTP(rr1, req1)

	cookies := rr1.Result().Cookies()
	var token string
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			token = c.Value
			break
		}
	}

	// POST with token in form
	formData := url.Values{}
	formData.Set("csrf_token", token)

	req2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})

	rr2 := httptest.NewRecorder()
	csrf.ServeHTTP(rr2, req2)

	zhtest.AssertWith(t, rr2).Status(http.StatusOK)
}

func TestCSRF_MultipartFormTokenLookup(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.CSRFConfig{
		HMACKey:     testHMACKey,
		TokenLookup: "form:csrf_token",
	})(handler)

	// Get token
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr1 := httptest.NewRecorder()
	csrf.ServeHTTP(rr1, req1)

	cookies := rr1.Result().Cookies()
	var token string
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			token = c.Value
			break
		}
	}

	// Build multipart form with token
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("csrf_token", token)
	_ = writer.WriteField("message", "hello")
	_ = writer.Close()

	req2 := httptest.NewRequest(http.MethodPost, "/", &body)
	req2.Header.Set("Content-Type", writer.FormDataContentType())
	req2.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})

	rr2 := httptest.NewRecorder()
	csrf.ServeHTTP(rr2, req2)

	zhtest.AssertWith(t, rr2).Status(http.StatusOK)
}

func TestCSRF_QueryTokenLookup(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.CSRFConfig{
		HMACKey:     testHMACKey,
		TokenLookup: "query:csrf_token",
	})(handler)

	// Get token
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr1 := httptest.NewRecorder()
	csrf.ServeHTTP(rr1, req1)

	cookies := rr1.Result().Cookies()
	var token string
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			token = c.Value
			break
		}
	}

	// POST with token in query string
	req2 := httptest.NewRequest(http.MethodPost, "/?csrf_token="+token, nil)
	req2.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})

	rr2 := httptest.NewRecorder()
	csrf.ServeHTTP(rr2, req2)

	zhtest.AssertWith(t, rr2).Status(http.StatusOK)
}

func TestCSRF_CustomCookieOptions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.CSRFConfig{
		HMACKey:        testHMACKey,
		CookieName:     "custom_csrf",
		CookieMaxAge:   3600,
		CookieDomain:   "example.com",
		CookiePath:     "/api",
		CookieSecure:   config.Bool(false),
		CookieSameSite: http.SameSiteLaxMode,
	})(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	csrf.ServeHTTP(rr, req)

	cookies := rr.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "custom_csrf" {
			csrfCookie = c
			break
		}
	}

	if csrfCookie == nil {
		t.Fatal("Expected custom CSRF cookie to be set")
	}

	if csrfCookie.MaxAge != 3600 {
		t.Errorf("Expected MaxAge=3600, got %d", csrfCookie.MaxAge)
	}

	if csrfCookie.Domain != "example.com" {
		t.Errorf("Expected Domain=example.com, got %s", csrfCookie.Domain)
	}

	if csrfCookie.Path != "/api" {
		t.Errorf("Expected Path=/api, got %s", csrfCookie.Path)
	}

	if csrfCookie.Secure != false {
		t.Error("Expected Secure=false")
	}

	if csrfCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("Expected SameSite=Lax, got %v", csrfCookie.SameSite)
	}
}

func TestCSRF_TokenRotation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.CSRFConfig{HMACKey: testHMACKey})(handler)

	// Get initial token
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr1 := httptest.NewRecorder()
	csrf.ServeHTTP(rr1, req1)

	cookies := rr1.Result().Cookies()
	var token1 string
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			token1 = c.Value
			break
		}
	}

	// POST with token should get new token
	req2 := httptest.NewRequest(http.MethodPost, "/", nil)
	req2.Header.Set("X-CSRF-Token", token1)
	req2.AddCookie(&http.Cookie{Name: "csrf_token", Value: token1})

	rr2 := httptest.NewRecorder()
	csrf.ServeHTTP(rr2, req2)

	zhtest.AssertWith(t, rr2).Status(http.StatusOK)

	// Check that a new cookie was set
	cookies2 := rr2.Result().Cookies()
	var token2 string
	for _, c := range cookies2 {
		if c.Name == "csrf_token" {
			token2 = c.Value
			break
		}
	}

	if token2 == "" {
		t.Error("Expected new token after POST")
	}

	if token1 == token2 {
		t.Error("Expected token to be rotated after POST")
	}
}

func TestCSRF_GetCSRFToken(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := GetCSRFToken(r)
		_, _ = w.Write([]byte(token))
	})

	csrf := CSRF(config.CSRFConfig{HMACKey: testHMACKey})(handler)

	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.Serve(csrf, req)

	// Should have a non-empty token in response body
	token := w.Body.String()
	if token == "" {
		t.Error("Expected non-empty CSRF token from GetCSRFToken")
	}

	// Verify it's a valid base64 token
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Errorf("Token is not valid base64: %v", err)
	}

	// Should be 32 bytes token + 32 bytes HMAC
	expectedLen := defaultTokenLength + sha256.Size
	if len(data) != expectedLen {
		t.Errorf("Expected token length %d, got %d", expectedLen, len(data))
	}
}

func TestCSRF_NoTokenInContext(t *testing.T) {
	// Direct request without CSRF middleware
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	token := GetCSRFToken(req)
	if token != "" {
		t.Errorf("Expected empty token without middleware, got %s", token)
	}
}

func TestCSRF_InvalidBase64Token(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.CSRFConfig{HMACKey: testHMACKey})(handler)

	// Make POST request with invalid base64 token
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-CSRF-Token", "!!!invalid-base64!!!")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "!!!invalid-base64!!!"})

	rr := httptest.NewRecorder()
	csrf.ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).Status(http.StatusForbidden)
}

func TestCSRF_TokenWithWrongSignature(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Use a specific HMAC key
	hmacKey := []byte("test-key-for-csrf-middleware-32!")
	csrf := CSRF(config.CSRFConfig{HMACKey: hmacKey})(handler)

	// Get valid token
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr1 := httptest.NewRecorder()
	csrf.ServeHTTP(rr1, req1)

	cookies := rr1.Result().Cookies()
	var validToken string
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			validToken = c.Value
			break
		}
	}

	// Create a different middleware with different key
	hmacKey2 := []byte("different-key-for-testing-32!!")
	csrf2 := CSRF(config.CSRFConfig{HMACKey: hmacKey2})(handler)

	// Try to use token from first middleware with second middleware
	req2 := httptest.NewRequest(http.MethodPost, "/", nil)
	req2.Header.Set("X-CSRF-Token", validToken)
	req2.AddCookie(&http.Cookie{Name: "csrf_token", Value: validToken})

	rr2 := httptest.NewRecorder()
	csrf2.ServeHTTP(rr2, req2)

	// Should fail because HMAC key is different
	zhtest.AssertWith(t, rr2).Status(http.StatusForbidden)
}

func TestCSRF_CustomExemptMethods(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Only exempt PUT, not GET
	csrf := CSRF(config.CSRFConfig{
		HMACKey:       testHMACKey,
		ExemptMethods: []string{http.MethodPut},
	})(handler)

	// GET without token should return 403 (not exempt)
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr1 := httptest.NewRecorder()
	csrf.ServeHTTP(rr1, req1)

	zhtest.AssertWith(t, rr1).Status(http.StatusForbidden)

	// PUT should be exempt and work without token
	req2 := httptest.NewRequest(http.MethodPut, "/", nil)
	rr2 := httptest.NewRecorder()
	csrf.ServeHTTP(rr2, req2)

	zhtest.AssertWith(t, rr2).Status(http.StatusOK)
}

func TestCSRF_EmptyCookieValue(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.CSRFConfig{HMACKey: testHMACKey})(handler)

	// POST with empty cookie value
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-CSRF-Token", "some-token")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: ""})

	rr := httptest.NewRecorder()
	csrf.ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).Status(http.StatusForbidden)
}

func TestCSRF_InvalidTokenFormatRegeneration(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.CSRFConfig{HMACKey: testHMACKey})(handler)

	// GET request with invalid token format cookie should regenerate
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "invalid-base64-token"})

	rr := httptest.NewRecorder()
	csrf.ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).Status(http.StatusOK)

	// Should have set a new valid cookie
	cookies := rr.Result().Cookies()
	var newCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			newCookie = c
			break
		}
	}

	if newCookie == nil {
		t.Error("Expected new CSRF cookie to be set")
	}
}

func TestCSRF_DefaultTokenLookup(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Use malformed token lookup to trigger default case
	csrf := CSRF(config.CSRFConfig{
		HMACKey:     testHMACKey,
		TokenLookup: "malformed",
	})(handler)

	// Get token
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr1 := httptest.NewRecorder()
	csrf.ServeHTTP(rr1, req1)

	cookies := rr1.Result().Cookies()
	var token string
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			token = c.Value
			break
		}
	}

	// POST with token in X-CSRF-Token header (default lookup)
	req2 := httptest.NewRequest(http.MethodPost, "/", nil)
	req2.Header.Set("X-CSRF-Token", token)
	req2.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})

	rr2 := httptest.NewRecorder()
	csrf.ServeHTTP(rr2, req2)

	// Should succeed because malformed lookup defaults to header:X-CSRF-Token
	zhtest.AssertWith(t, rr2).Status(http.StatusOK)
}

func TestCSRF_UnknownTokenLookupSource(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Use unknown source to trigger default case in extractToken
	csrf := CSRF(config.CSRFConfig{
		HMACKey:     testHMACKey,
		TokenLookup: "unknown:token",
	})(handler)

	// Get token
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr1 := httptest.NewRecorder()
	csrf.ServeHTTP(rr1, req1)

	cookies := rr1.Result().Cookies()
	var token string
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			token = c.Value
			break
		}
	}

	// POST - should fail because unknown source defaults to header lookup
	// but we're not providing the header
	req2 := httptest.NewRequest(http.MethodPost, "/", nil)
	req2.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})

	rr2 := httptest.NewRecorder()
	csrf.ServeHTTP(rr2, req2)

	// Should fail because token not in header
	zhtest.AssertWith(t, rr2).Status(http.StatusForbidden)
}

func TestCSRF_FormParseError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.CSRFConfig{
		HMACKey:     testHMACKey,
		TokenLookup: "form:csrf_token",
	})(handler)

	// Get token first
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr1 := httptest.NewRecorder()
	csrf.ServeHTTP(rr1, req1)

	cookies := rr1.Result().Cookies()
	var token string
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			token = c.Value
			break
		}
	}

	// POST with invalid form body (will cause ParseForm to fail)
	req2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("invalid%form"))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})

	rr2 := httptest.NewRecorder()
	csrf.ServeHTTP(rr2, req2)

	// Should fail because form parsing fails
	zhtest.AssertWith(t, rr2).Status(http.StatusForbidden)
}
