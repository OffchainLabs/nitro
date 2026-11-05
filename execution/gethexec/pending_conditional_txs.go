// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"sync/atomic"

	"github.com/ethereum/go-ethereum/metrics"
)

var (
	conditionalTxPendingGauge              = metrics.NewRegisteredGauge("arb/txprechecker/conditionaltx/pending/gauge", nil)
	conditionalTxRejectedPendingCapCounter = metrics.NewRegisteredCounter("arb/txprechecker/conditionaltx/rejected/pendingcap/counter", nil)
)

type pendingConditionalTxs struct {
	count atomic.Int64
}

func (c *pendingConditionalTxs) admit(limit int) (func(), bool) {
	for {
		current := c.count.Load()
		if limit > 0 && current >= int64(limit) {
			conditionalTxRejectedPendingCapCounter.Inc(1)
			return nil, false
		}
		if c.count.CompareAndSwap(current, current+1) {
			conditionalTxPendingGauge.Inc(1)
			return c.release, true
		}
	}
}

func (c *pendingConditionalTxs) release() {
	c.count.Add(-1)
	conditionalTxPendingGauge.Dec(1)
}
