package middleware

import (
	"net/http"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestHostValidation_Disabled(t *testing.T) {
	// When AllowedHosts is empty, all hosts should be allowed
	mw := HostValidation(config.HostValidationConfig{
		AllowedHosts: []string{},
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.Host = "evil.com"
	w := zhtest.Serve(handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
}

func TestHostValidation_ExactMatch(t *testing.T) {
	mw := HostValidation(config.HostValidationConfig{
		AllowedHosts: []string{"api.example.com", "example.com"},
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		host       string
		wantStatus int
	}{
		{"allowed exact match", "api.example.com", http.StatusOK},
		{"another allowed host", "example.com", http.StatusOK},
		{"disallowed host", "evil.com", http.StatusBadRequest},
		{"subdomain not allowed", "sub.example.com", http.StatusBadRequest},
		{"different TLD", "example.org", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := zhtest.NewRequest(http.MethodGet, "/test").Build()
			req.Host = tt.host
			w := zhtest.Serve(handler, req)

			zhtest.AssertWith(t, w).Status(tt.wantStatus)
		})
	}
}

func TestHostValidation_Subdomains(t *testing.T) {
	mw := HostValidation(config.HostValidationConfig{
		AllowedHosts:    []string{"example.com"},
		AllowSubdomains: true,
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		host       string
		wantStatus int
	}{
		{"exact match", "example.com", http.StatusOK},
		{"subdomain", "api.example.com", http.StatusOK},
		{"nested subdomain", "v1.api.example.com", http.StatusOK},
		{"different domain", "evil.com", http.StatusBadRequest},
		{"similar domain", "notexample.com", http.StatusBadRequest},
		{"partial match", "example.com.evil.com", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := zhtest.NewRequest(http.MethodGet, "/test").Build()
			req.Host = tt.host
			w := zhtest.Serve(handler, req)

			zhtest.AssertWith(t, w).Status(tt.wantStatus)
		})
	}
}

func TestHostValidation_CaseInsensitive(t *testing.T) {
	mw := HostValidation(config.HostValidationConfig{
		AllowedHosts: []string{"Example.COM"},
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		host       string
		wantStatus int
	}{
		{"lowercase", "example.com", http.StatusOK},
		{"uppercase", "EXAMPLE.COM", http.StatusOK},
		{"mixed case", "ExAmPlE.CoM", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := zhtest.NewRequest(http.MethodGet, "/test").Build()
			req.Host = tt.host
			w := zhtest.Serve(handler, req)

			zhtest.AssertWith(t, w).Status(tt.wantStatus)
		})
	}
}

func TestHostValidation_WithPort(t *testing.T) {
	mw := HostValidation(config.HostValidationConfig{
		AllowedHosts: []string{"example.com"},
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		host       string
		wantStatus int
	}{
		{"no port", "example.com", http.StatusOK},
		{"port 8080", "example.com:8080", http.StatusOK},
		{"port 443", "example.com:443", http.StatusOK},
		{"wrong host with port", "evil.com:8080", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := zhtest.NewRequest(http.MethodGet, "/test").Build()
			req.Host = tt.host
			w := zhtest.Serve(handler, req)

			zhtest.AssertWith(t, w).Status(tt.wantStatus)
		})
	}
}

func TestHostValidation_ExemptPaths(t *testing.T) {
	mw := HostValidation(config.HostValidationConfig{
		AllowedHosts: []string{"example.com"},
		ExemptPaths:  []string{"/health", "/metrics"},
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		path       string
		host       string
		wantStatus int
	}{
		{"exempt path with bad host", "/health", "evil.com", http.StatusOK},
		{"exempt path 2", "/metrics", "evil.com", http.StatusOK},
		{"non-exempt path with bad host", "/api", "evil.com", http.StatusBadRequest},
		{"non-exempt path with good host", "/api", "example.com", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := zhtest.NewRequest(http.MethodGet, tt.path).Build()
			req.Host = tt.host
			w := zhtest.Serve(handler, req)

			zhtest.AssertWith(t, w).Status(tt.wantStatus)
		})
	}
}

func TestHostValidation_CustomStatusCodeAndMessage(t *testing.T) {
	mw := HostValidation(config.HostValidationConfig{
		AllowedHosts: []string{"example.com"},
		StatusCode:   http.StatusForbidden,
		Message:      "Forbidden host",
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test JSON response
	req := zhtest.NewRequest(http.MethodGet, "/test").WithHeader("Accept", "application/json").Build()
	req.Host = "evil.com"
	w := zhtest.Serve(handler, req)
	zhtest.AssertWith(t, w).Status(http.StatusForbidden).IsProblemDetail().ProblemDetailDetail("Forbidden host")

	// Test plain text response
	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.Host = "evil.com"
	w = zhtest.Serve(handler, req)
	zhtest.AssertWith(t, w).Status(http.StatusForbidden).Header("Content-Type", "text/plain; charset=utf-8")
}

func TestHostValidation_DefaultsFallback(t *testing.T) {
	mw := HostValidation(config.HostValidationConfig{
		AllowedHosts: []string{"example.com"},
		StatusCode:   0,  // Should use default
		Message:      "", // Should use default
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test JSON response
	req := zhtest.NewRequest(http.MethodGet, "/test").WithHeader("Accept", "application/json").Build()
	req.Host = "evil.com"
	w := zhtest.Serve(handler, req)
	zhtest.AssertWith(t, w).Status(http.StatusBadRequest).IsProblemDetail().ProblemDetailDetail("Invalid Host header")

	// Test plain text response
	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.Host = "evil.com"
	w = zhtest.Serve(handler, req)
	zhtest.AssertWith(t, w).Status(http.StatusBadRequest).Header("Content-Type", "text/plain; charset=utf-8")
}

func TestHostValidation_StrictPort(t *testing.T) {
	tests := []struct {
		name       string
		host       string
		strictPort bool
		port       int
		wantStatus int
	}{
		// StrictPort disabled
		{"strict port disabled - no port in header", "example.com", false, 8080, http.StatusOK},
		{"strict port disabled - with port in header", "example.com:8080", false, 8080, http.StatusOK},

		// StrictPort enabled with non-standard port (8080)
		{"strict port - correct port", "example.com:8080", true, 8080, http.StatusOK},
		{"strict port - missing port", "example.com", true, 8080, http.StatusBadRequest},
		{"strict port - wrong port", "example.com:9090", true, 8080, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := HostValidation(config.HostValidationConfig{
				AllowedHosts: []string{"example.com"},
				StrictPort:   tt.strictPort,
				Port:         tt.port,
			})
			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := zhtest.NewRequest(http.MethodGet, "/test").Build()
			req.Host = tt.host
			w := zhtest.Serve(handler, req)

			zhtest.AssertWith(t, w).Status(tt.wantStatus)
		})
	}
}

func TestHostValidation_IPv6(t *testing.T) {
	tests := []struct {
		name       string
		host       string
		wantStatus int
	}{
		// IPv6 with port
		{"IPv6 localhost with port", "[::1]:8080", http.StatusOK},
		{"IPv6 full with port", "[2001:db8::1]:443", http.StatusOK},

		// IPv6 without port
		{"IPv6 localhost no port", "::1", http.StatusOK},
		{"IPv6 full no port", "2001:db8::1", http.StatusOK},
		{"IPv6 bracketed no port", "[::1]", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := HostValidation(config.HostValidationConfig{
				AllowedHosts: []string{"::1", "2001:db8::1"},
			})
			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := zhtest.NewRequest(http.MethodGet, "/test").Build()
			req.Host = tt.host
			w := zhtest.Serve(handler, req)

			zhtest.AssertWith(t, w).Status(tt.wantStatus)
		})
	}
}

func TestHostValidation_EmptyHost(t *testing.T) {
	mw := HostValidation(config.HostValidationConfig{
		AllowedHosts: []string{"example.com"},
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.Host = ""
	w := zhtest.Serve(handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusBadRequest)
}

func TestHostValidation_AllowedHostsWithPort(t *testing.T) {
	// Ports in AllowedHosts should be stripped automatically
	mw := HostValidation(config.HostValidationConfig{
		AllowedHosts: []string{"example.com:8080", "api.example.com:443"},
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		host       string
		wantStatus int
	}{
		{"host with matching port stripped", "example.com:8080", http.StatusOK},
		{"host with different port", "example.com:9090", http.StatusOK},
		{"host without port", "example.com", http.StatusOK},
		{"api host with port", "api.example.com:443", http.StatusOK},
		{"wrong host", "evil.com:8080", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := zhtest.NewRequest(http.MethodGet, "/test").Build()
			req.Host = tt.host
			w := zhtest.Serve(handler, req)

			zhtest.AssertWith(t, w).Status(tt.wantStatus)
		})
	}
}

func TestHostValidation_IPv6BracketStripping(t *testing.T) {
	// IPv6 in AllowedHosts with brackets should be stripped
	mw := HostValidation(config.HostValidationConfig{
		AllowedHosts: []string{"[::1]", "[2001:db8::1]"},
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		host       string
		wantStatus int
	}{
		{"allowed host brackets stripped", "::1", http.StatusOK},
		{"allowed host with port", "[::1]:8080", http.StatusOK},
		{"bracketed allowed host", "[::1]", http.StatusOK},
		{"different IPv6", "[2001:db8::1]", http.StatusOK},
		{"not allowed IPv6", "[::2]", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := zhtest.NewRequest(http.MethodGet, "/test").Build()
			req.Host = tt.host
			w := zhtest.Serve(handler, req)

			zhtest.AssertWith(t, w).Status(tt.wantStatus)
		})
	}
}

func TestHostValidation_StrictPortPanic(t *testing.T) {
	tests := []struct {
		name        string
		port        int
		expectedMsg string
	}{
		{
			name:        "Port not set",
			port:        0,
			expectedMsg: "StrictPort requires Port",
		},
		{
			name:        "Port 80 is standard",
			port:        80,
			expectedMsg: "no effect on standard ports",
		},
		{
			name:        "Port 443 is standard",
			port:        443,
			expectedMsg: "no effect on standard ports",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("expected panic, but got none")
				} else if !strings.Contains(r.(string), tt.expectedMsg) {
					t.Errorf("expected panic containing %q, got: %v", tt.expectedMsg, r)
				}
			}()

			HostValidation(config.HostValidationConfig{
				AllowedHosts: []string{"example.com"},
				StrictPort:   true,
				Port:         tt.port,
			})
		})
	}
}

// TestHostValidation_Comprehensive covers all edge cases in a single table-driven test.
// This documents the complete behavior matrix for the middleware.
func TestHostValidation_Comprehensive(t *testing.T) {
	tests := []struct {
		name            string
		allowedHosts    []string
		allowSubdomains bool
		strictPort      bool
		port            int
		host            string
		wantStatus      int
	}{
		// Exact match cases
		{"exact match", []string{"example.com"}, false, false, 0, "example.com", http.StatusOK},
		{"case insensitive exact", []string{"example.com"}, false, false, 0, "EXAMPLE.COM", http.StatusOK},
		{"FQDN trailing dot", []string{"example.com"}, false, false, 0, "example.com.", http.StatusOK},
		{"wrong host rejected", []string{"example.com"}, false, false, 0, "evil.com", http.StatusBadRequest},

		// Subdomain cases
		{"subdomain allowed", []string{"example.com"}, true, false, 0, "api.example.com", http.StatusOK},
		{"nested subdomain", []string{"example.com"}, true, false, 0, "v1.api.example.com", http.StatusOK},
		{"subdomain not allowed", []string{"example.com"}, false, false, 0, "api.example.com", http.StatusBadRequest},
		{"similar domain rejected", []string{"example.com"}, true, false, 0, "evilexample.com", http.StatusBadRequest},

		// Port stripping from request
		{"port stripped from request", []string{"example.com"}, false, false, 0, "example.com:8080", http.StatusOK},
		{"port stripped from request 443", []string{"example.com"}, false, false, 0, "example.com:443", http.StatusOK},

		// Port stripping from config
		{"port stripped from config", []string{"example.com:8080"}, false, false, 0, "example.com", http.StatusOK},
		{"port stripped from config request with port", []string{"example.com:8080"}, false, false, 0, "example.com:9090", http.StatusOK},

		// IPv6 cases
		{"IPv6 with port", []string{"::1"}, false, false, 0, "[::1]:8080", http.StatusOK},
		{"IPv6 without port", []string{"::1"}, false, false, 0, "[::1]", http.StatusOK},
		{"IPv6 bare", []string{"::1"}, false, false, 0, "::1", http.StatusOK},
		{"IPv6 bracketed in config", []string{"[::1]"}, false, false, 0, "::1", http.StatusOK},
		{"IPv6 wrong host", []string{"::1"}, false, false, 0, "::2", http.StatusBadRequest},

		// StrictPort cases
		{"strict port correct", []string{"example.com"}, false, true, 8080, "example.com:8080", http.StatusOK},
		{"strict port missing", []string{"example.com"}, false, true, 8080, "example.com", http.StatusBadRequest},
		{"strict port wrong", []string{"example.com"}, false, true, 8080, "example.com:9090", http.StatusBadRequest},

		// Edge cases
		{"empty host", []string{"example.com"}, false, false, 0, "", http.StatusBadRequest},
		{"partial match suffix", []string{"example.com"}, true, false, 0, "example.com.evil.com", http.StatusBadRequest},
		{"different TLD", []string{"example.com"}, false, false, 0, "example.org", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := HostValidation(config.HostValidationConfig{
				AllowedHosts:    tt.allowedHosts,
				AllowSubdomains: tt.allowSubdomains,
				StrictPort:      tt.strictPort,
				Port:            tt.port,
			})
			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := zhtest.NewRequest(http.MethodGet, "/test").Build()
			req.Host = tt.host
			w := zhtest.Serve(handler, req)

			zhtest.AssertWith(t, w).Status(tt.wantStatus)
		})
	}
}

func TestHostValidation_TrailingDot(t *testing.T) {
	// FQDN trailing dot should be handled
	mw := HostValidation(config.HostValidationConfig{
		AllowedHosts:    []string{"example.com"},
		AllowSubdomains: true,
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		host       string
		wantStatus int
	}{
		{"normal host", "example.com", http.StatusOK},
		{"FQDN with trailing dot", "example.com.", http.StatusOK},
		{"subdomain with trailing dot", "api.example.com.", http.StatusOK},
		{"wrong host with trailing dot", "evil.com.", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := zhtest.NewRequest(http.MethodGet, "/test").Build()
			req.Host = tt.host
			w := zhtest.Serve(handler, req)

			zhtest.AssertWith(t, w).Status(tt.wantStatus)
		})
	}
}

func TestIsValidHost(t *testing.T) {
	tests := []struct {
		name            string
		host            string
		allowed         []string
		allowSubdomains bool
		want            bool
	}{
		// Exact matches
		{"exact match", "example.com", []string{"example.com"}, false, true},
		{"no match", "evil.com", []string{"example.com"}, false, false},

		// Subdomains
		{"subdomain allowed", "api.example.com", []string{"example.com"}, true, true},
		{"subdomain not allowed", "api.example.com", []string{"example.com"}, false, false},
		{"nested subdomain", "v1.api.example.com", []string{"example.com"}, true, true},

		// Edge cases
		{"partial match rejected", "notexample.com", []string{"example.com"}, true, false},
		{"suffix match rejected", "example.com.evil.com", []string{"example.com"}, true, false},
		{"empty allowed list", "example.com", []string{}, false, false},

		// Multiple allowed hosts
		{"second host match", "api.example.com", []string{"example.com", "api.example.com"}, false, true},
		{"third host match", "other.com", []string{"example.com", "api.example.com", "other.com"}, false, true},

		// Case insensitive
		{"case insensitive", "EXAMPLE.COM", []string{"example.com"}, false, true},
		{"mixed case", "Api.Example.Com", []string{"example.com"}, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidHost(tt.host, tt.allowed, tt.allowSubdomains)
			if got != tt.want {
				t.Errorf("isValidHost(%q, %v, %v) = %v, want %v",
					tt.host, tt.allowed, tt.allowSubdomains, got, tt.want)
			}
		})
	}
}
