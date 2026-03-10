# zerohttp Issue Tracking

This document tracks known issues identified during code review. Items are organized by priority and should be checked off as they are resolved.

## Issue Checklist

### Medium Priority
- [x] **MD-1**: Rate Limit Middleware Uses Exact Match Instead of pathMatches()
- [x] **MD-2**: Request Logger Uses Exact Match Instead of pathMatches()
- [x] **MD-3**: Compress Middleware encoderGzip/encoderDeflate Return nil on Error
- [x] **MD-4**: Reverse Proxy Health Check Goroutine Leak
- [x] **MD-5**: FileHeader.ReadAll Doesn't Limit Size

### Low Priority
- [x] **LW-1**: HMAC Auth Middleware Modifies Request Header
- [x] **LW-2**: ETag Middleware Potential Data Race on buffer Field
- [x] **LW-3**: Rate Limit Store Inconsistent Lock Ordering
- [x] **LW-4**: Circuit Breaker GetState Double-Locking
- [x] **LW-5**: CSRF Token Generator Error Not Logged
- [x] **LW-6**: Compress Response Writer WriteHeader Called Multiple Times

---

## Medium Priority

### MD-1: Rate Limit Middleware Uses Exact Match Instead of pathMatches()
- **Location:** `middleware/rate_limit.go:55-60`
- **Issue:** The rate limit middleware uses exact string comparison for exempt paths, while other middleware (BasicAuth, CORS, CSRF, SecurityHeaders, etc.) use `pathMatches()`. This creates inconsistency - wildcard patterns won't work in rate limit exempt paths.
- **Current Code:**
  ```go
  for _, exemptPath := range c.ExemptPaths {
      if r.URL.Path == exemptPath {  // Uses exact match
          next.ServeHTTP(w, r)
          return
      }
  }
  ```
- **Fix:** Change to use `pathMatches()` for consistency:
  ```go
  if pathMatches(r.URL.Path, exemptPath) {
  ```

### MD-2: Request Logger Uses Exact Match Instead of pathMatches()
- **Location:** `middleware/request_logger.go:29-34`
- **Issue:** Same as MD-1 - inconsistent with other middleware that use `pathMatches()`.
- **Current Code:**
  ```go
  for _, exemptPath := range c.ExemptPaths {
      if r.URL.Path == exemptPath {  // Uses exact match
          next.ServeHTTP(w, r)
          return
      }
  }
  ```
- **Fix:** Change to use `pathMatches()` for consistency.

### MD-3: Compress Middleware encoderGzip/encoderDeflate Return nil on Error
- **Location:** `middleware/compress.go:340-353`
- **Issue:** These functions can return `nil` when an invalid compression level is provided. The `selectEncoder()` method at line 180-192 doesn't check for nil before using the encoder, which will cause a panic.
- **Current Code:**
  ```go
  func encoderGzip(w io.Writer, level int) io.Writer {
      gw, err := gzip.NewWriterLevel(w, level)
      if err != nil {
          return nil
      }
      return gw
  }
  ```
- **Fix:** Add nil check in `selectEncoder()` or return a safe fallback encoder.

### MD-4: Reverse Proxy Health Check Goroutine Leak
- **Location:** `middleware/reverse_proxy.go:75-79`
- **Issue:** The health check goroutine is started but never explicitly stopped. While there is a `cancelFunc`, there's no shutdown hook to call it. This could lead to goroutine leaks when the middleware is recreated.
- **Current Code:**
  ```go
  if cfg.HealthCheckInterval > 0 {
      ctx, cancel := context.WithCancel(context.Background())
      rp.cancelFunc = cancel
      go rp.healthCheckLoop(ctx)
  }
  ```
- **Fix:** Provide a shutdown mechanism or document the lifecycle expectations.

### MD-5: FileHeader.ReadAll Doesn't Limit Size
- **Location:** `bind.go:85-96`
- **Issue:** `ReadAll` reads the entire file into memory without any size limit. This could cause OOM if a large file is uploaded.
- **Current Code:**
  ```go
  func (fh *FileHeader) ReadAll() (data []byte, err error) {
      file, err := fh.Open()
      if err != nil {
          return nil, err
      }
      defer func() {
          if cerr := file.Close(); cerr != nil && err == nil {
              err = cerr
          }
      }()
      return io.ReadAll(file)
  }
  ```
- **Fix:** Add a max size limit parameter or use `io.LimitReader`.

---

## Low Priority

### LW-1: HMAC Auth Middleware Modifies Request Header
- **Location:** `middleware/hmac_auth.go:457`
- **Issue:** The `parsePresignedURLParams()` function modifies the request header as a side effect. While documented, it modifies the original request which might affect downstream handlers.
- **Current Code:**
  ```go
  // For presigned URLs, set the X-Timestamp header from the credential timestamp
  // so that required header checks pass
  r.Header.Set("X-Timestamp", ts.Format(time.RFC3339))
  ```

### LW-2: ETag Middleware Potential Data Race on buffer Field
- **Location:** `middleware/etag.go:46-94`
- **Issue:** The `etagResponseWriter` has a `buffer` field that is accessed in `Write()` and `Flush()`. While the `sync.Pool` usage for buffers is correct, there's potential for race conditions if `Flush()` is called concurrently with `Write()`.

### LW-3: Rate Limit Store Inconsistent Lock Ordering
- **Location:** `middleware/rate_limit_store.go`
- **Issue:** The `checkTokenBucket()` function holds the store lock while acquiring the entry lock. While safe currently, if the pattern changes in the future, it could lead to deadlocks.
- **Current Code:**
  ```go
  func (s *InMemoryStore) checkTokenBucket(key string, now time.Time) (bool, int, time.Time) {
      s.mu.Lock()
      defer s.mu.Unlock()
      // ...
      entry.mutex.Lock()
      defer entry.mutex.Unlock()
      // ...
  }
  ```

### LW-4: Circuit Breaker GetState Double-Locking
- **Location:** `middleware/circuit_breaker.go:197-208`
- **Issue:** This function acquires two read locks. While technically safe (multiple read locks can be held), it's inefficient.
- **Current Code:**
  ```go
  func (cbm *circuitBreakerMiddleware) GetState(key string) CircuitState {
      cbm.mu.RLock()
      defer cbm.mu.RUnlock()

      if c, exists := cbm.circuits[key]; exists {
          c.mu.RLock()
          defer c.mu.RUnlock()
          return c.state
      }
      return StateClosed
  }
  ```

### LW-5: CSRF Token Generator Error Not Logged
- **Location:** `middleware/csrf.go:114-120`
- **Issue:** The error from `tokenGenerator` is not logged, only counted as a metric. This could make debugging difficult.
- **Current Code:**
  ```go
  token, err = tokenGenerator(hmacKey)
  if err != nil {
      // Fail closed: reject request if we can't generate a token
      reg.Counter("csrf_rejected_total", "reason").WithLabelValues("token_generation_failed").Inc()
      errorHandler(w, r)
      return
  }
  ```

### LW-6: Compress Response Writer WriteHeader Called Multiple Times
- **Location:** `middleware/compress.go:240-262`
- **Issue:** If `wroteHeader` is true, it calls `ResponseWriter.WriteHeader(code)` directly. This deviates from standard library behavior where subsequent calls are ignored.
- **Current Code:**
  ```go
  func (cw *compressResponseWriter) WriteHeader(code int) {
      if cw.wroteHeader {
          cw.ResponseWriter.WriteHeader(code)
          return
      }
      // ...
  }
  ```

---

## Issue Reference Format

When fixing an issue, reference it in the commit message:
```
fix: use pathMatches for rate limit exempt paths

The rate limit middleware now uses pathMatches() instead of exact
string comparison for consistency with other middleware.

Fixes: MD-1
```

## Testing Requirements

Before marking an issue as fixed:
1. Add/update unit tests covering the fix
2. Run full test suite: `go test ./...`
3. Check for race conditions: `go test -race ./...`
4. Verify examples still work: `cd examples/xxx && go run .`
