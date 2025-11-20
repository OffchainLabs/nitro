package melrunner

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync/atomic"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/bold/containers/fsm"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	stuckFSMIndicatingGauge = metrics.NewRegisteredGauge("arb/mel/stuck", nil) // 1-stuck, 0-not_stuck
)

type MessageExtractionConfig struct {
	Enable                        bool          `koanf:"enable"`
	RetryInterval                 time.Duration `koanf:"retry-interval"`
	DelayedMessageBacklogCapacity int           `koanf:"delayed-message-backlog-capacity"`
	BlocksToPrefetch              uint64        `koanf:"blocks-to-prefetch"`
	ReadMode                      string        `koanf:"read-mode"`
	StallTolerance                uint64        `koanf:"stall-tolerance"`
}

func (c *MessageExtractionConfig) Validate() error {
	c.ReadMode = strings.ToLower(c.ReadMode)
	if c.ReadMode != "latest" && c.ReadMode != "safe" && c.ReadMode != "finalized" {
		return fmt.Errorf("inbox reader read-mode is invalid, want: latest or safe or finalized, got: %s", c.ReadMode)
	}
	return nil
}

var DefaultMessageExtractionConfig = MessageExtractionConfig{
	Enable: false,
	// The retry interval for the message extractor FSM. After each tick of the FSM,
	// the extractor service stop waiter will wait for this duration before trying to act again.
	RetryInterval:                 time.Millisecond * 500,
	DelayedMessageBacklogCapacity: 100, // TODO: right default? setting to a lower value means more calls to l1reader
	BlocksToPrefetch:              499, // 500 is the eth_getLogs block range limit
	ReadMode:                      "latest",
	StallTolerance:                10,
}

var TestMessageExtractionConfig = MessageExtractionConfig{
	Enable:                        false,
	RetryInterval:                 time.Millisecond * 10,
	DelayedMessageBacklogCapacity: 100,
	BlocksToPrefetch:              499,
	ReadMode:                      "latest",
	StallTolerance:                10,
}

func MessageExtractionConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultMessageExtractionConfig.Enable, "enable message extraction service")
	f.Duration(prefix+".retry-interval", DefaultMessageExtractionConfig.RetryInterval, "wait time before retring upon a failure")
	f.Int(prefix+".delayed-message-backlog-capacity", DefaultMessageExtractionConfig.DelayedMessageBacklogCapacity, "target capacity of the delayed message backlog")
	f.Uint64(prefix+".blocks-to-prefetch", DefaultMessageExtractionConfig.BlocksToPrefetch, "the number of blocks to prefetch relevant logs from")
	f.String(prefix+".read-mode", DefaultMessageExtractionConfig.ReadMode, "mode to only read latest or safe or finalized L1 blocks. Enabling safe or finalized disables feed input and output. Defaults to latest. Takes string input, valid strings- latest, safe, finalized")
	f.Uint64(prefix+".stall-tolerance", DefaultMessageExtractionConfig.StallTolerance, "max times the MEL fsm is allowed to be stuck without logging error")
}

// TODO (ganesh): cleanup unused methods from this interface after checking with wasm mode
type ParentChainReader interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	TransactionInBlock(ctx context.Context, blockHash common.Hash, index uint) (*types.Transaction, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error)
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
}

// Defines a message extraction service for a Nitro node which reads parent chain
// blocks one by one to transform them into messages for the execution layer.
type MessageExtractor struct {
	stopwaiter.StopWaiter
	config            MessageExtractionConfig
	parentChainReader ParentChainReader
	logsPreFetcher    *logsFetcher
	addrs             *chaininfo.RollupAddresses
	melDB             *Database
	msgConsumer       mel.MessageConsumer
	dataProviders     *daprovider.ReaderRegistry
	fsm               *fsm.Fsm[action, FSMState]
	caughtUp          bool
	caughtUpChan      chan struct{}
	lastBlockToRead   atomic.Uint64
	stuckCount        uint64
}

// Creates a message extractor instance with the specified parameters,
// including a parent chain reader, rollup addresses, and data providers
// to be used when extracting messages from the parent chain.
func NewMessageExtractor(
	config MessageExtractionConfig,
	parentChainReader ParentChainReader,
	rollupAddrs *chaininfo.RollupAddresses,
	melDB *Database,
	msgConsumer mel.MessageConsumer,
	dataProviders *daprovider.ReaderRegistry,
) (*MessageExtractor, error) {
	fsm, err := newFSM(Start)
	if err != nil {
		return nil, err
	}
	return &MessageExtractor{
		config:            config,
		parentChainReader: parentChainReader,
		addrs:             rollupAddrs,
		melDB:             melDB,
		msgConsumer:       msgConsumer,
		dataProviders:     dataProviders,
		fsm:               fsm,
		caughtUpChan:      make(chan struct{}),
	}, nil
}

// Starts a message extraction service using a stopwaiter. The message extraction
// "loop" consists of a ticking a finite state machine (FSM) that performs different
// responsibilities based on its current state. For instance, processing a parent chain
// block, saving data to a database, or handling reorgs. The FSM is designed to be
// resilient to errors, and each error will retry the same FSM state after a specified interval
// in this Start method.
func (m *MessageExtractor) Start(ctxIn context.Context) error {
	m.StopWaiter.Start(ctxIn, m)
	runChan := make(chan struct{}, 1)
	if m.config.ReadMode != "latest" {
		m.CallIteratively(m.updateLastBlockToRead)
	}
	return stopwaiter.CallIterativelyWith(
		&m.StopWaiterSafe,
		func(ctx context.Context, _ struct{}) time.Duration {
			actAgainInterval, err := m.Act(ctx)
			if err != nil {
				log.Error("Error in message extractor", "err", err)
				m.stuckCount++ // an error implies no change in the fsm state
			} else {
				m.stuckCount = 0
			}
			if m.stuckCount > m.config.StallTolerance {
				stuckFSMIndicatingGauge.Update(1)
				log.Error("Message extractor has been stuck at the same fsm state past the stall-tolerance number of times", "state", m.fsm.Current().State.String(), "stuckCount", m.stuckCount, "err", err)
			} else {
				stuckFSMIndicatingGauge.Update(0)
			}
			return actAgainInterval
		},
		runChan,
	)
}

func (m *MessageExtractor) updateLastBlockToRead(ctx context.Context) time.Duration {
	var header *types.Header
	var err error
	switch m.config.ReadMode {
	case "safe":
		header, err = m.parentChainReader.HeaderByNumber(ctx, big.NewInt(rpc.SafeBlockNumber.Int64()))
	case "finalized":
		header, err = m.parentChainReader.HeaderByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
	}
	if err != nil {
		log.Error("Error fetching header to update last block to read in MEL", "err", err)
		return m.config.RetryInterval
	}
	m.lastBlockToRead.Store(header.Number.Uint64())
	return m.config.RetryInterval
}

func (m *MessageExtractor) CurrentFSMState() FSMState {
	return m.fsm.Current().State
}

func (m *MessageExtractor) getStateByRPCBlockNum(ctx context.Context, blockNum rpc.BlockNumber) (*mel.State, error) {
	blk, err := m.parentChainReader.HeaderByNumber(ctx, big.NewInt(blockNum.Int64()))
	if err != nil {
		return nil, err
	}
	headMelStateBlockNum, err := m.melDB.GetHeadMelStateBlockNum()
	if err != nil {
		return nil, err
	}
	state, err := m.melDB.State(ctx, min(headMelStateBlockNum, blk.Number.Uint64()))
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (m *MessageExtractor) GetSafeMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	state, err := m.getStateByRPCBlockNum(ctx, rpc.SafeBlockNumber)
	if err != nil {
		return 0, err
	}
	return arbutil.MessageIndex(state.MsgCount), nil
}

func (m *MessageExtractor) GetFinalizedMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	state, err := m.getStateByRPCBlockNum(ctx, rpc.FinalizedBlockNumber)
	if err != nil {
		return 0, err
	}
	return arbutil.MessageIndex(state.MsgCount), nil
}

func (m *MessageExtractor) GetFinalizedDelayedMessagesRead(ctx context.Context) (uint64, error) {
	state, err := m.getStateByRPCBlockNum(ctx, rpc.FinalizedBlockNumber)
	if err != nil {
		return 0, err
	}
	return state.DelayedMessagesRead, nil
}

func (m *MessageExtractor) GetHeadState(ctx context.Context) (*mel.State, error) {
	return m.melDB.GetHeadMelState(ctx)
}

func (m *MessageExtractor) GetMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	headState, err := m.melDB.GetHeadMelState(ctx)
	if err != nil {
		return 0, err
	}
	return arbutil.MessageIndex(headState.MsgCount), nil
}

func (d *MessageExtractor) GetDelayedMessage(index uint64) (*mel.DelayedInboxMessage, error) {
	return d.melDB.fetchDelayedMessage(index)
}

func (m *MessageExtractor) GetDelayedCount(ctx context.Context, block uint64) (uint64, error) {
	var state *mel.State
	var err error
	if block == 0 {
		state, err = m.melDB.GetHeadMelState(ctx)
	} else {
		state, err = m.melDB.State(ctx, block)
	}
	if err != nil {
		return 0, err
	}
	return state.DelayedMessagesSeen, nil
}

func (m *MessageExtractor) GetBatchMetadata(seqNum uint64) (mel.BatchMetadata, error) {
	// TODO: have a check to error if seqNum is less than headMelState.BatchCount
	batchMetadata, err := m.melDB.fetchBatchMetadata(seqNum)
	if err != nil {
		return mel.BatchMetadata{}, err
	}
	return *batchMetadata, nil
}

func (m *MessageExtractor) GetBatchMessageCount(seqNum uint64) (arbutil.MessageIndex, error) {
	metadata, err := m.GetBatchMetadata(seqNum)
	return metadata.MessageCount, err
}

func (m *MessageExtractor) GetBatchParentChainBlock(seqNum uint64) (uint64, error) {
	metadata, err := m.GetBatchMetadata(seqNum)
	return metadata.ParentChainBlock, err
}

// err will return unexpected/internal errors
// bool will be false if batch not found (meaning, block not yet posted on a batch)
func (m *MessageExtractor) FindInboxBatchContainingMessage(ctx context.Context, pos arbutil.MessageIndex) (uint64, bool, error) {
	batchCount, err := m.GetBatchCount(ctx)
	if err != nil {
		return 0, false, err
	}
	low := uint64(0)
	high := batchCount - 1
	lastBatchMessageCount, err := m.GetBatchMessageCount(high)
	if err != nil {
		return 0, false, err
	}
	if lastBatchMessageCount <= pos {
		return 0, false, nil
	}
	// Iteration preconditions:
	// - high >= low
	// - msgCount(low - 1) <= pos implies low <= target
	// - msgCount(high) > pos implies high >= target
	// Therefore, if low == high, then low == high == target
	for {
		// Due to integer rounding, mid >= low && mid < high
		mid := (low + high) / 2
		count, err := m.GetBatchMessageCount(mid)
		if err != nil {
			return 0, false, err
		}
		if count < pos {
			// Must narrow as mid >= low, therefore mid + 1 > low, therefore newLow > oldLow
			// Keeps low precondition as msgCount(mid) < pos
			low = mid + 1
		} else if count == pos {
			return mid + 1, true, nil
		} else if count == pos+1 || mid == low { // implied: count > pos
			return mid, true, nil
		} else {
			// implied: count > pos + 1
			// Must narrow as mid < high, therefore newHigh < oldHigh
			// Keeps high precondition as msgCount(mid) > pos
			high = mid
		}
		if high == low {
			return high, true, nil
		}
	}
}

func (m *MessageExtractor) GetBatchCount(ctx context.Context) (uint64, error) {
	headState, err := m.melDB.GetHeadMelState(ctx)
	if err != nil {
		return 0, err
	}
	return headState.BatchCount, nil
}

func (m *MessageExtractor) CaughtUp() chan struct{} {
	return m.caughtUpChan
}

// Ticks the message extractor FSM and performs the action associated with the current state,
// such as processing the next block, saving messages, or handling reorgs.
// Question: do we want to make this private? System tests currently use it, but I believe this should only ever be called by start
func (m *MessageExtractor) Act(ctx context.Context) (time.Duration, error) {
	current := m.fsm.Current()
	switch current.State {
	// `Start` is the initial state of the FSM. It is responsible for
	// initializing the message extraction process. The FSM will transition to
	// the `ProcessingNextBlock` state after successfully fetching the initial
	// MEL state struct for the message extraction process.
	case Start:
		return m.initialize(ctx, current)
	// `ProcessingNextBlock` is the state responsible for processing the next block
	// in the parent chain and extracting messages from it. It uses the
	// `melextraction` package to extract messages and delayed messages
	// from the parent chain block. The FSM will transition to the `SavingMessages`
	// state after successfully extracting messages.
	case ProcessingNextBlock:
		return m.processNextBlock(ctx, current)
	// `SavingMessages` is the state responsible for saving the extracted messages
	// and delayed messages to the database. It stores data in the node's consensus database
	// and runs after the `ProcessingNextBlock` state.
	// After data is stored, the FSM will then transition to the `ProcessingNextBlock` state
	// yet again.
	case SavingMessages:
		return m.saveMessages(ctx, current)
	// `Reorging` is the state responsible for handling reorgs in the parent chain.
	// It is triggered when a reorg occurs, and it will revert the MEL state being processed to the
	// specified block. The FSM will transition to the `ProcessingNextBlock` state
	// based on this old state after the reorg is handled.
	case Reorging:
		return m.reorg(ctx, current)
	default:
		return m.config.RetryInterval, fmt.Errorf("invalid state: %s", current.State)
	}
}
