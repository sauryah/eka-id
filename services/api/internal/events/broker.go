package events

import (
	"encoding/json"
	"sync"
	"time"
)

type EventType string

const (
	EventConsentRequested        EventType = "CONSENT_REQUEST_CREATED"
	EventConsentResponded        EventType = "CONSENT_REQUEST_RESPONDED"
	EventIdentityUpdated         EventType = "IDENTITY_UPDATED"
	EventAmendmentStatusChanged EventType = "AMENDMENT_STATUS_CHANGED"
)

type Event struct {
	Type      EventType              `json:"type"`
	TargetID  string                 `json:"target_id"` // UserID or OrgID or EkaID
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
}

type EventBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]chan Event
}

var (
	globalBroker *EventBroker
	once         sync.Once
)

func GetBroker() *EventBroker {
	once.Do(func() {
		globalBroker = &EventBroker{
			subscribers: make(map[string][]chan Event),
		}
	})
	return globalBroker
}

func (b *EventBroker) Subscribe(topic string) chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, 16)
	b.subscribers[topic] = append(b.subscribers[topic], ch)
	return ch
}

func (b *EventBroker) Unsubscribe(topic string, ch chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[topic]
	for i, sub := range subs {
		if sub == ch {
			b.subscribers[topic] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
	if len(b.subscribers[topic]) == 0 {
		delete(b.subscribers, topic)
	}
}

func (b *EventBroker) Publish(topic string, event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	for _, ch := range b.subscribers[topic] {
		select {
		case ch <- event:
		default:
			// Non-blocking drop if subscriber channel buffer is full
		}
	}
}

func (e Event) ToSSE() []byte {
	data, _ := json.Marshal(e)
	return []byte("event: " + string(e.Type) + "\ndata: " + string(data) + "\n\n")
}
