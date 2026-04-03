package engine

import (
	"sync"
	"time"
)

type EventType string

const (
	EventTunnelStateChanged EventType = "tunnel.stateChanged"
	EventTunnelLog          EventType = "tunnel.log"
	EventForwardListening   EventType = "tunnel.forwardListening"
	EventForwardError       EventType = "tunnel.forwardError"
	EventChainConnected     EventType = "tunnel.chainConnected"
	EventChainError         EventType = "tunnel.chainError"
	EventCoreHealth         EventType = "core.health"
	EventSettingsChanged    EventType = "settings.changed"
)

type Event struct {
	Type     EventType      `json:"type"`
	TunnelID string         `json:"tunnelId,omitempty"`
	Level    string         `json:"level,omitempty"`
	TS       time.Time      `json:"ts"`
	Message  string         `json:"message"`
	Fields   map[string]any `json:"fields,omitempty"`
}

type EventBus interface {
	Publish(e Event)
	Subscribe(bufSize int) (ch <-chan Event, cancel func())
}

type subscriber struct {
	ch chan Event
}

const recentEventsCap = 200

type eventBus struct {
	mu     sync.RWMutex
	subs   map[*subscriber]struct{}
	recent []Event
}

func NewEventBus() EventBus {
	return &eventBus{
		subs:   make(map[*subscriber]struct{}),
		recent: make([]Event, 0, recentEventsCap),
	}
}

func (b *eventBus) Publish(e Event) {
	if e.TS.IsZero() {
		e.TS = time.Now().UTC()
	}
	b.mu.Lock()
	if len(b.recent) >= recentEventsCap {
		b.recent = b.recent[1:]
	}
	b.recent = append(b.recent, e)
	for sub := range b.subs {
		select {
		case sub.ch <- e:
		default:
		}
	}
	b.mu.Unlock()
}

// Subscribe returns a channel that receives new events, prefilled with recent history.
func (b *eventBus) Subscribe(bufSize int) (<-chan Event, func()) {
	if bufSize <= 0 {
		bufSize = 64
	}
	sub := &subscriber{ch: make(chan Event, bufSize+recentEventsCap)}
	b.mu.Lock()
	for _, e := range b.recent {
		sub.ch <- e
	}
	b.subs[sub] = struct{}{}
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		delete(b.subs, sub)
		close(sub.ch)
		b.mu.Unlock()
		for range sub.ch {
		}
	}
	return sub.ch, cancel
}
