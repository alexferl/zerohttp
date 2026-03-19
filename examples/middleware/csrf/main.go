package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	app := zh.New(
		config.Config{
			DisableDefaultMiddlewares: true,
		},
	)

	// Add CSRF middleware with form token lookup for traditional HTML forms
	// Using a fixed HMAC key so tokens remain valid across server restarts
	app.Use(middleware.CSRF(config.CSRFConfig{
		TokenLookup:  "form:csrf_token",
		CookieSecure: config.Bool(false), // Disable Secure flag for local HTTP testing
		HMACKey:      []byte("demo-csrf-key-for-local-testing-only!!"),
	}))

	app.GET("/", zh.HandlerFunc(indexHandler))
	app.GET("/form", zh.HandlerFunc(formHandler))
	app.POST("/submit", zh.HandlerFunc(submitHandler))
	app.GET("/api", zh.HandlerFunc(apiDemoHandler))
	app.POST("/api/update", zh.HandlerFunc(apiUpdateHandler))

	log.Fatal(app.Start())
}

func indexHandler(w http.ResponseWriter, r *http.Request) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>CSRF Protection Example</title>
    <style>
        body { font-family: sans-serif; max-width: 900px; margin: 40px auto; padding: 20px; }
        h1 { color: #333; }
        h2 { color: #555; margin-top: 30px; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
        pre { background: #f4f4f4; padding: 15px; border-radius: 5px; overflow-x: auto; }
        .endpoint { margin: 20px 0; padding: 15px; background: #f9f9f9; border-left: 4px solid #007bff; }
        .method { font-weight: bold; color: #007bff; }
        .warning { background: #fff3cd; padding: 15px; border-left: 4px solid #ffc107; margin: 20px 0; }
        .success { background: #d4edda; padding: 15px; border-left: 4px solid #28a745; margin: 20px 0; }
        a { color: #007bff; }
    </style>
</head>
<body>
    <h1>CSRF Protection Example</h1>
    <p>This example demonstrates zerohttp's CSRF middleware using the double-submit cookie pattern.</p>

    <div class="warning">
        <strong>Security Note:</strong> CSRF (Cross-Site Request Forgery) protection prevents malicious websites
        from performing actions on behalf of authenticated users. The double-submit cookie pattern works by:
        <ol>
            <li>Setting a random token in a cookie (HttpOnly, Secure, SameSite)</li>
            <li>Requiring that same token in form fields or headers for state-changing requests</li>
            <li>Validating both tokens match using constant-time comparison</li>
        </ol>
    </div>

    <h2>Interactive Examples</h2>

    <div class="endpoint">
        <a href="/form"><span class="method">GET/POST</span> <code>/form</code></a>
        <p>Traditional HTML form with automatic CSRF token injection. The token is:
        <ul>
            <li>Set as a cookie on page load</li>
            <li>Included in the form as a hidden field</li>
            <li>Validated on form submission</li>
        </ul>
        </p>
    </div>

    <div class="endpoint">
        <a href="/api"><span class="method">GET/POST</span> <code>/api</code></a>
        <p>AJAX/Fetch API demonstration showing how to include CSRF tokens in JavaScript requests.</p>
    </div>

    <h2>Curl Commands</h2>

    <h3>1. Without CSRF Token (Fails)</h3>
    <pre>curl -X POST http://localhost:8080/submit \
  -d "message=hello"</pre>
    <p>Result: <code>403 Forbidden - CSRF token invalid or missing</code></p>

    <h3>2. Get CSRF Token (save cookie jar)</h3>
    <pre>curl -s http://localhost:8080/form -c cookies.txt > /dev/null</pre>
    <p>Saves the CSRF cookie to <code>cookies.txt</code></p>

    <h3>3. Submit with Token (Succeeds)</h3>
    <pre>CSRF_TOKEN=$(grep csrf_token cookies.txt | tail -1 | awk '{print $7}')
curl -X POST http://localhost:8080/submit \
  -b cookies.txt \
  -d "csrf_token=$CSRF_TOKEN" \
  -d "message=hello"</pre>

    <h2>Configuration Options</h2>
    <pre>app.Use(middleware.CSRF(config.CSRFConfig{
    TokenLookup:    "form:csrf_token",
    CookieName:     "csrf_token",
    CookieMaxAge:   86400,
    CookieSecure:   config.Bool(true),
    CookieSameSite: http.SameSiteStrictMode,
    ExcludedPaths:    []string{"/api/webhook"},
}))</pre>

    <h2>Token Lookup Methods</h2>
    <ul>
        <li><code>header:X-CSRF-Token</code> - Default, for AJAX requests</li>
        <li><code>form:csrf_token</code> - For traditional HTML forms</li>
    </ul>
</body>
</html>`
	return zh.R.HTML(w, http.StatusOK, html)
}

func formHandler(w http.ResponseWriter, r *http.Request) error {
	// Get CSRF token from context (set by middleware)
	csrfToken := middleware.GetCSRFToken(r)

	html := `<!DOCTYPE html>
<html>
<head>
    <title>CSRF Protected Form</title>
    <style>
        body { font-family: sans-serif; max-width: 600px; margin: 40px auto; padding: 20px; }
        label { display: block; margin: 15px 0 5px; }
        input, textarea { width: 100%; padding: 8px; box-sizing: border-box; }
        button { margin-top: 20px; padding: 10px 20px; background: #007bff; color: white; border: none; cursor: pointer; }
        button:hover { background: #0056b3; }
        .info { background: #e7f3ff; padding: 15px; border-left: 4px solid #007bff; margin: 20px 0; }
        .token { background: #f4f4f4; padding: 10px; word-break: break-all; font-family: monospace; font-size: 12px; }
        a { color: #007bff; }
    </style>
</head>
<body>
    <h1>CSRF Protected Form</h1>
    <p><a href="/">← Back to overview</a></p>

    <div class="info">
        <strong>CSRF Token:</strong>
        <div class="token">` + csrfToken + `</div>
        <p>This token is automatically included in the form below as a hidden field.
        It's also stored in an HttpOnly cookie named <code>csrf_token</code>.</p>
    </div>

    <form method="POST" action="/submit">
        <!-- CSRF Token is automatically injected by middleware via GetCSRFToken -->
        <input type="hidden" name="csrf_token" value="` + csrfToken + `">

        <label>Name: <input type="text" name="name" required></label>
        <label>Email: <input type="email" name="email" required></label>
        <label>Message: <textarea name="message" rows="4" required></textarea></label>
        <button type="submit">Submit</button>
    </form>

    <h2>How it works</h2>
    <ol>
        <li>When you visit this page, the middleware generates a CSRF token</li>
        <li>The token is stored in a cookie (HttpOnly, Secure, SameSite)</li>
        <li>The token is also retrieved via <code>GetCSRFToken(r)</code> and embedded in the form</li>
        <li>On submission, both cookie and form field are compared</li>
        <li>If they match, the request is processed; otherwise, 403 Forbidden</li>
    </ol>
</body>
</html>`
	return zh.R.HTML(w, http.StatusOK, html)
}

func submitHandler(w http.ResponseWriter, r *http.Request) error {
	// If we get here, CSRF validation passed
	name := r.FormValue("name")
	email := r.FormValue("email")
	message := r.FormValue("message")

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"success": true,
		"message": "Form submitted successfully (CSRF token validated)",
		"data": zh.M{
			"name":    name,
			"email":   email,
			"message": message,
		},
	})
}

func apiDemoHandler(w http.ResponseWriter, r *http.Request) error {
	// Get CSRF token from context (set by middleware)
	csrfToken := middleware.GetCSRFToken(r)

	html := `<!DOCTYPE html>
<html>
<head>
    <title>AJAX CSRF Demo</title>
    <style>
        body { font-family: sans-serif; max-width: 700px; margin: 40px auto; padding: 20px; }
        button { padding: 10px 20px; margin: 10px 10px 10px 0; cursor: pointer; }
        .success { background: #d4edda; padding: 15px; border-left: 4px solid #28a745; }
        .error { background: #f8d7da; padding: 15px; border-left: 4px solid #dc3545; }
        .info { background: #e7f3ff; padding: 15px; border-left: 4px solid #007bff; }
        pre { background: #f4f4f4; padding: 15px; border-radius: 5px; overflow-x: auto; }
        a { color: #007bff; }
        #result { margin-top: 20px; }
        .token-box { background: #f4f4f4; padding: 10px; word-break: break-all; font-family: monospace; font-size: 12px; margin: 10px 0; }
    </style>
</head>
<body>
    <h1>AJAX CSRF Token Demo</h1>
    <p><a href="/">← Back to overview</a></p>

    <div class="info">
        This demo shows how to properly include CSRF tokens in AJAX requests.
        The CSRF token is embedded in the page since the cookie is HttpOnly.
        Open the browser's Developer Tools (F12) to see the network requests.
    </div>

    <h2>Current CSRF Token</h2>
    <div class="token-box">` + csrfToken + `</div>
    <p><small>This token was embedded by the server. The cookie is HttpOnly, so JavaScript cannot read it directly.</small></p>

    <h2>Test API Calls</h2>
    <button onclick="testWithoutToken()">POST without CSRF Token</button>
    <button onclick="testWithToken()">POST with CSRF Token</button>

    <div id="result"></div>

    <h2>JavaScript Code</h2>
    <pre>// CSRF token embedded by server (cookie is HttpOnly)
const CSRF_TOKEN = '` + csrfToken + `';

// Send as form data (configured with form:csrf_token lookup)
const formData = new FormData();
formData.append('csrf_token', CSRF_TOKEN);
formData.append('message', 'hello');

fetch('/api/update', {
    method: 'POST',
    credentials: 'same-origin',
    body: formData
})</pre>

    <script>
        // CSRF token embedded by server
        const CSRF_TOKEN = '` + csrfToken + `';

        function showResult(html, isError) {
            const result = document.getElementById('result');
            result.className = isError ? 'error' : 'success';
            result.innerHTML = html;
        }

        async function testWithoutToken() {
            try {
                const formData = new FormData();
                formData.append('message', 'hello');

                const response = await fetch('/api/update', {
                    method: 'POST',
                    credentials: 'same-origin',
                    body: formData
                });

                const text = await response.text();
                showResult(
                    '<strong>Status:</strong> ' + response.status + ' ' + response.statusText +
                    '<br><strong>Response:</strong> ' + text,
                    !response.ok
                );
            } catch (err) {
                showResult('<strong>Error:</strong> ' + err.message, true);
            }
        }

        async function testWithToken() {
            try {
                const formData = new FormData();
                formData.append('csrf_token', CSRF_TOKEN);
                formData.append('message', 'hello from AJAX');

                const response = await fetch('/api/update', {
                    method: 'POST',
                    credentials: 'same-origin',
                    body: formData
                });

                const text = await response.text();
                let displayText = text;
                try {
                    const json = JSON.parse(text);
                    displayText = '<pre>' + JSON.stringify(json, null, 2) + '</pre>';
                } catch (e) {
                    // Not JSON, display as-is
                }
                showResult(
                    '<strong>Status:</strong> ' + response.status + ' ' + response.statusText +
                    '<br><strong>Response:</strong> ' + displayText,
                    !response.ok
                );
            } catch (err) {
                showResult('<strong>Error:</strong> ' + err.message, true);
            }
        }
    </script>
</body>
</html>`
	return zh.R.HTML(w, http.StatusOK, html)
}

func apiUpdateHandler(w http.ResponseWriter, r *http.Request) error {
	// If we get here, CSRF validation passed
	message := r.FormValue("message")
	return zh.R.JSON(w, http.StatusOK, zh.M{
		"success": true,
		"message": "API update successful (CSRF token validated)",
		"data": zh.M{
			"received": message,
		},
		"timestamp": timeNow(),
	})
}

func timeNow() string {
	return "2024-01-01T00:00:00Z" // Simplified for demo
}
