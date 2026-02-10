// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !benchmarking-sequencer

package gethexec

import (
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"
)

func BenchmarkingSequencerConfigAddOptions(_ string, _ *pflag.FlagSet) {
	// don't add any options
}

func (c *BenchmarkingSequencerConfig) Validate() error {
	if c.Enable {
		log.Warn("benchmarking sequencer requested but not supported in this build (missing benchmarking-sequencer build tag)")
	}
	return nil
}

func NewBenchmarkingSequencer(sequencer *Sequencer) (TransactionPublisher, interface{}) {
	// do nothing
	return sequencer, nil
}
