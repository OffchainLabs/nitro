package gethexec

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/gethhook"
	"github.com/offchainlabs/nitro/statetransfer"
)

type CachingConfig struct {
	Archive                            bool          `koanf:"archive"`
	BlockCount                         uint64        `koanf:"block-count"`
	BlockAge                           time.Duration `koanf:"block-age"`
	TrieTimeLimit                      time.Duration `koanf:"trie-time-limit"`
	TrieDirtyCache                     int           `koanf:"trie-dirty-cache"`
	TrieCleanCache                     int           `koanf:"trie-clean-cache"`
	SnapshotCache                      int           `koanf:"snapshot-cache"`
	DatabaseCache                      int           `koanf:"database-cache"`
	SnapshotRestoreGasLimit            uint64        `koanf:"snapshot-restore-gas-limit"`
	MaxNumberOfBlocksToSkipStateSaving uint32        `koanf:"max-number-of-blocks-to-skip-state-saving"`
	MaxAmountOfGasToSkipStateSaving    uint64        `koanf:"max-amount-of-gas-to-skip-state-saving"`
	StylusLRUCache                     uint32        `koanf:"stylus-lru-cache"`
	StateScheme                        string        `koanf:"state-scheme"`
	StateHistory                       uint64        `koanf:"state-history"`
}

func CachingConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".archive", DefaultCachingConfig.Archive, "retain past block state")
	f.Uint64(prefix+".block-count", DefaultCachingConfig.BlockCount, "minimum number of recent blocks to keep in memory")
	f.Duration(prefix+".block-age", DefaultCachingConfig.BlockAge, "minimum age of recent blocks to keep in memory")
	f.Duration(prefix+".trie-time-limit", DefaultCachingConfig.TrieTimeLimit, "maximum block processing time before trie is written to hard-disk")
	f.Int(prefix+".trie-dirty-cache", DefaultCachingConfig.TrieDirtyCache, "amount of memory in megabytes to cache state diffs against disk with (larger cache lowers database growth)")
	f.Int(prefix+".trie-clean-cache", DefaultCachingConfig.TrieCleanCache, "amount of memory in megabytes to cache unchanged state trie nodes with")
	f.Int(prefix+".snapshot-cache", DefaultCachingConfig.SnapshotCache, "amount of memory in megabytes to cache state snapshots with")
	f.Int(prefix+".database-cache", DefaultCachingConfig.DatabaseCache, "amount of memory in megabytes to cache database contents with")
	f.Uint64(prefix+".snapshot-restore-gas-limit", DefaultCachingConfig.SnapshotRestoreGasLimit, "maximum gas rolled back to recover snapshot")
	f.Uint32(prefix+".max-number-of-blocks-to-skip-state-saving", DefaultCachingConfig.MaxNumberOfBlocksToSkipStateSaving, "maximum number of blocks to skip state saving to persistent storage (archive node only) -- warning: this option seems to cause issues")
	f.Uint64(prefix+".max-amount-of-gas-to-skip-state-saving", DefaultCachingConfig.MaxAmountOfGasToSkipStateSaving, "maximum amount of gas in blocks to skip saving state to Persistent storage (archive node only) -- warning: this option seems to cause issues")
	f.Uint32(prefix+".stylus-lru-cache", DefaultCachingConfig.StylusLRUCache, "initialized stylus programs to keep in LRU cache")
	f.String(prefix+".state-scheme", DefaultCachingConfig.StateScheme, "scheme to use for state trie storage (hash, path)")
	f.Uint64(prefix+".state-history", DefaultCachingConfig.StateHistory, "number of recent blocks to retain state history for (path state-scheme only)")
}

func getStateHistory(maxBlockSpeed time.Duration) uint64 {
	return uint64(24 * time.Hour / maxBlockSpeed)
}

var DefaultCachingConfig = CachingConfig{
	Archive:                            false,
	BlockCount:                         128,
	BlockAge:                           30 * time.Minute,
	TrieTimeLimit:                      time.Hour,
	TrieDirtyCache:                     1024,
	TrieCleanCache:                     600,
	SnapshotCache:                      400,
	DatabaseCache:                      2048,
	SnapshotRestoreGasLimit:            300_000_000_000,
	MaxNumberOfBlocksToSkipStateSaving: 0,
	MaxAmountOfGasToSkipStateSaving:    0,
	StylusLRUCache:                     256,
	StateScheme:                        rawdb.HashScheme,
	StateHistory:                       getStateHistory(DefaultSequencerConfig.MaxBlockSpeed),
}

// TODO remove stack from parameters as it is no longer needed here
func DefaultCacheConfigFor(stack *node.Node, cachingConfig *CachingConfig) *core.CacheConfig {
	baseConf := ethconfig.Defaults
	if cachingConfig.Archive {
		baseConf = ethconfig.ArchiveDefaults
	}

	return &core.CacheConfig{
		TrieCleanLimit:                     cachingConfig.TrieCleanCache,
		TrieCleanNoPrefetch:                baseConf.NoPrefetch,
		TrieDirtyLimit:                     cachingConfig.TrieDirtyCache,
		TrieDirtyDisabled:                  cachingConfig.Archive,
		TrieTimeLimit:                      cachingConfig.TrieTimeLimit,
		TriesInMemory:                      cachingConfig.BlockCount,
		TrieRetention:                      cachingConfig.BlockAge,
		SnapshotLimit:                      cachingConfig.SnapshotCache,
		Preimages:                          baseConf.Preimages,
		SnapshotRestoreMaxGas:              cachingConfig.SnapshotRestoreGasLimit,
		MaxNumberOfBlocksToSkipStateSaving: cachingConfig.MaxNumberOfBlocksToSkipStateSaving,
		MaxAmountOfGasToSkipStateSaving:    cachingConfig.MaxAmountOfGasToSkipStateSaving,
		StateScheme:                        cachingConfig.StateScheme,
		StateHistory:                       cachingConfig.StateHistory,
	}
}

func (c *CachingConfig) validateStateScheme() error {
	switch c.StateScheme {
	case rawdb.HashScheme:
	case rawdb.PathScheme:
		if c.Archive {
			return errors.New("archive cannot be set when using path as the state-scheme")
		}
	default:
		return errors.New("Invalid StateScheme")
	}
	return nil
}

func (c *CachingConfig) Validate() error {
	return c.validateStateScheme()
}

func WriteOrTestGenblock(chainDb ethdb.Database, cacheConfig *core.CacheConfig, initData statetransfer.InitDataReader, chainConfig *params.ChainConfig, initMessage *arbostypes.ParsedInitMessage, accountsPerSync uint) error {
	EmptyHash := common.Hash{}
	prevHash := EmptyHash
	prevDifficulty := big.NewInt(0)
	blockNumber, err := initData.GetNextBlockNumber()
	if err != nil {
		return err
	}
	storedGenHash := rawdb.ReadCanonicalHash(chainDb, blockNumber)
	timestamp := uint64(0)
	if blockNumber > 0 {
		prevHash = rawdb.ReadCanonicalHash(chainDb, blockNumber-1)
		if prevHash == EmptyHash {
			return fmt.Errorf("block number %d not found in database", chainDb)
		}
		prevHeader := rawdb.ReadHeader(chainDb, prevHash, blockNumber-1)
		if prevHeader == nil {
			return fmt.Errorf("block header for block %d not found in database", chainDb)
		}
		timestamp = prevHeader.Time
	}
	stateRoot, err := arbosState.InitializeArbosInDatabase(chainDb, cacheConfig, initData, chainConfig, initMessage, timestamp, accountsPerSync)
	if err != nil {
		return err
	}

	genBlock := arbosState.MakeGenesisBlock(prevHash, blockNumber, timestamp, stateRoot, chainConfig)
	blockHash := genBlock.Hash()

	if storedGenHash == EmptyHash {
		// chainDb did not have genesis block. Initialize it.
		core.WriteHeadBlock(chainDb, genBlock, prevDifficulty)
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
	height := rawdb.ReadHeaderNumber(chainDb, rawdb.ReadHeadHeaderHash(chainDb))
	if height == nil {
		return errors.New("non empty chain config but empty chain")
	}
	err := storedConfig.CheckCompatible(config, *height, 0)
	if err != nil {
		return err
	}
	rawdb.WriteChainConfig(chainDb, block0Hash, config)
	return nil
}

func GetBlockChain(chainDb ethdb.Database, cacheConfig *core.CacheConfig, chainConfig *params.ChainConfig, txLookupLimit uint64) (*core.BlockChain, error) {
	engine := arbos.Engine{
		IsSequencer: true,
	}

	vmConfig := vm.Config{
		EnablePreimageRecording: false,
	}

	return core.NewBlockChain(chainDb, cacheConfig, chainConfig, nil, nil, engine, vmConfig, shouldPreserveFalse, &txLookupLimit)
}

func WriteOrTestBlockChain(chainDb ethdb.Database, cacheConfig *core.CacheConfig, initData statetransfer.InitDataReader, chainConfig *params.ChainConfig, initMessage *arbostypes.ParsedInitMessage, txLookupLimit uint64, accountsPerSync uint) (*core.BlockChain, error) {
	emptyBlockChain := rawdb.ReadHeadHeader(chainDb) == nil
	if !emptyBlockChain && (cacheConfig.StateScheme == rawdb.PathScheme) {
		// When using path scheme, and the stored state trie is not empty,
		// WriteOrTestGenBlock is not able to recover EmptyRootHash state trie node.
		// In that case Nitro doesn't test genblock, but just returns the BlockChain.
		return GetBlockChain(chainDb, cacheConfig, chainConfig, txLookupLimit)
	}

	err := WriteOrTestGenblock(chainDb, cacheConfig, initData, chainConfig, initMessage, accountsPerSync)
	if err != nil {
		return nil, err
	}
	err = WriteOrTestChainConfig(chainDb, chainConfig)
	if err != nil {
		return nil, err
	}
	return GetBlockChain(chainDb, cacheConfig, chainConfig, txLookupLimit)
}

// Don't preserve reorg'd out blocks
func shouldPreserveFalse(_ *types.Header) bool {
	return false
}

func init() {
	gethhook.RequireHookedGeth()
}
