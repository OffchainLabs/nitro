// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"testing"
	"time"
)

func TestSequencerDoesntBlockWithoutTransactions(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)

	cleanup := builder.Build(t)
	defer cleanup()

	// This call will time out if the sequencer is blocking
	_, nextSequenceTime := builder.L2.ExecNode.StartSequencing(ctx)
	if nextSequenceTime == time.Duration(0) {
		t.Fatal("Expected non-zero next sequence time")
	}
}
