package gethexec

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/gethhook"
	"github.com/offchainlabs/nitro/statetransfer"
)

type CachingConfig struct {
	Archive                             bool          `koanf:"archive"`
	BlockCount                          uint64        `koanf:"block-count"`
	BlockAge                            time.Duration `koanf:"block-age"`
	TrieTimeLimitBeforeFlushMaintenance time.Duration `koanf:"trie-time-limit-before-flush-maintenance"`
	TrieTimeLimit                       time.Duration `koanf:"trie-time-limit"`
	TrieTimeLimitRandomOffset           time.Duration `koanf:"trie-time-limit-random-offset"`
	TrieDirtyCache                      int           `koanf:"trie-dirty-cache"`
	TrieCleanCache                      int           `koanf:"trie-clean-cache"`
	TrieCapLimit                        uint32        `koanf:"trie-cap-limit"`
	SnapshotCache                       int           `koanf:"snapshot-cache"`
	DatabaseCache                       int           `koanf:"database-cache"`
	SnapshotRestoreGasLimit             uint64        `koanf:"snapshot-restore-gas-limit"`
	HeadRewindBlocksLimit               uint64        `koanf:"head-rewind-blocks-limit"`
	MaxNumberOfBlocksToSkipStateSaving  uint32        `koanf:"max-number-of-blocks-to-skip-state-saving"`
	MaxAmountOfGasToSkipStateSaving     uint64        `koanf:"max-amount-of-gas-to-skip-state-saving"`
	StylusLRUCacheCapacity              uint32        `koanf:"stylus-lru-cache-capacity"`
	DisableStylusCacheMetricsCollection bool          `koanf:"disable-stylus-cache-metrics-collection"`
	StateScheme                         string        `koanf:"state-scheme"`
	StateHistory                        uint64        `koanf:"state-history"`
	EnablePreimages                     bool          `koanf:"enable-preimages"`
	PathdbMaxDiffLayers                 int           `koanf:"pathdb-max-diff-layers"`
}

func CachingConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".archive", DefaultCachingConfig.Archive, "retain past block state")
	f.Uint64(prefix+".block-count", DefaultCachingConfig.BlockCount, "minimum number of recent blocks to keep in memory")
	f.Duration(prefix+".block-age", DefaultCachingConfig.BlockAge, "minimum age of recent blocks to keep in memory")
	f.Duration(prefix+".trie-time-limit-before-flush-maintenance", DefaultCachingConfig.TrieTimeLimitBeforeFlushMaintenance, "Execution will suggest that maintenance is run if the block processing time required to reach trie-time-limit is smaller or equal than trie-time-limit-before-flush-maintenance")
	f.Duration(prefix+".trie-time-limit", DefaultCachingConfig.TrieTimeLimit, "maximum block processing time before trie is written to hard-disk")
	f.Duration(prefix+".trie-time-limit-random-offset", DefaultCachingConfig.TrieTimeLimitRandomOffset, "if greater then 0, the block processing time period of each trie write to hard-disk is shortened by a random value from range [0, trie-time-limit-random-offset)")
	f.Int(prefix+".trie-dirty-cache", DefaultCachingConfig.TrieDirtyCache, "amount of memory in megabytes to cache state diffs against disk with (larger cache lowers database growth)")
	f.Int(prefix+".trie-clean-cache", DefaultCachingConfig.TrieCleanCache, "amount of memory in megabytes to cache unchanged state trie nodes with")
	f.Int(prefix+".snapshot-cache", DefaultCachingConfig.SnapshotCache, "amount of memory in megabytes to cache state snapshots with")
	f.Int(prefix+".database-cache", DefaultCachingConfig.DatabaseCache, "amount of memory in megabytes to cache database contents with")
	f.Uint32(prefix+".trie-cap-limit", DefaultCachingConfig.TrieCapLimit, "amount of memory in megabytes to be used in the TrieDB Cap operation during maintenance")
	f.Uint64(prefix+".snapshot-restore-gas-limit", DefaultCachingConfig.SnapshotRestoreGasLimit, "maximum gas rolled back to recover snapshot")
	f.Uint64(prefix+".head-rewind-blocks-limit", DefaultCachingConfig.HeadRewindBlocksLimit, "maximum number of blocks rolled back to recover chain head (0 = use geth default limit)")
	f.Uint32(prefix+".max-number-of-blocks-to-skip-state-saving", DefaultCachingConfig.MaxNumberOfBlocksToSkipStateSaving, "maximum number of blocks to skip state saving to persistent storage (archive node only) -- warning: this option seems to cause issues")
	f.Uint64(prefix+".max-amount-of-gas-to-skip-state-saving", DefaultCachingConfig.MaxAmountOfGasToSkipStateSaving, "maximum amount of gas in blocks to skip saving state to Persistent storage (archive node only) -- warning: this option seems to cause issues")
	f.Uint32(prefix+".stylus-lru-cache-capacity", DefaultCachingConfig.StylusLRUCacheCapacity, "capacity, in megabytes, of the LRU cache that keeps initialized stylus programs")
	f.Bool(prefix+".disable-stylus-cache-metrics-collection", DefaultCachingConfig.DisableStylusCacheMetricsCollection, "disable metrics collection for the stylus cache")
	f.String(prefix+".state-scheme", DefaultCachingConfig.StateScheme, "scheme to use for state trie storage (hash, path)")
	f.Uint64(prefix+".state-history", DefaultCachingConfig.StateHistory, "number of recent blocks to retain state history for (path state-scheme only)")
	f.Bool(prefix+".enable-preimages", DefaultCachingConfig.EnablePreimages, "enable recording of preimages")
	f.Int(prefix+".pathdb-max-diff-layers", DefaultCachingConfig.PathdbMaxDiffLayers, "maximum number of diff layers to keep in pathdb (path state-scheme only)")
}

func getStateHistory(maxBlockSpeed time.Duration) uint64 {
	// #nosec G115
	return uint64(24 * time.Hour / maxBlockSpeed)
}

var DefaultCachingConfig = CachingConfig{
	Archive:                             false,
	BlockCount:                          128,
	BlockAge:                            30 * time.Minute,
	TrieTimeLimitBeforeFlushMaintenance: 0,
	TrieTimeLimit:                       time.Hour,
	TrieTimeLimitRandomOffset:           0,
	TrieDirtyCache:                      1024,
	TrieCleanCache:                      600,
	TrieCapLimit:                        100,
	SnapshotCache:                       400,
	DatabaseCache:                       2048,
	SnapshotRestoreGasLimit:             300_000_000_000,
	HeadRewindBlocksLimit:               4 * 7 * 24 * 3600, // 4 blocks per second over 7 days (an arbitrary value, should be greater than the number of blocks between state commits in full node; the state commit period depends both on chain activity and TrieTimeLimit)
	MaxNumberOfBlocksToSkipStateSaving:  0,
	MaxAmountOfGasToSkipStateSaving:     0,
	StylusLRUCacheCapacity:              256,
	StateScheme:                         rawdb.HashScheme,
	StateHistory:                        getStateHistory(DefaultSequencerConfig.MaxBlockSpeed),
	EnablePreimages:                     false,
	PathdbMaxDiffLayers:                 128,
}

func DefaultCacheConfigFor(cachingConfig *CachingConfig) *core.BlockChainConfig {
	return DefaultCacheConfigTrieNoFlushFor(cachingConfig, false)
}

func DefaultCacheConfigTrieNoFlushFor(cachingConfig *CachingConfig, trieNoAsyncFlush bool) *core.BlockChainConfig {
	baseConf := ethconfig.Defaults
	if cachingConfig.Archive {
		baseConf = ethconfig.ArchiveDefaults
	}

	return &core.BlockChainConfig{
		TrieCleanLimit:                     cachingConfig.TrieCleanCache,
		NoPrefetch:                         baseConf.NoPrefetch,
		TrieDirtyLimit:                     cachingConfig.TrieDirtyCache,
		ArchiveMode:                        cachingConfig.Archive,
		TrieTimeLimit:                      cachingConfig.TrieTimeLimit,
		TrieTimeLimitRandomOffset:          cachingConfig.TrieTimeLimitRandomOffset,
		TriesInMemory:                      cachingConfig.BlockCount,
		TrieRetention:                      cachingConfig.BlockAge,
		SnapshotLimit:                      cachingConfig.SnapshotCache,
		Preimages:                          baseConf.Preimages || cachingConfig.EnablePreimages,
		SnapshotRestoreMaxGas:              cachingConfig.SnapshotRestoreGasLimit,
		HeadRewindBlocksLimit:              cachingConfig.HeadRewindBlocksLimit,
		MaxNumberOfBlocksToSkipStateSaving: cachingConfig.MaxNumberOfBlocksToSkipStateSaving,
		MaxAmountOfGasToSkipStateSaving:    cachingConfig.MaxAmountOfGasToSkipStateSaving,
		StateScheme:                        cachingConfig.StateScheme,
		StateHistory:                       cachingConfig.StateHistory,
		MaxDiffLayers:                      cachingConfig.PathdbMaxDiffLayers,
		TrieNoAsyncFlush:                   trieNoAsyncFlush,
	}
}

func (c *CachingConfig) validateStateScheme() error {
	switch c.StateScheme {
	case rawdb.HashScheme:
	case rawdb.PathScheme:
		if c.Archive && c.StateHistory != 0 {
			log.Warn("Path scheme archive mode enabled, but state-history is not zero - the persisted state history will be limited to recent blocks", "StateHistory", c.StateHistory)
		}
	default:
		return errors.New("Invalid StateScheme")
	}
	return nil
}

func (c *CachingConfig) Validate() error {
	return c.validateStateScheme()
}

func WriteOrTestGenblock(chainDb ethdb.Database, cacheConfig *core.BlockChainConfig, initData statetransfer.InitDataReader, chainConfig *params.ChainConfig, genesisArbOSInit *params.ArbOSInit, initMessage *arbostypes.ParsedInitMessage, accountsPerSync uint) error {
	EmptyHash := common.Hash{}
	prevHash := EmptyHash
	blockNumber, err := initData.GetNextBlockNumber()
	if err != nil {
		return err
	}
	storedGenHash := rawdb.ReadCanonicalHash(chainDb, blockNumber)
	// #nosec G115
	timestamp := uint64(0)
	if blockNumber > 0 {
		prevHash = rawdb.ReadCanonicalHash(chainDb, blockNumber-1)
		if prevHash == EmptyHash {
			return fmt.Errorf("block number %d not found in database", blockNumber-1)
		}
		prevHeader := rawdb.ReadHeader(chainDb, prevHash, blockNumber-1)
		if prevHeader == nil {
			return fmt.Errorf("block header for block %d not found in database", blockNumber-1)
		}
		timestamp = prevHeader.Time
	}
	stateRoot, err := arbosState.InitializeArbosInDatabase(chainDb, cacheConfig, initData, chainConfig, genesisArbOSInit, initMessage, timestamp, accountsPerSync)
	if err != nil {
		return err
	}

	genBlock := arbosState.MakeGenesisBlock(prevHash, blockNumber, timestamp, stateRoot, chainConfig)
	blockHash := genBlock.Hash()

	if storedGenHash == EmptyHash {
		// chainDb did not have genesis block. Initialize it.
		batch := chainDb.NewBatch()
		core.WriteHeadBlock(batch, genBlock)
		err = batch.Write()
		if err != nil {
			return err
		}
		log.Info("wrote genesis block", "number", blockNumber, "hash", blockHash)
	} else if storedGenHash != blockHash {
		return fmt.Errorf("database contains data inconsistent with initialization: database has genesis hash %v but we built genesis hash %v", storedGenHash, blockHash)
	} else {
		log.Info("recreated existing genesis block", "number", blockNumber, "hash", blockHash)
	}

	return nil
}

func TryReadStoredChainConfig(chainDb ethdb.Database) *params.ChainConfig {
	EmptyHash := common.Hash{}

	block0Hash := rawdb.ReadCanonicalHash(chainDb, 0)
	if block0Hash == EmptyHash {
		return nil
	}
	return rawdb.ReadChainConfig(chainDb, block0Hash)
}

func WriteOrTestChainConfig(chainDb ethdb.Database, config *params.ChainConfig) error {
	EmptyHash := common.Hash{}

	block0Hash := rawdb.ReadCanonicalHash(chainDb, 0)
	if block0Hash == EmptyHash {
		return errors.New("block 0 not found")
	}
	storedConfig := rawdb.ReadChainConfig(chainDb, block0Hash)
	if storedConfig == nil {
		rawdb.WriteChainConfig(chainDb, block0Hash, config)
		return nil
	}
	height, found := rawdb.ReadHeaderNumber(chainDb, rawdb.ReadHeadHeaderHash(chainDb))
	if !found {
		return errors.New("non empty chain config but empty chain")
	}
	err := storedConfig.CheckCompatible(config, height, 0)
	if err != nil {
		return err
	}
	rawdb.WriteChainConfig(chainDb, block0Hash, config)
	return nil
}

func GetBlockChain(
	chainDb ethdb.Database,
	cacheConfig *core.BlockChainConfig,
	chainConfig *params.ChainConfig,
	tracer *tracing.Hooks,
	txIndexerConfig *TxIndexerConfig,
) (*core.BlockChain, error) {
	engine := arbos.Engine{
		IsSequencer: true,
	}

	vmConfig := vm.Config{
		EnablePreimageRecording: false,
		Tracer:                  tracer,
	}
	cacheConfig.VmConfig = vmConfig

	var coreTxIndexerConfig *core.TxIndexerConfig // nil if disabled
	if txIndexerConfig.Enable {
		coreTxIndexerConfig = &core.TxIndexerConfig{
			Limit:         txIndexerConfig.TxLookupLimit,
			Threads:       txIndexerConfig.Threads,
			MinBatchDelay: txIndexerConfig.MinBatchDelay,
		}
	}
	cacheConfig.TxIndexer = coreTxIndexerConfig
	return core.NewBlockChain(chainDb, chainConfig, nil, engine, cacheConfig)
}

func WriteOrTestBlockChain(
	chainDb ethdb.Database,
	cacheConfig *core.BlockChainConfig,
	initData statetransfer.InitDataReader,
	chainConfig *params.ChainConfig,
	genesisArbOSInit *params.ArbOSInit,
	tracer *tracing.Hooks,
	initMessage *arbostypes.ParsedInitMessage,
	txIndexerConfig *TxIndexerConfig,
	accountsPerSync uint,
) (*core.BlockChain, error) {
	emptyBlockChain := rawdb.ReadHeadHeader(chainDb) == nil
	if !emptyBlockChain && (cacheConfig.StateScheme == rawdb.PathScheme) {
		// When using path scheme, and the stored state trie is not empty,
		// WriteOrTestGenBlock is not able to recover EmptyRootHash state trie node.
		// In that case Nitro doesn't test genblock, but just returns the BlockChain.
		return GetBlockChain(chainDb, cacheConfig, chainConfig, tracer, txIndexerConfig)
	}

	err := WriteOrTestGenblock(chainDb, cacheConfig, initData, chainConfig, genesisArbOSInit, initMessage, accountsPerSync)
	if err != nil {
		return nil, err
	}
	err = WriteOrTestChainConfig(chainDb, chainConfig)
	if err != nil {
		return nil, err
	}
	return GetBlockChain(chainDb, cacheConfig, chainConfig, tracer, txIndexerConfig)
}

func init() {
	gethhook.RequireHookedGeth()
}
