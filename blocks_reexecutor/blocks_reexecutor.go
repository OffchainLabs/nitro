package blocksreexecutor

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"sync"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/hashdb"

	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type Config struct {
	Enable             bool   `koanf:"enable"`
	Mode               string `koanf:"mode"`
	StartBlock         uint64 `koanf:"start-block"`
	EndBlock           uint64 `koanf:"end-block"`
	Room               int    `koanf:"room"`
	MinBlocksPerThread uint64 `koanf:"min-blocks-per-thread"`
	TrieCleanLimit     int    `koanf:"trie-clean-limit"`
}

func (c *Config) Validate() error {
	c.Mode = strings.ToLower(c.Mode)
	if c.Enable && c.Mode != "random" && c.Mode != "full" {
		return errors.New("invalid mode for blocks re-execution")
	}
	if c.EndBlock < c.StartBlock {
		return errors.New("invalid block range for blocks re-execution")
	}
	if c.Room <= 0 {
		return errors.New("room for blocks re-execution should be greater than 0")
	}
	return nil
}

var DefaultConfig = Config{
	Enable: false,
	Mode:   "random",
	Room:   runtime.NumCPU(),
}

var TestConfig = Config{
	Enable:             true,
	Mode:               "full",
	Room:               runtime.NumCPU(),
	MinBlocksPerThread: 10,
	TrieCleanLimit:     600,
}

func ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, "enables re-execution of a range of blocks against historic state")
	f.String(prefix+".mode", DefaultConfig.Mode, "mode to run the blocks-reexecutor on. Valid modes full and random. full - execute all the blocks in the given range. random - execute a random sample range of blocks with in a given range")
	f.Uint64(prefix+".start-block", DefaultConfig.StartBlock, "first block number of the block range for re-execution")
	f.Uint64(prefix+".end-block", DefaultConfig.EndBlock, "last block number of the block range for re-execution")
	f.Int(prefix+".room", DefaultConfig.Room, "number of threads to parallelize blocks re-execution")
	f.Uint64(prefix+".min-blocks-per-thread", DefaultConfig.MinBlocksPerThread, "minimum number of blocks to execute per thread. When mode is random this acts as the size of random block range sample")
	f.Int(prefix+".trie-clean-limit", DefaultConfig.TrieCleanLimit, "memory allowance (MB) to use for caching trie nodes in memory")
}

type BlocksReExecutor struct {
	stopwaiter.StopWaiter
	config             *Config
	db                 state.Database
	blockchain         *core.BlockChain
	stateFor           arbitrum.StateForHeaderFunction
	done               chan struct{}
	fatalErrChan       chan error
	startBlock         uint64
	currentBlock       uint64
	minBlocksPerThread uint64
	mutex              sync.Mutex
}

func New(c *Config, blockchain *core.BlockChain, ethDb ethdb.Database, fatalErrChan chan error) (*BlocksReExecutor, error) {
	if blockchain.TrieDB().Scheme() == rawdb.PathScheme {
		return nil, errors.New("blocksReExecutor not supported on pathdb")
	}
	start := c.StartBlock
	end := c.EndBlock
	chainStart := blockchain.Config().ArbitrumChainParams.GenesisBlockNum
	chainEnd := blockchain.CurrentBlock().Number.Uint64()
	if start == 0 && end == 0 {
		start = chainStart
		end = chainEnd
	}
	if start < chainStart || start > chainEnd {
		log.Warn("invalid state reexecutor's start block number, resetting to genesis", "start", start, "genesis", chainStart)
		start = chainStart
	}
	if end > chainEnd || end < chainStart {
		log.Warn("invalid state reexecutor's end block number, resetting to latest", "end", end, "latest", chainEnd)
		end = chainEnd
	}
	minBlocksPerThread := uint64(10000)
	if c.MinBlocksPerThread != 0 {
		minBlocksPerThread = c.MinBlocksPerThread
	}
	if c.Mode == "random" && end != start {
		// Reexecute a range of 10000 or (non-zero) c.MinBlocksPerThread number of blocks between start to end picked randomly
		rng := minBlocksPerThread
		if rng > end-start {
			rng = end - start
		}
		// #nosec G115
		start += uint64(rand.Int63n(int64(end - start - rng + 1)))
		end = start + rng
	}
	// Inclusive of block reexecution [start, end]
	// Do not reexecute genesis block i,e chainStart
	if start > 0 && start != chainStart {
		start--
	}
	// Divide work equally among available threads when MinBlocksPerThread is zero
	if c.MinBlocksPerThread == 0 {
		// #nosec G115
		work := (end - start) / uint64(c.Room*2)
		if work > 0 {
			minBlocksPerThread = work
		}
	}
	hashConfig := *hashdb.Defaults
	hashConfig.CleanCacheSize = c.TrieCleanLimit * 1024 * 1024
	trieConfig := triedb.Config{
		Preimages: false,
		HashDB:    &hashConfig,
	}
	blocksReExecutor := &BlocksReExecutor{
		config:             c,
		db:                 state.NewDatabaseWithConfig(ethDb, &trieConfig),
		blockchain:         blockchain,
		currentBlock:       end,
		startBlock:         start,
		minBlocksPerThread: minBlocksPerThread,
		done:               make(chan struct{}, c.Room),
		fatalErrChan:       fatalErrChan,
	}
	blocksReExecutor.stateFor = func(header *types.Header) (*state.StateDB, arbitrum.StateReleaseFunc, error) {
		blocksReExecutor.mutex.Lock()
		defer blocksReExecutor.mutex.Unlock()
		sdb, err := state.New(header.Root, blocksReExecutor.db, nil)
		if err == nil {
			_ = blocksReExecutor.db.TrieDB().Reference(header.Root, common.Hash{}) // Will be dereferenced later in advanceStateUpToBlock
			return sdb, func() { blocksReExecutor.dereferenceRoot(header.Root) }, nil
		}
		return sdb, arbitrum.NoopStateRelease, err
	}
	return blocksReExecutor, nil
}

// LaunchBlocksReExecution launches the thread to apply blocks of range [currentBlock-s.config.MinBlocksPerThread, currentBlock] to the last available valid state
func (s *BlocksReExecutor) LaunchBlocksReExecution(ctx context.Context, currentBlock uint64) uint64 {
	start := arbmath.SaturatingUSub(currentBlock, s.minBlocksPerThread)
	if start < s.startBlock {
		start = s.startBlock
	}
	startState, startHeader, release, err := arbitrum.FindLastAvailableState(ctx, s.blockchain, s.stateFor, s.blockchain.GetHeaderByNumber(start), nil, -1)
	if err != nil {
		s.fatalErrChan <- fmt.Errorf("blocksReExecutor failed to get last available state while searching for state at %d, err: %w", start, err)
		return s.startBlock
	}
	start = startHeader.Number.Uint64()
	s.LaunchThread(func(ctx context.Context) {
		log.Info("Starting reexecution of blocks against historic state", "stateAt", start, "startBlock", start+1, "endBlock", currentBlock)
		if err := s.advanceStateUpToBlock(ctx, startState, s.blockchain.GetHeaderByNumber(currentBlock), startHeader, release); err != nil {
			s.fatalErrChan <- fmt.Errorf("blocksReExecutor errored advancing state from block %d to block %d, err: %w", start, currentBlock, err)
		} else {
			log.Info("Successfully reexecuted blocks against historic state", "stateAt", start, "startBlock", start+1, "endBlock", currentBlock)
		}
		s.done <- struct{}{}
	})
	return start
}

func (s *BlocksReExecutor) Impl(ctx context.Context) {
	var threadsLaunched uint64
	end := s.currentBlock
	for i := 0; i < s.config.Room && s.currentBlock > s.startBlock; i++ {
		threadsLaunched++
		s.currentBlock = s.LaunchBlocksReExecution(ctx, s.currentBlock)
	}
	for {
		select {
		case <-s.done:
			if s.currentBlock > s.startBlock {
				s.currentBlock = s.LaunchBlocksReExecution(ctx, s.currentBlock)
			} else {
				threadsLaunched--
			}

		case <-ctx.Done():
			return
		}
		if threadsLaunched == 0 {
			break
		}
	}
	log.Info("BlocksReExecutor successfully completed re-execution of blocks against historic state", "stateAt", s.startBlock, "startBlock", s.startBlock+1, "endBlock", end)
}

func (s *BlocksReExecutor) Start(ctx context.Context, done chan struct{}) {
	s.StopWaiter.Start(ctx, s)
	s.LaunchThread(func(ctx context.Context) {
		s.Impl(ctx)
		if done != nil {
			close(done)
		}
	})
}

func (s *BlocksReExecutor) StopAndWait() {
	s.StopWaiter.StopAndWait()
}

func (s *BlocksReExecutor) dereferenceRoot(root common.Hash) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_ = s.db.TrieDB().Dereference(root)
}

func (s *BlocksReExecutor) commitStateAndVerify(statedb *state.StateDB, expected common.Hash, blockNumber uint64) (*state.StateDB, arbitrum.StateReleaseFunc, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	result, err := statedb.Commit(blockNumber, true)
	if err != nil {
		return nil, arbitrum.NoopStateRelease, err
	}
	if result != expected {
		return nil, arbitrum.NoopStateRelease, fmt.Errorf("bad root hash expected: %v got: %v", expected, result)
	}
	sdb, err := state.New(result, s.db, nil)
	if err == nil {
		_ = s.db.TrieDB().Reference(result, common.Hash{})
		return sdb, func() { s.dereferenceRoot(result) }, nil
	}
	return sdb, arbitrum.NoopStateRelease, err
}

func (s *BlocksReExecutor) advanceStateUpToBlock(ctx context.Context, state *state.StateDB, targetHeader *types.Header, lastAvailableHeader *types.Header, lastRelease arbitrum.StateReleaseFunc) error {
	targetBlockNumber := targetHeader.Number.Uint64()
	blockToRecreate := lastAvailableHeader.Number.Uint64() + 1
	prevHash := lastAvailableHeader.Hash()
	var stateRelease arbitrum.StateReleaseFunc
	defer func() {
		lastRelease()
	}()
	var block *types.Block
	var err error
	for ctx.Err() == nil {
		state, block, err = arbitrum.AdvanceStateByBlock(ctx, s.blockchain, state, blockToRecreate, prevHash, nil)
		if err != nil {
			return err
		}
		prevHash = block.Hash()
		state, stateRelease, err = s.commitStateAndVerify(state, block.Root(), block.NumberU64())
		if err != nil {
			return fmt.Errorf("failed committing state for block %d : %w", blockToRecreate, err)
		}
		lastRelease()
		lastRelease = stateRelease
		if blockToRecreate >= targetBlockNumber {
			if block.Hash() != targetHeader.Hash() {
				return fmt.Errorf("blockHash doesn't match when recreating number: %d expected: %v got: %v", blockToRecreate, targetHeader.Hash(), block.Hash())
			}
			return nil
		}
		blockToRecreate++
	}
	return ctx.Err()
}
