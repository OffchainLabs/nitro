package client

import (
	flag "github.com/spf13/pflag"
)

type ConfigFetcher func() *Config

type Config struct {
	Protocol string `koanf:"protocol" reload:"hot"`
	Host     string `koanf:"host" reload:"hot"`
	Port     string `koanf:"port" reload:"hot"`
}

func AddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".protocol", DefaultConfig.Protocol, "the protocol used, either HTTP or HTTPS")
	f.String(prefix+".host", DefaultConfig.Host, "host to bind the feed's HTTP input to")
	f.String(prefix+".port", DefaultConfig.Port, "port to bind the feed's HTTP input to")
}

var (
	DefaultConfig = Config{
		Protocol: "https",
		Host:     "",
		Port:     "9463",
	}
	DefaultTestConfig = Config{
		Protocol: "https",
		Host:     "",
		Port:     "9463",
	}
)
