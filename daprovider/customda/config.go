package customda

import (
	flag "github.com/spf13/pflag"
)

type Config struct {
	Enable          bool   `koanf:"enable"`
	ValidatorType   string `koanf:"validator-type"`
	StorageType     string `koanf:"storage-type"`
	StorageLocation string `koanf:"storage-location"`
}

var DefaultConfig = Config{
	Enable:          false,
	ValidatorType:   "reference",
	StorageType:     "memory",
	StorageLocation: "",
}

func ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, "enable CustomDA mode")
	f.String(prefix+".validator-type", DefaultConfig.ValidatorType, "CustomDA validator implementation (reference)")
	f.String(prefix+".storage-type", DefaultConfig.StorageType, "CustomDA storage backend (memory)")
	f.String(prefix+".storage-location", DefaultConfig.StorageLocation, "CustomDA storage location (path for file storage)")
}
