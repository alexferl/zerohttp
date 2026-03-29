package timeout

import (
	"net/http"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestTimeoutConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertEqual(t, 30*time.Second, cfg.Duration)
	zhtest.AssertEqual(t, http.StatusGatewayTimeout, cfg.StatusCode)
	zhtest.AssertEqual(t, "", cfg.Message)
	zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))

	// Test default status code specifically
	expectedStatusCode := 504 // Gateway Timeout
	zhtest.AssertEqual(t, expectedStatusCode, cfg.StatusCode)
	zhtest.AssertEqual(t, http.StatusGatewayTimeout, cfg.StatusCode)
}

func TestTimeoutConfig_StructAssignment(t *testing.T) {
	t.Run("duration assignment", func(t *testing.T) {
		cfg := Config{
			Duration: 60 * time.Second,
		}
		zhtest.AssertEqual(t, 60*time.Second, cfg.Duration)
	})

	t.Run("status code assignment", func(t *testing.T) {
		cfg := Config{
			Duration:   DefaultConfig.Duration,
			StatusCode: http.StatusRequestTimeout,
		}
		zhtest.AssertEqual(t, http.StatusRequestTimeout, cfg.StatusCode)
	})

	t.Run("message assignment", func(t *testing.T) {
		message := "Request timed out, please try again later"
		cfg := Config{
			Duration: DefaultConfig.Duration,
			Message:  message,
		}
		zhtest.AssertEqual(t, message, cfg.Message)
	})

	t.Run("excluded paths assignment", func(t *testing.T) {
		excludedPaths := []string{"/api/long-running", "/upload", "/stream", "/websocket"}
		cfg := Config{
			Duration:      DefaultConfig.Duration,
			ExcludedPaths: excludedPaths,
		}
		zhtest.AssertEqual(t, 4, len(cfg.ExcludedPaths))
		zhtest.AssertDeepEqual(t, excludedPaths, cfg.ExcludedPaths)
	})

	t.Run("included paths assignment", func(t *testing.T) {
		includedPaths := []string{"/api/public", "/health"}
		cfg := Config{
			Duration:      DefaultConfig.Duration,
			IncludedPaths: includedPaths,
		}
		zhtest.AssertEqual(t, 2, len(cfg.IncludedPaths))
		zhtest.AssertDeepEqual(t, includedPaths, cfg.IncludedPaths)
	})

	t.Run("multiple fields", func(t *testing.T) {
		excludedPaths := []string{"/long-process", "/upload"}
		includedPaths := []string{"/api/public"}
		cfg := Config{
			Duration:      2 * time.Minute,
			StatusCode:    http.StatusServiceUnavailable,
			Message:       "Service unavailable due to timeout",
			ExcludedPaths: excludedPaths,
			IncludedPaths: includedPaths,
		}

		zhtest.AssertEqual(t, 2*time.Minute, cfg.Duration)
		zhtest.AssertEqual(t, http.StatusServiceUnavailable, cfg.StatusCode)
		zhtest.AssertEqual(t, "Service unavailable due to timeout", cfg.Message)
		zhtest.AssertDeepEqual(t, excludedPaths, cfg.ExcludedPaths)
		zhtest.AssertEqual(t, 1, len(cfg.IncludedPaths))
		zhtest.AssertDeepEqual(t, includedPaths, cfg.IncludedPaths)
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
			cfg := Config{
				Duration: tt.duration,
			}
			zhtest.AssertEqual(t, tt.duration, cfg.Duration)
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
			cfg := Config{
				Duration:   DefaultConfig.Duration,
				StatusCode: tt.statusCode,
			}
			zhtest.AssertEqual(t, tt.expected, cfg.StatusCode)
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
			cfg := Config{
				Duration: DefaultConfig.Duration,
				Message:  tt.message,
			}
			zhtest.AssertEqual(t, tt.message, cfg.Message)
		})
	}
}

func TestTimeoutConfig_EdgeCases(t *testing.T) {
	t.Run("empty excluded paths", func(t *testing.T) {
		cfg := Config{
			Duration:      DefaultConfig.Duration,
			ExcludedPaths: []string{},
		}
		zhtest.AssertNotNil(t, cfg.ExcludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
	})

	t.Run("nil excluded paths", func(t *testing.T) {
		cfg := Config{
			Duration:      DefaultConfig.Duration,
			ExcludedPaths: nil,
		}
		zhtest.AssertNil(t, cfg.ExcludedPaths)
	})

	t.Run("empty string paths", func(t *testing.T) {
		excludedPaths := []string{"", "/upload", ""}
		cfg := Config{
			Duration:      DefaultConfig.Duration,
			ExcludedPaths: excludedPaths,
		}
		zhtest.AssertEqual(t, 3, len(cfg.ExcludedPaths))
		zhtest.AssertDeepEqual(t, excludedPaths, cfg.ExcludedPaths)
	})

	t.Run("empty included paths", func(t *testing.T) {
		cfg := Config{
			Duration:      DefaultConfig.Duration,
			IncludedPaths: []string{},
		}
		zhtest.AssertNotNil(t, cfg.IncludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
	})

	t.Run("nil included paths", func(t *testing.T) {
		cfg := Config{
			Duration:      DefaultConfig.Duration,
			IncludedPaths: nil,
		}
		zhtest.AssertNil(t, cfg.IncludedPaths)
	})

	t.Run("custom included paths", func(t *testing.T) {
		includedPaths := []string{"/api/public", "/health"}
		cfg := Config{
			Duration:      DefaultConfig.Duration,
			IncludedPaths: includedPaths,
		}
		zhtest.AssertEqual(t, 2, len(cfg.IncludedPaths))
		zhtest.AssertDeepEqual(t, includedPaths, cfg.IncludedPaths)
	})
}

func TestTimeoutConfig_PathPatterns(t *testing.T) {
	t.Run("pattern paths", func(t *testing.T) {
		excludedPaths := []string{
			"/api/v1/upload/*", "/streaming/*", "/websocket", "/long-poll",
			"*.upload", "/admin/backup/*", "/reports/generate", "/sse/*",
		}
		cfg := Config{
			Duration:      DefaultConfig.Duration,
			ExcludedPaths: excludedPaths,
		}
		zhtest.AssertEqual(t, len(excludedPaths), len(cfg.ExcludedPaths))
		zhtest.AssertDeepEqual(t, excludedPaths, cfg.ExcludedPaths)
	})

	t.Run("special character paths", func(t *testing.T) {
		excludedPaths := []string{
			"/api-v1/upload", "/long_running_task", "/upload-service", "/stream.data",
			"/process (background)", "/path with spaces", "/path/with/unicode-\xc3\xb1", "/files/test@example.com",
		}
		cfg := Config{
			Duration:      DefaultConfig.Duration,
			ExcludedPaths: excludedPaths,
		}
		zhtest.AssertEqual(t, len(excludedPaths), len(cfg.ExcludedPaths))
		zhtest.AssertDeepEqual(t, excludedPaths, cfg.ExcludedPaths)
	})
}
