// Package sse provides Server-Sent Events (SSE) support for real-time
// server-to-client streaming using Go's standard library.
//
// # Overview
//
// Server-Sent Events enable servers to push real-time updates to clients over
// a single HTTP connection. Unlike WebSockets, SSE is unidirectional (server to
// client) and works over standard HTTP.
//
// This package provides:
//   - Connection management with automatic cleanup
//   - Broadcast hubs for multi-client scenarios
//   - Event replay for missed message recovery
//   - Spec-compliant event formatting
//
// # Basic Usage
//
// Create an SSE endpoint using the server helper:
//
//	app := zh.New(zh.Config{
//	    SSEProvider: sse.NewDefaultProvider(),
//	})
//
//	app.GET("/events", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
//	    stream, err := sse.New(w, r)
//	    if err != nil {
//	        return err
//	    }
//	    defer stream.Close()
//
//	    for {
//	        select {
//	        case <-r.Context().Done():
//	            return nil
//	        case msg := <-messages:
//	            stream.Send(sse.Event{Name: "message", Data: []byte(msg)})
//	        }
//	    }
//	})
//
// # Broadcasting with Hubs
//
// Use a Hub to manage multiple connections and broadcast to all clients:
//
//	hub := sse.NewHub()
//
//	app.GET("/events", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
//	    stream, err := sse.New(w, r)
//	    if err != nil {
//	        return err
//	    }
//
//	    hub.Register(stream)
//	    defer hub.Unregister(stream)
//
//	    <-r.Context().Done()
//	    return nil
//	})
//
//	// Broadcast to all connected clients
//	hub.Broadcast(sse.Event{Name: "update", Data: []byte("hello")})
//
// # Event Replay
//
// Enable clients to recover missed events using the Last-Event-ID header:
//
//	replayer := sse.NewMemoryReplayer(1000, time.Hour) // Keep 1000 events for 1 hour
//
//	app.GET("/events", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
//	    stream, err := sse.New(w, r)
//	    if err != nil {
//	        return err
//	    }
//	    defer stream.Close()
//
//	    // Replay missed events
//	    lastID := r.Header.Get("Last-Event-ID")
//	    if lastID != "" {
//	        replayer.Replay(lastID, stream.Send)
//	    }
//
//	    // Start listening for new events
//	    for msg := range messages {
//	        event := replayer.Store(sse.Event{Name: "message", Data: []byte(msg)})
//	        stream.Send(event)
//	    }
//	    return nil
//	})
//
// # Custom Provider
//
// Implement the Provider interface to use a custom SSE implementation:
//
//	type MyProvider struct{}
//
//	func (p *MyProvider) New(w http.ResponseWriter, r *http.Request) (sse.Connection, error) {
//	    return myCustomSSE(w, r)
//	}
//
//	app := zh.New(config.Config{
//	    SSEProvider: &MyProvider{},
//	})
//
// # Event Format
//
// Events follow the SSE specification:
//
//	id: 123
//	event: message
//	retry: 5000
//	data: Hello, World!
//
// Use the Event struct to create events:
//
//	event := sse.Event{
//	    ID:    "123",
//	    Name:  "message",
//	    Data:  []byte("Hello, World!"),
//	    Retry: 5 * time.Second,
//	}
//
// # Low-Level Writer
//
// For direct control, use Writer instead of the full Connection interface:
//
//	writer, err := sse.NewWriter(w, r)
//	if err != nil {
//	    return err
//	}
//	writer.WriteEvent(sse.Event{Data: []byte("hello")})
package sse
