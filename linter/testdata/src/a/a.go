package a

import (
	"flag"
)

type Config struct {
	// Field name doesn't match koanf tag.
	L2       int `koanf:"chain"`
	LogLevel int `koanf:"log-level"`
	LogType  int `koanf:"log-type"`
	Metrics  int `koanf:"metrics"`
	PProf    int `koanf:"pprof"`
	Node     int `koanf:"node"`
	Queue    int `koanf:"queue"`
}

type BatchPosterConfig struct {
	Enable  bool `koanf:"enable"`
	MaxSize int  `koanf:"max-size" reload:"hot"`
}

// Flag names don't match field names from default config.
// Contains 2 errors.
func BatchPosterConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enabled", DefaultBatchPosterConfig.Enable, "enable posting batches to l1")
	f.Int("max-sz", DefaultBatchPosterConfig.MaxSize, "maximum batch size")
}

func ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBatchPosterConfig.Enable, "enable posting batches to l1")
	f.Int("max-size", DefaultBatchPosterConfig.MaxSize, "maximum batch size")
}

var DefaultBatchPosterConfig = BatchPosterConfig{
	Enable:  false,
	MaxSize: 100000,
}
