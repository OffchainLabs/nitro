// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	pendingTxsCountGauge                = metrics.NewRegisteredGauge("arb/sequencer/pending/count/gauge", nil)
	pendingTxsCountHistogram            = metrics.NewRegisteredHistogram("arb/sequencer/pending/count/histogram", nil, metrics.NewBoundedHistogramSample())
	pendingTxsAlreadyKnownCounter       = metrics.NewRegisteredCounter("arb/sequencer/pending/rejected/alreadyknown/counter", nil)
	pendingTxsNoncePendingCounter       = metrics.NewRegisteredCounter("arb/sequencer/pending/rejected/noncepending/counter", nil)
	pendingTxsReplaceUnderpricedCounter = metrics.NewRegisteredCounter("arb/sequencer/pending/rejected/replaceunderpriced/counter", nil)
	pendingTxsReplacedCounter           = metrics.NewRegisteredCounter("arb/sequencer/pending/replaced/counter", nil)
)

// Only the RPC path takes the mutex; the block-building thread must never wait on it.
type pendingTxs struct {
	priceBump uint64
	mutex     sync.Mutex
	entries   map[addressAndNonce]*pendingEntry
}

type pendingEntry struct {
	latest  *types.Transaction
	holders int
}

func newPendingTxs(priceBump uint64) *pendingTxs {
	return &pendingTxs{
		priceBump: priceBump,
		entries:   make(map[addressAndNonce]*pendingEntry),
	}
}

// admit returns the release func the caller must run once its submission has returned.
func (f *pendingTxs) admit(sender common.Address, tx *types.Transaction, strategy orderingStrategy) (func(), error) {
	key := addressAndNonce{sender, tx.Nonce()}
	f.mutex.Lock()
	defer f.mutex.Unlock()
	entry, exists := f.entries[key]
	if !exists {
		f.entries[key] = &pendingEntry{latest: tx, holders: 1}
		f.updateCountMetrics()
		return func() { f.release(key) }, nil
	}
	if entry.latest.Hash() == tx.Hash() {
		pendingTxsAlreadyKnownCounter.Inc(1)
		return nil, txpool.ErrAlreadyKnown
	}
	if strategy != orderingStrategyPGA {
		pendingTxsNoncePendingCounter.Inc(1)
		// Assume the first transaction will be accepted and the second one fail with nonce-too-low.
		return nil, core.ErrNonceTooLow
	}
	if !isFeeBumped(entry.latest, tx, f.priceBump) {
		pendingTxsReplaceUnderpricedCounter.Inc(1)
		return nil, txpool.ErrReplaceUnderpriced
	}
	// The replaced tx stays queued, so its holder must keep the entry alive until it returns.
	pendingTxsReplacedCounter.Inc(1)
	entry.latest = tx
	entry.holders++
	return func() { f.release(key) }, nil
}

func (f *pendingTxs) release(key addressAndNonce) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	entry, exists := f.entries[key]
	if !exists {
		return
	}
	entry.holders--
	if entry.holders > 0 {
		return
	}
	delete(f.entries, key)
	f.updateCountMetrics()
}

func (f *pendingTxs) updateCountMetrics() {
	count := int64(len(f.entries))
	pendingTxsCountGauge.Update(count)
	pendingTxsCountHistogram.Update(count)
}

func (f *pendingTxs) len() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return len(f.entries)
}

func isFeeBumped(old, replacement *types.Transaction, priceBump uint64) bool {
	if old.GasFeeCapCmp(replacement) >= 0 || old.GasTipCapCmp(replacement) >= 0 {
		return false
	}
	bump := big.NewInt(100 + int64(priceBump)) // #nosec G115
	hundred := big.NewInt(100)
	thresholdFeeCap := new(big.Int).Mul(bump, old.GasFeeCap())
	thresholdFeeCap.Div(thresholdFeeCap, hundred)
	thresholdTip := new(big.Int).Mul(bump, old.GasTipCap())
	thresholdTip.Div(thresholdTip, hundred)
	return replacement.GasFeeCapIntCmp(thresholdFeeCap) >= 0 && replacement.GasTipCapIntCmp(thresholdTip) >= 0
}
