package events

import (
	"context"
	"log/slog"
)

// FuncSubscriber adapts a plain function to the Subscriber interface.
type FuncSubscriber struct {
	EventTypes []string
	Fn         func(ctx context.Context, e Event) error
}

// Handle runs the wrapped function.
func (f FuncSubscriber) Handle(ctx context.Context, e Event) error {
	if f.Fn == nil {
		return nil
	}
	return f.Fn(ctx, e)
}

// Types lists the subscribed event types (empty = all).
func (f FuncSubscriber) Types() []string { return f.EventTypes }

// NopPublisher discards events; useful in tests and for wiring that has no bus.
type NopPublisher struct{}

// Publish drops the event.
func (NopPublisher) Publish(context.Context, Event) {}

// Subscribe ignores the subscriber.
func (NopPublisher) Subscribe(Subscriber) {}

var (
	_ Publisher  = NopPublisher{}
	_ Subscriber = FuncSubscriber{}
)

// LogSubscriber logs every event it receives — handy in development. It logs at
// debug level: one line per event is useful while wiring things up, but at info
// it doubles the noise of every domain operation that publishes.
func LogSubscriber(log *slog.Logger) Subscriber {
	return FuncSubscriber{Fn: func(_ context.Context, e Event) error {
		log.Debug("domain event", "type", e.Type, "id", e.ID)
		return nil
	}}
}
