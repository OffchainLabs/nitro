// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

// testAfterOwnerNonceRead, when set, runs inside BuildL2OnL1 immediately after
// it samples Owner's nonce and before that value is used to send a transaction.
// Only armNonceRaceRepro sets it.
var testAfterOwnerNonceRead func(t *testing.T, b *NodeBuilder, nonceRead uint64)

// armNonceRaceRepro makes NIT-5360 reproduce on demand instead of only under CI
// load. BuildL2OnL1 samples Owner's nonce from a node that is still ingesting
// the parent chain inbox, pins that value, and sends with it; if ingestion
// finishes in between, the transaction is rejected as "nonce too low".
//
// Locally the node always finishes ingesting before the sample is taken, so the
// window never opens. This forces it open deterministically:
//
//  1. cap ingestion just below the last posted batch, so the node comes up
//     genuinely behind and the sample is taken mid-catch-up;
//  2. once the sample is taken, lift the cap and wait for the node to catch up,
//     so the pinned value is stale by the time it is used.
//
// Call immediately before the BuildL2OnL1 that should fail. Registers its own
// cleanup.
func armNonceRaceRepro(t *testing.T, builder *NodeBuilder) {
	t.Helper()

	seqInbox, err := bridgegen.NewSequencerInboxCaller(builder.addresses.SequencerInbox, builder.L1.Client)
	Require(t, err)
	posted, err := seqInbox.BatchCount(&bind.CallOpts{Context: builder.ctx})
	Require(t, err)
	if !posted.IsInt64() || posted.Int64() < 2 {
		t.Fatalf("nonce-race repro needs at least 2 posted batches to hold one back, got %v", posted)
	}
	// Hold back the final batch: it carries the Owner transfer that takes the
	// nonce to its last value, so the node settles one short of the truth.
	limit := posted.Int64() - 1
	arbnode.SetTestIngestLimit(limit)
	t.Logf("nonce-race repro: capping ingestion at batch %d of %d posted", limit, posted.Int64())

	t.Cleanup(func() {
		arbnode.SetTestIngestLimit(-1)
		testAfterOwnerNonceRead = nil
	})

	testAfterOwnerNonceRead = func(t *testing.T, b *NodeBuilder, nonceRead uint64) {
		t.Helper()
		// Fire once: later builds must not be perturbed.
		testAfterOwnerNonceRead = nil

		t.Logf("nonce-race repro: BuildL2OnL1 sampled Owner nonce=%d while behind", nonceRead)
		arbnode.SetTestIngestLimit(-1)

		owner := b.L2Info.GetAddress("Owner")
		deadline := time.Now().Add(30 * time.Second)
		for {
			current, err := b.L2.Client.PendingNonceAt(b.ctx, owner)
			Require(t, err)
			if current > nonceRead {
				t.Logf("nonce-race repro: node caught up, Owner nonce %d -> %d; the pinned %d is now stale",
					nonceRead, current, nonceRead)
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("nonce-race repro: Owner nonce stuck at %d after lifting the cap; "+
					"the held-back batch may no longer carry an Owner transaction", current)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}
