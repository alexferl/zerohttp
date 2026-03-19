package middleware

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/internal/rwutil"
	"github.com/alexferl/zerohttp/metrics"
)

var (
	// etagBufferPool is used to pool response buffers for ETag generation
	etagBufferPool = sync.Pool{
		New: func() any {
			return new(bytes.Buffer)
		},
	}

	// hashPool pools hash instances to reduce allocations
	fnvHashPool = sync.Pool{
		New: func() any {
			return fnv.New64a()
		},
	}

	md5HashPool = sync.Pool{
		New: func() any {
			return md5.New()
		},
	}
)

// etagResponseWriter wraps http.ResponseWriter to capture response body for ETag generation
type etagResponseWriter struct {
	*rwutil.ResponseBuffer
	config          config.ETagConfig
	hash            hash.Hash
	written         int64
	skipETag        bool
	ifNoneMatch     string
	ifMatch         string
	ifRange         string
	rangeHeader     string
	contentEncoding string
	reg             metrics.Registry
	etagGenerated   bool
	finalized       bool
	mu              sync.Mutex // protects buffer and state fields
}

// newETagResponseWriter creates a new etagResponseWriter
func newETagResponseWriter(w http.ResponseWriter, cfg config.ETagConfig, ifNoneMatch, ifMatch, ifRange, rangeHeader string, reg metrics.Registry) *etagResponseWriter {
	return &etagResponseWriter{
		ResponseBuffer: rwutil.NewResponseBuffer(w, cfg.MaxBufferSize),
		config:         cfg,
		ifNoneMatch:    ifNoneMatch,
		ifMatch:        ifMatch,
		ifRange:        ifRange,
		rangeHeader:    rangeHeader,
		reg:            reg,
	}
}

// release returns the hash to its pool
func (ew *etagResponseWriter) release() {
	if ew.hash != nil {
		ew.hash.Reset()
		switch ew.config.Algorithm {
		case config.FNV:
			fnvHashPool.Put(ew.hash)
		case config.MD5:
			md5HashPool.Put(ew.hash)
		}
	}
}

// writeHeaderLocked captures the status code and checks if ETag generation should be skipped.
// Caller must hold ew.mu.
func (ew *etagResponseWriter) writeHeaderLocked(status int) {
	if ew.HasWritten {
		return
	}
	ew.HasWritten = true
	ew.Status = status

	if _, skip := ew.config.SkipStatusCodes[status]; skip {
		ew.skipETag = true
	}

	if ew.ResponseWriter.Header().Get(httpx.HeaderETag) != "" {
		ew.skipETag = true
	}

	cacheControl := ew.ResponseWriter.Header().Get(httpx.HeaderCacheControl)
	if strings.Contains(cacheControl, httpx.CacheControlNoStore) {
		ew.skipETag = true
	}

	contentType := ew.ResponseWriter.Header().Get(httpx.HeaderContentType)
	if contentType != "" {
		// Strip charset suffix
		if idx := strings.Index(contentType, ";"); idx != -1 {
			contentType = strings.TrimSpace(contentType[:idx])
		}
		if _, skip := ew.config.SkipContentTypes[contentType]; skip {
			ew.skipETag = true
		}
	}

	if strings.Contains(ew.ResponseWriter.Header().Get(httpx.HeaderTransferEncoding), httpx.TransferEncodingChunked) {
		ew.skipETag = true
	}

	ew.contentEncoding = ew.ResponseWriter.Header().Get(httpx.HeaderContentEncoding)

	if ew.skipETag && !ew.HeaderWritten {
		ew.ResponseWriter.WriteHeader(status)
		ew.HeaderWritten = true
	}
}

// WriteHeader captures the status code and checks if ETag generation should be skipped
func (ew *etagResponseWriter) WriteHeader(status int) {
	ew.mu.Lock()
	defer ew.mu.Unlock()
	ew.writeHeaderLocked(status)
}

// writeLocked captures the response body for ETag generation.
// Caller must hold ew.mu.
func (ew *etagResponseWriter) writeLocked(p []byte) (int, error) {
	n := len(p)
	ew.written += int64(n)

	if ew.skipETag {
		return ew.ResponseWriter.Write(p)
	}

	if ew.written > ew.config.MaxBufferSize {
		// Flush buffered content first if any
		ew.skipETag = true
		if !ew.HeaderWritten {
			ew.ResponseWriter.WriteHeader(ew.Status)
			ew.HeaderWritten = true
		}
		if ew.Buf.Len() > 0 {
			_, _ = ew.ResponseWriter.Write(ew.Buf.Bytes())
			ew.Buf.Reset()
		}
		return ew.ResponseWriter.Write(p)
	}

	return ew.Buf.Write(p)
}

// Write captures the response body for ETag generation
func (ew *etagResponseWriter) Write(p []byte) (int, error) {
	ew.mu.Lock()

	if !ew.HasWritten {
		ew.writeHeaderLocked(http.StatusOK)
	}

	n, err := ew.writeLocked(p)
	ew.mu.Unlock()
	return n, err
}

// generateETag generates the ETag from the buffered content
// Content-encoding aware: includes encoding in the hash to prevent cache poisoning
func (ew *etagResponseWriter) generateETag() string {
	if ew.Buf.Len() == 0 {
		return ""
	}

	if ew.config.Algorithm == config.FNV {
		ew.hash = fnvHashPool.Get().(hash.Hash64)
	} else {
		ew.hash = md5HashPool.Get().(hash.Hash)
	}

	ew.hash.Reset()
	ew.hash.Write(ew.Buf.Bytes())

	// Include content encoding in the hash for content-encoding aware ETags
	// This prevents cache poisoning where the same ETag is returned for
	// both compressed and uncompressed content
	if ew.contentEncoding != "" {
		ew.hash.Write([]byte(ew.contentEncoding))
	}

	var hashStr string
	if ew.config.Algorithm == config.FNV {
		hashStr = strconv.FormatUint(ew.hash.(hash.Hash64).Sum64(), 16)
	} else {
		hashStr = hex.EncodeToString(ew.hash.Sum(nil))
	}

	if ew.config.Weak != nil && *ew.config.Weak {
		return `W/"` + hashStr + `"`
	}
	return `"` + hashStr + `"`
}

// shouldReturn304 checks if we should return a 304 Not Modified response
func (ew *etagResponseWriter) shouldReturn304(etag string) bool {
	if ew.ifNoneMatch == "" {
		return false
	}
	// 304 only applies to GET and HEAD
	return etagMatches(ew.ifNoneMatch, etag)
}

// shouldReturn412 checks if we should return a 412 Precondition Failed response
func (ew *etagResponseWriter) shouldReturn412(etag string) bool {
	if ew.ifMatch == "" {
		return false
	}
	// If-Match: * means any existing resource matches
	if ew.ifMatch == "*" {
		return false
	}
	return !etagMatches(ew.ifMatch, etag)
}

// shouldServeRange checks if we should serve a 206 Partial Content response
// based on If-Range validation
func (ew *etagResponseWriter) shouldServeRange(etag string) bool {
	if ew.ifRange == "" || ew.rangeHeader == "" {
		return false
	}

	// If-Range can be an ETag or a date
	// If it's a date, it won't match an ETag format
	if strings.HasPrefix(ew.ifRange, "W/") || strings.HasPrefix(ew.ifRange, `"`) {
		// It's an ETag, validate it matches
		return etagMatches(ew.ifRange, etag)
	}

	// Assume it's a date - we don't validate dates here, just return false
	// to fall back to full content
	return false
}

// Flush implements http.Flusher
func (ew *etagResponseWriter) Flush() {
	ew.mu.Lock()
	// If we have buffered data and haven't written to the response yet,
	// we need to flush the ETag processing first
	if ew.Buf.Len() > 0 && !ew.skipETag && !ew.finalized {
		ew.finalizeLocked()
		ew.skipETag = true // switch to pass-through for any writes after flush
	} else if !ew.HasWritten {
		// No data written yet but flush called - commit headers without ETag
		// to avoid implicit 200 on subsequent flush
		ew.HasWritten = true
		ew.skipETag = true
		ew.ResponseWriter.WriteHeader(ew.Status)
		ew.HeaderWritten = true
	} else if ew.HasWritten && !ew.HeaderWritten {
		// WriteHeader was called but no body written yet; Flush would implicitly
		// commit headers in production net/http, so we need to do it explicitly
		ew.skipETag = true
		ew.ResponseWriter.WriteHeader(ew.Status)
		ew.HeaderWritten = true
	}
	ew.mu.Unlock()

	if f, ok := ew.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// finalizeLocked processes the buffered response, generates ETag, and writes the response.
// Caller must hold ew.mu.
func (ew *etagResponseWriter) finalizeLocked() {
	if ew.skipETag || ew.finalized {
		return
	}
	ew.finalized = true

	etag := ew.generateETag()

	if etag != "" {
		ew.etagGenerated = true

		if ew.ifMatch != "" && ew.shouldReturn412(etag) {
			ew.ResponseWriter.Header().Set(httpx.HeaderETag, etag)
			if !ew.HeaderWritten {
				ew.ResponseWriter.WriteHeader(http.StatusPreconditionFailed)
				ew.HeaderWritten = true
			}
			ew.recordMetrics(http.StatusPreconditionFailed)
			return
		}

		if ew.shouldReturn304(etag) {
			ew.ResponseWriter.Header().Set(httpx.HeaderETag, etag)
			if !ew.HeaderWritten {
				ew.ResponseWriter.WriteHeader(http.StatusNotModified)
				ew.HeaderWritten = true
			}
			ew.recordMetrics(http.StatusNotModified)
			return
		}

		ew.ResponseWriter.Header().Set(httpx.HeaderETag, etag)

		// Check If-Range for range request handling
		// Note: Range requests with If-Range require special handling
		// If the ETag matches, we serve 206 with the range
		// If not, we fall through to serve 200 with full content
		if ew.shouldServeRange(etag) {
			// The range request is valid, but we need to serve partial content
			// Since we're buffering the full response, we can serve ranges
			ew.serveRange()
			ew.recordMetrics(http.StatusPartialContent)
			return
		}
	}

	if !ew.HeaderWritten {
		ew.ResponseWriter.WriteHeader(ew.Status)
		ew.HeaderWritten = true
	}
	if ew.Buf.Len() > 0 {
		_, _ = ew.ResponseWriter.Write(ew.Buf.Bytes())
	}
	ew.recordMetrics(ew.Status)
}

// finalize processes the buffered response, generates ETag, and writes the response
func (ew *etagResponseWriter) finalize() {
	ew.mu.Lock()
	defer ew.mu.Unlock()
	ew.finalizeLocked()
}

// recordMetrics records ETag metrics
func (ew *etagResponseWriter) recordMetrics(status int) {
	if ew.reg == nil {
		return
	}

	if ew.etagGenerated {
		ew.reg.Counter("etag_generated_total").Inc()
	}

	result := "miss"
	if status == http.StatusNotModified {
		result = "hit"
	}
	ew.reg.Counter("etag_requests_total", "result").WithLabelValues(result).Inc()
}

// serveRange serves a partial content response based on the Range header
// This is a basic implementation that supports single byte ranges
func (ew *etagResponseWriter) serveRange() {
	content := ew.Buf.Bytes()
	contentLength := len(content)

	// Parse Range header (e.g., "bytes=0-1023" or "bytes=1024-")
	rangeValue := strings.TrimPrefix(ew.rangeHeader, "bytes=")
	parts := strings.Split(rangeValue, "-")
	if len(parts) != 2 {
		// Invalid range, fall back to full content
		ew.ResponseWriter.WriteHeader(ew.Status)
		_, _ = ew.ResponseWriter.Write(content)
		return
	}

	start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	end := contentLength - 1
	if parts[1] != "" {
		var err2 error
		end, err2 = strconv.Atoi(strings.TrimSpace(parts[1]))
		if err2 != nil {
			// Invalid range, fall back to full content
			ew.ResponseWriter.WriteHeader(ew.Status)
			_, _ = ew.ResponseWriter.Write(content)
			return
		}
	}

	if err1 != nil || start < 0 || start > end || end >= contentLength {
		// Invalid range, fall back to full content
		ew.ResponseWriter.WriteHeader(ew.Status)
		_, _ = ew.ResponseWriter.Write(content)
		return
	}

	ew.ResponseWriter.Header().Set(httpx.HeaderContentRange, fmt.Sprintf("bytes %d-%d/%d", start, end, contentLength))
	ew.ResponseWriter.Header().Set(httpx.HeaderContentLength, strconv.Itoa(end-start+1))
	ew.ResponseWriter.WriteHeader(http.StatusPartialContent)
	_, _ = ew.ResponseWriter.Write(content[start : end+1])
}

// ETag creates ETag middleware with custom configuration
func ETag(cfg ...config.ETagConfig) func(http.Handler) http.Handler {
	c := config.DefaultETagConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	validatePathConfig(c.ExcludedPaths, c.IncludedPaths, "ETag")

	if c.Algorithm != config.FNV && c.Algorithm != config.MD5 {
		c.Algorithm = config.DefaultETagConfig.Algorithm
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reg := metrics.SafeRegistry(metrics.GetRegistry(r.Context()))

			if !shouldProcessMiddleware(r.URL.Path, c.IncludedPaths, c.ExcludedPaths) {
				next.ServeHTTP(w, r)
				return
			}

			if c.ExcludedFunc != nil && c.ExcludedFunc(r) {
				next.ServeHTTP(w, r)
				return
			}

			ifNoneMatch := r.Header.Get(httpx.HeaderIfNoneMatch)
			ifMatch := r.Header.Get(httpx.HeaderIfMatch)
			ifRange := r.Header.Get(httpx.HeaderIfRange)
			rangeHeader := r.Header.Get(httpx.HeaderRange)

			// Only generate ETags for GET and HEAD requests (for caching)
			// But still check If-Match for state-changing methods
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				// For PUT/PATCH/DELETE, check If-Match but don't generate/cache ETags
				if ifMatch != "" {
					// Pass through with a wrapper that checks If-Match
					ew := newETagResponseWriter(w, c, "", ifMatch, "", "", reg)
					defer ew.release()
					next.ServeHTTP(ew, r)
					ew.finalize()
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			ew := newETagResponseWriter(w, c, ifNoneMatch, ifMatch, ifRange, rangeHeader, reg)
			defer ew.release()

			next.ServeHTTP(ew, r)

			ew.finalize()
		})
	}
}

// GenerateFileETag generates an ETag for a file based on its modification time and size.
// This is much more efficient than hashing the file content, especially for large files.
// The format is: W/"mtime-size" (weak ETag) or "mtime-size" (strong ETag)
// Example: W/"1709999999-1024"
//
// Usage:
//
//	file, _ := os.Open("largefile.zip")
//	stat, _ := file.Stat()
//	etag := middleware.GenerateFileETag(stat, true) // weak ETag
//	w.Header().Set(httpx.HeaderETag, etag)
func GenerateFileETag(modTime int64, size int64, weak bool) string {
	tag := strconv.FormatInt(modTime, 10) + "-" + strconv.FormatInt(size, 10)
	if weak {
		return `W/"` + tag + `"`
	}
	return `"` + tag + `"`
}

// GenerateFileETagFromInfo generates an ETag from fs.FileInfo or os.FileInfo.
// This helper handles both interface types properly.
//
// Usage:
//
//	file, _ := os.Open("largefile.zip")
//	stat, _ := file.Stat()
//	etag := middleware.GenerateFileETagFromInfo(stat, true)
//	w.Header().Set(httpx.HeaderETag, etag)
func GenerateFileETagFromInfo(info interface {
	ModTime() time.Time
	Size() int64
}, weak bool,
) string {
	return GenerateFileETag(info.ModTime().Unix(), info.Size(), weak)
}

// ParseETag extracts the hash value from an ETag, handling weak ETags
// Returns the hash value and a boolean indicating if it was a weak ETag
// Example: ParseETag(`W/"abc123"`) returns ("abc123", true)
// Example: ParseETag(`"abc123"`) returns ("abc123", false)
func ParseETag(etag string) (string, bool) {
	if strings.HasPrefix(etag, `W/"`) && strings.HasSuffix(etag, `"`) {
		return etag[3 : len(etag)-1], true
	}
	if strings.HasPrefix(etag, `"`) && strings.HasSuffix(etag, `"`) {
		return etag[1 : len(etag)-1], false
	}
	return etag, false
}

// ServeContentWithETag serves content with automatic ETag support.
// It handles If-None-Match and If-Range headers properly.
// Similar to http.ServeContent but with our ETag generation logic.
func ServeContentWithETag(w http.ResponseWriter, r *http.Request, modTime int64, content io.ReadSeeker) {
	if content == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var size int64
	if seeker, ok := content.(io.Seeker); ok {
		end, err := seeker.Seek(0, io.SeekEnd)
		if err == nil {
			size = end
			_, _ = seeker.Seek(0, io.SeekStart)
		}
	}

	etag := GenerateFileETag(modTime, size, true)

	if ifNoneMatch := r.Header.Get(httpx.HeaderIfNoneMatch); ifNoneMatch != "" {
		if etagMatches(ifNoneMatch, etag) {
			w.Header().Set(httpx.HeaderETag, etag)
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	w.Header().Set(httpx.HeaderETag, etag)

	// Serve content using http.ServeContent which handles Range requests
	// We need to convert int64 modTime to time.Time
	http.ServeContent(w, r, "", time.Unix(modTime, 0), content)
}

// etagMatches checks if the provided ETag matches any in the If-None-Match header
func etagMatches(ifNoneMatch, etag string) bool {
	if ifNoneMatch == "*" {
		return true
	}

	for _, et := range strings.Split(ifNoneMatch, ",") {
		et = strings.TrimSpace(et)
		// Compare weak ETags ignoring the W/ prefix
		if strings.HasPrefix(et, "W/") && strings.HasPrefix(etag, "W/") {
			if et == etag {
				return true
			}
		} else if strings.HasPrefix(et, "W/") {
			// Client has weak, we have strong - compare values
			if et[2:] == etag {
				return true
			}
		} else if strings.HasPrefix(etag, "W/") {
			// Client has strong, we have weak - compare values
			if et == etag[2:] {
				return true
			}
		} else {
			if et == etag {
				return true
			}
		}
	}
	return false
}
