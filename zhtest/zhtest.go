package zhtest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/alexferl/zerohttp/httpx"
)

// RequestBuilder provides a fluent interface for building HTTP test requests.
type RequestBuilder struct {
	method  string
	path    string
	headers http.Header
	cookies []*http.Cookie
	body    io.Reader
}

// NewRequest creates a new RequestBuilder with the specified method and path.
// The path can include query parameters which will be parsed automatically.
//
// Example:
//
//	req := zhtest.NewRequest(http.MethodGet, "/users").Build()
//	req := zhtest.NewRequest(http.MethodGet, "/users?page=1").Build()
func NewRequest(method, path string) *RequestBuilder {
	return &RequestBuilder{
		method:  method,
		path:    path,
		headers: make(http.Header),
	}
}

// WithHeader adds a header to the request.
// Multiple calls with the same key will append values.
//
// Example:
//
//	req := zhtest.NewRequest(http.MethodGet, "/").
//	    WithHeader(consts.HeaderAuthorization, "Bearer token").
//	    Build()
func (rb *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	rb.headers.Add(key, value)
	return rb
}

// WithHeaders sets multiple headers at once.
// This replaces any existing headers with the same keys.
//
// Example:
//
//	req := zhtest.NewRequest(http.MethodGet, "/").
//	    WithHeaders(map[string]string{
//	        "Authorization": "Bearer token",
//	        "X-Request-ID":  "abc123",
//	    }).
//	    Build()
func (rb *RequestBuilder) WithHeaders(headers map[string]string) *RequestBuilder {
	for k, v := range headers {
		rb.headers.Set(k, v)
	}
	return rb
}

// WithCookie adds a cookie to the request.
//
// Example:
//
//	req := zhtest.NewRequest(http.MethodGet, "/").
//	    WithCookie(&http.Cookie{Name: "session", Value: "abc123"}).
//	    Build()
func (rb *RequestBuilder) WithCookie(cookie *http.Cookie) *RequestBuilder {
	rb.cookies = append(rb.cookies, cookie)
	return rb
}

// WithQuery adds query parameters to the request.
// These are appended to any query parameters already in the path.
//
// Example:
//
//	req := zhtest.NewRequest(http.MethodGet, "/users").
//	    WithQuery("page", "1").
//	    WithQuery("limit", "10").
//	    Build()
func (rb *RequestBuilder) WithQuery(key, value string) *RequestBuilder {
	// Parse existing query from path
	parsedURL, err := url.Parse(rb.path)
	if err != nil {
		// If path is invalid, just append query string
		if strings.Contains(rb.path, "?") {
			rb.path += "&" + key + "=" + url.QueryEscape(value)
		} else {
			rb.path += "?" + key + "=" + url.QueryEscape(value)
		}
		return rb
	}

	query := parsedURL.Query()
	query.Set(key, value)
	parsedURL.RawQuery = query.Encode()
	rb.path = parsedURL.String()
	return rb
}

// WithBody sets the request body directly.
//
// Example:
//
//	req := zhtest.NewRequest(http.MethodPost, "/upload").
//	    WithBody(strings.NewReader("raw data")).
//	    Build()
func (rb *RequestBuilder) WithBody(body io.Reader) *RequestBuilder {
	rb.body = body
	return rb
}

// WithBytes sets the request body from a byte slice.
//
// Example:
//
//	req := zhtest.NewRequest(http.MethodPost, "/upload").
//	    WithBytes([]byte("raw data")).
//	    Build()
func (rb *RequestBuilder) WithBytes(data []byte) *RequestBuilder {
	rb.body = bytes.NewReader(data)
	return rb
}

// WithJSON sets the request body from a JSON-serializable value
// and sets the Content-Type header to application/json.
//
// Example:
//
//	req := zhtest.NewRequest(http.MethodPost, "/users").
//	    WithJSON(map[string]string{"name": "John"}).
//	    Build()
func (rb *RequestBuilder) WithJSON(v any) *RequestBuilder {
	data, err := json.Marshal(v)
	if err != nil {
		// Store error in body that will fail during request
		rb.body = &errorReader{err: err}
		return rb
	}
	rb.body = bytes.NewReader(data)
	rb.headers.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	return rb
}

// WithForm sets the request body from form values
// and sets the Content-Type header to application/x-www-form-urlencoded.
//
// Example:
//
//	req := zhtest.NewRequest(http.MethodPost, "/login").
//	    WithForm(url.Values{"username": []string{"john"}, "password": []string{"secret"}}).
//	    Build()
func (rb *RequestBuilder) WithForm(values url.Values) *RequestBuilder {
	rb.body = strings.NewReader(values.Encode())
	rb.headers.Set(httpx.HeaderContentType, httpx.MIMEApplicationFormURLEncoded)
	return rb
}

// Build creates the http.Request.
// Returns the built request which can be used with httptest.NewRecorder.
//
// Example:
//
//	req := zhtest.NewRequest(http.MethodGet, "/users").Build()
//	w := httptest.NewRecorder()
//	router.ServeHTTP(w, req)
func (rb *RequestBuilder) Build() *http.Request {
	var body io.Reader
	if rb.body != nil {
		body = rb.body
	}

	req := httptest.NewRequest(rb.method, rb.path, body)

	// Set headers
	for k, v := range rb.headers {
		req.Header[k] = v
	}

	// Set cookies
	for _, c := range rb.cookies {
		req.AddCookie(c)
	}

	return req
}

// Response wraps httptest.ResponseRecorder with additional helper methods.
type Response struct {
	*httptest.ResponseRecorder
}

// NewRecorder creates a new Response wrapper.
//
// Example:
//
//	w := zhtest.NewRecorder()
//	router.ServeHTTP(w, req)
func NewRecorder() *Response {
	return &Response{ResponseRecorder: httptest.NewRecorder()}
}

// BodyString returns the response body as a string.
func (r *Response) BodyString() string {
	return r.Body.String()
}

// BodyBytes returns the response body as a byte slice.
func (r *Response) BodyBytes() []byte {
	return r.Body.Bytes()
}

// JSON decodes the response body as JSON into v.
// Returns an error if the body is not valid JSON.
//
// Example:
//
//	var result User
//	if err := w.JSON(&result); err != nil {
//	    t.Fatal(err)
//	}
func (r *Response) JSON(v any) error {
	return json.Unmarshal(r.Body.Bytes(), v)
}

// Cookie returns the cookie with the given name, or nil if not found.
//
// Example:
//
//	sessionCookie := w.Cookie("session")
//	if sessionCookie == nil {
//	    t.Error("session cookie not found")
//	}
func (r *Response) Cookie(name string) *http.Cookie {
	for _, c := range r.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// CookieValue returns the value of the cookie with the given name,
// or an empty string if not found.
func (r *Response) CookieValue(name string) string {
	if c := r.Cookie(name); c != nil {
		return c.Value
	}
	return ""
}

// HeaderValue returns the first value of the given header,
// or an empty string if the header is not present.
func (r *Response) HeaderValue(key string) string {
	return r.Header().Get(key)
}

// IsSuccess returns true if the status code is between 200 and 299.
func (r *Response) IsSuccess() bool {
	return r.Code >= 200 && r.Code < 300
}

// IsRedirect returns true if the status code is between 300 and 399.
func (r *Response) IsRedirect() bool {
	return r.Code >= 300 && r.Code < 400
}

// IsClientError returns true if the status code is between 400 and 499.
func (r *Response) IsClientError() bool {
	return r.Code >= 400 && r.Code < 500
}

// IsServerError returns true if the status code is between 500 and 599.
func (r *Response) IsServerError() bool {
	return r.Code >= 500 && r.Code < 600
}

// errorReader is used to propagate JSON marshaling errors
type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}
