package middleware

import (
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/metrics"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestCompress(t *testing.T) {
	middleware := Compress(config.CompressConfig{
		Types: []string{"text/html", "application/json"},
		Level: 9,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, err := w.Write([]byte("test content for compression"))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).Header("Content-Encoding", "gzip")

	// Test decompression
	reader, err := gzip.NewReader(rr.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			t.Fatalf("gzip reader close error (non-fatal): %v", err)
		}
	}()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read decompressed data: %v", err)
	}

	if string(decompressed) != "test content for compression" {
		t.Errorf("decompressed content doesn't match original")
	}
}

func TestCompressExemptPaths(t *testing.T) {
	middleware := Compress(config.CompressConfig{
		ExemptPaths: []string{"/health", "/metrics", "/api/internal/"},
	})

	tests := []struct {
		path           string
		shouldCompress bool
	}{
		{"/health", false},
		{"/metrics", false},
		{"/api/internal/", false},
		{"/api/internal/status", false},
		{"/api/public", true},
		{"/regular", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, err := w.Write([]byte(strings.Repeat("test content ", 10)))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Accept-Encoding", "gzip")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			hasCompression := rr.Header().Get("Content-Encoding") != ""
			if hasCompression != tt.shouldCompress {
				t.Errorf("path %s: expected compression=%v, got compression=%v", tt.path, tt.shouldCompress, hasCompression)
			}
		})
	}
}

func TestCompressAlgorithms(t *testing.T) {
	tests := []struct {
		name             string
		algorithms       []config.CompressionAlgorithm
		acceptEncoding   string
		expectedEncoding string
	}{
		{
			name:             "gzip only",
			algorithms:       []config.CompressionAlgorithm{config.Gzip},
			acceptEncoding:   "gzip, deflate",
			expectedEncoding: "gzip",
		},
		{
			name:             "deflate only",
			algorithms:       []config.CompressionAlgorithm{config.Deflate},
			acceptEncoding:   "gzip, deflate",
			expectedEncoding: "deflate",
		},
		{
			name:             "no matching algorithm",
			algorithms:       []config.CompressionAlgorithm{config.Gzip},
			acceptEncoding:   "deflate",
			expectedEncoding: "",
		},
		{
			name:             "both algorithms allowed",
			algorithms:       []config.CompressionAlgorithm{config.Gzip, config.Deflate},
			acceptEncoding:   "deflate, gzip",
			expectedEncoding: "gzip", // gzip has higher precedence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := Compress(config.CompressConfig{
				Algorithms: tt.algorithms,
			})

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, err := w.Write([]byte(strings.Repeat("test content ", 10)))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			zhtest.AssertWith(t, rr).Header("Content-Encoding", tt.expectedEncoding)
		})
	}
}

func TestCompressAllOptions(t *testing.T) {
	// Test all options working together
	middleware := Compress(config.CompressConfig{
		Level:       9,
		Types:       []string{"text/html", "application/json"},
		Algorithms:  []config.CompressionAlgorithm{config.Gzip},
		ExemptPaths: []string{"/health"},
	})

	tests := []struct {
		name             string
		path             string
		contentType      string
		content          string
		acceptEncoding   string
		expectedEncoding string
	}{
		{
			name:             "compressed HTML",
			path:             "/page",
			contentType:      "text/html",
			content:          strings.Repeat("html content ", 10),
			acceptEncoding:   "gzip",
			expectedEncoding: "gzip",
		},
		{
			name:             "exempt path not compressed",
			path:             "/health",
			contentType:      "text/html",
			content:          strings.Repeat("health check ", 10),
			acceptEncoding:   "gzip",
			expectedEncoding: "",
		},
		{
			name:             "unsupported content type",
			path:             "/image",
			contentType:      "image/png",
			content:          strings.Repeat("binary data ", 10),
			acceptEncoding:   "gzip",
			expectedEncoding: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				_, err := w.Write([]byte(tt.content))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			zhtest.AssertWith(t, rr).Header("Content-Encoding", tt.expectedEncoding)
		})
	}
}

func TestCompressor(t *testing.T) {
	mux := http.NewServeMux()

	compressor := NewCompressor(5, "text/html", "text/css")
	if len(compressor.encoders) != 0 || len(compressor.pooledEncoders) != 2 {
		t.Errorf("gzip and deflate should be pooled")
	}

	compressor.SetEncoder("nop", func(w io.Writer, _ int) io.Writer {
		return w
	})

	if len(compressor.encoders) != 1 {
		t.Errorf("nop encoder should be stored in the encoders map")
	}

	// Use the compressor middleware with HTTP handlers
	mux.Handle("/gethtml", compressor.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, err := w.Write([]byte("textstring"))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))

	mux.Handle("/getcss", compressor.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, err := w.Write([]byte("textstring"))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))

	mux.Handle("/getplain", compressor.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, err := w.Write([]byte("textstring"))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))

	ts := httptest.NewServer(mux)
	defer ts.Close()

	tests := []struct {
		name              string
		path              string
		expectedEncoding  string
		acceptedEncodings []string
	}{
		{
			name:              "no expected encodings due to no accepted encodings",
			path:              "/gethtml",
			acceptedEncodings: nil,
			expectedEncoding:  "",
		},
		{
			name:              "no expected encodings due to content type",
			path:              "/getplain",
			acceptedEncodings: nil,
			expectedEncoding:  "",
		},
		{
			name:              "gzip is only encoding",
			path:              "/gethtml",
			acceptedEncodings: []string{"gzip"},
			expectedEncoding:  "gzip",
		},
		{
			name:              "gzip is preferred over deflate",
			path:              "/getcss",
			acceptedEncodings: []string{"gzip", "deflate"},
			expectedEncoding:  "gzip",
		},
		{
			name:              "deflate is used",
			path:              "/getcss",
			acceptedEncodings: []string{"deflate"},
			expectedEncoding:  "deflate",
		},
		{
			name:              "nop is preferred",
			path:              "/getcss",
			acceptedEncodings: []string{"nop, gzip, deflate"},
			expectedEncoding:  "nop",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resp, respString := testRequestWithAcceptedEncodings(t, ts, http.MethodGet, tc.path, tc.acceptedEncodings...)
			if respString != "textstring" {
				t.Errorf("response text doesn't match; expected:%q, got:%q", "textstring", respString)
			}
			if got := resp.Header.Get("Content-Encoding"); got != tc.expectedEncoding {
				t.Errorf("expected encoding %q but got %q", tc.expectedEncoding, got)
			}
		})
	}
}

func TestCompressorWildcards(t *testing.T) {
	tests := []struct {
		name       string
		recover    string
		types      []string
		typesCount int
		wcCount    int
	}{
		{
			name:       "defaults",
			typesCount: 11,
		},
		{
			name:       "no wildcard",
			types:      []string{"text/plain", "text/html"},
			typesCount: 2,
		},
		{
			name:    "invalid wildcard #1",
			types:   []string{"audio/*wav"},
			recover: "middleware/compress: Unsupported content-type wildcard pattern 'audio/*wav'. Only '/*' supported",
		},
		{
			name:    "invalid wildcard #2",
			types:   []string{"application*/*"},
			recover: "middleware/compress: Unsupported content-type wildcard pattern 'application*/*'. Only '/*' supported",
		},
		{
			name:    "valid wildcard",
			types:   []string{"text/*"},
			wcCount: 1,
		},
		{
			name:       "mixed",
			types:      []string{"audio/wav", "text/*"},
			typesCount: 1,
			wcCount:    1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if tt.recover == "" {
					tt.recover = "<nil>"
				}
				if r := recover(); tt.recover != fmt.Sprintf("%v", r) {
					t.Errorf("Unexpected value recovered: %v", r)
				}
			}()
			compressor := NewCompressor(5, tt.types...)
			if len(compressor.allowedTypes) != tt.typesCount {
				t.Errorf("expected %d allowedTypes, got %d", tt.typesCount, len(compressor.allowedTypes))
			}
			if len(compressor.allowedWildcards) != tt.wcCount {
				t.Errorf("expected %d allowedWildcards, got %d", tt.wcCount, len(compressor.allowedWildcards))
			}
		})
	}
}

func TestCompressorLevels(t *testing.T) {
	tests := []struct {
		name  string
		level int
	}{
		{"best speed", 1},
		{"default", 6},
		{"best compression", 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressor := NewCompressor(tt.level, "text/plain")

			handler := compressor.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, err := w.Write([]byte(strings.Repeat("test data ", 100)))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			zhtest.AssertWith(t, rr).Header("Content-Encoding", "gzip")

			// Verify the data can be decompressed
			reader, err := gzip.NewReader(rr.Body)
			if err != nil {
				t.Fatalf("failed to create gzip reader: %v", err)
			}
			defer func() {
				if err := reader.Close(); err != nil {
					t.Fatalf("gzip reader close error (non-fatal): %v", err)
				}
			}()

			_, err = io.ReadAll(reader)
			if err != nil {
				t.Fatalf("failed to read compressed data: %v", err)
			}
		})
	}
}

func TestCompressConfigDefaults(t *testing.T) {
	tests := []struct {
		name           string
		config         config.CompressConfig
		description    string
		shouldCompress bool
	}{
		{
			name:           "no config - use all defaults",
			config:         config.CompressConfig{},
			description:    "Should use default level, types, algorithms, and exempt paths",
			shouldCompress: true,
		},
		{
			name:           "zero level - fallback to default",
			config:         config.CompressConfig{Level: 0},
			description:    "Level 0 should fallback to default level (6)",
			shouldCompress: true,
		},
		{
			name:           "negative level - fallback to default",
			config:         config.CompressConfig{Level: -1},
			description:    "Negative level should fallback to default",
			shouldCompress: true,
		},
		{
			name:           "nil types - use defaults",
			config:         config.CompressConfig{Types: nil},
			description:    "Nil types should use default compressible types",
			shouldCompress: true,
		},
		{
			name:           "nil algorithms - use defaults",
			config:         config.CompressConfig{Algorithms: nil},
			description:    "Nil algorithms should use default algorithms (gzip, deflate)",
			shouldCompress: true,
		},
		{
			name:           "nil exempt paths - use defaults",
			config:         config.CompressConfig{ExemptPaths: nil},
			description:    "Nil exempt paths should use default (empty list)",
			shouldCompress: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := Compress(tt.config)
			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, err := w.Write([]byte(strings.Repeat("test content ", 50)))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			hasCompression := rr.Header().Get("Content-Encoding") != ""
			if hasCompression != tt.shouldCompress {
				t.Errorf("%s: expected compression=%v, got compression=%v",
					tt.description, tt.shouldCompress, hasCompression)
			}
		})
	}
}

func TestCompressConfigExplicitEmptyValues(t *testing.T) {
	tests := []struct {
		name              string
		config            config.CompressConfig
		description       string
		expectCompression bool
	}{
		{
			name:              "empty algorithms slice - disable compression",
			config:            config.CompressConfig{Algorithms: []config.CompressionAlgorithm{}},
			description:       "Empty algorithms slice should disable all compression algorithms",
			expectCompression: false,
		},
		{
			name:              "empty exempt paths - allow compression",
			config:            config.CompressConfig{ExemptPaths: []string{}},
			description:       "Empty exempt paths should allow compression on all paths",
			expectCompression: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := Compress(tt.config)
			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, err := w.Write([]byte(strings.Repeat("test content ", 50)))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			hasCompression := rr.Header().Get("Content-Encoding") != ""
			if hasCompression != tt.expectCompression {
				t.Errorf("%s: expected compression=%v, got compression=%v",
					tt.description, tt.expectCompression, hasCompression)
			}
		})
	}
}

func TestCompressConfigDefaultsVsOverrides(t *testing.T) {
	t.Run("defaults used when config values are nil or invalid", func(t *testing.T) {
		mw := Compress(config.CompressConfig{
			Level:       0,   // Should fallback to default (6)
			Types:       nil, // Should use defaults
			Algorithms:  nil, // Should use defaults
			ExemptPaths: nil, // Should use defaults
		})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json") // Default compressible type
			_, err := w.Write([]byte(strings.Repeat("json data ", 50)))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		zhtest.AssertWith(t, rr).Header("Content-Encoding", "gzip")
	})

	t.Run("overrides work when explicitly set", func(t *testing.T) {
		mw := Compress(config.CompressConfig{
			Level:      9,                                          // Explicit level
			Types:      []string{"text/custom"},                    // Custom type only
			Algorithms: []config.CompressionAlgorithm{config.Gzip}, // Gzip only
		})

		// Test with non-matching type
		handler1 := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json") // Not in custom types
			_, err := w.Write([]byte(strings.Repeat("json data ", 50)))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))

		req1 := httptest.NewRequest(http.MethodGet, "/", nil)
		req1.Header.Set("Accept-Encoding", "gzip")
		rr1 := httptest.NewRecorder()
		handler1.ServeHTTP(rr1, req1)

		zhtest.AssertWith(t, rr1).HeaderNotExists("Content-Encoding")

		// Test with matching custom type
		handler2 := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/custom") // Matches custom types
			_, err := w.Write([]byte(strings.Repeat("custom data ", 50)))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))

		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.Header.Set("Accept-Encoding", "gzip")
		rr2 := httptest.NewRecorder()
		handler2.ServeHTTP(rr2, req2)

		zhtest.AssertWith(t, rr2).Header("Content-Encoding", "gzip")
	})
}

func testRequestWithAcceptedEncodings(t *testing.T, ts *httptest.Server, method, path string, encodings ...string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}
	if len(encodings) > 0 {
		encodingsString := strings.Join(encodings, ",")
		req.Header.Set("Accept-Encoding", encodingsString)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	respBody := decodeResponseBody(t, resp)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("response body close error (non-fatal): %v", err)
		}
	}()

	return resp, respBody
}

func decodeResponseBody(t *testing.T, resp *http.Response) string {
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		var err error
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
	case "deflate":
		reader = flate.NewReader(resp.Body)
	default:
		reader = resp.Body
	}
	respBody, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
		return ""
	}

	if reader != resp.Body {
		if err := reader.Close(); err != nil {
			t.Fatalf("reader close error (non-fatal): %v", err)
		}
	}

	return string(respBody)
}

func TestCompress_Metrics(t *testing.T) {
	reg := metrics.NewRegistry()
	mw := Compress(config.CompressConfig{
		Types: []string{"text/plain"},
	})

	metricsMw := metrics.NewMiddleware(reg, config.MetricsConfig{
		Enabled:       true,
		PathLabelFunc: func(p string) string { return p },
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(strings.Repeat("test content ", 100)))
	}))
	wrapped := metricsMw(handler)

	// Test gzip compression
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("expected gzip encoding")
	}

	// Check metrics
	families := reg.Gather()
	var reqCounter *metrics.MetricFamily
	for _, f := range families {
		if f.Name == "compress_requests_total" {
			reqCounter = &f
			break
		}
	}
	if reqCounter == nil {
		t.Fatal("expected compress_requests_total metric")
	}

	hasGzip := false
	for _, m := range reqCounter.Metrics {
		if m.Labels["encoding"] == "gzip" && m.Counter > 0 {
			hasGzip = true
			break
		}
	}
	if !hasGzip {
		t.Error("expected gzip encoding in metrics")
	}
}
