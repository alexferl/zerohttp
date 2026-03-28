// Package storage provides a shared storage interface for middleware backends.
//
// This package defines the Storage interface that users implement to provide
// custom storage backends (Redis, PostgreSQL, etc.) for middlewares like cache
// and idempotency.
//
// Example user implementation:
//
//	type MyRedisStorage struct { client *redis.Client }
//	func (s *MyRedisStorage) Get(ctx context.Context, key string) ([]byte, bool, error) { ... }
//	func (s *MyRedisStorage) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error { ... }
//	func (s *MyRedisStorage) Delete(ctx context.Context, key string) error { ... }
//
// Middlewares provide adapter types that wrap storage.Storage and handle
// serialization of their specific Record types.
package storage
