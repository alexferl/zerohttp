package realip

import (
	"net/http"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestRealIPConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertNotNil(t, cfg.IPExtractor)
}

func TestDefaultIPExtractor(t *testing.T) {
	t.Run("X-Forwarded-For header", func(t *testing.T) {
		tests := []struct {
			name       string
			xffHeader  string
			remoteAddr string
			expectedIP string
		}{
			{"single IP", "203.0.113.1", "192.168.1.1:8080", "203.0.113.1"},
			{"multiple IPs", "203.0.113.1, 198.51.100.1, 192.168.1.1", "192.168.1.1:8080", "203.0.113.1"},
			{"with spaces", " 203.0.113.1 , 198.51.100.1", "192.168.1.1:8080", "203.0.113.1"},
			{"empty falls back", "", "192.168.1.1:8080", "192.168.1.1"},
			{"whitespace only", " ", "192.168.1.1:8080", ""},
			{"only commas", ",,,", "192.168.1.1:8080", ""},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req, _ := http.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = tt.remoteAddr
				if tt.xffHeader != "" {
					req.Header.Set(httpx.HeaderXForwardedFor, tt.xffHeader)
				}
				result := DefaultIPExtractor(req)
				zhtest.AssertEqual(t, tt.expectedIP, result)
			})
		}
	})

	t.Run("X-Real-IP header", func(t *testing.T) {
		tests := []struct {
			name       string
			xRealIP    string
			remoteAddr string
			expectedIP string
		}{
			{"present", "203.0.113.2", "192.168.1.1:8080", "203.0.113.2"},
			{"empty falls back", "", "192.168.1.1:8080", "192.168.1.1"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req, _ := http.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = tt.remoteAddr
				if tt.xRealIP != "" {
					req.Header.Set(httpx.HeaderXRealIP, tt.xRealIP)
				}
				result := DefaultIPExtractor(req)
				zhtest.AssertEqual(t, tt.expectedIP, result)
			})
		}
	})

	t.Run("X-Forwarded header", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:8080"
		req.Header.Set(httpx.HeaderXForwarded, "203.0.113.3")
		result := DefaultIPExtractor(req)
		zhtest.AssertEqual(t, "203.0.113.3", result)
	})

	t.Run("Forwarded header", func(t *testing.T) {
		tests := []struct {
			name            string
			forwardedHeader string
			remoteAddr      string
			expectedIP      string
		}{
			{"with for= parameter", "for=203.0.113.4", "192.168.1.1:8080", "203.0.113.4"},
			{"quoted for= parameter", `for="203.0.113.5"`, "192.168.1.1:8080", "203.0.113.5"},
			{"multiple parameters", "proto=https;for=203.0.113.6;by=192.168.1.1", "192.168.1.1:8080", "203.0.113.6"},
			{"spaces around for=", "proto=https; for=203.0.113.7 ; by=192.168.1.1", "192.168.1.1:8080", "203.0.113.7"},
			{"without for= parameter", "proto=https;by=192.168.1.1", "192.168.1.1:8080", "192.168.1.1"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req, _ := http.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = tt.remoteAddr
				req.Header.Set(httpx.HeaderForwarded, tt.forwardedHeader)
				result := DefaultIPExtractor(req)
				zhtest.AssertEqual(t, tt.expectedIP, result)
			})
		}
	})

	t.Run("header priority", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:8080"
		req.Header.Set(httpx.HeaderXForwardedFor, "203.0.113.1")
		req.Header.Set(httpx.HeaderXRealIP, "203.0.113.2")
		req.Header.Set(httpx.HeaderXForwarded, "203.0.113.3")
		req.Header.Set(httpx.HeaderForwarded, "for=203.0.113.4")
		result := DefaultIPExtractor(req)
		zhtest.AssertEqual(t, "203.0.113.1", result)
	})

	t.Run("RemoteAddr fallback", func(t *testing.T) {
		tests := []struct {
			name       string
			remoteAddr string
			expectedIP string
		}{
			{"IPv4 with port", "192.168.1.100:8080", "192.168.1.100"},
			{"IPv6 with port", "[2001:db8::1]:8080", "2001:db8::1"},
			{"IPv4 without port", "192.168.1.100", "192.168.1.100"},
			{"malformed address", "malformed:address:with:colons", "malformed:address:with:colons"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req, _ := http.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = tt.remoteAddr
				result := DefaultIPExtractor(req)
				zhtest.AssertEqual(t, tt.expectedIP, result)
			})
		}
	})

	t.Run("IPv6 support", func(t *testing.T) {
		tests := []struct {
			name       string
			remoteAddr string
			xffHeader  string
			expectedIP string
		}{
			{"IPv6 in RemoteAddr", "[2001:db8::1]:8080", "", "2001:db8::1"},
			{"IPv6 in X-Forwarded-For", "192.168.1.1:8080", "2001:db8::2", "2001:db8::2"},
			{"IPv6 with brackets", "192.168.1.1:8080", "[2001:db8::3]", "[2001:db8::3]"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req, _ := http.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = tt.remoteAddr
				if tt.xffHeader != "" {
					req.Header.Set(httpx.HeaderXForwardedFor, tt.xffHeader)
				}
				result := DefaultIPExtractor(req)
				zhtest.AssertEqual(t, tt.expectedIP, result)
			})
		}
	})
}

func TestSpecializedIPExtractors(t *testing.T) {
	t.Run("RemoteAddrIPExtractor", func(t *testing.T) {
		tests := []struct {
			name       string
			remoteAddr string
			expectedIP string
		}{
			{"IPv4 with port", "203.0.113.10:9000", "203.0.113.10"},
			{"IPv6 with port", "[2001:db8::10]:9000", "2001:db8::10"},
			{"IPv4 without port", "203.0.113.10", "203.0.113.10"},
			{"malformed address", "invalid::address", "invalid::address"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req, _ := http.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = tt.remoteAddr
				req.Header.Set(httpx.HeaderXForwardedFor, "should-be-ignored")
				req.Header.Set(httpx.HeaderXRealIP, "should-be-ignored")
				result := RemoteAddrIPExtractor(req)
				zhtest.AssertEqual(t, tt.expectedIP, result)
			})
		}
	})

	t.Run("XForwardedForIPExtractor", func(t *testing.T) {
		tests := []struct {
			name       string
			xffHeader  string
			remoteAddr string
			expectedIP string
		}{
			{"present", "203.0.113.20", "192.168.1.1:8080", "203.0.113.20"},
			{"multiple IPs", "203.0.113.20, 198.51.100.20", "192.168.1.1:8080", "203.0.113.20"},
			{"fallback to RemoteAddr", "", "192.168.1.1:8080", "192.168.1.1"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req, _ := http.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = tt.remoteAddr
				if tt.xffHeader != "" {
					req.Header.Set(httpx.HeaderXForwardedFor, tt.xffHeader)
				}
				req.Header.Set(httpx.HeaderXRealIP, "should-be-ignored")
				result := XForwardedForIPExtractor(req)
				zhtest.AssertEqual(t, tt.expectedIP, result)
			})
		}
	})

	t.Run("XRealIPExtractor", func(t *testing.T) {
		tests := []struct {
			name       string
			xRealIP    string
			remoteAddr string
			expectedIP string
		}{
			{"present", "203.0.113.30", "192.168.1.1:8080", "203.0.113.30"},
			{"fallback to RemoteAddr", "", "192.168.1.1:8080", "192.168.1.1"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req, _ := http.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = tt.remoteAddr
				if tt.xRealIP != "" {
					req.Header.Set(httpx.HeaderXRealIP, tt.xRealIP)
				}
				req.Header.Set(httpx.HeaderXForwardedFor, "should-be-ignored")
				result := XRealIPExtractor(req)
				zhtest.AssertEqual(t, tt.expectedIP, result)
			})
		}
	})
}

func TestRealIPConfig_CustomExtractors(t *testing.T) {
	t.Run("custom extractor", func(t *testing.T) {
		customExtractor := func(r *http.Request) string {
			return "custom-ip"
		}
		cfg := Config{
			IPExtractor: customExtractor,
		}
		zhtest.AssertNotNil(t, cfg.IPExtractor)
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		result := cfg.IPExtractor(req)
		zhtest.AssertEqual(t, "custom-ip", result)
	})

	t.Run("nil extractor", func(t *testing.T) {
		cfg := Config{
			IPExtractor: nil,
		}
		zhtest.AssertNil(t, cfg.IPExtractor)
	})

	t.Run("various custom extractors", func(t *testing.T) {
		tests := []struct {
			name         string
			extractor    IPExtractor
			setupRequest func(*http.Request)
			expectedIP   string
		}{
			{
				name: "header-based extractor",
				extractor: func(r *http.Request) string {
					return r.Header.Get("X-Custom-IP")
				},
				setupRequest: func(r *http.Request) {
					r.Header.Set("X-Custom-IP", "203.0.113.100")
				},
				expectedIP: "203.0.113.100",
			},
			{
				name: "query parameter extractor",
				extractor: func(r *http.Request) string {
					return r.URL.Query().Get("client_ip")
				},
				setupRequest: func(r *http.Request) {
					r.URL.RawQuery = "client_ip=203.0.113.101"
				},
				expectedIP: "203.0.113.101",
			},
			{
				name: "combined extractor with fallback",
				extractor: func(r *http.Request) string {
					if customIP := r.Header.Get("X-Custom-IP"); customIP != "" {
						return customIP
					}
					return DefaultIPExtractor(r)
				},
				setupRequest: func(r *http.Request) {
					r.RemoteAddr = "192.168.1.1:8080"
					r.Header.Set("X-Forwarded-For", "203.0.113.102")
				},
				expectedIP: "203.0.113.102",
			},
			{
				name: "constant IP extractor",
				extractor: func(r *http.Request) string {
					return "127.0.0.1"
				},
				setupRequest: func(r *http.Request) {},
				expectedIP:   "127.0.0.1",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := Config{
					IPExtractor: tt.extractor,
				}
				req, _ := http.NewRequest(http.MethodGet, "/", nil)
				tt.setupRequest(req)
				result := cfg.IPExtractor(req)
				zhtest.AssertEqual(t, tt.expectedIP, result)
			})
		}
	})

	t.Run("default extractor through config", func(t *testing.T) {
		cfg := DefaultConfig
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:8080"
		req.Header.Set(httpx.HeaderXForwardedFor, "203.0.113.200")
		result := cfg.IPExtractor(req)
		zhtest.AssertEqual(t, "203.0.113.200", result)
	})
}

func TestRealIPConfig_ExtractorComparison(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:8080"
	req.Header.Set(httpx.HeaderXForwardedFor, "203.0.113.1")
	req.Header.Set(httpx.HeaderXRealIP, "203.0.113.2")

	tests := []struct {
		name      string
		extractor IPExtractor
		expected  string
	}{
		{"DefaultIPExtractor", DefaultIPExtractor, "203.0.113.1"},
		{"RemoteAddrIPExtractor", RemoteAddrIPExtractor, "192.168.1.1"},
		{"XForwardedForIPExtractor", XForwardedForIPExtractor, "203.0.113.1"},
		{"XRealIPExtractor", XRealIPExtractor, "203.0.113.2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.extractor(req)
			zhtest.AssertEqual(t, tt.expected, result)
		})
	}
}
