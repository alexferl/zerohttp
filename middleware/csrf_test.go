package middleware

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

var testHMACKey = []byte("test-key-for-csrf-middleware-32!!")

func TestCSRF_TokenGeneration(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := GetCSRFToken(r)
		if token == "" {
			t.Error("Expected CSRF token in context, got empty string")
		}
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.WithCSRFHMACKey(testHMACKey))(handler)

	// GET request should generate token
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	csrf.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

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

	csrf := CSRF(config.WithCSRFHMACKey(testHMACKey))(handler)

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

	if rr2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr2.Code)
	}

	if rr2.Body.String() != "success" {
		t.Errorf("Expected body 'success', got %s", rr2.Body.String())
	}
}

func TestCSRF_InvalidToken(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.WithCSRFHMACKey(testHMACKey))(handler)

	// Make POST request with invalid token
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-CSRF-Token", "invalid-token")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "invalid-token"})

	rr := httptest.NewRecorder()
	csrf.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "CSRF token invalid or missing") {
		t.Errorf("Expected error message, got %s", body)
	}
}

func TestCSRF_MissingToken(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.WithCSRFHMACKey(testHMACKey))(handler)

	// Make POST request without token
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	rr := httptest.NewRecorder()
	csrf.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rr.Code)
	}
}

func TestCSRF_MismatchedTokens(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.WithCSRFHMACKey(testHMACKey))(handler)

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

	if rr2.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rr2.Code)
	}
}

func TestCSRF_ExemptMethods(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.WithCSRFHMACKey(testHMACKey))(handler)

	exemptMethods := []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace}

	for _, method := range exemptMethods {
		req := httptest.NewRequest(method, "/", nil)
		rr := httptest.NewRecorder()

		csrf.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Method %s: Expected status 200, got %d", method, rr.Code)
		}
	}
}

func TestCSRF_ExemptPaths(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(
		config.WithCSRFHMACKey(testHMACKey),
		config.WithCSRFExemptPaths([]string{"/api/webhook", "/public/"}),
	)(handler)

	// Test exact path match
	req1 := httptest.NewRequest(http.MethodPost, "/api/webhook", nil)
	rr1 := httptest.NewRecorder()
	csrf.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("Expected status 200 for exempt path, got %d", rr1.Code)
	}

	// Test prefix path match
	req2 := httptest.NewRequest(http.MethodPost, "/public/something", nil)
	rr2 := httptest.NewRecorder()
	csrf.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("Expected status 200 for exempt prefix path, got %d", rr2.Code)
	}

	// Test non-exempt path (should require token)
	req3 := httptest.NewRequest(http.MethodPost, "/api/other", nil)
	rr3 := httptest.NewRecorder()
	csrf.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for non-exempt path, got %d", rr3.Code)
	}
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

	csrf := CSRF(
		config.WithCSRFHMACKey(testHMACKey),
		config.WithCSRFErrorHandler(customHandler),
	)(handler)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()
	csrf.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	if !strings.Contains(rr.Body.String(), "custom csrf error") {
		t.Errorf("Expected custom error message, got %s", rr.Body.String())
	}
}

func TestCSRF_FormTokenLookup(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(
		config.WithCSRFHMACKey(testHMACKey),
		config.WithCSRFTokenLookup("form:csrf_token"),
	)(handler)

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

	if rr2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr2.Code)
	}
}

func TestCSRF_QueryTokenLookup(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(
		config.WithCSRFHMACKey(testHMACKey),
		config.WithCSRFTokenLookup("query:csrf_token"),
	)(handler)

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

	if rr2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr2.Code)
	}
}

func TestCSRF_CustomCookieOptions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(
		config.WithCSRFHMACKey(testHMACKey),
		config.WithCSRFCookieName("custom_csrf"),
		config.WithCSRFCookieMaxAge(3600),
		config.WithCSRFCookieDomain("example.com"),
		config.WithCSRFCookiePath("/api"),
		config.WithCSRFCookieSecure(false),
		config.WithCSRFCookieSameSite(http.SameSiteLaxMode),
	)(handler)

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

	csrf := CSRF(config.WithCSRFHMACKey(testHMACKey))(handler)

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

	if rr2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr2.Code)
	}

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

	csrf := CSRF(config.WithCSRFHMACKey(testHMACKey))(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	csrf.ServeHTTP(rr, req)

	// Should have a non-empty token in response body
	token := rr.Body.String()
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

	csrf := CSRF(config.WithCSRFHMACKey(testHMACKey))(handler)

	// Make POST request with invalid base64 token
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-CSRF-Token", "!!!invalid-base64!!!")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "!!!invalid-base64!!!"})

	rr := httptest.NewRecorder()
	csrf.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rr.Code)
	}
}

func TestCSRF_TokenWithWrongSignature(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Use a specific HMAC key
	hmacKey := []byte("test-key-for-csrf-middleware-32!")
	csrf := CSRF(
		config.WithCSRFHMACKey(hmacKey),
	)(handler)

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
	csrf2 := CSRF(
		config.WithCSRFHMACKey(hmacKey2),
	)(handler)

	// Try to use token from first middleware with second middleware
	req2 := httptest.NewRequest(http.MethodPost, "/", nil)
	req2.Header.Set("X-CSRF-Token", validToken)
	req2.AddCookie(&http.Cookie{Name: "csrf_token", Value: validToken})

	rr2 := httptest.NewRecorder()
	csrf2.ServeHTTP(rr2, req2)

	// Should fail because HMAC key is different
	if rr2.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for token with wrong signature, got %d", rr2.Code)
	}
}

func TestCSRF_CustomExemptMethods(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Only exempt PUT, not GET
	csrf := CSRF(
		config.WithCSRFHMACKey(testHMACKey),
		config.WithCSRFExemptMethods([]string{http.MethodPut}),
	)(handler)

	// GET without token should return 403 (not exempt)
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr1 := httptest.NewRecorder()
	csrf.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for GET without token (not exempt), got %d", rr1.Code)
	}

	// PUT should be exempt and work without token
	req2 := httptest.NewRequest(http.MethodPut, "/", nil)
	rr2 := httptest.NewRecorder()
	csrf.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("Expected status 200 for exempt PUT, got %d", rr2.Code)
	}
}

func TestCSRF_EmptyCookieValue(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.WithCSRFHMACKey(testHMACKey))(handler)

	// POST with empty cookie value
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-CSRF-Token", "some-token")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: ""})

	rr := httptest.NewRecorder()
	csrf.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for empty cookie, got %d", rr.Code)
	}
}

func TestCSRF_InvalidTokenFormatRegeneration(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(config.WithCSRFHMACKey(testHMACKey))(handler)

	// GET request with invalid token format cookie should regenerate
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "invalid-base64-token"})

	rr := httptest.NewRecorder()
	csrf.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

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
	csrf := CSRF(
		config.WithCSRFHMACKey(testHMACKey),
		config.WithCSRFTokenLookup("malformed"),
	)(handler)

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
	if rr2.Code != http.StatusOK {
		t.Errorf("Expected status 200 with default header lookup, got %d", rr2.Code)
	}
}

func TestCSRF_UnknownTokenLookupSource(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Use unknown source to trigger default case in extractToken
	csrf := CSRF(
		config.WithCSRFHMACKey(testHMACKey),
		config.WithCSRFTokenLookup("unknown:token"),
	)(handler)

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
	if rr2.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 when token not in header, got %d", rr2.Code)
	}
}

func TestCSRF_FormParseError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	csrf := CSRF(
		config.WithCSRFHMACKey(testHMACKey),
		config.WithCSRFTokenLookup("form:csrf_token"),
	)(handler)

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
	if rr2.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for invalid form, got %d", rr2.Code)
	}
}
