package conf

import (
	"time"

	"github.com/spf13/pflag"
)

type InitConfig struct {
	Force           bool          `koanf:"force"`
	Url             string        `koanf:"url"`
	DownloadPath    string        `koanf:"download-path"`
	DownloadPoll    time.Duration `koanf:"download-poll"`
	DevInit         bool          `koanf:"dev-init"`
	DevInitAddress  string        `koanf:"dev-init-address"`
	DevInitBlockNum uint64        `koanf:"dev-init-blocknum"`
	Empty           bool          `koanf:"empty"`
	AccountsPerSync uint          `koanf:"accounts-per-sync"`
	ImportFile      string        `koanf:"import-file"`
	ThenQuit        bool          `koanf:"then-quit"`
	Prune           string        `koanf:"prune"`
	PruneBloomSize  uint64        `koanf:"prune-bloom-size"`
	ResetToMessage  int64         `koanf:"reset-to-message"`
}

var InitConfigDefault = InitConfig{
	Force:           false,
	Url:             "",
	DownloadPath:    "/tmp/",
	DownloadPoll:    time.Minute,
	DevInit:         false,
	DevInitAddress:  "",
	DevInitBlockNum: 0,
	Empty:           false,
	ImportFile:      "",
	AccountsPerSync: 100000,
	ThenQuit:        false,
	Prune:           "",
	PruneBloomSize:  2048,
	ResetToMessage:  -1,
}

func InitConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".force", InitConfigDefault.Force, "if true: in case database exists init code will be reexecuted and genesis block compared to database")
	f.String(prefix+".url", InitConfigDefault.Url, "url to download initializtion data - will poll if download fails")
	f.String(prefix+".download-path", InitConfigDefault.DownloadPath, "path to save temp downloaded file")
	f.Duration(prefix+".download-poll", InitConfigDefault.DownloadPoll, "how long to wait between polling attempts")
	f.Bool(prefix+".dev-init", InitConfigDefault.DevInit, "init with dev data (1 account with balance) instead of file import")
	f.String(prefix+".dev-init-address", InitConfigDefault.DevInitAddress, "Address of dev-account. Leave empty to use the dev-wallet.")
	f.Uint64(prefix+".dev-init-blocknum", InitConfigDefault.DevInitBlockNum, "Number of preinit blocks. Must exist in ancient database.")
	f.Bool(prefix+".empty", InitConfigDefault.Empty, "init with empty state")
	f.Bool(prefix+".then-quit", InitConfigDefault.ThenQuit, "quit after init is done")
	f.String(prefix+".import-file", InitConfigDefault.ImportFile, "path for json data to import")
	f.Uint(prefix+".accounts-per-sync", InitConfigDefault.AccountsPerSync, "during init - sync database every X accounts. Lower value for low-memory systems. 0 disables.")
	f.String(prefix+".prune", InitConfigDefault.Prune, "pruning for a given use: \"full\" for full nodes serving RPC requests, or \"validator\" for validators")
	f.Uint64(prefix+".prune-bloom-size", InitConfigDefault.PruneBloomSize, "the amount of memory in megabytes to use for the pruning bloom filter (higher values prune better)")
	f.Int64(prefix+".reset-to-message", InitConfigDefault.ResetToMessage, "forces a reset to an old message height. Also set max-reorg-resequence-depth=0 to force re-reading messages")
}
