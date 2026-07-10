// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package transactionfeed

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestBroadcastDroppedCounter(t *testing.T) {
	cfg := DefaultServerConfig
	cfg.BroadcastBuf = 1
	s := NewServer(cfg)

	msg := &TransactionFeedMessage{Version: TransactionFeedV1}
	start := broadcastDroppedCounter.Snapshot().Count()

	s.BroadcastTransaction(msg)
	s.BroadcastTransaction(msg)
	s.BroadcastTransaction(msg)

	if delta := broadcastDroppedCounter.Snapshot().Count() - start; delta < 2 {
		t.Fatalf("expected >= 2 drops, got delta=%d", delta)
	}
}

// TestServerConcurrentClients churns clients connecting, reading, and
// disconnecting while broadcasts are in flight, then checks the server drains
// to zero clients and shuts down without deadlocking or leaking sessions.
func TestServerConcurrentClients(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := DefaultServerConfig
	cfg.Addr = "127.0.0.1"
	cfg.Port = "0"
	s := NewServer(cfg)
	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer s.StopAndWait()

	url := fmt.Sprintf("ws://%s/", s.ListenerAddr().String())

	stopBroadcast := make(chan struct{})
	var broadcasterWg sync.WaitGroup
	broadcasterWg.Add(1)
	go func() {
		defer broadcasterWg.Done()
		msg := &TransactionFeedMessage{Version: TransactionFeedV1}
		for {
			select {
			case <-stopBroadcast:
				return
			default:
				s.BroadcastTransaction(msg)
				time.Sleep(time.Millisecond)
			}
		}
	}()

	const numClients = 20
	var clientWg sync.WaitGroup
	for i := 0; i < numClients; i++ {
		clientWg.Add(1)
		go func(i int) {
			defer clientWg.Done()

			time.Sleep(time.Duration(i%5) * 5 * time.Millisecond)
			cctx, ccancel := context.WithTimeout(ctx, 5*time.Second)
			defer ccancel()
			conn, _, err := websocket.Dial(cctx, url, nil)
			if err != nil {
				t.Errorf("client %d: dial failed: %v", i, err)
				return
			}

			for j := 0; j < i%5; j++ {
				if _, _, err := conn.Read(cctx); err != nil {
					break
				}
			}
			_ = conn.Close(websocket.StatusNormalClosure, "")
		}(i)
	}
	clientWg.Wait()
	close(stopBroadcast)
	broadcasterWg.Wait()

	deadline := time.Now().Add(3 * time.Second)
	for s.ClientCount() != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if c := s.ClientCount(); c != 0 {
		t.Fatalf("ClientCount = %d after all clients disconnected, want 0", c)
	}
}
