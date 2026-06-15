// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package transactionfeed

import "testing"

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
