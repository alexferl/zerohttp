package main

import (
	"fmt"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware/securityheaders"
)

func main() {
	app := zh.New()

	// Configure security headers with CSP nonce generation
	// The {{nonce}} placeholder will be replaced with a unique nonce per request
	app.Use(securityheaders.New(securityheaders.Config{
		ContentSecurityPolicyNonceEnabled: true,
		ContentSecurityPolicy: "default-src 'self'; " +
			"script-src 'nonce-{{nonce}}' 'strict-dynamic'; " +
			"style-src 'nonce-{{nonce}}' 'self'; " +
			"img-src 'self' data:; " +
			"font-src 'self'; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self';",
	}))

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		// Get the nonce from the request context
		nonce := securityheaders.GetCSPNonce(r)

		html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>CSP Nonce Demo</title>
    <!-- This inline style is allowed because it has the correct nonce -->
    <style nonce="%[1]s">
        body { font-family: sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .success { color: green; }
        .nonce { background: #f0f0f0; padding: 10px; border-radius: 4px; word-break: break-all; }
    </style>
</head>
<body>
    <h1>Content Security Policy Nonce Demo</h1>

    <h2>Current CSP Nonce:</h2>
    <div class="nonce">%[1]s</div>

    <h2>Status:</h2>
    <p id="status">Checking if inline scripts work...</p>

    <!-- This inline script is allowed because it has the correct nonce -->
    <script nonce="%[1]s">
        document.getElementById('status').innerHTML =
            '<span class="success">✓ Inline scripts are working! The CSP nonce is valid.</span>';
        console.log('CSP nonce used:', '%[1]s');
    </script>

    <!-- This script block also works with the same nonce -->
    <script nonce="%[1]s">
        console.log('Multiple script blocks work with the same nonce');
    </script>
</body>
</html>`, nonce)

		return zh.Render.HTML(w, http.StatusOK, html)
	}))

	// API route that returns the current nonce (for debugging)
	app.GET("/api/nonce", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		nonce := securityheaders.GetCSPNonce(r)
		return zh.Render.JSON(w, http.StatusOK, zh.M{
			"nonce":   nonce,
			"message": "This nonce is unique per request and valid for the current page load",
		})
	}))

	log.Fatal(app.Start())
}
