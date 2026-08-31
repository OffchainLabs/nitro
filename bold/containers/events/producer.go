// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package events

import (
	"context"
	"sync"
	"time"
)

const (
	defaultBroadcastTimeout       = time.Millisecond * 500
	defaultSubscriptionBufferSize = 10
)

// Producer manages event subscriptions and broadcasts events to them.
type Producer[T any] struct {
	sync.RWMutex
	subscriptionBufferSize int
	subs                   []*Subscription[T]
	doneListener           chan subId    // channel to listen for IDs of subscriptions to be remove.
	broadcastTimeout       time.Duration // maximum duration to wait for an event to be sent.
	nextId                 subId         // monotonically increasing id for stable subscription identification
}

type ProducerOpt[T any] func(*Producer[T])

// WithBroadcastTimeout enables the amount of time the broadcaster will wait to send
// to each subscriber before dropping the send.
func WithBroadcastTimeout[T any](timeout time.Duration) ProducerOpt[T] {
	return func(ep *Producer[T]) {
		ep.broadcastTimeout = timeout
	}
}

// WithSubscriptionBuffer customizes the size of the subscription buffer channel.
func WithSubscriptionBuffer[T any](size int) ProducerOpt[T] {
	return func(ep *Producer[T]) {
		ep.subscriptionBufferSize = size
	}
}

func NewProducer[T any](opts ...ProducerOpt[T]) *Producer[T] {
	producer := &Producer[T]{
		subs:                   make([]*Subscription[T], 0),
		subscriptionBufferSize: defaultSubscriptionBufferSize,
		doneListener:           make(chan subId, 100),
		broadcastTimeout:       defaultBroadcastTimeout,
	}
	for _, opt := range opts {
		opt(producer)
	}
	return producer
}

// Start begins listening for subscription cancelation requests or context cancelation.
func (ep *Producer[T]) Start(ctx context.Context) {
	for {
		select {
		case id := <-ep.doneListener:
			ep.Lock()
			// Find the subscription by stable id and remove it if present.
			idx := -1
			for i, s := range ep.subs {
				if s.id == id {
					idx = i
					break
				}
			}
			if idx >= 0 {
				ep.subs = append(ep.subs[:idx], ep.subs[idx+1:]...)
			}
			ep.Unlock()
		case <-ctx.Done():
			// doneListener is intentionally NOT closed here. Subscriptions send
			// their id to it while tearing down (see Subscription.Next), so
			// closing it would race with those sends and panic ("send on closed
			// channel"). Nothing receives from it after we return; that is safe
			// because Next's send is non-blocking, so late teardowns drop their
			// id rather than blocking on a full buffer. The channel is freed
			// once the Producer and all of its Subscriptions become unreachable.
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
	ctx, cancel := context.WithCancel(context.Background())
	sub := &Subscription[T]{
		id:     ep.nextId, // Assign a stable, monotonically increasing ID
		events: make(chan T),
		done:   ep.doneListener,
		ctx:    ctx,
		cancel: cancel,
	}
	ep.nextId++
	ep.subs = append(ep.subs, sub)
	return sub
}

// Broadcast sends an event to all active subscriptions, respecting a configured timeout or context.
// It spawns goroutines to send events to each subscription so as to not block the producer to submitting
// to all consumers. Broadcast should be used if not all consumers are expected to consume the event,
// within a reasonable time, or if the configured broadcast timeout is short enough.
func (ep *Producer[T]) Broadcast(ctx context.Context, event T) {
	ep.RLock()
	defer ep.RUnlock()
	for _, sub := range ep.subs {
		go func(listener *Subscription[T]) {
			select {
			case listener.events <- event:
			case <-time.After(ep.broadcastTimeout):
			case <-ctx.Done():
			case <-listener.ctx.Done():
			}
		}(sub)
	}
}

type subId int

// Subscription defines a generic handle to a subscription of
// events from a producer.
type Subscription[T any] struct {
	id     subId
	events chan T
	done   chan subId
	ctx    context.Context
	cancel context.CancelFunc
}

// Next waits for the next event or for ctx to be canceled. It returns
// (event, false) when an event is delivered, and (zeroVal, true) once the
// subscription is finished. Cancelling ctx tears the subscription down; callers
// must keep calling Next until it reports true, or the subscription is never
// cleaned up. Next is idempotent: after it has reported true once, every later
// call reports true immediately.
func (es *Subscription[T]) Next(ctx context.Context) (T, bool) {
	var zeroVal T
	for {
		select {
		case ev := <-es.events:
			return ev, false
		case <-es.ctx.Done():
			return zeroVal, true
		case <-ctx.Done():
			es.cancel()
			select {
			case es.done <- es.id:
			default:
			}
			return zeroVal, true
		}
	}
}
