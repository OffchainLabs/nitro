// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"math/rand/v2"

	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/execution/gethexec/pga"
	"github.com/offchainlabs/nitro/util/arbmath"
)

var (
	fifoSizeSamplingSkippedCounter    = metrics.NewRegisteredCounter("arb/sequencer/fifo/sizesampling/skipped", nil)
	fifoSizeSamplingExponentHistogram = metrics.NewRegisteredHistogram("arb/sequencer/fifo/sizesampling/exponent", nil, metrics.NewBoundedHistogramSample())
)

type sizeSampling struct {
	exponent      uint
	minTxSize     int
	maxTxDataSize int
	randomBips    func() arbmath.UBips
	skipped       []txQueueItem
}

func newSizeSampling(config FIFOConfig, maxTxDataSize int, dataLimitedBlockStreak uint) sizeSampling {
	s := sizeSampling{
		minTxSize:     config.SizeSamplingMinTxSize,
		maxTxDataSize: maxTxDataSize,
		randomBips:    defaultRandomBips,
	}
	if config.SizeSamplingMaxExponent > 0 && dataLimitedBlockStreak > config.SizeSamplingGraceBlocks {
		s.exponent = min(dataLimitedBlockStreak-config.SizeSamplingGraceBlocks, config.SizeSamplingMaxExponent)
	}
	return s
}

func defaultRandomBips() arbmath.UBips {
	return arbmath.UBips(rand.Uint64N(uint64(arbmath.OneInUBips)))
}

func (s *sizeSampling) admits(item txQueueItem) bool {
	if s.exponent == 0 || item.txSize <= s.minTxSize {
		return true
	}
	return s.randomBips() < pga.SizeFactorBips(item.txSize, s.maxTxDataSize, s.exponent)
}

// The successors of a skipped tx wait in the sequencer's nonce-failure cache and can expire
// there with a nonce error while the tx is still skipped. That is accepted.
func (s *sizeSampling) skip(item txQueueItem) bool {
	if s.admits(item) {
		return false
	}
	s.skipped = append(s.skipped, item)
	fifoSizeSamplingSkippedCounter.Inc(1)
	return true
}

func (s *sizeSampling) fallback() ([]txQueueItem, bool) {
	if len(s.skipped) == 0 {
		return nil, false
	}
	// NextQueueItem feeds the returned items back through skip. With the exponent kept, a tx
	// the sampling never admits loops forever.
	s.exponent = 0
	skipped := s.skipped
	s.skipped = nil
	return skipped, true
}

func (s *sizeSampling) recordExponent() {
	// #nosec G115
	fifoSizeSamplingExponentHistogram.Update(int64(s.exponent))
}

func (s *sizeSampling) takeSkipped() []txQueueItem {
	skipped := s.skipped
	s.skipped = nil
	return skipped
}
