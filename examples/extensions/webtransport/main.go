package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/quic-go/quic-go/http3"
	webtransport "github.com/quic-go/webtransport-go"
)

func main() {
	certFile, keyFile := "localhost+2.pem", "localhost+2-key.pem"

	app := zh.New(
		config.Config{
			DisableDefaultMiddlewares: true,
			TLS: config.TLSConfig{
				Addr:     ":8443",
				CertFile: certFile,
				KeyFile:  keyFile,
			},
		},
	)

	h3 := &http3.Server{
		Addr: ":8443",
		TLSConfig: &tls.Config{
			NextProtos: []string{"h3"},
		},
		Handler: app,
	}

	wtServer := &webtransport.Server{
		H3:          h3,
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	webtransport.ConfigureHTTP3Server(h3)
	app.SetWebTransportServer(wtServer)

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Add(httpx.HeaderAltSvc, `h3=":8443"; ma=86400`)
		return zh.R.File(w, r, "static/index.html")
	}))

	app.CONNECT("/wt", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		sess, err := wtServer.Upgrade(w, r)
		if err != nil {
			return err
		}
		go handleSession(sess)
		return nil
	}))

	if err := app.ListenAndServeTLS(certFile, keyFile); err != nil {
		log.Fatal(err)
	}
}

func handleSession(sess *webtransport.Session) {
	defer sess.CloseWithError(0, "done")

	log.Printf("WebTransport session from %s", sess.RemoteAddr())

	go func() {
		for {
			msg, err := sess.ReceiveDatagram(context.Background())
			if err != nil {
				return
			}
			sess.SendDatagram(append([]byte("Echo: "), msg...))
		}
	}()

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
