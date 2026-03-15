package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	zhlog "github.com/alexferl/zerohttp/log"
)

func main() {
	app := zh.New(
		config.Config{
			Lifecycle: config.LifecycleConfig{
				// Pre-startup: run before servers start and before startup hooks
				PreStartupHooks: []config.StartupHookConfig{
					{
						Name: "validate-config",
						Hook: func(ctx context.Context) error {
							log.Println("Validating configuration...")
							return nil
						},
					},
				},
				// Startup: run concurrently with servers starting up
				StartupHooks: []config.StartupHookConfig{
					{
						Name: "warmup-cache",
						Hook: func(ctx context.Context) error {
							log.Println("Warming up cache...")
							time.Sleep(100 * time.Millisecond)
							return nil
						},
					},
				},
				// Post-startup: run after servers have started accepting connections
				PostStartupHooks: []config.StartupHookConfig{
					{
						Name: "announce-ready",
						Hook: func(ctx context.Context) error {
							log.Println("Server is ready!")
							return nil
						},
					},
				},
				// Pre-shutdown: run before server shutdown begins, before servers stop
				PreShutdownHooks: []config.ShutdownHookConfig{
					{
						Name: "health",
						Hook: func(ctx context.Context) error {
							log.Println("Marking service unhealthy...")
							return nil
						},
					},
				},
				// Shutdown: run concurrently with server shutdown
				ShutdownHooks: []config.ShutdownHookConfig{
					{
						Name: "flush-logs",
						Hook: func(ctx context.Context) error {
							log.Println("Flushing logs...")
							time.Sleep(100 * time.Millisecond)
							return nil
						},
					},
					{
						Name: "close-connections",
						Hook: func(ctx context.Context) error {
							log.Println("Closing connections...")
							time.Sleep(100 * time.Millisecond)
							return nil
						},
					},
				},
				// Post-shutdown: run after servers have shut down
				PostShutdownHooks: []config.ShutdownHookConfig{
					{
						Name: "cleanup",
						Hook: func(ctx context.Context) error {
							log.Println("Cleaning up...")
							return nil
						},
					},
				},
			},
		},
	)

	// Hooks can also be registered programmatically
	app.RegisterShutdownHook("metrics-flush", func(ctx context.Context) error {
		return nil
	})

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, 200, zh.M{"message": "Hello, World!"})
	}))

	go func() {
		if err := app.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.Logger().Fatal("Server failed to start", zhlog.E(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		app.Logger().Fatal("Server forced to shutdown", zhlog.E(err))
	}
}
