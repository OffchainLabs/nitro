package timeboost

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type bidCache struct {
	auctionContractDomainSeparator [32]byte
	sync.RWMutex
	bidsByExpressLaneControllerAddr map[common.Address]*ValidatedBid
}

func newBidCache(auctionContractDomainSeparator [32]byte) *bidCache {
	return &bidCache{
		bidsByExpressLaneControllerAddr: make(map[common.Address]*ValidatedBid),
		auctionContractDomainSeparator:  auctionContractDomainSeparator,
	}
}

func (bc *bidCache) add(bid *ValidatedBid) {
	bc.Lock()
	defer bc.Unlock()
	bc.bidsByExpressLaneControllerAddr[bid.ExpressLaneController] = bid
}

// TwoTopBids returns the top two bids for the given chain ID and round
type auctionResult struct {
	firstPlace  *ValidatedBid
	secondPlace *ValidatedBid
}

func (bc *bidCache) size() int {
	bc.RLock()
	defer bc.RUnlock()
	return len(bc.bidsByExpressLaneControllerAddr)

}

// topTwoBids returns the top two bids in the cache.
func (bc *bidCache) topTwoBids() *auctionResult {
	bc.RLock()
	defer bc.RUnlock()

	result := &auctionResult{}

	for _, bid := range bc.bidsByExpressLaneControllerAddr {
		if result.firstPlace == nil {
			result.firstPlace = bid
		} else if bid.Amount.Cmp(result.firstPlace.Amount) > 0 {
			result.secondPlace = result.firstPlace
			result.firstPlace = bid
		} else if bid.Amount.Cmp(result.firstPlace.Amount) == 0 {
			if bid.BigIntHash(bc.auctionContractDomainSeparator).Cmp(result.firstPlace.BigIntHash(bc.auctionContractDomainSeparator)) > 0 {
				result.secondPlace = result.firstPlace
				result.firstPlace = bid
			} else if result.secondPlace == nil || bid.BigIntHash(bc.auctionContractDomainSeparator).Cmp(result.secondPlace.BigIntHash(bc.auctionContractDomainSeparator)) > 0 {
				result.secondPlace = bid
			}
		} else if result.secondPlace == nil || bid.Amount.Cmp(result.secondPlace.Amount) > 0 {
			result.secondPlace = bid
		} else if bid.Amount.Cmp(result.secondPlace.Amount) == 0 {
			if bid.BigIntHash(bc.auctionContractDomainSeparator).Cmp(result.secondPlace.BigIntHash(bc.auctionContractDomainSeparator)) > 0 {
				result.secondPlace = bid
			}
		}
	}

	return result
}
