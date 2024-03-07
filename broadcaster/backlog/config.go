package backlog

import (
	flag "github.com/spf13/pflag"
)

type ConfigFetcher func() *Config

type Config struct {
	SegmentLimit int `koanf:"segment-limit" reload:"hot"`
}

func AddOptions(prefix string, f *flag.FlagSet) {
	f.Int(prefix+".segment-limit", DefaultConfig.SegmentLimit, "the maximum number of messages each segment within the backlog can contain")
}

var (
	DefaultConfig = Config{
		SegmentLimit: 240,
	}
	DefaultTestConfig = Config{
		SegmentLimit: 3,
	}
)
