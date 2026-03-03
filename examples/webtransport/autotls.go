//go:build ignore

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
	"github.com/alexferl/zerohttp/config"
	"github.com/quic-go/quic-go/http3"
	webtransport "github.com/quic-go/webtransport-go"
	"golang.org/x/crypto/acme/autocert"
)

var (
	_ config.WebTransportServer             = (*webtransportAutocertServer)(nil)
	_ config.WebTransportServerWithAutocert = (*webtransportAutocertServer)(nil)
)

// webtransportAutocertServer wraps quic-go's webtransport.Server to implement
// config.WebTransportServerWithAutocert interface
type webtransportAutocertServer struct {
	server *webtransport.Server
}

func (w *webtransportAutocertServer) ListenAndServeTLS(certFile, keyFile string) error {
	return w.server.ListenAndServeTLS(certFile, keyFile)
}

func (w *webtransportAutocertServer) Close() error {
	return w.server.Close()
}

func (w *webtransportAutocertServer) ListenAndServeTLSWithAutocert(manager config.AutocertManager) error {
	// Configure TLS with autocert on the underlying HTTP/3 server
	w.server.H3.TLSConfig = &tls.Config{
		GetCertificate: manager.GetCertificate,
		NextProtos:     []string{"h3"},
	}

	// ListenAndServe on the HTTP/3 server (WebTransport runs over it)
	err := w.server.H3.ListenAndServe()
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

	// Create zerohttp app with autocert manager
	app := zh.New(
		config.WithDisableDefaultMiddlewares(),
		config.WithAddr(":80"),     // HTTP port for ACME challenges
		config.WithTLSAddr(":443"), // HTTPS port
		config.WithAutocertManager(mgr),
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
		w.Header().Set("Content-Type", "text/html")
		w.Header().Add("Alt-Svc", `h3=":443"; ma=86400`)
		w.Write([]byte(clientHTML))
		return nil
	}))

	// WebTransport endpoint - register CONNECT handler
	app.CONNECT("/wt", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := wt.Upgrade(w, r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		go handleSession(sess)
	}))

	log.Printf("Starting WebTransport server with AutoTLS on https://%s", *domain)
	log.Println("Ports 80 and 443 must be open and accessible from the internet")

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

const clientHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>WebTransport with AutoTLS</title>
    <style>
        body { font-family: system-ui, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; background: #f5f5f5; }
        h1 { color: #333; }
        .container { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-bottom: 20px; }
        button { background: #0066cc; color: white; border: none; padding: 10px 20px; border-radius: 4px; cursor: pointer; margin: 5px; }
        button:hover { background: #0052a3; }
        button:disabled { background: #ccc; cursor: not-allowed; }
        input[type="text"] { padding: 10px; border: 1px solid #ddd; border-radius: 4px; width: 300px; margin-right: 10px; }
        #log { background: #1a1a1a; color: #00ff00; padding: 15px; border-radius: 4px; font-family: monospace; font-size: 13px; height: 300px; overflow-y: auto; white-space: pre-wrap; }
        .status { display: inline-block; padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: bold; }
        .status.connected { background: #4caf50; color: white; }
        .status.disconnected { background: #f44336; color: white; }
        .status.connecting { background: #ff9800; color: white; }
        .section { margin: 15px 0; padding: 15px; background: #f9f9f9; border-radius: 4px; }
        .section h3 { margin-top: 0; color: #555; }
        .info { background: #e3f2fd; padding: 15px; border-radius: 4px; margin-bottom: 20px; }
    </style>
</head>
<body>
    <h1>WebTransport with AutoTLS</h1>

    <div class="info">
        <strong>AutoTLS Enabled:</strong> This server uses Let's Encrypt for automatic certificate management.
        The certificate is obtained and renewed automatically.
    </div>

    <div class="container">
        <h2>Connection</h2>
        <p>Status: <span id="status" class="status disconnected">Disconnected</span></p>
        <button id="connectBtn" onclick="connect()">Connect</button>
        <button id="disconnectBtn" onclick="disconnect()" disabled>Disconnect</button>
    </div>

    <div class="container">
        <h2>Messaging</h2>
        <div class="section">
            <h3>Datagrams (Unreliable)</h3>
            <input type="text" id="datagramInput" placeholder="Enter message..." disabled>
            <button id="sendDatagramBtn" onclick="sendDatagram()" disabled>Send Datagram</button>
        </div>
        <div class="section">
            <h3>Bidirectional Streams</h3>
            <input type="text" id="streamInput" placeholder="Enter message..." disabled>
            <button id="sendStreamBtn" onclick="sendStreamMessage()" disabled>Send on Stream</button>
            <button id="createStreamBtn" onclick="createStream()" disabled>Create New Stream</button>
        </div>
    </div>

    <div class="container">
        <h2>Log</h2>
        <div id="log"></div>
        <button onclick="clearLog()">Clear Log</button>
    </div>

    <script>
        let wt = null;
        let currentStream = null;
        const logEl = document.getElementById('log');
        const statusEl = document.getElementById('status');

        function log(msg) {
            const timestamp = new Date().toLocaleTimeString();
            logEl.textContent += '[' + timestamp + '] ' + msg + '\n';
            logEl.scrollTop = logEl.scrollHeight;
        }

        function setStatus(status) {
            statusEl.className = 'status ' + status;
            statusEl.textContent = status.charAt(0).toUpperCase() + status.slice(1);
        }

        function updateUI(connected) {
            document.getElementById('connectBtn').disabled = connected;
            document.getElementById('disconnectBtn').disabled = !connected;
            document.getElementById('datagramInput').disabled = !connected;
            document.getElementById('sendDatagramBtn').disabled = !connected;
            document.getElementById('streamInput').disabled = !connected;
            document.getElementById('sendStreamBtn').disabled = !connected;
            document.getElementById('createStreamBtn').disabled = !connected;
        }

        async function connect() {
            try {
                setStatus('connecting');
                log('Connecting to WebTransport server with AutoTLS...');

                if (typeof WebTransport === 'undefined') {
                    throw new Error('WebTransport is not supported in this browser');
                }

                wt = new WebTransport('https://' + window.location.host + '/wt');

                wt.ready.then(() => {
                    log('Connected successfully! (AutoTLS certificate)');
                    setStatus('connected');
                    updateUI(true);
                    startReceiving();
                });

                wt.closed.then((info) => {
                    log('Connection closed: ' + JSON.stringify(info));
                    setStatus('disconnected');
                    updateUI(false);
                    wt = null;
                    currentStream = null;
                });

                wt.ready.catch((error) => {
                    log('Connection failed: ' + error.message);
                    setStatus('disconnected');
                    updateUI(false);
                });

            } catch (error) {
                log('Error: ' + error.message);
                setStatus('disconnected');
                updateUI(false);
            }
        }

        async function disconnect() {
            if (wt) {
                log('Closing connection...');
                await wt.close();
            }
        }

        async function startReceiving() {
            if (wt.datagrams) {
                const datagramReader = wt.datagrams.readable.getReader();
                try {
                    while (true) {
                        const { value, done } = await datagramReader.read();
                        if (done) break;
                        const text = new TextDecoder().decode(value);
                        log('Received datagram: ' + text);
                    }
                } catch (e) {
                    log('Datagram error: ' + e.message);
                }
            }

            try {
                for await (const stream of wt.incomingBidirectionalStreams) {
                    log('New incoming bidirectional stream');
                    handleStream(stream);
                }
            } catch (e) {
                log('Stream error: ' + e.message);
            }
        }

        async function handleStream(stream) {
            const reader = stream.readable.getReader();
            try {
                while (true) {
                    const { value, done } = await reader.read();
                    if (done) break;
                    const text = new TextDecoder().decode(value);
                    log('Received on stream: ' + text);
                }
            } catch (e) {
                log('Stream read error: ' + e.message);
            }
        }

        async function sendDatagram() {
            if (!wt || !wt.datagrams) {
                log('Datagrams not available');
                return;
            }

            const input = document.getElementById('datagramInput');
            const message = input.value.trim();
            if (!message) return;

            try {
                const writer = wt.datagrams.writable.getWriter();
                const data = new TextEncoder().encode(message);
                await writer.write(data);
                writer.releaseLock();
                log('Sent datagram: ' + message);
                input.value = '';
            } catch (e) {
                log('Send datagram error: ' + e.message);
            }
        }

        async function createStream() {
            if (!wt) {
                log('Not connected');
                return;
            }

            try {
                currentStream = await wt.createBidirectionalStream();
                log('Created new bidirectional stream');
                handleStream(currentStream);
            } catch (e) {
                log('Create stream error: ' + e.message);
            }
        }

        async function sendStreamMessage() {
            if (!wt) {
                log('Not connected');
                return;
            }

            if (!currentStream) {
                await createStream();
            }

            const input = document.getElementById('streamInput');
            const message = input.value.trim();
            if (!message) return;

            try {
                const writer = currentStream.writable.getWriter();
                const data = new TextEncoder().encode(message + '\n');
                await writer.write(data);
                writer.releaseLock();
                log('Sent on stream: ' + message);
                input.value = '';
            } catch (e) {
                log('Send stream error: ' + e.message);
            }
        }

        function clearLog() {
            logEl.textContent = '';
        }

        document.getElementById('datagramInput').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') sendDatagram();
        });
        document.getElementById('streamInput').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') sendStreamMessage();
        });

        log('Page loaded. Click "Connect" to start.');
    </script>
</body>
</html>`
