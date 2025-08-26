package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

func TestTrailingSlash_PreferTrailingSlash(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("path: " + r.URL.Path))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})
	middleware := TrailingSlash(
		config.WithTrailingSlashPreference(true),
		config.WithTrailingSlashAction(config.RedirectAction),
		config.WithTrailingSlashRedirectCode(http.StatusMovedPermanently),
	)(handler)
	tests := []struct {
		name, requestPath, expectedPath, expectedHeader string
		expectedCode                                    int
	}{
		{"Root path unchanged", "/", "/", "", http.StatusOK},
		{"Path without trailing slash redirects", "/api/users", "", "/api/users/", http.StatusMovedPermanently},
		{"Path with trailing slash passes", "/api/users/", "/api/users/", "", http.StatusOK},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.requestPath, nil)
			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)
			if w.Code != tc.expectedCode {
				t.Errorf("Expected status %d, got %d", tc.expectedCode, w.Code)
			}
			if tc.expectedCode == http.StatusMovedPermanently {
				location := w.Header().Get("Location")
				if location != tc.expectedHeader {
					t.Errorf("Expected redirect to %s, got %s", tc.expectedHeader, location)
				}
			} else {
				body := w.Body.String()
				expectedBody := "path: " + tc.expectedPath
				if body != expectedBody {
					t.Errorf("Expected body %s, got %s", expectedBody, body)
				}
			}
		})
	}
}

func TestTrailingSlash_StripAction(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("path: " + r.URL.Path))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})
	middleware := TrailingSlash(
		config.WithTrailingSlashAction(config.StripAction),
		config.WithTrailingSlashPreference(false),
	)(handler)
	tests := []struct {
		name, requestPath, expectedPath string
		expectedCode                    int
	}{
		{"Path with trailing slash gets stripped", "/api/users/", "/api/users", http.StatusOK},
		{"Path without trailing slash unchanged", "/api/users", "/api/users", http.StatusOK},
		{"Root path unchanged", "/", "/", http.StatusOK},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.requestPath, nil)
			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)
			if w.Code != tc.expectedCode {
				t.Errorf("Expected status %d, got %d", tc.expectedCode, w.Code)
			}
			body := w.Body.String()
			expectedBody := "path: " + tc.expectedPath
			if body != expectedBody {
				t.Errorf("Expected body %s, got %s", expectedBody, body)
			}
		})
	}
}

func TestTrailingSlash_AppendAction(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("path: " + r.URL.Path))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})
	middleware := TrailingSlash(
		config.WithTrailingSlashAction(config.AppendAction),
		config.WithTrailingSlashPreference(true),
	)(handler)
	tests := []struct {
		name, requestPath, expectedPath string
		expectedCode                    int
	}{
		{"Path without trailing slash gets appended", "/api/users", "/api/users/", http.StatusOK},
		{"Path with trailing slash unchanged", "/api/users/", "/api/users/", http.StatusOK},
		{"Root path unchanged", "/", "/", http.StatusOK},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.requestPath, nil)
			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)
			if w.Code != tc.expectedCode {
				t.Errorf("Expected status %d, got %d", tc.expectedCode, w.Code)
			}
			body := w.Body.String()
			expectedBody := "path: " + tc.expectedPath
			if body != expectedBody {
				t.Errorf("Expected body %s, got %s", expectedBody, body)
			}
		})
	}
}

func TestTrailingSlash_CustomRedirectCode(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	middleware := TrailingSlash(
		config.WithTrailingSlashAction(config.RedirectAction),
		config.WithTrailingSlashPreference(false),
		config.WithTrailingSlashRedirectCode(http.StatusFound),
	)(handler)
	req := httptest.NewRequest("GET", "/api/users/", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Errorf("Expected status %d, got %d", http.StatusFound, w.Code)
	}
	if location := w.Header().Get("Location"); location != "/api/users" {
		t.Errorf("Expected redirect to /api/users, got %s", location)
	}
}

func TestTrailingSlash_WithQueryString(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	middleware := TrailingSlash(
		config.WithTrailingSlashAction(config.RedirectAction),
		config.WithTrailingSlashPreference(false),
		config.WithTrailingSlashRedirectCode(http.StatusMovedPermanently),
	)(handler)
	req := httptest.NewRequest("GET", "/api/users/?param=value&other=test", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)
	if w.Code != http.StatusMovedPermanently {
		t.Errorf("Expected status %d, got %d", http.StatusMovedPermanently, w.Code)
	}
	expected := "/api/users?param=value&other=test"
	if location := w.Header().Get("Location"); location != expected {
		t.Errorf("Expected redirect to %s, got %s", expected, location)
	}
}

func TestTrailingSlash_WithFragment(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	middleware := TrailingSlash(
		config.WithTrailingSlashAction(config.RedirectAction),
		config.WithTrailingSlashPreference(true),
		config.WithTrailingSlashRedirectCode(http.StatusMovedPermanently),
	)(handler)
	targetURL, _ := url.Parse("/api/users?param=value#section")
	req := &http.Request{Method: "GET", URL: targetURL, Header: make(http.Header)}
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)
	if w.Code != http.StatusMovedPermanently {
		t.Errorf("Expected status %d, got %d", http.StatusMovedPermanently, w.Code)
	}
	expected := "/api/users/?param=value#section"
	if location := w.Header().Get("Location"); location != expected {
		t.Errorf("Expected redirect to %s, got %s", expected, location)
	}
}

func TestTrailingSlash_ConfigEdgeCases(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	t.Run("Empty action uses default", func(t *testing.T) {
		middleware := TrailingSlash(config.WithTrailingSlashAction(""), config.WithTrailingSlashPreference(false))(handler)
		req := httptest.NewRequest("GET", "/api/users/", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
		if w.Code != http.StatusMovedPermanently {
			t.Errorf("Expected default redirect status %d, got %d", http.StatusMovedPermanently, w.Code)
		}
	})
	t.Run("Zero redirect code uses default", func(t *testing.T) {
		middleware := TrailingSlash(config.WithTrailingSlashAction(config.RedirectAction), config.WithTrailingSlashPreference(false), config.WithTrailingSlashRedirectCode(0))(handler)
		req := httptest.NewRequest("GET", "/api/users/", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
		if w.Code != http.StatusMovedPermanently {
			t.Errorf("Expected default redirect code %d, got %d", http.StatusMovedPermanently, w.Code)
		}
	})
	t.Run("Invalid action is pass-through", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("path: " + r.URL.Path))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		})
		middleware := TrailingSlash(config.WithTrailingSlashAction("invalid"), config.WithTrailingSlashPreference(false))(handler)
		req := httptest.NewRequest("GET", "/api/users/", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		body := w.Body.String()
		expected := "path: /api/users/"
		if body != expected {
			t.Errorf("Expected body %s, got %s", expected, body)
		}
	})
}

func TestTrailingSlash_MultipleOptions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("path: " + r.URL.Path))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})
	middleware := TrailingSlash(
		config.WithTrailingSlashAction(config.StripAction),
		config.WithTrailingSlashAction(config.AppendAction),
		config.WithTrailingSlashPreference(true),
	)(handler)
	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)
	body := w.Body.String()
	expected := "path: /api/users/"
	if body != expected {
		t.Errorf("Expected last option to be used, got %s", body)
	}
}

func TestTrailingSlash_DifferentHTTPMethods(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	middleware := TrailingSlash(
		config.WithTrailingSlashAction(config.RedirectAction),
		config.WithTrailingSlashPreference(false),
	)(handler)
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/users/", nil)
			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)
			if w.Code != http.StatusMovedPermanently {
				t.Errorf("Expected redirect for %s method, got status %d", method, w.Code)
			}
			if location := w.Header().Get("Location"); location != "/api/users" {
				t.Errorf("Expected redirect to /api/users for %s method, got %s", method, location)
			}
		})
	}
}

func TestTrailingSlash_ComplexPaths(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("path: " + r.URL.Path))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})
	middleware := TrailingSlash(
		config.WithTrailingSlashAction(config.StripAction),
		config.WithTrailingSlashPreference(false),
	)(handler)
	tests := []struct {
		name, requestPath, expectedPath string
	}{
		{"Nested path with trailing slash", "/api/v1/users/123/posts/", "/api/v1/users/123/posts"},
		{"Path with dots", "/api/users/user.json/", "/api/users/user.json"},
		{"Path with dashes and underscores", "/api/some-endpoint/sub_path/", "/api/some-endpoint/sub_path"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.requestPath, nil)
			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)
			body := w.Body.String()
			expected := "path: " + tc.expectedPath
			if body != expected {
				t.Errorf("Expected body %s, got %s", expected, body)
			}
		})
	}
}

func TestDefaultTrailingSlashConfig(t *testing.T) {
	cfg := config.DefaultTrailingSlashConfig
	if cfg.Action != config.RedirectAction {
		t.Errorf("Expected default action %s, got %s", config.RedirectAction, cfg.Action)
	}
	if cfg.PreferTrailingSlash != false {
		t.Errorf("Expected default PreferTrailingSlash false, got %t", cfg.PreferTrailingSlash)
	}
	if cfg.RedirectCode != http.StatusMovedPermanently {
		t.Errorf("Expected default redirect code %d, got %d", http.StatusMovedPermanently, cfg.RedirectCode)
	}
}

func TestTrailingSlash_PreserveRequestData(t *testing.T) {
	var capturedMethod string
	var capturedHeaders http.Header
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	})
	middleware := TrailingSlash(
		config.WithTrailingSlashAction(config.StripAction),
		config.WithTrailingSlashPreference(false),
	)(handler)
	req := httptest.NewRequest("POST", "/api/users/", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token123")
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)
	if capturedMethod != "POST" {
		t.Errorf("Expected method POST to be preserved, got %s", capturedMethod)
	}
	if capturedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type header to be preserved")
	}
	if capturedHeaders.Get("Authorization") != "Bearer token123" {
		t.Errorf("Expected Authorization header to be preserved")
	}
}
