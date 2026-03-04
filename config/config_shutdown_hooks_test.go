package config

import (
	"context"
	"errors"
	"testing"
)

func TestWithPreShutdownHook(t *testing.T) {
	called := false
	hook := func(ctx context.Context) error {
		called = true
		return nil
	}

	cfg := DefaultConfig
	opt := WithPreShutdownHook("test-hook", hook)
	opt(&cfg)

	if len(cfg.PreShutdownHooks) != 1 {
		t.Errorf("Expected 1 pre-shutdown hook, got %d", len(cfg.PreShutdownHooks))
	}

	if cfg.PreShutdownHooks[0].Name != "test-hook" {
		t.Errorf("Expected hook name 'test-hook', got '%s'", cfg.PreShutdownHooks[0].Name)
	}

	// Verify the hook works
	err := cfg.PreShutdownHooks[0].Hook(context.Background())
	if err != nil {
		t.Errorf("Expected no error from hook, got %v", err)
	}

	if !called {
		t.Error("Expected hook to be called")
	}
}

func TestWithShutdownHook(t *testing.T) {
	called := false
	hook := func(ctx context.Context) error {
		called = true
		return nil
	}

	cfg := DefaultConfig
	opt := WithShutdownHook("test-hook", hook)
	opt(&cfg)

	if len(cfg.ShutdownHooks) != 1 {
		t.Errorf("Expected 1 shutdown hook, got %d", len(cfg.ShutdownHooks))
	}

	if cfg.ShutdownHooks[0].Name != "test-hook" {
		t.Errorf("Expected hook name 'test-hook', got '%s'", cfg.ShutdownHooks[0].Name)
	}

	err := cfg.ShutdownHooks[0].Hook(context.Background())
	if err != nil {
		t.Errorf("Expected no error from hook, got %v", err)
	}

	if !called {
		t.Error("Expected hook to be called")
	}
}

func TestWithPostShutdownHook(t *testing.T) {
	called := false
	hook := func(ctx context.Context) error {
		called = true
		return nil
	}

	cfg := DefaultConfig
	opt := WithPostShutdownHook("test-hook", hook)
	opt(&cfg)

	if len(cfg.PostShutdownHooks) != 1 {
		t.Errorf("Expected 1 post-shutdown hook, got %d", len(cfg.PostShutdownHooks))
	}

	if cfg.PostShutdownHooks[0].Name != "test-hook" {
		t.Errorf("Expected hook name 'test-hook', got '%s'", cfg.PostShutdownHooks[0].Name)
	}

	err := cfg.PostShutdownHooks[0].Hook(context.Background())
	if err != nil {
		t.Errorf("Expected no error from hook, got %v", err)
	}

	if !called {
		t.Error("Expected hook to be called")
	}
}

func TestWithMultipleHooks(t *testing.T) {
	cfg := DefaultConfig

	opt1 := WithPreShutdownHook("hook-1", func(ctx context.Context) error {
		return nil
	})
	opt2 := WithPreShutdownHook("hook-2", func(ctx context.Context) error {
		return nil
	})

	opt1(&cfg)
	opt2(&cfg)

	if len(cfg.PreShutdownHooks) != 2 {
		t.Errorf("Expected 2 pre-shutdown hooks, got %d", len(cfg.PreShutdownHooks))
	}

	if cfg.PreShutdownHooks[0].Name != "hook-1" {
		t.Errorf("Expected first hook name 'hook-1', got '%s'", cfg.PreShutdownHooks[0].Name)
	}

	if cfg.PreShutdownHooks[1].Name != "hook-2" {
		t.Errorf("Expected second hook name 'hook-2', got '%s'", cfg.PreShutdownHooks[1].Name)
	}
}

func TestShutdownHookWithError(t *testing.T) {
	expectedErr := errors.New("hook error")
	hook := func(ctx context.Context) error {
		return expectedErr
	}

	cfg := DefaultConfig
	opt := WithShutdownHook("error-hook", hook)
	opt(&cfg)

	err := cfg.ShutdownHooks[0].Hook(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error '%v', got '%v'", expectedErr, err)
	}
}

func TestShutdownHookContextCancellation(t *testing.T) {
	hook := func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	cfg := DefaultConfig
	opt := WithShutdownHook("ctx-hook", hook)
	opt(&cfg)

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := cfg.ShutdownHooks[0].Hook(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}
