// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/util/containers"
)

var nonceFailureCacheOverflowCounter = metrics.NewRegisteredCounter("arb/sequencer/noncefailurecache/overflow", nil)

type addressAndNonce struct {
	address common.Address
	nonce   uint64
}

type nonceFailure struct {
	queueItem txQueueItem
	nonceErr  error
	expiry    time.Time
	revived   bool
}

// nonceFailureCache parks nonce-too-high txs until their predecessor arrives. It owns its
// mutex, so callers need no external lock. The eviction hook runs under the mutex and must
// not touch the cache.
type nonceFailureCache struct {
	mutex     sync.Mutex
	cache     *containers.LruCache[addressAndNonce, *nonceFailure]
	getExpiry func() time.Duration
}

func newNonceFailureCache(size int, getExpiry func() time.Duration) *nonceFailureCache {
	return &nonceFailureCache{
		cache:     containers.NewLruCacheWithOnEvict(size, onNonceFailureEvict),
		getExpiry: getExpiry,
	}
}

func onNonceFailureEvict(_ addressAndNonce, failure *nonceFailure) {
	if failure.revived {
		return
	}
	queueItem := failure.queueItem
	if err := queueItem.ctx.Err(); err != nil {
		queueItem.returnResult(err)
		return
	}
	queueItem.returnResult(failure.nonceErr)
}

func (c *nonceFailureCache) Add(err NonceError, queueItem txQueueItem) {
	// A parked tx forfeits its anti-starvation boost: it re-enters the PGA auction fresh when revived.
	queueItem.ResetBoost()
	expiry := queueItem.firstAppearance.Add(c.getExpiry())
	key := addressAndNonce{err.sender, err.txNonce}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.cache.Contains(key) || time.Now().After(expiry) {
		queueItem.returnResult(err)
		return
	}
	val := &nonceFailure{
		queueItem: queueItem,
		nonceErr:  err,
		expiry:    expiry,
		revived:   false,
	}
	evicted := c.cache.Add(key, val)
	if evicted {
		nonceFailureCacheOverflowCounter.Inc(1)
	}
}

// Take removes and returns the failure parked under key, marking it revived so the eviction
// hook stays silent; the caller owns the result.
func (c *nonceFailureCache) Take(key addressAndNonce) (*nonceFailure, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	failure, ok := c.cache.Get(key)
	if !ok {
		return nil, false
	}
	failure.revived = true
	c.cache.Remove(key)
	return failure, true
}

// TakeOldest removes and returns the oldest parked failure, marking it revived so the
// eviction hook stays silent.
func (c *nonceFailureCache) TakeOldest() (*nonceFailure, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	_, failure, ok := c.cache.GetOldest()
	if !ok {
		return nil, false
	}
	failure.revived = true
	c.cache.RemoveOldest()
	return failure, true
}

// TakeExpired removes and returns the oldest parked failure if it has expired, marking it
// revived so the eviction hook stays silent.
func (c *nonceFailureCache) TakeExpired() (*nonceFailure, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	_, failure, ok := c.cache.GetOldest()
	if !ok || time.Until(failure.expiry) > 0 {
		return nil, false
	}
	failure.revived = true
	c.cache.RemoveOldest()
	return failure, true
}

func (c *nonceFailureCache) Len() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.cache.Len()
}

func (c *nonceFailureCache) Resize(newSize int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache.Resize(newSize)
}
