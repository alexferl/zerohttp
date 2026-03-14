package zerohttp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
)

func TestServer_RegisterPreShutdownHook(t *testing.T) {
	server := New()

	called := false
	server.RegisterPreShutdownHook("test-hook", func(ctx context.Context) error {
		called = true
		return nil
	})

	if len(server.preShutdownHooks) != 1 {
		t.Errorf("Expected 1 pre-shutdown hook, got %d", len(server.preShutdownHooks))
	}

	if server.preShutdownHooks[0].Name != "test-hook" {
		t.Errorf("Expected hook name 'test-hook', got '%s'", server.preShutdownHooks[0].Name)
	}

	// Verify hook works
	err := server.preShutdownHooks[0].Hook(context.Background())
	if err != nil {
		t.Errorf("Expected no error from hook, got %v", err)
	}

	if !called {
		t.Error("Expected hook to be called")
	}
}

func TestServer_RegisterShutdownHook(t *testing.T) {
	server := New()

	called := false
	server.RegisterShutdownHook("test-hook", func(ctx context.Context) error {
		called = true
		return nil
	})

	if len(server.shutdownHooks) != 1 {
		t.Errorf("Expected 1 shutdown hook, got %d", len(server.shutdownHooks))
	}

	if server.shutdownHooks[0].Name != "test-hook" {
		t.Errorf("Expected hook name 'test-hook', got '%s'", server.shutdownHooks[0].Name)
	}

	err := server.shutdownHooks[0].Hook(context.Background())
	if err != nil {
		t.Errorf("Expected no error from hook, got %v", err)
	}

	if !called {
		t.Error("Expected hook to be called")
	}
}

func TestServer_RegisterPostShutdownHook(t *testing.T) {
	server := New()

	called := false
	server.RegisterPostShutdownHook("test-hook", func(ctx context.Context) error {
		called = true
		return nil
	})

	if len(server.postShutdownHooks) != 1 {
		t.Errorf("Expected 1 post-shutdown hook, got %d", len(server.postShutdownHooks))
	}

	if server.postShutdownHooks[0].Name != "test-hook" {
		t.Errorf("Expected hook name 'test-hook', got '%s'", server.postShutdownHooks[0].Name)
	}

	err := server.postShutdownHooks[0].Hook(context.Background())
	if err != nil {
		t.Errorf("Expected no error from hook, got %v", err)
	}

	if !called {
		t.Error("Expected hook to be called")
	}
}

func TestServer_Shutdown_WithPreShutdownHooks(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server := New()
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	var order []string
	server.RegisterPreShutdownHook("hook-1", func(ctx context.Context) error {
		order = append(order, "hook-1")
		return nil
	})
	server.RegisterPreShutdownHook("hook-2", func(ctx context.Context) error {
		order = append(order, "hook-2")
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(order) != 2 {
		t.Errorf("Expected 2 hooks to run, got %d", len(order))
	}

	if order[0] != "hook-1" || order[1] != "hook-2" {
		t.Errorf("Expected hooks to run in registration order, got %v", order)
	}
}

func TestServer_Shutdown_WithPostShutdownHooks(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server := New()
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	var order []string
	server.RegisterPostShutdownHook("hook-1", func(ctx context.Context) error {
		order = append(order, "hook-1")
		return nil
	})
	server.RegisterPostShutdownHook("hook-2", func(ctx context.Context) error {
		order = append(order, "hook-2")
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(order) != 2 {
		t.Errorf("Expected 2 hooks to run, got %d", len(order))
	}

	if order[0] != "hook-1" || order[1] != "hook-2" {
		t.Errorf("Expected hooks to run in registration order, got %v", order)
	}
}

func TestServer_Shutdown_WithShutdownHooks(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server := New()
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	var mu sync.Mutex
	var calls []string
	server.RegisterShutdownHook("hook-1", func(ctx context.Context) error {
		mu.Lock()
		calls = append(calls, "hook-1")
		mu.Unlock()
		return nil
	})
	server.RegisterShutdownHook("hook-2", func(ctx context.Context) error {
		mu.Lock()
		calls = append(calls, "hook-2")
		mu.Unlock()
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	mu.Lock()
	if len(calls) != 2 {
		t.Errorf("Expected 2 shutdown hooks to run, got %d", len(calls))
	}
	mu.Unlock()
}

func TestServer_Shutdown_HooksContinueOnError(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server := New()
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	var calls []string
	server.RegisterPreShutdownHook("failing-hook", func(ctx context.Context) error {
		calls = append(calls, "failing")
		return errors.New("hook error")
	})
	server.RegisterPreShutdownHook("success-hook", func(ctx context.Context) error {
		calls = append(calls, "success")
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Shutdown should complete without returning the hook error
	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error from server shutdown, got %v", err)
	}

	// Both hooks should have been called
	if len(calls) != 2 {
		t.Errorf("Expected both hooks to run, got %v", calls)
	}

	if calls[0] != "failing" || calls[1] != "success" {
		t.Errorf("Expected hooks to run in order despite first failing, got %v", calls)
	}
}

func TestServer_Shutdown_HooksRespectContextCancellation(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server := New()
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false
	server.RegisterPreShutdownHook("should-not-run", func(ctx context.Context) error {
		called = true
		return nil
	})

	err := server.Shutdown(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	// Hook should not have been called due to cancelled context
	if called {
		t.Error("Expected hook not to be called due to cancelled context")
	}
}

func TestServer_ConfigWithShutdownHooks(t *testing.T) {
	var preCalled, shutdownCalled, postCalled bool

	server := New(config.Config{
		Lifecycle: config.LifecycleConfig{
			PreShutdownHooks: []config.ShutdownHookConfig{
				{Name: "pre", Hook: func(ctx context.Context) error {
					preCalled = true
					return nil
				}},
			},
			ShutdownHooks: []config.ShutdownHookConfig{
				{Name: "shutdown", Hook: func(ctx context.Context) error {
					shutdownCalled = true
					return nil
				}},
			},
			PostShutdownHooks: []config.ShutdownHookConfig{
				{Name: "post", Hook: func(ctx context.Context) error {
					postCalled = true
					return nil
				}},
			},
		},
	})

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !preCalled {
		t.Error("Expected pre-shutdown hook to be called")
	}
	if !shutdownCalled {
		t.Error("Expected shutdown hook to be called")
	}
	if !postCalled {
		t.Error("Expected post-shutdown hook to be called")
	}
}

// ============================================================================
// Startup Hook Tests
// ============================================================================

func TestServer_RegisterStartupHook(t *testing.T) {
	server := New()

	called := false
	server.RegisterStartupHook("test-hook", func(ctx context.Context) error {
		called = true
		return nil
	})

	if len(server.startupHooks) != 1 {
		t.Errorf("Expected 1 startup hook, got %d", len(server.startupHooks))
	}

	if server.startupHooks[0].Name != "test-hook" {
		t.Errorf("Expected hook name 'test-hook', got '%s'", server.startupHooks[0].Name)
	}

	err := server.startupHooks[0].Hook(context.Background())
	if err != nil {
		t.Errorf("Expected no error from hook, got %v", err)
	}

	if !called {
		t.Error("Expected hook to be called")
	}
}

func TestServer_StartupHooks_RunInOrder(t *testing.T) {
	var order []string
	server := New(config.Config{
		Lifecycle: config.LifecycleConfig{
			StartupHooks: []config.StartupHookConfig{
				{Name: "hook-1", Hook: func(ctx context.Context) error {
					order = append(order, "hook-1")
					return nil
				}},
				{Name: "hook-2", Hook: func(ctx context.Context) error {
					order = append(order, "hook-2")
					return nil
				}},
			},
		},
	})

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	// Start server in goroutine - it will fail because we don't actually run it
	// but startup hooks will run
	go func() {
		_ = server.Start()
	}()

	// Give time for startup hooks to run
	time.Sleep(100 * time.Millisecond)

	// Shutdown to clean up
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	if len(order) != 2 {
		t.Errorf("Expected 2 hooks to run, got %d", len(order))
	}

	if len(order) >= 2 && (order[0] != "hook-1" || order[1] != "hook-2") {
		t.Errorf("Expected hooks to run in registration order, got %v", order)
	}
}

func TestServer_StartupHook_FailsServerStart(t *testing.T) {
	server := New(config.Config{
		Lifecycle: config.LifecycleConfig{
			StartupHooks: []config.StartupHookConfig{
				{Name: "failing-hook", Hook: func(ctx context.Context) error {
					return errors.New("startup failed")
				}},
			},
		},
	})

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	err := server.Start()
	if err == nil {
		t.Error("Expected server start to fail due to startup hook error")
	}

	if !errors.Is(err, context.Canceled) && err.Error() != `startup hook "failing-hook" failed: startup failed` {
		t.Errorf("Expected startup hook error, got: %v", err)
	}
}

func TestServer_StartupHook_StopsOnFirstError(t *testing.T) {
	var calls []string
	server := New(config.Config{
		Lifecycle: config.LifecycleConfig{
			StartupHooks: []config.StartupHookConfig{
				{Name: "first", Hook: func(ctx context.Context) error {
					calls = append(calls, "first")
					return errors.New("first failed")
				}},
				{Name: "second", Hook: func(ctx context.Context) error {
					calls = append(calls, "second")
					return nil
				}},
			},
		},
	})

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	_ = server.Start()

	// Only first hook should have run
	if len(calls) != 1 {
		t.Errorf("Expected only 1 hook to run, got %d: %v", len(calls), calls)
	}

	if len(calls) > 0 && calls[0] != "first" {
		t.Errorf("Expected first hook to run, got %v", calls)
	}
}

func TestServer_StartupHook_RespectsContextCancellation(t *testing.T) {
	// Create a server with a startup hook that checks context
	var hookCalled bool
	server := New(config.Config{
		Lifecycle: config.LifecycleConfig{
			StartupHooks: []config.StartupHookConfig{
				{Name: "context-check", Hook: func(ctx context.Context) error {
					hookCalled = true
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
						return nil
					}
				}},
			},
		},
	})

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	// Cancel the base context before starting
	server.cancelBaseCtx()

	err := server.Start()
	if err == nil {
		t.Error("Expected server start to fail due to cancelled context")
	}

	// When context is cancelled before hooks run, hooks should not be called
	if hookCalled {
		t.Error("Expected hook not to be called when context is already cancelled")
	}
}

func TestServer_StartupHook_ViaRegisterMethod(t *testing.T) {
	var order []string
	server := New()

	server.RegisterStartupHook("hook-a", func(ctx context.Context) error {
		order = append(order, "hook-a")
		return nil
	})
	server.RegisterStartupHook("hook-b", func(ctx context.Context) error {
		order = append(order, "hook-b")
		return nil
	})

	// Also add some via config (simulating what New() does with c.StartupHooks)
	server.startupHooks = append(server.startupHooks, config.StartupHookConfig{
		Name: "hook-config",
		Hook: func(ctx context.Context) error {
			order = append(order, "hook-config")
			return nil
		},
	})

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	go func() {
		_ = server.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	// Should have 3 hooks: hook-a, hook-b, hook-config
	if len(order) != 3 {
		t.Errorf("Expected 3 hooks to run, got %d: %v", len(order), order)
	}
}
