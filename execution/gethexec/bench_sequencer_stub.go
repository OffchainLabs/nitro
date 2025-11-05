//go:build !benchsequencer

package gethexec

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/spf13/pflag"
)

func BenchSequencerConfigAddOptions(_ string, _ *pflag.FlagSet) {
	// don't add any options
}

func (c *BenchSequencerConfig) Validate() error {
	if c.Enable {
		log.Warn("BenchSequencer is not supported in this build")
	}
	return nil
}

func NewBenchSequencer(sequencer *Sequencer) (TransactionPublisher, interface{}) {
	// do nothing
	return sequencer, nil
}
