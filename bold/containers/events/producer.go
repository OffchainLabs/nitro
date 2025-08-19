// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

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
	subsIndex              map[subId]int
	nextID                 subId
	doneListener           chan subId    // channel to listen for IDs of subscriptions to be remove.
	broadcastTimeout       time.Duration // maximum duration to wait for an event to be sent.
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
		subsIndex:              make(map[subId]int),
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
			idx, ok := ep.subsIndex[id]
			if !ok {
				ep.Unlock()
				continue
			}
			last := len(ep.subs) - 1
			if idx != last {
				moved := ep.subs[last]
				ep.subs[idx] = moved
				ep.subsIndex[moved.id] = idx
			}
			ep.subs = ep.subs[:last]
			delete(ep.subsIndex, id)
			ep.Unlock()
		case <-ctx.Done():
			close(ep.doneListener)
			ep.subs = nil
			return
		}
	}
}

// Subscribe returns a handle to a new event subscription,
// adding it to the list of active subscriptions.
func (ep *Producer[T]) Subscribe() *Subscription[T] {
	ep.Lock()
	defer ep.Unlock()
	id := ep.nextID
	ep.nextID++
	sub := &Subscription[T]{
		id:     id, // stable unique ID
		events: make(chan T),
		done:   ep.doneListener,
	}
	ep.subs = append(ep.subs, sub)
	ep.subsIndex[id] = len(ep.subs) - 1
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
}

// Next waits for the next event or context cancelation, returning the event or an error.
func (es *Subscription[T]) Next(ctx context.Context) (T, bool) {
	var zeroVal T
	for {
		select {
		case ev := <-es.events:
			return ev, false
		case <-ctx.Done():
			es.done <- es.id
			close(es.events)
			return zeroVal, true
		}
	}
}
