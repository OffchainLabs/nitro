// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package sharedmetrics

import (
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/arbutil"
)

var (
	latestSequenceNumberGauge  = metrics.NewRegisteredGauge("arb/sequencenumber/latest", nil)
	sequenceNumberInBlockGauge = metrics.NewRegisteredGauge("arb/sequencenumber/inblock", nil)
)

func UpdateSequenceNumberGauge(sequenceNumber arbutil.MessageIndex) {
	// #nosec G115
	latestSequenceNumberGauge.Update(int64(sequenceNumber))
}
func UpdateSequenceNumberInBlockGauge(sequenceNumber arbutil.MessageIndex) {
	// #nosec G115
	sequenceNumberInBlockGauge.Update(int64(sequenceNumber))
}
