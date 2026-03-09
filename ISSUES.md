# zerohttp Issue Tracking

This document tracks known issues identified during code review. Items are organized by priority and should be checked off as they are resolved.

## Critical Priority

### CR-1: Panic-Based Error Handling in Critical Paths
- **Location:** `router.go:589`, `router.go:599`, `middleware/rate_limit.go:174`, `middleware/circuit_breaker.go:109`
- **Issue:** Multiple locations use `panic()` for error conditions that could be triggered by external input
- **Impact:** Denial of service through intentional triggering of panics; resource exhaustion from recovery overhead
- **Fix:** Replace panic-based error handling with proper error returns and logging

### CR-2: CSRF Token Returns Empty String on Random Generation Failure
- **Location:** `middleware/csrf.go:157-169`
- **Issue:** If `rand.Read()` fails, `generateToken()` returns empty string, bypassing CSRF protection
- **Impact:** Complete bypass of CSRF protection when random generation fails
- **Fix:** Fail closed - reject the request or retry with exponential backoff

### CR-3: HMAC Secret Key Length Not Validated
- **Location:** `middleware/jwt_hs256.go:35-39`, `middleware/jwt_hs256.go:274-278`
- **Issue:** HS256 accepts any length secret key; short keys can be brute-forced
- **Impact:** Weak secret keys enable brute-force attacks against JWT tokens
- **Fix:** Add minimum key length validation (32 bytes for HS256, 48 for HS384, 64 for HS512)

### CR-4: Unbounded Memory Growth in Rate Limiter
- **Location:** `middleware/rate_limit.go` (sliding window implementation)
- **Issue:** Stores all request timestamps indefinitely per key: `slidingWindows := make(map[string][]time.Time)`
- **Impact:** OOM under high traffic with many unique keys (DoS vector)
- **Fix:** Implement automatic expiration/cleanup of old entries; add max keys limit

---

## High Priority

### HP-1: Ignored Errors in JSON Encoding
- **Location:** `router.go:56`, `router.go:69`, `router.go:81`
- **Issue:** Error response encoding failures silently ignored: `_ = json.NewEncoder(w).Encode(response)`
- **Impact:** Clients receive incomplete/no error information; hard to debug
- **Fix:** Log encoding errors even if response has already started

### HP-2: Race Condition in SSE Hub Broadcast
- **Location:** `sse.go:518-538`
- **Issue:** Connection could be closed between RUnlock and Send()
- **Impact:** Operations on closed connections; potential panic
- **Fix:** Add individual connection state checks or use channel-based communication

### HP-3: Potential Deadlock in Rate Limiter
- **Location:** `middleware/rate_limit.go:79-158`
- **Issue:** Nested locking on global mutex + per-bucket mutexes
- **Impact:** Deadlock under high concurrency with multiple keys
- **Fix:** Use sharded locks or lock-free data structures

### HP-4: JWT Validation Timing Side-Channel
- **Location:** `middleware/jwt_hs256.go:180-186`
- **Issue:** `time.Now().Unix() > int64(exp)` comparison not constant-time
- **Impact:** Timing attacks could reveal token expiration validity
- **Fix:** Use constant-time comparison for security-sensitive operations

---

## Medium Priority

### MP-1: Duplicate Response Writer Wrappers
- **Location:** `middleware/request_logger.go`, `middleware/circuit_breaker.go`, `middleware/tracing.go`, `middleware/compress.go`
- **Issue:** Each middleware defines nearly identical response writer wrappers
- **Impact:** Code duplication; maintenance burden
- **Fix:** Create a shared response writer wrapper package

### MP-2: Insecure Random for Request IDs
- **Location:** `config/request_id.go:35`
- **Issue:** Request IDs use `time.Now().UnixNano()` which is predictable
- **Impact:** Predictable request IDs can aid session fixation attacks
- **Fix:** Use crypto/rand for request ID generation

### MP-3: Missing Context Cancellation in Health Checks
- **Location:** `middleware/reverse_proxy.go:321-342`
- **Issue:** Health check HTTP client doesn't use request context
- **Impact:** Health checks can't be cancelled; hang indefinitely
- **Fix:** Pass context through health check operations

### MP-4: Missing Content Security Policy Headers
- **Location:** `config/security_headers.go`
- **Issue:** No CSP header support in security headers middleware
- **Impact:** Reduced XSS protection
- **Fix:** Add CSP header configuration

### MP-5: File Handle Leak Risk in Static File Serving
- **Location:** `router.go:435-446`
- **Issue:** File may not close if Stat() fails in certain edge cases
- **Impact:** File handle exhaustion
- **Fix:** Use defer for file closing or ensure all paths close the file

### MP-6: Reflection Overhead in Binding
- **Location:** `bind.go:132-339`
- **Issue:** Heavy reflection use without type information caching
- **Impact:** Performance degradation on high-throughput endpoints
- **Fix:** Cache reflection results using a type registry

### MP-7: Missing Minimum Secret Length for HMAC Auth
- **Location:** `middleware/hmac_auth.go`
- **Issue:** No validation that HMAC secrets meet minimum length requirements
- **Impact:** Weak secrets can be brute-forced
- **Fix:** Add configurable minimum secret length validation

---

## Low Priority

### LP-1: Inconsistent Configuration Patterns
- **Location:** Various config files
- **Issue:** Some configs use `*bool` for optional fields, others use `bool` with defaults
- **Impact:** Inconsistent API
- **Fix:** Standardize on single pattern

### LP-2: Missing Interface Compliance Checks
- **Location:** Various
- **Issue:** Some interfaces lack compile-time compliance checks
- **Impact:** Runtime errors instead of compile-time errors
- **Fix:** Add `var _ Interface = (*Implementation)(nil)` checks

### LP-3: Magic Numbers in CSRF
- **Location:** `middleware/csrf.go:229`
- **Issue:** `32 << 20` for multipart form size limit without constant
- **Impact:** Unclear intent
- **Fix:** Define named constant

### LP-4: Atomic Field Alignment on 32-bit Systems
- **Location:** `middleware/reverse_proxy.go`
- **Issue:** Atomic operations on potentially misaligned int64 fields
- **Impact:** Crashes on 32-bit architectures
- **Fix:** Ensure proper field ordering or use sync/atomic types

---

## Test Coverage Gaps

- Error paths in JWT validation (mostly tests success cases)
- Circuit breaker state transitions (complex state machine)
- Rate limiter under high concurrency (race conditions)
- SSE reconnection and replay edge cases
- Reverse proxy health check failure scenarios

---

## Fixed Issues

*Check off issues as they are resolved:*

### High Priority (from previous review)
- [x] HP-1: Hardcoded Request ID Header in Recover Middleware
- [x] HP-2: Metrics Label Cardinality Risk
- [x] HP-3: File Handle Held Too Long in Static Handler

### Medium Priority (from previous review)
- [x] MP-1: String-Based Error Detection for Binding Errors
- [x] MP-2: JWT Claims Type Handling
- [x] MP-3: Metrics Gauge Precision Not Documented
- [x] MP-4: JWT Token Store Nil Check Returns Wrong Error
- [x] MP-5: Panic-Based Error Propagation (in router)

### Low Priority (from previous review)
- [x] LP-1: Missing Request Context Cancellation Wiring
- [x] LP-2: Consider pprof Endpoint Security
- [x] LP-3: SSE Goroutine Cleanup Verification
- [x] LP-4: OpenTelemetry Integration

### New Issues (from current review)
- [x] CR-1: Panic-Based Error Handling in Critical Paths
- [x] CR-2: CSRF Token Returns Empty on Random Failure
- [x] CR-3: HMAC Secret Key Length Not Validated
- [ ] CR-4: Unbounded Memory Growth in Rate Limiter
- [ ] HP-1: Ignored Errors in JSON Encoding
- [ ] HP-2: Race Condition in SSE Hub Broadcast
- [ ] HP-3: Potential Deadlock in Rate Limiter
- [ ] HP-4: JWT Validation Timing Side-Channel

---

## Issue Reference Format

When fixing an issue, reference it in the commit message:
```
fix: correct error message for nil token store

The JWT middleware now returns a clear error message when
TokenStore is not configured, instead of misleading "invalid token".
```

## Testing Requirements

Before marking an issue as fixed:
1. Add/update unit tests covering the fix
2. Run full test suite: `go test ./...`
3. Check for race conditions: `go test -race ./...`
4. Verify examples still work: `cd examples/xxx && go run .`
