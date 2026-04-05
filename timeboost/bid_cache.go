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

func (bc *bidCache) size() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return len(bc.bidsByBidder)
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

// computeTopTwo returns the highest and second-highest bids. When amounts are
// equal, ties are broken by comparing BigIntHash (a deterministic hash of the
// bid) — the higher hash wins, ensuring fair tiebreaking independent of map
// iteration order.
func (bc *bidCache) computeTopTwo() *auctionResult {
	result := &auctionResult{}

	// hashBeats returns true if a's BigIntHash is greater than b's.
	hashBeats := func(a, b *ValidatedBid) bool {
		return a.BigIntHash(bc.auctionContractDomainSeparator).Cmp(b.BigIntHash(bc.auctionContractDomainSeparator)) > 0
	}

	for _, bid := range bc.bidsByBidder {
		if result.firstPlace == nil {
			result.firstPlace = bid
		} else if bid.Amount.Cmp(result.firstPlace.Amount) > 0 {
			result.secondPlace = result.firstPlace
			result.firstPlace = bid
		} else if bid.Amount.Cmp(result.firstPlace.Amount) == 0 {
			if hashBeats(bid, result.firstPlace) {
				result.secondPlace = result.firstPlace
				result.firstPlace = bid
			} else if result.secondPlace == nil || hashBeats(bid, result.secondPlace) {
				result.secondPlace = bid
			}
		} else if result.secondPlace == nil || bid.Amount.Cmp(result.secondPlace.Amount) > 0 {
			result.secondPlace = bid
		} else if bid.Amount.Cmp(result.secondPlace.Amount) == 0 {
			if hashBeats(bid, result.secondPlace) {
				result.secondPlace = bid
			}
		}
	}

	return result
}
