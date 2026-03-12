package middleware

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/metrics"
)

// Cache creates HTTP caching middleware with automatic ETag generation,
// conditional requests, and Cache-Control handling.
func Cache(cfg ...config.CacheConfig) func(http.Handler) http.Handler {
	c := config.DefaultCacheConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}

	if c.CacheControl == "" {
		c.CacheControl = config.DefaultCacheConfig.CacheControl
	}
	if c.DefaultTTL == 0 {
		c.DefaultTTL = config.DefaultCacheConfig.DefaultTTL
	}
	if c.MaxBodySize == 0 {
		c.MaxBodySize = config.DefaultCacheConfig.MaxBodySize
	}
	if c.Vary == nil {
		c.Vary = config.DefaultCacheConfig.Vary
	}
	if c.StatusCodes == nil {
		c.StatusCodes = config.DefaultCacheConfig.StatusCodes
	}

	statusCodeMap := make(map[int]bool)
	for _, code := range c.StatusCodes {
		statusCodeMap[code] = true
	}

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

			for _, exemptPath := range c.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			if r.Header.Get("Cache-Control") == "no-cache" || r.Header.Get("Cache-Control") == "no-store" {
				next.ServeHTTP(w, r)
				return
			}

			key := generateCacheKey(r, c.Vary)

			ifNoneMatch := r.Header.Get("If-None-Match")
			ifModifiedSince := r.Header.Get("If-Modified-Since")

			record, found, err := store.Get(r.Context(), key)
			if err != nil {
				// Log error and continue to handler on cache fetch failure
				// (fail open - better to serve fresh content than error)
				log.GetGlobalLogger().Error("Cache store get failed", log.E(err), log.F("key", key))
			} else if found {
				reg.Counter("cache_requests_total", "result").WithLabelValues("hit").Inc()
				if ifNoneMatch != "" && ifNoneMatch == record.ETag {
					w.Header().Set("ETag", record.ETag)
					w.WriteHeader(http.StatusNotModified)
					return
				}

				if ifModifiedSince != "" {
					if parsedTime, err := http.ParseTime(ifModifiedSince); err == nil {
						if !record.LastModified.IsZero() && !record.LastModified.After(parsedTime) {
							w.Header().Set("Last-Modified", record.LastModified.UTC().Format(http.TimeFormat))
							w.WriteHeader(http.StatusNotModified)
							return
						}
					}
				}

				for k, v := range record.Headers {
					for _, val := range v {
						w.Header().Add(k, val)
					}
				}
				w.Header().Set("ETag", record.ETag)
				w.Header().Set("Cache-Control", c.CacheControl)
				w.WriteHeader(record.StatusCode)
				if r.Method != http.MethodHead {
					_, _ = w.Write(record.Body)
				}
				return
			}

			reg.Counter("cache_requests_total", "result").WithLabelValues("miss").Inc()

			recorder := &cacheResponseRecorder{
				ResponseWriter: w,
				maxBodySize:    c.MaxBodySize,
				statusCodeMap:  statusCodeMap,
			}

			next.ServeHTTP(recorder, r)

			if recorder.shouldCache {
				record := config.CacheRecord{
					StatusCode:   recorder.statusCode,
					Headers:      recorder.headers,
					Body:         recorder.body.Bytes(),
					LastModified: time.Now().UTC().Truncate(time.Second),
					VaryHeaders:  extractVaryHeaders(r, c.Vary),
				}

				if c.ETag {
					hash := sha256.Sum256(record.Body)
					record.ETag = fmt.Sprintf(`"%s"`, hex.EncodeToString(hash[:])[:16])
				}

				if err := store.Set(r.Context(), key, record, c.DefaultTTL); err != nil {
					// Log error but don't fail the request
					// (better to serve the response than fail because cache is unavailable)
					log.GetGlobalLogger().Error("Cache store set failed", log.E(err), log.F("key", key))
				}

				if c.ETag && record.ETag != "" {
					w.Header().Set("ETag", record.ETag)
				}
				if c.LastModified {
					w.Header().Set("Last-Modified", record.LastModified.UTC().Format(http.TimeFormat))
				}
				w.Header().Set("Cache-Control", c.CacheControl)
			}
		})
	}
}

// cacheResponseRecorder captures response data for caching.
type cacheResponseRecorder struct {
	http.ResponseWriter
	statusCode    int
	headers       map[string][]string
	body          bytes.Buffer
	maxBodySize   int64
	shouldCache   bool
	statusCodeMap map[int]bool
	written       bool
}

func (c *cacheResponseRecorder) WriteHeader(statusCode int) {
	if c.written {
		return
	}
	c.written = true
	c.statusCode = statusCode
	c.shouldCache = c.statusCodeMap[statusCode]

	if c.shouldCache {
		c.headers = make(map[string][]string)
		for k, v := range c.Header() {
			// Skip headers that shouldn't be cached
			if k == "Set-Cookie" || k == "Connection" || k == "Keep-Alive" {
				continue
			}
			c.headers[k] = v
		}
	}

	c.ResponseWriter.WriteHeader(statusCode)
}

func (c *cacheResponseRecorder) Write(p []byte) (int, error) {
	if !c.written {
		c.WriteHeader(http.StatusOK)
	}

	if c.shouldCache && int64(c.body.Len()+len(p)) <= c.maxBodySize {
		c.body.Write(p)
	} else if c.shouldCache {
		c.shouldCache = false
	}

	return c.ResponseWriter.Write(p)
}

func (c *cacheResponseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := c.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

func (c *cacheResponseRecorder) Flush() {
	if flusher, ok := c.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// generateCacheKey creates a cache key from request method, URL, and vary headers.
// GET and HEAD share the same cache key per RFC 7231.
func generateCacheKey(r *http.Request, vary []string) string {
	var parts []string
	// Treat HEAD as GET for cache key purposes
	method := r.Method
	if method == http.MethodHead {
		method = http.MethodGet
	}
	parts = append(parts, method)
	parts = append(parts, r.URL.Path)
	parts = append(parts, r.URL.RawQuery)

	for _, header := range vary {
		if value := r.Header.Get(header); value != "" {
			parts = append(parts, fmt.Sprintf("%s=%s", header, value))
		}
	}

	hash := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(hash[:])
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
