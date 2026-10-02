// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package timeboost

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type bidCache struct {
	auctionContractDomainSeparator [32]byte
	mu                             sync.RWMutex
	bidsByBidder                   map[common.Address]*ValidatedBid
}

func newBidCache(auctionContractDomainSeparator [32]byte) *bidCache {
	return &bidCache{
		bidsByBidder:                   make(map[common.Address]*ValidatedBid),
		auctionContractDomainSeparator: auctionContractDomainSeparator,
	}
}

func (bc *bidCache) add(bid *ValidatedBid) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.bidsByBidder[bid.Bidder] = bid
}

type auctionResult struct {
	firstPlace  *ValidatedBid
	secondPlace *ValidatedBid
}

func (bc *bidCache) clear() {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.bidsByBidder = make(map[common.Address]*ValidatedBid)
}

func (bc *bidCache) size() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return len(bc.bidsByBidder)
}

func (bc *bidCache) getBid(bidder common.Address) *ValidatedBid {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.bidsByBidder[bidder]
}

// topTwoBids returns the top two bids without modifying the cache.
func (bc *bidCache) topTwoBids() *auctionResult {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.computeTopTwo()
}

// topTwoBidsAndClear atomically reads the top two bids and clears the cache,
// so a bid is never silently dropped between read and clear.
func (bc *bidCache) topTwoBidsAndClear() *auctionResult {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	result := bc.computeTopTwo()
	bc.bidsByBidder = make(map[common.Address]*ValidatedBid)
	return result
}

// beats reports whether bid a should be ordered ahead of bid b: the higher
// amount wins, and equal amounts are broken by the higher BigIntHash. This is
// the same total order the auction contract enforces in resolveMultiBidAuction,
// which reverts with TieBidsWrongOrder when two equal bids are submitted with
// the lower hash first.
func (bc *bidCache) beats(a, b *ValidatedBid) bool {
	if c := a.Amount.Cmp(b.Amount); c != 0 {
		return c > 0
	}
	return a.BigIntHash(bc.auctionContractDomainSeparator).Cmp(b.BigIntHash(bc.auctionContractDomainSeparator)) > 0
}

// computeTopTwo returns the highest and second-highest bids under that order,
// independently of the map iteration order.
func (bc *bidCache) computeTopTwo() *auctionResult {
	result := &auctionResult{}
	for _, bid := range bc.bidsByBidder {
		switch {
		case result.firstPlace == nil || bc.beats(bid, result.firstPlace):
			result.secondPlace = result.firstPlace
			result.firstPlace = bid
		case result.secondPlace == nil || bc.beats(bid, result.secondPlace):
			result.secondPlace = bid
		}
	}
	return result
}
