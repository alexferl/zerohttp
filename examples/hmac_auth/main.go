// Example: HMAC Request Signing Authentication
//
// This example demonstrates HMAC request signing for machine-to-machine authentication.
// Run with: go run main.go
//
// The server will start on :8080 with three endpoints:
//   - GET /health (no auth required)
//   - GET /api/data (HMAC auth required)
//   - GET /api/download (HMAC auth or presigned URL required)
//
// The client makes requests to demonstrate authentication and presigned URLs.
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	// In production, load these from environment variables or secrets manager
	// Supports multiple secrets per key for rotation (e.g., [oldKey, newKey])
	credentials := map[string][]string{
		"service-a": {"super-secret-key-at-least-32-bytes-long!!"},
		"service-b": {"another-secret-key-for-service-b-abc123"},
	}

	// Create server
	app := zh.New(config.Config{
		Addr: ":8080",
	})

	// Public endpoint - no auth required
	app.GET("/health", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{"status": "healthy"})
	}))

	// Protected API endpoints (header-based auth)
	app.Group(func(api zh.Router) {
		api.Use(middleware.HMACAuth(config.HMACAuthConfig{
			CredentialStore: func(accessKeyID string) []string {
				return credentials[accessKeyID]
			},
			MaxSkew:     5 * time.Minute,
			ExemptPaths: []string{},
		}))

		api.GET("/api/data", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			// Access the authenticated client's access key ID
			accessKeyID := middleware.GetHMACAccessKeyID(r)

			return zh.R.JSON(w, http.StatusOK, zh.M{
				"message":          "Hello from protected API",
				"authenticated_as": accessKeyID,
			})
		}))
	})

	// Protected download endpoint (supports presigned URLs)
	app.Group(func(api zh.Router) {
		api.Use(middleware.HMACAuth(config.HMACAuthConfig{
			CredentialStore: func(accessKeyID string) []string {
				return credentials[accessKeyID]
			},
			MaxSkew:            5 * time.Minute,
			AllowPresignedURLs: true, // Enable presigned URL support
		}))

		api.GET("/api/download", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			accessKeyID := middleware.GetHMACAccessKeyID(r)

			return zh.R.JSON(w, http.StatusOK, zh.M{
				"message":          "Download access granted",
				"authenticated_as": accessKeyID,
				"filename":         "report.pdf",
				"size_bytes":       1024567,
			})
		}))
	})

	// Start server in background
	go func() {
		fmt.Println("Server starting on http://localhost:8080")
		fmt.Println("Endpoints:")
		fmt.Println("  GET /health        - No authentication required")
		fmt.Println("  GET /api/data      - HMAC header authentication required")
		fmt.Println("  GET /api/download  - HMAC header or presigned URL required")
		fmt.Println()
		if err := app.ListenAndServe(); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Run client examples
	fmt.Println("=== Client Examples ===")
	fmt.Println()

	// Example 1: Request without authentication (should fail)
	fmt.Println("1. Request to /api/data WITHOUT authentication:")
	makeRequest("http://localhost:8080/api/data", nil)

	// Example 2: Request with authentication (should succeed)
	fmt.Println("\n2. Request to /api/data WITH authentication:")
	signer := middleware.NewHMACSigner("service-a", credentials["service-a"][0])
	makeRequest("http://localhost:8080/api/data", signer)

	// Example 3: Request to public endpoint (should succeed without auth)
	fmt.Println("\n3. Request to /health (no auth required):")
	makeRequest("http://localhost:8080/health", nil)

	// Example 4: Request with wrong secret (should fail)
	fmt.Println("\n4. Request with WRONG secret:")
	badSigner := middleware.NewHMACSigner("service-a", "wrong-secret")
	makeRequest("http://localhost:8080/api/data", badSigner)

	// Example 5: Request with different algorithm (SHA512)
	fmt.Println("\n5. Request using HMAC-SHA512:")
	sha512Signer := middleware.NewHMACSignerWithAlgorithm(
		"service-a",
		credentials["service-a"][0],
		config.HMACSHA512,
	)
	// Note: This will fail because server defaults to SHA256
	// To make it work, configure server with Algorithm: config.HMACSHA512
	makeRequest("http://localhost:8080/api/data", sha512Signer)

	// Example 6: Presigned URL (valid for 5 minutes)
	fmt.Println("\n6. Request using presigned URL (valid for 5 minutes):")
	downloadURL, _ := url.Parse("http://localhost:8080/api/download")
	req, _ := http.NewRequest(http.MethodGet, downloadURL.String(), nil)
	_, err := signer.PresignURL(req, 5*time.Minute)
	if err != nil {
		fmt.Printf("  Error creating presigned URL: %v\n", err)
	} else {
		makePresignedRequest(req.URL.String())
	}

	// Example 7: Expired presigned URL (should fail)
	fmt.Println("\n7. Request using expired presigned URL (should fail):")
	expiredReq, _ := http.NewRequest(http.MethodGet, downloadURL.String(), nil)
	// Create a URL that expired 1 hour ago
	_, err = signer.PresignURLWithTime(expiredReq, time.Now().UTC().Add(-1*time.Hour))
	if err != nil {
		fmt.Printf("  Error creating presigned URL: %v\n", err)
	} else {
		makePresignedRequest(expiredReq.URL.String())
	}
}

func makeRequest(url string, signer *middleware.HMACSigner) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Printf("  Error creating request: %v\n", err)
		return
	}

	if signer != nil {
		if err := signer.SignRequest(req); err != nil {
			fmt.Printf("  Error signing request: %v\n", err)
			return
		}
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("  Status: %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
	fmt.Printf("  Response: %s\n", string(body))
}

func makePresignedRequest(presignedURL string) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(presignedURL)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("  Status: %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
	fmt.Printf("  Response: %s\n", string(body))
}
