// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// A tx parked in the nonce-failure cache forfeits its PGA anti-starvation boost, so it re-enters
// the auction fresh when revived.
func TestNonceFailureCacheDropsPgaBoost(t *testing.T) {
	cache := newNonceFailureCache(16, func() time.Duration { return time.Hour })
	item, _ := makeTestQueueItem(t, 1, testBaseFee)
	item.ApplyRoundBoundary(25)

	nonceErr := NonceError{sender: common.Address{1}, txNonce: 1, stateNonce: 0}
	cache.Add(nonceErr, item)

	failure, ok := cache.Take(addressAndNonce{nonceErr.sender, nonceErr.txNonce})
	if !ok {
		t.Fatal("Take found no parked failure")
	}
	if failure.queueItem.GetPriority() != 0 {
		t.Fatalf("revived priority = %d, want 0 (parked txs forfeit their boost)", failure.queueItem.GetPriority())
	}
}
