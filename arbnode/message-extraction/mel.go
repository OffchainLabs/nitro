package mel

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
	extractionfunction "github.com/offchainlabs/nitro/arbnode/message-extraction/extraction-function"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// The default retry interval for the message extractor FSM. After each tick of the FSM,
// the extractor service stop waiter will wait for this duration before trying to act again.
const defaultRetryInterval = time.Second

type MessageExtractionConfig struct {
	Enable        bool          `koanf:"enable"`
	RetryInterval time.Duration `koanf:"retry-interval"`
}

func (c *MessageExtractionConfig) Validate() error {
	return nil
}

var DefaultMessageExtractionConfig = MessageExtractionConfig{
	Enable:        false,
	RetryInterval: defaultRetryInterval,
}

func MessageExtractionConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultMessageExtractionConfig.Enable, "enable message extraction service")
	f.Duration(prefix+".retry-interval", DefaultMessageExtractionConfig.RetryInterval, "wait time before retring upon a failure")
}

type ParentChainReader interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	TransactionInBlock(ctx context.Context, blockHash common.Hash, index uint) (*types.Transaction, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

// Defines a message extraction service for a Nitro node which reads parent chain
// blocks one by one to transform them into messages for the execution layer.
type MessageExtractor struct {
	stopwaiter.StopWaiter
	parentChainReader         ParentChainReader
	initialStateFetcher       meltypes.InitialStateFetcher
	addrs                     *chaininfo.RollupAddresses
	melDB                     meltypes.StateDatabase
	msgConsumer               meltypes.MessageConsumer
	dataProviders             []daprovider.Reader
	startParentChainBlockHash common.Hash
	fsm                       *fsm.Fsm[action, FSMState]
	retryInterval             time.Duration
}

// Creates a message extractor instance with the specified parameters,
// including a parent chain reader, rollup addresses, and data providers
// to be used when extracting messages from the parent chain.
func NewMessageExtractor(
	parentChainReader ParentChainReader,
	rollupAddrs *chaininfo.RollupAddresses,
	initialStateFetcher meltypes.InitialStateFetcher,
	melDB meltypes.StateDatabase,
	msgConsumer meltypes.MessageConsumer,
	dataProviders []daprovider.Reader,
	startParentChainBlockHash common.Hash,
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
		parentChainReader:         parentChainReader,
		addrs:                     rollupAddrs,
		initialStateFetcher:       initialStateFetcher,
		melDB:                     melDB,
		msgConsumer:               msgConsumer,
		dataProviders:             dataProviders,
		startParentChainBlockHash: startParentChainBlockHash,
		fsm:                       fsm,
		retryInterval:             retryInterval,
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

// Instantiates a receipt fetcher for a specific parent chain block.
type blockReceiptFetcher struct {
	client           ParentChainReader
	parentChainBlock *types.Block
}

func newBlockReceiptFetcher(client ParentChainReader, parentChainBlock *types.Block) *blockReceiptFetcher {
	return &blockReceiptFetcher{
		client:           client,
		parentChainBlock: parentChainBlock,
	}
}

func (rf *blockReceiptFetcher) ReceiptForTransactionIndex(
	ctx context.Context,
	txIndex uint,
) (*types.Receipt, error) {
	tx, err := rf.client.TransactionInBlock(ctx, rf.parentChainBlock.Hash(), txIndex)
	if err != nil {
		return nil, err
	}
	return rf.client.TransactionReceipt(ctx, tx.Hash())
}

type blockTxsFetcher struct {
	client           ParentChainReader
	parentChainBlock *types.Block
}

func newBlockTxsFetcher(client ParentChainReader, parentChainBlock *types.Block) *blockTxsFetcher {
	return &blockTxsFetcher{
		client:           client,
		parentChainBlock: parentChainBlock,
	}
}

func (tf *blockTxsFetcher) TransactionsByHeader(
	ctx context.Context,
	parentChainHeaderHash common.Hash,
) (types.Transactions, error) {
	blk, err := tf.client.BlockByHash(ctx, parentChainHeaderHash)
	if err != nil {
		return nil, err
	}
	return blk.Transactions(), nil
}

func (m *MessageExtractor) CurrentFSMState() FSMState {
	return m.fsm.Current().State
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
		// TODO: Start from the latest MEL state we have in the database if it exists as the first step.
		// Check if the specified start block hash exists in the parent chain.
		if _, err := m.parentChainReader.HeaderByHash(
			ctx,
			m.startParentChainBlockHash,
		); err != nil {
			return m.retryInterval, fmt.Errorf(
				"failed to get start block by hash %s from parent chain: %w",
				m.startParentChainBlockHash,
				err,
			)
		}
		// Finalized block number is used in FetchInitialState to initialize seenUnreadDelayedMetaDeque
		finalizedBlk, err := m.parentChainReader.BlockByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
		if err != nil {
			return m.retryInterval, err
		}
		// Fetch the initial state for MEL from a state fetcher interface by parent chain block hash.
		melState, err := m.initialStateFetcher.FetchInitialState(
			ctx,
			m.startParentChainBlockHash,
			finalizedBlk.NumberU64(),
		)
		if err != nil {
			return m.retryInterval, err
		}
		// Initialize seenUnreadDelayedMetaDeque if its nil
		if melState.GetSeenUnreadDelayedMetaDeque() == nil {
			melState.SetSeenUnreadDelayedMetaDeque(&meltypes.DelayedMetaDeque{})
		}
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
		if preState.GetSeenUnreadDelayedMetaDeque() == nil { // Safety check to avoid panics in the later codepath
			return m.retryInterval, errors.New("detected nil seenUnreadDelayedMetaDeque of melState, shouldnt be possible")
		}

		parentChainBlock, err := m.parentChainReader.BlockByNumber(
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
		if parentChainBlock.ParentHash() != preState.ParentChainBlockHash {
			// Reorg detected
			return 0, m.fsm.Do(reorgToOldBlock{
				melState: preState,
			})
		}
		finalizedBlk, err := m.parentChainReader.BlockByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
		if err != nil {
			log.Error("Error fetching FinalizedBlockNumber from parent chain, clearing of read and finalized delayedMeta from the SeenUnreadDelayedMetaDeque will be retried again later", "err", err)
		}
		if preState.ParentChainBlockNumber <= finalizedBlk.NumberU64() {
			preState.GetSeenUnreadDelayedMetaDeque().ClearReadAndFinalized(preState.DelayedMessagesRead)
		} else {
			if finalizedMelState, err := m.melDB.State(ctx, finalizedBlk.NumberU64()); err != nil {
				log.Error("Error fetching melState corresponding to FinalizedBlockNumber from parent chain, clearing of read and finalized delayedMeta from the SeenUnreadDelayedMetaDeque will be retried again later", "err", err)
			} else {
				preState.GetSeenUnreadDelayedMetaDeque().ClearReadAndFinalized(finalizedMelState.DelayedMessagesRead)
			}
		}
		// Creates a receipt fetcher for the specific parent chain block, to be used
		// by the message extraction function.
		receiptFetcher := newBlockReceiptFetcher(m.parentChainReader, parentChainBlock)
		txsFetcher := newBlockTxsFetcher(m.parentChainReader, parentChainBlock)
		postState, msgs, delayedMsgs, err := extractionfunction.ExtractMessages(
			ctx,
			preState,
			parentChainBlock.Header(),
			m.dataProviders,
			m.melDB,
			receiptFetcher,
			txsFetcher,
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
		// Adjust seenUnreadDelayedMetaDeque
		seenUnreadDelayedMetaDeque := currentDirtyState.GetSeenUnreadDelayedMetaDeque()
		seenUnreadDelayedMetaDeque.ClearReorged(previousState.DelayedMessagedSeen)
		previousState.SetSeenUnreadDelayedMetaDeque(seenUnreadDelayedMetaDeque)
		return 0, m.fsm.Do(processNextBlock{
			melState: previousState,
		})
	default:
		return m.retryInterval, fmt.Errorf("invalid state: %s", current.State)
	}
}
