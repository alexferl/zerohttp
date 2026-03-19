package middleware

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/internal/rwutil"
	"github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/metrics"
)

// Cache creates HTTP caching middleware with automatic ETag generation,
// conditional requests, and Cache-Control handling.
func Cache(cfg ...config.CacheConfig) func(http.Handler) http.Handler {
	c := config.DefaultCacheConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	validatePathConfig(c.ExcludedPaths, c.IncludedPaths, "Cache")

	statusCodeMap := make(map[int]bool)
	for _, code := range c.StatusCodes {
		statusCodeMap[code] = true
	}

	// Determine cache status header name (empty string means disabled)
	cacheStatusHeader := config.StringOrDefault(c.CacheStatusHeader, httpx.HeaderXCache)

	var store CacheStore
	if c.Store != nil {
		store = c.Store
	} else {
		store = NewCacheMemoryStore(c.MaxEntries)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reg := metrics.SafeRegistry(metrics.GetRegistry(r.Context()))

			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				next.ServeHTTP(w, r)
				return
			}

			if !shouldProcessMiddleware(r.URL.Path, c.IncludedPaths, c.ExcludedPaths) {
				next.ServeHTTP(w, r)
				return
			}

			// Note: Per RFC 9111, no-cache means revalidate, not bypass. We treat it as
			// bypass for simplicity, which is a common implementation choice.
			if r.Header.Get(httpx.HeaderCacheControl) == httpx.CacheControlNoCache || r.Header.Get(httpx.HeaderCacheControl) == httpx.CacheControlNoStore {
				next.ServeHTTP(w, r)
				return
			}

			key := generateCacheKey(r, c.Vary)

			ifNoneMatch := r.Header.Get(httpx.HeaderIfNoneMatch)
			ifModifiedSince := r.Header.Get(httpx.HeaderIfModifiedSince)

			record, found, err := store.Get(r.Context(), key)
			if err != nil {
				// Log error and continue to handler on cache fetch failure
				// (fail open - better to serve fresh content than error)
				log.GetGlobalLogger().Error("Cache store get failed", log.E(err), log.F("key", key))
			} else if found {
				reg.Counter("cache_requests_total", "result").WithLabelValues("hit").Inc()
				if cacheStatusHeader != "" {
					w.Header().Set(cacheStatusHeader, httpx.XCacheHit)
				}
				// Only return 304 if If-None-Match was actually provided
				if ifNoneMatch != "" && etagMatches(ifNoneMatch, record.ETag) {
					if record.ETag != "" {
						w.Header().Set(httpx.HeaderETag, record.ETag)
					}
					w.WriteHeader(http.StatusNotModified)
					return
				}

				if ifModifiedSince != "" {
					if parsedTime, err := http.ParseTime(ifModifiedSince); err == nil {
						if !record.LastModified.IsZero() && !record.LastModified.After(parsedTime) {
							w.Header().Set(httpx.HeaderLastModified, record.LastModified.UTC().Format(http.TimeFormat))
							w.WriteHeader(http.StatusNotModified)
							return
						}
					}
				}

				// Replay headers from cache, skipping keys already set by other middleware
				for k, v := range record.Headers {
					// Skip if this header key is already present (from security middleware, etc.)
					if w.Header().Get(k) != "" {
						continue
					}
					for _, val := range v {
						w.Header().Add(k, val)
					}
				}
				if record.ETag != "" {
					w.Header().Set(httpx.HeaderETag, record.ETag)
				}
				w.Header().Set(httpx.HeaderCacheControl, c.CacheControl)
				w.WriteHeader(record.StatusCode)
				if r.Method != http.MethodHead {
					_, _ = w.Write(record.Body)
				}
				return
			}

			reg.Counter("cache_requests_total", "result").WithLabelValues("miss").Inc()
			if cacheStatusHeader != "" {
				w.Header().Set(cacheStatusHeader, httpx.XCacheMiss)
			}

			recorder := &cacheResponseRecorder{
				ResponseBuffer: rwutil.NewResponseBuffer(w, c.MaxBodySize),
				statusCodeMap:  statusCodeMap,
			}

			next.ServeHTTP(recorder, r)

			var etag string
			var lastModified time.Time

			if recorder.shouldCache {
				lastModified = time.Now().UTC().Truncate(time.Second)
				record := config.CacheRecord{
					StatusCode:   recorder.Status,
					Headers:      recorder.headers,
					Body:         recorder.Buf.Bytes(),
					LastModified: lastModified,
					VaryHeaders:  extractVaryHeaders(r, c.Vary),
				}

				if c.ETag {
					hash := sha256.Sum256(record.Body)
					etag = fmt.Sprintf(`"%s"`, hex.EncodeToString(hash[:]))
					record.ETag = etag
				}

				if err := store.Set(r.Context(), key, record, c.DefaultTTL); err != nil {
					// Log error but don't fail the request
					// (better to serve the response than fail because cache is unavailable)
					log.GetGlobalLogger().Error("Cache store set failed", log.E(err), log.F("key", key))
				}
			}

			// Finalize writes the response with proper headers
			recorder.Finalize(etag, c.CacheControl, c.ETag, c.LastModified, lastModified)
		})
	}
}

// cacheResponseRecorder captures response data for caching.
type cacheResponseRecorder struct {
	*rwutil.ResponseBuffer
	headers       map[string][]string
	shouldCache   bool
	statusCodeMap map[int]bool
	hijacked      bool
}

// WriteHeader captures the status code and determines if response should be cached.
func (c *cacheResponseRecorder) WriteHeader(statusCode int) {
	if c.HasWritten || c.hijacked {
		return
	}
	c.ResponseBuffer.WriteHeader(statusCode)
	c.shouldCache = c.statusCodeMap[statusCode]

	if c.shouldCache {
		c.headers = make(map[string][]string)
		for k, v := range c.Header() {
			// Skip hop-by-hop and sensitive headers per RFC 9111
			switch k {
			case httpx.HeaderSetCookie, httpx.HeaderConnection, httpx.HeaderKeepAlive,
				httpx.HeaderAuthorization, httpx.HeaderProxyAuthenticate, httpx.HeaderProxyAuthorization,
				httpx.HeaderTE, httpx.HeaderTrailer, httpx.HeaderTransferEncoding, httpx.HeaderUpgrade:
				continue
			}
			copied := make([]string, len(v))
			copy(copied, v)
			c.headers[k] = copied
		}
		// Don't write header yet - defer until Finalize (we're caching)
	} else if !c.HeaderWritten {
		// Not caching - write through immediately to avoid status code loss
		c.ResponseWriter.WriteHeader(statusCode)
		c.HeaderWritten = true
	}
}

// Write captures the response body for caching or passes through for non-cacheable responses.
func (c *cacheResponseRecorder) Write(p []byte) (int, error) {
	if c.hijacked {
		return c.ResponseWriter.Write(p)
	}
	if !c.ShouldCache() {
		return c.ResponseWriter.Write(p)
	}
	return c.ResponseBuffer.Write(p)
}

// ShouldCache returns true if the response should be cached.
func (c *cacheResponseRecorder) ShouldCache() bool {
	return c.shouldCache && c.Buffering
}

// Finalize writes the buffered response to the underlying ResponseWriter.
// etag is the computed ETag to add (if enabled and non-empty).
func (c *cacheResponseRecorder) Finalize(etag, cacheControl string, enableETag, enableLastMod bool, lastModified time.Time) {
	if c.hijacked || !c.Buffering {
		return
	}

	// If we're not caching, the response has already been written through
	if !c.shouldCache {
		return
	}

	// Ensure we have a valid status code
	if !c.HasWritten {
		c.Status = http.StatusOK
	}

	// Set cache headers before writing
	if cacheControl != "" {
		c.ResponseWriter.Header().Set(httpx.HeaderCacheControl, cacheControl)
	}
	if enableETag && etag != "" {
		c.ResponseWriter.Header().Set(httpx.HeaderETag, etag)
	}
	if enableLastMod && !lastModified.IsZero() {
		c.ResponseWriter.Header().Set(httpx.HeaderLastModified, lastModified.UTC().Format(http.TimeFormat))
	}

	c.Commit()
}

func (c *cacheResponseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	c.hijacked = true
	return c.ResponseBuffer.Hijack()
}

func (c *cacheResponseRecorder) Flush() {
	if c.hijacked {
		if flusher, ok := c.ResponseWriter.(http.Flusher); ok {
			flusher.Flush()
		}
		return
	}

	c.shouldCache = false
	flusher, _ := c.ResponseWriter.(http.Flusher)
	c.FlushTo(flusher, nil)
}

// generateCacheKey creates a cache key from request method, URL, and vary headers.
// GET and HEAD share the same cache key per RFC 7231.
func generateCacheKey(r *http.Request, vary []string) string {
	var b strings.Builder
	b.Grow(len(r.URL.Path) + len(r.URL.RawQuery) + 32)
	// Treat HEAD as GET for cache key purposes
	if r.Method == http.MethodHead {
		b.WriteString(http.MethodGet)
	} else {
		b.WriteString(r.Method)
	}
	b.WriteByte('|')
	b.WriteString(r.URL.Path)
	b.WriteByte('|')
	b.WriteString(r.URL.RawQuery)
	for _, header := range vary {
		if value := r.Header.Get(header); value != "" {
			b.WriteByte('|')
			b.WriteString(header)
			b.WriteByte('=')
			b.WriteString(value)
		}
	}
	return b.String()
}

// extractVaryHeaders extracts vary header values from request.
func extractVaryHeaders(r *http.Request, vary []string) map[string]string {
	result := make(map[string]string)
	for _, header := range vary {
		if value := r.Header.Get(header); value != "" {
			result[header] = value
		}
	}
	return result
}
