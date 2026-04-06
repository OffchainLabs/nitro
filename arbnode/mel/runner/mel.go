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
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbnode/mel/extraction"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/bold/containers/fsm"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var stuckFSMIndicatingGauge = metrics.NewRegisteredGauge("arb/mel/stuck", nil) // 1-stuck, 0-not_stuck

// Valid values for the ReadMode config field.
const (
	ReadModeLatest    = "latest"
	ReadModeSafe      = "safe"
	ReadModeFinalized = "finalized"
)

type MessageExtractionConfig struct {
	Enable           bool          `koanf:"enable"`
	RetryInterval    time.Duration `koanf:"retry-interval"`
	BlocksToPrefetch uint64        `koanf:"blocks-to-prefetch"`
	ReadMode         string        `koanf:"read-mode"`
	StallTolerance   uint64        `koanf:"stall-tolerance"`
}

// Validate normalizes and validates the config.
// Note: this method mutates c.ReadMode (lowercases it) in addition to validating.
func (c *MessageExtractionConfig) Validate() error {
	c.ReadMode = strings.ToLower(c.ReadMode)
	if c.ReadMode != ReadModeLatest && c.ReadMode != ReadModeSafe && c.ReadMode != ReadModeFinalized {
		return fmt.Errorf("message extraction read-mode is invalid, want: latest or safe or finalized, got: %s", c.ReadMode)
	}
	return nil
}

var DefaultMessageExtractionConfig = MessageExtractionConfig{
	Enable: false,
	// The retry interval for the message extractor FSM. After each tick of the FSM,
	// the extractor service stop waiter will wait for this duration before trying to act again.
	RetryInterval:    time.Millisecond * 500,
	BlocksToPrefetch: 499, // 499 so that eth_getLogs spans at most 500 blocks (from..from+499 inclusive)
	ReadMode:         ReadModeLatest,
	StallTolerance:   10,
}

var TestMessageExtractionConfig = MessageExtractionConfig{
	Enable:           false,
	RetryInterval:    time.Millisecond * 10,
	BlocksToPrefetch: 499,
	ReadMode:         ReadModeLatest,
	StallTolerance:   10,
}

func MessageExtractionConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultMessageExtractionConfig.Enable, "enable message extraction service")
	f.Duration(prefix+".retry-interval", DefaultMessageExtractionConfig.RetryInterval, "wait time before retrying upon a failure")
	f.Uint64(prefix+".blocks-to-prefetch", DefaultMessageExtractionConfig.BlocksToPrefetch, "the number of blocks to prefetch relevant logs from. Recommend using max allowed range for eth_getLogs rpc query")
	f.String(prefix+".read-mode", ReadModeLatest, "mode to only read latest or safe or finalized L1 blocks. When safe or finalized is used, the node should be configured without feed input/output. Defaults to latest. Valid values: latest, safe, finalized")
	f.Uint64(prefix+".stall-tolerance", DefaultMessageExtractionConfig.StallTolerance, "max times the MEL fsm is allowed to be stuck without logging error")
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

// MessageExtractor reads parent chain blocks one by one and transforms them into
// messages for the execution layer.
type MessageExtractor struct {
	stopwaiter.StopWaiter
	config                      MessageExtractionConfig
	parentChainReader           ParentChainReader
	chainConfig                 *params.ChainConfig
	logsAndHeadersPreFetcher    *logsAndHeadersFetcher
	addrs                       *chaininfo.RollupAddresses
	melDB                       *Database
	msgConsumer                 mel.MessageConsumer
	dataProviders               *daprovider.DAProviderRegistry
	fsm                         *fsm.Fsm[action, FSMState]
	caughtUp                    bool
	caughtUpChan                chan struct{}
	lastBlockToRead             atomic.Uint64
	stuckCount                  uint64
	consecutiveNotFound         uint64
	consecutivePreimageRebuilds int
	reorgEventsNotifier         chan uint64
	seqBatchCounter             SequencerBatchCountFetcher
	l1Reader                    *headerreader.HeaderReader
	lastBlockToReadFailures     uint64

	blockValidator *staker.BlockValidator
	fatalErrChan   chan<- error
}

// NewMessageExtractor returns a new MessageExtractor configured with the given
// parent chain reader, rollup addresses, and data providers.
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
	fatalErrChan chan<- error,
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
		fatalErrChan:        fatalErrChan,
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

// Start begins the message extraction loop. The loop ticks a finite state machine (FSM)
// that processes parent chain blocks, saves data, or handles reorgs. On error, the FSM
// retries the same state after RetryInterval. If errors persist beyond 2x StallTolerance
// and fatalErrChan was provided, a fatal error is sent to stop the node.
func (m *MessageExtractor) Start(ctxIn context.Context) error {
	if m.msgConsumer == nil {
		return errors.New("message consumer not set")
	}
	m.StopWaiter.Start(ctxIn, m)
	runChan := make(chan struct{}, 1)
	if m.config.ReadMode != ReadModeLatest {
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
				m.escalateIfPersistent(ctx, m.stuckCount,
					fmt.Errorf("message extractor stuck for %d consecutive errors (state %s): %w", m.stuckCount, m.fsm.Current().State.String(), err))
			} else {
				stuckFSMIndicatingGauge.Update(0)
			}
			return actAgainInterval
		},
		runChan,
	)
}

// escalateIfPersistent sends a fatal error to shut down the node gracefully
// when the failure count exceeds the escalation threshold (2x StallTolerance).
// The caller is responsible for incrementing the counter before calling.
func (m *MessageExtractor) escalateIfPersistent(ctx context.Context, failures uint64, err error) {
	if m.fatalErrChan != nil && m.config.StallTolerance > 0 && failures > 2*m.config.StallTolerance {
		select {
		case m.fatalErrChan <- err:
		case <-ctx.Done():
		}
	}
}

func (m *MessageExtractor) updateLastBlockToRead(ctx context.Context) time.Duration {
	var header *types.Header
	var err error
	switch m.config.ReadMode {
	case ReadModeSafe:
		header, err = m.parentChainReader.HeaderByNumber(ctx, big.NewInt(rpc.SafeBlockNumber.Int64()))
	case ReadModeFinalized:
		header, err = m.parentChainReader.HeaderByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
	default:
		log.Error("updateLastBlockToRead called with unexpected ReadMode", "mode", m.config.ReadMode)
		return m.config.RetryInterval
	}

	var failReason string
	if err != nil {
		failReason = fmt.Sprintf("fetch error: %v", err)
	} else if header == nil {
		failReason = "nil header"
	} else if header.Number == nil {
		failReason = "nil header.Number"
	}
	if failReason != "" {
		m.lastBlockToReadFailures++
		log.Error("Error updating last block to read in MEL", "reason", failReason, "mode", m.config.ReadMode, "consecutiveFailures", m.lastBlockToReadFailures)
		m.escalateIfPersistent(ctx, m.lastBlockToReadFailures,
			fmt.Errorf("updateLastBlockToRead: %s for %d consecutive attempts (mode=%s)", failReason, m.lastBlockToReadFailures, m.config.ReadMode))
		return m.config.RetryInterval
	}
	m.lastBlockToReadFailures = 0
	m.lastBlockToRead.Store(header.Number.Uint64())
	return m.config.RetryInterval
}

func (m *MessageExtractor) CurrentFSMState() FSMState {
	return m.fsm.Current().State
}

// clampToInitialBlock ensures blockNum is not below the MEL migration boundary.
func (m *MessageExtractor) clampToInitialBlock(blockNum uint64) uint64 {
	if initialBlockNum, ok := m.melDB.InitialBlockNum(); ok && blockNum < initialBlockNum {
		log.Debug("Clamping requested block to MEL migration boundary", "requested", blockNum, "clamped", initialBlockNum)
		return initialBlockNum
	}
	return blockNum
}

// getStateByRPCBlockNum supports only safe and finalized block numbers; returns an error for other values.
func (m *MessageExtractor) getStateByRPCBlockNum(ctx context.Context, blockNum rpc.BlockNumber) (*mel.State, error) {
	if m.l1Reader == nil {
		return nil, errors.New("l1Reader is not configured; cannot resolve safe/finalized block number")
	}
	var resolvedBlockNum uint64
	var err error
	switch blockNum {
	case rpc.SafeBlockNumber:
		resolvedBlockNum, err = m.l1Reader.LatestSafeBlockNr(ctx)
	case rpc.FinalizedBlockNumber:
		resolvedBlockNum, err = m.l1Reader.LatestFinalizedBlockNr(ctx)
	default:
		return nil, fmt.Errorf("getStateByRPCBlockNum requested with unknown blockNum: %v", blockNum)
	}
	if err != nil {
		return nil, fmt.Errorf("getStateByRPCBlockNum: resolving %v block number: %w", blockNum, err)
	}
	headMelStateBlockNum, err := m.melDB.GetHeadMelStateBlockNum()
	if err != nil {
		return nil, err
	}
	rawBlockNum := min(headMelStateBlockNum, resolvedBlockNum)
	stateBlockNum := m.clampToInitialBlock(rawBlockNum)
	if stateBlockNum != rawBlockNum {
		log.Info("getStateByRPCBlockNum clamped to MEL migration boundary", "requested", blockNum, "resolved", rawBlockNum, "clamped", stateBlockNum)
	}
	return m.melDB.StateAtOrBelowHead(stateBlockNum)
}

func (m *MessageExtractor) SetBlockValidator(blockValidator *staker.BlockValidator) error {
	if m.Started() {
		return errors.New("cannot set block validator after start")
	}
	if m.blockValidator != nil {
		return errors.New("block validator already set")
	}
	m.blockValidator = blockValidator
	return nil
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
	batchSeen := headState.BatchCount
	batchSeenIsEstimate := false
	if m.seqBatchCounter != nil {
		seen, err := m.seqBatchCounter.GetBatchCount(ctx, new(big.Int).SetUint64(headState.ParentChainBlockNumber))
		if err != nil {
			if ctx.Err() != nil {
				return mel.MessageSyncProgress{}, ctx.Err()
			}
			// TODO: Replace with a sentinel error check once geth exposes one for "header not found".
			// This error originates from the RPC/header lookup path, distinct from the database-level
			// not-found errors handled by rawdb.IsDbErrNotFound in FinalizedDelayedMessageAtPosition.
			if strings.Contains(err.Error(), "header not found") {
				batchSeenIsEstimate = true
				log.Info("SequencerInbox GetBatchCount header not found, using headState.BatchCount fallback", "parentChainBlock", headState.ParentChainBlockNumber)
			} else {
				return mel.MessageSyncProgress{}, fmt.Errorf("SequencerInbox GetBatchCount error at block %d: %w", headState.ParentChainBlockNumber, err)
			}
		} else {
			batchSeen = seen
		}
	}
	return mel.MessageSyncProgress{
		BatchSeen:           batchSeen,
		BatchSeenIsEstimate: batchSeenIsEstimate,
		BatchProcessed:      headState.BatchCount,
		MsgCount:            arbutil.MessageIndex(headState.MsgCount),
	}, nil
}

func (m *MessageExtractor) GetL1Reader() *headerreader.HeaderReader {
	return m.l1Reader
}

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
	return m.melDB.StateAtOrBelowHead(parentchainBlocknumber)
}

func (m *MessageExtractor) RebuildStateDelayedMsgPreimages(state *mel.State) error {
	return state.RebuildDelayedMsgPreimages(m.melDB.FetchDelayedMessage)
}

func (m *MessageExtractor) GetMsgCount() (arbutil.MessageIndex, error) {
	headState, err := m.melDB.GetHeadMelState()
	if err != nil {
		return 0, err
	}
	return arbutil.MessageIndex(headState.MsgCount), nil
}

func (m *MessageExtractor) GetDelayedMessage(index uint64) (*mel.DelayedInboxMessage, error) {
	headState, err := m.melDB.GetHeadMelState()
	if err != nil {
		return nil, err
	}
	if index >= headState.DelayedMessagesSeen {
		return nil, fmt.Errorf("%w: delayed message index %d >= seen count %d", mel.ErrAccumulatorNotFound, index, headState.DelayedMessagesSeen)
	}
	return m.melDB.FetchDelayedMessage(index)
}

func (m *MessageExtractor) GetDelayedMessageBytes(ctx context.Context, seqNum uint64) ([]byte, error) {
	delayedMsg, err := m.GetDelayedMessage(seqNum)
	if err != nil {
		return nil, err
	}
	if delayedMsg.Message == nil {
		return nil, fmt.Errorf("delayed message %d has nil Message", seqNum)
	}
	return delayedMsg.Message.Serialize()
}

func (m *MessageExtractor) GetDelayedAcc(seqNum uint64) (common.Hash, error) {
	delayedMsg, err := m.GetDelayedMessage(seqNum)
	if err != nil {
		return common.Hash{}, err
	}
	return delayedMsg.AfterInboxAcc()
}

func (m *MessageExtractor) GetDelayedCountAtParentChainBlock(ctx context.Context, parentChainBlockNum uint64) (uint64, error) {
	state, err := m.melDB.StateAtOrBelowHead(m.clampToInitialBlock(parentChainBlockNum))
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

// FindParentChainBlockContainingDelayed is not supported under MEL. The transaction
// streamer handles ErrNotImplementedUnderMEL by falling back to GetSequencerMessageBytes
// (without a specific parent chain block), which resolves the block internally via batch metadata.
func (m *MessageExtractor) FindParentChainBlockContainingDelayed(context.Context, uint64) (uint64, error) {
	return 0, fmt.Errorf("FindParentChainBlockContainingDelayed: %w", mel.ErrNotImplementedUnderMEL)
}

func (m *MessageExtractor) GetBatchMetadata(seqNum uint64) (mel.BatchMetadata, error) {
	headState, err := m.melDB.GetHeadMelState()
	if err != nil {
		return mel.BatchMetadata{}, err
	}
	if seqNum >= headState.BatchCount {
		return mel.BatchMetadata{}, fmt.Errorf("batchMetadata not available for seqNum %d: head MEL state batch count is %d", seqNum, headState.BatchCount)
	}
	batchMetadata, err := m.melDB.fetchBatchMetadata(seqNum)
	if err != nil {
		return mel.BatchMetadata{}, err
	}
	return *batchMetadata, nil
}

func (m *MessageExtractor) SupportsPushingFinalityData() bool {
	return true
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
		// Both db-not-found and "above head" errors mean MEL hasn't processed
		// this block yet, so the message is not yet finalized.
		headBlockNum, headErr := m.melDB.GetHeadMelStateBlockNum()
		if headErr != nil {
			log.Warn("MEL GetHeadMelStateBlockNum failed during finalized delayed message check",
				"parentChainBlock", finalizedBlock, "headErr", headErr, "originalErr", err)
		}
		if rawdb.IsDbErrNotFound(err) {
			log.Debug("MEL delayed count not available for finalized block, treating as not yet finalized", "parentChainBlock", finalizedBlock)
			return nil, common.Hash{}, msg.ParentChainBlockNumber, mel.ErrDelayedMessageNotYetFinalized
		}
		if headErr == nil && finalizedBlock > headBlockNum {
			log.Debug("Finalized block is above MEL head, treating as not yet finalized", "parentChainBlock", finalizedBlock, "headBlock", headBlockNum, "originalErr", err)
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
	acc, err := msg.AfterInboxAcc()
	if err != nil {
		return nil, common.Hash{}, 0, fmt.Errorf("MEL: failed to compute AfterInboxAcc at position %d: %w", requestedPosition, err)
	}
	return msg.Message, acc, msg.ParentChainBlockNumber, nil
}

func (m *MessageExtractor) GetSequencerMessageBytes(ctx context.Context, seqNum uint64) ([]byte, common.Hash, error) {
	metadata, err := m.GetBatchMetadata(seqNum)
	if err != nil {
		return nil, common.Hash{}, err
	}
	return m.GetSequencerMessageBytesForParentBlock(ctx, seqNum, metadata.ParentChainBlock)
}

func (m *MessageExtractor) GetSequencerMessageBytesForParentBlock(ctx context.Context, seqNum uint64, parentChainBlock uint64) ([]byte, common.Hash, error) {
	// blocksToFetch=0: single-block lookup, no range prefetch needed.
	logsFetcher := newLogsAndHeadersFetcher(m.parentChainReader, 0)
	if err := logsFetcher.fetchSequencerBatchLogs(ctx, parentChainBlock, parentChainBlock); err != nil {
		return nil, common.Hash{}, err
	}
	parentChainHeader, err := m.parentChainReader.HeaderByNumber(ctx, new(big.Int).SetUint64(parentChainBlock))
	if err != nil {
		return nil, common.Hash{}, err
	}
	if parentChainHeader == nil {
		return nil, common.Hash{}, fmt.Errorf("parent chain block %d not found", parentChainBlock)
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
	return nil, common.Hash{}, fmt.Errorf("sequencer batch %v not found in L1 block %v (found batches %v)", seqNum, parentChainBlock, seenBatches)
}

// sendReorgNotification sends a reorg notification on the reorgEventsNotifier channel.
// Returns nil immediately if the notifier is not set.
func (m *MessageExtractor) sendReorgNotification(ctx context.Context, blockNum uint64) error {
	if m.reorgEventsNotifier == nil {
		return nil
	}
	select {
	case m.reorgEventsNotifier <- blockNum:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ReorgTo rewrites the head MEL state block number and notifies reorg listeners.
// When called before Start() (e.g. during node init), the notification is skipped
// because the channel consumer hasn't started yet. Downstream consumers (block
// validator, batch poster) must not be started before MEL, so they will initialize
// from the current (rewound) head state when they start.
func (m *MessageExtractor) ReorgTo(parentChainBlockNumber uint64) error {
	if err := m.melDB.RewriteHeadBlockNum(parentChainBlockNumber); err != nil {
		return err
	}
	if m.reorgEventsNotifier == nil {
		return nil
	}
	if !m.Started() {
		log.Info("ReorgTo applied during init (MEL not running); downstream consumers will start from rewound state", "block", parentChainBlockNumber)
		return nil
	}
	ctx, err := m.GetContextSafe()
	if err != nil {
		return err
	}
	return m.sendReorgNotification(ctx, parentChainBlockNumber)
}

func (m *MessageExtractor) GetBatchAcc(seqNum uint64) (common.Hash, error) {
	batchMetadata, err := m.GetBatchMetadata(seqNum)
	if err != nil {
		return common.Hash{}, err
	}
	return batchMetadata.Accumulator, nil
}

func (m *MessageExtractor) GetBatchMessageCount(seqNum uint64) (arbutil.MessageIndex, error) {
	metadata, err := m.GetBatchMetadata(seqNum)
	if err != nil {
		return 0, err
	}
	return metadata.MessageCount, nil
}

func (m *MessageExtractor) GetBatchParentChainBlock(seqNum uint64) (uint64, error) {
	metadata, err := m.GetBatchMetadata(seqNum)
	if err != nil {
		return 0, err
	}
	return metadata.ParentChainBlock, nil
}

func (m *MessageExtractor) FindInboxBatchContainingMessage(pos arbutil.MessageIndex) (uint64, bool, error) {
	return arbutil.FindInboxBatchContainingMessage(m, pos)
}

func (m *MessageExtractor) GetBatchCount() (uint64, error) {
	headState, err := m.melDB.GetHeadMelState()
	if err != nil {
		return 0, err
	}
	return headState.BatchCount, nil
}

func (m *MessageExtractor) LegacyDelayedBound() uint64 {
	return m.melDB.LegacyDelayedCount()
}

func (m *MessageExtractor) CaughtUp() chan struct{} {
	return m.caughtUpChan
}

// Act ticks the message extractor FSM and performs the action associated with the current state,
// such as processing the next block, saving messages, or handling reorgs.
func (m *MessageExtractor) Act(ctx context.Context) (time.Duration, error) {
	current := m.fsm.Current()
	switch current.State {
	// `Start` is the initial state of the FSM. It is responsible for
	// initializing the message extraction process. The FSM will transition to
	// `ProcessingNextBlock` after successfully loading and validating the initial
	// MEL state, or to `Reorging` if a parent chain reorg is detected at the
	// stored head block.
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
	// and delayed messages. It first pushes extracted messages to the transaction
	// streamer, then atomically writes batch metadata, delayed messages, and the
	// new head MEL state to the consensus database.
	// The FSM transitions to `ProcessingNextBlock` after both writes succeed.
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
