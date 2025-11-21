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

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbnode/mel/extraction"
	"github.com/offchainlabs/nitro/bold/containers/fsm"
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
	melDB             *Database
	msgConsumer       mel.MessageConsumer
	dataProviders     *daprovider.ReaderRegistry
	fsm               *fsm.Fsm[action, FSMState]
	retryInterval     time.Duration
}

// Creates a message extractor instance with the specified parameters,
// including a parent chain reader, rollup addresses, and data providers
// to be used when extracting messages from the parent chain.
func NewMessageExtractor(
	parentChainReader ParentChainReader,
	rollupAddrs *chaininfo.RollupAddresses,
	melDB *Database,
	msgConsumer mel.MessageConsumer,
	dataProviders *daprovider.ReaderRegistry,
	retryInterval time.Duration,
) (*MessageExtractor, error) {
	if retryInterval == 0 {
		retryInterval = defaultRetryInterval
	}
	fsm, err := newFSM(Start)
	if err != nil {
		return nil, err
	}
	return &MessageExtractor{
		parentChainReader: parentChainReader,
		addrs:             rollupAddrs,
		melDB:             melDB,
		msgConsumer:       msgConsumer,
		dataProviders:     dataProviders,
		fsm:               fsm,
		retryInterval:     retryInterval,
		config:            &DefaultMessageExtractionConfig, //TODO: remove retryInterval as a struct instead use config
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

func (m *MessageExtractor) GetFinalizedDelayedMessagesRead(ctx context.Context) (uint64, error) {
	state, err := m.getStateByRPCBlockNum(ctx, rpc.FinalizedBlockNumber)
	if err != nil {
		return 0, err
	}
	return state.DelayedMessagesRead, nil
}

// Ticks the message extractor FSM and performs the action associated with the current state,
// such as processing the next block, saving messages, or handling reorgs.
func (m *MessageExtractor) Act(ctx context.Context) (time.Duration, error) {
	current := m.fsm.Current()
	switch current.State {
	// `Start` is the initial state of the FSM. It is responsible for
	// initializing the message extraction process. The FSM will transition to
	// the `ProcessingNextBlock` state after successfully fetching the initial
	// MEL state struct for the message extraction process.
	case Start:
		// Start from the latest MEL state we have in the database
		melState, err := m.melDB.GetHeadMelState(ctx)
		if err != nil {
			return m.retryInterval, err
		}
		// Initialize delayedMessageBacklog and add it to the melState
		delayedMessageBacklog, err := mel.NewDelayedMessageBacklog(m.GetContext(), m.config.DelayedMessageBacklogCapacity, m.GetFinalizedDelayedMessagesRead)
		if err != nil {
			return m.retryInterval, err
		}
		if err = InitializeDelayedMessageBacklog(ctx, delayedMessageBacklog, m.melDB, melState, m.GetFinalizedDelayedMessagesRead); err != nil {
			return m.retryInterval, err
		}
		delayedMessageBacklog.CommitDirties()
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
	// `extractionfunction` package to extract messages and delayed messages
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
		if err = m.logsPreFetcher.fetch(ctx, preState); err != nil {
			return m.retryInterval, err
		}
		postState, msgs, delayedMsgs, batchMetas, err := melextraction.ExtractMessages(
			ctx,
			preState,
			parentChainBlock,
			m.dataProviders,
			m.melDB,
			&txByLogFetcher{m.parentChainReader},
			m.logsPreFetcher,
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
		saveAction.postState.GetDelayedMessageBacklog().CommitDirties()
		if err := m.melDB.SaveBatchMetas(ctx, saveAction.postState, saveAction.batchMetas); err != nil {
			return m.retryInterval, err
		}
		if err := m.melDB.SaveDelayedMessages(ctx, saveAction.postState, saveAction.delayedMessages); err != nil {
			return m.retryInterval, err
		}
		if err := m.msgConsumer.PushMessages(ctx, saveAction.preStateMsgCount, saveAction.messages); err != nil {
			return m.retryInterval, err
		}
		if err := m.melDB.SaveState(ctx, saveAction.postState); err != nil {
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
		previousState, err := m.melDB.State(ctx, currentDirtyState.ParentChainBlockNumber-1)
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
