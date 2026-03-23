package sse

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/alexferl/zerohttp/httpx"
)

// Replayer defines the interface for event replay storage.
// Implementations can use in-memory storage, Redis, databases, etc.
type Replayer interface {
	// Store saves an event to the replay buffer and returns the event with assigned ID.
	Store(event Event) Event
	// Replay sends all events after the given ID to the provided send function.
	// Returns the number of events replayed and any error.
	Replay(afterID string, send func(Event) error) (int, error)
}

// Ensure MemoryReplayer implements Replayer
var _ Replayer = (*MemoryReplayer)(nil)

// MemoryReplayer stores events in memory with a circular buffer.
// Events can be limited by max count and/or TTL.
type MemoryReplayer struct {
	events    []storedEvent
	maxEvents int
	ttl       time.Duration
	mu        sync.RWMutex
	lastID    int64
}

type storedEvent struct {
	id        int64
	event     Event
	timestamp time.Time
}

// NewMemoryReplayer creates a new in-memory event replayer.
// maxEvents is the maximum number of events to keep (0 = unlimited).
// ttl is how long to keep events (0 = no expiration).
func NewMemoryReplayer(maxEvents int, ttl time.Duration) *MemoryReplayer {
	if maxEvents < 0 {
		maxEvents = 0
	}
	return &MemoryReplayer{
		events:    make([]storedEvent, 0),
		maxEvents: maxEvents,
		ttl:       ttl,
	}
}

// Store saves an event to the replay buffer with an auto-generated ID.
// Returns the event with the assigned ID so it can be used for broadcasting.
func (r *MemoryReplayer) Store(event Event) Event {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	if r.ttl > 0 {
		valid := make([]storedEvent, 0, len(r.events))
		for _, e := range r.events {
			if now.Sub(e.timestamp) < r.ttl {
				valid = append(valid, e)
			}
		}
		r.events = valid
	}

	r.lastID++
	event.ID = strconv.FormatInt(r.lastID, 10)

	r.events = append(r.events, storedEvent{
		id:        r.lastID,
		event:     event,
		timestamp: now,
	})

	if r.maxEvents > 0 && len(r.events) > r.maxEvents {
		r.events = r.events[len(r.events)-r.maxEvents:]
	}

	return event
}

// Replay sends all events after the given ID to the provided send function.
func (r *MemoryReplayer) Replay(afterID string, send func(Event) error) (int, error) {
	startID := int64(0)
	if afterID != "" {
		var err error
		startID, err = strconv.ParseInt(afterID, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("sse: invalid Last-Event-ID: %w", err)
		}
	}

	// Snapshot events under lock, then release before I/O
	r.mu.RLock()
	var snapshot []Event
	for _, se := range r.events {
		if se.id > startID {
			snapshot = append(snapshot, se.event)
		}
	}
	r.mu.RUnlock()

	count := 0
	for _, event := range snapshot {
		if err := send(event); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// WithReplay creates a new SSE connection and replays missed events if Last-Event-ID header is present.
// After replay completes, the connection is ready for new events.
func WithReplay(w http.ResponseWriter, r *http.Request, replayer Replayer) (*SSE, error) {
	stream, err := New(w, r)
	if err != nil {
		return nil, err
	}

	// Check for Last-Event-ID header for replay
	lastEventID := r.Header.Get(httpx.HeaderLastEventId)
	if lastEventID != "" {
		if replayer == nil {
			_ = stream.Close()
			return nil, fmt.Errorf("sse: Last-Event-ID header present but no replayer configured")
		}
		_, err := replayer.Replay(lastEventID, func(event Event) error {
			return stream.Send(event)
		})
		if err != nil {
			_ = stream.Close()
			return nil, fmt.Errorf("sse: replay failed: %w", err)
		}
	}

	return stream, nil
}
