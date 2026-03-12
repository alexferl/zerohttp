package middleware

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/log"
)

// noopLogger is a logger that discards all output for benchmarking
type noopLogger struct{}

func (n *noopLogger) Debug(msg string, fields ...log.Field)      {}
func (n *noopLogger) Info(msg string, fields ...log.Field)       {}
func (n *noopLogger) Warn(msg string, fields ...log.Field)       {}
func (n *noopLogger) Error(msg string, fields ...log.Field)      {}
func (n *noopLogger) Panic(msg string, fields ...log.Field)      {}
func (n *noopLogger) Fatal(msg string, fields ...log.Field)      {}
func (n *noopLogger) WithFields(fields ...log.Field) log.Logger  { return n }
func (n *noopLogger) WithContext(ctx context.Context) log.Logger { return n }

// BenchmarkRequestLogger_FieldConfiguration measures overhead with different field sets
func BenchmarkRequestLogger_FieldConfiguration(b *testing.B) {
	fieldConfigs := []struct {
		name   string
		fields []config.LogField
	}{
		{"Minimal", []config.LogField{config.FieldMethod, config.FieldPath, config.FieldStatus}},
		{"Default", config.DefaultRequestLoggerConfig.Fields},
		{"AllFields", []config.LogField{
			config.FieldMethod, config.FieldURI, config.FieldPath, config.FieldHost,
			config.FieldProtocol, config.FieldReferer, config.FieldUserAgent,
			config.FieldStatus, config.FieldDurationNS, config.FieldDurationHuman,
			config.FieldRemoteAddr, config.FieldClientIP, config.FieldRequestID,
		}},
	}

	for _, fc := range fieldConfigs {
		b.Run(fc.name, func(b *testing.B) {
			logger := &noopLogger{}
			mw := RequestLogger(logger, config.RequestLoggerConfig{
				Fields: fc.fields,
			})

			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-Request-Id", "test-123")

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
			}
		})
	}
}

// BenchmarkRequestLogger_BodyCapturing measures body capture overhead
func BenchmarkRequestLogger_BodyCapturing(b *testing.B) {
	bodySizes := []struct {
		name string
		size int
	}{
		{"NoBody", 0},
		{"100B", 100},
		{"1KB", 1024},
		{"10KB", 10 * 1024},
	}

	for _, bs := range bodySizes {
		b.Run("RequestBody_"+bs.name, func(b *testing.B) {
			logger := &noopLogger{}
			mw := RequestLogger(logger, config.RequestLoggerConfig{
				Fields:         []config.LogField{config.FieldMethod, config.FieldRequestBody},
				LogRequestBody: true,
				MaxBodySize:    1024,
			})

			bodyData := bytes.Repeat([]byte("x"), bs.size)

			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				w.WriteHeader(http.StatusOK)
			}))

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				rr := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(bodyData))
				handler.ServeHTTP(rr, req)
			}
		})

		b.Run("ResponseBody_"+bs.name, func(b *testing.B) {
			logger := &noopLogger{}
			mw := RequestLogger(logger, config.RequestLoggerConfig{
				Fields:          []config.LogField{config.FieldMethod, config.FieldResponseBody},
				LogResponseBody: true,
				MaxBodySize:     1024,
			})

			bodyData := bytes.Repeat([]byte("x"), bs.size)

			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(bodyData)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			b.ReportAllocs()
			b.SetBytes(int64(bs.size))
			b.ResetTimer()

			for b.Loop() {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
			}
		})
	}
}

// BenchmarkRequestLogger_SensitiveDataMasking measures JSON masking performance
func BenchmarkRequestLogger_SensitiveDataMasking(b *testing.B) {
	testCases := []struct {
		name string
		json string
	}{
		{"SimpleObject", `{"username":"john","password":"secret123"}`},
		{"NestedObject", `{"user":{"name":"john","password":"secret"},"data":{"token":"abc123"}}`},
		{"ArrayOfObjects", `[{"id":1,"password":"pass1"},{"id":2,"password":"pass2"}]`},
		{"LargeObject", `{"field1":"value1","field2":"value2","password":"secret","token":"abc","key":"xyz","field3":"value3","field4":"value4"}`},
	}

	sensitiveFields := []string{"password", "token", "key"}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			logger := &noopLogger{}
			mw := RequestLogger(logger, config.RequestLoggerConfig{
				Fields:          []config.LogField{config.FieldMethod, config.FieldRequestBody},
				LogRequestBody:  true,
				MaxBodySize:     4096,
				SensitiveFields: sensitiveFields,
			})

			body := []byte(tc.json)

			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				rr := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
				handler.ServeHTTP(rr, req)
			}
		})
	}
}

// BenchmarkRequestLogger_LogLevelRouting measures log level selection overhead
func BenchmarkRequestLogger_LogLevelRouting(b *testing.B) {
	statusCodes := []int{http.StatusOK, http.StatusBadRequest, http.StatusInternalServerError}

	for _, code := range statusCodes {
		b.Run(http.StatusText(code), func(b *testing.B) {
			logger := &noopLogger{}
			mw := RequestLogger(logger, config.RequestLoggerConfig{
				Fields:    []config.LogField{config.FieldMethod, config.FieldStatus},
				LogErrors: true,
			})

			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
			}
		})
	}
}

// BenchmarkRequestLogger_ExemptPaths measures exempt path checking overhead
func BenchmarkRequestLogger_ExemptPaths(b *testing.B) {
	testCases := []struct {
		name        string
		exemptPaths []string
		path        string
	}{
		{"NoExemptions", nil, "/api/users"},
		{"OneExemption", []string{"/health"}, "/api/users"},
		{"ManyExemptions", []string{"/health", "/metrics", "/ready", "/live"}, "/api/users"},
		{"ExemptMatch", []string{"/health"}, "/health"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			logger := &noopLogger{}
			mw := RequestLogger(logger, config.RequestLoggerConfig{
				Fields:      []config.LogField{config.FieldMethod, config.FieldPath},
				ExemptPaths: tc.exemptPaths,
			})

			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
			}
		})
	}
}

// BenchmarkRequestLogger_Baseline compares against no middleware
func BenchmarkRequestLogger_Baseline(b *testing.B) {
	b.Run("NoMiddleware", func(b *testing.B) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}
	})

	b.Run("WithMiddleware", func(b *testing.B) {
		logger := &noopLogger{}
		mw := RequestLogger(logger, config.RequestLoggerConfig{
			Fields: []config.LogField{config.FieldMethod, config.FieldPath, config.FieldStatus},
		})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}
	})
}

// BenchmarkRequestLogger_Concurrent measures concurrent logging performance
func BenchmarkRequestLogger_Concurrent(b *testing.B) {
	concurrencyLevels := []int{1, 10, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Goroutines%d", concurrency), func(b *testing.B) {
			logger := &noopLogger{}
			mw := RequestLogger(logger, config.RequestLoggerConfig{
				Fields: []config.LogField{config.FieldMethod, config.FieldPath, config.FieldStatus},
			})

			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			b.ReportAllocs()
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					rr := httptest.NewRecorder()
					handler.ServeHTTP(rr, req)
				}
			})
		})
	}
}

// BenchmarkRequestLogger_MaskSensitiveData benchmarks the masking function directly
func BenchmarkRequestLogger_MaskSensitiveData(b *testing.B) {
	testCases := []struct {
		name string
		data string
	}{
		{"SmallObject", `{"password":"secret"}`},
		{"MediumObject", `{"username":"john","password":"secret123","email":"john@example.com"}`},
		{"NestedObject", `{"user":{"name":"john","password":"secret"},"api_key":"xyz"}`},
		{"ArrayOfObjects", `[{"id":1,"password":"pass1"},{"id":2,"password":"pass2"},{"id":3,"password":"pass3"}]`},
		{"NoSensitive", `{"name":"john","email":"john@example.com"}`},
	}

	sensitiveFields := []string{"password", "api_key", "secret"}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				maskSensitiveData(tc.data, sensitiveFields)
			}
		})
	}
}

// BenchmarkRequestLogger_MaskObject benchmarks the maskObject function directly
func BenchmarkRequestLogger_MaskObject(b *testing.B) {
	obj := map[string]any{
		"username": "john",
		"password": "secret123",
		"email":    "john@example.com",
		"nested": map[string]any{
			"token": "abc123",
			"data":  "value",
		},
	}

	sensitiveFields := []string{"password", "token"}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		maskObject(obj, sensitiveFields)
	}
}

// BenchmarkRequestLogger_CaptureRequestBody benchmarks body capture directly
func BenchmarkRequestLogger_CaptureRequestBody(b *testing.B) {
	bodySizes := []int{0, 100, 1024, 10 * 1024}

	for _, size := range bodySizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := bytes.Repeat([]byte("x"), size)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(data))
				captureRequestBody(req, 1024)
			}
		})
	}
}

// BenchmarkRequestLogger_LogRequest benchmarks the LogRequest function directly
func BenchmarkRequestLogger_LogRequest(b *testing.B) {
	cfg := config.RequestLoggerConfig{
		Fields: []config.LogField{
			config.FieldMethod, config.FieldURI, config.FieldPath,
			config.FieldStatus, config.FieldDurationNS,
		},
	}

	fieldMap := make(map[config.LogField]bool)
	for _, f := range cfg.Fields {
		fieldMap[f] = true
	}

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("X-Request-Id", "test-123")

	logger := &noopLogger{}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		LogRequest(logger, cfg, fieldMap, req, http.StatusOK, 5*time.Millisecond, "", "")
	}
}
