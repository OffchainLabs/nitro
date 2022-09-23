package arbutil

import "github.com/ethereum/go-ethereum/metrics"

var (
	latestSequenceNumberGauge  = metrics.NewRegisteredGauge("arb/sequencennumber/latest", nil)
	sequenceNumberInBlockGauge = metrics.NewRegisteredGauge("arb/sequencennumber/inblock", nil)
)

func UpdateSequenceNumberGauge(sequenceNumber MessageIndex) {
	latestSequenceNumberGauge.Update(int64(sequenceNumber))
}
func UpdateSequenceNumberInBlockGauge(sequenceNumber MessageIndex) {
	sequenceNumberInBlockGauge.Update(int64(sequenceNumber))
}
