// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package events

import (
	"context"
	"slices"
	"sync"
)

const (
	defaultSubscriptionBufferSize = 10
)

// Producer manages event subscriptions and broadcasts events to them.
type Producer[T any] struct {
	sync.RWMutex
	subscriptionBufferSize int
	subs                   []*Subscription[T]
	doneListener           chan subId // channel to listen for IDs of subscriptions to be removed.
	nextId                 subId      // monotonically increasing id for stable subscription identification
}

type ProducerOpt[T any] func(*Producer[T])

// WithSubscriptionBuffer customizes the size of the subscription buffer channel.
// If size is less than 1, the default buffer size is used.
func WithSubscriptionBuffer[T any](size int) ProducerOpt[T] {
	return func(ep *Producer[T]) {
		if size < 1 {
			size = defaultSubscriptionBufferSize
		}
		ep.subscriptionBufferSize = size
	}
}

func NewProducer[T any](opts ...ProducerOpt[T]) *Producer[T] {
	producer := &Producer[T]{
		subs:                   make([]*Subscription[T], 0),
		subscriptionBufferSize: defaultSubscriptionBufferSize,
		doneListener:           make(chan subId, 100),
	}
	for _, opt := range opts {
		opt(producer)
	}
	return producer
}

// Start begins listening for subscription cancellation requests or context cancellation.
func (ep *Producer[T]) Start(ctx context.Context) {
	for {
		select {
		case id := <-ep.doneListener:
			ep.Lock()
			ep.subs = slices.DeleteFunc(ep.subs, func(s *Subscription[T]) bool {
				return s.id == id
			})
			ep.Unlock()
		case <-ctx.Done():
			ep.Lock()
			ep.subs = nil
			ep.Unlock()
			return
		}
	}
}

// Subscribe returns a handle to a new event subscription,
// adding it to the list of active subscriptions.
func (ep *Producer[T]) Subscribe() *Subscription[T] {
	ep.Lock()
	defer ep.Unlock()
	sub := &Subscription[T]{
		id:     ep.nextId,
		events: make(chan T, ep.subscriptionBufferSize),
		done:   ep.doneListener,
	}
	ep.nextId++
	ep.subs = append(ep.subs, sub)
	return sub
}

// Broadcast sends an event to all active subscriptions. If a subscription's
// buffer is full the event is dropped, as the subscriber already has pending
// events to process. This avoids spawning goroutines per broadcast and
// eliminates goroutine leaks when subscribers are slow. The ctx parameter is
// unused but retained for API compatibility.
func (ep *Producer[T]) Broadcast(_ context.Context, event T) {
	ep.RLock()
	defer ep.RUnlock()
	for _, sub := range ep.subs {
		select {
		case sub.events <- event:
		default:
		}
	}
}

type subId int

// Subscription defines a generic handle to a subscription of
// events from a producer.
type Subscription[T any] struct {
	id     subId
	events chan T
	done   chan subId
}

// Next waits for the next event or context cancellation. It returns the event
// and false on success, or a zero value and true if the context was cancelled.
func (es *Subscription[T]) Next(ctx context.Context) (T, bool) {
	var zeroVal T
	select {
	case ev := <-es.events:
		return ev, false
	case <-ctx.Done():
		// Non-blocking send: Start() may have already exited, leaving no receiver.
		select {
		case es.done <- es.id:
		default:
		}
		return zeroVal, true
	}
}
