// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package sequencer

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

type pgaBid struct {
	label string
	tx    *types.Transaction

	seqResult error
	receipt   *types.Receipt
}

// txSize mirrors the sequencer's queueItem.txSize computation.
func txSize(tx *types.Transaction) int {
	return int(tx.Size()) // #nosec G115
}

// enqueueAndSequence submits the bids to the paused sequencer, whose queue must start empty,
// and resumes it once they are all queued.
func enqueueAndSequence(env *systest.Env, bids []*pgaBid) {
	sequencer := env.L2.ExecNode.Sequencer
	sequencer.Pause()
	// Submissions block until sequenced, so the burst needs concurrent senders.
	var wg sync.WaitGroup
	defer func() {
		sequencer.Activate()
		wg.Wait() // Workers return once env.Ctx dies, so Wait cannot hang
		env.Require(env.Ctx.Err(), "burst interrupted")
	}()
	env.Zero(sequencer.QueuedTxCount(), "sequencer queue is not empty")
	for i, bid := range bids {
		done := make(chan struct{})
		wg.Go(func() {
			defer close(done)
			bid.seqResult = env.L2.Client.SendTransaction(env.Ctx, bid.tx)
		})
		isDone := func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}
		// Wait for each tx to reach the queue before sending the next, pinning the queue order.
		env.WaitFor(fmt.Sprintf("%d txs in the sequencer queue", i+1), func() bool {
			return isDone() || sequencer.QueuedTxCount() >= i+1
		})
		if isDone() {
			env.True(false, "bid %s left the paused sequencer with result %v instead of staying in the queue", bid.label, bid.seqResult)
		}
	}
}

func requireIncluded(env *systest.Env, bids []*pgaBid) {
	for _, bid := range bids {
		env.Require(bid.seqResult, "bid %s", bid.label)
		bid.receipt = env.L2.EnsureTxSucceeded(bid.tx)
	}
}

func inclusionOrder(bids []*pgaBid) []int {
	order := make([]int, len(bids))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool {
		ra, rb := bids[order[a]].receipt, bids[order[b]].receipt
		if cmp := ra.BlockNumber.Cmp(rb.BlockNumber); cmp != 0 {
			return cmp < 0
		}
		return ra.TransactionIndex < rb.TransactionIndex
	})
	return order
}

func describeInclusions(bids []*pgaBid) string {
	var out []string
	for _, i := range inclusionOrder(bids) {
		out = append(out, fmt.Sprintf("[block %v tx %d %s]", bids[i].receipt.BlockNumber, bids[i].receipt.TransactionIndex, bids[i].label))
	}
	return strings.Join(out, " ")
}
