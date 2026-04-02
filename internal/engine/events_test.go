package engine

import (
	"testing"
	"time"
)

func TestEventBus_PublishSubscribe(t *testing.T) {
	bus := NewEventBus()
	ch, cancel := bus.Subscribe(16)
	defer cancel()

	bus.Publish(Event{Type: EventTunnelStateChanged, TunnelID: "t1", Message: "started"})

	select {
	case e := <-ch:
		if e.TunnelID != "t1" {
			t.Errorf("TunnelID = %s, want t1", e.TunnelID)
		}
		if e.TS.IsZero() {
			t.Error("TS should be auto-set")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewEventBus()
	ch1, cancel1 := bus.Subscribe(16)
	defer cancel1()
	ch2, cancel2 := bus.Subscribe(16)
	defer cancel2()

	bus.Publish(Event{Type: EventTunnelLog, Message: "test"})

	for _, ch := range []<-chan Event{ch1, ch2} {
		select {
		case e := <-ch:
			if e.Message != "test" {
				t.Errorf("Message = %s, want test", e.Message)
			}
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestEventBus_NonBlockingPublish(t *testing.T) {
	bus := NewEventBus()
	ch, cancel := bus.Subscribe(1)
	defer cancel()

	bus.Publish(Event{Message: "first"})
	bus.Publish(Event{Message: "second"})

	select {
	case e := <-ch:
		if e.Message != "first" {
			t.Errorf("Message = %s, want first", e.Message)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestEventBus_CancelSubscription(t *testing.T) {
	bus := NewEventBus()
	ch, cancel := bus.Subscribe(16)
	cancel()

	bus.Publish(Event{Message: "after cancel"})

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("should not receive events after cancel")
		}
	case <-time.After(100 * time.Millisecond):
	}
}
