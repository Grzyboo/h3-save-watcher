package main

import (
	"reflect"
	"sync"
)

// Handler processes a single event. Handlers run sequentially on the bus
// goroutine, so they may mutate shared state without locks.
type Handler func(Event)

// Bus is a small channel-based event bus. Producers Publish events from any
// goroutine; subscribed handlers run sequentially on the single bus goroutine
// (Run), so event processing order equals publish order.
type Bus struct {
	mu       sync.Mutex
	subs     map[reflect.Type][]Handler
	queue    chan Event
	done     chan struct{}
	stopOnce sync.Once
}

func NewBus() *Bus {
	return &Bus{
		subs:  make(map[reflect.Type][]Handler),
		queue: make(chan Event, 256),
		done:  make(chan struct{}),
	}
}

// Subscribe registers h for events of type T. The handler runs on the bus
// goroutine; it may mutate shared state freely and may publish further events.
func Subscribe[T any](b *Bus, h func(T)) {
	t := reflect.TypeFor[T]()
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[t] = append(b.subs[t], func(e Event) { h(e.(T)) })
}

// Publish enqueues an event. Safe to call from any goroutine, including the
// bus goroutine itself. Events published after Stop are dropped.
func (b *Bus) Publish(e Event) {
	select {
	case b.queue <- e:
	case <-b.done:
	}
}

// Run dispatches events to their handlers until Stop is called.
func (b *Bus) Run() {
	for {
		select {
		case e := <-b.queue:
			t := reflect.TypeOf(e)
			b.mu.Lock()
			handlers := append([]Handler(nil), b.subs[t]...)
			b.mu.Unlock()
			for _, h := range handlers {
				h(e)
			}
		case <-b.done:
			return
		}
	}
}

// Stop shuts the bus down; events still in the queue are dropped.
func (b *Bus) Stop() {
	b.stopOnce.Do(func() { close(b.done) })
}
