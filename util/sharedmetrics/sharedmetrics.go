package sharedmetrics

import (
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/offchainlabs/nitro/arbutil"
)

var (
	latestSequenceNumberGauge  = metrics.NewRegisteredGauge("arb/sequencennumber/latest", nil)
	sequenceNumberInBlockGauge = metrics.NewRegisteredGauge("arb/sequencennumber/inblock", nil)
)

func UpdateSequenceNumberGauge(sequenceNumber arbutil.MessageIndex) {
	latestSequenceNumberGauge.Update(int64(sequenceNumber))
}
func UpdateSequenceNumberInBlockGauge(sequenceNumber arbutil.MessageIndex) {
	sequenceNumberInBlockGauge.Update(int64(sequenceNumber))
}
