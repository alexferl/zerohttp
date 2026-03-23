package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/sse"
)

func main() {
	customServer := &http.Server{
		Addr:         "localhost:8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // needed for SSE
		IdleTimeout:  120 * time.Second,
	}

	replayer := sse.NewMemoryReplayer(100, 5*time.Minute)
	hub := sse.NewHub()
	broadcastCounter := int32(0)

	app := zh.New(
		zh.Config{
			Server:                    customServer,
			DisableDefaultMiddlewares: true,
			Extensions: zh.ExtensionsConfig{
				SSEProvider: sse.NewDefaultProvider(),
			},
		},
	)

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.File(w, r, "static/index.html")
	}))

	app.GET("/time", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		provider := app.SSEProvider()
		if provider == nil {
			return fmt.Errorf("SSE not configured")
		}

		stream, err := provider.New(w, r)
		if err != nil {
			return err
		}
		defer func() { _ = stream.Close() }()

		_ = stream.SetRetry(5 * time.Second)

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return nil
			case t := <-ticker.C:
				event := sse.Event{
					Name: "time",
					Data: []byte(t.Format(time.RFC3339)),
					ID:   fmt.Sprintf("%d", t.Unix()),
				}
				if err := stream.Send(event); err != nil {
					return nil
				}
			}
		}
	}))

	app.GET("/notifications", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		lastEventID := r.Header.Get("Last-Event-ID")
		if lastEventID == "" {
			lastEventID = r.URL.Query().Get("last_id")
		}

		stream, err := sse.New(w, r)
		if err != nil {
			return err
		}
		defer func() { _ = stream.Close() }()

		if lastEventID != "" {
			count, err := replayer.Replay(lastEventID, func(event sse.Event) error {
				return stream.Send(event)
			})
			if err != nil {
				log.Printf("Replay error: %v", err)
			} else if count > 0 {
				_ = stream.Send(sse.Event{
					Name: "info",
					Data: []byte(fmt.Sprintf("Replayed %d missed events", count)),
				})
			}
		}

		hub.Subscribe(stream, "notifications")
		defer hub.Unsubscribe(stream, "notifications")

		_ = stream.Send(sse.Event{
			Name: "info",
			Data: []byte("Connected!"),
		})

		<-r.Context().Done()
		return nil
	}))

	app.GET("/broadcast", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		stream, err := sse.New(w, r)
		if err != nil {
			return err
		}
		defer func() { _ = stream.Close() }()

		hub.Register(stream)
		defer hub.Unregister(stream)

		_ = stream.Send(sse.Event{
			Name: "info",
			Data: []byte("Subscribed to broadcast channel"),
		})

		<-r.Context().Done()
		return nil
	}))

	app.POST("/notify", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		msg := r.URL.Query().Get("msg")
		if msg == "" {
			msg = "System notification"
		}

		event := sse.Event{
			Name: "notification",
			Data: []byte(msg),
		}
		replayer.Store(event)

		_, err := w.Write([]byte("Notification stored"))
		return err
	}))

	app.POST("/broadcast", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		msg := r.URL.Query().Get("msg")
		if msg == "" {
			msg = fmt.Sprintf("Broadcast #%d", atomic.AddInt32(&broadcastCounter, 1))
		}

		event := sse.Event{
			Name: "broadcast",
			Data: []byte(msg),
		}
		hub.Broadcast(event)

		_, err := fmt.Fprintf(w, "Broadcast sent to %d clients", hub.ConnectionCount())
		return err
	}))

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		counter := 0
		for range ticker.C {
			counter++
			event := sse.Event{
				Name: "auto",
				Data: []byte(fmt.Sprintf("Auto notification #%d", counter)),
			}
			event = replayer.Store(event)
			hub.BroadcastTo("notifications", event)
		}
	}()

	log.Fatal(app.ListenAndServe())
}
