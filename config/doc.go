// Package config provides configuration structs for zerohttp servers and middleware.
//
// This package contains all configuration types used to customize zerohttp behavior,
// from server settings to individual middleware options.
//
// # Server Configuration
//
// The main [Config] struct holds all server and middleware configuration:
//
//	app := zh.New(config.Config{
//	    Addr: ":8080",
//	    ReadTimeout:  10 * time.Second,
//	    WriteTimeout: 15 * time.Second,
//	    Logger: myLogger,
//	})
//
// # Middleware Configuration
//
// Each middleware has its own configuration struct:
//
// # CORS
//
//	app.Use(middleware.CORS(config.CORSConfig{
//	    AllowedOrigins: []string{"https://example.com"},
//	    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
//	    AllowCredentials: true,
//	    MaxAge: 86400,
//	}))
//
// Or use [DefaultCORSConfig] as a starting point.
//
// # Basic Authentication
//
//	app.Use(middleware.BasicAuth(config.BasicAuthConfig{
//	    Credentials: map[string]string{
//	        "admin": "secret-password",
//	    },
//	    Realm: "Restricted Area",
//	    ExemptPaths: []string{"/health"},
//	}))
//
// # JWT Authentication
//
//	cfg := config.JWTAuthConfig{
//	    TokenStore:     myTokenStore,
//	    RequiredClaims: []string{"sub"},
//	    ExemptPaths:    []string{"/login", "/register"},
//	}
//	app.Use(middleware.JWTAuth(cfg))
//
// For a zero-dependency JWT solution, use the built-in HS256:
//
//	store := middleware.NewHS256TokenStore(secret, middleware.HS256Options{
//	    Issuer: "my-app",
//	    AccessTokenTTL:  15 * time.Minute,
//	    RefreshTokenTTL: 7 * 24 * time.Hour,
//	})
//
// # Rate Limiting
//
//	app.Use(middleware.RateLimit(config.RateLimitConfig{
//	    Rate:      100,
//	    Window:    time.Minute,
//	    Algorithm: config.TokenBucket,
//	}))
//
// Algorithms: [TokenBucket] or [SlidingWindow].
//
// # Compression
//
//	app.Use(middleware.Compress(config.CompressConfig{
//	    Level:     6,
//	    Types:     []string{"text/html", "application/json"},
//	    MinLength: 1024,
//	}))
//
// # Security Headers
//
//	app.Use(middleware.SecurityHeaders(config.SecurityHeadersConfig{
//	    CSP:           "default-src 'self'; script-src 'self'",
//	    XFrameOptions: "DENY",
//	    HSTS: config.HSTSConfig{
//	        MaxAge: 31536000,
//	        Preload: true,
//	    },
//	}))
//
// # Request Logging
//
//	app.Use(middleware.RequestLogger(logger, config.RequestLoggerConfig{
//	    Fields: []string{"method", "path", "status", "duration", "ip"},
//	}))
//
// # Circuit Breaker
//
//	app.Use(middleware.CircuitBreaker(config.CircuitBreakerConfig{
//	    FailureThreshold: 5,
//	    RecoveryTimeout:  30 * time.Second,
//	    SuccessThreshold: 3,
//	}))
//
// # Request Body Size Limit
//
//	app.Use(middleware.RequestBodySize(config.RequestBodySizeConfig{
//	    MaxBytes: 1024 * 1024, // 1MB
//	}))
//
// # Request ID
//
//	app.Use(middleware.RequestID(config.RequestIDConfig{
//	    Header: "X-Request-ID",
//	}))
//
// # Timeout
//
//	app.Use(middleware.Timeout(config.TimeoutConfig{
//	    Duration: 30 * time.Second,
//	}))
//
// # CSRF
//
//	app.Use(middleware.CSRF(config.CSRFConfig{
//	    TokenLength: 32,
//	    CookieName:  "csrf_token",
//	    HeaderName:  "X-CSRF-Token",
//	}))
//
// # TLS Configuration
//
//	app := zh.New(config.Config{
//	    TLS: config.TLSConfig{
//	        Addr:     ":8443",
//	        CertFile: "server.crt",
//	        KeyFile:  "server.key",
//	    },
//	})
//
// # Metrics Configuration
//
//	app := zh.New(config.Config{
//	    Metrics: config.MetricsConfig{
//	        Enabled:  true,
//	        Endpoint: "/metrics",
//	        ExcludePaths: []string{"/health", "/readyz"},
//	    },
//	})
//
// # Custom Validator
//
// Bring your own struct validator:
//
//	app := zh.New(config.Config{
//	    Validator: myCustomValidator, // implements Validator interface
//	})
//
// # Default Configurations
//
// Most middlewares provide a Default*Config variable with sensible defaults.
// These can be used as-is or as a base for customization:
//
//	// Use defaults
//	app.Use(middleware.CORS(config.DefaultCORSConfig))
//
//	// Customize from defaults
//	cfg := config.DefaultCORSConfig
//	cfg.AllowedOrigins = []string{"https://example.com"}
//	app.Use(middleware.CORS(cfg))
package config
