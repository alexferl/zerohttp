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
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/metrics"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestMatchAcceptEncoding(t *testing.T) {
	tests := []struct {
		name     string
		accepted []string
		encoding string
		want     bool
	}{
		// Basic matching
		{"exact match", []string{"gzip"}, "gzip", true},
		{"no match", []string{"deflate"}, "gzip", false},
		{"wildcard", []string{"*"}, "gzip", true},

		// Quality values (RFC 7231)
		{"q=0 rejected", []string{"gzip;q=0"}, "gzip", false},
		{"q=0.0 rejected", []string{"gzip;q=0.0"}, "gzip", false},
		{"q=0.00 rejected", []string{"gzip;q=0.00"}, "gzip", false},
		{"q=0.000 rejected", []string{"gzip;q=0.000"}, "gzip", false},
		{"q=0 with spaces", []string{"gzip; q=0"}, "gzip", false},
		{"q=0.0 with spaces", []string{"gzip; q=0.0"}, "gzip", false},

		// Non-zero q-values accepted
		{"q=1 accepted", []string{"gzip;q=1"}, "gzip", true},
		{"q=0.5 accepted", []string{"gzip;q=0.5"}, "gzip", true},
		{"q=0.05 accepted", []string{"gzip;q=0.05"}, "gzip", true},
		{"q=0.001 accepted", []string{"gzip;q=0.001"}, "gzip", true},

		// No substring matching (br vs brotli)
		{"br not match brotli", []string{"brotli"}, "br", false},
		{"gzip not match gzip2", []string{"gzip2"}, "gzip", false},
		{"deflate not match deflatefast", []string{"deflatefast"}, "deflate", false},

		// Multiple encodings
		{"second encoding matches", []string{"br", "gzip"}, "gzip", true},
		{"first encoding q=0, second matches", []string{"br;q=0", "gzip"}, "br", false},
		{"first encoding q=0, second matches encoding", []string{"br;q=0", "gzip"}, "gzip", true},

		// Wildcard with q=0
		{"wildcard q=0 rejected", []string{"*;q=0"}, "gzip", false},
		{"wildcard q=0.0 rejected", []string{"*;q=0.0"}, "gzip", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchAcceptEncoding(tt.accepted, tt.encoding)
			if got != tt.want {
				t.Errorf("matchAcceptEncoding(%v, %q) = %v, want %v",
					tt.accepted, tt.encoding, got, tt.want)
			}
		})
	}
}

func TestCompress(t *testing.T) {
	middleware := Compress(config.CompressConfig{
		Types: []string{"text/html", "application/json"},
		Level: 9,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, "text/html")
		_, err := w.Write([]byte("test content for compression"))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).Header(httpx.HeaderContentEncoding, httpx.ContentEncodingGzip)

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

func TestCompressExcludedPaths(t *testing.T) {
	middleware := Compress(config.CompressConfig{
		ExcludedPaths: []string{"/health", "/metrics", "/api/internal/"},
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
				w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
				_, err := w.Write([]byte(strings.Repeat("test content ", 10)))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			hasCompression := rr.Header().Get(httpx.HeaderContentEncoding) != ""
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
				w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
				_, err := w.Write([]byte(strings.Repeat("test content ", 10)))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set(httpx.HeaderAcceptEncoding, tt.acceptEncoding)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			zhtest.AssertWith(t, rr).Header(httpx.HeaderContentEncoding, tt.expectedEncoding)
		})
	}
}

func TestCompressAllOptions(t *testing.T) {
	// Test all options working together
	middleware := Compress(config.CompressConfig{
		Level:         9,
		Types:         []string{"text/html", "application/json"},
		Algorithms:    []config.CompressionAlgorithm{config.Gzip},
		ExcludedPaths: []string{"/health"},
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
			name:             "excluded path not compressed",
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
				w.Header().Set(httpx.HeaderContentType, tt.contentType)
				_, err := w.Write([]byte(tt.content))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set(httpx.HeaderAcceptEncoding, tt.acceptEncoding)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			zhtest.AssertWith(t, rr).Header(httpx.HeaderContentEncoding, tt.expectedEncoding)
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
	// Add nop to algorithm order so selectEncoder will consider it
	compressor.algorithmOrder = append([]config.CompressionAlgorithm{config.CompressionAlgorithm("nop")}, compressor.algorithmOrder...)
	compressor.algorithms[config.CompressionAlgorithm("nop")] = true

	if len(compressor.encoders) != 1 {
		t.Errorf("nop encoder should be stored in the encoders map")
	}

	// Use the compressor middleware with HTTP handlers
	mux.Handle("/gethtml", compressor.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, "text/html")
		_, err := w.Write([]byte("textstring"))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))

	mux.Handle("/getcss", compressor.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, "text/css")
		_, err := w.Write([]byte("textstring"))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})))

	mux.Handle("/getplain", compressor.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
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
			if got := resp.Header.Get(httpx.HeaderContentEncoding); got != tc.expectedEncoding {
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
				w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
				_, err := w.Write([]byte(strings.Repeat("test data ", 100)))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			zhtest.AssertWith(t, rr).Header(httpx.HeaderContentEncoding, httpx.ContentEncodingGzip)

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
			description:    "Should use default level, types, algorithms, and excluded paths",
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
			name:           "nil excluded paths - use defaults",
			config:         config.CompressConfig{ExcludedPaths: nil},
			description:    "Nil excluded paths should use default (empty list)",
			shouldCompress: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := Compress(tt.config)
			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(httpx.HeaderContentType, "text/html")
				_, err := w.Write([]byte(strings.Repeat("test content ", 50)))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			hasCompression := rr.Header().Get(httpx.HeaderContentEncoding) != ""
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
			name:              "empty excluded paths - allow compression",
			config:            config.CompressConfig{ExcludedPaths: []string{}},
			description:       "Empty excluded paths should allow compression on all paths",
			expectCompression: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := Compress(tt.config)
			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(httpx.HeaderContentType, "text/html")
				_, err := w.Write([]byte(strings.Repeat("test content ", 50)))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			hasCompression := rr.Header().Get(httpx.HeaderContentEncoding) != ""
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
			Level:         0,   // Should fallback to default (6)
			Types:         nil, // Should use defaults
			Algorithms:    nil, // Should use defaults
			ExcludedPaths: nil, // Should use defaults
		})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, "application/json") // Default compressible type
			_, err := w.Write([]byte(strings.Repeat("json data ", 50)))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		zhtest.AssertWith(t, rr).Header(httpx.HeaderContentEncoding, httpx.ContentEncodingGzip)
	})

	t.Run("overrides work when explicitly set", func(t *testing.T) {
		mw := Compress(config.CompressConfig{
			Level:      9,                                          // Explicit level
			Types:      []string{"text/custom"},                    // Custom type only
			Algorithms: []config.CompressionAlgorithm{config.Gzip}, // Gzip only
		})

		// Test with non-matching type
		handler1 := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, "application/json") // Not in custom types
			_, err := w.Write([]byte(strings.Repeat("json data ", 50)))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))

		req1 := httptest.NewRequest(http.MethodGet, "/", nil)
		req1.Header.Set(httpx.HeaderAcceptEncoding, "gzip")
		rr1 := httptest.NewRecorder()
		handler1.ServeHTTP(rr1, req1)

		zhtest.AssertWith(t, rr1).HeaderNotExists(httpx.HeaderContentEncoding)

		// Test with matching custom type
		handler2 := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, "text/custom") // Matches custom types
			_, err := w.Write([]byte(strings.Repeat("custom data ", 50)))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))

		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.Header.Set(httpx.HeaderAcceptEncoding, "gzip")
		rr2 := httptest.NewRecorder()
		handler2.ServeHTTP(rr2, req2)

		zhtest.AssertWith(t, rr2).Header(httpx.HeaderContentEncoding, httpx.ContentEncodingGzip)
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
		req.Header.Set(httpx.HeaderAcceptEncoding, encodingsString)
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
	switch resp.Header.Get(httpx.HeaderContentEncoding) {
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
		Enabled:       config.Bool(true),
		PathLabelFunc: func(p string) string { return p },
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
		_, _ = w.Write([]byte(strings.Repeat("test content ", 100)))
	}))
	wrapped := metricsMw(handler)

	// Test gzip compression
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Header().Get(httpx.HeaderContentEncoding) != "gzip" {
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

func TestCompress_WriteHeader_MultipleCalls(t *testing.T) {
	mw := Compress(config.CompressConfig{
		Types: []string{"text/html"},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, "text/html")
		w.WriteHeader(http.StatusOK)
		// Subsequent WriteHeader calls should be ignored (standard library behavior)
		w.WriteHeader(http.StatusInternalServerError)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("test content"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// The status code should be 200 (first WriteHeader), not 404 or 500
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d (subsequent WriteHeader calls should be ignored)", rr.Code)
	}
}

func TestCompress_HEADRequest_ContentEncoding(t *testing.T) {
	// Test that HEAD requests return Content-Encoding header when compression is applicable
	// even when Content-Type is not explicitly set (relies on default detection)
	mw := Compress(config.CompressConfig{
		Types: []string{"text/html"},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handler does NOT set Content-Type explicitly
		// This simulates real-world cases where WriteHeader is called before Content-Type is determined
		w.WriteHeader(http.StatusOK)
		// Note: No body written, which is typical for HEAD requests
	}))

	req := httptest.NewRequest(http.MethodHead, "/test", nil)
	req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// HEAD requests should still return Content-Encoding header when Accept-Encoding is sent
	// because the middleware should assume compression is possible
	encoding := rr.Header().Get(httpx.HeaderContentEncoding)
	if encoding != httpx.ContentEncodingGzip {
		t.Errorf("HEAD request: expected Content-Encoding=%q, got %q", httpx.ContentEncodingGzip, encoding)
	}
}

func TestCompress_HEADRequest_WithContentType_ContentEncoding(t *testing.T) {
	// Test that HEAD requests return Content-Encoding header when Content-Type is set
	mw := Compress(config.CompressConfig{
		Types: []string{"text/html"},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, "text/html")
		w.WriteHeader(http.StatusOK)
		// Note: No body written, which is typical for HEAD requests
	}))

	req := httptest.NewRequest(http.MethodHead, "/test", nil)
	req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// HEAD requests should return Content-Encoding header when Content-Type is compressible
	encoding := rr.Header().Get(httpx.HeaderContentEncoding)
	if encoding != httpx.ContentEncodingGzip {
		t.Errorf("HEAD request with Content-Type: expected Content-Encoding=%q, got %q", httpx.ContentEncodingGzip, encoding)
	}
}

func TestCompress_HEADRequest_NoContentLength(t *testing.T) {
	// Test that HEAD requests don't have Content-Length set for compressed responses
	// since we can't know the compressed size without actually compressing
	mw := Compress(config.CompressConfig{
		Types: []string{"text/html"},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, "text/html")
		_, _ = w.Write([]byte("test content that would be compressed"))
	}))

	req := httptest.NewRequest(http.MethodHead, "/test", nil)
	req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// HEAD requests should not have Content-Length for compressed responses
	// since we can't determine the compressed size without actually compressing
	contentLength := rr.Header().Get(httpx.HeaderContentLength)
	if contentLength != "" {
		t.Errorf("HEAD request: expected no Content-Length for compressed response, got %q", contentLength)
	}

	// But Content-Encoding should still be set
	encoding := rr.Header().Get(httpx.HeaderContentEncoding)
	if encoding != httpx.ContentEncodingGzip {
		t.Errorf("HEAD request: expected Content-Encoding=%q, got %q", httpx.ContentEncodingGzip, encoding)
	}
}

func TestCompress_IncludedPaths(t *testing.T) {
	middleware := Compress(config.CompressConfig{
		IncludedPaths: []string{"/api/", "/compress/*"},
	})

	tests := []struct {
		path           string
		shouldCompress bool
	}{
		{"/api/data", true},
		{"/api/users", true},
		{"/compress/anything", true},
		{"/other", false},
		{"/health", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
				_, err := w.Write([]byte(strings.Repeat("test content ", 10)))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			hasCompression := rr.Header().Get(httpx.HeaderContentEncoding) != ""
			if hasCompression != tt.shouldCompress {
				t.Errorf("path %s: expected compression=%v, got compression=%v", tt.path, tt.shouldCompress, hasCompression)
			}
		})
	}
}

func TestCompress_BothExcludedAndIncludedPathsPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when both ExcludedPaths and IncludedPaths are set")
		}
	}()

	_ = Compress(config.CompressConfig{
		ExcludedPaths: []string{"/health"},
		IncludedPaths: []string{"/api"},
	})
}

// TestCompressionProvider tests the pluggable CompressionProvider interface
func TestCompressionProvider(t *testing.T) {
	t.Run("custom encoder is registered and used", func(t *testing.T) {
		nopEncoder := &testEncoder{encoding: "nop"}
		provider := &testProvider{encoders: map[string]config.CompressionEncoder{
			"nop": nopEncoder,
		}}

		mw := Compress(config.CompressConfig{
			Types:      []string{"text/plain"},
			Algorithms: []config.CompressionAlgorithm{"nop", config.Gzip},
			Providers:  []config.CompressionProvider{provider},
		})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
			_, _ = w.Write([]byte("test content"))
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(httpx.HeaderAcceptEncoding, "nop")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		zhtest.AssertWith(t, rr).Header(httpx.HeaderContentEncoding, "nop")
		if !nopEncoder.used {
			t.Error("expected custom encoder to be used")
		}
	})

	t.Run("provider returns nil for unsupported encoding", func(t *testing.T) {
		provider := &testProvider{encoders: map[string]config.CompressionEncoder{}}

		mw := Compress(config.CompressConfig{
			Types:      []string{"text/plain"},
			Algorithms: []config.CompressionAlgorithm{config.Gzip},
			Providers:  []config.CompressionProvider{provider},
		})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
			_, _ = w.Write([]byte("test content"))
		}))

		// Request gzip which is built-in, not from provider
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		zhtest.AssertWith(t, rr).Header(httpx.HeaderContentEncoding, httpx.ContentEncodingGzip)
	})

	t.Run("custom encoder with level parameter", func(t *testing.T) {
		levelEncoder := &testEncoderWithLevel{encoding: "testlevel"}
		provider := &testProvider{encoders: map[string]config.CompressionEncoder{
			"testlevel": levelEncoder,
		}}

		mw := Compress(config.CompressConfig{
			Types:      []string{"text/plain"},
			Algorithms: []config.CompressionAlgorithm{"testlevel"},
			Level:      9,
			Providers:  []config.CompressionProvider{provider},
		})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
			_, _ = w.Write([]byte("test content"))
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(httpx.HeaderAcceptEncoding, "testlevel")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if levelEncoder.receivedLevel != 9 {
			t.Errorf("expected level 9, got %d", levelEncoder.receivedLevel)
		}
	})

	t.Run("multiple custom encoders from provider", func(t *testing.T) {
		encoder1 := &testEncoder{encoding: "custom1"}
		encoder2 := &testEncoder{encoding: "custom2"}
		provider := &testProvider{encoders: map[string]config.CompressionEncoder{
			"custom1": encoder1,
			"custom2": encoder2,
		}}

		mw := Compress(config.CompressConfig{
			Types:      []string{"text/plain"},
			Algorithms: []config.CompressionAlgorithm{"custom1", "custom2"},
			Providers:  []config.CompressionProvider{provider},
		})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
			_, _ = w.Write([]byte("test content"))
		}))

		// Test first custom encoder
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req1.Header.Set(httpx.HeaderAcceptEncoding, "custom1")
		rr1 := httptest.NewRecorder()
		handler.ServeHTTP(rr1, req1)
		zhtest.AssertWith(t, rr1).Header(httpx.HeaderContentEncoding, "custom1")

		// Test second custom encoder
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req2.Header.Set(httpx.HeaderAcceptEncoding, "custom2")
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, req2)
		zhtest.AssertWith(t, rr2).Header(httpx.HeaderContentEncoding, "custom2")
	})

	t.Run("custom encoder alongside built-in", func(t *testing.T) {
		customEncoder := &testEncoder{encoding: "custom"}
		provider := &testProvider{encoders: map[string]config.CompressionEncoder{
			"custom": customEncoder,
		}}

		mw := Compress(config.CompressConfig{
			Types:      []string{"text/plain"},
			Algorithms: []config.CompressionAlgorithm{config.Gzip, "custom"},
			Providers:  []config.CompressionProvider{provider},
		})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
			_, _ = w.Write([]byte("test content"))
		}))

		// Test built-in gzip still works
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req1.Header.Set(httpx.HeaderAcceptEncoding, "gzip")
		rr1 := httptest.NewRecorder()
		handler.ServeHTTP(rr1, req1)
		zhtest.AssertWith(t, rr1).Header(httpx.HeaderContentEncoding, httpx.ContentEncodingGzip)

		// Test custom encoder
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req2.Header.Set(httpx.HeaderAcceptEncoding, "custom")
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, req2)
		zhtest.AssertWith(t, rr2).Header(httpx.HeaderContentEncoding, "custom")
	})

	t.Run("multiple providers", func(t *testing.T) {
		encoder1 := &testEncoder{encoding: "brotli"}
		encoder2 := &testEncoder{encoding: "zstd"}
		provider1 := &testProvider{encoders: map[string]config.CompressionEncoder{
			"brotli": encoder1,
		}}
		provider2 := &testProvider{encoders: map[string]config.CompressionEncoder{
			"zstd": encoder2,
		}}

		mw := Compress(config.CompressConfig{
			Types:      []string{"text/plain"},
			Algorithms: []config.CompressionAlgorithm{"brotli", "zstd"},
			Providers:  []config.CompressionProvider{provider1, provider2},
		})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
			_, _ = w.Write([]byte("test content"))
		}))

		// Test first provider's encoder
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req1.Header.Set(httpx.HeaderAcceptEncoding, "brotli")
		rr1 := httptest.NewRecorder()
		handler.ServeHTTP(rr1, req1)
		zhtest.AssertWith(t, rr1).Header(httpx.HeaderContentEncoding, "brotli")

		// Test second provider's encoder
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req2.Header.Set(httpx.HeaderAcceptEncoding, "zstd")
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, req2)
		zhtest.AssertWith(t, rr2).Header(httpx.HeaderContentEncoding, "zstd")
	})

	t.Run("nil provider uses defaults only", func(t *testing.T) {
		mw := Compress(config.CompressConfig{
			Types:      []string{"text/plain"},
			Algorithms: []config.CompressionAlgorithm{config.Gzip},
			Providers:  nil,
		})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
			_, _ = w.Write([]byte("test content"))
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		zhtest.AssertWith(t, rr).Header(httpx.HeaderContentEncoding, httpx.ContentEncodingGzip)
	})

	t.Run("algorithms order determines precedence with providers", func(t *testing.T) {
		// When both algorithms are acceptable, the first one in Algorithms list should be chosen
		brotliEncoder := &testEncoder{encoding: "br"}
		zstdEncoder := &testEncoder{encoding: "zstd"}

		brotliProvider := &testProvider{encoders: map[string]config.CompressionEncoder{
			"br": brotliEncoder,
		}}
		zstdProvider := &testProvider{encoders: map[string]config.CompressionEncoder{
			"zstd": zstdEncoder,
		}}

		// Test with zstd first in algorithms - it should be preferred
		mw := Compress(config.CompressConfig{
			Types:      []string{"text/plain"},
			Algorithms: []config.CompressionAlgorithm{"zstd", "br"},
			Providers:  []config.CompressionProvider{brotliProvider, zstdProvider},
		})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
			_, _ = w.Write([]byte("test content"))
		}))

		// Client accepts both, but zstd should be chosen (first in Algorithms)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(httpx.HeaderAcceptEncoding, "br, zstd")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		zhtest.AssertWith(t, rr).Header(httpx.HeaderContentEncoding, "zstd")

		// Test with brotli first in algorithms - it should be preferred
		mw2 := Compress(config.CompressConfig{
			Types:      []string{"text/plain"},
			Algorithms: []config.CompressionAlgorithm{"br", "zstd"},
			Providers:  []config.CompressionProvider{zstdProvider, brotliProvider},
		})

		handler2 := mw2(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
			_, _ = w.Write([]byte("test content"))
		}))

		// Client accepts both, but brotli should be chosen (first in Algorithms)
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req2.Header.Set(httpx.HeaderAcceptEncoding, "br, zstd")
		rr2 := httptest.NewRecorder()
		handler2.ServeHTTP(rr2, req2)
		zhtest.AssertWith(t, rr2).Header(httpx.HeaderContentEncoding, "br")
	})
}

// testEncoder is a simple test implementation of config.CompressionEncoder
type testEncoder struct {
	encoding string
	used     bool
}

func (e *testEncoder) Encode(w io.Writer, level int) io.Writer {
	e.used = true
	return w
}

func (e *testEncoder) Encoding() string {
	return e.encoding
}

// testEncoderWithLevel captures the level parameter for verification
type testEncoderWithLevel struct {
	encoding      string
	receivedLevel int
}

func (e *testEncoderWithLevel) Encode(w io.Writer, level int) io.Writer {
	e.receivedLevel = level
	return w
}

func (e *testEncoderWithLevel) Encoding() string {
	return e.encoding
}

// testProvider is a test implementation of config.CompressionProvider
type testProvider struct {
	encoders map[string]config.CompressionEncoder
}

func (p *testProvider) GetEncoder(encoding string) config.CompressionEncoder {
	if enc, ok := p.encoders[encoding]; ok {
		return enc
	}
	return nil
}

// TestCompress_WriteHeaderBeforeWrite tests that calling WriteHeader before Write
// still works correctly with compression middleware
func TestCompress_WriteHeaderBeforeWrite(t *testing.T) {
	t.Run("status code is preserved when WriteHeader is called before Write", func(t *testing.T) {
		mw := Compress(config.CompressConfig{
			Types: []string{"text/html"},
		})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMETextHTML)
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("<html><body>Not Found</body></html>"))
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d", http.StatusNotFound, rr.Code)
		}

		if rr.Header().Get(httpx.HeaderContentEncoding) != httpx.ContentEncodingGzip {
			t.Errorf("expected Content-Encoding=%q, got %q", httpx.ContentEncodingGzip, rr.Header().Get(httpx.HeaderContentEncoding))
		}
	})

	t.Run("non-200 status codes work correctly", func(t *testing.T) {
		mw := Compress(config.CompressConfig{
			Types: []string{"application/json"},
		})

		testCases := []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusInternalServerError,
		}

		for _, status := range testCases {
			t.Run(fmt.Sprintf("status_%d", status), func(t *testing.T) {
				handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
					w.WriteHeader(status)
					_, _ = w.Write([]byte(`{"error":"test"}`))
				}))

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
				rr := httptest.NewRecorder()

				handler.ServeHTTP(rr, req)

				if rr.Code != status {
					t.Errorf("expected status code %d, got %d", status, rr.Code)
				}
			})
		}
	})

	t.Run("multiple WriteHeader calls use first status", func(t *testing.T) {
		mw := Compress(config.CompressConfig{
			Types: []string{"text/plain"},
		})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
			w.WriteHeader(http.StatusNotFound)
			w.WriteHeader(http.StatusOK) // Should be ignored
			_, _ = w.Write([]byte("content"))
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d", http.StatusNotFound, rr.Code)
		}
	})

	t.Run("implicit 200 when Write is called without WriteHeader", func(t *testing.T) {
		mw := Compress(config.CompressConfig{
			Types: []string{"text/html"},
		})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMETextHTML)
			// No WriteHeader call - Write should trigger implicit 200
			_, _ = w.Write([]byte("<html>content</html>"))
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}

		if rr.Header().Get(httpx.HeaderContentEncoding) != httpx.ContentEncodingGzip {
			t.Errorf("expected Content-Encoding=%q, got %q", httpx.ContentEncodingGzip, rr.Header().Get(httpx.HeaderContentEncoding))
		}
	})

	t.Run("non-compressible types are not compressed", func(t *testing.T) {
		mw := Compress(config.CompressConfig{
			Types: []string{"text/html"}, // Only HTML, not images
		})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, "image/png")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("fake-image-data"))
		}))

		req := httptest.NewRequest(http.MethodGet, "/test.png", nil)
		req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}

		// Should NOT have Content-Encoding header since image/png is not compressible
		if rr.Header().Get(httpx.HeaderContentEncoding) != "" {
			t.Errorf("expected no Content-Encoding for image/png, got %q", rr.Header().Get(httpx.HeaderContentEncoding))
		}
	})

	t.Run("no compression when Accept-Encoding not set", func(t *testing.T) {
		mw := Compress(config.CompressConfig{
			Types: []string{"text/html"},
		})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(httpx.HeaderContentType, httpx.MIMETextHTML)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html>content</html>"))
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		// No Accept-Encoding header
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Header().Get(httpx.HeaderContentEncoding) != "" {
			t.Errorf("expected no Content-Encoding without Accept-Encoding, got %q", rr.Header().Get(httpx.HeaderContentEncoding))
		}
	})
}
