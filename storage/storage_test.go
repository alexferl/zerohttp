package storage

import (
	"errors"
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestErrLockNotSupported(t *testing.T) {
	zhtest.AssertNotNil(t, ErrLockNotSupported)
	zhtest.AssertTrue(t, errors.Is(ErrLockNotSupported, ErrLockNotSupported))
}
