package broadcastclient

import (
	"time"

	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

type FeedConfig struct {
	Output wsbroadcastserver.BroadcasterConfig `koanf:"output"`
	Input  BroadcastClientConfig               `koanf:"input"`
}

func FeedConfigAddOptions(prefix string, f *flag.FlagSet) {
	wsbroadcastserver.BroadcasterConfigAddOptions(prefix+".output", f)
	BroadcastClientConfigAddOptions(prefix+".input", f)
}

var DefaultFeedConfig = FeedConfig{
	Output: wsbroadcastserver.DefaultBroadcasterConfig,
	Input:  DefaultBroadcastClientConfig,
}

type BroadcastClientConfig struct {
	URLs    []string      `koanf:"url"`
	Timeout time.Duration `koanf:"timeout"`
}

func BroadcastClientConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.StringSlice(prefix+".url", DefaultBroadcastClientConfig.URLs, "URL of sequencer feed source")
	f.Duration(prefix+".timeout", DefaultBroadcastClientConfig.Timeout, "duration to wait before timing out connection to sequencer feed")
}

var DefaultBroadcastClientConfig = BroadcastClientConfig{
	URLs:    []string{""},
	Timeout: 20 * time.Second,
}
