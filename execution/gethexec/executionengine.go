// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build !wasm
// +build !wasm

package gethexec

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
#cgo LDFLAGS: ${SRCDIR}/../../target/lib/libstylus.a -ldl -lm
#include "arbitrator.h"
*/
import "C"
import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path"
	"runtime/pprof"
	"runtime/trace"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"
	"github.com/google/uuid"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbos/programs"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/sharedmetrics"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	baseFeeGauge               = metrics.NewRegisteredGauge("arb/block/basefee", nil)
	blockGasUsedHistogram      = metrics.NewRegisteredHistogram("arb/block/gasused", nil, metrics.NewBoundedHistogramSample())
	txCountHistogram           = metrics.NewRegisteredHistogram("arb/block/transactions/count", nil, metrics.NewBoundedHistogramSample())
	txGasUsedHistogram         = metrics.NewRegisteredHistogram("arb/block/transactions/gasused", nil, metrics.NewBoundedHistogramSample())
	gasUsedSinceStartupCounter = metrics.NewRegisteredCounter("arb/gas_used", nil)
)

type L1PriceDataOfMsg struct {
	callDataUnits            uint64
	cummulativeCallDataUnits uint64
	l1GasCharged             uint64
	cummulativeL1GasCharged  uint64
}

type L1PriceData struct {
	mutex                       sync.RWMutex
	startOfL1PriceDataCache     arbutil.MessageIndex
	endOfL1PriceDataCache       arbutil.MessageIndex
	msgToL1PriceData            []L1PriceDataOfMsg
	currentEstimateOfL1GasPrice uint64
}

type ExecutionEngine struct {
	stopwaiter.StopWaiter

	bc        *core.BlockChain
	consensus execution.FullConsensusClient
	recorder  *BlockRecorder

	resequenceChan    chan []*arbostypes.MessageWithMetadata
	createBlocksMutex sync.Mutex

	newBlockNotifier chan struct{}
	latestBlockMutex sync.Mutex
	latestBlock      *types.Block

	nextScheduledVersionCheck time.Time // protected by the createBlocksMutex

	reorgSequencing bool

	prefetchBlock bool

	cachedL1PriceData *L1PriceData
}

func NewL1PriceData() *L1PriceData {
	return &L1PriceData{
		msgToL1PriceData: []L1PriceDataOfMsg{},
	}
}

func NewExecutionEngine(bc *core.BlockChain) (*ExecutionEngine, error) {
	return &ExecutionEngine{
		bc:                bc,
		resequenceChan:    make(chan []*arbostypes.MessageWithMetadata),
		newBlockNotifier:  make(chan struct{}, 1),
		cachedL1PriceData: NewL1PriceData(),
	}, nil
}

func (s *ExecutionEngine) backlogCallDataUnits() uint64 {
	s.cachedL1PriceData.mutex.RLock()
	defer s.cachedL1PriceData.mutex.RUnlock()

	size := len(s.cachedL1PriceData.msgToL1PriceData)
	if size == 0 {
		return 0
	}
	return (s.cachedL1PriceData.msgToL1PriceData[size-1].cummulativeCallDataUnits -
		s.cachedL1PriceData.msgToL1PriceData[0].cummulativeCallDataUnits +
		s.cachedL1PriceData.msgToL1PriceData[0].callDataUnits)
}

func (s *ExecutionEngine) backlogL1GasCharged() uint64 {
	s.cachedL1PriceData.mutex.RLock()
	defer s.cachedL1PriceData.mutex.RUnlock()

	size := len(s.cachedL1PriceData.msgToL1PriceData)
	if size == 0 {
		return 0
	}
	return (s.cachedL1PriceData.msgToL1PriceData[size-1].cummulativeL1GasCharged -
		s.cachedL1PriceData.msgToL1PriceData[0].cummulativeL1GasCharged +
		s.cachedL1PriceData.msgToL1PriceData[0].l1GasCharged)
}

func (s *ExecutionEngine) TrimCache(to arbutil.MessageIndex) {
	s.cachedL1PriceData.mutex.Lock()
	defer s.cachedL1PriceData.mutex.Unlock()

	if to < s.cachedL1PriceData.startOfL1PriceDataCache {
		log.Info("trying to trim older cache which doesnt exist anymore")
	} else if to >= s.cachedL1PriceData.endOfL1PriceDataCache {
		s.cachedL1PriceData.startOfL1PriceDataCache = 0
		s.cachedL1PriceData.endOfL1PriceDataCache = 0
		s.cachedL1PriceData.msgToL1PriceData = []L1PriceDataOfMsg{}
	} else {
		newStart := to - s.cachedL1PriceData.startOfL1PriceDataCache + 1
		s.cachedL1PriceData.msgToL1PriceData = s.cachedL1PriceData.msgToL1PriceData[newStart:]
		s.cachedL1PriceData.startOfL1PriceDataCache = to + 1
	}
}

func (s *ExecutionEngine) Initialize(rustCacheSize uint32) {
	if rustCacheSize != 0 {
		programs.ResizeWasmLruCache(rustCacheSize)
	}
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

func (s *ExecutionEngine) SetConsensus(consensus execution.FullConsensusClient) {
	if s.Started() {
		panic("trying to set transaction consensus after start")
	}
	if s.consensus != nil {
		panic("trying to set transaction consensus when already set")
	}
	s.consensus = consensus
}

func (s *ExecutionEngine) GetBatchFetcher() execution.BatchFetcher {
	return s.consensus
}

func (s *ExecutionEngine) Reorg(count arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockHash, oldMessages []*arbostypes.MessageWithMetadata) ([]*execution.MessageResult, error) {
	if count == 0 {
		return nil, errors.New("cannot reorg out genesis")
	}
	s.createBlocksMutex.Lock()
	resequencing := false
	defer func() {
		// if we are resequencing old messages - don't release the lock
		// lock will be released by thread listening to resequenceChan
		if !resequencing {
			s.createBlocksMutex.Unlock()
		}
	}()
	blockNum := s.MessageIndexToBlockNumber(count - 1)
	// We can safely cast blockNum to a uint64 as it comes from MessageCountToBlockNumber
	targetBlock := s.bc.GetBlockByNumber(uint64(blockNum))
	if targetBlock == nil {
		log.Warn("reorg target block not found", "block", blockNum)
		return nil, nil
	}

	tag := s.bc.StateCache().WasmCacheTag()
	// reorg Rust-side VM state
	C.stylus_reorg_vm(C.uint64_t(blockNum), C.uint32_t(tag))

	err := s.bc.ReorgToOldBlock(targetBlock)
	if err != nil {
		return nil, err
	}

	newMessagesResults := make([]*execution.MessageResult, 0, len(oldMessages))
	for i := range newMessages {
		var msgForPrefetch *arbostypes.MessageWithMetadata
		if i < len(newMessages)-1 {
			msgForPrefetch = &newMessages[i].MessageWithMeta
		}
		msgResult, err := s.digestMessageWithBlockMutex(count+arbutil.MessageIndex(i), &newMessages[i].MessageWithMeta, msgForPrefetch)
		if err != nil {
			return nil, err
		}
		newMessagesResults = append(newMessagesResults, msgResult)
	}
	if s.recorder != nil {
		s.recorder.ReorgTo(targetBlock.Header())
	}
	if len(oldMessages) > 0 {
		s.resequenceChan <- oldMessages
		resequencing = true
	}
	return newMessagesResults, nil
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

func MessageFromTxes(header *arbostypes.L1IncomingMessageHeader, txes types.Transactions, txErrors []error) (*arbostypes.L1IncomingMessage, error) {
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
	if len(l2Message) > arbostypes.MaxL2MessageSize {
		return nil, errors.New("l2message too long")
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
		chosenErr := s.consensus.ExpectChosenSequencer()
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

// SequenceTransactionsWithProfiling runs SequenceTransactions with tracing and
// CPU profiling enabled. If the block creation takes longer than 2 seconds, it
// keeps both and prints out filenames in an error log line.
func (s *ExecutionEngine) SequenceTransactionsWithProfiling(header *arbostypes.L1IncomingMessageHeader, txes types.Transactions, hooks *arbos.SequencingHooks) (*types.Block, error) {
	pprofBuf, traceBuf := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	if err := pprof.StartCPUProfile(pprofBuf); err != nil {
		log.Error("Starting CPU profiling", "error", err)
	}
	if err := trace.Start(traceBuf); err != nil {
		log.Error("Starting tracing", "error", err)
	}
	start := time.Now()
	res, err := s.SequenceTransactions(header, txes, hooks)
	elapsed := time.Since(start)
	pprof.StopCPUProfile()
	trace.Stop()
	if elapsed > 2*time.Second {
		writeAndLog(pprofBuf, traceBuf)
		return res, err
	}
	return res, err
}

func writeAndLog(pprof, trace *bytes.Buffer) {
	id := uuid.NewString()
	pprofFile := path.Join(os.TempDir(), id+".pprof")
	if err := os.WriteFile(pprofFile, pprof.Bytes(), 0o600); err != nil {
		log.Error("Creating temporary file for pprof", "fileName", pprofFile, "error", err)
		return
	}
	traceFile := path.Join(os.TempDir(), id+".trace")
	if err := os.WriteFile(traceFile, trace.Bytes(), 0o600); err != nil {
		log.Error("Creating temporary file for trace", "fileName", traceFile, "error", err)
		return
	}
	log.Info("Transactions sequencing took longer than 2 seconds, created pprof and trace files", "pprof", pprofFile, "traceFile", traceFile)
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
		false,
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

	msg, err := MessageFromTxes(header, txes, hooks.TxErrors)
	if err != nil {
		return nil, err
	}

	pos, err := s.BlockNumberToMessageIndex(lastBlockHeader.Number.Uint64() + 1)
	if err != nil {
		return nil, err
	}

	msgWithMeta := arbostypes.MessageWithMetadata{
		Message:             msg,
		DelayedMessagesRead: delayedMessagesRead,
	}
	msgResult, err := s.resultFromHeader(block.Header())
	if err != nil {
		return nil, err
	}

	err = s.consensus.WriteMessageFromSequencer(pos, msgWithMeta, *msgResult)
	if err != nil {
		return nil, err
	}

	// Only write the block after we've written the messages, so if the node dies in the middle of this,
	// it will naturally recover on startup by regenerating the missing block.
	err = s.appendBlock(block, statedb, receipts, blockCalcTime)
	if err != nil {
		return nil, err
	}

	s.cacheL1PriceDataOfMsg(pos, receipts, block)

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

	startTime := time.Now()
	block, statedb, receipts, err := s.createBlockFromNextMessage(&messageWithMeta, false)
	if err != nil {
		return nil, err
	}
	blockCalcTime := time.Since(startTime)

	msgResult, err := s.resultFromHeader(block.Header())
	if err != nil {
		return nil, err
	}

	err = s.consensus.WriteMessageFromSequencer(lastMsg+1, messageWithMeta, *msgResult)
	if err != nil {
		return nil, err
	}

	err = s.appendBlock(block, statedb, receipts, blockCalcTime)
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
func (s *ExecutionEngine) createBlockFromNextMessage(msg *arbostypes.MessageWithMetadata, isMsgForPrefetch bool) (*types.Block, *state.StateDB, types.Receipts, error) {
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

	batchFetcher := func(num uint64) ([]byte, error) {
		data, _, err := s.consensus.FetchBatch(s.GetContext(), num)
		return data, err
	}

	block, receipts, err := arbos.ProduceBlock(
		msg.Message,
		msg.DelayedMessagesRead,
		currentHeader,
		statedb,
		s.bc,
		s.bc.Config(),
		batchFetcher,
		isMsgForPrefetch,
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
	baseFeeGauge.Update(block.BaseFee().Int64())
	txCountHistogram.Update(int64(len(block.Transactions()) - 1))
	var blockGasused uint64
	for i := 1; i < len(receipts); i++ {
		val := arbmath.SaturatingUSub(receipts[i].GasUsed, receipts[i].GasUsedForL1)
		txGasUsedHistogram.Update(int64(val))
		blockGasused += val
	}
	blockGasUsedHistogram.Update(int64(blockGasused))
	gasUsedSinceStartupCounter.Inc(int64(blockGasused))
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

func (s *ExecutionEngine) getL1GasPriceEstimate() uint64 {
	bc := s.bc
	latestHeader := bc.CurrentBlock()
	latestState, err := bc.StateAt(latestHeader.Root)
	if err != nil {
		log.Error("error getting latest statedb while fetching l2 Estimate of L1 GasPrice")
		return s.cachedL1PriceData.currentEstimateOfL1GasPrice
	}
	arbState, err := arbosState.OpenSystemArbosState(latestState, nil, true)
	if err != nil {
		log.Error("error opening system arbos state while fetching l2 Estimate of L1 GasPrice")
		return s.cachedL1PriceData.currentEstimateOfL1GasPrice
	}
	l2EstimateL1GasPrice, err := arbState.L1PricingState().PricePerUnit()
	if err != nil {
		log.Error("error fetching l2 Estimate of L1 GasPrice")
		return s.cachedL1PriceData.currentEstimateOfL1GasPrice
	}
	s.cachedL1PriceData.currentEstimateOfL1GasPrice = l2EstimateL1GasPrice.Uint64()
	return s.cachedL1PriceData.currentEstimateOfL1GasPrice
}

func (s *ExecutionEngine) getL1PricingSurplus() (int64, error) {
	bc := s.bc
	latestHeader := bc.CurrentBlock()
	latestState, err := bc.StateAt(latestHeader.Root)
	if err != nil {
		return 0, errors.New("error getting latest statedb while fetching current L1 pricing surplus")
	}
	arbState, err := arbosState.OpenSystemArbosState(latestState, nil, true)
	if err != nil {
		return 0, errors.New("error opening system arbos state while fetching current L1 pricing surplus")
	}
	surplus, err := arbState.L1PricingState().GetL1PricingSurplus()
	if err != nil {
		return 0, errors.New("error fetching current L1 pricing surplus")
	}
	return surplus.Int64(), nil
}

func (s *ExecutionEngine) cacheL1PriceDataOfMsg(seqNum arbutil.MessageIndex, receipts types.Receipts, block *types.Block) {
	var gasUsedForL1 uint64
	for i := 1; i < len(receipts); i++ {
		gasUsedForL1 += receipts[i].GasUsedForL1
	}
	l1GasCharged := gasUsedForL1 * block.BaseFee().Uint64()
	var callDataUnits uint64
	for _, tx := range block.Transactions() {
		callDataUnits += tx.CalldataUnits
	}

	s.cachedL1PriceData.mutex.Lock()
	defer s.cachedL1PriceData.mutex.Unlock()

	resetCache := func() {
		s.cachedL1PriceData.startOfL1PriceDataCache = seqNum
		s.cachedL1PriceData.endOfL1PriceDataCache = seqNum
		s.cachedL1PriceData.msgToL1PriceData = []L1PriceDataOfMsg{{
			callDataUnits:            callDataUnits,
			cummulativeCallDataUnits: callDataUnits,
			l1GasCharged:             l1GasCharged,
			cummulativeL1GasCharged:  l1GasCharged,
		}}
	}
	size := len(s.cachedL1PriceData.msgToL1PriceData)
	if size == 0 ||
		s.cachedL1PriceData.startOfL1PriceDataCache == 0 ||
		s.cachedL1PriceData.endOfL1PriceDataCache == 0 ||
		arbutil.MessageIndex(size) != s.cachedL1PriceData.endOfL1PriceDataCache-s.cachedL1PriceData.startOfL1PriceDataCache+1 {
		resetCache()
		return
	}
	if seqNum != s.cachedL1PriceData.endOfL1PriceDataCache+1 {
		if seqNum > s.cachedL1PriceData.endOfL1PriceDataCache+1 {
			log.Info("message position higher then current end of l1 price data cache, resetting cache to this message")
			resetCache()
		} else if seqNum < s.cachedL1PriceData.startOfL1PriceDataCache {
			log.Info("message position lower than start of l1 price data cache, ignoring")
		} else {
			log.Info("message position already seen in l1 price data cache, ignoring")
		}
	} else {
		cummulativeCallDataUnits := s.cachedL1PriceData.msgToL1PriceData[size-1].cummulativeCallDataUnits
		cummulativeL1GasCharged := s.cachedL1PriceData.msgToL1PriceData[size-1].cummulativeL1GasCharged
		s.cachedL1PriceData.msgToL1PriceData = append(s.cachedL1PriceData.msgToL1PriceData, L1PriceDataOfMsg{
			callDataUnits:            callDataUnits,
			cummulativeCallDataUnits: cummulativeCallDataUnits + callDataUnits,
			l1GasCharged:             l1GasCharged,
			cummulativeL1GasCharged:  cummulativeL1GasCharged + l1GasCharged,
		})
		s.cachedL1PriceData.endOfL1PriceDataCache = seqNum
	}
}

// DigestMessage is used to create a block by executing msg against the latest state and storing it.
// Also, while creating a block by executing msg against the latest state,
// in parallel, creates a block by executing msgForPrefetch (msg+1) against the latest state
// but does not store the block.
// This helps in filling the cache, so that the next block creation is faster.
func (s *ExecutionEngine) DigestMessage(num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) (*execution.MessageResult, error) {
	if !s.createBlocksMutex.TryLock() {
		return nil, errors.New("createBlock mutex held")
	}
	defer s.createBlocksMutex.Unlock()
	return s.digestMessageWithBlockMutex(num, msg, msgForPrefetch)
}

func (s *ExecutionEngine) digestMessageWithBlockMutex(num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) (*execution.MessageResult, error) {
	currentHeader, err := s.getCurrentHeader()
	if err != nil {
		return nil, err
	}
	curMsg, err := s.BlockNumberToMessageIndex(currentHeader.Number.Uint64())
	if err != nil {
		return nil, err
	}
	if curMsg+1 != num {
		return nil, fmt.Errorf("wrong message number in digest got %d expected %d", num, curMsg+1)
	}

	startTime := time.Now()
	if s.prefetchBlock && msgForPrefetch != nil {
		go func() {
			_, _, _, err := s.createBlockFromNextMessage(msgForPrefetch, true)
			if err != nil {
				return
			}
		}()
	}

	block, statedb, receipts, err := s.createBlockFromNextMessage(msg, false)
	if err != nil {
		return nil, err
	}

	err = s.appendBlock(block, statedb, receipts, time.Since(startTime))
	if err != nil {
		return nil, err
	}

	if time.Now().After(s.nextScheduledVersionCheck) {
		s.nextScheduledVersionCheck = time.Now().Add(time.Minute)
		arbState, err := arbosState.OpenSystemArbosState(statedb, nil, true)
		if err != nil {
			return nil, err
		}
		version, timestampInt, err := arbState.GetScheduledUpgrade()
		if err != nil {
			return nil, err
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

	msgResult, err := s.resultFromHeader(block.Header())
	if err != nil {
		return nil, err
	}
	return msgResult, nil
}

func (s *ExecutionEngine) ArbOSVersionForMessageNumber(messageNum arbutil.MessageIndex) (uint64, error) {
	block := s.bc.GetBlockByNumber(s.MessageIndexToBlockNumber(messageNum))
	if block == nil {
		return 0, fmt.Errorf("couldn't find block for message number %d", messageNum)
	}
	extra := types.DeserializeHeaderExtraInformation(block.Header())
	return extra.ArbOSFormatVersion, nil
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
