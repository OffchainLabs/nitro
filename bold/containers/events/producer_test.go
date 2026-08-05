// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package events

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSubscribe(t *testing.T) {
	producer := NewProducer[int]()
	sub := producer.Subscribe()
	require.Equal(t, 1, len(producer.subs))
	require.NotNil(t, sub)
}

func TestBroadcast(t *testing.T) {
	producer := NewProducer[int]()
	sub := producer.Subscribe()
	done := make(chan bool)
	go func() {
		event, shouldEnd := sub.Next(context.Background())
		require.False(t, shouldEnd)
		require.Equal(t, 42, event)
		done <- true
	}()
	ctx := context.Background()
	producer.Broadcast(ctx, 42)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out waiting for event")
	}
}

func TestBroadcastTimeout(t *testing.T) {
	timeout := 50 * time.Millisecond
	producer := NewProducer(WithBroadcastTimeout[int](timeout))
	sub := producer.Subscribe()

	go func() {
		// Delay sending to simulate timeout scenario
		time.Sleep(100 * time.Millisecond)
		sub.events <- 42
	}()

	event, shouldEnd := sub.Next(context.Background())
	require.False(t, shouldEnd)
	require.Equal(t, 42, event)
}

func TestEventProducer_Start(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	producer := NewProducer[int]()
	go producer.Start(ctx)

	sub := producer.Subscribe()

	// Simulate removing the subscription.
	cancel()
	_, shouldEnd := sub.Next(ctx)
	if !shouldEnd {
		t.Error("Expected to end after context cancellation")
	}
}

func TestRemovalUsesStableId(t *testing.T) {
	// This test ensures that removing subscriptions uses stable IDs rather than slice indices.
	// Before the fix, deleting two subscriptions by their IDs 0 and 1 would incorrectly
	// remove the first (index 0) and the third (now at index 1 after compaction), leaving
	// the second subscription in place instead of the third.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	producer := NewProducer[int]()
	go producer.Start(ctx)

	s0 := producer.Subscribe()
	s1 := producer.Subscribe()
	s2 := producer.Subscribe()

	// Cancel first two subscriptions; they will send their IDs to doneListener via Next.
	for _, s := range []*Subscription[int]{s0, s1} {
		c, cancelSub := context.WithCancel(context.Background())
		cancelSub()
		_, shouldEnd := s.Next(c)
		require.True(t, shouldEnd)
	}

	// Wait until the producer processes removal and only one subscription remains.
	deadline := time.Now().Add(2 * time.Second)
	for {
		producer.RLock()
		remaining := len(producer.subs)
		producer.RUnlock()
		if remaining == 1 || time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	producer.RLock()
	require.Equal(t, 1, len(producer.subs))
	require.Same(t, s2, producer.subs[0])
	producer.RUnlock()
}

// TestBroadcastAfterSubscriberDone verifies the "send on closed channel" panic
// is gone. Once a subscription is finished (Next returned done), delivering a
// broadcast to it must not panic and the delivery goroutine must exit via the
// subscription's canceled context instead of parking on the send.
func TestBroadcastAfterSubscriberDone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		producer := NewProducer[int]()
		sub := producer.Subscribe()

		// Finish the subscription: with an already-canceled context, Next returns
		// done and cancels the subscription's own context.
		subCtx, cancelSub := context.WithCancel(context.Background())
		cancelSub()
		_, done := sub.Next(subCtx)
		require.True(t, done)

		producer.Broadcast(context.Background(), 41)
		synctest.Wait()
	})
}

// TestTeardownDoesNotBlockWhenDoneListenerFull verifies that tearing a
// subscription down never parks on the producer's doneListener. Nothing
// receives from that channel unless Start is running, so once its buffer fills,
// Next must drop its id rather than block forever on a receiver that will never
// arrive. Without the non-blocking send, the teardown past the buffer's
// capacity deadlocks the bubble.
func TestTeardownDoesNotBlockWhenDoneListenerFull(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Start is deliberately not running: nothing drains doneListener.
		producer := NewProducer[int]()

		for i := 0; i < cap(producer.doneListener)+5; i++ {
			sub := producer.Subscribe()
			subCtx, cancelSub := context.WithCancel(context.Background())
			cancelSub()
			_, done := sub.Next(subCtx)
			require.True(t, done)
		}

		// The teardowns past capacity dropped their ids instead of blocking.
		require.Equal(t, cap(producer.doneListener), len(producer.doneListener))
	})
}

// TestTeardownAfterProducerStopped verifies a subscription can still finish
// after the producer's Start loop has exited. doneListener is intentionally
// left open on shutdown: closing it would race with the send in Next and panic
// with "send on closed channel".
func TestTeardownAfterProducerStopped(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		producer := NewProducer[int]()
		go producer.Start(ctx)

		sub := producer.Subscribe()

		// Stop the producer and wait until Start has provably returned, so the
		// teardown below runs with no receiver on doneListener.
		cancel()
		synctest.Wait()

		subCtx, cancelSub := context.WithCancel(context.Background())
		cancelSub()
		_, done := sub.Next(subCtx)
		require.True(t, done)

		// The id was buffered rather than dropped or sent on a closed channel.
		require.Equal(t, 1, len(producer.doneListener))
	})
}

// TestNextIsIdempotentOnceDone verifies a finished subscription stays finished:
// later calls to Next report done immediately instead of blocking on a channel
// nobody will send to, and an in-flight broadcast cannot hand an event to a
// subscriber that has already torn down.
func TestNextIsIdempotentOnceDone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		producer := NewProducer[int]()
		sub := producer.Subscribe()

		subCtx, cancelSub := context.WithCancel(context.Background())
		cancelSub()
		_, done := sub.Next(subCtx)
		require.True(t, done)

		producer.Broadcast(context.Background(), 41)

		ev, done := sub.Next(context.Background())
		require.True(t, done)
		require.Zero(t, ev)
	})
}
