package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	app := zh.New()

	// Simple payment endpoint with idempotency
	// Demonstrates basic idempotency with in-memory store
	app.POST("/api/payments", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		// In a real app, process the payment here
		paymentID := fmt.Sprintf("pay_%d", time.Now().Unix())

		return zh.R.JSON(w, http.StatusCreated, map[string]any{
			"id":        paymentID,
			"status":    "completed",
			"amount":    100.00,
			"currency":  "USD",
			"createdAt": time.Now().Format(time.RFC3339),
		})
	}),
		middleware.Idempotency(config.IdempotencyConfig{
			TTL:         24 * time.Hour,
			MaxBodySize: 1024 * 1024, // 1MB
		}),
	)

	// Required idempotency - fails if no key provided
	// Good for critical financial operations
	app.POST("/api/transfers", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		transferID := fmt.Sprintf("xfer_%d", time.Now().Unix())

		return zh.R.JSON(w, http.StatusCreated, map[string]any{
			"id":        transferID,
			"status":    "pending",
			"from":      "account_a",
			"to":        "account_b",
			"amount":    500.00,
			"createdAt": time.Now().Format(time.RFC3339),
		})
	}),
		middleware.Idempotency(config.IdempotencyConfig{
			Required:    true, // Must provide Idempotency-Key header
			TTL:         24 * time.Hour,
			MaxBodySize: 1024 * 1024,
		}),
	)

	// Webhook endpoint - exempt from idempotency
	// External systems may retry webhooks with same payload
	app.POST("/api/webhooks", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"status":  "received",
			"message": "Webhook processed",
		})
	}),
		middleware.Idempotency(config.IdempotencyConfig{
			TTL:         1 * time.Hour,
			MaxBodySize: 1024 * 1024,
			ExemptPaths: []string{"/api/webhooks"}, // Skip idempotency for webhooks
		}),
	)

	// Large request body - not cached if exceeds limit
	app.POST("/api/bulk-import", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusAccepted, map[string]any{
			"jobId":       fmt.Sprintf("job_%d", time.Now().Unix()),
			"status":      "queued",
			"recordCount": 10000,
		})
	}),
		middleware.Idempotency(config.IdempotencyConfig{
			TTL:         1 * time.Hour,
			MaxBodySize: 1024, // Only cache small requests (1KB)
		}),
	)

	// Regular GET endpoint - idempotency only applies to state-changing methods
	app.GET("/api/status", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"status": "operational",
		})
	}),
		middleware.Idempotency(config.IdempotencyConfig{
			TTL: 1 * time.Hour,
		}),
	)

	log.Fatal(app.Start())
}
