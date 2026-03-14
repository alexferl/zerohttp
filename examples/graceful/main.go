package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/log"
)

func main() {
	app := zh.New(
		config.Config{
			Lifecycle: config.LifecycleConfig{
				// Pre-shutdown: mark service as unhealthy first
				PreShutdownHooks: []config.ShutdownHookConfig{
					{
						Name: "health",
						Hook: func(ctx context.Context) error {
							// In a real app, this would update a health check endpoint
							return nil
						},
					},
				},
				// During shutdown: close resources concurrently with server shutdown
				ShutdownHooks: []config.ShutdownHookConfig{
					{
						Name: "flush-logs",
						Hook: func(ctx context.Context) error {
							// Simulate log flush
							time.Sleep(100 * time.Millisecond)
							return nil
						},
					},
					{
						Name: "close-connections",
						Hook: func(ctx context.Context) error {
							// Simulate DB connection close
							time.Sleep(100 * time.Millisecond)
							return nil
						},
					},
				},
				// Post-shutdown: final cleanup after servers are stopped
				PostShutdownHooks: []config.ShutdownHookConfig{
					{
						Name: "cleanup",
						Hook: func(ctx context.Context) error {
							// Cleanup temporary files
							return nil
						},
					},
				},
			},
		},
	)

	// Hooks can also be registered programmatically
	app.RegisterShutdownHook("metrics-flush", func(ctx context.Context) error {
		// Flush metrics to external system
		return nil
	})

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, 200, zh.M{"message": "Hello, World!"})
	}))

	go func() {
		if err := app.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.Logger().Fatal("Server failed to start", log.E(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit

	app.Logger().Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		app.Logger().Fatal("Server forced to shutdown", log.E(err))
	}

	app.Logger().Info("Server shutdown complete")
}
