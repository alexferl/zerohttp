package main

import (
	"context"
	"crypto/tls"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/queryparam"
)

// MultipartMaxMemory is the maximum memory to use when parsing multipart
// form data.
var MultipartMaxMemory int64 = 8 * 1024

// Unwrap extracts the underlying HTTP request and response writer from a Huma
// context. If passed a context from a different adapter it will panic.
func Unwrap(ctx huma.Context) (*http.Request, http.ResponseWriter) {
	for {
		if c, ok := ctx.(interface{ Unwrap() huma.Context }); ok {
			ctx = c.Unwrap()
			continue
		}
		break
	}
	if c, ok := ctx.(*zerohttpContext); ok {
		return c.Unwrap()
	}
	panic("not a humazerohttp context")
}

type zerohttpContext struct {
	op     *huma.Operation
	r      *http.Request
	w      http.ResponseWriter
	status int
}

// check that zerohttpContext implements huma.Context
var _ huma.Context = &zerohttpContext{}

func (c *zerohttpContext) Unwrap() (*http.Request, http.ResponseWriter) {
	return c.r, c.w
}

func (c *zerohttpContext) Operation() *huma.Operation {
	return c.op
}

func (c *zerohttpContext) Context() context.Context {
	return c.r.Context()
}

func (c *zerohttpContext) Method() string {
	return c.r.Method
}

func (c *zerohttpContext) Host() string {
	return c.r.Host
}

func (c *zerohttpContext) RemoteAddr() string {
	return c.r.RemoteAddr
}

func (c *zerohttpContext) URL() url.URL {
	return *c.r.URL
}

func (c *zerohttpContext) Param(name string) string {
	v := c.r.PathValue(name)
	if c.r.URL.RawPath == "" {
		return v // RawPath empty means no escaping was done
	}
	u, err := url.PathUnescape(v)
	if err != nil {
		return v // not supposed to happen, but if it does, return the original value
	}
	return u
}

func (c *zerohttpContext) Query(name string) string {
	return queryparam.Get(c.r.URL.RawQuery, name)
}

func (c *zerohttpContext) Header(name string) string {
	return c.r.Header.Get(name)
}

func (c *zerohttpContext) EachHeader(cb func(name, value string)) {
	for name, values := range c.r.Header {
		for _, value := range values {
			cb(name, value)
		}
	}
}

func (c *zerohttpContext) BodyReader() io.Reader {
	return c.r.Body
}

func (c *zerohttpContext) GetMultipartForm() (*multipart.Form, error) {
	err := c.r.ParseMultipartForm(MultipartMaxMemory)
	return c.r.MultipartForm, err
}

func (c *zerohttpContext) SetReadDeadline(deadline time.Time) error {
	return huma.SetReadDeadline(c.w, deadline)
}

func (c *zerohttpContext) SetStatus(code int) {
	c.status = code
	c.w.WriteHeader(code)
}

func (c *zerohttpContext) Status() int {
	return c.status
}

func (c *zerohttpContext) AppendHeader(name string, value string) {
	c.w.Header().Add(name, value)
}

func (c *zerohttpContext) SetHeader(name string, value string) {
	c.w.Header().Set(name, value)
}

func (c *zerohttpContext) BodyWriter() io.Writer {
	return c.w
}

func (c *zerohttpContext) TLS() *tls.ConnectionState {
	return c.r.TLS
}

func (c *zerohttpContext) Version() huma.ProtoVersion {
	return huma.ProtoVersion{
		Proto:      c.r.Proto,
		ProtoMajor: c.r.ProtoMajor,
		ProtoMinor: c.r.ProtoMinor,
	}
}

// NewContext creates a new Huma context from an HTTP request and response.
func NewContext(op *huma.Operation, r *http.Request, w http.ResponseWriter) huma.Context {
	return &zerohttpContext{op: op, r: r, w: w}
}

type zerohttpAdapter struct {
	router zh.Router
}

func (a *zerohttpAdapter) Handle(op *huma.Operation, handler func(huma.Context)) {
	humaHandler := zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		handler(&zerohttpContext{op: op, r: r, w: w})
		return nil
	})

	method := strings.ToUpper(op.Method)
	switch method {
	case "GET":
		a.router.GET(op.Path, humaHandler)
	case "POST":
		a.router.POST(op.Path, humaHandler)
	case "PUT":
		a.router.PUT(op.Path, humaHandler)
	case "DELETE":
		a.router.DELETE(op.Path, humaHandler)
	case "PATCH":
		a.router.PATCH(op.Path, humaHandler)
	case "HEAD":
		a.router.HEAD(op.Path, humaHandler)
	case "OPTIONS":
		a.router.OPTIONS(op.Path, humaHandler)
	}
}

func (a *zerohttpAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

// NewAdapter creates a new adapter for the given zerohttp server.
func NewAdapter(server *zh.Server) huma.Adapter {
	return &zerohttpAdapter{router: server.Router}
}

// New creates a new Huma API using a zerohttp server.
func New(server *zh.Server, config huma.Config) huma.API {
	return huma.NewAPI(config, &zerohttpAdapter{router: server.Router})
}
