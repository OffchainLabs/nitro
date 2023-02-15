package execution

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"testing"
	"time"

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
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/sharedmetrics"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/pkg/errors"
)

type TransactionStreamerInterface interface {
	WriteMessageFromSequencer(pos arbutil.MessageIndex, msgWithMeta arbostypes.MessageWithMetadata) error
	ExpectChosenSequencer() error
	FetchBatch(batchNum uint64) ([]byte, error)
}

type ExecutionEngine struct {
	stopwaiter.StopWaiter

	bc       *core.BlockChain
	streamer TransactionStreamerInterface
	recorder *BlockRecorder

	resequenceChan    chan []*arbostypes.MessageWithMetadata
	createBlocksMutex sync.Mutex

	newBlockNotifier chan struct{}
	latestBlockMutex sync.Mutex
	latestBlock      *types.Block

	nextScheduledVersionCheck time.Time // protected by the createBlocksMutex

	reorgSequencing func() *arbos.SequencingHooks
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
	if s.reorgSequencing != nil {
		panic("trying to set recorder policy when already set")
	}
	s.recorder = recorder
}

func (s *ExecutionEngine) SetReorgSequencingPolicy(reorgSequencing func() *arbos.SequencingHooks) {
	if s.Started() {
		panic("trying to set reorg sequencing policy after start")
	}
	if s.reorgSequencing != nil {
		panic("trying to set reorg sequencing policy when already set")
	}
	s.reorgSequencing = reorgSequencing
}

func (s *ExecutionEngine) SetTransactionStreamer(streamer TransactionStreamerInterface) {
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
	successful := false
	defer func() {
		if !successful {
			s.createBlocksMutex.Unlock()
		}
	}()
	blockNum := s.MessageIndexToBlockNumber(count - 1)
	// We can safely cast blockNum to a uint64 as we checked count == 0 above
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
		err := s.digestMessageWithBlockMutex(count+arbutil.MessageIndex(i), &newMessages[i])
		if err != nil {
			return err
		}
	}
	if s.recorder != nil {
		s.recorder.ReorgTo(targetBlock.Header())
	}
	s.resequenceChan <- oldMessages
	successful = true
	return nil
}

func (s *ExecutionEngine) HeadMessageNumber() (arbutil.MessageIndex, error) {
	return s.BlockNumberToMessageIndex(s.bc.CurrentBlock().Header().Number.Uint64())
}

func (s *ExecutionEngine) HeadMessageNumberSync(t *testing.T) (arbutil.MessageIndex, error) {
	s.createBlocksMutex.Lock()
	defer s.createBlocksMutex.Unlock()
	return s.HeadMessageNumber()
}

func (s *ExecutionEngine) NextDelayedMessageNumber() (uint64, error) {
	return s.bc.CurrentHeader().Nonce.Uint64(), nil
}

// The caller must hold the createBlocksMutex
func (s *ExecutionEngine) resequenceReorgedMessages(messages []*arbostypes.MessageWithMetadata) {
	if s.reorgSequencing == nil {
		return
	}

	log.Info("Trying to resequence messages", "number", len(messages))
	lastBlockHeader := s.bc.CurrentBlock().Header()
	if lastBlockHeader == nil {
		log.Error("block header not found during resequence")
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
			err := s.sequenceDelayedMessageWithBlockMutex(msg.Message, delayedSeqNum)
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
		_, err = s.sequenceTransactionsWithBlockMutex(msg.Message.Header, txes, s.reorgSequencing())
		if err != nil {
			log.Error("failed to re-sequence old user message removed by reorg", "err", err)
			return
		}
	}
}

var ErrSequencerInsertLockTaken = errors.New("insert lock taken")

func (s *ExecutionEngine) SequenceTransactions(header *arbostypes.L1IncomingMessageHeader, txes types.Transactions, hooks *arbos.SequencingHooks) (*types.Block, error) {
	for {
		hooks.TxErrors = nil
		s.createBlocksMutex.Lock()
		block, err := s.sequenceTransactionsWithBlockMutex(header, txes, hooks)
		s.createBlocksMutex.Unlock()
		if !errors.Is(err, ErrSequencerInsertLockTaken) {
			return block, err
		}
		<-time.After(time.Millisecond * 100)
	}
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

func (s *ExecutionEngine) sequenceTransactionsWithBlockMutex(header *arbostypes.L1IncomingMessageHeader, txes types.Transactions, hooks *arbos.SequencingHooks) (*types.Block, error) {
	lastBlockHeader := s.bc.CurrentBlock().Header()
	if lastBlockHeader == nil {
		return nil, errors.New("current block header not found")
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

func (s *ExecutionEngine) SequenceDelayedMessage(message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) error {
	for {
		s.createBlocksMutex.Lock()
		err := s.sequenceDelayedMessageWithBlockMutex(message, delayedSeqNum)
		s.createBlocksMutex.Unlock()
		if !errors.Is(err, ErrSequencerInsertLockTaken) {
			return err
		}
		<-time.After(time.Millisecond * 100)
	}
}

func (s *ExecutionEngine) sequenceDelayedMessageWithBlockMutex(message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) error {
	currentHeader := s.bc.CurrentBlock().Header()

	expectedDelayed := currentHeader.Nonce.Uint64()

	lastMsg, err := s.BlockNumberToMessageIndex(currentHeader.Number.Uint64())
	if err != nil {
		return err
	}

	if expectedDelayed != delayedSeqNum {
		return fmt.Errorf("wrong delayed message sequenced got %d expected %d", delayedSeqNum, expectedDelayed)
	}

	messageWithMeta := arbostypes.MessageWithMetadata{
		Message:             message,
		DelayedMessagesRead: delayedSeqNum + 1,
	}

	err = s.streamer.WriteMessageFromSequencer(lastMsg+1, messageWithMeta)
	if err != nil {
		return err
	}

	startTime := time.Now()
	block, statedb, receipts, err := s.createBlockFromNextMessage(&messageWithMeta)
	if err != nil {
		return err
	}

	err = s.appendBlock(block, statedb, receipts, time.Since(startTime))
	if err != nil {
		return err
	}

	log.Info("ExecutionEngine: Added DelayedMessages", "pos", lastMsg+1, "delayed", delayedSeqNum, "block-header", block.Header())

	return nil
}

// must hold createBlockMutex
func (s *ExecutionEngine) createBlockFromNextMessage(msg *arbostypes.MessageWithMetadata) (*types.Block, *state.StateDB, types.Receipts, error) {
	currentHeader := s.bc.CurrentBlock().Header()
	if currentHeader == nil {
		return nil, nil, nil, errors.New("failed to get current header")
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

type MessageResult struct {
	BlockHash common.Hash
	SendRoot  common.Hash
}

func (s *ExecutionEngine) resultFromHeader(header *types.Header) (*MessageResult, error) {
	if header == nil {
		return nil, fmt.Errorf("result not found")
	}
	info, err := types.DeserializeHeaderExtraInformation(header)
	if err != nil {
		return nil, err
	}
	return &MessageResult{header.Hash(), info.SendRoot}, nil
}

func (s *ExecutionEngine) ResultAtPos(pos arbutil.MessageIndex) (*MessageResult, error) {
	return s.resultFromHeader(s.bc.GetHeaderByNumber(s.MessageIndexToBlockNumber(pos)))
}

func (s *ExecutionEngine) DigestMessage(num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata) error {
	if !s.createBlocksMutex.TryLock() {
		return errors.New("mutex held")
	}
	defer s.createBlocksMutex.Unlock()
	return s.digestMessageWithBlockMutex(num, msg)
}

func (s *ExecutionEngine) digestMessageWithBlockMutex(num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata) error {
	currentHeader := s.bc.CurrentHeader()
	curMsg, err := s.BlockNumberToMessageIndex(currentHeader.Number.Uint64())
	if err != nil {
		return err
	}
	if curMsg+1 != num {
		return fmt.Errorf("wrong message number in digest got %d expected %d", num, curMsg+1)
	}

	startTime := time.Now()
	block, statedb, receipts, err := s.createBlockFromNextMessage(msg)
	if err != nil {
		return err
	}

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
			if block != lastBlock && block != nil {
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
