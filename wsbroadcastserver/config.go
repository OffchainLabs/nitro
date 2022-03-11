package wsbroadcastserver

import (
	"time"

	flag "github.com/spf13/pflag"
)

type BroadcasterConfig struct {
	Enable        bool          `koanf:"enable"`
	Addr          string        `koanf:"addr"`
	IOTimeout     time.Duration `koanf:"io-timeout"`
	Port          string        `koanf:"port"`
	Ping          time.Duration `koanf:"ping"`
	ClientTimeout time.Duration `koanf:"client-timeout"`
	Queue         int           `koanf:"queue"`
	Workers       int           `koanf:"workers"`
}

func BroadcasterConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBroadcasterConfig.Enable, "enable broadcaster")
	f.String(prefix+".addr", DefaultBroadcasterConfig.Addr, "address to bind the relay feed output to")
	f.Duration(prefix+".io-timeout", DefaultBroadcasterConfig.IOTimeout, "duration to wait before timing out HTTP to WS upgrade")
	f.String(prefix+".port", DefaultBroadcasterConfig.Port, "port to bind the relay feed output to")
	f.Duration(prefix+".ping", DefaultBroadcasterConfig.Ping, "duration for ping interval")
	f.Duration(prefix+".client-timeout", DefaultBroadcasterConfig.ClientTimeout, "duration to wait before timing out connections to client")
	f.Int(prefix+".queue", DefaultBroadcasterConfig.Queue, "queue size")
	f.Int(prefix+".workers", DefaultBroadcasterConfig.Workers, "number of threads to reserve for HTTP to WS upgrade")
}

var DefaultBroadcasterConfig = BroadcasterConfig{
	Enable:        false,
	Addr:          "",
	IOTimeout:     5 * time.Second,
	Port:          "9642",
	Ping:          5 * time.Second,
	ClientTimeout: 15 * time.Second,
	Queue:         100,
	Workers:       100,
}

type ClientConfig struct {
	URL     string        `koanf:"url"`
	Timeout time.Duration `koanf:"timeout"`
}

func ClientConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".url", DefaultClientConfig.URL, "URL of sequencer feed source")
	f.Duration(prefix+".timeout", DefaultClientConfig.Timeout, "duration to wait before timing out connection to server")
}

var DefaultClientConfig = ClientConfig{
	URL:     "",
	Timeout: 20 * time.Second,
}
