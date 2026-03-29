package zerohttp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestServer_RegisterPreShutdownHook(t *testing.T) {
	server := New()

	called := false
	server.RegisterPreShutdownHook("test-hook", func(ctx context.Context) error {
		called = true
		return nil
	})

	zhtest.AssertEqual(t, 1, len(server.preShutdownHooks))
	zhtest.AssertEqual(t, "test-hook", server.preShutdownHooks[0].Name)

	// Verify hook works
	err := server.preShutdownHooks[0].Hook(context.Background())
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, called)
}

func TestServer_RegisterShutdownHook(t *testing.T) {
	server := New()

	called := false
	server.RegisterShutdownHook("test-hook", func(ctx context.Context) error {
		called = true
		return nil
	})

	zhtest.AssertEqual(t, 1, len(server.shutdownHooks))
	zhtest.AssertEqual(t, "test-hook", server.shutdownHooks[0].Name)

	err := server.shutdownHooks[0].Hook(context.Background())
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, called)
}

func TestServer_RegisterPostShutdownHook(t *testing.T) {
	server := New()

	called := false
	server.RegisterPostShutdownHook("test-hook", func(ctx context.Context) error {
		called = true
		return nil
	})

	zhtest.AssertEqual(t, 1, len(server.postShutdownHooks))
	zhtest.AssertEqual(t, "test-hook", server.postShutdownHooks[0].Name)

	err := server.postShutdownHooks[0].Hook(context.Background())
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, called)
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
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, 2, len(order))
	zhtest.AssertEqual(t, "hook-1", order[0])
	zhtest.AssertEqual(t, "hook-2", order[1])
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
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, 2, len(order))
	zhtest.AssertEqual(t, "hook-1", order[0])
	zhtest.AssertEqual(t, "hook-2", order[1])
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
	zhtest.AssertNoError(t, err)

	mu.Lock()
	zhtest.AssertEqual(t, 2, len(calls))
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
	zhtest.AssertNoError(t, err)

	// Both hooks should have been called
	zhtest.AssertEqual(t, 2, len(calls))
	zhtest.AssertEqual(t, "failing", calls[0])
	zhtest.AssertEqual(t, "success", calls[1])
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
	zhtest.AssertErrorIs(t, err, context.Canceled)

	// Hook should not have been called due to cancelled context
	zhtest.AssertFalse(t, called)
}

func TestServer_ConfigWithShutdownHooks(t *testing.T) {
	var preCalled, shutdownCalled, postCalled bool

	server := New(Config{
		Lifecycle: LifecycleConfig{
			PreShutdownHooks: []ShutdownHookConfig{
				{Name: "pre", Hook: func(ctx context.Context) error {
					preCalled = true
					return nil
				}},
			},
			ShutdownHooks: []ShutdownHookConfig{
				{Name: "shutdown", Hook: func(ctx context.Context) error {
					shutdownCalled = true
					return nil
				}},
			},
			PostShutdownHooks: []ShutdownHookConfig{
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
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, preCalled)
	zhtest.AssertTrue(t, shutdownCalled)
	zhtest.AssertTrue(t, postCalled)
}

// ============================================================================
// Startup Hook Tests
// ============================================================================

func TestServer_RegisterPreStartupHook(t *testing.T) {
	server := New()

	called := false
	server.RegisterPreStartupHook("test-hook", func(ctx context.Context) error {
		called = true
		return nil
	})

	zhtest.AssertEqual(t, 1, len(server.preStartupHooks))
	zhtest.AssertEqual(t, "test-hook", server.preStartupHooks[0].Name)

	err := server.preStartupHooks[0].Hook(context.Background())
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, called)
}

func TestServer_RegisterStartupHook(t *testing.T) {
	server := New()

	called := false
	server.RegisterStartupHook("test-hook", func(ctx context.Context) error {
		called = true
		return nil
	})

	zhtest.AssertEqual(t, 1, len(server.startupHooks))
	zhtest.AssertEqual(t, "test-hook", server.startupHooks[0].Name)

	err := server.startupHooks[0].Hook(context.Background())
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, called)
}

func TestServer_RegisterPostStartupHook(t *testing.T) {
	server := New()

	called := false
	server.RegisterPostStartupHook("test-hook", func(ctx context.Context) error {
		called = true
		return nil
	})

	zhtest.AssertEqual(t, 1, len(server.postStartupHooks))
	zhtest.AssertEqual(t, "test-hook", server.postStartupHooks[0].Name)

	err := server.postStartupHooks[0].Hook(context.Background())
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, called)
}

func TestServer_StartupHooks_RunInOrder(t *testing.T) {
	var order []string
	var mu sync.Mutex
	hooksDone := make(chan struct{})
	server := New(Config{
		Lifecycle: LifecycleConfig{
			StartupHooks: []StartupHookConfig{
				{Name: "hook-1", Hook: func(ctx context.Context) error {
					mu.Lock()
					order = append(order, "hook-1")
					mu.Unlock()
					return nil
				}},
				{Name: "hook-2", Hook: func(ctx context.Context) error {
					mu.Lock()
					order = append(order, "hook-2")
					mu.Unlock()
					return nil
				}},
			},
			PostStartupHooks: []StartupHookConfig{
				{Name: "done", Hook: func(ctx context.Context) error {
					close(hooksDone)
					return nil
				}},
			},
		},
	})

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	// Start server in goroutine
	go func() {
		_ = server.Start()
	}()

	// Wait for hooks to complete
	select {
	case <-hooksDone:
	case <-time.After(2 * time.Second):
		zhtest.AssertFail(t, "timeout waiting for hooks")
	}

	// Shutdown to clean up
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	mu.Lock()
	defer mu.Unlock()
	zhtest.AssertEqual(t, 2, len(order))
	zhtest.AssertEqual(t, "hook-1", order[0])
	zhtest.AssertEqual(t, "hook-2", order[1])
}

func TestServer_StartupHook_FailsServerStart(t *testing.T) {
	hookRan := make(chan bool, 1)
	server := New(Config{
		Lifecycle: LifecycleConfig{
			StartupHooks: []StartupHookConfig{
				{Name: "failing-hook", Hook: func(ctx context.Context) error {
					hookRan <- true
					return errors.New("startup failed")
				}},
			},
		},
	})

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	// Start in goroutine since it blocks
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	// Wait for hook to run or timeout
	var hookRanOK bool
	select {
	case <-hookRan:
		hookRanOK = true
	case <-time.After(500 * time.Millisecond):
		hookRanOK = false
	}
	zhtest.AssertTrue(t, hookRanOK)

	// Wait for Start() to return
	var startErr error
	var gotErr bool
	select {
	case startErr = <-errChan:
		gotErr = true
	case <-time.After(500 * time.Millisecond):
		gotErr = false
	}
	zhtest.AssertTrue(t, gotErr)
	zhtest.AssertError(t, startErr)
}

func TestServer_StartupHook_StopsOnFirstError(t *testing.T) {
	var calls []string
	var mu sync.Mutex
	server := New(Config{
		Lifecycle: LifecycleConfig{
			StartupHooks: []StartupHookConfig{
				{Name: "first", Hook: func(ctx context.Context) error {
					mu.Lock()
					calls = append(calls, "first")
					mu.Unlock()
					return errors.New("first failed")
				}},
				{Name: "second", Hook: func(ctx context.Context) error {
					mu.Lock()
					calls = append(calls, "second")
					mu.Unlock()
					return nil
				}},
			},
		},
	})

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	go func() {
		_ = server.Start()
	}()

	// Give time for startup hooks to run
	time.Sleep(50 * time.Millisecond)

	// Shutdown to clean up
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	mu.Lock()
	defer mu.Unlock()

	// Only first hook should have run
	zhtest.AssertEqual(t, 1, len(calls))
	if len(calls) > 0 {
		zhtest.AssertEqual(t, "first", calls[0])
	}
}

func TestServer_StartupHook_RespectsContextCancellation(t *testing.T) {
	// Create a server with a startup hook that checks context
	var hookCalled bool
	server := New(Config{
		Lifecycle: LifecycleConfig{
			StartupHooks: []StartupHookConfig{
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
	zhtest.AssertError(t, err)

	// When context is cancelled before hooks run, hooks should not be called
	zhtest.AssertFalse(t, hookCalled)
}

func TestServer_StartupHook_ViaRegisterMethod(t *testing.T) {
	var order []string
	var mu sync.Mutex
	hooksDone := make(chan struct{})
	server := New()

	server.RegisterStartupHook("hook-a", func(ctx context.Context) error {
		mu.Lock()
		order = append(order, "hook-a")
		mu.Unlock()
		return nil
	})
	server.RegisterStartupHook("hook-b", func(ctx context.Context) error {
		mu.Lock()
		order = append(order, "hook-b")
		mu.Unlock()
		return nil
	})

	// Also add some via config (simulating what New() does with c.StartupHooks)
	server.startupHooks = append(server.startupHooks, StartupHookConfig{
		Name: "hook-config",
		Hook: func(ctx context.Context) error {
			mu.Lock()
			order = append(order, "hook-config")
			mu.Unlock()
			return nil
		},
	})

	// Add post-startup hook to signal completion
	server.postStartupHooks = append(server.postStartupHooks, StartupHookConfig{
		Name: "done",
		Hook: func(ctx context.Context) error {
			close(hooksDone)
			return nil
		},
	})

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	go func() {
		_ = server.Start()
	}()

	// Wait for hooks to complete
	select {
	case <-hooksDone:
	case <-time.After(2 * time.Second):
		zhtest.AssertFail(t, "timeout waiting for hooks")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	// Should have 3 hooks: hook-a, hook-b, hook-config
	mu.Lock()
	defer mu.Unlock()
	zhtest.AssertEqual(t, 3, len(order))
}

func TestServer_StartupHookOrder(t *testing.T) {
	var order []string
	var mu sync.Mutex
	hooksDone := make(chan struct{})
	server := New(Config{
		Lifecycle: LifecycleConfig{
			PreStartupHooks: []StartupHookConfig{
				{Name: "pre", Hook: func(ctx context.Context) error {
					mu.Lock()
					order = append(order, "pre")
					mu.Unlock()
					return nil
				}},
			},
			StartupHooks: []StartupHookConfig{
				{Name: "startup", Hook: func(ctx context.Context) error {
					mu.Lock()
					order = append(order, "startup")
					mu.Unlock()
					return nil
				}},
			},
			PostStartupHooks: []StartupHookConfig{
				{Name: "post", Hook: func(ctx context.Context) error {
					mu.Lock()
					order = append(order, "post")
					mu.Unlock()
					close(hooksDone)
					return nil
				}},
			},
		},
	})

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	go func() {
		_ = server.Start()
	}()

	// Wait for hooks to complete
	select {
	case <-hooksDone:
	case <-time.After(2 * time.Second):
		zhtest.AssertFail(t, "timeout waiting for hooks")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	mu.Lock()
	defer mu.Unlock()
	zhtest.AssertEqual(t, 3, len(order))
	zhtest.AssertEqual(t, "pre", order[0])
	zhtest.AssertEqual(t, "startup", order[1])
	zhtest.AssertEqual(t, "post", order[2])
}

func TestServer_PreStartupHook_FailsServerStart(t *testing.T) {
	server := New(Config{
		Lifecycle: LifecycleConfig{
			PreStartupHooks: []StartupHookConfig{
				{Name: "failing-hook", Hook: func(ctx context.Context) error {
					return errors.New("pre-startup failed")
				}},
			},
		},
	})

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	err := server.Start()
	zhtest.AssertError(t, err)
	zhtest.AssertTrue(t, strings.Contains(err.Error(), "pre-startup hook \"failing-hook\" failed"))
}

func TestServer_StartupHook_FailsAndShutsDownServers(t *testing.T) {
	hookRan := make(chan bool, 1)
	server := New(Config{
		Lifecycle: LifecycleConfig{
			StartupHooks: []StartupHookConfig{
				{Name: "failing-hook", Hook: func(ctx context.Context) error {
					hookRan <- true
					return errors.New("startup failed")
				}},
			},
		},
	})

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	// Start should fail
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	// Wait for hook to run or timeout
	var hookRanOK bool
	select {
	case <-hookRan:
		hookRanOK = true
	case <-time.After(500 * time.Millisecond):
		hookRanOK = false
	}
	zhtest.AssertTrue(t, hookRanOK)

	// Wait for Start() to return
	var startErr error
	var gotErr bool
	select {
	case startErr = <-errChan:
		gotErr = true
	case <-time.After(500 * time.Millisecond):
		gotErr = false
	}
	zhtest.AssertTrue(t, gotErr)
	zhtest.AssertError(t, startErr)
}

func TestServer_PostStartupHook_RunsAfterStartupHookCompletes(t *testing.T) {
	var order []string
	var startupComplete bool
	var mu sync.Mutex

	server := New(Config{
		Lifecycle: LifecycleConfig{
			StartupHooks: []StartupHookConfig{
				{Name: "slow-startup", Hook: func(ctx context.Context) error {
					time.Sleep(100 * time.Millisecond)
					mu.Lock()
					order = append(order, "startup")
					startupComplete = true
					mu.Unlock()
					return nil
				}},
			},
			PostStartupHooks: []StartupHookConfig{
				{Name: "post", Hook: func(ctx context.Context) error {
					mu.Lock()
					defer mu.Unlock()
					// Verify startup hook completed before this ran
					zhtest.AssertTrue(t, startupComplete)
					order = append(order, "post")
					return nil
				}},
			},
		},
	})

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	server.listener = listener
	server.server = &http.Server{Addr: listener.Addr().String()}

	go func() {
		_ = server.Start()
	}()

	time.Sleep(200 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	mu.Lock()
	defer mu.Unlock()

	zhtest.AssertEqual(t, 2, len(order))
	if len(order) >= 2 {
		zhtest.AssertEqual(t, "startup", order[0])
		zhtest.AssertEqual(t, "post", order[1])
	}
}
