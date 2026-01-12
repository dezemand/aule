package event

import (
	"context"
	"log/slog"
	"sync"
)

// dispatchItem represents an event to be dispatched to handlers.
type dispatchItem struct {
	topic string
	event any
}

// Bus is an async, channel-based event bus for publishing and subscribing to events.
// It uses worker goroutines to process events concurrently.
//
// Usage:
//
//	bus := event.NewBus(event.WithWorkerCount(4))
//	bus.Start()
//	defer bus.Stop()
//
//	// Subscribe to events (type-safe)
//	sub := event.Subscribe(bus, TopicProjectCreated, func(ctx context.Context, e *event.BaseEvent[ProjectCreated]) error {
//	    // handle event
//	    return nil
//	})
//	defer sub.Unsubscribe()
//
//	// Publish events (type-safe)
//	event.Publish(bus, event.NewEvent(TopicProjectCreated, ProjectCreated{...}))
type Bus struct {
	config    *busConfig
	handlers  map[string][]*subscription // topic string -> handlers
	eventChan chan dispatchItem
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	started   bool
}

// NewBus creates a new event bus with the given options.
// Call Start() to begin processing events.
func NewBus(opts ...Option) *Bus {
	config := defaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Bus{
		config:    config,
		handlers:  make(map[string][]*subscription),
		eventChan: make(chan dispatchItem, config.bufferSize),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start begins processing events with worker goroutines.
// Must be called before Publish will have any effect.
// Safe to call multiple times (subsequent calls are no-ops).
func (b *Bus) Start() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.started {
		return
	}
	b.started = true

	// Start worker goroutines
	for i := 0; i < b.config.workerCount; i++ {
		b.wg.Add(1)
		go b.worker(i)
	}

	b.log(slog.LevelInfo, "event bus started",
		slog.Int("workers", b.config.workerCount),
		slog.Int("buffer_size", b.config.bufferSize),
	)
}

// Stop gracefully shuts down the event bus.
// It stops accepting new events and waits for all pending events to be processed.
// Safe to call multiple times.
func (b *Bus) Stop() {
	b.mu.Lock()
	if !b.started {
		b.mu.Unlock()
		return
	}
	b.started = false
	b.mu.Unlock()

	// Signal workers to stop
	b.cancel()

	// Close the channel to unblock workers
	close(b.eventChan)

	// Wait for workers to finish
	b.wg.Wait()

	b.log(slog.LevelInfo, "event bus stopped")
}

// publish queues an event for async dispatch.
// Returns false if the bus is not started or the buffer is full.
func (b *Bus) publish(topic string, event any) bool {
	b.mu.RLock()
	started := b.started
	b.mu.RUnlock()

	if !started {
		return false
	}

	select {
	case b.eventChan <- dispatchItem{topic: topic, event: event}:
		return true
	case <-b.ctx.Done():
		return false
	default:
		// Buffer full, log warning
		b.log(slog.LevelWarn, "event buffer full, dropping event",
			slog.String("topic", topic),
		)
		return false
	}
}

// subscribe registers a handler for a topic string.
func (b *Bus) subscribe(topic string, handler Handler) Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub := newSubscription(topic, handler, nil)

	// Set the unsubscribe function
	sub.unsubscribe = func() {
		b.unsubscribe(sub)
	}

	b.handlers[topic] = append(b.handlers[topic], sub)

	b.log(slog.LevelDebug, "subscription added",
		slog.String("topic", topic),
		slog.String("subscription_id", sub.id.String()),
	)

	return sub
}

// unsubscribe removes a subscription from the bus.
func (b *Bus) unsubscribe(sub *subscription) {
	b.mu.Lock()
	defer b.mu.Unlock()

	handlers := b.handlers[sub.topic]
	for i, h := range handlers {
		if h.id == sub.id {
			// Remove by swapping with last element
			b.handlers[sub.topic] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	b.log(slog.LevelDebug, "subscription removed",
		slog.String("topic", sub.topic),
		slog.String("subscription_id", sub.id.String()),
	)
}

// worker processes events from the channel.
func (b *Bus) worker(id int) {
	defer b.wg.Done()

	for {
		select {
		case item, ok := <-b.eventChan:
			if !ok {
				// Channel closed
				return
			}
			b.dispatch(item)
		case <-b.ctx.Done():
			// Drain remaining events
			for item := range b.eventChan {
				b.dispatch(item)
			}
			return
		}
	}
}

// dispatch sends an event to all matching handlers.
func (b *Bus) dispatch(item dispatchItem) {
	b.mu.RLock()
	handlers := make([]*subscription, len(b.handlers[item.topic]))
	copy(handlers, b.handlers[item.topic])
	b.mu.RUnlock()

	ctx := context.Background()

	for _, sub := range handlers {
		if err := sub.handler.Handle(ctx, item.event); err != nil {
			b.config.errorHandler(item.topic, err)
			b.log(slog.LevelError, "handler error",
				slog.String("topic", item.topic),
				slog.String("subscription_id", sub.id.String()),
				slog.String("error", err.Error()),
			)
		}
	}
}

// log writes a log message if a logger is configured.
func (b *Bus) log(level slog.Level, msg string, attrs ...any) {
	if b.config.logger != nil {
		b.config.logger.Log(context.Background(), level, msg, attrs...)
	}
}

// Subscribe registers a type-safe handler for events on the given topic.
// The handler receives the full BaseEvent[T] including metadata.
//
// Example:
//
//	sub := event.Subscribe(bus, TopicProjectCreated, func(ctx context.Context, e *event.BaseEvent[ProjectCreated]) error {
//	    project := e.Payload()
//	    meta := e.Metadata()
//	    return nil
//	})
//	defer sub.Unsubscribe()
func Subscribe[T any](bus *Bus, topic Topic[T], handler func(ctx context.Context, event Event[T]) error) Subscription {
	return bus.subscribe(topic.String(), HandlerFunc(func(ctx context.Context, evt any) error {
		typedEvent, ok := evt.(*BaseEvent[T])
		if !ok {
			// Skip events that don't match the expected type
			return nil
		}
		return handler(ctx, typedEvent)
	}))
}

// Publish sends a type-safe event to all matching subscribers.
// This method is non-blocking; events are queued for async processing.
// Returns false if the bus is not started or the buffer is full.
//
// Example:
//
//	evt := event.NewEvent(TopicProjectCreated, ProjectCreated{Name: "My Project"})
//	event.Publish(bus, evt)
func Publish[T any](bus *Bus, event Event[T]) bool {
	bus.log(slog.LevelInfo, "event", "topic", event.Topic().String(), "payload", event.Payload(), "metadata", event.Metadata())
	return bus.publish(event.Topic().String(), event)
}
