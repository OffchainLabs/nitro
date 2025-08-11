package gethexec

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"runtime/debug"
	"sync"
	"time"

	protos "github.com/EspressoSystems/timeboost-proto/go-generated"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type TransactionType uint8

const (
	Normal TransactionType = iota
	Delayed
)

type timeboostTransactionQueueItem struct {
	tx                 *types.Transaction
	txSize             int
	options            *arbitrum_types.ConditionalOptions
	roundId            uint64
	consensusTimestamp uint64
	delayedMessageRead uint64
	txType             TransactionType
}

type synchronizedTimeboostTransactionQueue struct {
	queue []timeboostTransactionQueueItem
	mutex sync.RWMutex
}

type DelayedMessageCommand struct {
	DelayedMessagesRead uint64
}

func (q *synchronizedTimeboostTransactionQueue) enqueue(item timeboostTransactionQueueItem) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.queue = append(q.queue, item)
}

func (q *synchronizedTimeboostTransactionQueue) enqueueItems(items []timeboostTransactionQueueItem) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.queue = append(q.queue, items...)
}

func (q *synchronizedTimeboostTransactionQueue) dequeue() timeboostTransactionQueueItem {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	// Remove the first element from the queue and then return it
	item := q.queue[0]
	q.queue = q.queue[1:]
	return item
}

func (q *synchronizedTimeboostTransactionQueue) Len() int {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return len(q.queue)
}

func (q *synchronizedTimeboostTransactionQueue) Peek() *timeboostTransactionQueueItem {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return &q.queue[0]
}

type TimeboostSequencer struct {
	stopwaiter.StopWaiter
	config TimeboostSequencerConfigFetcher
	// TODO: we should read this from the storage
	txQueue    synchronizedTimeboostTransactionQueue
	execEngine *ExecutionEngine
	l1Reader   *headerreader.HeaderReader
	// TODO: We should probably also store the txRetryQueue in storage
	txRetryQueue         synchronizedTimeboostTransactionQueue
	nonceCache           *nonceCache
	timeboostTxnListener *TimeboostBridge
	delayedMessagesRead  uint64
	channel              chan DelayedMessageCommand
}

type TimeboostSequencerConfigFetcher func() *TimeboostSequencerConfig

type TimeboostSequencerConfig struct {
	Enable             bool          `koanf:"enable"`
	BlockRetryDuration time.Duration `koanf:"block-retry-duration"`
	// TODO: - should these be configurable or should it be hardcoded?
	MaxTxDataSize               int                   `koanf:"max-tx-data-size"`
	NonceCacheSize              int                   `koanf:"nonce-cache-size"`
	MaxRevertGasReject          uint64                `koanf:"max-revert-gas-reject"`
	ParentChainFinalizationTime time.Duration         `koanf:"parent-chain-finalization-time"`
	MaxAcceptableTimestampDelta time.Duration         `koanf:"max-acceptable-timestamp-delta"`
	EnableProfiling             bool                  `koanf:"enable-profiling"`
	TimeboostBridgeConfig       TimeboostBridgeConfig `koanf:"timeboost-bridge-config"`
	MetricTimeForBlockCreation  time.Duration         `koanf:"metric-time-for-block-creation"`
}

var DefaultTimeboostSequencerConfig = TimeboostSequencerConfig{
	Enable:                      false,
	BlockRetryDuration:          time.Second * 5,
	MaxTxDataSize:               95000,
	NonceCacheSize:              1024,
	MaxRevertGasReject:          0,
	ParentChainFinalizationTime: 20 * time.Minute,
	MaxAcceptableTimestampDelta: time.Hour,
	EnableProfiling:             false,
	TimeboostBridgeConfig:       DefaultTimeboostBridgeConfig,
	MetricTimeForBlockCreation:  time.Second * 5,
}

func TimeboostSequencerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultTimeboostSequencerConfig.Enable, "enable timeboost sequencer")
	f.Duration(prefix+".block-retry-duration", DefaultTimeboostSequencerConfig.BlockRetryDuration, "retry duration after failing to create a block")
	f.Int(prefix+".max-tx-data-size", DefaultTimeboostSequencerConfig.MaxTxDataSize, "maximum transaction size the sequencer will accept")
	f.Int(prefix+".nonce-cache-size", DefaultTimeboostSequencerConfig.NonceCacheSize, "size of the tx sender nonce cache")
	f.Uint64(prefix+".max-revert-gas-reject", DefaultTimeboostSequencerConfig.MaxRevertGasReject, "maximum gas executed in a revert for the sequencer to reject the transaction instead of posting it (anti-DOS)")
	f.Duration(prefix+".parent-chain-finalization-time", DefaultTimeboostSequencerConfig.ParentChainFinalizationTime, "parent chain finalization time")
	f.Duration(prefix+".max-acceptable-timestamp-delta", DefaultTimeboostSequencerConfig.MaxAcceptableTimestampDelta, "maximum acceptable time difference between the local time and the latest L1 block's timestamp")
	f.Bool(prefix+".enable-profiling", DefaultTimeboostSequencerConfig.EnableProfiling, "enable CPU profiling and tracing")
	f.Duration(prefix+".metric-time-for-block-creation", DefaultTimeboostSequencerConfig.MetricTimeForBlockCreation, "time to measure the time it takes to create a block")
	TimeboostBridgeConfigAddOptions(prefix+".timeboost-bridge-config", f)
}

func NewTimeboostSequencer(execEngine *ExecutionEngine, l1Reader *headerreader.HeaderReader, channel chan DelayedMessageCommand, configFetcher TimeboostSequencerConfigFetcher) (*TimeboostSequencer, error) {
	return &TimeboostSequencer{
		config:     configFetcher,
		execEngine: execEngine,
		l1Reader:   l1Reader,
		nonceCache: newNonceCache(configFetcher().NonceCacheSize),
		timeboostTxnListener: &TimeboostBridge{
			config:     configFetcher().TimeboostBridgeConfig,
			grpcClient: nil,
		},
		delayedMessagesRead: 0,
		channel:             channel,
	}, nil
}

func (s *TimeboostSequencer) handleDelayedMessages(delayedMsgsRead uint64) bool {
	log.Info("sending delayed messages", "read", delayedMsgsRead)
	for {
		s.channel <- DelayedMessageCommand{delayedMsgsRead}
		delayedMsgNum, err := s.execEngine.NextDelayedMessageNumber()
		log.Info("next delayed msg num", "num", delayedMsgNum)
		if err != nil {
			log.Error("failed to get next delayed message", "error", err)
			time.Sleep(50 * time.Millisecond)
			continue
		}
		if delayedMsgNum == delayedMsgsRead {
			s.txQueue.dequeue()
			return true
		} else {
			log.Info("waiting for delayed messages to be sequenced", "read", delayedMsgsRead, "next", delayedMsgNum)
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func (s *TimeboostSequencer) createBlock(ctx context.Context) (returnValue bool) {
	// First we need to create the current list of transactions that we will process
	queueItems := make([]timeboostTransactionQueueItem, 0)
	var totalBlockSize int
	madeBlock := false

	defer func() {
		panicErr := recover()
		if panicErr != nil {
			log.Error("sequencer block creation panicked", "panic", panicErr, "backtrace", string(debug.Stack()))
			// TODO: we should return errors to the user here
			// For now we are logging
			for _, queueItem := range queueItems {
				log.Error("Error processing transaction", "err", sequencerInternalError, "queueItem", queueItem)
			}
		}

		returnValue = true
	}()

	lastBlock := s.execEngine.bc.CurrentBlock()
	config := s.config()

outer:
	for {
		var queueItem timeboostTransactionQueueItem
		//  Transaction retry queue should only
		//  have transactions from a given round id
		if s.txRetryQueue.Len() > 0 {
			queueItem = s.txRetryQueue.dequeue()
		} else if s.txQueue.Len() == 0 {
			// This means we have no transactions in the txRetryQueue and
			// we also dont have any sailfish rounds to process
			break
		} else {
			// Only add transactions from the same round id or if the queue is empty
			tx := s.txQueue.Peek()
			if tx == nil {
				break
			}
			empty := len(queueItems) == 0
			switch tx.txType {
			case Normal:
				if empty {
					queueItem = s.txQueue.dequeue()
				} else if queueItems[len(queueItems)-1].roundId == tx.roundId {
					queueItem = s.txQueue.dequeue()
				} else {
					break outer
				}
			case Delayed:
				// create block with non delayed transactions from same round first
				if !empty {
					break outer
				}
				return s.handleDelayedMessages(tx.delayedMessageRead)
			default:
				log.Info("unexpected tx type, discarding", "type", tx.txType)
				s.txQueue.dequeue()
				continue
			}
		}

		// If context is done, return false
		select {
		case <-ctx.Done():
			return madeBlock
		default:
		}

		if queueItem.txSize > s.config().MaxTxDataSize {
			// This tx is too large
			// Even if its a priority item this should be skipped,
			// TODO: return the error to the user here
			log.Warn("timeboost transaction is too large", "txSize", queueItem.txSize, "maxTxDataSize", s.config().MaxTxDataSize, "hash", queueItem.tx.Hash().Hex())
			continue
		}

		if arbmath.BigLessThan(queueItem.tx.GasFeeCap(), lastBlock.BaseFee) {
			// This tx is too low gas fee
			// TODO: return the error to the user here
			log.Warn("timeboost transaction has too low gas fee", "txSize", queueItem.txSize, "gasFeeCap", queueItem.tx.GasFeeCap(), "baseFee", lastBlock.BaseFee, "hash", queueItem.tx.Hash().Hex())
			continue
		}

		if totalBlockSize+queueItem.txSize > s.config().MaxTxDataSize {
			// This tx would be too large to add to this batch
			log.Info("timeboost transaction is too large, adding to retry queue", "txSize", queueItem.txSize, "maxTxDataSize", s.config().MaxTxDataSize, "hash", queueItem.tx.Hash().Hex())
			s.txRetryQueue.enqueue(queueItem)
			// End the batch here to put this tx in the next one
			break
		}
		totalBlockSize += queueItem.txSize
		queueItems = append(queueItems, queueItem)
	}

	if len(queueItems) == 0 {
		return madeBlock
	}

	s.nonceCache.Resize(config.NonceCacheSize)
	// Nonce cache is updated to indicate a new block creation has started
	s.nonceCache.BeginNewBlock()
	// Check nonces for each transaction in the queue
	queueItems = s.precheckNonces(queueItems)
	txes := make([]*types.Transaction, len(queueItems))
	// Add hooks which include pre tx filter and post tx filter
	hooks := s.makeSequencingHooks()
	hooks.ConditionalOptionsForTx = make([]*arbitrum_types.ConditionalOptions, len(queueItems))
	totalBlockSize = 0
	// Add each queue's item to the txes list and add the total block size
	for i, queueItem := range queueItems {
		txes[i] = queueItem.tx
		totalBlockSize = arbmath.SaturatingAdd(totalBlockSize, queueItem.txSize)
		hooks.ConditionalOptionsForTx[i] = queueItem.options
	}

	// if for some reason the total block size is greater than the max tx data size
	// then we need to add the transactions to the retry queue
	if totalBlockSize > config.MaxTxDataSize {
		for _, queueItem := range queueItems {
			s.txRetryQueue.enqueue(queueItem)
		}
		log.Error(
			"put too many transactions in a block",
			"numTxes", len(queueItems),
			"totalBlockSize", totalBlockSize,
			"maxTxDataSize", config.MaxTxDataSize,
		)
		return madeBlock
	}

	if len(queueItems) == 0 {
		return madeBlock
	}

	// Get the consensus timestamp of the first transaction in the queue
	// It should be the same for all transactions in the queue because
	// each transaction is a part of the same round
	timestamp := queueItems[0].consensusTimestamp
	header, err := s.l1Reader.LatestFinalizedBlockHeader(ctx)
	if err != nil {
		log.Error("failed to get latest finalized block header", "err", err)
		return madeBlock
	}
	// finalized l1 block <= consensus timestamp - parent chain finalization time
	l1Block, err := s.getL1BlockNumber(ctx, header.Number.Int64(), header.Time)
	if err != nil {
		return madeBlock
	}

	l1IncomingMessageHeader := &arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_L2Message,
		Poster:      l1pricing.BatchPosterAddress,
		BlockNumber: l1Block.NumberU64(),
		Timestamp:   timestamp,
		RequestId:   nil,
		L1BaseFee:   nil,
	}

	start := time.Now()
	var block *types.Block
	if config.EnableProfiling {
		block, err = s.execEngine.SequenceTransactionsWithProfiling(l1IncomingMessageHeader, txes, hooks, nil)
	} else {
		block, err = s.execEngine.SequenceTransactions(l1IncomingMessageHeader, txes, hooks, nil)
	}

	// The hooks.TxErrors should match the txes. For case where there is no error, we should have a nil error
	if err == nil && len(hooks.TxErrors) != len(txes) {
		err = fmt.Errorf("unexpected number of error results: %v vs number of txes %v", len(hooks.TxErrors), len(txes))
	}

	if errors.Is(err, execution.ErrRetrySequencer) {
		log.Warn("error sequencing transactions", "err", err)

		for _, queueItem := range queueItems {
			s.txRetryQueue.enqueue(queueItem)
		}
		return madeBlock
	}

	if err != nil {
		if errors.Is(err, context.Canceled) {
			// thread closed. We'll later try to forward these messages.
			for _, queueItem := range queueItems {
				s.txRetryQueue.enqueue(queueItem)
			}
			return madeBlock
		}
		log.Error("error sequencing transactions", "err", err)
		for _, queueItem := range queueItems {
			// TODO: should send the error back to the user
			log.Error("error sequencing transactions", "err", err, "tx", queueItem.tx.Hash())
		}
		return madeBlock
	}

	if block != nil {
		successfulBlocksCounter.Inc(1)
		s.nonceCache.Finalize(block)
		// Add a metric to indicate how long it took to create the block
		elapsed := time.Since(start)
		blockCreationTimer.Update(elapsed)
		if elapsed >= config.MetricTimeForBlockCreation {
			blockNum := block.Number()
			log.Warn("took over 5 seconds to sequence a block", "elapsed", elapsed, "numTxes", len(txes), "success", block != nil, "l2Block", blockNum)
		}
	}

	for i, err := range hooks.TxErrors {
		if err == nil {
			madeBlock = true
		}
		queueItem := queueItems[i]
		if errors.Is(err, core.ErrGasLimitReached) {
			// There's not enough gas left in the block for this tx.
			if madeBlock {
				// There was already an earlier tx in the block; retry in a fresh block.
				s.txRetryQueue.enqueue(queueItem)
				continue
			}
		}
		if errors.Is(err, core.ErrIntrinsicGas) {
			// Strip additional information, as it's incorrect due to L1 data gas.
			err = core.ErrIntrinsicGas
			log.Error("error sequencing transactions", "err", err)
		}
		var nonceError NonceError
		if errors.As(err, &nonceError) && nonceError.txNonce > nonceError.stateNonce {
			log.Error("nonce error", "err", err, "txHash", queueItem.tx.Hash())
			continue
		}
	}

	return madeBlock
}

func (s *TimeboostSequencer) getL1BlockNumber(ctx context.Context, blockNumber int64, consensusTimestamp uint64) (*types.Block, error) {

	block, err := s.l1Reader.Client().BlockByNumber(ctx, big.NewInt(blockNumber))
	if err != nil {
		return nil, err
	}

	// Only return the header if its less than equal to the consensus timestamp - parent chain finalization time
	if block.Time() <= consensusTimestamp-uint64(s.config().ParentChainFinalizationTime.Seconds()) {
		return block, nil
	}
	// Keep going backward only block at a time until we find a block which satisfies the constraint
	return s.getL1BlockNumber(ctx, blockNumber-1, consensusTimestamp)
}

func (s *TimeboostSequencer) makeSequencingHooks() *arbos.SequencingHooks {
	return &arbos.SequencingHooks{
		PreTxFilter:             s.preTxFilter,
		PostTxFilter:            s.postTxFilter,
		DiscardInvalidTxsEarly:  true,
		TxErrors:                []error{},
		ConditionalOptionsForTx: nil,
	}
}

func (s *TimeboostSequencer) preTxFilter(_ *params.ChainConfig, header *types.Header, statedb *state.StateDB, _ *arbosState.ArbosState, tx *types.Transaction, options *arbitrum_types.ConditionalOptions, sender common.Address, l1Info *arbos.L1Info) error {
	if s.nonceCache.Caching() {
		stateNonce := s.nonceCache.Get(header, statedb, sender)
		err := MakeNonceError(sender, tx.Nonce(), stateNonce)
		if err != nil {
			nonceCacheRejectedCounter.Inc(1)
			return err
		}
	}

	if options != nil {
		err := options.Check(l1Info.L1BlockNumber(), header.Time, statedb)
		if err != nil {
			conditionalTxRejectedBySequencerCounter.Inc(1)
			return err
		}
		conditionalTxAcceptedBySequencerCounter.Inc(1)
	}
	return nil
}

func (s *TimeboostSequencer) postTxFilter(header *types.Header, statedb *state.StateDB, _ *arbosState.ArbosState, tx *types.Transaction, sender common.Address, dataGas uint64, result *core.ExecutionResult) error {
	if statedb.IsTxFiltered() {
		return state.ErrArbTxFilter
	}
	if result.Err != nil && result.UsedGas > dataGas && result.UsedGas-dataGas <= s.config().MaxRevertGasReject {
		return arbitrum.NewRevertReason(result)
	}
	newNonce := tx.Nonce() + 1
	s.nonceCache.Update(header, sender, newNonce)
	return nil
}

func (s *TimeboostSequencer) precheckNonces(queueItems []timeboostTransactionQueueItem) []timeboostTransactionQueueItem {
	bc := s.execEngine.bc
	latestHeader := bc.CurrentBlock()
	latestState, err := bc.StateAt(latestHeader.Root)
	if err != nil {
		log.Error("failed to get current state to pre-check nonces", "err", err)
		return queueItems
	}
	nextHeaderNumber := arbmath.BigAdd(latestHeader.Number, common.Big1)
	arbosVersion := types.DeserializeHeaderExtraInformation(latestHeader).ArbOSFormatVersion
	signer := types.MakeSigner(bc.Config(), nextHeaderNumber, latestHeader.Time, arbosVersion)
	outputQueueItems := make([]timeboostTransactionQueueItem, 0, len(queueItems))
	var nextQueueItem *timeboostTransactionQueueItem
	var queueItemsIdx int
	pendingNonces := make(map[common.Address]uint64)
	for {
		var queueItem timeboostTransactionQueueItem
		if nextQueueItem != nil {
			queueItem = *nextQueueItem
			nextQueueItem = nil
		} else if queueItemsIdx < len(queueItems) {
			queueItem = queueItems[queueItemsIdx]
			queueItemsIdx++
		} else {
			break
		}
		tx := queueItem.tx
		sender, err := types.Sender(signer, tx)
		if err != nil {
			// TODO: should send the error back to the user
			log.Warn("failed to get sender", "err", err, "txHash", tx.Hash())
			continue
		}
		stateNonce := s.nonceCache.Get(latestHeader, latestState, sender)
		pendingNonce, pending := pendingNonces[sender]
		if !pending {
			pendingNonce = stateNonce
		}
		txNonce := tx.Nonce()

		if txNonce == pendingNonce {
			// We already found a tx with pendingNonce
			// so now we increase the pendingNonce
			pendingNonces[sender] = txNonce + 1
		} else if txNonce < stateNonce || txNonce > pendingNonce {
			// It's impossible for this tx to succeed so far,
			// because its nonce is lower than the state nonce
			// or higher than the highest tx nonce we've seen.
			err := MakeNonceError(sender, txNonce, stateNonce)
			if errors.Is(err, core.ErrNonceTooHigh) {
				var nonceError NonceError
				if !errors.As(err, &nonceError) {
					log.Warn("unreachable nonce error is not nonceError")
					continue
				}
				// TODO send the error back to the user
				log.Error("failed to process transaction nonce", "err", err, "sender", sender, "txNonce", txNonce, "txHash", tx.Hash())
				continue
			} else if err != nil {
				nonceCacheRejectedCounter.Inc(1)
				log.Warn("failed to process transaction nonce", "err", err, "sender", sender, "txNonce", txNonce, "txHash", tx.Hash())
				continue
			} else {
				log.Warn("unreachable nonce err == nil condition hit in precheckNonces")
			}

		}
		outputQueueItems = append(outputQueueItems, queueItem)
	}

	return outputQueueItems
}

func (s *TimeboostSequencer) ProcessInclusionList(ctx context.Context, inclusionList *protos.InclusionList, options *arbitrum_types.ConditionalOptions) error {
	log.Info("processing inclusion list", "round", inclusionList.Round, "len", len(inclusionList.EncodedTxns), "delayed messages index", inclusionList.DelayedMessagesRead)
	var items []timeboostTransactionQueueItem
	for _, protoTx := range inclusionList.EncodedTxns {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(protoTx.EncodedTxn); err != nil {
			log.Warn("error unmarshalling encoded transaction", "err", err)
			return err
		}
		txQueueItem := timeboostTransactionQueueItem{
			tx:                 &tx,
			txSize:             len(protoTx.EncodedTxn),
			options:            options,
			roundId:            inclusionList.Round,
			consensusTimestamp: inclusionList.ConsensusTimestamp,
			delayedMessageRead: 0,
			txType:             Normal,
		}
		items = append(items, txQueueItem)
	}
	// add delayed messages to the end
	if s.delayedMessagesRead < inclusionList.DelayedMessagesRead {
		read := inclusionList.DelayedMessagesRead + 1
		// We will fetch the transaction when we go to make a block, so just set to nil
		txQueueItem := timeboostTransactionQueueItem{
			tx:                 nil,
			txSize:             0,
			options:            options,
			roundId:            inclusionList.Round,
			consensusTimestamp: inclusionList.ConsensusTimestamp,
			delayedMessageRead: read,
			txType:             Delayed,
		}
		items = append(items, txQueueItem)
	}
	// we need to append all the items at once, otherwise the timers can be off
	// between the different nodes sequencers, where they may start to make the block
	// with only a few of the transactions
	s.txQueue.enqueueItems(items)
	s.delayedMessagesRead = inclusionList.DelayedMessagesRead
	return nil
}

func (s *TimeboostSequencer) Start(ctx context.Context) error {
	s.StopWaiter.Start(ctx, s)
	if s.l1Reader == nil {
		return errors.New("l1Reader is nil")
	}

	if err := s.timeboostTxnListener.Start(ctx, s.ProcessInclusionList); err != nil {
		return err
	}

	err := s.CallIterativelySafe(func(ctx context.Context) time.Duration {
		if s.createBlock(ctx) {
			return 0
		}
		return s.config().BlockRetryDuration
	})
	return err
}

func (s *TimeboostSequencer) StopAndWait() {
	s.StopWaiter.StopAndWait()

	if s.txRetryQueue.Len() == 0 &&
		s.txQueue.Len() == 0 {
		return
	}

	log.Warn("Sequencer has queued items while shutting down",
		"txQueue", s.txQueue.Len(),
		"retryQueue", s.txRetryQueue.Len(),
	)
}
