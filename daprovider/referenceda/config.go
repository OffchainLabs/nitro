// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package referenceda

import (
	flag "github.com/spf13/pflag"
)

type Config struct {
	Enable bool `koanf:"enable"`
}

var DefaultConfig = Config{
	Enable: false,
}

func ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, "enable CustomDA mode")
}
