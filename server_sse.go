package zerohttp

import "github.com/alexferl/zerohttp/config"

// SetSSEProvider sets the SSE provider instance. This can be used to inject
// an SSE implementation after creating the server.
//
// Users can implement their own SSE provider or use the built-in stdlib provider:
//
//	app := zerohttp.New()
//	app.SetSSEProvider(zh.NewDefaultProvider())
//
//	app.GET("/events", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
//	    provider := app.SSEProvider()
//	    sse, err := provider.NewSSE(w, r)
//	    if err != nil {
//	        return err
//	    }
//	    defer sse.Close()
//	    // ... stream events ...
//	}))
//
// Parameters:
//   - provider: An SSE provider instance implementing the config.SSEProvider interface
func (s *Server) SetSSEProvider(provider config.SSEProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sseProvider = provider
}

// SSEProvider returns the configured SSE provider (if any).
// Returns nil if no SSE provider has been configured.
func (s *Server) SSEProvider() config.SSEProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sseProvider
}
