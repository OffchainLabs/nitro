package arbutil

import "github.com/ethereum/go-ethereum/metrics"

var (
	latestSequenceNumberGauge = metrics.NewRegisteredGauge("arb/sequencennumber/latest", nil)
)

func UpdateSequenceNumberGauge(sequenceNumber MessageIndex) {
	latestSequenceNumberGauge.Update(int64(sequenceNumber))
}
