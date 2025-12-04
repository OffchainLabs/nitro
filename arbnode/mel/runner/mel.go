package melrunner

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

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
	addrs                     *chaininfo.RollupAddresses
	melDB                     *Database
	msgConsumer               mel.MessageConsumer
	dataProviders             *daprovider.DAProviderRegistry
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
	melDB *Database,
	msgConsumer mel.MessageConsumer,
	dataProviders *daprovider.DAProviderRegistry,
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
		// Start from the latest MEL state we have in the database
		melState, err := m.melDB.GetHeadMelState(ctx)
		if err != nil {
			return m.retryInterval, err
		}
		// We check if our current head mel state corresponds to this parentChainBlockHash
		if melState.ParentChainBlockHash != m.startParentChainBlockHash {
			return m.retryInterval, fmt.Errorf("head mel state's parentChainBlockHash in db: %v does not match the given parentChainBlockHash: %v ", melState.ParentChainBlockHash, m.startParentChainBlockHash)
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
		// Creates a receipt fetcher for the specific parent chain block, to be used
		// by the message extraction function.
		receiptFetcher := newBlockReceiptFetcher(m.parentChainReader, parentChainBlock)
		txsFetcher := newBlockTxsFetcher(m.parentChainReader, parentChainBlock)
		postState, msgs, delayedMsgs, err := melextraction.ExtractMessages(
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
		// TODO: Implement reorging logic.
		return m.retryInterval, fmt.Errorf("reorg state not implemented")
	default:
		return m.retryInterval, fmt.Errorf("invalid state: %s", current.State)
	}
}
