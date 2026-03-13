package middleware

import (
	"bytes"
	"io"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/internal/problem"
	"github.com/alexferl/zerohttp/internal/rwutil"
	"github.com/alexferl/zerohttp/log"
)

// Idempotency creates middleware for idempotent request handling.
// It caches responses for state-changing operations and replays them for identical requests.
func Idempotency(cfg ...config.IdempotencyConfig) func(http.Handler) http.Handler {
	c := config.DefaultIdempotencyConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}

	if c.HeaderName == "" {
		c.HeaderName = config.DefaultIdempotencyConfig.HeaderName
	}
	if c.TTL == 0 {
		c.TTL = config.DefaultIdempotencyConfig.TTL
	}
	if c.MaxBodySize == 0 {
		c.MaxBodySize = config.DefaultIdempotencyConfig.MaxBodySize
	}
	if c.ExemptPaths == nil {
		c.ExemptPaths = config.DefaultIdempotencyConfig.ExemptPaths
	}
	if c.MaxKeys == 0 {
		c.MaxKeys = config.DefaultIdempotencyConfig.MaxKeys
	}
	if c.LockRetryInterval == 0 {
		c.LockRetryInterval = config.DefaultIdempotencyConfig.LockRetryInterval
	}
	if c.LockMaxRetries == 0 {
		c.LockMaxRetries = config.DefaultIdempotencyConfig.LockMaxRetries
	}
	if c.LockMaxInterval == 0 {
		c.LockMaxInterval = config.DefaultIdempotencyConfig.LockMaxInterval
	}
	if c.LockBackoffMultiplier == 0 {
		c.LockBackoffMultiplier = config.DefaultIdempotencyConfig.LockBackoffMultiplier
	}

	var store config.IdempotencyStore
	if c.Store != nil {
		store = c.Store
	} else {
		store = NewIdempotencyMemoryStore(c.MaxKeys)
	}

	stateChangingMethods := map[string]bool{
		http.MethodPost:   true,
		http.MethodPut:    true,
		http.MethodPatch:  true,
		http.MethodDelete: true,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !stateChangingMethods[r.Method] {
				next.ServeHTTP(w, r)
				return
			}

			for _, exemptPath := range c.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			idempotencyKey := r.Header.Get(c.HeaderName)

			if c.Required && idempotencyKey == "" {
				detail := problem.NewDetail(http.StatusBadRequest, "Idempotency-Key header is required")
				_ = detail.Render(w)
				return
			}

			if idempotencyKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(io.LimitReader(r.Body, c.MaxBodySize+1))
			if err != nil {
				log.GetGlobalLogger().Error("Failed to read request body for idempotency", log.E(err))
				detail := problem.NewDetail(http.StatusInternalServerError, "Failed to read request body")
				_ = detail.Render(w)
				return
			}
			_ = r.Body.Close()

			if int64(len(body)) > c.MaxBodySize {
				r.Body = io.NopCloser(bytes.NewReader(body))
				next.ServeHTTP(w, r)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))

			// Generate composite cache key using simple string join
			// (not SHA-256 since this is just a map key, not a security primitive)
			cacheKey := idempotencyKey + ":" + r.Method + ":" + r.URL.Path + ":" + string(body)

			record, found, err := store.Get(r.Context(), cacheKey)
			if err != nil {
				// Log error and continue (fail open)
				log.GetGlobalLogger().Error("Idempotency store get failed", log.E(err), log.F("key", cacheKey))
			} else if found {
				// Replay headers from flat slice [key1, val1, key2, val2, ...]
				for i := 0; i < len(record.Headers)-1; i += 2 {
					w.Header().Add(record.Headers[i], record.Headers[i+1])
				}
				w.Header().Set("X-Idempotency-Replay", "true")
				w.WriteHeader(record.StatusCode)
				_, _ = w.Write(record.Body)
				return
			}

			locked, err := store.Lock(r.Context(), cacheKey)
			if err != nil {
				log.GetGlobalLogger().Error("Idempotency store lock failed", log.E(err), log.F("key", cacheKey))
				next.ServeHTTP(w, r)
				return
			}
			if !locked {
				// Another request is in-flight, wait for it to complete with exponential backoff and jitter
				sleepInterval := c.LockRetryInterval
				for retries := 0; retries < c.LockMaxRetries; retries++ {
					jitteredInterval := addJitter(sleepInterval)

					select {
					case <-time.After(jitteredInterval):
					case <-r.Context().Done():
						detail := problem.NewDetail(http.StatusServiceUnavailable, "Request cancelled")
						_ = detail.Render(w)
						return
					}

					sleepInterval = time.Duration(math.Min(
						float64(sleepInterval)*c.LockBackoffMultiplier,
						float64(c.LockMaxInterval),
					))

					record, found, err = store.Get(r.Context(), cacheKey)
					if err != nil {
						log.GetGlobalLogger().Error("Idempotency store get failed while waiting", log.E(err), log.F("key", cacheKey))
						next.ServeHTTP(w, r)
						return
					}
					if found {
						// Replay headers from flat slice
						for i := 0; i < len(record.Headers)-1; i += 2 {
							w.Header().Add(record.Headers[i], record.Headers[i+1])
						}
						w.Header().Set("X-Idempotency-Replay", "true")
						w.WriteHeader(record.StatusCode)
						_, _ = w.Write(record.Body)
						return
					}
				}
				// Max retries exhausted, another request is still in-flight
				detail := problem.NewDetail(http.StatusConflict, "Idempotent request is still being processed")
				_ = detail.Render(w)
				return
			}

			// Ensure unlock happens even if handler panics
			defer func() {
				if err := store.Unlock(r.Context(), cacheKey); err != nil {
					log.GetGlobalLogger().Error("Idempotency store unlock failed", log.E(err), log.F("key", cacheKey))
				}
			}()

			recorder := &idempotencyResponseRecorder{
				ResponseBuffer: rwutil.NewResponseBuffer(w, 0), // 0 = unlimited buffering
			}

			next.ServeHTTP(recorder, r)

			if recorder.HasWritten && recorder.Status >= 200 && recorder.Status < 300 {
				record := config.IdempotencyRecord{
					StatusCode: recorder.Status,
					Headers:    recorder.headers,
					Body:       recorder.Buf.Bytes(),
					CreatedAt:  time.Now().UTC(),
				}

				if err := store.Set(r.Context(), cacheKey, record, c.TTL); err != nil {
					log.GetGlobalLogger().Error("Idempotency store set failed", log.E(err), log.F("key", cacheKey))
				}
			}
		})
	}
}

// idempotencyResponseRecorder captures response data for idempotency caching.
type idempotencyResponseRecorder struct {
	*rwutil.ResponseBuffer
	headers []string // flat slice: [key1, val1, key2, val2, ...]
}

func (i *idempotencyResponseRecorder) WriteHeader(statusCode int) {
	if i.HasWritten {
		return
	}
	i.ResponseBuffer.WriteHeader(statusCode)

	// Build flat header slice for efficient storage and replay
	for k, v := range i.Header() {
		// Skip hop-by-hop headers
		if k == "Connection" || k == "Keep-Alive" {
			continue
		}
		for _, val := range v {
			i.headers = append(i.headers, k, val)
		}
	}

	i.ResponseWriter.WriteHeader(statusCode)
	i.HeaderWritten = true
}

// Write captures the response body and forwards to the underlying ResponseWriter.
func (i *idempotencyResponseRecorder) Write(p []byte) (int, error) {
	if !i.HasWritten {
		i.WriteHeader(http.StatusOK)
	}
	// Buffer for caching and write through to client
	_, _ = i.Buf.Write(p)
	return i.ResponseWriter.Write(p)
}

// addJitter returns a duration with random jitter between 0.5x and 1.5x the base duration.
// This helps prevent thundering herd problems when many requests wait for the same lock.
// Uses math/rand (not crypto/rand) since jitter doesn't need cryptographic randomness.
func addJitter(base time.Duration) time.Duration {
	// Random value between 0.5 and 1.5 (represents +/- 50% jitter)
	jitter := 0.5 + rand.Float64()
	return time.Duration(float64(base) * jitter)
}
