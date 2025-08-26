package config

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/log"
	"golang.org/x/crypto/acme/autocert"
)

func TestDefaultConfigValues(t *testing.T) {
	cfg := DefaultConfig
	if cfg.Addr != "localhost:8080" {
		t.Errorf("expected Addr = localhost:8080, got %s", cfg.Addr)
	}
	if cfg.TLSAddr != "localhost:8443" {
		t.Errorf("expected TLSAddr = localhost:8443, got %s", cfg.TLSAddr)
	}
	if cfg.DisableDefaultMiddlewares {
		t.Error("expected DisableDefaultMiddlewares to be false")
	}
	if cfg.DefaultMiddlewares != nil {
		t.Error("expected DefaultMiddlewares to be nil")
	}
	if cfg.Logger != nil {
		t.Error("expected Logger to be nil")
	}
	if cfg.CertFile != "" {
		t.Errorf("expected CertFile to be empty, got %s", cfg.CertFile)
	}
	if cfg.KeyFile != "" {
		t.Errorf("expected KeyFile to be empty, got %s", cfg.KeyFile)
	}
	if cfg.AutocertManager != nil {
		t.Error("expected AutocertManager to be nil")
	}
	if cfg.Server != nil {
		t.Error("expected Server to be nil in default config")
	}
	if cfg.TLSServer != nil {
		t.Error("expected TLSServer to be nil in default config")
	}
	if cfg.Listener != nil {
		t.Error("expected Listener to be nil")
	}
	if cfg.TLSListener != nil {
		t.Error("expected TLSListener to be nil")
	}

	// Test middleware option slices are initialized
	if len(cfg.RecoverOptions) == 0 {
		t.Error("expected RecoverOptions to be initialized with defaults")
	}
	if len(cfg.RequestBodySizeOptions) == 0 {
		t.Error("expected RequestBodySizeOptions to be initialized with defaults")
	}
	if len(cfg.RequestIDOptions) == 0 {
		t.Error("expected RequestIDOptions to be initialized with defaults")
	}
	if len(cfg.RequestLoggerOptions) == 0 {
		t.Error("expected RequestLoggerOptions to be initialized with defaults")
	}
	if len(cfg.SecurityHeadersOptions) == 0 {
		t.Error("expected SecurityHeadersOptions to be initialized with defaults")
	}
}

func TestConfigBuild(t *testing.T) {
	cfg := Config{
		RecoverOptions: []RecoverOption{
			WithRecoverStackSize(8192),
			WithRecoverEnableStackTrace(false),
		},
		RequestBodySizeOptions: []RequestBodySizeOption{
			WithRequestBodySizeMaxBytes(2048),
		},
		RequestIDOptions: []RequestIDOption{
			WithRequestIDHeader("X-Custom-ID"),
		},
		SecurityHeadersOptions: []SecurityHeadersOption{
			WithSecurityHeadersXFrameOptions("SAMEORIGIN"),
		},
	}

	cfg.Build()

	if cfg.Recover.StackSize != 8192 {
		t.Errorf("expected Recover.StackSize = 8192, got %d", cfg.Recover.StackSize)
	}
	if cfg.Recover.EnableStackTrace != false {
		t.Error("expected Recover.EnableStackTrace to be false")
	}
	if cfg.RequestBodySize.MaxBytes != 2048 {
		t.Errorf("expected RequestBodySize.MaxBytes = 2048, got %d", cfg.RequestBodySize.MaxBytes)
	}
	if cfg.RequestID.Header != "X-Custom-ID" {
		t.Errorf("expected RequestID.Header = X-Custom-ID, got %s", cfg.RequestID.Header)
	}
	if cfg.SecurityHeaders.XFrameOptions != "SAMEORIGIN" {
		t.Errorf("expected SecurityHeaders.XFrameOptions = SAMEORIGIN, got %s", cfg.SecurityHeaders.XFrameOptions)
	}
}

func TestConfigOptions(t *testing.T) {
	t.Run("basic options", func(t *testing.T) {
		cfg := DefaultConfig
		WithAddr(":3000")(&cfg)
		if cfg.Addr != ":3000" {
			t.Errorf("expected Addr = :3000, got %s", cfg.Addr)
		}
		WithTLSAddr(":3443")(&cfg)
		if cfg.TLSAddr != ":3443" {
			t.Errorf("expected TLSAddr = :3443, got %s", cfg.TLSAddr)
		}
		WithDisableDefaultMiddlewares()(&cfg)
		if !cfg.DisableDefaultMiddlewares {
			t.Error("expected DisableDefaultMiddlewares to be true")
		}
		WithCertFile("cert.pem")(&cfg)
		if cfg.CertFile != "cert.pem" {
			t.Errorf("expected CertFile = cert.pem, got %s", cfg.CertFile)
		}
		WithKeyFile("key.pem")(&cfg)
		if cfg.KeyFile != "key.pem" {
			t.Errorf("expected KeyFile = key.pem, got %s", cfg.KeyFile)
		}
	})

	t.Run("custom middlewares", func(t *testing.T) {
		customMiddlewares := []func(http.Handler) http.Handler{
			func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					next.ServeHTTP(w, r)
				})
			},
		}
		cfg := DefaultConfig
		WithDefaultMiddlewares(customMiddlewares)(&cfg)
		if len(cfg.DefaultMiddlewares) != 1 {
			t.Errorf("expected 1 custom middleware, got %d", len(cfg.DefaultMiddlewares))
		}
		h := cfg.DefaultMiddlewares[0](http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		req, _ := http.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
	})

	t.Run("server options", func(t *testing.T) {
		customServer := &http.Server{Addr: ":9000"}
		cfg := DefaultConfig
		WithServer(customServer)(&cfg)
		if cfg.Server != customServer {
			t.Error("expected custom Server to be set")
		}
		if cfg.Server.Addr != ":9000" {
			t.Errorf("expected Server.Addr = :9000, got %s", cfg.Server.Addr)
		}

		customTLSServer := &http.Server{Addr: ":9443"}
		WithTLSServer(customTLSServer)(&cfg)
		if cfg.TLSServer != customTLSServer {
			t.Error("expected custom TLSServer to be set")
		}
		if cfg.TLSServer.Addr != ":9443" {
			t.Errorf("expected TLSServer.Addr = :9443, got %s", cfg.TLSServer.Addr)
		}
	})

	t.Run("listener options", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to create test listener: %v", err)
		}
		t.Cleanup(func() {
			if err := listener.Close(); err != nil {
				t.Logf("failed to close listener: %v", err)
			}
		})

		cfg := DefaultConfig
		WithListener(listener)(&cfg)
		if cfg.Listener != listener {
			t.Error("expected custom Listener to be set")
		}
		if cfg.Listener.Addr() == nil {
			t.Error("expected listener to have a valid address")
		}

		tlsListener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to create test TLS listener: %v", err)
		}
		t.Cleanup(func() {
			if err := tlsListener.Close(); err != nil {
				t.Logf("failed to close TLS listener: %v", err)
			}
		})

		WithTLSListener(tlsListener)(&cfg)
		if cfg.TLSListener != tlsListener {
			t.Error("expected custom TLSListener to be set")
		}
		if cfg.TLSListener.Addr() == nil {
			t.Error("expected TLS listener to have a valid address")
		}
	})

	t.Run("autocert manager", func(t *testing.T) {
		manager := &autocert.Manager{
			Cache:      autocert.DirCache("/tmp/test-certs"),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist("example.com"),
		}
		cfg := DefaultConfig
		WithAutocertManager(manager)(&cfg)
		if cfg.AutocertManager != manager {
			t.Error("expected custom AutocertManager to be set")
		}
		if cfg.AutocertManager.Cache == nil {
			t.Error("expected autocert manager to have cache configured")
		}
		if cfg.AutocertManager.Prompt == nil {
			t.Error("expected autocert manager to have prompt configured")
		}
		if cfg.AutocertManager.HostPolicy == nil {
			t.Error("expected autocert manager to have host policy configured")
		}
	})

	t.Run("logger option", func(t *testing.T) {
		mockLogger := log.NewDefaultLogger()
		cfg := DefaultConfig
		WithLogger(mockLogger)(&cfg)
		if cfg.Logger != mockLogger {
			t.Error("expected custom Logger to be set")
		}
	})
}

func TestConfigMiddlewareOptions(t *testing.T) {
	t.Run("recover options", func(t *testing.T) {
		cfg := DefaultConfig
		WithRecoverOptions(
			WithRecoverStackSize(16384),
			WithRecoverEnableStackTrace(false),
		)(&cfg)
		if len(cfg.RecoverOptions) != 2 {
			t.Errorf("expected 2 recover options, got %d", len(cfg.RecoverOptions))
		}
		cfg.Build()
		if cfg.Recover.StackSize != 16384 {
			t.Errorf("expected StackSize = 16384, got %d", cfg.Recover.StackSize)
		}
		if cfg.Recover.EnableStackTrace != false {
			t.Error("expected EnableStackTrace to be false")
		}
	})

	t.Run("request body size options", func(t *testing.T) {
		cfg := DefaultConfig
		WithRequestBodySizeOptions(WithRequestBodySizeMaxBytes(4096))(&cfg)
		if len(cfg.RequestBodySizeOptions) != 1 {
			t.Errorf("expected 1 request body size option, got %d", len(cfg.RequestBodySizeOptions))
		}
		cfg.Build()
		if cfg.RequestBodySize.MaxBytes != 4096 {
			t.Errorf("expected MaxBytes = 4096, got %d", cfg.RequestBodySize.MaxBytes)
		}
	})

	t.Run("request ID options", func(t *testing.T) {
		cfg := DefaultConfig
		WithRequestIDOptions(WithRequestIDHeader("X-Trace-ID"))(&cfg)
		if len(cfg.RequestIDOptions) != 1 {
			t.Errorf("expected 1 request ID option, got %d", len(cfg.RequestIDOptions))
		}
		cfg.Build()
		if cfg.RequestID.Header != "X-Trace-ID" {
			t.Errorf("expected Header = X-Trace-ID, got %s", cfg.RequestID.Header)
		}
	})

	t.Run("request logger options", func(t *testing.T) {
		cfg := DefaultConfig
		WithRequestLoggerOptions(
			WithRequestLoggerLogErrors(true),
			WithRequestLoggerExemptPaths([]string{"/health", "/metrics"}),
		)(&cfg)
		if len(cfg.RequestLoggerOptions) != 2 {
			t.Errorf("expected 2 request logger options, got %d", len(cfg.RequestLoggerOptions))
		}
		cfg.Build()
		if len(cfg.RequestLogger.ExemptPaths) != 2 {
			t.Errorf("expected ExemptPaths length = 2, got %d", len(cfg.RequestLogger.ExemptPaths))
		}
		if cfg.RequestLogger.ExemptPaths[0] != "/health" {
			t.Errorf("expected first exempt path = /health, got %s", cfg.RequestLogger.ExemptPaths[0])
		}
		if cfg.RequestLogger.ExemptPaths[1] != "/metrics" {
			t.Errorf("expected second exempt path = /metrics, got %s", cfg.RequestLogger.ExemptPaths[1])
		}
	})

	t.Run("security headers options", func(t *testing.T) {
		cfg := DefaultConfig
		WithSecurityHeadersOptions(
			WithSecurityHeadersXFrameOptions("DENY"),
			WithSecurityHeadersCSP("default-src 'self'"),
		)(&cfg)
		if len(cfg.SecurityHeadersOptions) != 2 {
			t.Errorf("expected 2 security headers options, got %d", len(cfg.SecurityHeadersOptions))
		}
		cfg.Build()
		if cfg.SecurityHeaders.XFrameOptions != "DENY" {
			t.Errorf("expected XFrameOptions = DENY, got %s", cfg.SecurityHeaders.XFrameOptions)
		}
		if cfg.SecurityHeaders.ContentSecurityPolicy != "default-src 'self'" {
			t.Errorf("expected CSP = default-src 'self', got %s", cfg.SecurityHeaders.ContentSecurityPolicy)
		}
	})
}

func TestConfigZeroValues(t *testing.T) {
	var cfg Config
	if cfg.Addr != "" {
		t.Error("expected zero value for Addr to be empty string")
	}
	if cfg.TLSAddr != "" {
		t.Error("expected zero value for TLSAddr to be empty string")
	}
	if cfg.DisableDefaultMiddlewares != false {
		t.Error("expected zero value for DisableDefaultMiddlewares to be false")
	}
	if cfg.DefaultMiddlewares != nil {
		t.Error("expected zero value for DefaultMiddlewares to be nil")
	}
	if cfg.Logger != nil {
		t.Error("expected zero value for Logger to be nil")
	}
	if cfg.Server != nil {
		t.Error("expected zero value for Server to be nil")
	}
	if cfg.TLSServer != nil {
		t.Error("expected zero value for TLSServer to be nil")
	}
	if cfg.Listener != nil {
		t.Error("expected zero value for Listener to be nil")
	}
	if cfg.TLSListener != nil {
		t.Error("expected zero value for TLSListener to be nil")
	}
	if cfg.CertFile != "" {
		t.Error("expected zero value for CertFile to be empty string")
	}
	if cfg.KeyFile != "" {
		t.Error("expected zero value for KeyFile to be empty string")
	}
	if cfg.AutocertManager != nil {
		t.Error("expected zero value for AutocertManager to be nil")
	}
}

func TestMultipleOptionsApplication(t *testing.T) {
	cfg := DefaultConfig
	WithAddr(":9000")(&cfg)
	WithTLSAddr(":9443")(&cfg)
	WithDisableDefaultMiddlewares()(&cfg)
	WithCertFile("custom.pem")(&cfg)
	WithKeyFile("custom.key")(&cfg)

	if cfg.Addr != ":9000" {
		t.Errorf("expected Addr = :9000, got %s", cfg.Addr)
	}
	if cfg.TLSAddr != ":9443" {
		t.Errorf("expected TLSAddr = :9443, got %s", cfg.TLSAddr)
	}
	if !cfg.DisableDefaultMiddlewares {
		t.Error("expected DisableDefaultMiddlewares to be true")
	}
	if cfg.CertFile != "custom.pem" {
		t.Errorf("expected CertFile = custom.pem, got %s", cfg.CertFile)
	}
	if cfg.KeyFile != "custom.key" {
		t.Errorf("expected KeyFile = custom.key, got %s", cfg.KeyFile)
	}
}
