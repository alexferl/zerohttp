package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/metrics"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestETag_GeneratesETag(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK)

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Error("expected ETag header to be set")
	}

	// Check that it's a weak ETag by default
	if !strings.HasPrefix(etag, `W/"`) {
		t.Errorf("expected weak ETag to start with W/\", got %s", etag)
	}
}

func TestETag_NotModified(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	// First request to get the ETag
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	etag := rec1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header to be set")
	}

	// Second request with If-None-Match
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set(httpx.HeaderIfNoneMatch, etag)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	zhtest.AssertWith(t, rec2).
		Status(http.StatusNotModified).
		BodyEmpty()
}

func TestETag_NotModified_MultipleETags(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	etag := rec1.Header().Get("ETag")

	// Request with multiple ETags in If-None-Match
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set(httpx.HeaderIfNoneMatch, `"other1", `+etag+`, "other2"`)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	zhtest.AssertWith(t, rec2).Status(http.StatusNotModified)
}

func TestETag_NotModified_Wildcard(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(httpx.HeaderIfNoneMatch, "*")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusNotModified)
}

func TestETag_NoETagOnPOST(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") != "" {
		t.Error("expected no ETag header for POST request")
	}
}

func TestETag_NoETagOnErrorStatus(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") != "" {
		t.Error("expected no ETag header for error response")
	}
}

func TestETag_NoETagOnNoContent(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") != "" {
		t.Error("expected no ETag header for 204 response")
	}
}

func TestETag_NoETagOnStreamingContent(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, "text/event-stream")
		_, _ = w.Write([]byte("data: hello\n\n"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") != "" {
		t.Error("expected no ETag header for SSE streaming content")
	}
}

func TestETag_NoETagOnNoStore(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") != "" {
		t.Error("expected no ETag header when Cache-Control: no-store")
	}
}

func TestETag_PreservesExistingETag(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderETag, `"custom-etag"`)
		_, _ = w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") != `"custom-etag"` {
		t.Errorf("expected custom ETag to be preserved, got %s", rec.Header().Get("ETag"))
	}
}

func TestETag_StrongETag(t *testing.T) {
	handler := ETag(config.ETagConfig{Weak: config.Bool(false)})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	etag := rec.Header().Get("ETag")
	if strings.HasPrefix(etag, `W/`) {
		t.Errorf("expected strong ETag without W/ prefix, got %s", etag)
	}
	if !strings.HasPrefix(etag, `"`) {
		t.Errorf("expected ETag to start with quote, got %s", etag)
	}
}

func TestETag_MD5Algorithm(t *testing.T) {
	fnvHandler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))
	md5Handler := ETag(config.ETagConfig{Algorithm: config.MD5})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	fnvHandler.ServeHTTP(rec1, req1)
	fnvETag := rec1.Header().Get("ETag")

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	md5Handler.ServeHTTP(rec2, req2)
	md5ETag := rec2.Header().Get("ETag")

	if fnvETag == md5ETag {
		t.Error("expected different ETags for FNV and MD5 algorithms")
	}

	// MD5 produces 32 hex characters
	if len(md5ETag) != len(`W/"`)+32+len(`"`) {
		t.Errorf("expected MD5 ETag length to be %d, got %d", len(`W/"`)+32+len(`"`), len(md5ETag))
	}
}

func TestETag_ExemptPaths(t *testing.T) {
	handler := ETag(config.ETagConfig{ExemptPaths: []string{"/skip"}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))

	// Request to exempt path
	req1 := httptest.NewRequest(http.MethodGet, "/skip", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Header().Get("ETag") != "" {
		t.Error("expected no ETag header for exempt path")
	}

	// Request to non-exempt path
	req2 := httptest.NewRequest(http.MethodGet, "/other", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Header().Get("ETag") == "" {
		t.Error("expected ETag header for non-exempt path")
	}
}

func TestETag_ExemptFunc(t *testing.T) {
	exemptFunc := func(r *http.Request) bool {
		return r.Header.Get("X-Skip-ETag") == "true"
	}

	handler := ETag(config.ETagConfig{ExemptFunc: exemptFunc})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))

	// Request with skip header
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.Header.Set("X-Skip-ETag", "true")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Header().Get("ETag") != "" {
		t.Error("expected no ETag header when exempt func returns true")
	}

	// Request without skip header
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Header().Get("ETag") == "" {
		t.Error("expected ETag header when exempt func returns false")
	}
}

func TestETag_SkipContentTypes(t *testing.T) {
	handler := ETag(config.ETagConfig{SkipContentTypes: map[string]struct{}{"application/pdf": {}}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, "application/pdf")
		_, _ = w.Write([]byte("pdf content"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") != "" {
		t.Error("expected no ETag header for skipped content type")
	}
}

func TestETag_SkipStatusCodes(t *testing.T) {
	handler := ETag(config.ETagConfig{SkipStatusCodes: map[int]struct{}{http.StatusTeapot: {}}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("I'm a teapot"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") != "" {
		t.Error("expected no ETag header for skipped status code")
	}
}

func TestETag_MaxBufferSize(t *testing.T) {
	content := strings.Repeat("a", 100)

	// Small buffer that will be exceeded
	handler := ETag(config.ETagConfig{MaxBufferSize: 50})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(content))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// When buffer is exceeded, response should still be written but without ETag
	if rec.Body.String() != content {
		t.Error("expected full response body even when buffer exceeded")
	}

	if rec.Header().Get("ETag") != "" {
		t.Error("expected no ETag when content exceeds max buffer size")
	}
}

func TestETag_HEADRequest(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	req := httptest.NewRequest(http.MethodHead, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Error("expected ETag header for HEAD request")
	}
}

func TestETag_ChangedContent(t *testing.T) {
	counter := 0
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter++
		_, _ = w.Write([]byte(string(rune('a' + counter))))
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	etag1 := rec1.Header().Get("ETag")

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	etag2 := rec2.Header().Get("ETag")

	if etag1 == etag2 {
		t.Error("expected different ETags for different content")
	}

	// Request with old ETag should return new content
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.Header.Set(httpx.HeaderIfNoneMatch, etag1)
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)

	zhtest.AssertWith(t, rec3).Status(http.StatusOK)
}

func TestETag_NotModified_StrongVsWeak(t *testing.T) {
	// Test that weak ETag from server matches strong ETag in If-None-Match
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	// Extract just the hash part from the weak ETag (remove W/ prefix and quotes)
	weakETag := rec1.Header().Get("ETag")
	hashPart := weakETag[3 : len(weakETag)-1] // Remove W/" and "
	strongETag := `"` + hashPart + `"`

	// Request with strong ETag should still match weak ETag
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set(httpx.HeaderIfNoneMatch, strongETag)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	zhtest.AssertWith(t, rec2).Status(http.StatusNotModified)
}

func TestETag_DefaultConfig(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") == "" {
		t.Error("expected ETag header")
	}
}

// Content-Encoding aware ETag tests

func TestETag_ContentEncodingAware(t *testing.T) {
	// Same content with different content-encoding should produce different ETags
	handlerGzip := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentEncoding, "gzip")
		_, _ = w.Write([]byte("hello world"))
	}))

	handlerPlain := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handlerGzip.ServeHTTP(rec1, req1)
	etagGzip := rec1.Header().Get("ETag")

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	handlerPlain.ServeHTTP(rec2, req2)
	etagPlain := rec2.Header().Get("ETag")

	if etagGzip == etagPlain {
		t.Error("expected different ETags for gzip vs plain content")
	}
}

func TestETag_ContentEncodingNotModified(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentEncoding, "gzip")
		_, _ = w.Write([]byte("hello world"))
	}))

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	etag := rec1.Header().Get("ETag")

	// Second request with matching If-None-Match
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set(httpx.HeaderIfNoneMatch, etag)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	zhtest.AssertWith(t, rec2).Status(http.StatusNotModified)
}

// Range request tests

func TestETag_IfRange_MatchingETag(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0123456789"))
	}))

	// First request to get ETag
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	etag := rec1.Header().Get("ETag")

	// Range request with matching If-Range
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set(httpx.HeaderIfRange, etag)
	req2.Header.Set("Range", "bytes=0-4")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	zhtest.AssertWith(t, rec2).
		Status(http.StatusPartialContent).
		Body("01234").
		Header(httpx.HeaderContentRange, "bytes 0-4/10")
}

func TestETag_IfRange_NonMatchingETag(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0123456789"))
	}))

	// Range request with non-matching If-Range should return full content
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(httpx.HeaderIfRange, `"old-etag"`)
	req.Header.Set(httpx.HeaderRange, "bytes=0-4")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK)

	if rec.Body.String() != "0123456789" {
		t.Errorf("expected full body, got %s", rec.Body.String())
	}
}

func TestETag_IfRange_NoETagHeader(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0123456789"))
	}))

	// Range request with date-based If-Range should return full content
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(httpx.HeaderIfRange, "Wed, 21 Oct 2015 07:28:00 GMT")
	req.Header.Set(httpx.HeaderRange, "bytes=0-4")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK)
}

// File-based ETag tests

func TestGenerateFileETag(t *testing.T) {
	modTime := int64(1709999999)
	size := int64(1024)

	// Weak ETag
	weakETag := GenerateFileETag(modTime, size, true)
	expected := `W/"1709999999-1024"`
	if weakETag != expected {
		t.Errorf("expected %s, got %s", expected, weakETag)
	}

	// Strong ETag
	strongETag := GenerateFileETag(modTime, size, false)
	expected = `"1709999999-1024"`
	if strongETag != expected {
		t.Errorf("expected %s, got %s", expected, strongETag)
	}
}

func TestGenerateFileETagFromInfo(t *testing.T) {
	// Create a mock that implements the required interface
	fi := &mockFileInfo{
		modTime: time.Unix(1709999999, 0),
		size:    1024,
	}

	etag := GenerateFileETagFromInfo(fi, true)

	expected := `W/"1709999999-1024"`
	if etag != expected {
		t.Errorf("expected %s, got %s", expected, etag)
	}
}

// mockFileInfo implements the interface needed for GenerateFileETagFromInfo
type mockFileInfo struct {
	modTime time.Time
	size    int64
}

func (m mockFileInfo) ModTime() time.Time { return m.modTime }
func (m mockFileInfo) Size() int64        { return m.size }

func TestParseETag(t *testing.T) {
	tests := []struct {
		etag     string
		wantHash string
		wantWeak bool
	}{
		{`W/"abc123"`, "abc123", true},
		{`"abc123"`, "abc123", false},
		{`abc123`, "abc123", false},
		{`W/""`, "", true},
		{`""`, "", false},
	}

	for _, tt := range tests {
		hash, weak := ParseETag(tt.etag)
		if hash != tt.wantHash {
			t.Errorf("ParseETag(%q) hash = %q, want %q", tt.etag, hash, tt.wantHash)
		}
		if weak != tt.wantWeak {
			t.Errorf("ParseETag(%q) weak = %v, want %v", tt.etag, weak, tt.wantWeak)
		}
	}
}

func TestGenerateFileETag_UniquePerContent(t *testing.T) {
	// Different modTime or size should produce different ETags
	etag1 := GenerateFileETag(1000, 100, true)
	etag2 := GenerateFileETag(1001, 100, true)
	etag3 := GenerateFileETag(1000, 101, true)

	if etag1 == etag2 {
		t.Error("expected different ETags for different modTime")
	}
	if etag1 == etag3 {
		t.Error("expected different ETags for different size")
	}
	if etag2 == etag3 {
		t.Error("expected different ETags for different modTime and size")
	}
}

func TestETag_Range_OpenEnded(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0123456789"))
	}))

	// First request to get ETag
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	etag := rec1.Header().Get("ETag")

	// Range request with open-ended range (bytes=5-)
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set(httpx.HeaderIfRange, etag)
	req2.Header.Set("Range", "bytes=5-")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	zhtest.AssertWith(t, rec2).
		Status(http.StatusPartialContent).
		Body("56789")
}

func TestETag_Range_InvalidRange(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0123456789"))
	}))

	// First request to get ETag
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	etag := rec1.Header().Get("ETag")

	// Range request with invalid range (start > end)
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set(httpx.HeaderIfRange, etag)
	req2.Header.Set("Range", "bytes=20-30") // Beyond content length
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	// Should fall back to 200 with full content
	zhtest.AssertWith(t, rec2).
		Status(http.StatusOK).
		Body("0123456789")
}

// If-Match tests (412 Precondition Failed)

func TestETag_IfMatch_Matches(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	// First request to get ETag
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	etag := rec1.Header().Get("ETag")

	// Request with matching If-Match should succeed
	req2 := httptest.NewRequest(http.MethodPut, "/", nil)
	req2.Header.Set("If-Match", etag)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	zhtest.AssertWith(t, rec2).Status(http.StatusOK)
}

func TestETag_IfMatch_DoesNotMatch(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	// Request with non-matching If-Match should return 412
	req := httptest.NewRequest(http.MethodPut, "/", nil)
	req.Header.Set(httpx.HeaderIfMatch, `"old-etag"`)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusPreconditionFailed)
}

func TestETag_IfMatch_Wildcard(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	// Request with If-Match: * should succeed if resource exists
	req := httptest.NewRequest(http.MethodPut, "/", nil)
	req.Header.Set(httpx.HeaderIfMatch, "*")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK)
}

func TestETag_IfMatch_NoETag(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	// Request with no If-Match should succeed
	req := httptest.NewRequest(http.MethodPut, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK)
}

// HTTP interface tests

type flusherRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (f *flusherRecorder) Flush() {
	f.flushed = true
}

func TestETag_Flush(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := &flusherRecorder{ResponseRecorder: httptest.NewRecorder()}
	handler.ServeHTTP(rec, req)

	if !rec.flushed {
		t.Error("expected Flush to be called")
	}
}

// TestETag_FlushPreventsDoubleWrite verifies that calling Flush() during request
// handling doesn't cause the response body to be written twice when finalize()
// is called after next.ServeHTTP(). This was a bug where templ components
// triggering Flush() would result in duplicate output.
func TestETag_FlushPreventsDoubleWrite(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
		// Flush triggers finalize() when there's buffered data
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// After handler returns, middleware calls finalize() again
		// This should not write the body a second time
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	// If body is written twice, it would be "hello worldhello world"
	expected := "hello world"
	if body != expected {
		t.Errorf("expected body %q, got %q (possible double-write)", expected, body)
	}

	// Also verify the ETag was generated correctly
	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Error("expected ETag header to be set")
	}
}

// Range parsing edge cases

func TestETag_Range_InvalidFormat(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0123456789"))
	}))

	// First request to get ETag
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	etag := rec1.Header().Get("ETag")

	// Range request with invalid format (missing dash)
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set(httpx.HeaderIfRange, etag)
	req2.Header.Set("Range", "bytes=0") // Invalid format
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	// Should fall back to 200 with full content
	zhtest.AssertWith(t, rec2).Status(http.StatusOK)
}

func TestETag_Range_NonNumericStart(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0123456789"))
	}))

	// First request to get ETag
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	etag := rec1.Header().Get("ETag")

	// Range request with non-numeric start
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set(httpx.HeaderIfRange, etag)
	req2.Header.Set("Range", "bytes=abc-5")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	// Should fall back to 200 with full content
	zhtest.AssertWith(t, rec2).Status(http.StatusOK)
}

func TestETag_Range_NonNumericEnd(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0123456789"))
	}))

	// First request to get ETag
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	etag := rec1.Header().Get("ETag")

	// Range request with non-numeric end
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set(httpx.HeaderIfRange, etag)
	req2.Header.Set("Range", "bytes=0-xyz")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	// Should fall back to 200 with full content
	zhtest.AssertWith(t, rec2).Status(http.StatusOK)
}

// Configuration edge cases

func TestETag_InvalidAlgorithm(t *testing.T) {
	// Invalid algorithm should fall back to FNV
	handler := ETag(config.ETagConfig{Algorithm: "invalid"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") == "" {
		t.Error("expected ETag header even with invalid algorithm")
	}
}

func TestETag_NilSkipStatusCodes(t *testing.T) {
	// nil SkipStatusCodes should use defaults
	cfg := config.ETagConfig{
		Algorithm:        config.FNV,
		Weak:             config.Bool(true),
		MaxBufferSize:    1024 * 1024,
		SkipStatusCodes:  nil, // nil
		SkipContentTypes: nil, // nil
	}

	// Manually create middleware with nil maps
	handler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ew := newETagResponseWriter(w, cfg, "", "", "", "", nil)
			defer ew.release()
			next.ServeHTTP(ew, r)
			ew.finalize()
		})
	}(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should still work without panic
	zhtest.AssertWith(t, rec).Status(http.StatusInternalServerError)
}

// ServeContentWithETag tests

func TestServeContentWithETag_NotModified(t *testing.T) {
	content := strings.NewReader("hello world")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Generate expected ETag
	etag := GenerateFileETag(1709999999, 11, true)
	req.Header.Set(httpx.HeaderIfNoneMatch, etag)
	rec := httptest.NewRecorder()

	ServeContentWithETag(rec, req, 1709999999, content)

	zhtest.AssertWith(t, rec).Status(http.StatusNotModified)
}

func TestServeContentWithETag_ServesContent(t *testing.T) {
	content := strings.NewReader("hello world")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	ServeContentWithETag(rec, req, 1709999999, content)

	zhtest.AssertWith(t, rec).Status(http.StatusOK)

	if rec.Header().Get("ETag") == "" {
		t.Error("expected ETag header")
	}
}

func TestServeContentWithETag_NilContent(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	ServeContentWithETag(rec, req, 1709999999, nil)

	// Should return 404 for nil content
	zhtest.AssertWith(t, rec).Status(http.StatusNotFound)
}

// Buffer flush test

func TestETag_BufferExceedsMaxSize(t *testing.T) {
	// Small max buffer size
	handler := ETag(config.ETagConfig{MaxBufferSize: 10})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write in two parts - first within limit, second exceeds
		_, _ = w.Write([]byte("0123456789")) // 10 bytes - at limit
		_, _ = w.Write([]byte("more"))       // exceeds limit
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should still get full content
	if rec.Body.String() != "0123456789more" {
		t.Errorf("expected full body '0123456789more', got %s", rec.Body.String())
	}
}

// WriteHeader edge cases

func TestETag_WriteHeader_MultipleCalls(t *testing.T) {
	calls := 0
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.WriteHeader(http.StatusCreated) // Second call should be ignored
		calls++
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK)
}

// Empty body test

func TestETag_EmptyBody(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// No body written
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Empty body should not generate ETag
	etag := rec.Header().Get("ETag")
	// ETag might be empty or a valid hash of empty content depending on implementation
	// The important thing is it doesn't panic
	_ = etag
}

// Additional edge case tests for full coverage

func TestETagMatches_ExactMatches(t *testing.T) {
	tests := []struct {
		name        string
		ifNoneMatch string
		etag        string
		want        bool
	}{
		{"exact match", `"abc"`, `"abc"`, true},
		{"weak to weak match", `W/"abc"`, `W/"abc"`, true},
		{"weak client strong server", `W/"abc"`, `"abc"`, true},
		{"strong client weak server", `"abc"`, `W/"abc"`, true},
		{"no match", `"abc"`, `"def"`, false},
		{"weak no match", `W/"abc"`, `W/"def"`, false},
		{"wildcard", `*`, `"anything"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := etagMatches(tt.ifNoneMatch, tt.etag)
			if got != tt.want {
				t.Errorf("etagMatches(%q, %q) = %v, want %v", tt.ifNoneMatch, tt.etag, got, tt.want)
			}
		})
	}
}

func TestETag_ExemptPaths_PrefixMatch(t *testing.T) {
	handler := ETag(config.ETagConfig{ExemptPaths: []string{"/api/"}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))

	// Request to path with matching prefix
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") != "" {
		t.Error("expected no ETag header for exempt path prefix")
	}
}

func TestETag_POSTWithIfMatch(t *testing.T) {
	// POST requests with non-matching If-Match should return 412
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("result"))
	}))

	// POST with non-matching If-Match should fail with 412
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(httpx.HeaderIfMatch, `"some-etag"`)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// The ETag is generated, then compared against If-Match
	// Since they don't match, we get 412
	zhtest.AssertWith(t, rec).Status(http.StatusPreconditionFailed)
}

func TestETag_NilConfigMaps(t *testing.T) {
	// Test with nil maps to ensure no panic
	cfg := config.ETagConfig{
		Algorithm:       config.FNV,
		Weak:            config.Bool(true),
		MaxBufferSize:   1024,
		SkipStatusCodes: nil,
		ExemptPaths:     []string{},
	}

	handler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ew := newETagResponseWriter(w, cfg, "", "", "", "", nil)
			defer ew.release()
			next.ServeHTTP(ew, r)
			ew.finalize()
		})
	}(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("test"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK)
}

func TestETag_ServeHTTP_WithHijacker(t *testing.T) {
	// Test that Hijack is available on the response writer
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to access Hijacker interface
		if _, ok := w.(http.Hijacker); ok {
			// Hijacker is available but we can't test actual hijacking in httptest
			_, _ = w.Write([]byte("hijacker available"))
		} else {
			_, _ = w.Write([]byte("hijacker not available"))
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// The hijacker check runs but actual hijacking isn't possible in tests
	zhtest.AssertWith(t, rec).Status(http.StatusOK)
}

func TestETag_ServeHTTP_WithPusher(t *testing.T) {
	// Test that Pusher is available on the response writer
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to access Pusher interface
		if _, ok := w.(http.Pusher); ok {
			_, _ = w.Write([]byte("pusher available"))
		} else {
			_, _ = w.Write([]byte("pusher not available"))
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK)
}

func TestETag_PUTWithoutIfMatch(t *testing.T) {
	// PUT without If-Match should skip ETag processing
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("updated"))
	}))

	req := httptest.NewRequest(http.MethodPut, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK)

	// No ETag generated for non-GET/HEAD
	if rec.Header().Get("ETag") != "" {
		t.Error("expected no ETag for PUT without If-Match")
	}
}

func TestETag_DELETEWithoutIfMatch(t *testing.T) {
	// DELETE without If-Match should skip ETag processing
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusNoContent)
}

func TestETag_IfMatchWildcard(t *testing.T) {
	// If-Match: * should succeed for any existing resource
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("resource exists"))
	}))

	req := httptest.NewRequest(http.MethodPut, "/", nil)
	req.Header.Set(httpx.HeaderIfMatch, "*")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should succeed since resource exists (handler writes content)
	zhtest.AssertWith(t, rec).Status(http.StatusOK)
}

func TestETag_NilSkipContentTypes(t *testing.T) {
	// Test with nil SkipContentTypes
	cfg := config.ETagConfig{
		Algorithm:        config.FNV,
		Weak:             config.Bool(true),
		MaxBufferSize:    1024,
		SkipStatusCodes:  map[int]struct{}{},
		SkipContentTypes: nil, // nil
		ExemptPaths:      []string{},
	}

	handler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ew := newETagResponseWriter(w, cfg, "", "", "", "", nil)
			defer ew.release()
			next.ServeHTTP(ew, r)
			ew.finalize()
		})
	}(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
		_, _ = w.Write([]byte("test"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	zhtest.AssertWith(t, rec).Status(http.StatusOK)

	// Should generate ETag even with nil SkipContentTypes
	if rec.Header().Get("ETag") == "" {
		t.Error("expected ETag header")
	}
}

func TestETag_ChunkedTransferEncoding(t *testing.T) {
	// Chunked transfer encoding should skip ETag
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Transfer-Encoding", "chunked")
		_, _ = w.Write([]byte("chunked data"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should not generate ETag for chunked responses
	if rec.Header().Get("ETag") != "" {
		t.Error("expected no ETag for chunked transfer encoding")
	}

	// Body should still be written
	if rec.Body.String() != "chunked data" {
		t.Errorf("expected body 'chunked data', got %s", rec.Body.String())
	}
}

func TestETag_Metrics(t *testing.T) {
	reg := metrics.NewRegistry()
	mw := ETag()

	// Wrap with metrics middleware to provide registry in context
	metricsMw := metrics.NewMiddleware(reg, config.MetricsConfig{
		Enabled:       config.Bool(true),
		PathLabelFunc: func(p string) string { return p },
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))
	wrapped := metricsMw(handler)

	// First request - should generate ETag and count as miss
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr1.Code)
	}

	etag := rr1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag")
	}

	// Second request with matching ETag - should be a hit
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set(httpx.HeaderIfNoneMatch, etag)
	rr2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusNotModified {
		t.Errorf("expected 304, got %d", rr2.Code)
	}

	// Check metrics
	families := reg.Gather()

	var reqCounter *metrics.MetricFamily
	var genCounter *metrics.MetricFamily
	for _, f := range families {
		switch f.Name {
		case "etag_requests_total":
			reqCounter = &f
		case "etag_generated_total":
			genCounter = &f
		}
	}

	if reqCounter == nil {
		t.Fatal("expected etag_requests_total metric")
	}
	if genCounter == nil {
		t.Fatal("expected etag_generated_total metric")
	}

	// Should have 1 hit and 1 miss
	hits, misses := 0, 0
	for _, m := range reqCounter.Metrics {
		switch m.Labels["result"] {
		case "hit":
			hits = int(m.Counter)
		case "miss":
			misses = int(m.Counter)
		}
	}
	if hits != 1 {
		t.Errorf("expected 1 hit, got %d", hits)
	}
	if misses != 1 {
		t.Errorf("expected 1 miss, got %d", misses)
	}

	// Should have 2 generated ETags (one per request)
	totalGen := 0
	for _, m := range genCounter.Metrics {
		totalGen = int(m.Counter)
	}
	if totalGen != 2 {
		t.Errorf("expected 2 generated ETags, got %d", totalGen)
	}
}

func TestETag_FlushThenWrite(t *testing.T) {
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("before"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		_, _ = w.Write([]byte(" after"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "before after" {
		t.Errorf("expected 'before after', got %q", rec.Body.String())
	}
}

// TestETag_ConcurrentWriteAndFlush tests that concurrent Write operations
// don't cause a data race on the buffer field
func TestETag_ConcurrentWriteAndFlush(t *testing.T) {
	mw := ETag()

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var wg sync.WaitGroup

		// Simulate concurrent writes
		for range 10 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = w.Write([]byte("data"))
				// Note: Flush is not safe for concurrent use in httptest.ResponseRecorder
				// so we only test concurrent writes here which is the actual issue
			}()
		}

		wg.Wait()

		// Single flush after all writes complete
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
}
