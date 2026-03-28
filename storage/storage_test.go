package storage

import (
	"errors"
	"testing"
)

func TestErrLockNotSupported(t *testing.T) {
	if ErrLockNotSupported == nil {
		t.Error("ErrLockNotSupported should not be nil")
	}
	if !errors.Is(ErrLockNotSupported, ErrLockNotSupported) {
		t.Error("ErrLockNotSupported should match itself")
	}
}
