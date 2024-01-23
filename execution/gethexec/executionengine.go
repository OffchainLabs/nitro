package gethexec

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/sharedmetrics"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type ExecutionEngine struct {
	stopwaiter.StopWaiter

	bc       *core.BlockChain
	streamer execution.TransactionStreamer
	recorder *BlockRecorder

	resequenceChan    chan []*arbostypes.MessageWithMetadata
	createBlocksMutex sync.Mutex

	newBlockNotifier chan struct{}
	latestBlockMutex sync.Mutex
	latestBlock      *types.Block

	nextScheduledVersionCheck time.Time // protected by the createBlocksMutex

	reorgSequencing bool

	prefetchBlock bool
}

func NewExecutionEngine(bc *core.BlockChain) (*ExecutionEngine, error) {
	return &ExecutionEngine{
		bc:               bc,
		resequenceChan:   make(chan []*arbostypes.MessageWithMetadata),
		newBlockNotifier: make(chan struct{}, 1),
	}, nil
}

func (s *ExecutionEngine) SetRecorder(recorder *BlockRecorder) {
	if s.Started() {
		panic("trying to set recorder after start")
	}
	if s.recorder != nil {
		panic("trying to set recorder policy when already set")
	}
	s.recorder = recorder
}

func (s *ExecutionEngine) EnableReorgSequencing() {
	if s.Started() {
		panic("trying to enable reorg sequencing after start")
	}
	if s.reorgSequencing {
		panic("trying to enable reorg sequencing when already set")
	}
	s.reorgSequencing = true
}

func (s *ExecutionEngine) EnablePrefetchBlock() {
	if s.Started() {
		panic("trying to enable prefetch block after start")
	}
	if s.prefetchBlock {
		panic("trying to enable prefetch block when already set")
	}
	s.prefetchBlock = true
}

func (s *ExecutionEngine) SetTransactionStreamer(streamer execution.TransactionStreamer) {
	if s.Started() {
		panic("trying to set transaction streamer after start")
	}
	if s.streamer != nil {
		panic("trying to set transaction streamer when already set")
	}
	s.streamer = streamer
}

func (s *ExecutionEngine) Reorg(count arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadata, oldMessages []*arbostypes.MessageWithMetadata) error {
	if count == 0 {
		return errors.New("cannot reorg out genesis")
	}
	s.createBlocksMutex.Lock()
	resequencing := false
	defer func() {
		// if we are resequencing old messages - don't release the lock
		// lock will be relesed by thread listening to resequenceChan
		if !resequencing {
			s.createBlocksMutex.Unlock()
		}
	}()
	blockNum := s.MessageIndexToBlockNumber(count - 1)
	// We can safely cast blockNum to a uint64 as it comes from MessageCountToBlockNumber
	targetBlock := s.bc.GetBlockByNumber(uint64(blockNum))
	if targetBlock == nil {
		log.Warn("reorg target block not found", "block", blockNum)
		return nil
	}

	err := s.bc.ReorgToOldBlock(targetBlock)
	if err != nil {
		return err
	}
	for i := range newMessages {
		var msgForPrefetch *arbostypes.MessageWithMetadata
		if i < len(newMessages)-1 {
			msgForPrefetch = &newMessages[i]
		}
		err := s.digestMessageWithBlockMutex(count+arbutil.MessageIndex(i), &newMessages[i], msgForPrefetch)
		if err != nil {
			return err
		}
	}
	if s.recorder != nil {
		s.recorder.ReorgTo(targetBlock.Header())
	}
	if len(oldMessages) > 0 {
		s.resequenceChan <- oldMessages
		resequencing = true
	}
	return nil
}

func (s *ExecutionEngine) getCurrentHeader() (*types.Header, error) {
	currentBlock := s.bc.CurrentBlock()
	if currentBlock == nil {
		return nil, errors.New("failed to get current block")
	}
	return currentBlock, nil
}

func (s *ExecutionEngine) HeadMessageNumber() (arbutil.MessageIndex, error) {
	currentHeader, err := s.getCurrentHeader()
	if err != nil {
		return 0, err
	}
	return s.BlockNumberToMessageIndex(currentHeader.Number.Uint64())
}

func (s *ExecutionEngine) HeadMessageNumberSync(t *testing.T) (arbutil.MessageIndex, error) {
	s.createBlocksMutex.Lock()
	defer s.createBlocksMutex.Unlock()
	return s.HeadMessageNumber()
}

func (s *ExecutionEngine) NextDelayedMessageNumber() (uint64, error) {
	currentHeader, err := s.getCurrentHeader()
	if err != nil {
		return 0, err
	}
	return currentHeader.Nonce.Uint64(), nil
}

func messageFromTxes(header *arbostypes.L1IncomingMessageHeader, txes types.Transactions, txErrors []error) (*arbostypes.L1IncomingMessage, error) {
	var l2Message []byte
	if len(txes) == 1 && txErrors[0] == nil {
		txBytes, err := txes[0].MarshalBinary()
		if err != nil {
			return nil, err
		}
		l2Message = append(l2Message, arbos.L2MessageKind_SignedTx)
		l2Message = append(l2Message, txBytes...)
	} else {
		l2Message = append(l2Message, arbos.L2MessageKind_Batch)
		sizeBuf := make([]byte, 8)
		for i, tx := range txes {
			if txErrors[i] != nil {
				continue
			}
			txBytes, err := tx.MarshalBinary()
			if err != nil {
				return nil, err
			}
			binary.BigEndian.PutUint64(sizeBuf, uint64(len(txBytes)+1))
			l2Message = append(l2Message, sizeBuf...)
			l2Message = append(l2Message, arbos.L2MessageKind_SignedTx)
			l2Message = append(l2Message, txBytes...)
		}
	}
	return &arbostypes.L1IncomingMessage{
		Header: header,
		L2msg:  l2Message,
	}, nil
}

// The caller must hold the createBlocksMutex
func (s *ExecutionEngine) resequenceReorgedMessages(messages []*arbostypes.MessageWithMetadata) {
	if !s.reorgSequencing {
		return
	}

	log.Info("Trying to resequence messages", "number", len(messages))
	lastBlockHeader, err := s.getCurrentHeader()
	if err != nil {
		log.Error("block header not found during resequence", "err", err)
		return
	}

	nextDelayedSeqNum := lastBlockHeader.Nonce.Uint64()

	for _, msg := range messages {
		// Check if the message is non-nil just to be safe
		if msg == nil || msg.Message == nil || msg.Message.Header == nil {
			continue
		}
		header := msg.Message.Header
		if header.RequestId != nil {
			delayedSeqNum := header.RequestId.Big().Uint64()
			if delayedSeqNum != nextDelayedSeqNum {
				log.Info("not resequencing delayed message due to unexpected index", "expected", nextDelayedSeqNum, "found", delayedSeqNum)
				continue
			}
			_, err := s.sequenceDelayedMessageWithBlockMutex(msg.Message, delayedSeqNum)
			if err != nil {
				log.Error("failed to re-sequence old delayed message removed by reorg", "err", err)
			}
			nextDelayedSeqNum += 1
			continue
		}
		if header.Kind != arbostypes.L1MessageType_L2Message || header.Poster != l1pricing.BatchPosterAddress {
			// This shouldn't exist?
			log.Warn("skipping non-standard sequencer message found from reorg", "header", header)
			continue
		}
		// We don't need a batch fetcher as this is an L2 message
		txes, err := arbos.ParseL2Transactions(msg.Message, s.bc.Config().ChainID, nil)
		if err != nil {
			log.Warn("failed to parse sequencer message found from reorg", "err", err)
			continue
		}
		hooks := arbos.NoopSequencingHooks()
		hooks.DiscardInvalidTxsEarly = true
		_, err = s.sequenceTransactionsWithBlockMutex(msg.Message.Header, txes, hooks)
		if err != nil {
			log.Error("failed to re-sequence old user message removed by reorg", "err", err)
			return
		}
	}
}

func (s *ExecutionEngine) sequencerWrapper(sequencerFunc func() (*types.Block, error)) (*types.Block, error) {
	attempts := 0
	for {
		s.createBlocksMutex.Lock()
		block, err := sequencerFunc()
		s.createBlocksMutex.Unlock()
		if !errors.Is(err, execution.ErrSequencerInsertLockTaken) {
			return block, err
		}
		// We got SequencerInsertLockTaken
		// option 1: there was a race, we are no longer main sequencer
		chosenErr := s.streamer.ExpectChosenSequencer()
		if chosenErr != nil {
			return nil, chosenErr
		}
		// option 2: we are in a test without very orderly sequencer coordination
		if !s.bc.Config().ArbitrumChainParams.AllowDebugPrecompiles {
			// option 3: something weird. send warning
			log.Warn("sequence transactions: insert lock takent", "attempts", attempts)
		}
		// options 2/3 fail after too many attempts
		attempts++
		if attempts > 20 {
			return nil, err
		}
		<-time.After(time.Millisecond * 100)
	}
}

func (s *ExecutionEngine) SequenceTransactions(header *arbostypes.L1IncomingMessageHeader, txes types.Transactions, hooks *arbos.SequencingHooks) (*types.Block, error) {
	return s.sequencerWrapper(func() (*types.Block, error) {
		hooks.TxErrors = nil
		return s.sequenceTransactionsWithBlockMutex(header, txes, hooks)
	})
}

func (s *ExecutionEngine) sequenceTransactionsWithBlockMutex(header *arbostypes.L1IncomingMessageHeader, txes types.Transactions, hooks *arbos.SequencingHooks) (*types.Block, error) {
	lastBlockHeader, err := s.getCurrentHeader()
	if err != nil {
		return nil, err
	}

	statedb, err := s.bc.StateAt(lastBlockHeader.Root)
	if err != nil {
		return nil, err
	}

	delayedMessagesRead := lastBlockHeader.Nonce.Uint64()

	startTime := time.Now()
	block, receipts, err := arbos.ProduceBlockAdvanced(
		header,
		txes,
		delayedMessagesRead,
		lastBlockHeader,
		statedb,
		s.bc,
		s.bc.Config(),
		hooks,
	)
	if err != nil {
		return nil, err
	}
	blockCalcTime := time.Since(startTime)
	if len(hooks.TxErrors) != len(txes) {
		return nil, fmt.Errorf("unexpected number of error results: %v vs number of txes %v", len(hooks.TxErrors), len(txes))
	}

	if len(receipts) == 0 {
		return nil, nil
	}

	allTxsErrored := true
	for _, err := range hooks.TxErrors {
		if err == nil {
			allTxsErrored = false
			break
		}
	}
	if allTxsErrored {
		return nil, nil
	}

	msg, err := messageFromTxes(header, txes, hooks.TxErrors)
	if err != nil {
		return nil, err
	}

	msgWithMeta := arbostypes.MessageWithMetadata{
		Message:             msg,
		DelayedMessagesRead: delayedMessagesRead,
	}

	pos, err := s.BlockNumberToMessageIndex(lastBlockHeader.Number.Uint64() + 1)
	if err != nil {
		return nil, err
	}

	err = s.streamer.WriteMessageFromSequencer(pos, msgWithMeta)
	if err != nil {
		return nil, err
	}

	// Only write the block after we've written the messages, so if the node dies in the middle of this,
	// it will naturally recover on startup by regenerating the missing block.
	err = s.appendBlock(block, statedb, receipts, blockCalcTime)
	if err != nil {
		return nil, err
	}

	return block, nil
}

func (s *ExecutionEngine) SequenceDelayedMessage(message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) error {
	_, err := s.sequencerWrapper(func() (*types.Block, error) {
		return s.sequenceDelayedMessageWithBlockMutex(message, delayedSeqNum)
	})
	return err
}

func (s *ExecutionEngine) sequenceDelayedMessageWithBlockMutex(message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) (*types.Block, error) {
	currentHeader, err := s.getCurrentHeader()
	if err != nil {
		return nil, err
	}

	expectedDelayed := currentHeader.Nonce.Uint64()

	lastMsg, err := s.BlockNumberToMessageIndex(currentHeader.Number.Uint64())
	if err != nil {
		return nil, err
	}

	if expectedDelayed != delayedSeqNum {
		return nil, fmt.Errorf("wrong delayed message sequenced got %d expected %d", delayedSeqNum, expectedDelayed)
	}

	messageWithMeta := arbostypes.MessageWithMetadata{
		Message:             message,
		DelayedMessagesRead: delayedSeqNum + 1,
	}

	err = s.streamer.WriteMessageFromSequencer(lastMsg+1, messageWithMeta)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	block, statedb, receipts, err := s.createBlockFromNextMessage(&messageWithMeta)
	if err != nil {
		return nil, err
	}

	err = s.appendBlock(block, statedb, receipts, time.Since(startTime))
	if err != nil {
		return nil, err
	}

	log.Info("ExecutionEngine: Added DelayedMessages", "pos", lastMsg+1, "delayed", delayedSeqNum, "block-header", block.Header())

	return block, nil
}

func (s *ExecutionEngine) GetGenesisBlockNumber() uint64 {
	return s.bc.Config().ArbitrumChainParams.GenesisBlockNum
}

func (s *ExecutionEngine) BlockNumberToMessageIndex(blockNum uint64) (arbutil.MessageIndex, error) {
	genesis := s.GetGenesisBlockNumber()
	if blockNum < genesis {
		return 0, fmt.Errorf("blockNum %d < genesis %d", blockNum, genesis)
	}
	return arbutil.MessageIndex(blockNum - genesis), nil
}

func (s *ExecutionEngine) MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) uint64 {
	return uint64(messageNum) + s.GetGenesisBlockNumber()
}

// must hold createBlockMutex
func (s *ExecutionEngine) createBlockFromNextMessage(msg *arbostypes.MessageWithMetadata) (*types.Block, *state.StateDB, types.Receipts, error) {
	currentHeader := s.bc.CurrentBlock()
	if currentHeader == nil {
		return nil, nil, nil, errors.New("failed to get current block header")
	}

	currentBlock := s.bc.GetBlock(currentHeader.Hash(), currentHeader.Number.Uint64())
	if currentBlock == nil {
		return nil, nil, nil, errors.New("can't find block for current header")
	}

	err := s.bc.RecoverState(currentBlock)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to recover block %v state: %w", currentBlock.Number(), err)
	}

	statedb, err := s.bc.StateAt(currentHeader.Root)
	if err != nil {
		return nil, nil, nil, err
	}
	statedb.StartPrefetcher("TransactionStreamer")
	defer statedb.StopPrefetcher()

	block, receipts, err := arbos.ProduceBlock(
		msg.Message,
		msg.DelayedMessagesRead,
		currentHeader,
		statedb,
		s.bc,
		s.bc.Config(),
		s.streamer.FetchBatch,
	)

	return block, statedb, receipts, err
}

// must hold createBlockMutex
func (s *ExecutionEngine) appendBlock(block *types.Block, statedb *state.StateDB, receipts types.Receipts, duration time.Duration) error {
	var logs []*types.Log
	for _, receipt := range receipts {
		logs = append(logs, receipt.Logs...)
	}
	status, err := s.bc.WriteBlockAndSetHeadWithTime(block, receipts, logs, statedb, true, duration)
	if err != nil {
		return err
	}
	if status == core.SideStatTy {
		return errors.New("geth rejected block as non-canonical")
	}
	return nil
}

func (s *ExecutionEngine) resultFromHeader(header *types.Header) (*execution.MessageResult, error) {
	if header == nil {
		return nil, fmt.Errorf("result not found")
	}
	info := types.DeserializeHeaderExtraInformation(header)
	return &execution.MessageResult{
		BlockHash: header.Hash(),
		SendRoot:  info.SendRoot,
	}, nil
}

func (s *ExecutionEngine) ResultAtPos(pos arbutil.MessageIndex) (*execution.MessageResult, error) {
	return s.resultFromHeader(s.bc.GetHeaderByNumber(s.MessageIndexToBlockNumber(pos)))
}

// DigestMessage is used to create a block by executing msg against the latest state and storing it.
// Also, while creating a block by executing msg against the latest state,
// in parallel, creates a block by executing msgForPrefetch (msg+1) against the latest state
// but does not store the block.
// This helps in filling the cache, so that the next block creation is faster.
func (s *ExecutionEngine) DigestMessage(num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) error {
	if !s.createBlocksMutex.TryLock() {
		return errors.New("createBlock mutex held")
	}
	defer s.createBlocksMutex.Unlock()
	return s.digestMessageWithBlockMutex(num, msg, msgForPrefetch)
}

func (s *ExecutionEngine) digestMessageWithBlockMutex(num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) error {
	currentHeader, err := s.getCurrentHeader()
	if err != nil {
		return err
	}
	curMsg, err := s.BlockNumberToMessageIndex(currentHeader.Number.Uint64())
	if err != nil {
		return err
	}
	if curMsg+1 != num {
		return fmt.Errorf("wrong message number in digest got %d expected %d", num, curMsg+1)
	}

	startTime := time.Now()
	var wg sync.WaitGroup
	if s.prefetchBlock && msgForPrefetch != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _, err := s.createBlockFromNextMessage(msgForPrefetch)
			if err != nil {
				return
			}
		}()
	}

	block, statedb, receipts, err := s.createBlockFromNextMessage(msg)
	if err != nil {
		return err
	}
	wg.Wait()
	err = s.appendBlock(block, statedb, receipts, time.Since(startTime))
	if err != nil {
		return err
	}

	if time.Now().After(s.nextScheduledVersionCheck) {
		s.nextScheduledVersionCheck = time.Now().Add(time.Minute)
		arbState, err := arbosState.OpenSystemArbosState(statedb, nil, true)
		if err != nil {
			return err
		}
		version, timestampInt, err := arbState.GetScheduledUpgrade()
		if err != nil {
			return err
		}
		var timeUntilUpgrade time.Duration
		var timestamp time.Time
		if timestampInt == 0 {
			// This upgrade will take effect in the next block
			timestamp = time.Now()
		} else {
			// This upgrade is scheduled for the future
			timestamp = time.Unix(int64(timestampInt), 0)
			timeUntilUpgrade = time.Until(timestamp)
		}
		maxSupportedVersion := params.ArbitrumDevTestChainConfig().ArbitrumChainParams.InitialArbOSVersion
		logLevel := log.Warn
		if timeUntilUpgrade < time.Hour*24 {
			logLevel = log.Error
		}
		if version > maxSupportedVersion {
			logLevel(
				"you need to update your node to the latest version before this scheduled ArbOS upgrade",
				"timeUntilUpgrade", timeUntilUpgrade,
				"upgradeScheduledFor", timestamp,
				"maxSupportedArbosVersion", maxSupportedVersion,
				"pendingArbosUpgradeVersion", version,
			)
		}
	}

	sharedmetrics.UpdateSequenceNumberInBlockGauge(num)
	s.latestBlockMutex.Lock()
	s.latestBlock = block
	s.latestBlockMutex.Unlock()
	select {
	case s.newBlockNotifier <- struct{}{}:
	default:
	}
	return nil
}

func (s *ExecutionEngine) Start(ctx_in context.Context) {
	s.StopWaiter.Start(ctx_in, s)
	s.LaunchThread(func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case resequence := <-s.resequenceChan:
				s.resequenceReorgedMessages(resequence)
				s.createBlocksMutex.Unlock()
			}
		}
	})
	s.LaunchThread(func(ctx context.Context) {
		var lastBlock *types.Block
		for {
			select {
			case <-s.newBlockNotifier:
			case <-ctx.Done():
				return
			}
			s.latestBlockMutex.Lock()
			block := s.latestBlock
			s.latestBlockMutex.Unlock()
			if block != nil && (lastBlock == nil || block.Hash() != lastBlock.Hash()) {
				log.Info(
					"created block",
					"l2Block", block.Number(),
					"l2BlockHash", block.Hash(),
				)
				lastBlock = block
				select {
				case <-time.After(time.Second):
				case <-ctx.Done():
					return
				}
			}
		}
	})
}
