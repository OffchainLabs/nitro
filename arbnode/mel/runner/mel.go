// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melrunner

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync/atomic"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/mel"
	melextraction "github.com/offchainlabs/nitro/arbnode/mel/extraction"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/bold/containers/fsm"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type MessageExtractionConfig struct {
	Enable                             bool          `koanf:"enable"`
	RetryInterval                      time.Duration `koanf:"retry-interval"`
	DelayedMessageBacklogCapacity      int           `koanf:"delayed-message-backlog-capacity"`
	BlocksToPrefetch                   uint64        `koanf:"blocks-to-prefetch"`
	ReadMode                           string        `koanf:"read-mode"`
	StallTolerance                     uint64        `koanf:"stall-tolerance"`
	LogExtractionStatusFrequencyBlocks uint64        `koanf:"log-extraction-status-frequency-blocks"`
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
	RetryInterval:                      time.Millisecond * 500,
	DelayedMessageBacklogCapacity:      100, // TODO: right default? setting to a lower value means more calls to l1reader
	BlocksToPrefetch:                   499, // 500 is the eth_getLogs block range limit
	ReadMode:                           "latest",
	StallTolerance:                     10,
	LogExtractionStatusFrequencyBlocks: 100,
}

var TestMessageExtractionConfig = MessageExtractionConfig{
	Enable:                             false,
	RetryInterval:                      time.Millisecond * 10,
	DelayedMessageBacklogCapacity:      100,
	BlocksToPrefetch:                   499,
	ReadMode:                           "latest",
	StallTolerance:                     10,
	LogExtractionStatusFrequencyBlocks: 100,
}

func MessageExtractionConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultMessageExtractionConfig.Enable, "enable message extraction service")
	f.Duration(prefix+".retry-interval", DefaultMessageExtractionConfig.RetryInterval, "wait time before retring upon a failure")
	f.Int(prefix+".delayed-message-backlog-capacity", DefaultMessageExtractionConfig.DelayedMessageBacklogCapacity, "target capacity of the delayed message backlog")
	f.Uint64(prefix+".blocks-to-prefetch", DefaultMessageExtractionConfig.BlocksToPrefetch, "the number of blocks to prefetch relevant logs from. Recommend using max allowed range for eth_getLogs rpc query")
	f.String(prefix+".read-mode", DefaultMessageExtractionConfig.ReadMode, "mode to only read latest or safe or finalized L1 blocks. Enabling safe or finalized disables feed input and output. Defaults to latest. Takes string input, valid strings- latest, safe, finalized")
	f.Uint64(prefix+".stall-tolerance", DefaultMessageExtractionConfig.StallTolerance, "max times the MEL fsm is allowed to be stuck without logging error")
	f.Uint64(prefix+".log-extraction-status-frequency-blocks", DefaultMessageExtractionConfig.LogExtractionStatusFrequencyBlocks, "frequency of logging message extraction status in terms of number of blocks processed")
}

// SequencerBatchCountFetcher queries the on-chain sequencer inbox batch count at a given parent chain block.
type SequencerBatchCountFetcher interface {
	GetBatchCount(ctx context.Context, blockNum *big.Int) (uint64, error)
}

// TODO (ganesh): cleanup unused methods from this interface after checking with wasm mode
type ParentChainReader interface {
	Client() rpc.ClientInterface // to make BatchCallContext requests
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
	config                   MessageExtractionConfig
	parentChainReader        ParentChainReader
	chainConfig              *params.ChainConfig
	logsAndHeadersPreFetcher *logsAndHeadersFetcher
	addrs                    *chaininfo.RollupAddresses
	melDB                    *Database
	msgConsumer              mel.MessageConsumer
	dataProviders            *daprovider.DAProviderRegistry
	fsm                      *fsm.Fsm[action, FSMState]
	caughtUp                 bool
	caughtUpChan             chan struct{}
	lastBlockToRead          atomic.Uint64
	stuckCount               uint64
	reorgEventsNotifier      chan uint64
	seqBatchCounter          SequencerBatchCountFetcher
	l1Reader                 *headerreader.HeaderReader
}

// Creates a message extractor instance with the specified parameters,
// including a parent chain reader, rollup addresses, and data providers
// to be used when extracting messages from the parent chain.
func NewMessageExtractor(
	config MessageExtractionConfig,
	parentChainReader ParentChainReader,
	chainConfig *params.ChainConfig,
	rollupAddrs *chaininfo.RollupAddresses,
	melDB *Database,
	dapRegistry *daprovider.DAProviderRegistry,
	seqBatchCounter SequencerBatchCountFetcher,
	l1Reader *headerreader.HeaderReader,
	reorgEventsNotifier chan uint64,
) (*MessageExtractor, error) {
	fsm, err := newFSM(Start)
	if err != nil {
		return nil, err
	}
	return &MessageExtractor{
		config:              config,
		parentChainReader:   parentChainReader,
		chainConfig:         chainConfig,
		addrs:               rollupAddrs,
		melDB:               melDB,
		dataProviders:       dapRegistry,
		fsm:                 fsm,
		caughtUpChan:        make(chan struct{}),
		reorgEventsNotifier: reorgEventsNotifier,
		seqBatchCounter:     seqBatchCounter,
		l1Reader:            l1Reader,
	}, nil
}

func (m *MessageExtractor) SetMessageConsumer(consumer mel.MessageConsumer) error {
	if m.Started() {
		return errors.New("cannot set message consumer after start")
	}
	if m.msgConsumer != nil {
		return errors.New("message consumer already set")
	}
	m.msgConsumer = consumer
	return nil
}

// Starts a message extraction service using a stopwaiter. The message extraction
// "loop" consists of a ticking a finite state machine (FSM) that performs different
// responsibilities based on its current state. For instance, processing a parent chain
// block, saving data to a database, or handling reorgs. The FSM is designed to be
// resilient to errors, and each error will retry the same FSM state after a specified interval
// in this Start method.
func (m *MessageExtractor) Start(ctxIn context.Context) error {
	if m.msgConsumer == nil {
		return errors.New("message consumer not set")
	}
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
	default:
		log.Error("updateLastBlockToRead called with unexpected ReadMode", "mode", m.config.ReadMode)
		return m.config.RetryInterval
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
	state, err := m.melDB.State(min(headMelStateBlockNum, blk.Number.Uint64()))
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

func (m *MessageExtractor) GetSyncProgress(ctx context.Context) (mel.MessageSyncProgress, error) {
	headState, err := m.melDB.GetHeadMelState()
	if err != nil {
		return mel.MessageSyncProgress{}, err
	}
	batchSeen := headState.BatchCount // fallback when seqBatchCounter is nil or returns error
	if m.seqBatchCounter != nil {
		seen, err := m.seqBatchCounter.GetBatchCount(ctx, new(big.Int).SetUint64(headState.ParentChainBlockNumber))
		if err != nil {
			// TODO: Replace with a sentinel error check once geth exposes one for "header not found".
			// This error originates from the RPC/header lookup path, distinct from the database-level
			// not-found errors handled by rawdb.IsDbErrNotFound in FinalizedDelayedMessageAtPosition.
			if strings.Contains(err.Error(), "header not found") {
				log.Debug("SequencerInbox GetBatchCount header not found, using headState.BatchCount fallback", "parentChainBlock", headState.ParentChainBlockNumber)
			} else {
				log.Error("SequencerInbox GetBatchCount error, using headState.BatchCount fallback", "err", err, "parentChainBlock", headState.ParentChainBlockNumber)
			}
		} else {
			batchSeen = seen
		}
	}
	return mel.MessageSyncProgress{
		BatchSeen:      batchSeen,
		BatchProcessed: headState.BatchCount,
		MsgCount:       arbutil.MessageIndex(headState.MsgCount),
	}, nil
}

func (m *MessageExtractor) GetL1Reader() *headerreader.HeaderReader {
	return m.l1Reader
}

// GetFinalizedDelayedMessagesRead uses MessageExtractor's context for calls to parentChainReader
func (m *MessageExtractor) GetFinalizedDelayedMessagesRead() (uint64, error) {
	ctx, err := m.GetContextSafe()
	if err != nil {
		return 0, fmt.Errorf("message extractor not running: %w", err)
	}
	state, err := m.getStateByRPCBlockNum(ctx, rpc.FinalizedBlockNumber)
	if err != nil {
		return 0, err
	}
	return state.DelayedMessagesRead, nil
}

func (m *MessageExtractor) GetHeadState() (*mel.State, error) {
	return m.melDB.GetHeadMelState()
}

func (m *MessageExtractor) GetState(parentchainBlocknumber uint64) (*mel.State, error) {
	return m.melDB.State(parentchainBlocknumber)
}

func (m *MessageExtractor) GetMsgCount() (arbutil.MessageIndex, error) {
	headState, err := m.melDB.GetHeadMelState()
	if err != nil {
		return 0, err
	}
	return arbutil.MessageIndex(headState.MsgCount), nil
}

func (m *MessageExtractor) GetDelayedMessageBytes(ctx context.Context, seqNum uint64) ([]byte, error) {
	msg, err := m.GetDelayedMessage(seqNum)
	if err != nil {
		return nil, err
	}
	return rlp.EncodeToBytes(msg)
}

func (m *MessageExtractor) GetDelayedMessage(index uint64) (*mel.DelayedInboxMessage, error) {
	headState, err := m.melDB.GetHeadMelState()
	if err != nil {
		return nil, err
	}
	if index >= headState.DelayedMessagesSeen {
		return nil, fmt.Errorf("DelayedInboxMessage not available for index: %d greater than head MEL state DelayedMessagesSeen count: %d", index, headState.DelayedMessagesSeen)
	}
	return m.melDB.fetchDelayedMessage(index)
}

func (m *MessageExtractor) GetDelayedAcc(seqNum uint64) (common.Hash, error) {
	delayedMsg, err := m.GetDelayedMessage(seqNum)
	if err != nil {
		return common.Hash{}, err
	}
	return delayedMsg.AfterInboxAcc(), nil
}

// GetDelayedCountAtParentChainBlock uses the caller-provided ctx (not m.GetContext())
// because it is called from FinalizedDelayedMessageAtPosition, which receives its
// context from the DelayedSequencer — a running component that supplies a valid context.
func (m *MessageExtractor) GetDelayedCountAtParentChainBlock(ctx context.Context, parentChainBlockNum uint64) (uint64, error) {
	state, err := m.melDB.State(parentChainBlockNum)
	if err != nil {
		return 0, err
	}
	return state.DelayedMessagesSeen, nil
}

func (m *MessageExtractor) GetDelayedCount() (uint64, error) {
	state, err := m.melDB.GetHeadMelState()
	if err != nil {
		return 0, err
	}
	return state.DelayedMessagesSeen, nil
}

func (m *MessageExtractor) GetBatchMetadata(seqNum uint64) (mel.BatchMetadata, error) {
	headState, err := m.melDB.GetHeadMelState()
	if err != nil {
		return mel.BatchMetadata{}, err
	}
	if seqNum >= headState.BatchCount {
		return mel.BatchMetadata{}, fmt.Errorf("batchMetadata not available for seqNum: %d greater than head MEL state batch count: %d", seqNum, headState.BatchCount)
	}
	batchMetadata, err := m.melDB.fetchBatchMetadata(seqNum)
	if err != nil {
		return mel.BatchMetadata{}, err
	}
	return *batchMetadata, nil
}

func (m *MessageExtractor) SupportsPushingFinalityData() bool {
	return false
}

// FinalizedDelayedMessageAtPosition returns the delayed message at the
// requested position if it is finalized. Returns mel.ErrDelayedMessageNotYetFinalized
// if the delayed count at the finalized block position is not yet available in the
// database, or if the requested position is at or beyond the finalized delayed count.
// Other errors indicate failures fetching the finalized position or the message itself.
// When lastDelayedAccumulator is non-zero, it is validated against the message's
// BeforeInboxAcc to ensure accumulator chain consistency.
func (m *MessageExtractor) FinalizedDelayedMessageAtPosition(
	ctx context.Context,
	finalizedBlock uint64,
	lastDelayedAccumulator common.Hash,
	requestedPosition uint64,
) (*arbostypes.L1IncomingMessage, common.Hash, uint64, error) {
	msg, err := m.GetDelayedMessage(requestedPosition)
	if err != nil {
		return nil, common.Hash{}, 0, fmt.Errorf("MEL: failed to get delayed message at position %d: %w", requestedPosition, err)
	}
	finalizedDelayedCount, err := m.GetDelayedCountAtParentChainBlock(ctx, finalizedBlock)
	if err != nil {
		if rawdb.IsDbErrNotFound(err) {
			log.Debug("MEL delayed count not found for finalized block, treating as not yet finalized", "parentChainBlock", finalizedBlock)
			return nil, common.Hash{}, msg.ParentChainBlockNumber, mel.ErrDelayedMessageNotYetFinalized
		}
		log.Warn("MEL GetDelayedCountAtParentChainBlock failed with unexpected error", "parentChainBlock", finalizedBlock, "err", err)
		return nil, common.Hash{}, 0, err
	}
	if requestedPosition >= finalizedDelayedCount {
		return nil, common.Hash{}, msg.ParentChainBlockNumber, mel.ErrDelayedMessageNotYetFinalized
	}
	if lastDelayedAccumulator != (common.Hash{}) && msg.BeforeInboxAcc != lastDelayedAccumulator {
		return nil, common.Hash{}, 0, fmt.Errorf("position %d (finalized block %d): BeforeInboxAcc %v != lastDelayedAccumulator %v: %w", requestedPosition, finalizedBlock, msg.BeforeInboxAcc, lastDelayedAccumulator, mel.ErrDelayedAccumulatorMismatch)
	}
	return msg.Message, msg.AfterInboxAcc(), msg.ParentChainBlockNumber, nil
}

func (m *MessageExtractor) GetSequencerMessageBytes(ctx context.Context, seqNum uint64) ([]byte, common.Hash, error) {
	metadata, err := m.GetBatchMetadata(seqNum)
	if err != nil {
		return nil, common.Hash{}, err
	}
	// No need to specify a max headers to fetch, as we are using the logs fetcher only, so we can pass in a 0.
	logsFetcher := newLogsAndHeadersFetcher(m.parentChainReader, 0)
	if err = logsFetcher.fetchSequencerBatchLogs(ctx, metadata.ParentChainBlock, metadata.ParentChainBlock); err != nil {
		return nil, common.Hash{}, err
	}
	parentChainHeader, err := m.parentChainReader.HeaderByNumber(ctx, new(big.Int).SetUint64(metadata.ParentChainBlock))
	if err != nil {
		return nil, common.Hash{}, err
	}
	seqBatches, batchTxs, err := melextraction.ParseBatchesFromBlock(ctx, parentChainHeader, &txByLogFetcher{m.parentChainReader}, logsFetcher, &melextraction.LogUnpacker{})
	if err != nil {
		return nil, common.Hash{}, err
	}
	var seenBatches []uint64
	for i, batch := range seqBatches {
		if batch.SequenceNumber == seqNum {
			data, err := melextraction.SerializeBatch(ctx, batch, batchTxs[i], logsFetcher)
			return data, batch.BlockHash, err
		}
		seenBatches = append(seenBatches, batch.SequenceNumber)
	}
	return nil, common.Hash{}, fmt.Errorf("sequencer batch %v not found in L1 block %v (found batches %v)", seqNum, metadata.ParentChainBlock, seenBatches)
}

// ReorgTo, when reorgEventsNotifier is set, should only be called after the readers of the channel are started as this is a blocking operation. To be only
// called during init when reorging to a message batch
func (m *MessageExtractor) ReorgTo(parentChainBlockNumber uint64) error {
	dbBatch := m.melDB.db.NewBatch()
	if err := m.melDB.setHeadMelStateBlockNum(dbBatch, parentChainBlockNumber); err != nil {
		return err
	}
	if err := dbBatch.Write(); err != nil {
		return err
	}
	if m.reorgEventsNotifier != nil {
		m.reorgEventsNotifier <- parentChainBlockNumber
	}
	return nil
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
func (m *MessageExtractor) FindInboxBatchContainingMessage(pos arbutil.MessageIndex) (uint64, bool, error) {
	batchCount, err := m.GetBatchCount()
	if err != nil {
		return 0, false, err
	}
	if batchCount == 0 {
		return 0, false, nil
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

func (m *MessageExtractor) SetBlockValidator(_ *staker.BlockValidator) {
}

func (m *MessageExtractor) GetBatchCount() (uint64, error) {
	headState, err := m.melDB.GetHeadMelState()
	if err != nil {
		return 0, err
	}
	return headState.BatchCount, nil
}

func (m *MessageExtractor) GetBatchAcc(_ uint64) (common.Hash, error) {
	return common.Hash{}, errors.New("unimplemented")
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
		fsmBlocksProcessedCounter.Inc(1)
		return m.processNextBlock(ctx, current)
	// `SavingMessages` is the state responsible for saving the extracted messages
	// and delayed messages to the database. It stores data in the node's consensus database
	// and runs after the `ProcessingNextBlock` state.
	// After data is stored, the FSM will then transition to the `ProcessingNextBlock` state
	// yet again.
	case SavingMessages:
		fsmSaveMessagesCounter.Inc(1)
		return m.saveMessages(ctx, current)
	// `Reorging` is the state responsible for handling reorgs in the parent chain.
	// It is triggered when a reorg occurs, and it will revert the MEL state being processed to the
	// specified block. The FSM will transition to the `ProcessingNextBlock` state
	// based on this old state after the reorg is handled.
	case Reorging:
		fsmReorgsCounter.Inc(1)
		return m.reorg(ctx, current)
	default:
		return m.config.RetryInterval, fmt.Errorf("invalid state: %s", current.State)
	}
}
