package sse

import "sync"

// Broadcaster is the interface for SSE broadcast hubs.
// The built-in Hub satisfies this interface, and users can provide
// their own implementations (e.g. backed by Redis) for cross-instance broadcast.
type Broadcaster interface {
	Register(s *SSE)
	Unregister(s *SSE)
	Subscribe(s *SSE, topic string)
	Unsubscribe(s *SSE, topic string)
	Broadcast(event Event)
	BroadcastTo(topic string, event Event)
	ConnectionCount() int
	TopicCount(topic string) int
}

// Ensure Hub implements Broadcaster
var _ Broadcaster = (*Hub)(nil)

// Hub manages multiple SSE connections for broadcasting.
type Hub struct {
	connections   map[*SSE]struct{}
	topics        map[string]map[*SSE]struct{}
	mu            sync.RWMutex
	OnBroadcast   func(event Event)
	OnBroadcastTo func(topic string, event Event)
}

// NewHub creates a new SSE broadcast hub.
func NewHub() *Hub {
	return &Hub{
		connections: make(map[*SSE]struct{}),
		topics:      make(map[string]map[*SSE]struct{}),
	}
}

// Register adds an SSE connection to the hub.
func (h *Hub) Register(s *SSE) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connections[s] = struct{}{}
}

// Unregister removes an SSE connection from the hub.
func (h *Hub) Unregister(s *SSE) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.connections, s)
	for topic, subs := range h.topics {
		delete(subs, s)
		if len(subs) == 0 {
			delete(h.topics, topic)
		}
	}
}

// Subscribe adds an SSE connection to a topic.
func (h *Hub) Subscribe(s *SSE, topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.topics[topic] == nil {
		h.topics[topic] = make(map[*SSE]struct{})
	}
	h.topics[topic][s] = struct{}{}
}

// Unsubscribe removes an SSE connection from a topic.
func (h *Hub) Unsubscribe(s *SSE, topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs, ok := h.topics[topic]; ok {
		delete(subs, s)
		if len(subs) == 0 {
			delete(h.topics, topic)
		}
	}
}

// Broadcast sends an event to all registered connections.
// Connections that fail to receive the event are automatically unregistered.
func (h *Hub) Broadcast(event Event) {
	if h.OnBroadcast != nil {
		h.OnBroadcast(event)
	}

	h.mu.RLock()
	connections := make([]*SSE, 0, len(h.connections))
	for conn := range h.connections {
		connections = append(connections, conn)
	}
	h.mu.RUnlock()

	var failed []*SSE
	for _, conn := range connections {
		if err := conn.Send(event); err != nil {
			failed = append(failed, conn)
		}
	}

	// Unregister failed connections
	for _, conn := range failed {
		h.Unregister(conn)
		_ = conn.Close()
	}
}

// BroadcastTo sends an event to all connections subscribed to a topic.
// Connections that fail to receive the event are automatically unregistered.
func (h *Hub) BroadcastTo(topic string, event Event) {
	if h.OnBroadcastTo != nil {
		h.OnBroadcastTo(topic, event)
	}

	h.mu.RLock()
	var connections []*SSE
	if subs, ok := h.topics[topic]; ok {
		connections = make([]*SSE, 0, len(subs))
		for conn := range subs {
			connections = append(connections, conn)
		}
	}
	h.mu.RUnlock()

	var failed []*SSE
	for _, conn := range connections {
		if err := conn.Send(event); err != nil {
			failed = append(failed, conn)
		}
	}

	// Unregister failed connections
	for _, conn := range failed {
		h.Unregister(conn)
		_ = conn.Close()
	}
}

// ConnectionCount returns the number of registered connections.
func (h *Hub) ConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connections)
}

// TopicCount returns the number of connections subscribed to a topic.
func (h *Hub) TopicCount(topic string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.topics[topic])
}
