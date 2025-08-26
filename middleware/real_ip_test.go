package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

func TestRealIPMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		remoteAddr     string
		headers        map[string]string
		expectedRemote string
	}{
		{"basic remote addr", "192.168.1.1:12345", map[string]string{}, "192.168.1.1:12345"},
		{"X-Forwarded-For single IP", "192.168.1.1:12345", map[string]string{"X-Forwarded-For": "203.0.113.1"}, "203.0.113.1:12345"},
		{"X-Forwarded-For multiple IPs", "192.168.1.1:12345", map[string]string{"X-Forwarded-For": "203.0.113.1, 198.51.100.1, 192.168.1.1"}, "203.0.113.1:12345"},
		{"X-Real-IP header", "192.168.1.1:12345", map[string]string{"X-Real-IP": "203.0.113.2"}, "203.0.113.2:12345"},
		{"X-Forwarded header", "192.168.1.1:12345", map[string]string{"X-Forwarded": "203.0.113.3"}, "203.0.113.3:12345"},
		{"Forwarded header RFC 7239", "192.168.1.1:12345", map[string]string{"Forwarded": `for="203.0.113.4"; proto=https`}, "203.0.113.4:12345"},
		{"Forwarded header without quotes", "192.168.1.1:12345", map[string]string{"Forwarded": "for=203.0.113.5; proto=https"}, "203.0.113.5:12345"},
	}
	middleware := RealIP()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for header, value := range tt.headers {
				req.Header.Set(header, value)
			}
			rr := httptest.NewRecorder()
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.RemoteAddr != tt.expectedRemote {
					t.Errorf("expected RemoteAddr '%s', got '%s'", tt.expectedRemote, r.RemoteAddr)
				}
				w.WriteHeader(http.StatusOK)
			})
			middleware(next).ServeHTTP(rr, req)
		})
	}
}

func TestRealIPMiddlewareNoPort(t *testing.T) {
	middleware := RealIP()
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	rr := httptest.NewRecorder()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RemoteAddr != "203.0.113.1" {
			t.Errorf("expected RemoteAddr '203.0.113.1', got '%s'", r.RemoteAddr)
		}
		w.WriteHeader(http.StatusOK)
	})
	middleware(next).ServeHTTP(rr, req)
}

func TestRealIPCustomExtractor(t *testing.T) {
	middleware := RealIP(config.WithRealIPExtractor(func(r *http.Request) string {
		return "custom.ip.address"
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RemoteAddr != "custom.ip.address:12345" {
			t.Errorf("expected RemoteAddr 'custom.ip.address:12345', got '%s'", r.RemoteAddr)
		}
		w.WriteHeader(http.StatusOK)
	})
	middleware(next).ServeHTTP(rr, req)
}

func TestRealIPNilExtractor(t *testing.T) {
	middleware := RealIP(config.WithRealIPExtractor(nil))
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	rr := httptest.NewRecorder()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RemoteAddr != "203.0.113.1:12345" {
			t.Errorf("expected RemoteAddr '203.0.113.1:12345', got '%s'", r.RemoteAddr)
		}
		w.WriteHeader(http.StatusOK)
	})
	middleware(next).ServeHTTP(rr, req)
}

func TestDefaultIPExtractor(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expectedIP string
	}{
		{"X-Forwarded-For priority", "192.168.1.1:12345", map[string]string{"X-Forwarded-For": "203.0.113.1"}, "203.0.113.1"},
		{"X-Forwarded-For with spaces", "192.168.1.1:12345", map[string]string{"X-Forwarded-For": " 203.0.113.1 , 198.51.100.1"}, "203.0.113.1"},
		{"X-Real-IP fallback", "192.168.1.1:12345", map[string]string{"X-Real-IP": "203.0.113.2"}, "203.0.113.2"},
		{"X-Forwarded fallback", "192.168.1.1:12345", map[string]string{"X-Forwarded": "203.0.113.3"}, "203.0.113.3"},
		{"Forwarded fallback", "192.168.1.1:12345", map[string]string{"Forwarded": "for=203.0.113.4; proto=https"}, "203.0.113.4"},
		{"Forwarded with quotes", "192.168.1.1:12345", map[string]string{"Forwarded": `for="203.0.113.5"; proto=https`}, "203.0.113.5"},
		{"Forwarded complex", "192.168.1.1:12345", map[string]string{"Forwarded": "proto=https; for=203.0.113.6; host=example.com"}, "203.0.113.6"},
		{"RemoteAddr fallback with port", "192.168.1.1:12345", map[string]string{}, "192.168.1.1"},
		{"RemoteAddr fallback without port", "192.168.1.1", map[string]string{}, "192.168.1.1"},
		{"IPv6 RemoteAddr", "[2001:db8::1]:12345", map[string]string{}, "2001:db8::1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for header, value := range tt.headers {
				req.Header.Set(header, value)
			}
			ip := config.DefaultIPExtractor(req)
			if ip != tt.expectedIP {
				t.Errorf("expected IP '%s', got '%s'", tt.expectedIP, ip)
			}
		})
	}
}

func TestRemoteAddrIPExtractor(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expectedIP string
	}{
		{"with port", "192.168.1.1:12345", map[string]string{"X-Forwarded-For": "203.0.113.1"}, "192.168.1.1"},
		{"without port", "192.168.1.1", map[string]string{"X-Forwarded-For": "203.0.113.1"}, "192.168.1.1"},
		{"IPv6 with port", "[2001:db8::1]:12345", map[string]string{}, "2001:db8::1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for header, value := range tt.headers {
				req.Header.Set(header, value)
			}
			ip := config.RemoteAddrIPExtractor(req)
			if ip != tt.expectedIP {
				t.Errorf("expected IP '%s', got '%s'", tt.expectedIP, ip)
			}
		})
	}
}

func TestXForwardedForIPExtractor(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expectedIP string
	}{
		{"X-Forwarded-For present", "192.168.1.1:12345", map[string]string{"X-Forwarded-For": "203.0.113.1, 198.51.100.1"}, "203.0.113.1"},
		{"X-Forwarded-For with spaces", "192.168.1.1:12345", map[string]string{"X-Forwarded-For": " 203.0.113.2 , 198.51.100.1"}, "203.0.113.2"},
		{"X-Forwarded-For missing", "192.168.1.1:12345", map[string]string{"X-Real-IP": "203.0.113.3"}, "192.168.1.1"},
		{"empty X-Forwarded-For", "192.168.1.1:12345", map[string]string{"X-Forwarded-For": ""}, "192.168.1.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for header, value := range tt.headers {
				req.Header.Set(header, value)
			}
			ip := config.XForwardedForIPExtractor(req)
			if ip != tt.expectedIP {
				t.Errorf("expected IP '%s', got '%s'", tt.expectedIP, ip)
			}
		})
	}
}

func TestXRealIPExtractor(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expectedIP string
	}{
		{"X-Real-IP present", "192.168.1.1:12345", map[string]string{"X-Real-IP": "203.0.113.1"}, "203.0.113.1"},
		{"X-Real-IP missing", "192.168.1.1:12345", map[string]string{"X-Forwarded-For": "203.0.113.2"}, "192.168.1.1"},
		{"empty X-Real-IP", "192.168.1.1:12345", map[string]string{"X-Real-IP": ""}, "192.168.1.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for header, value := range tt.headers {
				req.Header.Set(header, value)
			}
			ip := config.XRealIPExtractor(req)
			if ip != tt.expectedIP {
				t.Errorf("expected IP '%s', got '%s'", tt.expectedIP, ip)
			}
		})
	}
}

func TestRealIPWithDifferentExtractors(t *testing.T) {
	tests := []struct {
		name      string
		extractor config.IPExtractor
		headers   map[string]string
		expected  string
	}{
		{
			"RemoteAddrIPExtractor ignores headers",
			config.RemoteAddrIPExtractor,
			map[string]string{"X-Forwarded-For": "203.0.113.1", "X-Real-IP": "203.0.113.2"},
			"192.168.1.1:12345",
		},
		{
			"XForwardedForIPExtractor only uses X-Forwarded-For",
			config.XForwardedForIPExtractor,
			map[string]string{"X-Forwarded-For": "203.0.113.1", "X-Real-IP": "203.0.113.2"},
			"203.0.113.1:12345",
		},
		{
			"XRealIPExtractor only uses X-Real-IP",
			config.XRealIPExtractor,
			map[string]string{"X-Forwarded-For": "203.0.113.1", "X-Real-IP": "203.0.113.2"},
			"203.0.113.2:12345",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := RealIP(config.WithRealIPExtractor(tt.extractor))
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			for header, value := range tt.headers {
				req.Header.Set(header, value)
			}
			rr := httptest.NewRecorder()
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.RemoteAddr != tt.expected {
					t.Errorf("expected RemoteAddr '%s', got '%s'", tt.expected, r.RemoteAddr)
				}
				w.WriteHeader(http.StatusOK)
			})
			middleware(next).ServeHTTP(rr, req)
		})
	}
}

func TestRealIPEdgeCases(t *testing.T) {
	middleware := RealIP()
	t.Run("empty X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		req.Header.Set("X-Forwarded-For", "")
		rr := httptest.NewRecorder()
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.RemoteAddr != "192.168.1.1:12345" {
				t.Errorf("expected RemoteAddr '192.168.1.1:12345', got '%s'", r.RemoteAddr)
			}
			w.WriteHeader(http.StatusOK)
		})
		middleware(next).ServeHTTP(rr, req)
	})
	t.Run("malformed Forwarded header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		req.Header.Set("Forwarded", "invalid-format")
		rr := httptest.NewRecorder()
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.RemoteAddr != "192.168.1.1:12345" {
				t.Errorf("expected RemoteAddr '192.168.1.1:12345', got '%s'", r.RemoteAddr)
			}
			w.WriteHeader(http.StatusOK)
		})
		middleware(next).ServeHTTP(rr, req)
	})
	t.Run("X-Forwarded-For only commas", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		req.Header.Set("X-Forwarded-For", ",,,")
		rr := httptest.NewRecorder()
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Should handle empty gracefully
			w.WriteHeader(http.StatusOK)
		})
		middleware(next).ServeHTTP(rr, req)
	})
}
