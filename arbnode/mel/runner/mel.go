package melrunner

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/bold/containers/fsm"
	"github.com/offchainlabs/nitro/arbnode/mel"
	melextraction "github.com/offchainlabs/nitro/arbnode/mel/extraction"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// The default retry interval for the message extractor FSM. After each tick of the FSM,
// the extractor service stop waiter will wait for this duration before trying to act again.
const defaultRetryInterval = time.Second

type MessageExtractionConfig struct {
	Enable                        bool          `koanf:"enable"`
	RetryInterval                 time.Duration `koanf:"retry-interval"`
	DelayedMessageBacklogCapacity int           `koanf:"delayed-message-backlog-capacity"`
	BlocksToPrefetch              uint64        `koanf:"blocks-to-prefetch" reload:"hot"`
}

func (c *MessageExtractionConfig) Validate() error {
	return nil
}

var DefaultMessageExtractionConfig = MessageExtractionConfig{
	Enable:                        false,
	RetryInterval:                 defaultRetryInterval,
	DelayedMessageBacklogCapacity: 100, // TODO: right default? setting to a lower value means more calls to l1reader
	BlocksToPrefetch:              499, // 500 is the eth_getLogs block range limit
}

func MessageExtractionConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultMessageExtractionConfig.Enable, "enable message extraction service")
	f.Duration(prefix+".retry-interval", DefaultMessageExtractionConfig.RetryInterval, "wait time before retring upon a failure")
	f.Int(prefix+".delayed-message-backlog-capacity", DefaultMessageExtractionConfig.DelayedMessageBacklogCapacity, "target capacity of the delayed message backlog")
	f.Uint64(prefix+".blocks-to-prefetch", DefaultMessageExtractionConfig.BlocksToPrefetch, "the number of blocks to prefetch relevant logs from")
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
	config            *MessageExtractionConfig
	parentChainReader ParentChainReader
	logsPreFetcher    *logsFetcher
	addrs             *chaininfo.RollupAddresses
	msgConsumer       mel.MessageConsumer
	dataProviders     []daprovider.Reader
	fsm               *fsm.Fsm[action, FSMState]
	retryInterval     time.Duration
	recorderForArbOS  *ArbOSExtractionRecorder
	caughtUp          bool
	caughtUpChan      chan struct{}
}

// Creates a message extractor instance with the specified parameters,
// including a parent chain reader, rollup addresses, and data providers
// to be used when extracting messages from the parent chain.
func NewMessageExtractor(
	parentChainReader ParentChainReader,
	rollupAddrs *chaininfo.RollupAddresses,
	melDB *Database,
	msgConsumer mel.MessageConsumer,
	dataProviders []daprovider.Reader,
	retryInterval time.Duration,
) (*MessageExtractor, error) {
	if retryInterval == 0 {
		retryInterval = defaultRetryInterval
	}
	fsm, err := newFSM(Start)
	if err != nil {
		return nil, err
	}
	recorder := &ArbOSExtractionRecorder{
		melDB:      melDB,
		txFetcher:  &txByLogFetcher{client: parentChainReader},
		preFetcher: newLogsFetcher(parentChainReader, DefaultMessageExtractionConfig.BlocksToPrefetch),
		caches: &caches{
			delayedMsgs:   make(map[uint64]*mel.DelayedInboxMessage),
			logsByHash:    make(map[common.Hash][]*types.Log),
			logsByTxIndex: make(map[common.Hash]map[uint][]*types.Log),
			txsByHash:     make(map[common.Hash]*types.Transaction),
		},
	}
	return &MessageExtractor{
		parentChainReader: parentChainReader,
		addrs:             rollupAddrs,
		msgConsumer:       msgConsumer,
		dataProviders:     dataProviders,
		fsm:               fsm,
		retryInterval:     retryInterval,
		config:            &DefaultMessageExtractionConfig, //TODO: remove retryInterval as a struct instead use config
		caughtUpChan:      make(chan struct{}),
		recorderForArbOS:  recorder,
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
	return stopwaiter.CallIterativelyWith(
		&m.StopWaiterSafe,
		func(ctx context.Context, ignored struct{}) time.Duration {
			actAgainInterval, err := m.Act(ctx)
			if err != nil {
				log.Error("Error in message extractor", "err", err)
			}
			return actAgainInterval
		},
		runChan,
	)
}

func (m *MessageExtractor) MessageExtractionRecorder() *ArbOSExtractionRecorder {
	return m.recorderForArbOS
}

// txByLogFetcher is wrapper around ParentChainReader to implement TransactionByLog method
type txByLogFetcher struct {
	client ParentChainReader
}

func (f *txByLogFetcher) TransactionByLog(ctx context.Context, log *types.Log) (*types.Transaction, error) {
	if log == nil {
		return nil, errors.New("transactionByLog got nil log value")
	}
	tx, _, err := f.client.TransactionByHash(ctx, log.TxHash)
	return tx, err
}

func (m *MessageExtractor) CurrentFSMState() FSMState {
	return m.fsm.Current().State
}

func (m *MessageExtractor) getStateByRPCBlockNum(ctx context.Context, blockNum rpc.BlockNumber) (*mel.State, error) {
	blk, err := m.parentChainReader.HeaderByNumber(ctx, big.NewInt(blockNum.Int64()))
	if err != nil {
		return nil, err
	}
	headMelStateBlockNum, err := m.recorderForArbOS.melDB.GetHeadMelStateBlockNum()
	if err != nil {
		return nil, err
	}
	state, err := m.recorderForArbOS.melDB.State(ctx, min(headMelStateBlockNum, blk.Number.Uint64()))
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
	return m.recorderForArbOS.melDB.GetHeadMelState(ctx)
}

func (m *MessageExtractor) GetMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	headState, err := m.recorderForArbOS.melDB.GetHeadMelState(ctx)
	if err != nil {
		return 0, err
	}
	return arbutil.MessageIndex(headState.MsgCount), nil
}

func (d *MessageExtractor) GetDelayedMessage(index uint64) (*mel.DelayedInboxMessage, error) {
	return d.recorderForArbOS.melDB.fetchDelayedMessage(index)
}

func (m *MessageExtractor) GetDelayedCount(ctx context.Context, block uint64) (uint64, error) {
	var state *mel.State
	var err error
	if block == 0 {
		state, err = m.recorderForArbOS.melDB.GetHeadMelState(ctx)
	} else {
		state, err = m.recorderForArbOS.melDB.State(ctx, block)
	}
	if err != nil {
		return 0, err
	}
	return state.DelayedMessagedSeen, nil
}

func (m *MessageExtractor) GetBatchMetadata(seqNum uint64) (mel.BatchMetadata, error) {
	// TODO: have a check to error if seqNum is less than headMelState.BatchCount
	batchMetadata, err := m.recorderForArbOS.melDB.fetchBatchMetadata(seqNum)
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
	headState, err := m.recorderForArbOS.melDB.GetHeadMelState(ctx)
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
		// Start from the latest MEL state we have in the database
		melState, err := m.recorderForArbOS.melDB.GetHeadMelState(ctx)
		if err != nil {
			return m.retryInterval, err
		}
		// Initialize delayedMessageBacklog and add it to the melState
		delayedMessageBacklog, err := mel.NewDelayedMessageBacklog(m.GetContext(), m.config.DelayedMessageBacklogCapacity, m.GetFinalizedDelayedMessagesRead)
		if err != nil {
			return m.retryInterval, err
		}
		if err = InitializeDelayedMessageBacklog(ctx, delayedMessageBacklog, m.recorderForArbOS.melDB, melState, m.GetFinalizedDelayedMessagesRead); err != nil {
			return m.retryInterval, err
		}
		melState.SetDelayedMessageBacklog(delayedMessageBacklog)
		// Start mel state is now ready. Check if the state's parent chain block hash exists in the parent chain
		startBlock, err := m.parentChainReader.HeaderByNumber(ctx, new(big.Int).SetUint64(melState.ParentChainBlockNumber))
		if err != nil {
			return m.retryInterval, fmt.Errorf("failed to get start parent chain block: %d corresponding to head mel state from parent chain: %w", melState.ParentChainBlockNumber, err)
		}
		// We check if our head mel state's parentChainBlockHash matches the one on-chain, if it doesnt then we detected a reorg
		if melState.ParentChainBlockHash != startBlock.Hash() {
			log.Info("MEL detected L1 reorg at the start", "block", melState.ParentChainBlockNumber, "parentChainBlockHash", melState.ParentChainBlockHash, "onchainParentChainBlockHash", startBlock.Hash()) // Log level is Info because L1 reorgs are a common occurrence
			return 0, m.fsm.Do(reorgToOldBlock{
				melState: melState,
			})
		}
		// Initialize logsPreFetcher
		m.logsPreFetcher = newLogsFetcher(m.parentChainReader, m.config.BlocksToPrefetch)
		// Begin the next FSM state immediately.
		return 0, m.fsm.Do(processNextBlock{
			melState: melState,
		})
	// `ProcessingNextBlock` is the state responsible for processing the next block
	// in the parent chain and extracting messages from it. It uses the
	// `melextraction` package to extract messages and delayed messages
	// from the parent chain block. The FSM will transition to the `SavingMessages`
	// state after successfully extracting messages.
	case ProcessingNextBlock:
		// Process the next block in the parent chain and extracts messages.
		processAction, ok := current.SourceEvent.(processNextBlock)
		if !ok {
			return m.retryInterval, fmt.Errorf("invalid action: %T", current.SourceEvent)
		}
		preState := processAction.melState
		if preState.GetDelayedMessageBacklog() == nil { // Safety check since its relevant for native mode
			return m.retryInterval, errors.New("detected nil DelayedMessageBacklog of melState, shouldnt be possible")
		}
		parentChainBlock, err := m.parentChainReader.HeaderByNumber(
			ctx,
			new(big.Int).SetUint64(preState.ParentChainBlockNumber+1),
		)
		if err != nil {
			if errors.Is(err, ethereum.NotFound) {
				// If the block with the specified number is not found, it likely has not
				// been posted yet to the parent chain, so we can retry
				// without returning an error from the FSM.
				if !m.caughtUp {
					if latestBlk, err := m.parentChainReader.HeaderByNumber(ctx, big.NewInt(rpc.LatestBlockNumber.Int64())); err != nil {
						log.Error("Error fetching LatestBlockNumber from parent chain to determine if mel has caught up", "err", err)
					} else if latestBlk.Number.Uint64()-preState.ParentChainBlockNumber <= 5 { // tolerance of catching up i.e parent chain might have progressed in the time between the above two function calls
						m.caughtUp = true
						close(m.caughtUpChan)
					}
				}
				return m.retryInterval, nil
			} else {
				return m.retryInterval, err
			}
		}
		if parentChainBlock.ParentHash != preState.ParentChainBlockHash {
			log.Info("MEL detected L1 reorg", "block", preState.ParentChainBlockNumber) // Log level is Info because L1 reorgs are a common occurrence
			return 0, m.fsm.Do(reorgToOldBlock{
				melState: preState,
			})
		}
		// Conditionally prefetch logs for upcoming block/s
		if err = m.recorderForArbOS.preFetcher.fetch(ctx, preState); err != nil {
			return m.retryInterval, err
		}
		postState, msgs, delayedMsgs, batchMetas, err := melextraction.ExtractMessages(
			ctx,
			preState,
			parentChainBlock,
			m.dataProviders,
			m.recorderForArbOS,
			m.recorderForArbOS,
			m.recorderForArbOS,
		)
		if err != nil {
			return m.retryInterval, err
		}
		// Begin the next FSM state immediately.
		return 0, m.fsm.Do(saveMessages{
			preStateMsgCount: preState.MsgCount,
			postState:        postState,
			messages:         msgs,
			delayedMessages:  delayedMsgs,
			batchMetas:       batchMetas,
		})
	// `SavingMessages` is the state responsible for saving the extracted messages
	// and delayed messages to the database. It stores data in the node's consensus database
	// and runs after the `ProcessingNextBlock` state.
	// After data is stored, the FSM will then transition to the `ProcessingNextBlock` state
	// yet again.
	case SavingMessages:
		// Persists messages and a processed MEL state to the database.
		saveAction, ok := current.SourceEvent.(saveMessages)
		if !ok {
			return m.retryInterval, fmt.Errorf("invalid action: %T", current.SourceEvent)
		}
		if err := m.recorderForArbOS.melDB.SaveBatchMetas(ctx, saveAction.postState, saveAction.batchMetas); err != nil {
			return m.retryInterval, err
		}
		if err := m.recorderForArbOS.melDB.SaveDelayedMessages(ctx, saveAction.postState, saveAction.delayedMessages); err != nil {
			return m.retryInterval, err
		}
		if err := m.msgConsumer.PushMessages(ctx, saveAction.preStateMsgCount, saveAction.messages); err != nil {
			return m.retryInterval, err
		}
		if err := m.recorderForArbOS.melDB.SaveState(ctx, saveAction.postState); err != nil {
			log.Error("Error saving messages from MessageExtractor to MessageConsumer", "err", err)
			return m.retryInterval, err
		}
		return 0, m.fsm.Do(processNextBlock{
			melState: saveAction.postState,
		})
	// `Reorging` is the state responsible for handling reorgs in the parent chain.
	// It is triggered when a reorg occurs, and it will revert the MEL state being processed to the
	// specified block. The FSM will transition to the `ProcessingNextBlock` state
	// based on this old state after the reorg is handled.
	case Reorging:
		reorgAction, ok := current.SourceEvent.(reorgToOldBlock)
		if !ok {
			return m.retryInterval, fmt.Errorf("invalid action: %T", current.SourceEvent)
		}
		currentDirtyState := reorgAction.melState
		if currentDirtyState.ParentChainBlockNumber == 0 {
			return m.retryInterval, errors.New("invalid reorging stage, ParentChainBlockNumber of current mel state has reached 0")
		}
		previousState, err := m.recorderForArbOS.melDB.State(ctx, currentDirtyState.ParentChainBlockNumber-1)
		if err != nil {
			return m.retryInterval, err
		}
		// This adjusts delayedMessageBacklog
		if err := currentDirtyState.ReorgTo(previousState); err != nil {
			return m.retryInterval, err
		}
		m.logsPreFetcher.reset()
		return 0, m.fsm.Do(processNextBlock{
			melState: previousState,
		})
	default:
		return m.retryInterval, fmt.Errorf("invalid state: %s", current.State)
	}
}
