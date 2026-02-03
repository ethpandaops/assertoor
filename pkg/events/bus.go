package events

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

// FilterFunc is a function that filters events for a subscriber.
type FilterFunc func(*Event) bool

// Subscriber represents a subscriber to the event bus.
type Subscriber struct {
	id      uint64
	filter  FilterFunc
	channel chan *Event
	closed  atomic.Bool
}

// Channel returns the channel for receiving events.
func (s *Subscriber) Channel() <-chan *Event {
	return s.channel
}

// Bus is the interface for the event bus.
type Bus interface {
	Start(ctx context.Context) error
	Stop() error
	Publish(event *Event)
	Subscribe(filter FilterFunc) *Subscriber
	Unsubscribe(sub *Subscriber)
}

// EventBus is the default implementation of the Bus interface.
type EventBus struct {
	logger        logrus.FieldLogger
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	eventChan     chan *Event
	subscribersMu sync.RWMutex
	subscribers   map[uint64]*Subscriber
	nextSubID     atomic.Uint64
	nextEventID   atomic.Uint64
	bufferSize    int
	subscriberBuf int
}

// NewEventBus creates a new event bus.
func NewEventBus(logger logrus.FieldLogger) *EventBus {
	return &EventBus{
		logger:        logger.WithField("component", "eventbus"),
		eventChan:     make(chan *Event, 1000),
		subscribers:   make(map[uint64]*Subscriber, 16),
		bufferSize:    1000,
		subscriberBuf: 100,
	}
}

// Start starts the event bus processing loop.
func (eb *EventBus) Start(ctx context.Context) error {
	eb.ctx, eb.cancel = context.WithCancel(ctx)

	eb.wg.Add(1)

	go eb.processEvents()

	eb.logger.Info("event bus started")

	return nil
}

// Stop stops the event bus.
func (eb *EventBus) Stop() error {
	if eb.cancel != nil {
		eb.cancel()
	}

	eb.wg.Wait()

	// Close all subscriber channels
	eb.subscribersMu.Lock()

	for _, sub := range eb.subscribers {
		if sub.closed.CompareAndSwap(false, true) {
			close(sub.channel)
		}
	}

	eb.subscribers = make(map[uint64]*Subscriber, 16)
	eb.subscribersMu.Unlock()

	eb.logger.Info("event bus stopped")

	return nil
}

// Publish publishes an event to all subscribers.
func (eb *EventBus) Publish(event *Event) {
	if event == nil {
		return
	}

	// Assign event ID
	event.ID = eb.nextEventID.Add(1)

	select {
	case eb.eventChan <- event:
	default:
		eb.logger.Warn("event channel full, dropping event")
	}
}

// Subscribe creates a new subscription with an optional filter.
func (eb *EventBus) Subscribe(filter FilterFunc) *Subscriber {
	sub := &Subscriber{
		id:      eb.nextSubID.Add(1),
		filter:  filter,
		channel: make(chan *Event, eb.subscriberBuf),
	}

	eb.subscribersMu.Lock()
	eb.subscribers[sub.id] = sub
	eb.subscribersMu.Unlock()

	eb.logger.WithField("subscriber_id", sub.id).Debug("new subscriber added")

	return sub
}

// Unsubscribe removes a subscriber from the event bus.
func (eb *EventBus) Unsubscribe(sub *Subscriber) {
	if sub == nil {
		return
	}

	eb.subscribersMu.Lock()

	if _, exists := eb.subscribers[sub.id]; exists {
		delete(eb.subscribers, sub.id)

		if sub.closed.CompareAndSwap(false, true) {
			close(sub.channel)
		}
	}

	eb.subscribersMu.Unlock()

	eb.logger.WithField("subscriber_id", sub.id).Debug("subscriber removed")
}

// processEvents is the main event processing loop.
func (eb *EventBus) processEvents() {
	defer eb.wg.Done()

	for {
		select {
		case <-eb.ctx.Done():
			return
		case event := <-eb.eventChan:
			eb.dispatchEvent(event)
		}
	}
}

// dispatchEvent sends an event to all matching subscribers.
func (eb *EventBus) dispatchEvent(event *Event) {
	eb.subscribersMu.RLock()
	defer eb.subscribersMu.RUnlock()

	for _, sub := range eb.subscribers {
		// Skip if filter doesn't match
		if sub.filter != nil && !sub.filter(event) {
			continue
		}

		// Try to send event, drop if subscriber channel is full
		select {
		case sub.channel <- event:
		default:
			eb.logger.WithFields(logrus.Fields{
				"subscriber_id": sub.id,
				"event_type":    event.Type,
			}).Debug("subscriber channel full, dropping event")
		}
	}
}

// CreateTestRunFilter creates a filter for a specific test run.
func CreateTestRunFilter(testRunID uint64) FilterFunc {
	return func(e *Event) bool {
		return e.TestRunID == testRunID
	}
}

// CreateEventTypeFilter creates a filter for specific event types.
func CreateEventTypeFilter(eventTypes ...EventType) FilterFunc {
	typeSet := make(map[EventType]struct{}, len(eventTypes))
	for _, t := range eventTypes {
		typeSet[t] = struct{}{}
	}

	return func(e *Event) bool {
		_, ok := typeSet[e.Type]
		return ok
	}
}

// CombineFilters combines multiple filters with AND logic.
func CombineFilters(filters ...FilterFunc) FilterFunc {
	return func(e *Event) bool {
		for _, f := range filters {
			if f != nil && !f(e) {
				return false
			}
		}

		return true
	}
}

// NewCustomEvent creates a new event with a custom event type string.
func (eb *EventBus) NewCustomEvent(eventType string, testRunID, taskIndex uint64, data any) (*Event, error) {
	return NewCustomEvent(eventType, testRunID, taskIndex, data)
}
