package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	zautocert "github.com/alexferl/zerohttp/extensions/autocert"
	zhttp3 "github.com/alexferl/zerohttp/extensions/http3"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/crypto/acme/autocert"
)

var (
	_ zhttp3.Server             = (*http3AutocertServer)(nil)
	_ zhttp3.ServerWithAutocert = (*http3AutocertServer)(nil)
	_ zautocert.Manager         = (*autocertManagerWrapper)(nil)
)

// autocertManagerWrapper wraps autocert.Manager to implement zautocert.Manager
type autocertManagerWrapper struct {
	mgr       *autocert.Manager
	hostnames []string
}

func (a *autocertManagerWrapper) Hostnames() []string {
	return a.hostnames
}

func (a *autocertManagerWrapper) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return a.mgr.GetCertificate(hello)
}

func (a *autocertManagerWrapper) HTTPHandler(fallback http.Handler) http.Handler {
	return a.mgr.HTTPHandler(fallback)
}

// http3AutocertServer wraps quic-go's http3.Server to implement
// zhttp3.ServerWithAutocert interface
type http3AutocertServer struct {
	server *http3.Server
}

func (h *http3AutocertServer) ListenAndServeTLS(certFile, keyFile string) error {
	return h.server.ListenAndServeTLS(certFile, keyFile)
}

func (h *http3AutocertServer) Shutdown(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

func (h *http3AutocertServer) Close() error {
	return nil
}

func (h *http3AutocertServer) ListenAndServeTLSWithAutocert(manager zautocert.Manager) error {
	tlsConfig := &tls.Config{
		GetCertificate: manager.GetCertificate,
		NextProtos:     []string{"h3"},
	}
	h.server.TLSConfig = tlsConfig

	err := h.server.ListenAndServe()
	if err != nil {
		log.Printf("[ERROR] HTTP/3 server failed: %v", err)
	}
	return err
}

func main() {
	domain := flag.String("domain", "", "Domain name for Let's Encrypt certificate (required)")
	flag.Parse()

	if *domain == "" {
		log.Fatal("Please provide a domain name with -domain flag")
	}

	// Create autocert manager for automatic certificates
	manager := &autocert.Manager{
		Cache:      autocert.DirCache("/var/cache/certs"),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(*domain),
	}

	// Wrap the manager to implement zautocert.Manager
	wrappedManager := &autocertManagerWrapper{
		mgr:       manager,
		hostnames: []string{*domain},
	}

	// Create zerohttp server with autocert manager
	app := zh.New(
		zh.Config{
			Addr: ":80",
			TLS: zh.TLSConfig{
				Addr: ":443",
			},
			Extensions: zh.ExtensionsConfig{
				AutocertManager: wrappedManager,
			},
		},
	)

	// Add Alt-Svc header to advertise HTTP/3 support
	app.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add(httpx.HeaderAltSvc, `h3=":443"; ma=86400`)
			next.ServeHTTP(w, r)
		})
	})

	// Add routes
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
		_, err := w.Write([]byte("Hello over HTTP/3!\n"))
		return err
	}))

	// Create HTTP/3 server with autocert support
	h3Server := &http3AutocertServer{
		server: &http3.Server{
			Addr:    ":443",
			Handler: app,
		},
	}
	app.SetHTTP3Server(h3Server)

	// Start server with AutoTLS (HTTP, HTTPS, and HTTP/3)
	// This starts:
	// - HTTP server on :80 (for ACME challenges and redirects)
	// - HTTPS server on :443 (HTTP/1 and HTTP/2 with AutoTLS)
	// - HTTP/3 server on :443 (if HTTP3Server implements HTTP3ServerWithAutocert)
	log.Fatal(app.StartAutoTLS())
}
