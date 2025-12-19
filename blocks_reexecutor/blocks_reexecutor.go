package blocksreexecutor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/hashdb"

	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// lint:require-exhaustive-initialization
type Config struct {
	Enable             bool   `koanf:"enable"`
	Mode               string `koanf:"mode"`
	Blocks             string `koanf:"blocks"` // Range of blocks to be executed in json format
	Room               int    `koanf:"room"`
	MinBlocksPerThread uint64 `koanf:"min-blocks-per-thread"`
	TrieCleanLimit     int    `koanf:"trie-clean-limit"`
	ValidateMultiGas   bool   `koanf:"validate-multigas"`

	blocks [][2]uint64
}

func (c *Config) Validate() error {
	c.Mode = strings.ToLower(c.Mode)
	if c.Enable && c.Mode != "random" && c.Mode != "full" {
		return errors.New("invalid mode for blocks re-execution")
	}
	if c.Blocks == "" {
		return errors.New("list of block ranges to be re-executed cannot be empty")
	}
	var blocks [][2]uint64
	if err := json.Unmarshal([]byte(c.Blocks), &blocks); err != nil {
		return fmt.Errorf("failed to parse blocks re-execution's blocks string: %w", err)
	}
	c.blocks = blocks
	for _, blockRange := range c.blocks {
		if blockRange[1] < blockRange[0] {
			return errors.New("invalid block range for blocks re-execution")
		}
	}
	if c.Room <= 0 {
		return errors.New("room for blocks re-execution should be greater than 0")
	}
	return nil
}

var DefaultConfig = Config{
	Enable:             false,
	Mode:               "random",
	Room:               util.GoMaxProcs(),
	Blocks:             `[[0,0]]`, // execute from chain start to chain end
	MinBlocksPerThread: 0,
	TrieCleanLimit:     0,
	ValidateMultiGas:   false,
	blocks:             nil,
}

var TestConfig = Config{
	Enable:             true,
	Mode:               "full",
	Blocks:             `[[0,0]]`, // execute from chain start to chain end
	Room:               util.GoMaxProcs(),
	TrieCleanLimit:     600,
	MinBlocksPerThread: 0,
	ValidateMultiGas:   true,

	blocks: [][2]uint64{},
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, "enables re-execution of a range of blocks against historic state")
	f.String(prefix+".mode", DefaultConfig.Mode, "mode to run the blocks-reexecutor on. Valid modes full and random. full - execute all the blocks in the given range. random - execute a random sample range of blocks with in a given range")
	f.String(prefix+".blocks", DefaultConfig.Blocks, "json encoded list of block ranges in the form of start and end block numbers in a list of size 2")
	f.Int(prefix+".room", DefaultConfig.Room, "number of threads to parallelize blocks re-execution")
	f.Uint64(prefix+".min-blocks-per-thread", DefaultConfig.MinBlocksPerThread, "minimum number of blocks to execute per thread. When mode is random this acts as the size of random block range sample")
	f.Int(prefix+".trie-clean-limit", DefaultConfig.TrieCleanLimit, "memory allowance (MB) to use for caching trie nodes in memory")
	f.Bool(prefix+".validate-multigas", DefaultConfig.ValidateMultiGas, "if set, validate the sum of multi-gas dimensions match the single-gas")
}

// lint:require-exhaustive-initialization
type BlocksReExecutor struct {
	stopwaiter.StopWaiter
	config       *Config
	db           state.Database
	blockchain   *core.BlockChain
	stateFor     arbitrum.StateForHeaderFunction
	done         chan struct{}
	fatalErrChan chan error
	blocks       [][3]uint64 // start, end and minBlocksPerThread of block ranges
	mutex        sync.Mutex
}

func New(c *Config, blockchain *core.BlockChain, ethDb ethdb.Database, fatalErrChan chan error) (*BlocksReExecutor, error) {
	if blockchain.TrieDB().Scheme() == rawdb.PathScheme {
		return nil, errors.New("blocksReExecutor not supported on pathdb")
	}
	chainStart := blockchain.Config().ArbitrumChainParams.GenesisBlockNum
	chainEnd := blockchain.CurrentBlock().Number.Uint64()
	minBlocksPerThread := uint64(10000)
	if c.MinBlocksPerThread != 0 {
		minBlocksPerThread = c.MinBlocksPerThread
	}
	var blocks [][3]uint64
	for _, blockRange := range c.blocks {
		start := blockRange[0]
		end := blockRange[1]
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
		var work uint64
		if c.MinBlocksPerThread == 0 {
			// #nosec G115
			work = (end - start) / uint64(c.Room*2)
		}
		if work > 0 {
			blocks = append(blocks, [3]uint64{start, end, work})
		} else {
			blocks = append(blocks, [3]uint64{start, end, minBlocksPerThread})
		}
	}
	// We sort the block ranges in descending order of their endBlocks to avoid duplicate reexecution of blocks
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i][1] > blocks[j][1]
	})
	hashConfig := *hashdb.Defaults
	hashConfig.CleanCacheSize = c.TrieCleanLimit * 1024 * 1024
	trieConfig := triedb.Config{
		Preimages: false,
		HashDB:    &hashConfig,
	}

	var blocksReExecutor *BlocksReExecutor

	stateForFunc := func(header *types.Header) (*state.StateDB, arbitrum.StateReleaseFunc, error) {
		blocksReExecutor.mutex.Lock()
		defer blocksReExecutor.mutex.Unlock()
		sdb, err := state.New(header.Root, blocksReExecutor.db)
		if err == nil {
			_ = blocksReExecutor.db.TrieDB().Reference(header.Root, common.Hash{}) // Will be dereferenced later in advanceStateUpToBlock
			return sdb, func() { blocksReExecutor.dereferenceRoot(header.Root) }, nil
		}
		return sdb, arbitrum.NoopStateRelease, err
	}

	blocksReExecutor = &BlocksReExecutor{
		StopWaiter:   stopwaiter.StopWaiter{},
		config:       c,
		db:           state.NewDatabase(triedb.NewDatabase(ethDb, &trieConfig), nil),
		blockchain:   blockchain,
		stateFor:     stateForFunc,
		blocks:       blocks,
		done:         make(chan struct{}, c.Room),
		fatalErrChan: fatalErrChan,
		mutex:        sync.Mutex{},
	}
	return blocksReExecutor, nil
}

func logState(header *types.Header, hasState bool) {
	if height := header.Number.Uint64(); height%1_000_000 == 0 {
		log.Info("Finding last available state.", "block", height, "hash", header.Hash(), "hasState", hasState)
	}
}

// LaunchBlocksReExecution launches the thread to apply blocks of range [currentBlock-s.config.MinBlocksPerThread, currentBlock] to the last available valid state
func (s *BlocksReExecutor) LaunchBlocksReExecution(ctx context.Context, startBlock, currentBlock, minBlocksPerThread uint64) uint64 {
	start := arbmath.SaturatingUSub(currentBlock, minBlocksPerThread)
	if start < startBlock {
		start = startBlock
	}
	startState, startHeader, release, err := arbitrum.FindLastAvailableState(ctx, s.blockchain, s.stateFor, s.blockchain.GetHeaderByNumber(start), logState, -1)
	if err != nil {
		s.fatalErrChan <- fmt.Errorf("blocksReExecutor failed to get last available state while searching for state at %d, err: %w", start, err)
		return startBlock
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

func (s *BlocksReExecutor) Impl(ctx context.Context, startBlock, currentBlock, minBlocksPerThread uint64) uint64 {
	var threadsLaunched uint64
	end := currentBlock
	for i := 0; i < s.config.Room && currentBlock > startBlock; i++ {
		threadsLaunched++
		currentBlock = s.LaunchBlocksReExecution(ctx, startBlock, currentBlock, minBlocksPerThread)
	}
	for {
		select {
		case <-s.done:
			if currentBlock > startBlock {
				currentBlock = s.LaunchBlocksReExecution(ctx, startBlock, currentBlock, minBlocksPerThread)
			} else {
				threadsLaunched--
			}

		case <-ctx.Done():
			return 0
		}
		if threadsLaunched == 0 {
			break
		}
	}
	log.Info("BlocksReExecutor successfully completed re-execution of blocks against historic state", "stateAt", startBlock, "startBlock", startBlock+1, "endBlock", end)
	return currentBlock
}

func (s *BlocksReExecutor) Start(ctx context.Context, done chan struct{}) {
	s.StopWaiter.Start(ctx, s)
	s.LaunchThread(func(ctx context.Context) {
		// Using returned value from Impl we can avoid duplicate reexecution of blocks
		// lowestBlockNotReExecuted represents the block after which either all the blocks have already been reexecuted or not in scope of reexecution
		lowestBlockNotReExecuted := s.blocks[0][1] + 1
		for _, blocks := range s.blocks {
			if lowestBlockNotReExecuted > blocks[0] {
				lowestBlockNotReExecuted = s.Impl(ctx, blocks[0], min(lowestBlockNotReExecuted, blocks[1]), blocks[2])
			} else {
				log.Info("BlocksReExecutor successfully completed re-execution of blocks against historic state", "stateAt", blocks[0], "startBlock", blocks[0]+1, "endBlock", blocks[1])
			}
		}
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
	result, err := statedb.Commit(blockNumber, true, false)
	if err != nil {
		return nil, arbitrum.NoopStateRelease, err
	}
	if result != expected {
		return nil, arbitrum.NoopStateRelease, fmt.Errorf("bad root hash expected: %v got: %v", expected, result)
	}
	sdb, err := state.New(result, s.db)
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
	vmConfig := vm.Config{
		ExposeMultiGas: s.config.ValidateMultiGas,
	}
	for ctx.Err() == nil {
		var receipts types.Receipts
		state, block, receipts, err = arbitrum.AdvanceStateByBlock(ctx, s.blockchain, state, blockToRecreate, prevHash, nil, vmConfig)
		if err != nil {
			return err
		}

		if vmConfig.ExposeMultiGas {
			for _, receipt := range receipts {
				if receipt.GasUsed != receipt.MultiGasUsed.SingleGas() {
					return fmt.Errorf("multi-dimensional gas mismatch in block %d, txHash %s: gasUsed=%d, multiGasUsed=%d",
						block.NumberU64(), receipt.TxHash, receipt.GasUsed, receipt.MultiGasUsed.SingleGas())
				}
			}
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
