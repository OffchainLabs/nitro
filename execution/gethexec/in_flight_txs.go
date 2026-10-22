// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	inFlightTxsSizeGauge                 = metrics.NewRegisteredGauge("arb/sequencer/inflight/size/gauge", nil)
	inFlightTxsAlreadyKnownCounter       = metrics.NewRegisteredCounter("arb/sequencer/inflight/rejected/alreadyknown/counter", nil)
	inFlightTxsNonceInFlightCounter      = metrics.NewRegisteredCounter("arb/sequencer/inflight/rejected/nonceinflight/counter", nil)
	inFlightTxsReplaceUnderpricedCounter = metrics.NewRegisteredCounter("arb/sequencer/inflight/rejected/replaceunderpriced/counter", nil)
	inFlightTxsReplacedCounter           = metrics.NewRegisteredCounter("arb/sequencer/inflight/replaced/counter", nil)
)

var ErrNonceInFlight = errors.New("another transaction with the same nonce is already in flight")

// Only the RPC path takes the mutex; the block-building thread must never wait on it.
type inFlightTxs struct {
	priceBump uint64
	mutex     sync.Mutex
	entries   map[addressAndNonce]*inFlightEntry
}

type inFlightEntry struct {
	latest  *types.Transaction
	holders int
}

func newInFlightTxs(priceBump uint64) *inFlightTxs {
	return &inFlightTxs{
		priceBump: priceBump,
		entries:   make(map[addressAndNonce]*inFlightEntry),
	}
}

// admit returns the release func the caller must run once its submission has returned.
func (f *inFlightTxs) admit(sender common.Address, tx *types.Transaction, strategy orderingStrategy) (func(), error) {
	key := addressAndNonce{sender, tx.Nonce()}
	f.mutex.Lock()
	defer f.mutex.Unlock()
	entry, exists := f.entries[key]
	if !exists {
		f.entries[key] = &inFlightEntry{latest: tx, holders: 1}
		inFlightTxsSizeGauge.Update(int64(len(f.entries)))
		return func() { f.release(key) }, nil
	}
	if entry.latest.Hash() == tx.Hash() {
		inFlightTxsAlreadyKnownCounter.Inc(1)
		return nil, txpool.ErrAlreadyKnown
	}
	if strategy != orderingStrategyPGA {
		inFlightTxsNonceInFlightCounter.Inc(1)
		return nil, ErrNonceInFlight
	}
	if !isFeeBumped(entry.latest, tx, f.priceBump) {
		inFlightTxsReplaceUnderpricedCounter.Inc(1)
		return nil, txpool.ErrReplaceUnderpriced
	}
	// The replaced tx stays queued, so its holder must keep the entry alive until it returns.
	inFlightTxsReplacedCounter.Inc(1)
	entry.latest = tx
	entry.holders++
	return func() { f.release(key) }, nil
}

func (f *inFlightTxs) release(key addressAndNonce) {
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
	inFlightTxsSizeGauge.Update(int64(len(f.entries)))
}

func (f *inFlightTxs) len() int {
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
