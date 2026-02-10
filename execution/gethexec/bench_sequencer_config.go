// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// DANGER! this file is included in all builds
// DANGER! do not place any benchmarking-sequencer tag logic here

package gethexec

type BenchmarkingSequencerConfig struct {
	Enable bool `koanf:"enable"`
}

var BenchmarkingSequencerConfigDefault = BenchmarkingSequencerConfig{
	Enable: false,
}
