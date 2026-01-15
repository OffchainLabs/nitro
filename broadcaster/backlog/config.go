package backlog

import (
	"github.com/spf13/pflag"
)

type ConfigFetcher func() *Config

type Config struct {
	SegmentLimit          int  `koanf:"segment-limit" reload:"hot"`
	EnableBacklogDeepCopy bool `koanf:"enable-backlog-deep-copy" reload:"hot"`
}

func AddOptions(prefix string, f *pflag.FlagSet) {
	f.Int(prefix+".segment-limit", DefaultConfig.SegmentLimit, "the maximum number of messages each segment within the backlog can contain")
	f.Bool(prefix+".enable-backlog-deep-copy", DefaultConfig.EnableBacklogDeepCopy, "enable deep copying of L2 messages for memory profiling (debug only)")
}

var (
	DefaultConfig = Config{
		SegmentLimit:          240,
		EnableBacklogDeepCopy: false,
	}
	DefaultTestConfig = Config{
		SegmentLimit:          3,
		EnableBacklogDeepCopy: false,
	}
)
