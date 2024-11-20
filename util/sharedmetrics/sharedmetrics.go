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
