package jwtauth

import (
	"context"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestGetJTI(t *testing.T) {
	t.Run("jti present", func(t *testing.T) {
		claims := map[string]any{JWTClaimJWTID: "abc-123"}
		jti := GetJTI(claims)
		zhtest.AssertEqual(t, "abc-123", jti)
	})

	t.Run("jti missing", func(t *testing.T) {
		claims := map[string]any{"sub": "user123"}
		jti := GetJTI(claims)
		zhtest.AssertEqual(t, "", jti)
	})

	t.Run("jti not string", func(t *testing.T) {
		claims := map[string]any{JWTClaimJWTID: 123}
		jti := GetJTI(claims)
		zhtest.AssertEqual(t, "", jti)
	})

	t.Run("nil claims", func(t *testing.T) {
		jti := GetJTI(nil)
		zhtest.AssertEqual(t, "", jti)
	})

	t.Run("empty claims", func(t *testing.T) {
		jti := GetJTI(map[string]any{})
		zhtest.AssertEqual(t, "", jti)
	})
}

// Verify RevocationStore interface can be implemented
type testRevocationStore struct{}

func (t *testRevocationStore) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	return nil
}

func (t *testRevocationStore) IsRevoked(ctx context.Context, jti string) (bool, error) {
	return false, nil
}

func (t *testRevocationStore) Close() error {
	return nil
}

var _ RevocationStore = (*testRevocationStore)(nil)
