package client

import (
	"time"

	flag "github.com/spf13/pflag"
)

type ConfigFetcher func() *Config

type Config struct {
	Protocol string        `koanf:"protocol" reload:"hot"`
	Host     string        `koanf:"host" reload:"hot"`
	Port     string        `koanf:"port" reload:"hot"`
	Timeout  time.Duration `koanf:"timeout" reload:"hot"`
}

func AddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".protocol", DefaultConfig.Protocol, "the protocol used, either HTTP or HTTPS")
	f.String(prefix+".host", DefaultConfig.Host, "host to bind the feed's HTTP input to")
	f.String(prefix+".port", DefaultConfig.Port, "port to bind the feed's HTTP input to")
	f.Duration(prefix+".timeout", DefaultConfig.Timeout, "time limit for requests made by the client")
}

var (
	DefaultConfig = Config{
		Protocol: "http",
		Host:     "",
		Port:     "9463",
		Timeout:  5 * time.Second,
	}
	DefaultTestConfig = Config{
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     "", // this should be filled in by the test once the HTTP server has started and chosen it's port
		Timeout:  5 * time.Second,
	}
)
