package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/alexferl/zerohttp/middleware/hmacauth"
)

func main() {
	// Credentials
	accessKey := "service-a"
	secretKey := "super-secret-key-at-least-32-bytes-long!!"

	// Create signer
	signer := hmacauth.NewSigner(accessKey, secretKey)

	// Example 1: Request without authentication (fails)
	fmt.Println("1. Request WITHOUT authentication:")
	makeRequest("http://localhost:8080/api/data", nil)

	// Example 2: Request with authentication (succeeds)
	fmt.Println("\n2. Request WITH authentication:")
	makeRequest("http://localhost:8080/api/data", signer)

	// Example 3: Presigned URL (valid for 5 minutes)
	fmt.Println("\n3. Presigned URL request:")
	downloadURL, _ := url.Parse("http://localhost:8080/api/download")
	req, _ := http.NewRequest(http.MethodGet, downloadURL.String(), nil)
	presignedURL, err := signer.PresignURL(req, 5*time.Minute)
	if err != nil {
		fmt.Printf("  Error presigning: %v\n", err)
		return
	}
	makePresignedRequest(presignedURL)
}

func makeRequest(url string, signer *hmacauth.Signer) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}

	if signer != nil {
		if err := signer.SignRequest(req); err != nil {
			fmt.Printf("  Error signing: %v\n", err)
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
	fmt.Printf("  Status: %d\n  Response: %s\n", resp.StatusCode, string(body))
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
	fmt.Printf("  Status: %d\n  Response: %s\n", resp.StatusCode, string(body))
}
