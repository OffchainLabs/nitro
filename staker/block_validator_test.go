// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package staker

import (
	"context"
	"errors"
	"testing"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/validator"
)

type catchUpCancellationTracker struct {
	InboxTrackerInterface
	batchCount     uint64
	batchMsgCounts map[uint64]arbutil.MessageIndex
}

func (t *catchUpCancellationTracker) GetBatchCount() (uint64, error) {
	return t.batchCount, nil
}

func (t *catchUpCancellationTracker) GetBatchMessageCount(batch uint64) (arbutil.MessageIndex, error) {
	return t.batchMsgCounts[batch], nil
}

type catchUpCancellationStreamer struct {
	TransactionStreamerInterface
	cancel context.CancelFunc
	calls  int
}

func (s *catchUpCancellationStreamer) GetProcessedMessageCount(ctx context.Context) (arbutil.MessageIndex, error) {
	s.calls++
	if s.calls == 1 {
		s.cancel()
		return 5, nil
	}
	return 0, ctx.Err()
}

func TestCheckValidatedGSCaughtUpPropagatesDiagnosticCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	streamer := &catchUpCancellationStreamer{cancel: cancel}
	tracker := &catchUpCancellationTracker{
		batchCount:     2,
		batchMsgCounts: map[uint64]arbutil.MessageIndex{1: 10},
	}
	blockValidator := &BlockValidator{
		StatelessBlockValidator: &StatelessBlockValidator{
			inboxTracker: tracker,
			streamer:     streamer,
		},
		lastValidGS: validator.GoGlobalState{Batch: 2},
	}

	caughtUp, err := blockValidator.checkValidatedGSCaughtUp(ctx)
	if caughtUp {
		t.Fatal("expected validator not to be caught up")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}
