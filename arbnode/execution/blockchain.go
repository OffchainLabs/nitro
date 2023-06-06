package execution

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
	"github.com/offchainlabs/nitro/gethhook"
	"github.com/offchainlabs/nitro/statetransfer"
)

type CachingConfig struct {
	Archive               bool          `koanf:"archive"`
	BlockCount            uint64        `koanf:"block-count"`
	BlockAge              time.Duration `koanf:"block-age"`
	TrieTimeLimit         time.Duration `koanf:"trie-time-limit"`
	TrieDirtyCache        int           `koanf:"trie-dirty-cache"`
	TrieCleanCache        int           `koanf:"trie-clean-cache"`
	SnapshotCache         int           `koanf:"snapshot-cache"`
	DatabaseCache         int           `koanf:"database-cache"`
	SnapshotRestoreMaxGas uint64        `koanf:"snapshot-restore-gas-limit"`
}

func CachingConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".archive", DefaultCachingConfig.Archive, "retain past block state")
	f.Uint64(prefix+".block-count", DefaultCachingConfig.BlockCount, "minimum number of recent blocks to keep in memory")
	f.Duration(prefix+".block-age", DefaultCachingConfig.BlockAge, "minimum age a block must be to be pruned")
	f.Duration(prefix+".trie-time-limit", DefaultCachingConfig.TrieTimeLimit, "maximum block processing time before trie is written to hard-disk")
	f.Int(prefix+".trie-dirty-cache", DefaultCachingConfig.TrieDirtyCache, "amount of memory in megabytes to cache state diffs against disk with (larger cache lowers database growth)")
	f.Int(prefix+".trie-clean-cache", DefaultCachingConfig.TrieCleanCache, "amount of memory in megabytes to cache unchanged state trie nodes with")
	f.Int(prefix+".snapshot-cache", DefaultCachingConfig.SnapshotCache, "amount of memory in megabytes to cache state snapshots with")
	f.Int(prefix+".database-cache", DefaultCachingConfig.DatabaseCache, "amount of memory in megabytes to cache database contents with")
	f.Uint64(prefix+".snapshot-restore-gas-limit", DefaultCachingConfig.SnapshotRestoreMaxGas, "maximum gas rolled back to recover snapshot")
}

var DefaultCachingConfig = CachingConfig{
	Archive:               false,
	BlockCount:            128,
	BlockAge:              30 * time.Minute,
	TrieTimeLimit:         time.Hour,
	TrieDirtyCache:        1024,
	TrieCleanCache:        600,
	SnapshotCache:         400,
	DatabaseCache:         2048,
	SnapshotRestoreMaxGas: 300_000_000_000,
}

func DefaultCacheConfigFor(stack *node.Node, cachingConfig *CachingConfig) *core.CacheConfig {
	baseConf := ethconfig.Defaults
	if cachingConfig.Archive {
		baseConf = ethconfig.ArchiveDefaults
	}

	return &core.CacheConfig{
		TrieCleanLimit:        cachingConfig.TrieCleanCache,
		TrieCleanJournal:      stack.ResolvePath(baseConf.TrieCleanCacheJournal),
		TrieCleanRejournal:    baseConf.TrieCleanCacheRejournal,
		TrieCleanNoPrefetch:   baseConf.NoPrefetch,
		TrieDirtyLimit:        cachingConfig.TrieDirtyCache,
		TrieDirtyDisabled:     cachingConfig.Archive,
		TrieTimeLimit:         cachingConfig.TrieTimeLimit,
		TriesInMemory:         cachingConfig.BlockCount,
		TrieRetention:         cachingConfig.BlockAge,
		SnapshotLimit:         cachingConfig.SnapshotCache,
		Preimages:             baseConf.Preimages,
		SnapshotRestoreMaxGas: cachingConfig.SnapshotRestoreMaxGas,
	}
}

func WriteOrTestGenblock(chainDb ethdb.Database, initData statetransfer.InitDataReader, chainConfig *params.ChainConfig, serializedChainConfig []byte, accountsPerSync uint) error {
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
	stateRoot, err := arbosState.InitializeArbosInDatabase(chainDb, initData, chainConfig, serializedChainConfig, timestamp, accountsPerSync)
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

func BlockChain(chainDb ethdb.Database, cacheConfig *core.CacheConfig, chainConfig *params.ChainConfig, txLookupLimit uint64) (*core.BlockChain, error) {
	engine := arbos.Engine{
		IsSequencer: true,
	}

	vmConfig := vm.Config{
		EnablePreimageRecording: false,
	}

	return core.NewBlockChain(chainDb, cacheConfig, chainConfig, nil, nil, engine, vmConfig, shouldPreserveFalse, &txLookupLimit)
}

func WriteOrTestBlockChain(chainDb ethdb.Database, cacheConfig *core.CacheConfig, initData statetransfer.InitDataReader, chainConfig *params.ChainConfig, serializedChainConfig []byte, txLookupLimit uint64, accountsPerSync uint) (*core.BlockChain, error) {
	err := WriteOrTestGenblock(chainDb, initData, chainConfig, serializedChainConfig, accountsPerSync)
	if err != nil {
		return nil, err
	}
	err = WriteOrTestChainConfig(chainDb, chainConfig)
	if err != nil {
		return nil, err
	}
	return BlockChain(chainDb, cacheConfig, chainConfig, txLookupLimit)
}

// Don't preserve reorg'd out blocks
func shouldPreserveFalse(_ *types.Header) bool {
	return false
}

func ReorgToBlock(chain *core.BlockChain, blockNum uint64) (*types.Block, error) {
	genesisNum := chain.Config().ArbitrumChainParams.GenesisBlockNum
	if blockNum < genesisNum {
		return nil, fmt.Errorf("cannot reorg to block %v past nitro genesis of %v", blockNum, genesisNum)
	}
	reorgingToBlock := chain.GetBlockByNumber(blockNum)
	if reorgingToBlock == nil {
		return nil, fmt.Errorf("didn't find reorg target block number %v", blockNum)
	}
	err := chain.ReorgToOldBlock(reorgingToBlock)
	if err != nil {
		return nil, err
	}
	return reorgingToBlock, nil
}

func init() {
	gethhook.RequireHookedGeth()
}
