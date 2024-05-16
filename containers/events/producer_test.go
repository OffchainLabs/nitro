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
		event, err := sub.Next(context.Background())
		require.NoError(t, err)
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

	event, err := sub.Next(context.Background())
	require.NoError(t, err)
	require.Equal(t, 42, event)
}

func TestEventProducer_Start(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	producer := NewProducer[int]()
	go producer.Start(ctx)

	sub := producer.Subscribe()

	// Simulate removing the subscription.
	cancel()
	_, err := sub.Next(ctx)
	if err == nil {
		t.Error("Expected an error after context cancellation, got none")
	}
}
