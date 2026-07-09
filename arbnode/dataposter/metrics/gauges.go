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
)
