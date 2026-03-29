package trailingslash

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestTrailingSlash_PreferTrailingSlash(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("path: " + r.URL.Path))
	})
	middleware := New(Config{
		PreferTrailingSlash: true,
		Action:              RedirectAction,
		RedirectCode:        http.StatusMovedPermanently,
	})(handler)
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
			req := zhtest.NewRequest(http.MethodGet, tc.requestPath).Build()
			w := zhtest.Serve(middleware, req)
			zhtest.AssertEqual(t, tc.expectedCode, w.Code)
			if tc.expectedCode == http.StatusMovedPermanently {
				zhtest.AssertWith(t, w).Header("Location", tc.expectedHeader)
			} else {
				zhtest.AssertWith(t, w).Body("path: " + tc.expectedPath)
			}
		})
	}
}

func TestTrailingSlash_StripAction(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("path: " + r.URL.Path))
	})
	middleware := New(Config{
		Action:              StripAction,
		PreferTrailingSlash: false,
	})(handler)
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
			req := zhtest.NewRequest(http.MethodGet, tc.requestPath).Build()
			w := zhtest.Serve(middleware, req)

			zhtest.AssertWith(t, w).Status(tc.expectedCode).Body("path: " + tc.expectedPath)
		})
	}
}

func TestTrailingSlash_AppendAction(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("path: " + r.URL.Path))
	})
	middleware := New(Config{
		Action:              AppendAction,
		PreferTrailingSlash: true,
	})(handler)
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
			req := zhtest.NewRequest(http.MethodGet, tc.requestPath).Build()
			w := zhtest.Serve(middleware, req)

			zhtest.AssertWith(t, w).Status(tc.expectedCode).Body("path: " + tc.expectedPath)
		})
	}
}

func TestTrailingSlash_CustomRedirectCode(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	middleware := New(Config{
		Action:              RedirectAction,
		PreferTrailingSlash: false,
		RedirectCode:        http.StatusFound,
	})(handler)
	req := zhtest.NewRequest(http.MethodGet, "/api/users/").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusFound).Header("Location", "/api/users")
}

func TestTrailingSlash_WithQueryString(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	middleware := New(Config{
		Action:              RedirectAction,
		PreferTrailingSlash: false,
		RedirectCode:        http.StatusMovedPermanently,
	})(handler)
	req := zhtest.NewRequest(http.MethodGet, "/api/users/?param=value&other=test").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusMovedPermanently).Header("Location", "/api/users?param=value&other=test")
}

func TestTrailingSlash_WithFragment(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	middleware := New(Config{
		Action:              RedirectAction,
		PreferTrailingSlash: true,
		RedirectCode:        http.StatusMovedPermanently,
	})(handler)
	targetURL, _ := url.Parse("/api/users?param=value#section")
	req := &http.Request{Method: http.MethodGet, URL: targetURL, Header: make(http.Header)}
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Status(http.StatusMovedPermanently).Header("Location", "/api/users/?param=value#section")
}

func TestTrailingSlash_ConfigEdgeCases(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	t.Run("Empty action uses default", func(t *testing.T) {
		middleware := New(Config{Action: "", PreferTrailingSlash: false})(handler)
		req := zhtest.NewRequest(http.MethodGet, "/api/users/").Build()
		w := zhtest.Serve(middleware, req)

		zhtest.AssertWith(t, w).Status(http.StatusMovedPermanently)
	})
	t.Run("Zero redirect code uses default", func(t *testing.T) {
		middleware := New(Config{Action: RedirectAction, PreferTrailingSlash: false, RedirectCode: 0})(handler)
		req := zhtest.NewRequest(http.MethodGet, "/api/users/").Build()
		w := zhtest.Serve(middleware, req)

		zhtest.AssertWith(t, w).Status(http.StatusMovedPermanently)
	})
	t.Run("Invalid action is pass-through", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("path: " + r.URL.Path))
		})
		middleware := New(Config{Action: "invalid", PreferTrailingSlash: false})(handler)
		req := zhtest.NewRequest(http.MethodGet, "/api/users/").Build()
		w := zhtest.Serve(middleware, req)

		zhtest.AssertWith(t, w).Status(http.StatusOK).Body("path: /api/users/")
	})
}

func TestTrailingSlash_MultipleOptions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("path: " + r.URL.Path))
	})
	middleware := New(Config{
		Action:              AppendAction,
		PreferTrailingSlash: true,
	})(handler)
	req := zhtest.NewRequest(http.MethodGet, "/api/users").Build()
	w := zhtest.Serve(middleware, req)

	zhtest.AssertWith(t, w).Body("path: /api/users/")
}

func TestTrailingSlash_DifferentHTTPMethods(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	middleware := New(Config{
		Action:              RedirectAction,
		PreferTrailingSlash: false,
	})(handler)
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodHead, http.MethodOptions}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := zhtest.NewRequest(method, "/api/users/").Build()
			w := zhtest.Serve(middleware, req)

			zhtest.AssertWith(t, w).Status(http.StatusMovedPermanently).Header("Location", "/api/users")
		})
	}
}

func TestTrailingSlash_ComplexPaths(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("path: " + r.URL.Path))
	})
	middleware := New(Config{
		Action:              StripAction,
		PreferTrailingSlash: false,
	})(handler)
	tests := []struct {
		name, requestPath, expectedPath string
	}{
		{"Nested path with trailing slash", "/api/v1/users/123/posts/", "/api/v1/users/123/posts"},
		{"Path with dots", "/api/users/user.json/", "/api/users/user.json"},
		{"Path with dashes and underscores", "/api/some-endpoint/sub_path/", "/api/some-endpoint/sub_path"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := zhtest.NewRequest(http.MethodGet, tc.requestPath).Build()
			w := zhtest.Serve(middleware, req)

			zhtest.AssertWith(t, w).Body("path: " + tc.expectedPath)
		})
	}
}

func TestDefaultTrailingSlashConfig(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertEqual(t, RedirectAction, cfg.Action)
	zhtest.AssertFalse(t, cfg.PreferTrailingSlash)
	zhtest.AssertEqual(t, http.StatusMovedPermanently, cfg.RedirectCode)
}

func TestTrailingSlash_PreserveRequestData(t *testing.T) {
	var capturedMethod string
	var capturedHeaders http.Header
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	})
	middleware := New(Config{
		Action:              StripAction,
		PreferTrailingSlash: false,
	})(handler)
	req := zhtest.NewRequest(http.MethodPost, "/api/users/").
		WithHeader(httpx.HeaderContentType, "application/json").
		WithHeader(httpx.HeaderAuthorization, "Bearer token123").
		Build()
	zhtest.Serve(middleware, req)
	zhtest.AssertEqual(t, http.MethodPost, capturedMethod)
	zhtest.AssertEqual(t, "application/json", capturedHeaders.Get(httpx.HeaderContentType))
	zhtest.AssertEqual(t, "Bearer token123", capturedHeaders.Get(httpx.HeaderAuthorization))
}
