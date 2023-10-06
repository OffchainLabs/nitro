package server

import (
	"time"

	"github.com/offchainlabs/nitro/broadcaster/http/backlog"
	flag "github.com/spf13/pflag"
)

type ConfigFetcher func() *Config

type Config struct {
	Enabled           bool           `koanf:"enabled" reload:"hot"`
	Host              string         `koanf:"host" reload:"hot"`
	Port              string         `koanf:"port" reload:"hot"`
	ReadTimeout       time.Duration  `koanf:"read-timeout" reload:"hot"`
	ReadHeaderTimeout time.Duration  `koanf:"read-header-timeout" reload:"hot"`
	WriteTimeout      time.Duration  `koanf:"write-timeout" reload:"hot"`
	IdleTimeout       time.Duration  `koanf:"idle-timeout" reload:"hot"`
	Backlog           backlog.Config `koanf:"backlog" reload:"hot"`
}

func AddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enabled", DefaultConfig.Enabled, "enable Broadcaster's HTTP server")
	f.String(prefix+".host", DefaultConfig.Host, "host to bind the feed's HTTP output to")
	f.String(prefix+".port", DefaultConfig.Port, "port to bind the feed's HTTP output to")
	f.Duration(prefix+".read-timeout", DefaultConfig.ReadTimeout, "the maximum duration for reading the entire request, including the body, from clients")
	f.Duration(prefix+".read-header-timeout", DefaultConfig.ReadHeaderTimeout, "the maximum duration for reading the request headers from clients. If ReadHeaderTimeout is zero, the value of ReadTimeout is used.")
	f.Duration(prefix+".write-timeout", DefaultConfig.WriteTimeout, "the maximum duration for writing the response to clients")
	f.Duration(prefix+".idle-timeout", DefaultConfig.IdleTimeout, "the maximum amount of time to wait for the next request when keep-alives are enabled. If IdleTimeout is zero, the value of ReadTimeout is used. If both are zero, there is no timeout.")
	backlog.AddOptions(prefix+".backlog", f)
}

var (
	DefaultConfig = Config{
		Enabled:           false,
		Host:              "",
		Port:              "9643",
		ReadTimeout:       time.Second,
		ReadHeaderTimeout: 0 * time.Second,
		WriteTimeout:      time.Second,
		IdleTimeout:       0 * time.Second,
		Backlog:           backlog.DefaultConfig,
	}
	DefaultTestConfig = Config{
		Enabled:           false,
		Host:              "127.0.0.1",
		Port:              "0",
		ReadTimeout:       time.Second,
		ReadHeaderTimeout: 0 * time.Second,
		WriteTimeout:      time.Second,
		IdleTimeout:       0 * time.Second,
		Backlog:           backlog.DefaultTestConfig,
	}
)
