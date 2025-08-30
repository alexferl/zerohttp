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
)

func TestCompress(t *testing.T) {
	middleware := Compress(
		config.WithCompressTypes([]string{"text/html", "application/json"}),
		config.WithCompressLevel(9),
	)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, err := w.Write([]byte("test content for compression"))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("expected gzip encoding, got %q", rr.Header().Get("Content-Encoding"))
	}

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
	middleware := Compress(
		config.WithCompressExemptPaths([]string{"/health", "/metrics", "/api/internal/"}),
	)

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

			req := httptest.NewRequest("GET", tt.path, nil)
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
			middleware := Compress(
				config.WithCompressAlgorithms(tt.algorithms),
			)

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, err := w.Write([]byte(strings.Repeat("test content ", 10)))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			actualEncoding := rr.Header().Get("Content-Encoding")
			if actualEncoding != tt.expectedEncoding {
				t.Errorf("expected encoding %q, got %q", tt.expectedEncoding, actualEncoding)
			}
		})
	}
}

func TestCompressAllOptions(t *testing.T) {
	// Test all options working together
	middleware := Compress(
		config.WithCompressLevel(9),
		config.WithCompressTypes([]string{"text/html", "application/json"}),
		config.WithCompressAlgorithms([]config.CompressionAlgorithm{config.Gzip}),
		config.WithCompressExemptPaths([]string{"/health"}),
	)

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

			req := httptest.NewRequest("GET", tt.path, nil)
			req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			actualEncoding := rr.Header().Get("Content-Encoding")
			if actualEncoding != tt.expectedEncoding {
				t.Errorf("expected encoding %q, got %q", tt.expectedEncoding, actualEncoding)
			}
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
			resp, respString := testRequestWithAcceptedEncodings(t, ts, "GET", tc.path, tc.acceptedEncodings...)
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

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Header().Get("Content-Encoding") != "gzip" {
				t.Errorf("expected gzip encoding for level %d", tt.level)
			}

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
		options        []config.CompressOption
		description    string
		shouldCompress bool
	}{
		{
			name:           "no options - use all defaults",
			options:        []config.CompressOption{},
			description:    "Should use default level, types, algorithms, and exempt paths",
			shouldCompress: true,
		},
		{
			name:           "zero level - fallback to default",
			options:        []config.CompressOption{config.WithCompressLevel(0)},
			description:    "Level 0 should fallback to default level (6)",
			shouldCompress: true,
		},
		{
			name:           "negative level - fallback to default",
			options:        []config.CompressOption{config.WithCompressLevel(-1)},
			description:    "Negative level should fallback to default",
			shouldCompress: true,
		},
		{
			name:           "nil types - use defaults",
			options:        []config.CompressOption{config.WithCompressTypes(nil)},
			description:    "Nil types should use default compressible types",
			shouldCompress: true,
		},
		{
			name:           "nil algorithms - use defaults",
			options:        []config.CompressOption{config.WithCompressAlgorithms(nil)},
			description:    "Nil algorithms should use default algorithms (gzip, deflate)",
			shouldCompress: true,
		},
		{
			name:           "nil exempt paths - use defaults",
			options:        []config.CompressOption{config.WithCompressExemptPaths(nil)},
			description:    "Nil exempt paths should use default (empty list)",
			shouldCompress: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := Compress(tt.options...)
			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, err := w.Write([]byte(strings.Repeat("test content ", 50)))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))

			req := httptest.NewRequest("GET", "/", nil)
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
		options           []config.CompressOption
		description       string
		expectCompression bool
	}{
		{
			name:              "empty algorithms slice - disable compression",
			options:           []config.CompressOption{config.WithCompressAlgorithms([]config.CompressionAlgorithm{})},
			description:       "Empty algorithms slice should disable all compression algorithms",
			expectCompression: false,
		},
		{
			name:              "empty exempt paths - allow compression",
			options:           []config.CompressOption{config.WithCompressExemptPaths([]string{})},
			description:       "Empty exempt paths should allow compression on all paths",
			expectCompression: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := Compress(tt.options...)
			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, err := w.Write([]byte(strings.Repeat("test content ", 50)))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))

			req := httptest.NewRequest("GET", "/", nil)
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
		mw := Compress(
			config.WithCompressLevel(0),         // Should fallback to default (6)
			config.WithCompressTypes(nil),       // Should use defaults
			config.WithCompressAlgorithms(nil),  // Should use defaults
			config.WithCompressExemptPaths(nil), // Should use defaults
		)

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json") // Default compressible type
			_, err := w.Write([]byte(strings.Repeat("json data ", 50)))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Header().Get("Content-Encoding") != "gzip" {
			t.Error("Expected compression with nil config values falling back to defaults")
		}
	})

	t.Run("overrides work when explicitly set", func(t *testing.T) {
		mw := Compress(
			config.WithCompressLevel(9),                                               // Explicit level
			config.WithCompressTypes([]string{"text/custom"}),                         // Custom type only
			config.WithCompressAlgorithms([]config.CompressionAlgorithm{config.Gzip}), // Gzip only
		)

		// Test with non-matching type
		handler1 := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json") // Not in custom types
			_, err := w.Write([]byte(strings.Repeat("json data ", 50)))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))

		req1 := httptest.NewRequest("GET", "/", nil)
		req1.Header.Set("Accept-Encoding", "gzip")
		rr1 := httptest.NewRecorder()
		handler1.ServeHTTP(rr1, req1)

		if rr1.Header().Get("Content-Encoding") != "" {
			t.Error("Should not compress non-matching content type")
		}

		// Test with matching custom type
		handler2 := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/custom") // Matches custom types
			_, err := w.Write([]byte(strings.Repeat("custom data ", 50)))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))

		req2 := httptest.NewRequest("GET", "/", nil)
		req2.Header.Set("Accept-Encoding", "gzip")
		rr2 := httptest.NewRecorder()
		handler2.ServeHTTP(rr2, req2)

		if rr2.Header().Get("Content-Encoding") != "gzip" {
			t.Error("Should compress matching custom content type")
		}
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
