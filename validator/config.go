package validator

import (
	flag "github.com/spf13/pflag"
)

type BlockValidatorConfig struct {
	OutputPath          string
	ConcurrentRunsLimit int // 0 - default (CPU#)
	BlocksToRecord      []uint64
}

func BlockValidatorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+"output-path", DefaultBlockValidatorConfig.OutputPath, "")
	f.Int(prefix+"concurrent-runs-limit", DefaultBlockValidatorConfig.ConcurrentRunsLimit, "")
	// Don't get BlocksToRecord from command line options
}

var DefaultBlockValidatorConfig = BlockValidatorConfig{
	OutputPath:          "./target/output",
	ConcurrentRunsLimit: 0,
	BlocksToRecord:      []uint64{},
}
