package conf

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/spf13/pflag"
)

type InitConfig struct {
	Force                    bool          `koanf:"force"`
	Url                      string        `koanf:"url"`
	Latest                   string        `koanf:"latest"`
	LatestBase               string        `koanf:"latest-base"`
	ValidateChecksum         bool          `koanf:"validate-checksum"`
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
	PruneThreads             int           `koanf:"prune-threads"`
	PruneTrieCleanCache      int           `koanf:"prune-trie-clean-cache"`
	RecreateMissingStateFrom uint64        `koanf:"recreate-missing-state-from"`
	RebuildLocalWasm         bool          `koanf:"rebuild-local-wasm"`
	ReorgToBatch             int64         `koanf:"reorg-to-batch"`
	ReorgToMessageBatch      int64         `koanf:"reorg-to-message-batch"`
	ReorgToBlockBatch        int64         `koanf:"reorg-to-block-batch"`
}

var InitConfigDefault = InitConfig{
	Force:                    false,
	Url:                      "",
	Latest:                   "",
	LatestBase:               "https://snapshot.arbitrum.foundation/",
	ValidateChecksum:         true,
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
	PruneThreads:             runtime.NumCPU(),
	PruneTrieCleanCache:      gethexec.DefaultCachingConfig.TrieCleanCache,
	RecreateMissingStateFrom: 0, // 0 = disabled
	RebuildLocalWasm:         true,
	ReorgToBatch:             -1,
	ReorgToMessageBatch:      -1,
	ReorgToBlockBatch:        -1,
}

func InitConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".force", InitConfigDefault.Force, "if true: in case database exists init code will be reexecuted and genesis block compared to database")
	f.String(prefix+".url", InitConfigDefault.Url, "url to download initialization data - will poll if download fails")
	f.String(prefix+".latest", InitConfigDefault.Latest, "if set, searches for the latest snapshot of the given kind "+acceptedSnapshotKindsStr)
	f.String(prefix+".latest-base", InitConfigDefault.LatestBase, "base url used when searching for the latest")
	f.Bool(prefix+".validate-checksum", InitConfigDefault.ValidateChecksum, "if true: validate the checksum after downloading the snapshot")
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
	f.Int(prefix+".prune-threads", InitConfigDefault.PruneThreads, "the number of threads to use when pruning")
	f.Int(prefix+".prune-trie-clean-cache", InitConfigDefault.PruneTrieCleanCache, "amount of memory in megabytes to cache unchanged state trie nodes with when traversing state database during pruning")
	f.Uint64(prefix+".recreate-missing-state-from", InitConfigDefault.RecreateMissingStateFrom, "block number to start recreating missing states from (0 = disabled)")
	f.Bool(prefix+".rebuild-local-wasm", InitConfigDefault.RebuildLocalWasm, "rebuild local wasm database on boot if needed (otherwise-will be done lazily)")
	f.Int64(prefix+".reorg-to-batch", InitConfigDefault.ReorgToBatch, "rolls back the blockchain to a specified batch number")
	f.Int64(prefix+".reorg-to-message-batch", InitConfigDefault.ReorgToMessageBatch, "rolls back the blockchain to the first batch at or before a given message index")
	f.Int64(prefix+".reorg-to-block-batch", InitConfigDefault.ReorgToBlockBatch, "rolls back the blockchain to the first batch at or before a given block number")
}

func (c *InitConfig) Validate() error {
	if c.Force && c.RecreateMissingStateFrom > 0 {
		log.Warn("force init enabled, recreate-missing-state-from will have no effect")
	}
	if c.Latest != "" && !isAcceptedSnapshotKind(c.Latest) {
		return fmt.Errorf("invalid value for latest option: \"%s\" %s", c.Latest, acceptedSnapshotKindsStr)
	}
	if c.Prune != "" && c.PruneThreads <= 0 {
		return fmt.Errorf("invalid number of pruning threads: %d, has to be greater then 0", c.PruneThreads)
	}
	if c.PruneTrieCleanCache < 0 {
		return fmt.Errorf("invalid trie clean cache size: %d, has to be greater or equal 0", c.PruneTrieCleanCache)
	}
	numReorgOptionsSpecified := 0
	for _, reorgOption := range []int64{c.ReorgToBatch, c.ReorgToMessageBatch, c.ReorgToBlockBatch} {
		if reorgOption >= 0 {
			numReorgOptionsSpecified++
			if numReorgOptionsSpecified > 1 {
				return fmt.Errorf("at most one init reorg option can be specified")
			}
		}
	}
	return nil
}

func (c *InitConfig) IsReorgRequested() bool {
	return c.ReorgToBatch >= 0 || c.ReorgToBlockBatch >= 0 || c.ReorgToMessageBatch >= 0
}

var (
	acceptedSnapshotKinds    = []string{"archive", "pruned", "genesis"}
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
