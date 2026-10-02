// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package legacystaker

import (
	"context"
	"errors"
	"testing"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/staker"
)

type currentPositionTestStreamer struct {
	staker.TransactionStreamerInterface
	head arbutil.MessageIndex
	err  error
}

func (s *currentPositionTestStreamer) GetProcessedMessageCount(context.Context) (arbutil.MessageIndex, error) {
	return s.head, s.err
}

type currentPositionTestTracker struct {
	staker.InboxTrackerInterface
	batchCounts map[uint64]arbutil.MessageIndex
	batch       uint64
	found       bool
	err         error
}

func (t *currentPositionTestTracker) FindInboxBatchContainingMessage(arbutil.MessageIndex) (uint64, bool, error) {
	return t.batch, t.found, t.err
}

func (t *currentPositionTestTracker) GetBatchMessageCount(batch uint64) (arbutil.MessageIndex, error) {
	return t.batchCounts[batch], nil
}

func TestCurrentGlobalStatePosition(t *testing.T) {
	t.Run("available without block validator", func(t *testing.T) {
		l1Validator := &L1Validator{
			txStreamer: &currentPositionTestStreamer{head: 4},
			inboxTracker: &currentPositionTestTracker{
				batch:       1,
				found:       true,
				batchCounts: map[uint64]arbutil.MessageIndex{0: 2, 1: 5},
			},
			blockValidator: nil,
		}

		current, ok := l1Validator.currentGlobalStatePosition(t.Context())
		if !ok {
			t.Fatal("expected current global state position to be available")
		}
		expected := staker.GlobalStatePosition{BatchNumber: 1, PosInBatch: 2}
		if current != expected {
			t.Fatalf("expected current position %v, got %v", expected, current)
		}
	})

	t.Run("unavailable when processed count fails", func(t *testing.T) {
		l1Validator := &L1Validator{
			txStreamer:   &currentPositionTestStreamer{err: errors.New("execution unavailable")},
			inboxTracker: &currentPositionTestTracker{},
		}

		_, ok := l1Validator.currentGlobalStatePosition(t.Context())
		if ok {
			t.Fatal("expected current global state position to be unavailable")
		}
	})

	t.Run("unavailable when batch is unknown", func(t *testing.T) {
		l1Validator := &L1Validator{
			txStreamer:   &currentPositionTestStreamer{head: 4},
			inboxTracker: &currentPositionTestTracker{},
		}

		_, ok := l1Validator.currentGlobalStatePosition(t.Context())
		if ok {
			t.Fatal("expected current global state position to be unavailable")
		}
	})
}
