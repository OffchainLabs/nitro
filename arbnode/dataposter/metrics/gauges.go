// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package datapostermetrics

import (
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	LatestFinalizedNonceGauge     = metrics.NewRegisteredGauge("arb/dataposter/nonce/finalized", nil)
	LatestSoftConfirmedNonceGauge = metrics.NewRegisteredGauge("arb/dataposter/nonce/softconfirmed", nil)
	LatestUnconfirmedNonceGauge   = metrics.NewRegisteredGauge("arb/dataposter/nonce/unconfirmed", nil)
	TotalQueueLengthGauge         = metrics.NewRegisteredGauge("arb/dataposter/queue/length", nil)
	TotalQueueWeightGauge         = metrics.NewRegisteredGauge("arb/dataposter/queue/weight", nil)
	TipBumpsCounter               = metrics.NewRegisteredCounter("arb/dataposter/tip/bumps", nil)
	DeferredTipBumpsCounter       = metrics.NewRegisteredCounter("arb/dataposter/tip/bumps/deferred", nil)
	LastTipCapGauge               = metrics.NewRegisteredGauge("arb/dataposter/tip/last", nil)
	SuggestedTipCapGauge          = metrics.NewRegisteredGauge("arb/dataposter/tip/suggested", nil)
	LastFeeCapGauge               = metrics.NewRegisteredGauge("arb/dataposter/fee/last", nil)
	LastBlobFeeCapGauge           = metrics.NewRegisteredGauge("arb/dataposter/blobfee/last", nil)
)
