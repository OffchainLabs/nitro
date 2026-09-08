// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func TestSyncMonitorRequiresFeedBeforeReportingSynced(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, streamer, _, _ := NewTransactionStreamerForTest(t, ctx, common.Address{})
	syncMonitor := NewSyncMonitor(func() *SyncMonitorConfig { return &TestSyncMonitorConfig })
	syncMonitor.Initialize(nil, streamer, nil, true)
	syncMonitor.Start(ctx)
	defer syncMonitor.StopAndWait()

	requireEventually(t, func() bool {
		return syncMonitor.SyncTargetMessageCount() > 0
	}, "sync target was never initialized")

	if syncMonitor.Synced() {
		t.Fatal("expected not synced before feed delivers a message")
	}

	streamer.broadcasterQueuedMessagesFirstMsgIdx.Store(1)

	requireEventually(t, func() bool {
		return syncMonitor.Synced()
	}, "expected synced after feed pending message count becomes nonzero")
}

func TestSyncMonitorAllowsLocalSyncWithoutFeedOrCoordinator(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, streamer, _, _ := NewTransactionStreamerForTest(t, ctx, common.Address{})
	syncMonitor := NewSyncMonitor(func() *SyncMonitorConfig { return &TestSyncMonitorConfig })
	syncMonitor.Initialize(nil, streamer, nil, false)
	syncMonitor.Start(ctx)
	defer syncMonitor.StopAndWait()

	requireEventually(t, func() bool {
		return syncMonitor.SyncTargetMessageCount() > 0
	}, "sync target was never initialized")

	if !syncMonitor.Synced() {
		t.Fatal("expected local-only node to report synced without feed input")
	}
}

func requireEventually(t *testing.T, condition func() bool, message string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal(message)
}
