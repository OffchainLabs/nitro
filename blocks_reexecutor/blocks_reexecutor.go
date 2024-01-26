package blocksreexecutor

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"runtime"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	flag "github.com/spf13/pflag"
)

type Config struct {
	Enable          bool   `koanf:"enable"`
	Mode            string `koanf:"mode"`
	StartBlock      uint64 `koanf:"start-block"`
	EndBlock        uint64 `koanf:"end-block"`
	Room            int    `koanf:"room"`
	BlocksPerThread uint64 `koanf:"blocks-per-thread"`
}

func (c *Config) Validate() error {
	if c.Enable && c.Mode != "random" && c.Mode != "full" {
		return errors.New("invalid mode for blocks re-execution")
	}
	if c.EndBlock < c.StartBlock {
		return errors.New("invalid block range for blocks re-execution")
	}
	if c.Room == 0 {
		return errors.New("room for blocks re-execution cannot be zero")
	}
	return nil
}

var DefaultConfig = Config{
	Enable:          false,
	Mode:            "random",
	Room:            runtime.NumCPU(),
	BlocksPerThread: 10000,
}

var TestConfig = Config{
	Enable:          true,
	Mode:            "full",
	Room:            runtime.NumCPU(),
	BlocksPerThread: 10,
}

func ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, "enables re-execution of a range of blocks against historic state")
	f.String(prefix+".mode", DefaultConfig.Mode, "mode to run the blocks-reexecutor on. Valid modes full and random. full - execute all the blocks in the given range. random - execute a random sample range of blocks with in a given range")
	f.Uint64(prefix+".start-block", DefaultConfig.StartBlock, "first block number of the block range for re-execution")
	f.Uint64(prefix+".end-block", DefaultConfig.EndBlock, "last block number of the block range for re-execution")
	f.Int(prefix+".room", DefaultConfig.Room, "number of threads to parallelize blocks re-execution")
	f.Uint64(prefix+".blocks-per-thread", DefaultConfig.BlocksPerThread, "minimum number of blocks to execute per thread. When mode is random this acts as the size of random block range sample")
}

type BlocksReExecutor struct {
	stopwaiter.StopWaiter
	config       *Config
	blockchain   *core.BlockChain
	stateFor     func(header *types.Header) (*state.StateDB, error)
	done         chan struct{}
	fatalErrChan chan error
	startBlock   uint64
	currentBlock uint64
}

func New(c *Config, blockchain *core.BlockChain, fatalErrChan chan error) *BlocksReExecutor {
	start := c.StartBlock
	end := c.EndBlock
	chainStart := blockchain.Config().ArbitrumChainParams.GenesisBlockNum
	chainEnd := blockchain.CurrentBlock().Number.Uint64()
	if start == 0 && end == 0 {
		start = chainStart
		end = chainEnd
	}
	if start < chainStart {
		log.Warn("state reexecutor's start block number is lower than genesis, resetting to genesis")
		start = chainStart
	}
	if end > chainEnd {
		log.Warn("state reexecutor's end block number is greater than latest, resetting to latest")
		end = chainEnd
	}
	if c.Mode == "random" && end != start {
		if c.BlocksPerThread > end-start {
			c.BlocksPerThread = end - start
		}
		start += uint64(rand.Intn(int(end - start - c.BlocksPerThread + 1)))
		end = start + c.BlocksPerThread
	}
	// inclusive of block reexecution [start, end]
	if start > 0 {
		start--
	}
	return &BlocksReExecutor{
		config:       c,
		blockchain:   blockchain,
		currentBlock: end,
		startBlock:   start,
		done:         make(chan struct{}, c.Room),
		fatalErrChan: fatalErrChan,
		stateFor:     func(header *types.Header) (*state.StateDB, error) { return blockchain.StateAt(header.Root) },
	}
}

// LaunchBlocksReExecution launches the thread to apply blocks of range [currentBlock-s.config.BlocksPerThread, currentBlock] to the last available valid state
func (s *BlocksReExecutor) LaunchBlocksReExecution(ctx context.Context, currentBlock uint64) uint64 {
	start := arbmath.SaturatingUSub(currentBlock, s.config.BlocksPerThread)
	if start < s.startBlock {
		start = s.startBlock
	}
	startState, startHeader, err := arbitrum.FindLastAvailableState(ctx, s.blockchain, s.stateFor, s.blockchain.GetHeaderByNumber(start), nil, -1)
	if err != nil {
		s.fatalErrChan <- fmt.Errorf("blocksReExecutor failed to get last available state while searching for state at %d, err: %w", start, err)
		return s.startBlock
	}
	start = startHeader.Number.Uint64()
	s.LaunchThread(func(ctx context.Context) {
		_, err := arbitrum.AdvanceStateUpToBlock(ctx, s.blockchain, startState, s.blockchain.GetHeaderByNumber(currentBlock), startHeader, nil)
		if err != nil {
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

func (s *BlocksReExecutor) Start(ctx context.Context) {
	s.StopWaiter.Start(ctx, s)
	s.LaunchThread(s.Impl)
}

func (s *BlocksReExecutor) StopAndWait() {
	s.StopWaiter.StopAndWait()
}
