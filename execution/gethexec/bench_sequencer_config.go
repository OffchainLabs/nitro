// DANGER! this file is included in all builds
// DANGER! do not place any of the experimental logic and features here

package gethexec

type BenchSequencerConfig struct {
	Enable bool `koanf:"enable"`
}

var BenchSequencerConfigDefault = BenchSequencerConfig{
	Enable: false,
}
