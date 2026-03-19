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
	if len(cfg.ExcludedPaths) != 0 {
		t.Errorf("expected default excluded paths to be empty, got %d paths", len(cfg.ExcludedPaths))
	}
	if len(cfg.IncludedPaths) != 0 {
		t.Errorf("expected default included paths to be empty, got %d paths", len(cfg.IncludedPaths))
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

func TestTimeoutConfig_StructAssignment(t *testing.T) {
	t.Run("duration assignment", func(t *testing.T) {
		cfg := TimeoutConfig{
			Timeout: 60 * time.Second,
		}
		if cfg.Timeout != 60*time.Second {
			t.Errorf("expected timeout = 60s, got %v", cfg.Timeout)
		}
	})

	t.Run("status code assignment", func(t *testing.T) {
		cfg := TimeoutConfig{
			Timeout:    DefaultTimeoutConfig.Timeout,
			StatusCode: http.StatusRequestTimeout,
		}
		if cfg.StatusCode != http.StatusRequestTimeout {
			t.Errorf("expected status code = %d, got %d", http.StatusRequestTimeout, cfg.StatusCode)
		}
	})

	t.Run("message assignment", func(t *testing.T) {
		message := "Request timed out, please try again later"
		cfg := TimeoutConfig{
			Timeout: DefaultTimeoutConfig.Timeout,
			Message: message,
		}
		if cfg.Message != message {
			t.Errorf("expected message = %s, got %s", message, cfg.Message)
		}
	})

	t.Run("excluded paths assignment", func(t *testing.T) {
		excludedPaths := []string{"/api/long-running", "/upload", "/stream", "/websocket"}
		cfg := TimeoutConfig{
			Timeout:       DefaultTimeoutConfig.Timeout,
			ExcludedPaths: excludedPaths,
		}
		if len(cfg.ExcludedPaths) != 4 {
			t.Errorf("expected 4 excluded paths, got %d", len(cfg.ExcludedPaths))
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
			t.Errorf("expected excluded paths = %v, got %v", excludedPaths, cfg.ExcludedPaths)
		}
	})

	t.Run("included paths assignment", func(t *testing.T) {
		includedPaths := []string{"/api/public", "/health"}
		cfg := TimeoutConfig{
			Timeout:       DefaultTimeoutConfig.Timeout,
			IncludedPaths: includedPaths,
		}
		if len(cfg.IncludedPaths) != 2 {
			t.Errorf("expected 2 included paths, got %d", len(cfg.IncludedPaths))
		}
		if !reflect.DeepEqual(cfg.IncludedPaths, includedPaths) {
			t.Errorf("expected included paths = %v, got %v", includedPaths, cfg.IncludedPaths)
		}
	})

	t.Run("multiple fields", func(t *testing.T) {
		excludedPaths := []string{"/long-process", "/upload"}
		includedPaths := []string{"/api/public"}
		cfg := TimeoutConfig{
			Timeout:       2 * time.Minute,
			StatusCode:    http.StatusServiceUnavailable,
			Message:       "Service unavailable due to timeout",
			ExcludedPaths: excludedPaths,
			IncludedPaths: includedPaths,
		}

		if cfg.Timeout != 2*time.Minute {
			t.Errorf("expected timeout = 2m, got %v", cfg.Timeout)
		}
		if cfg.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("expected status code = %d, got %d", http.StatusServiceUnavailable, cfg.StatusCode)
		}
		if cfg.Message != "Service unavailable due to timeout" {
			t.Error("expected timeout message to be set correctly")
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
			t.Error("expected excluded paths to be set correctly")
		}
		if len(cfg.IncludedPaths) != 1 {
			t.Errorf("expected 1 allowed path, got %d", len(cfg.IncludedPaths))
		}
		if !reflect.DeepEqual(cfg.IncludedPaths, includedPaths) {
			t.Error("expected included paths to be set correctly")
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
			cfg := TimeoutConfig{
				Timeout: tt.duration,
			}
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
			cfg := TimeoutConfig{
				Timeout:    DefaultTimeoutConfig.Timeout,
				StatusCode: tt.statusCode,
			}
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
			cfg := TimeoutConfig{
				Timeout: DefaultTimeoutConfig.Timeout,
				Message: tt.message,
			}
			if cfg.Message != tt.message {
				t.Errorf("expected message = %q, got %q", tt.message, cfg.Message)
			}
		})
	}
}

func TestTimeoutConfig_EdgeCases(t *testing.T) {
	t.Run("empty excluded paths", func(t *testing.T) {
		cfg := TimeoutConfig{
			Timeout:       DefaultTimeoutConfig.Timeout,
			ExcludedPaths: []string{},
		}
		if cfg.ExcludedPaths == nil {
			t.Error("expected excluded paths slice to be initialized, not nil")
		}
		if len(cfg.ExcludedPaths) != 0 {
			t.Errorf("expected empty excluded paths slice, got %d entries", len(cfg.ExcludedPaths))
		}
	})

	t.Run("nil excluded paths", func(t *testing.T) {
		cfg := TimeoutConfig{
			Timeout:       DefaultTimeoutConfig.Timeout,
			ExcludedPaths: nil,
		}
		if cfg.ExcludedPaths != nil {
			t.Error("expected excluded paths to remain nil when nil is passed")
		}
	})

	t.Run("empty string paths", func(t *testing.T) {
		excludedPaths := []string{"", "/upload", ""}
		cfg := TimeoutConfig{
			Timeout:       DefaultTimeoutConfig.Timeout,
			ExcludedPaths: excludedPaths,
		}
		if len(cfg.ExcludedPaths) != 3 {
			t.Errorf("expected 3 excluded paths, got %d", len(cfg.ExcludedPaths))
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
			t.Errorf("expected excluded paths = %v, got %v", excludedPaths, cfg.ExcludedPaths)
		}
	})

	t.Run("empty included paths", func(t *testing.T) {
		cfg := TimeoutConfig{
			Timeout:       DefaultTimeoutConfig.Timeout,
			IncludedPaths: []string{},
		}
		if cfg.IncludedPaths == nil {
			t.Error("expected included paths slice to be initialized, not nil")
		}
		if len(cfg.IncludedPaths) != 0 {
			t.Errorf("expected empty included paths slice, got %d entries", len(cfg.IncludedPaths))
		}
	})

	t.Run("nil included paths", func(t *testing.T) {
		cfg := TimeoutConfig{
			Timeout:       DefaultTimeoutConfig.Timeout,
			IncludedPaths: nil,
		}
		if cfg.IncludedPaths != nil {
			t.Error("expected included paths to remain nil when nil is passed")
		}
	})

	t.Run("custom included paths", func(t *testing.T) {
		includedPaths := []string{"/api/public", "/health"}
		cfg := TimeoutConfig{
			Timeout:       DefaultTimeoutConfig.Timeout,
			IncludedPaths: includedPaths,
		}
		if len(cfg.IncludedPaths) != 2 {
			t.Errorf("expected 2 included paths, got %d", len(cfg.IncludedPaths))
		}
		if !reflect.DeepEqual(cfg.IncludedPaths, includedPaths) {
			t.Errorf("expected included paths = %v, got %v", includedPaths, cfg.IncludedPaths)
		}
	})
}

func TestTimeoutConfig_PathPatterns(t *testing.T) {
	t.Run("pattern paths", func(t *testing.T) {
		excludedPaths := []string{
			"/api/v1/upload/*", "/streaming/*", "/websocket", "/long-poll",
			"*.upload", "/admin/backup/*", "/reports/generate", "/sse/*",
		}
		cfg := TimeoutConfig{
			Timeout:       DefaultTimeoutConfig.Timeout,
			ExcludedPaths: excludedPaths,
		}
		if len(cfg.ExcludedPaths) != len(excludedPaths) {
			t.Errorf("expected %d excluded paths, got %d", len(excludedPaths), len(cfg.ExcludedPaths))
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
			t.Errorf("expected excluded paths = %v, got %v", excludedPaths, cfg.ExcludedPaths)
		}
	})

	t.Run("special character paths", func(t *testing.T) {
		excludedPaths := []string{
			"/api-v1/upload", "/long_running_task", "/upload-service", "/stream.data",
			"/process (background)", "/path with spaces", "/path/with/unicode-ñ", "/files/test@example.com",
		}
		cfg := TimeoutConfig{
			Timeout:       DefaultTimeoutConfig.Timeout,
			ExcludedPaths: excludedPaths,
		}
		if len(cfg.ExcludedPaths) != len(excludedPaths) {
			t.Errorf("expected %d excluded paths, got %d", len(excludedPaths), len(cfg.ExcludedPaths))
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
			t.Errorf("expected excluded paths = %v, got %v", excludedPaths, cfg.ExcludedPaths)
		}
	})
}
