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

	fmt.Println("Idempotency Example Server")
	fmt.Println("==========================")
	fmt.Println()
	fmt.Println("Test commands:")
	fmt.Println()
	fmt.Println("1. Create payment with idempotency key (first request - cached):")
	fmt.Println("   curl -i -X POST http://localhost:8080/api/payments \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println("     -H 'Idempotency-Key: key-123' \\")
	fmt.Println(`     -d '{"amount":100}'`)
	fmt.Println()
	fmt.Println("2. Same payment with same key (returns cached response):")
	fmt.Println("   curl -i -X POST http://localhost:8080/api/payments \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println("     -H 'Idempotency-Key: key-123' \\")
	fmt.Println(`     -d '{"amount":100}'`)
	fmt.Println("   Notice: Same payment ID returned, X-Idempotency-Replay: true header")
	fmt.Println()
	fmt.Println("3. Different body with same key (not replayed - body differs):")
	fmt.Println("   curl -i -X POST http://localhost:8080/api/payments \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println("     -H 'Idempotency-Key: key-123' \\")
	fmt.Println(`     -d '{"amount":200}'`)
	fmt.Println()
	fmt.Println("4. Required idempotency key (fails without key):")
	fmt.Println("   curl -i -X POST http://localhost:8080/api/transfers \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println(`     -d '{"amount":500}'`)
	fmt.Println()
	fmt.Println("5. Required idempotency key (succeeds with key):")
	fmt.Println("   curl -i -X POST http://localhost:8080/api/transfers \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println("     -H 'Idempotency-Key: transfer-456' \\")
	fmt.Println(`     -d '{"amount":500}'`)
	fmt.Println()
	fmt.Println("6. Exempt path (webhook - no idempotency check):")
	fmt.Println("   curl -i -X POST http://localhost:8080/api/webhooks \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println("     -H 'Idempotency-Key: webhook-789' \\")
	fmt.Println(`     -d '{"event":"payment.received"}'`)
	fmt.Println()
	fmt.Println("7. Large body (exceeds MaxBodySize, not cached):")
	fmt.Println("   curl -i -X POST http://localhost:8080/api/bulk-import \\")
	fmt.Println("     -H 'Content-Type: application/json' \\")
	fmt.Println("     -H 'Idempotency-Key: bulk-001' \\")
	fmt.Println(`     -d '{"data":"'$(head -c 2000 < /dev/zero | tr '\0' 'a')'"}'`)
	fmt.Println()
	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println()

	log.Fatal(app.Start())
}
