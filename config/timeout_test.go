package config

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestTimeoutConfig_DefaultValues(t *testing.T) {
	cfg := DefaultTimeoutConfig
	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected default timeout = 30s, got %v", cfg.Timeout)
	}
	if cfg.StatusCode != http.StatusGatewayTimeout {
		t.Errorf("expected default status code = %d, got %d", http.StatusGatewayTimeout, cfg.StatusCode)
	}
	if cfg.Message != "" {
		t.Errorf("expected default message = '', got %s", cfg.Message)
	}
	if len(cfg.ExemptPaths) != 0 {
		t.Errorf("expected default exempt paths to be empty, got %d paths", len(cfg.ExemptPaths))
	}

	// Test default status code specifically
	expectedStatusCode := 504 // Gateway Timeout
	if cfg.StatusCode != expectedStatusCode {
		t.Errorf("expected default status code = %d (Gateway Timeout), got %d", expectedStatusCode, cfg.StatusCode)
	}
	if cfg.StatusCode != http.StatusGatewayTimeout {
		t.Errorf("expected status code to equal http.StatusGatewayTimeout (%d), got %d", http.StatusGatewayTimeout, cfg.StatusCode)
	}
}

func TestTimeoutOptions(t *testing.T) {
	t.Run("duration option", func(t *testing.T) {
		cfg := DefaultTimeoutConfig
		WithTimeoutDuration(60 * time.Second)(&cfg)
		if cfg.Timeout != 60*time.Second {
			t.Errorf("expected timeout = 60s, got %v", cfg.Timeout)
		}
	})

	t.Run("status code option", func(t *testing.T) {
		cfg := DefaultTimeoutConfig
		WithTimeoutStatusCode(http.StatusRequestTimeout)(&cfg)
		if cfg.StatusCode != http.StatusRequestTimeout {
			t.Errorf("expected status code = %d, got %d", http.StatusRequestTimeout, cfg.StatusCode)
		}
	})

	t.Run("message option", func(t *testing.T) {
		message := "Request timed out, please try again later"
		cfg := DefaultTimeoutConfig
		WithTimeoutMessage(message)(&cfg)
		if cfg.Message != message {
			t.Errorf("expected message = %s, got %s", message, cfg.Message)
		}
	})

	t.Run("exempt paths option", func(t *testing.T) {
		exemptPaths := []string{"/api/long-running", "/upload", "/stream", "/websocket"}
		cfg := DefaultTimeoutConfig
		WithTimeoutExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != 4 {
			t.Errorf("expected 4 exempt paths, got %d", len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})

	t.Run("multiple options", func(t *testing.T) {
		exemptPaths := []string{"/long-process", "/upload"}
		cfg := DefaultTimeoutConfig
		WithTimeoutDuration(2 * time.Minute)(&cfg)
		WithTimeoutStatusCode(http.StatusServiceUnavailable)(&cfg)
		WithTimeoutMessage("Service unavailable due to timeout")(&cfg)
		WithTimeoutExemptPaths(exemptPaths)(&cfg)

		if cfg.Timeout != 2*time.Minute {
			t.Errorf("expected timeout = 2m, got %v", cfg.Timeout)
		}
		if cfg.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("expected status code = %d, got %d", http.StatusServiceUnavailable, cfg.StatusCode)
		}
		if cfg.Message != "Service unavailable due to timeout" {
			t.Error("expected timeout message to be set correctly")
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Error("expected exempt paths to be set correctly")
		}
	})
}

func TestTimeoutConfig_DurationVariations(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
	}{
		{"1 second", time.Second},
		{"5 seconds", 5 * time.Second},
		{"1 minute", time.Minute},
		{"30 seconds", 30 * time.Second},
		{"2 minutes", 2 * time.Minute},
		{"10 minutes", 10 * time.Minute},
		{"1 hour", time.Hour},
		{"24 hours", 24 * time.Hour},
		{"zero duration", 0},
		{"negative duration", -time.Second},
		{"nanosecond", time.Nanosecond},
		{"microsecond", time.Microsecond},
		{"millisecond", time.Millisecond},
		{"max duration", time.Duration(1<<63 - 1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultTimeoutConfig
			WithTimeoutDuration(tt.duration)(&cfg)
			if cfg.Timeout != tt.duration {
				t.Errorf("expected timeout = %v, got %v", tt.duration, cfg.Timeout)
			}
		})
	}
}

func TestTimeoutConfig_StatusCodeVariations(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   int
	}{
		{"Request Timeout", http.StatusRequestTimeout, 408},
		{"Gateway Timeout", http.StatusGatewayTimeout, 504},
		{"Service Unavailable", http.StatusServiceUnavailable, 503},
		{"Internal Server Error", http.StatusInternalServerError, 500},
		{"Bad Gateway", http.StatusBadGateway, 502},
		{"Custom 599", 599, 599},
		{"Zero status code", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultTimeoutConfig
			WithTimeoutStatusCode(tt.statusCode)(&cfg)
			if cfg.StatusCode != tt.expected {
				t.Errorf("expected status code = %d, got %d", tt.expected, cfg.StatusCode)
			}
		})
	}
}

func TestTimeoutConfig_MessageVariations(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{"simple message", "Timeout occurred"},
		{"detailed message", "Request timeout after 30 seconds. Please try again."},
		{"json message", `{"error":"timeout","code":504}`},
		{"empty message", ""},
		{"multiline message", "Request timed out.\nPlease try again later."},
		{"unicode message", "请求超时，请稍后重试"},
		{"html message", "<p>Request timeout. Please try again later.</p>"},
		{"xml message", `<error><code>504</code><message>Gateway Timeout</message></error>`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultTimeoutConfig
			WithTimeoutMessage(tt.message)(&cfg)
			if cfg.Message != tt.message {
				t.Errorf("expected message = %q, got %q", tt.message, cfg.Message)
			}
		})
	}
}

func TestTimeoutConfig_EdgeCases(t *testing.T) {
	t.Run("empty exempt paths", func(t *testing.T) {
		cfg := DefaultTimeoutConfig
		WithTimeoutExemptPaths([]string{})(&cfg)
		if cfg.ExemptPaths == nil {
			t.Error("expected exempt paths slice to be initialized, not nil")
		}
		if len(cfg.ExemptPaths) != 0 {
			t.Errorf("expected empty exempt paths slice, got %d entries", len(cfg.ExemptPaths))
		}
	})

	t.Run("nil exempt paths", func(t *testing.T) {
		cfg := DefaultTimeoutConfig
		WithTimeoutExemptPaths(nil)(&cfg)
		if cfg.ExemptPaths != nil {
			t.Error("expected exempt paths to remain nil when nil is passed")
		}
	})

	t.Run("empty string paths", func(t *testing.T) {
		exemptPaths := []string{"", "/upload", ""}
		cfg := DefaultTimeoutConfig
		WithTimeoutExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != 3 {
			t.Errorf("expected 3 exempt paths, got %d", len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})
}

func TestTimeoutConfig_PathPatterns(t *testing.T) {
	t.Run("pattern paths", func(t *testing.T) {
		exemptPaths := []string{
			"/api/v1/upload/*", "/streaming/*", "/websocket", "/long-poll",
			"*.upload", "/admin/backup/*", "/reports/generate", "/sse/*",
		}
		cfg := DefaultTimeoutConfig
		WithTimeoutExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != len(exemptPaths) {
			t.Errorf("expected %d exempt paths, got %d", len(exemptPaths), len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})

	t.Run("special character paths", func(t *testing.T) {
		exemptPaths := []string{
			"/api-v1/upload", "/long_running_task", "/upload-service", "/stream.data",
			"/process (background)", "/path with spaces", "/path/with/unicode-ñ", "/files/test@example.com",
		}
		cfg := DefaultTimeoutConfig
		WithTimeoutExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != len(exemptPaths) {
			t.Errorf("expected %d exempt paths, got %d", len(exemptPaths), len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})
}
