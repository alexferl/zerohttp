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
	"github.com/alexferl/zerohttp/log"
)

func main() {
	app := zh.New()

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, 200, zh.M{"message": "Hello, World!"})
	}))

	// Fast endpoint - completes before shutdown timeout
	app.GET("/fast", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		time.Sleep(2 * time.Second)
		return zh.R.JSON(w, 200, zh.M{"message": "Fast response completed"})
	}))

	// Slow endpoint - takes longer than shutdown timeout
	// Note: This will be force-terminated if shutdown happens during the request
	app.GET("/slow", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		time.Sleep(10 * time.Second)
		return zh.R.JSON(w, 200, zh.M{"message": "Slow response completed"})
	}))

	go func() {
		if err := app.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.Logger().Fatal("Server failed to start", log.E(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		app.Logger().Fatal("Server forced to shutdown", log.E(err))
	}
}
