//go:build !benchsequencer

package gethexec

import (
	"errors"

	"github.com/spf13/pflag"
)

func BenchSequencerConfigAddOptions(_ string, _ *pflag.FlagSet) {
	// don't add any options
}

func (c *BenchSequencerConfig) Validate() error {
	if c.Enable {
		return errors.New("BenchSeqeuncer is not supported in this build")
	}
	return nil
}

func NewBenchSequencer(sequencer *Sequencer) (TransactionPublisher, interface{}) {
	// do nothing
	return sequencer, nil
}
