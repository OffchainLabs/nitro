// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package events

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSubscribe(t *testing.T) {
	producer := NewProducer[int]()
	sub := producer.Subscribe()
	require.Equal(t, 1, len(producer.subs))
	require.NotNil(t, sub)
}

func TestBroadcastToMultipleSubscribers(t *testing.T) {
	producer := NewProducer[int]()
	ctx := context.Background()

	sub1 := producer.Subscribe()
	sub2 := producer.Subscribe()
	sub3 := producer.Subscribe()

	producer.Broadcast(ctx, 99)

	for i, sub := range []*Subscription[int]{sub1, sub2, sub3} {
		event, shouldEnd := sub.Next(ctx)
		require.False(t, shouldEnd, "subscriber %d should not end", i)
		require.Equal(t, 99, event, "subscriber %d got wrong event", i)
	}
}

func TestBroadcastDropsWhenFull(t *testing.T) {
	producer := NewProducer(WithSubscriptionBuffer[int](1))
	sub := producer.Subscribe()
	ctx := context.Background()

	// First broadcast fills the buffer
	producer.Broadcast(ctx, 1)
	// Second broadcast should be dropped (buffer full)
	producer.Broadcast(ctx, 2)

	event, shouldEnd := sub.Next(ctx)
	require.False(t, shouldEnd)
	require.Equal(t, 1, event)

	// After draining, subsequent broadcasts should work
	producer.Broadcast(ctx, 3)
	event, shouldEnd = sub.Next(ctx)
	require.False(t, shouldEnd)
	require.Equal(t, 3, event)
}

func TestSubscriptionBufferSize(t *testing.T) {
	bufSize := 5
	producer := NewProducer(WithSubscriptionBuffer[int](bufSize))
	sub := producer.Subscribe()
	ctx := context.Background()

	// Fill the buffer completely
	for i := 0; i < bufSize; i++ {
		producer.Broadcast(ctx, i)
	}
	// One more should be dropped
	producer.Broadcast(ctx, 999)

	// Drain and verify we get exactly the buffered events
	for i := 0; i < bufSize; i++ {
		event, shouldEnd := sub.Next(ctx)
		require.False(t, shouldEnd)
		require.Equal(t, i, event)
	}
}

func TestWithSubscriptionBufferValidation(t *testing.T) {
	// Zero and negative sizes should fall back to the default buffer size.
	for _, size := range []int{0, -1, -100} {
		producer := NewProducer(WithSubscriptionBuffer[int](size))
		require.Equal(t, defaultSubscriptionBufferSize, producer.subscriptionBufferSize,
			"size %d should fall back to default", size)
	}
}

func TestNoGoroutineLeakOnBroadcast(t *testing.T) {
	// Verify that Broadcast does not leak goroutines when subscribers are slow.
	producer := NewProducer(WithSubscriptionBuffer[int](1))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go producer.Start(ctx)

	_ = producer.Subscribe()
	_ = producer.Subscribe()
	_ = producer.Subscribe()

	before := runtime.NumGoroutine()
	for i := 0; i < 1000; i++ {
		producer.Broadcast(ctx, i)
	}
	after := runtime.NumGoroutine()
	// A naive goroutine-per-send implementation would create 3000+ goroutines
	// here. Non-blocking sends create none.
	require.Less(t, after-before, 10, "broadcast should not spawn goroutines")
}

func TestEventProducer_Start(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	producer := NewProducer[int]()
	go producer.Start(ctx)

	sub := producer.Subscribe()

	cancel()
	_, shouldEnd := sub.Next(ctx)
	require.True(t, shouldEnd, "Expected to end after context cancellation")
}

func TestNextExitsOnCancel(t *testing.T) {
	producer := NewProducer[int]()
	sub := producer.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bool)
	go func() {
		_, shouldEnd := sub.Next(ctx)
		done <- shouldEnd
	}()

	cancel()
	select {
	case shouldEnd := <-done:
		require.True(t, shouldEnd)
	case <-time.After(2 * time.Second):
		t.Fatal("Next did not return after context cancellation")
	}
}

func TestNextExitsCleanlyWhenStartNotRunning(t *testing.T) {
	// Verify Next returns without blocking even if Start() was never called
	// (no receiver on doneListener).
	producer := NewProducer[int]()
	sub := producer.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan bool)
	go func() {
		_, shouldEnd := sub.Next(ctx)
		done <- shouldEnd
	}()

	select {
	case shouldEnd := <-done:
		require.True(t, shouldEnd)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Next blocked when Start was not running")
	}
}

func TestConcurrentBroadcastAndShutdown(t *testing.T) {
	// Verify no panics when broadcasting and cancelling concurrently.
	for i := 0; i < 100; i++ {
		producer := NewProducer(WithSubscriptionBuffer[int](1))
		ctx, cancel := context.WithCancel(context.Background())
		go producer.Start(ctx)

		subs := make([]*Subscription[int], 5)
		for j := range subs {
			subs[j] = producer.Subscribe()
		}

		// Broadcast and cancel concurrently
		done := make(chan struct{})
		go func() {
			for k := 0; k < 100; k++ {
				producer.Broadcast(ctx, k)
			}
			close(done)
		}()
		// Cancel mid-broadcast
		cancel()
		<-done

		// All subscribers should eventually exit cleanly
		for _, sub := range subs {
			for {
				_, shouldEnd := sub.Next(ctx)
				if shouldEnd {
					break
				}
				// Subscriber drained a buffered event; keep reading until ctx cancellation
			}
		}
	}
}

func TestRemovalUsesStableId(t *testing.T) {
	// This test ensures that removing subscriptions uses stable IDs rather than slice indices.
	// Using indices would cause incorrect removals when earlier subscriptions are deleted,
	// shifting remaining subscriptions to lower indices.
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
	require.Eventually(t, func() bool {
		producer.RLock()
		defer producer.RUnlock()
		return len(producer.subs) == 1 && producer.subs[0] == s2
	}, 2*time.Second, 5*time.Millisecond)
}
