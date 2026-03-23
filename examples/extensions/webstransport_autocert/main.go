package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	zautocert "github.com/alexferl/zerohttp/extensions/autocert"
	zwebtransport "github.com/alexferl/zerohttp/extensions/webtransport"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
	"golang.org/x/crypto/acme/autocert"
)

var (
	_ zwebtransport.Server             = (*webtransportAutocertServer)(nil)
	_ zwebtransport.ServerWithAutocert = (*webtransportAutocertServer)(nil)
	_ zautocert.Manager                = (*autocertManagerWrapper)(nil)
)

// autocertManagerWrapper wraps golangacme.Manager to implement zautocert.Manager
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

// webtransportAutocertServer wraps quic-go's webtransport.Server to implement
// zwebtransport.ServerWithAutocert interface
type webtransportAutocertServer struct {
	server *webtransport.Server
}

func (w *webtransportAutocertServer) ListenAndServeTLS(certFile, keyFile string) error {
	return w.server.ListenAndServeTLS(certFile, keyFile)
}

func (w *webtransportAutocertServer) Close() error {
	return w.server.Close()
}

func (w *webtransportAutocertServer) ListenAndServeTLSWithAutocert(manager zautocert.Manager) error {
	// Configure TLS with autocert on the underlying HTTP/3 server
	w.server.H3.TLSConfig = &tls.Config{
		GetCertificate: manager.GetCertificate,
		NextProtos:     []string{"h3"},
	}

	// webtransport.Server.ListenAndServe uses the H3's TLSConfig
	err := w.server.ListenAndServe()
	if err != nil {
		log.Printf("[ERROR] WebTransport server failed: %v", err)
	}
	return err
}

func main() {
	domain := flag.String("domain", "", "Domain name for Let's Encrypt certificate (required)")
	flag.Parse()

	if *domain == "" {
		log.Fatal("Please provide a domain name with -domain flag")
	}

	// Create autocert manager for Let's Encrypt
	mgr := &autocert.Manager{
		Cache:      autocert.DirCache("/var/cache/certs"),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(*domain),
	}

	// Wrap the manager to implement zautocert.Manager
	wrappedMgr := &autocertManagerWrapper{
		mgr:       mgr,
		hostnames: []string{*domain},
	}

	// Create zerohttp app with autocert manager
	app := zh.New(
		zh.Config{
			DisableDefaultMiddlewares: true,
			Addr:                      ":80", // HTTP port for ACME challenges
			TLS: zh.TLSConfig{
				Addr: ":443", // HTTPS port
			},
			Extensions: zh.ExtensionsConfig{
				AutocertManager: wrappedMgr,
			},
		},
	)

	// Create HTTP/3 server
	h3 := &http3.Server{
		Addr:    ":443",
		Handler: app,
	}

	// Create WebTransport server
	wt := &webtransport.Server{
		H3:          h3,
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// Wire WebTransport into HTTP/3 server
	webtransport.ConfigureHTTP3Server(h3)

	// Wrap the server to implement WebTransportServerWithAutocert
	wtServer := &webtransportAutocertServer{server: wt}

	// Set WebTransport server - zerohttp will start it automatically
	app.SetWebTransportServer(wtServer)

	// Serve the HTML client
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Add("Alt-Svc", `h3=":443"; ma=86400`)
		return zh.R.File(w, r, "static/index.html")
	}))

	// WebTransport endpoint - register CONNECT handler
	app.CONNECT("/wt", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		sess, err := wt.Upgrade(w, r)
		if err != nil {
			return err
		}
		go handleSession(sess)
		return nil
	}))

	// Start with AutoTLS - WebTransport starts automatically with Let's Encrypt!
	log.Fatal(app.StartAutoTLS())
}

func handleSession(sess *webtransport.Session) {
	defer sess.CloseWithError(0, "done")

	log.Printf("WebTransport session from %s", sess.RemoteAddr())

	// Handle datagrams
	go func() {
		for {
			msg, err := sess.ReceiveDatagram(context.Background())
			if err != nil {
				return
			}
			sess.SendDatagram(append([]byte("Echo: "), msg...))
		}
	}()

	// Handle streams
	for {
		stream, err := sess.AcceptStream(context.Background())
		if err != nil {
			return
		}
		go func(str *webtransport.Stream) {
			defer str.Close()
			buf := make([]byte, 1024)
			for {
				n, err := str.Read(buf)
				if n > 0 {
					msg := string(buf[:n])
					response := fmt.Sprintf("[%s] Echo: %s", time.Now().Format("15:04:05"), msg)
					str.Write([]byte(response))
				}
				if err != nil {
					return
				}
			}
		}(stream)
	}
}
