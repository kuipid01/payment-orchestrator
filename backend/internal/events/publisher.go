package events

import (
	"context"
	"log/slog"
	"sync"
)

// Subscriber reacts to a published event. Implementations must be non-blocking:
// delivery failures are the subscriber's problem to persist and retry.
type Subscriber interface {
	// Handle processes one event. Returning an error only logs — it must never
	// roll back the business transaction that published the event.
	Handle(ctx context.Context, e Event) error
	// Types lists the event types this subscriber wants (empty = all).
	Types() []string
}

// Publisher fans out domain events to registered subscribers.
type Publisher interface {
	Publish(ctx context.Context, e Event)
	Subscribe(s Subscriber)
}

// Bus is an in-process, synchronous fan-out publisher.
//
// Synchronous on purpose: the payment write is already committed when we
// publish, so a subscriber failure cannot corrupt the ledger, and tests stay
// deterministic. Swap for a durable queue when running more than one instance.
type Bus struct {
	log         *slog.Logger
	mu          sync.RWMutex
	subscribers []Subscriber
}

// NewBus builds a publisher that logs subscriber failures.
func NewBus(log *slog.Logger) *Bus {
	return &Bus{log: log}
}

// Subscribe registers a subscriber.
func (b *Bus) Subscribe(s Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers = append(b.subscribers, s)
}

// Publish delivers an event to every interested subscriber.
func (b *Bus) Publish(ctx context.Context, e Event) {
	b.mu.RLock()
	subs := make([]Subscriber, len(b.subscribers))
	copy(subs, b.subscribers)
	b.mu.RUnlock()

	for _, s := range subs {
		if !wants(s, e.Type) {
			continue
		}
		if err := s.Handle(ctx, e); err != nil && b.log != nil {
			b.log.Error("event subscriber failed", "event_type", e.Type, "event_id", e.ID, "error", err)
		}
	}
}

func wants(s Subscriber, eventType string) bool {
	types := s.Types()
	if len(types) == 0 {
		return true
	}
	for _, t := range types {
		if t == eventType {
			return true
		}
	}
	return false
}

var _ Publisher = (*Bus)(nil)
