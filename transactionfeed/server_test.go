// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package transactionfeed

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/ethereum/go-ethereum/metrics"
)

// Histogram samples silently no-op unless metrics are enabled, so enable them
// for the metric assertions in this package.
//
// Tests in this package must not use t.Parallel(): they assert on the shared
// package-level metrics, which would race across concurrently running tests.
func TestMain(m *testing.M) {
	metrics.Enable()
	os.Exit(m.Run())
}

func TestBroadcastDroppedCounter(t *testing.T) {
	cfg := DefaultServerConfig
	cfg.BroadcastBuf = 1
	s := NewServer(cfg, make(chan error, 1))

	msg := &TransactionFeedMessage{Version: TransactionFeedV1}
	droppedStart := broadcastDroppedCounter.Snapshot().Count()
	sentStart := broadcastSentCounter.Snapshot().Count()
	sizeCountStart := messageSizeBytesHistogram.Snapshot().Count()
	broadcastQueueDepthGauge.Update(0)

	s.BroadcastTransaction(msg)
	s.BroadcastTransaction(msg)
	s.BroadcastTransaction(msg)

	if delta := broadcastDroppedCounter.Snapshot().Count() - droppedStart; delta < 2 {
		t.Fatalf("expected >= 2 drops, got delta=%d", delta)
	}
	if delta := broadcastSentCounter.Snapshot().Count() - sentStart; delta < 1 {
		t.Fatalf("expected >= 1 sent, got delta=%d", delta)
	}
	if delta := messageSizeBytesHistogram.Snapshot().Count() - sizeCountStart; delta < 3 {
		t.Fatalf("expected >= 3 size samples, got delta=%d", delta)
	}
	if depth := broadcastQueueDepthGauge.Snapshot().Value(); depth != 1 {
		t.Fatalf("expected queue depth 1, got %d", depth)
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
	s := NewServer(cfg, make(chan error, 1))
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
	connectedTotalStart := clientsConnectedTotalCounter.Snapshot().Count()
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

	connectedTotalDelta := func() int64 {
		return clientsConnectedTotalCounter.Snapshot().Count() - connectedTotalStart
	}
	deadline := time.Now().Add(3 * time.Second)
	for (s.ClientCount() != 0 || connectedTotalDelta() < numClients) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if c := s.ClientCount(); c != 0 {
		t.Fatalf("ClientCount = %d after all clients disconnected, want 0", c)
	}
	if delta := connectedTotalDelta(); delta < numClients {
		t.Fatalf("connected total counter delta = %d, want >= %d", delta, numClients)
	}
}

// TestServerFatalOnServeFailure verifies that an unexpected http server
// failure (as opposed to a graceful StopAndWait) is reported on the fatal
// error channel so the hosting node shuts down instead of running with a
// silently dead feed.
func TestServerFatalOnServeFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := DefaultServerConfig
	cfg.Addr = "127.0.0.1"
	cfg.Port = "0"
	fatalErrChan := make(chan error, 1)
	s := NewServer(cfg, fatalErrChan)
	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer s.StopAndWait()

	// Close the listener out from under the http server to make Serve fail
	// without going through the graceful-shutdown path.
	if err := s.listener.Close(); err != nil {
		t.Fatalf("failed to close listener: %v", err)
	}

	select {
	case err := <-fatalErrChan:
		if err == nil {
			t.Fatal("got nil fatal error, want non-nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for fatal error after listener failure")
	}
}
