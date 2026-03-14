package zerohttp

import (
	"context"
	"fmt"
	"sync"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/log"
)

// RegisterStartupHook registers a hook to run before the server starts accepting connections.
// Startup hooks execute sequentially in registration order.
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

// RegisterPreShutdownHook registers a hook to run before server shutdown begins.
// Pre-shutdown hooks execute sequentially in registration order.
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

// RegisterPostShutdownHook registers a hook to run after servers are shut down.
// Post-shutdown hooks execute sequentially in registration order.
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
