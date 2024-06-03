package conf

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/spf13/pflag"
)

type InitConfig struct {
	Force                    bool          `koanf:"force"`
	Url                      string        `koanf:"url"`
	Latest                   bool          `koanf:"latest"`
	LatestKind               string        `koanf:"latest-kind"`
	LatestMirror             string        `koanf:"latest-mirror"`
	DownloadPath             string        `koanf:"download-path"`
	DownloadPoll             time.Duration `koanf:"download-poll"`
	DevInit                  bool          `koanf:"dev-init"`
	DevInitAddress           string        `koanf:"dev-init-address"`
	DevInitBlockNum          uint64        `koanf:"dev-init-blocknum"`
	Empty                    bool          `koanf:"empty"`
	AccountsPerSync          uint          `koanf:"accounts-per-sync"`
	ImportFile               string        `koanf:"import-file"`
	ThenQuit                 bool          `koanf:"then-quit"`
	Prune                    string        `koanf:"prune"`
	PruneBloomSize           uint64        `koanf:"prune-bloom-size"`
	ResetToMessage           int64         `koanf:"reset-to-message"`
	RecreateMissingStateFrom uint64        `koanf:"recreate-missing-state-from"`
}

var InitConfigDefault = InitConfig{
	Force:                    false,
	Url:                      "",
	Latest:                   false,
	LatestKind:               acceptedSnapshotKinds[0],
	LatestMirror:             "https://snapshot.arbitrum.foundation/",
	DownloadPath:             "/tmp/",
	DownloadPoll:             time.Minute,
	DevInit:                  false,
	DevInitAddress:           "",
	DevInitBlockNum:          0,
	Empty:                    false,
	ImportFile:               "",
	AccountsPerSync:          100000,
	ThenQuit:                 false,
	Prune:                    "",
	PruneBloomSize:           2048,
	ResetToMessage:           -1,
	RecreateMissingStateFrom: 0, // 0 = disabled
}

func InitConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".force", InitConfigDefault.Force, "if true: in case database exists init code will be reexecuted and genesis block compared to database")
	f.String(prefix+".url", InitConfigDefault.Url, "url to download initialization data - will poll if download fails")
	f.Bool(prefix+".latest", InitConfigDefault.Latest, "if true: search for the latest snapshot")
	f.String(prefix+".latest-kind", InitConfigDefault.LatestKind, "snapshot kind when searching for the latest "+acceptedSnapshotKindsStr)
	f.String(prefix+".latest-mirror", InitConfigDefault.LatestMirror, "base url used when searching for the latest")
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
	f.Uint64(prefix+".recreate-missing-state-from", InitConfigDefault.RecreateMissingStateFrom, "block number to start recreating missing states from (0 = disabled)")
}

func (c *InitConfig) Validate() error {
	if c.Force && c.RecreateMissingStateFrom > 0 {
		log.Warn("force init enabled, recreate-missing-state-from will have no effect")
	}
	if !isAcceptedSnapshotKind(c.LatestKind) {
		return fmt.Errorf("invalid value for latest-kind option: \"%s\" %s", c.LatestKind, acceptedSnapshotKindsStr)
	}
	return nil
}

var (
	acceptedSnapshotKinds    = []string{"archive", "pruned"}
	acceptedSnapshotKindsStr = "(accepted values: \"" + strings.Join(acceptedSnapshotKinds, "\" | \"") + "\")"
)

func isAcceptedSnapshotKind(kind string) bool {
	for _, valid := range acceptedSnapshotKinds {
		if kind == valid {
			return true
		}
	}
	return false
}
