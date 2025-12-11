// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package events

import (
	"context"
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
