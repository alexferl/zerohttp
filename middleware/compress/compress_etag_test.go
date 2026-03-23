package compress

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/middleware/etag"
)

// TestCompressETag_OrderIndependent verifies correct behavior regardless of
// middleware order. Per RFC 7232, ETags must represent the actual bytes sent.
// RECOMMENDED: Place ETag BEFORE New (ETag outer) so ETag captures
// compressed bytes from the inner New middleware.
// When New is before ETag, ETag captures uncompressed content.
func TestCompressETag_OrderIndependent(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
		_, _ = w.Write([]byte("hello world test content"))
	})

	// Order 1: Compress -> ETag (Compress inner, ETag outer - RECOMMENDED)
	// ETag captures compressed content from inner Compress and computes correct ETag
	compressMw := New(Config{
		Types: []string{"text/plain"},
	})
	etagMw := etag.New()
	chain1 := etagMw(compressMw(handler))

	// Order 2: ETag -> Compress (ETag inner, Compress outer - NOT RECOMMENDED)
	// ETag captures uncompressed content before outer Compress compresses it
	chain2 := compressMw(etagMw(handler))

	// Request with Accept-Encoding: gzip
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
	rec1 := httptest.NewRecorder()
	chain1.ServeHTTP(rec1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
	rec2 := httptest.NewRecorder()
	chain2.ServeHTTP(rec2, req2)

	// Both should return Content-Encoding: gzip
	if rec1.Header().Get(httpx.HeaderContentEncoding) != httpx.ContentEncodingGzip {
		t.Errorf("chain1: expected Content-Encoding=%q, got %q",
			httpx.ContentEncodingGzip, rec1.Header().Get(httpx.HeaderContentEncoding))
	}
	if rec2.Header().Get(httpx.HeaderContentEncoding) != httpx.ContentEncodingGzip {
		t.Errorf("chain2: expected Content-Encoding=%q, got %q",
			httpx.ContentEncodingGzip, rec2.Header().Get(httpx.HeaderContentEncoding))
	}

	// Both should have ETags
	etag1 := rec1.Header().Get(httpx.HeaderETag)
	etag2 := rec2.Header().Get(httpx.HeaderETag)
	if etag1 == "" {
		t.Error("chain1 (ETag->Compress): expected ETag header")
	}
	if etag2 == "" {
		t.Error("chain2 (Compress->ETag): expected ETag header")
	}

	// ETags will be different because they're computed on different content:
	// - chain1 (RECOMMENDED): ETag is for compressed content (correct)
	// - chain2: ETag is for uncompressed content (incorrect when compressed)
	if etag1 == etag2 {
		t.Error("ETags should be different (one for compressed, one for uncompressed)")
	}

	// Both should return the same compressed body that decompresses to same content
	body1 := decompressGzip(t, rec1.Body.Bytes())
	body2 := decompressGzip(t, rec2.Body.Bytes())
	if body1 != body2 {
		t.Errorf("Bodies should be identical. Got:\n%s\nvs\n%s", body1, body2)
	}
}

// TestCompressETag_AlreadyEncoded verifies behavior when response
// already has Content-Encoding header set (should skip compression)
func TestCompressETag_AlreadyEncoded(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentEncoding, "br") // Already encoded
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
		_, _ = w.Write([]byte("already compressed"))
	})

	compressMw := New(Config{
		Types: []string{"text/plain"},
	})
	etagMw := etag.New()
	chain := compressMw(etagMw(handler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	// Should preserve the original Content-Encoding (br), not gzip
	encoding := rec.Header().Get(httpx.HeaderContentEncoding)
	if encoding != "br" {
		t.Errorf("expected Content-Encoding to remain 'br', got %q", encoding)
	}
}

// TestCompressETag_RangeRequest verifies that 206 Partial Content responses
// are not compressed (per spec, range requests should not be transformed)
func TestCompressETag_RangeRequest(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("partial"))
	})

	compressMw := New(Config{
		Types: []string{"text/plain"},
	})
	etagMw := etag.New()
	chain := compressMw(etagMw(handler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	// 206 responses should not be compressed
	if rec.Code != http.StatusPartialContent {
		t.Errorf("expected status %d, got %d", http.StatusPartialContent, rec.Code)
	}
	if rec.Header().Get(httpx.HeaderContentEncoding) != "" {
		t.Error("206 Partial Content should not be compressed")
	}
}

// TestCompressETag_CacheControlNoTransform verifies that Cache-Control: no-transform
// prevents compression
func TestCompressETag_CacheControlNoTransform(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderCacheControl, httpx.CacheControlNoTransform)
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
		_, _ = w.Write([]byte("do not transform"))
	})

	compressMw := New(Config{
		Types: []string{"text/plain"},
	})
	etagMw := etag.New()
	chain := compressMw(etagMw(handler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	// Should not be compressed when no-transform is set
	if rec.Header().Get(httpx.HeaderContentEncoding) != "" {
		t.Error("Cache-Control: no-transform should prevent compression")
	}
}

// TestCompressETag_HeadRequest verifies HEAD requests negotiate compression
// (headers reflect encoded representation) but body is empty
func TestCompressETag_HeadRequest(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
		// Write would be called but body discarded for HEAD
		_, _ = w.Write([]byte("this is the body content"))
	})

	compressMw := New(Config{
		Types: []string{"text/plain"},
	})
	etagMw := etag.New()
	chain := compressMw(etagMw(handler))

	req := httptest.NewRequest(http.MethodHead, "/", nil)
	req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	// HEAD request should have Content-Encoding header set (negotiated)
	if rec.Header().Get(httpx.HeaderContentEncoding) != httpx.ContentEncodingGzip {
		t.Errorf("HEAD request should negotiate Content-Encoding, got %q",
			rec.Header().Get(httpx.HeaderContentEncoding))
	}

	// HEAD request body should be empty
	if rec.Body.Len() != 0 {
		t.Errorf("HEAD request body should be empty, got %d bytes", rec.Body.Len())
	}

	// ETag should be present (reflecting the encoded representation)
	if rec.Header().Get(httpx.HeaderETag) == "" {
		t.Error("HEAD request should have ETag header")
	}
}

// TestCompressETag_ETagRecomputedFromCompressedBytes verifies that ETags
// are computed differently for compressed vs uncompressed content.
// When ETag wraps New (RECOMMENDED order), the ETag is computed on compressed bytes.
func TestCompressETag_ETagRecomputedFromCompressedBytes(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
		_, _ = w.Write([]byte("test content for etag"))
	})

	// First, get ETag without compression
	etagMw := etag.New()
	chainNoCompress := etagMw(handler)

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	// No Accept-Encoding, so no compression
	rec1 := httptest.NewRecorder()
	chainNoCompress.ServeHTTP(rec1, req1)

	etagNoCompress := rec1.Header().Get(httpx.HeaderETag)
	if etagNoCompress == "" {
		t.Fatal("Expected ETag for uncompressed response")
	}

	// Now get ETag with compression using RECOMMENDED order (ETag wraps Compress)
	compressMw := New(Config{
		Types: []string{"text/plain"},
	})
	chainWithCompress := etagMw(compressMw(handler))

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
	rec2 := httptest.NewRecorder()
	chainWithCompress.ServeHTTP(rec2, req2)

	etagWithCompress := rec2.Header().Get(httpx.HeaderETag)
	if etagWithCompress == "" {
		t.Fatal("Expected ETag for compressed response when ETag wraps Compress")
	}

	// ETags should be different because content is different (compressed vs uncompressed)
	if etagNoCompress == etagWithCompress {
		t.Error("ETag should be different for compressed vs uncompressed content")
	}

	// Both should be strong ETags (no W/ prefix by default)
	if !strings.HasPrefix(etagNoCompress, `"`) {
		t.Errorf("Expected strong ETag format for uncompressed, got %s", etagNoCompress)
	}
	if !strings.HasPrefix(etagWithCompress, `"`) {
		t.Errorf("Expected strong ETag format for compressed, got %s", etagWithCompress)
	}
}

// TestCompress_StatusCodesWithoutBodies verifies that status codes without
// bodies (1xx, 204, 304) are not compressed
func TestCompress_StatusCodesWithoutBodies(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"100 Continue", http.StatusContinue},
		{"101 Switching Protocols", http.StatusSwitchingProtocols},
		{"204 No Content", http.StatusNoContent},
		{"304 Not Modified", http.StatusNotModified},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
				w.WriteHeader(tt.status)
			})

			compressMw := New(Config{
				Types: []string{"text/plain"},
			})
			chain := compressMw(handler)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
			rec := httptest.NewRecorder()
			chain.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, rec.Code)
			}
			// These status codes should not be compressed
			if rec.Header().Get(httpx.HeaderContentEncoding) != "" {
				t.Errorf("status %d should not be compressed", tt.status)
			}
		})
	}
}

// TestCompressETag_VaryHeaderAdded verifies that Accept-Encoding is added to Vary
func TestCompressETag_VaryHeaderAdded(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
		_, _ = w.Write([]byte("test content"))
	})

	compressMw := New(Config{
		Types: []string{"text/plain"},
	})
	etagMw := etag.New()
	chain := compressMw(etagMw(handler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(httpx.HeaderAcceptEncoding, httpx.ContentEncodingGzip)
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	// Vary header should include Accept-Encoding
	vary := rec.Header()["Vary"]
	if !slices.Contains(vary, httpx.HeaderAcceptEncoding) {
		t.Errorf("Vary header should include %q, got %v", httpx.HeaderAcceptEncoding, vary)
	}
}

// decompressGzip decompresses gzip-encoded bytes for testing
func decompressGzip(t *testing.T, data []byte) string {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer func() { _ = reader.Close() }()

	result, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to decompress: %v", err)
	}
	return string(result)
}
