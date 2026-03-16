// Package zerohttp provides server lifecycle hooks. See [Server.RegisterPreStartupHook], [Server.RegisterStartupHook], and [Server.RegisterShutdownHook].
package zerohttp

import (
	"context"
	"fmt"
	"sync"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/log"
)

// RegisterPreStartupHook registers a hook to run before servers start and before startup hooks.
// Pre-startup hooks execute sequentially in registration order.
// If any pre-startup hook returns an error, the server will not start.
//
// Hooks must respect context cancellation by checking ctx.Done().
// If a hook blocks without respecting the context, startup will hang.
//
// Example:
//
//	app.RegisterPreStartupHook("validate-config", func(ctx context.Context) error {
//	    return validateConfig()
//	})
func (s *Server) RegisterPreStartupHook(name string, hook config.StartupHook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.preStartupHooks = append(s.preStartupHooks, config.StartupHookConfig{Name: name, Hook: hook})
}

// RegisterStartupHook registers a hook to run concurrently with servers starting up.
// Startup hooks execute sequentially in registration order, after PreStartupHooks.
// If any startup hook returns an error, the server will not start.
//
// Hooks must respect context cancellation by checking ctx.Done().
// If a hook blocks without respecting the context, startup will hang.
//
// Example:
//
//	app.RegisterStartupHook("migrations", func(ctx context.Context) error {
//	    return goose.Up(db.DB, "migrations")
//	})
func (s *Server) RegisterStartupHook(name string, hook config.StartupHook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startupHooks = append(s.startupHooks, config.StartupHookConfig{Name: name, Hook: hook})
}

// RegisterPostStartupHook registers a hook to run after servers have started accepting connections.
// Post-startup hooks execute sequentially in registration order.
// Errors from post-startup hooks are logged but do not stop the server.
//
// Hooks must respect context cancellation by checking ctx.Done().
// If a hook blocks without respecting the context, startup will hang.
//
// Example:
//
//	app.RegisterPostStartupHook("announce-ready", func(ctx context.Context) error {
//	    return notifyServiceDiscovery()
//	})
func (s *Server) RegisterPostStartupHook(name string, hook config.StartupHook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.postStartupHooks = append(s.postStartupHooks, config.StartupHookConfig{Name: name, Hook: hook})
}

// runPreStartupHooks executes pre-startup hooks sequentially in registration order.
func (s *Server) runPreStartupHooks(ctx context.Context) error {
	s.mu.RLock()
	hooks := s.preStartupHooks
	s.mu.RUnlock()

	if len(hooks) == 0 {
		return nil
	}

	s.logger.Debug("Running pre-startup hooks", log.F("count", len(hooks)))

	for _, hook := range hooks {
		select {
		case <-ctx.Done():
			s.logger.Warn("Pre-startup hook aborted due to context cancellation", log.F("hook", hook.Name))
			return ctx.Err()
		default:
		}

		s.logger.Debug("Running pre-startup hook", log.F("hook", hook.Name))
		if err := hook.Hook(ctx); err != nil {
			s.logger.Error("Pre-startup hook failed", log.F("hook", hook.Name), log.E(err))
			return fmt.Errorf("pre-startup hook %q failed: %w", hook.Name, err)
		}
	}

	s.logger.Debug("All pre-startup hooks completed successfully")
	return nil
}

// runStartupHooks executes startup hooks sequentially in registration order.
// If any hook returns an error, execution stops and the error is returned.
func (s *Server) runStartupHooks(ctx context.Context) error {
	s.mu.RLock()
	hooks := s.startupHooks
	s.mu.RUnlock()

	if len(hooks) == 0 {
		return nil
	}

	s.logger.Debug("Running startup hooks", log.F("count", len(hooks)))

	for _, hook := range hooks {
		select {
		case <-ctx.Done():
			s.logger.Warn("Startup hook aborted due to context cancellation", log.F("hook", hook.Name))
			return ctx.Err()
		default:
		}

		s.logger.Debug("Running startup hook", log.F("hook", hook.Name))
		if err := hook.Hook(ctx); err != nil {
			s.logger.Error("Startup hook failed", log.F("hook", hook.Name), log.E(err))
			return fmt.Errorf("startup hook %q failed: %w", hook.Name, err)
		}
	}

	s.logger.Debug("All startup hooks completed successfully")
	return nil
}

// runPostStartupHooks executes post-startup hooks sequentially in registration order.
func (s *Server) runPostStartupHooks(ctx context.Context) error {
	s.mu.RLock()
	hooks := s.postStartupHooks
	s.mu.RUnlock()

	if len(hooks) == 0 {
		return nil
	}

	s.logger.Debug("Running post-startup hooks", log.F("count", len(hooks)))

	for _, hook := range hooks {
		select {
		case <-ctx.Done():
			s.logger.Warn("Post-startup hook aborted due to context cancellation", log.F("hook", hook.Name))
			return ctx.Err()
		default:
		}

		s.logger.Debug("Running post-startup hook", log.F("hook", hook.Name))
		if err := hook.Hook(ctx); err != nil {
			s.logger.Error("Post-startup hook failed", log.F("hook", hook.Name), log.E(err))
			// Continue with other hooks despite error
		}
	}

	s.logger.Debug("All post-startup hooks completed successfully")
	return nil
}

// RegisterPreShutdownHook registers a hook to run before server shutdown begins.
// Pre-shutdown hooks execute sequentially in registration order, before servers stop.
// Errors from pre-shutdown hooks are logged but do not stop shutdown.
//
// Hooks must respect context cancellation by checking ctx.Done().
// If a hook blocks without respecting the context, shutdown will hang.
//
// Example:
//
//	app.RegisterPreShutdownHook("health", func(ctx context.Context) error {
//	    health.SetUnhealthy()
//	    return nil
//	})
func (s *Server) RegisterPreShutdownHook(name string, hook config.ShutdownHook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.preShutdownHooks = append(s.preShutdownHooks, config.ShutdownHookConfig{Name: name, Hook: hook})
}

// RegisterShutdownHook registers a hook to run concurrently with server shutdown.
// Shutdown hooks execute concurrently alongside server shutdown.
// Errors from shutdown hooks are logged but do not stop shutdown.
//
// Hooks must respect context cancellation by checking ctx.Done().
// If a hook blocks without respecting the context, shutdown will hang.
//
// Example:
//
//	app.RegisterShutdownHook("close-db", func(ctx context.Context) error {
//	    return db.Close()
//	})
func (s *Server) RegisterShutdownHook(name string, hook config.ShutdownHook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shutdownHooks = append(s.shutdownHooks, config.ShutdownHookConfig{Name: name, Hook: hook})
}

// RegisterPostShutdownHook registers a hook to run after servers have shut down.
// Post-shutdown hooks execute sequentially in registration order.
// Errors from post-shutdown hooks are logged but do not affect shutdown.
//
// Hooks must respect context cancellation by checking ctx.Done().
// If a hook blocks without respecting the context, shutdown will hang.
//
// Example:
//
//	app.RegisterPostShutdownHook("cleanup", func(ctx context.Context) error {
//	    return os.RemoveAll("/tmp/app-*")
//	})
func (s *Server) RegisterPostShutdownHook(name string, hook config.ShutdownHook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.postShutdownHooks = append(s.postShutdownHooks, config.ShutdownHookConfig{Name: name, Hook: hook})
}

// runPreShutdownHooks executes pre-shutdown hooks sequentially in registration order.
func (s *Server) runPreShutdownHooks(ctx context.Context) error {
	s.mu.RLock()
	hooks := s.preShutdownHooks
	s.mu.RUnlock()

	if len(hooks) == 0 {
		return nil
	}

	s.logger.Debug("Running pre-shutdown hooks", log.F("count", len(hooks)))

	for _, hook := range hooks {
		select {
		case <-ctx.Done():
			s.logger.Warn("Pre-shutdown hook aborted due to context cancellation", log.F("hook", hook.Name))
			return ctx.Err()
		default:
		}

		s.logger.Debug("Running pre-shutdown hook", log.F("hook", hook.Name))
		if err := hook.Hook(ctx); err != nil {
			s.logger.Error("Pre-shutdown hook failed", log.F("hook", hook.Name), log.E(err))
			// Continue with other hooks despite error
		}
	}

	return nil
}

// startShutdownHooks starts shutdown hooks concurrently and returns a WaitGroup and error channel.
// The caller must wait on the returned WaitGroup and then close the error channel.
func (s *Server) startShutdownHooks(ctx context.Context) (*sync.WaitGroup, chan error) {
	s.mu.RLock()
	hooks := s.shutdownHooks
	s.mu.RUnlock()

	var wg sync.WaitGroup
	errCh := make(chan error, len(hooks))

	if len(hooks) == 0 {
		return &wg, errCh
	}

	s.logger.Debug("Starting shutdown hooks", log.F("count", len(hooks)))

	for _, hook := range hooks {
		wg.Add(1)
		go func(h config.ShutdownHookConfig) {
			defer wg.Done()

			s.logger.Debug("Running shutdown hook", log.F("hook", h.Name))
			if err := h.Hook(ctx); err != nil {
				s.logger.Error("Shutdown hook failed", log.F("hook", h.Name), log.E(err))
				errCh <- err
			}
		}(hook)
	}

	return &wg, errCh
}

// runPostShutdownHooks executes post-shutdown hooks sequentially in registration order.
func (s *Server) runPostShutdownHooks(ctx context.Context) error {
	s.mu.RLock()
	hooks := s.postShutdownHooks
	s.mu.RUnlock()

	if len(hooks) == 0 {
		return nil
	}

	s.logger.Debug("Running post-shutdown hooks", log.F("count", len(hooks)))

	for _, hook := range hooks {
		select {
		case <-ctx.Done():
			s.logger.Warn("Post-shutdown hook aborted due to context cancellation", log.F("hook", hook.Name))
			return ctx.Err()
		default:
		}

		s.logger.Debug("Running post-shutdown hook", log.F("hook", hook.Name))
		if err := hook.Hook(ctx); err != nil {
			s.logger.Error("Post-shutdown hook failed", log.F("hook", hook.Name), log.E(err))
			// Continue with other hooks despite error
		}
	}

	return nil
}
