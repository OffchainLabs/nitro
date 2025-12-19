// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !wasm

package gethexec

/*
#cgo CFLAGS: -g -I../../target/include/
#cgo LDFLAGS: ${SRCDIR}/../../target/lib/libstylus.a -ldl -lm
#include "arbitrator.h"
*/
import "C"

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"runtime/pprof"
	"runtime/trace"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbos/programs"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/sharedmetrics"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	l1GasPriceEstimateGauge              = metrics.NewRegisteredGauge("arb/l1gasprice/estimate", nil)
	baseFeeGauge                         = metrics.NewRegisteredGauge("arb/block/basefee", nil)
	blockGasUsedHistogram                = metrics.NewRegisteredHistogram("arb/block/gasused", nil, metrics.NewBoundedHistogramSample())
	txCountHistogram                     = metrics.NewRegisteredHistogram("arb/block/transactions/count", nil, metrics.NewBoundedHistogramSample())
	txGasUsedHistogram                   = metrics.NewRegisteredHistogram("arb/block/transactions/gasused", nil, metrics.NewBoundedHistogramSample())
	gasUsedSinceStartupCounter           = metrics.NewRegisteredCounter("arb/gas_used", nil)
	multiGasUsedSinceStartupCounters     = make([]*metrics.Counter, multigas.NumResourceKind)
	totalMultiGasUsedSinceStartupCounter = metrics.NewRegisteredCounter("arb/multigas_used/total", nil)
	blockExecutionTimer                  = metrics.NewRegisteredHistogram("arb/block/execution", nil, metrics.NewBoundedHistogramSample())
	blockWriteToDbTimer                  = metrics.NewRegisteredHistogram("arb/block/writetodb", nil, metrics.NewBoundedHistogramSample())
)

var ExecutionEngineBlockCreationStopped = errors.New("block creation stopped in execution engine")
var ResultNotFound = errors.New("result not found")

type L1PriceDataOfMsg struct {
	callDataUnits            uint64
	cummulativeCallDataUnits uint64
}

type L1PriceData struct {
	mutex                   sync.RWMutex
	startOfL1PriceDataCache arbutil.MessageIndex
	endOfL1PriceDataCache   arbutil.MessageIndex
	msgToL1PriceData        []L1PriceDataOfMsg
}

type ExecutionEngine struct {
	stopwaiter.StopWaiter

	bc        *core.BlockChain
	consensus execution.FullConsensusClient
	recorder  *BlockRecorder

	resequenceChan    chan []*arbostypes.MessageWithMetadata
	createBlocksMutex sync.Mutex

	newBlockNotifier    chan struct{}
	reorgEventsNotifier chan struct{}
	latestBlockMutex    sync.Mutex
	latestBlock         *types.Block

	nextScheduledVersionCheck time.Time // protected by the createBlocksMutex

	reorgSequencing bool

	disableStylusCacheMetricsCollection bool

	prefetchBlock bool

	cachedL1PriceData *L1PriceData

	wasmTargets []rawdb.WasmTarget

	syncTillBlock uint64

	exposeMultiGas bool

	runningMaintenance atomic.Bool
}

func NewL1PriceData() *L1PriceData {
	return &L1PriceData{
		msgToL1PriceData: []L1PriceDataOfMsg{},
	}
}

func init() {
	for dimension := multigas.ResourceKind(0); dimension < multigas.NumResourceKind; dimension++ {
		metricName := fmt.Sprintf("arb/multigas_used/%v", strings.ToLower(dimension.String()))
		multiGasUsedSinceStartupCounters[dimension] = metrics.NewRegisteredCounter(metricName, nil)
	}
}

func NewExecutionEngine(bc *core.BlockChain, syncTillBlock uint64, exposeMultiGas bool) (*ExecutionEngine, error) {
	return &ExecutionEngine{
		bc:                bc,
		resequenceChan:    make(chan []*arbostypes.MessageWithMetadata),
		newBlockNotifier:  make(chan struct{}, 1),
		cachedL1PriceData: NewL1PriceData(),
		exposeMultiGas:    exposeMultiGas,
		syncTillBlock:     syncTillBlock,
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

func (s *ExecutionEngine) MarkFeedStart(to arbutil.MessageIndex) {
	s.cachedL1PriceData.mutex.Lock()
	defer s.cachedL1PriceData.mutex.Unlock()

	if to < s.cachedL1PriceData.startOfL1PriceDataCache {
		log.Debug("trying to trim older L1 price data cache which doesn't exist anymore")
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

func PopulateStylusTargetCache(targetConfig *StylusTargetConfig) error {
	localTarget := rawdb.LocalTarget()
	targets := targetConfig.WasmTargets()
	var nativeSet bool
	for _, target := range targets {
		var effectiveStylusTarget string
		switch target {
		case rawdb.TargetWavm:
			// skip wavm target
			continue
		case rawdb.TargetArm64:
			effectiveStylusTarget = targetConfig.Arm64
		case rawdb.TargetAmd64:
			effectiveStylusTarget = targetConfig.Amd64
		case rawdb.TargetHost:
			effectiveStylusTarget = targetConfig.Host
		default:
			return fmt.Errorf("unsupported stylus target: %v", target)
		}
		isNative := target == localTarget
		err := programs.SetTarget(target, effectiveStylusTarget, isNative)
		if err != nil {
			return fmt.Errorf("failed to set stylus target: %w", err)
		}
		nativeSet = nativeSet || isNative
	}
	if !nativeSet {
		return fmt.Errorf("local target %v missing in list of archs %v", localTarget, targets)
	}
	return nil
}

func (s *ExecutionEngine) Initialize(rustCacheCapacityMB uint32, targetConfig *StylusTargetConfig) error {
	if rustCacheCapacityMB != 0 {
		programs.SetWasmLruCacheCapacity(arbmath.SaturatingUMul(uint64(rustCacheCapacityMB), 1024*1024))
	}
	if err := PopulateStylusTargetCache(targetConfig); err != nil {
		return fmt.Errorf("error populating stylus target cache: %w", err)
	}
	s.wasmTargets = targetConfig.WasmTargets()
	return nil
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

func (s *ExecutionEngine) SetReorgEventsNotifier(reorgEventsNotifier chan struct{}) {
	if s.Started() {
		panic("trying to set reorg events notifier after start")
	}
	if s.reorgEventsNotifier != nil {
		panic("trying to set reorg events notifier when already set")
	}
	s.reorgEventsNotifier = reorgEventsNotifier
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

func (s *ExecutionEngine) DisableStylusCacheMetricsCollection() {
	if s.Started() {
		panic("trying to disable stylus cache metrics collection after start")
	}
	if s.disableStylusCacheMetricsCollection {
		panic("trying to disable stylus cache metrics collection when already set")
	}
	s.disableStylusCacheMetricsCollection = true
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

func (s *ExecutionEngine) BlockMetadataAtMessageIndex(ctx context.Context, msgIdx arbutil.MessageIndex) (common.BlockMetadata, error) {
	if s.consensus != nil {
		return s.consensus.BlockMetadataAtMessageIndex(msgIdx).Await(ctx)
	}
	return nil, errors.New("FullConsensusClient is not accessible to execution")
}

func (s *ExecutionEngine) GetBatchFetcher() execution.BatchFetcher {
	return s.consensus
}

func (s *ExecutionEngine) Reorg(msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) ([]*execution.MessageResult, error) {
	if msgIdxOfFirstMsgToAdd == 0 {
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
	lastBlockNumToKeep := s.MessageIndexToBlockNumber(msgIdxOfFirstMsgToAdd - 1)
	// We can safely cast lastBlockNumToKeep to a uint64 as it comes from MessageIndexToBlockNumber
	lastBlockToKeep := s.bc.GetBlockByNumber(uint64(lastBlockNumToKeep))
	if lastBlockToKeep == nil {
		log.Warn("reorg target block not found", "block", lastBlockNumToKeep)
		return nil, nil
	}

	currentSafeBlock := s.bc.CurrentSafeBlock()
	if currentSafeBlock != nil && lastBlockToKeep.Number().Cmp(currentSafeBlock.Number) < 0 {
		log.Warn("reorg target block is below safe block", "lastBlockNumToKeep", lastBlockNumToKeep, "currentSafeBlock", currentSafeBlock.Number)
		s.bc.SetSafe(nil)
	}
	currentFinalBlock := s.bc.CurrentFinalBlock()
	if currentFinalBlock != nil && lastBlockToKeep.Number().Cmp(currentFinalBlock.Number) < 0 {
		log.Warn("reorg target block is below final block", "lastBlockNumToKeep", lastBlockNumToKeep, "currentFinalBlock", currentFinalBlock.Number)
		s.bc.SetFinalized(nil)
	}

	tag := core.NewMessageCommitContext(nil).WasmCacheTag() // we don't pass any targets, we just want the tag
	// reorg Rust-side VM state
	C.stylus_reorg_vm(C.uint64_t(lastBlockNumToKeep), C.uint32_t(tag))

	err := s.bc.ReorgToOldBlock(lastBlockToKeep)
	if err != nil {
		return nil, err
	}

	if s.reorgEventsNotifier != nil {
		select {
		case s.reorgEventsNotifier <- struct{}{}:
		default:
		}
	}

	newMessagesResults := make([]*execution.MessageResult, 0, len(newMessages))
	for i := range newMessages {
		var msgForPrefetch *arbostypes.MessageWithMetadata
		if i < len(newMessages)-1 {
			msgForPrefetch = &newMessages[i].MessageWithMeta
		}
		nextMsgIdx := msgIdxOfFirstMsgToAdd + arbutil.MessageIndex(i)
		msgResult, err := s.digestMessageWithBlockMutex(nextMsgIdx, &newMessages[i].MessageWithMeta, msgForPrefetch)
		if err != nil {
			return nil, err
		}
		newMessagesResults = append(newMessagesResults, msgResult)
	}
	if s.recorder != nil {
		s.recorder.ReorgTo(lastBlockToKeep.Header())
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

func (s *ExecutionEngine) HeadMessageIndex() (arbutil.MessageIndex, error) {
	currentHeader, err := s.getCurrentHeader()
	if err != nil {
		return 0, err
	}
	return s.BlockNumberToMessageIndex(currentHeader.Number.Uint64())
}

func (s *ExecutionEngine) HeadMessageIndexSync(t *testing.T) (arbutil.MessageIndex, error) {
	s.createBlocksMutex.Lock()
	defer s.createBlocksMutex.Unlock()
	return s.HeadMessageIndex()
}

func (s *ExecutionEngine) NextDelayedMessageNumber() (uint64, error) {
	currentHeader, err := s.getCurrentHeader()
	if err != nil {
		return 0, err
	}
	return currentHeader.Nonce.Uint64(), nil
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

	nextDelayedMsgIdx := lastBlockHeader.Nonce.Uint64()

	for _, msg := range messages {
		// Check if the message is non-nil just to be safe
		if msg == nil || msg.Message == nil || msg.Message.Header == nil {
			continue
		}
		header := msg.Message.Header
		if header.RequestId != nil {
			delayedMsgIdx := header.RequestId.Big().Uint64()
			if delayedMsgIdx != nextDelayedMsgIdx {
				log.Info("not resequencing delayed message due to unexpected index", "expected", nextDelayedMsgIdx, "found", delayedMsgIdx)
				continue
			}
			_, err := s.sequenceDelayedMessageWithBlockMutex(msg.Message, delayedMsgIdx)
			if err != nil {
				log.Error("failed to re-sequence old delayed message removed by reorg", "err", err)
			}
			nextDelayedMsgIdx += 1
			continue
		}
		if header.Kind != arbostypes.L1MessageType_L2Message || header.Poster != l1pricing.BatchPosterAddress {
			// This shouldn't exist?
			log.Warn("skipping non-standard sequencer message found from reorg", "header", header)
			continue
		}
		lastArbosVersion := types.DeserializeHeaderExtraInformation(lastBlockHeader).ArbOSFormatVersion
		txes, err := arbos.ParseL2Transactions(msg.Message, s.bc.Config().ChainID, lastArbosVersion)
		if err != nil {
			log.Warn("failed to parse sequencer message found from reorg", "err", err)
			continue
		}
		hooks := MakeZeroTxSizeSequencingHooksForTesting(txes, nil, nil, nil)
		block, err := s.sequenceTransactionsWithBlockMutex(msg.Message.Header, hooks, nil)
		if err != nil {
			log.Error("failed to re-sequence old user message removed by reorg", "err", err)
			return
		}
		if block != nil {
			lastBlockHeader = block.Header()
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
		_, chosenErr := s.consensus.ExpectChosenSequencer().Await(s.GetContext())
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

func (s *ExecutionEngine) SequenceTransactions(header *arbostypes.L1IncomingMessageHeader, hooks *FullSequencingHooks, timeboostedTxs map[common.Hash]struct{}) (*types.Block, error) {
	return s.sequencerWrapper(func() (*types.Block, error) {
		return s.sequenceTransactionsWithBlockMutex(header, hooks, timeboostedTxs)
	})
}

// SequenceTransactionsWithProfiling runs SequenceTransactions with tracing and
// CPU profiling enabled. If the block creation takes longer than 2 seconds, it
// keeps both and prints out filenames in an error log line.
func (s *ExecutionEngine) SequenceTransactionsWithProfiling(header *arbostypes.L1IncomingMessageHeader, hooks *FullSequencingHooks, timeboostedTxs map[common.Hash]struct{}) (*types.Block, error) {
	pprofBuf, traceBuf := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	if err := pprof.StartCPUProfile(pprofBuf); err != nil {
		log.Error("Starting CPU profiling", "error", err)
	}
	if err := trace.Start(traceBuf); err != nil {
		log.Error("Starting tracing", "error", err)
	}
	start := time.Now()
	res, err := s.SequenceTransactions(header, hooks, timeboostedTxs)
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

func (s *ExecutionEngine) sequenceTransactionsWithBlockMutex(header *arbostypes.L1IncomingMessageHeader, hooks *FullSequencingHooks, timeboostedTxs map[common.Hash]struct{}) (*types.Block, error) {
	lastBlockHeader, err := s.getCurrentHeader()
	if err != nil {
		return nil, err
	}

	statedb, err := s.bc.StateAt(lastBlockHeader.Root)
	if err != nil {
		return nil, err
	}
	lastBlock := s.bc.GetBlock(lastBlockHeader.Hash(), lastBlockHeader.Number.Uint64())
	if lastBlock == nil {
		return nil, errors.New("can't find block for current header")
	}
	var witness *stateless.Witness
	var witnessStats *stateless.WitnessStats
	if s.bc.GetVMConfig().StatelessSelfValidation {
		witness, err = stateless.NewWitness(lastBlock.Header(), s.bc)
		if err != nil {
			return nil, err
		}
		if s.bc.GetVMConfig().EnableWitnessStats {
			witnessStats = stateless.NewWitnessStats()
		}
	}
	statedb.StartPrefetcher("Sequencer", witness, witnessStats)
	defer statedb.StopPrefetcher()
	delayedMessagesRead := lastBlockHeader.Nonce.Uint64()

	startTime := time.Now()
	block, receipts, err := arbos.ProduceBlockAdvanced(
		header,
		delayedMessagesRead,
		lastBlockHeader,
		statedb,
		s.bc,
		hooks,
		false,
		core.NewMessageCommitContext(s.wasmTargets),
		s.exposeMultiGas,
	)
	if err != nil {
		return nil, err
	}
	blockCalcTime := time.Since(startTime)
	blockExecutionTimer.Update(blockCalcTime.Nanoseconds())

	if len(receipts) == 0 {
		return nil, nil
	}

	allTxsErrored := true
	for _, err := range hooks.txErrors {
		if err == nil {
			allTxsErrored = false
			break
		}
	}
	if allTxsErrored {
		return nil, nil
	}

	msg, err := hooks.MessageFromTxes(header)
	if err != nil {
		return nil, err
	}

	msgIdx, err := s.BlockNumberToMessageIndex(lastBlockHeader.Number.Uint64() + 1)
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

	blockMetadata := s.blockMetadataFromBlock(block, timeboostedTxs)
	_, err = s.consensus.WriteMessageFromSequencer(msgIdx, msgWithMeta, *msgResult, blockMetadata).Await(s.GetContext())
	if err != nil {
		return nil, err
	}

	// Only write the block after we've written the messages, so if the node dies in the middle of this,
	// it will naturally recover on startup by regenerating the missing block.
	err = s.appendBlock(block, statedb, receipts, blockCalcTime)
	if err != nil {
		return nil, err
	}
	s.cacheL1PriceDataOfMsg(msgIdx, block, false)

	return block, nil
}

// blockMetadataFromBlock returns timeboosted byte array which says whether a transaction in the block was timeboosted
// or not. The first byte of blockMetadata byte array is reserved to indicate the version,
// starting from the second byte, (N)th bit would represent if (N)th tx is timeboosted or not, 1 means yes and 0 means no
// blockMetadata[index / 8 + 1] & (1 << (index % 8)) != 0; where index = (N - 1), implies whether (N)th tx in a block is timeboosted
// note that number of txs in a block will always lag behind (len(blockMetadata) - 1) * 8 but it won't lag more than a value of 7
func (s *ExecutionEngine) blockMetadataFromBlock(block *types.Block, timeboostedTxs map[common.Hash]struct{}) common.BlockMetadata {
	bits := make(common.BlockMetadata, 1+arbmath.DivCeil(uint64(len(block.Transactions())), 8))
	if len(timeboostedTxs) == 0 {
		return bits
	}
	for i, tx := range block.Transactions() {
		if _, ok := timeboostedTxs[tx.Hash()]; ok {
			bits[1+i/8] |= 1 << (i % 8)
		}
	}
	return bits
}

func (s *ExecutionEngine) SequenceDelayedMessage(message *arbostypes.L1IncomingMessage, delayedMsgIdx uint64) error {
	_, err := s.sequencerWrapper(func() (*types.Block, error) {
		return s.sequenceDelayedMessageWithBlockMutex(message, delayedMsgIdx)
	})
	return err
}

func (s *ExecutionEngine) sequenceDelayedMessageWithBlockMutex(message *arbostypes.L1IncomingMessage, delayedMsgIdx uint64) (*types.Block, error) {
	if s.syncTillBlock > 0 && s.latestBlock != nil && s.latestBlock.NumberU64() >= s.syncTillBlock {
		return nil, ExecutionEngineBlockCreationStopped
	}
	currentHeader, err := s.getCurrentHeader()
	if err != nil {
		return nil, err
	}

	expectedDelayedMsgIdx := currentHeader.Nonce.Uint64()

	msgIdx, err := s.BlockNumberToMessageIndex(currentHeader.Number.Uint64() + 1)
	if err != nil {
		return nil, err
	}

	if expectedDelayedMsgIdx != delayedMsgIdx {
		return nil, fmt.Errorf("wrong delayed message sequenced got %d expected %d", delayedMsgIdx, expectedDelayedMsgIdx)
	}

	messageWithMeta := arbostypes.MessageWithMetadata{
		Message:             message,
		DelayedMessagesRead: delayedMsgIdx + 1,
	}

	startTime := time.Now()
	block, statedb, receipts, err := s.createBlockFromNextMessage(&messageWithMeta, false)
	if err != nil {
		return nil, err
	}
	blockCalcTime := time.Since(startTime)
	blockExecutionTimer.Update(blockCalcTime.Nanoseconds())

	msgResult, err := s.resultFromHeader(block.Header())
	if err != nil {
		return nil, err
	}

	_, err = s.consensus.WriteMessageFromSequencer(msgIdx, messageWithMeta, *msgResult, s.blockMetadataFromBlock(block, nil)).Await(s.GetContext())
	if err != nil {
		return nil, err
	}

	err = s.appendBlock(block, statedb, receipts, blockCalcTime)
	if err != nil {
		return nil, err
	}
	s.cacheL1PriceDataOfMsg(msgIdx, block, true)

	log.Info("ExecutionEngine: Added DelayedMessages", "msgIdx", msgIdx, "delayedMsgIdx", delayedMsgIdx, "block-header", block.Header())

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

func (s *ExecutionEngine) MessageIndexToBlockNumber(msgIdx arbutil.MessageIndex) uint64 {
	return uint64(msgIdx) + s.GetGenesisBlockNumber()
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
	var witness *stateless.Witness
	var witnessStats *stateless.WitnessStats
	if s.bc.GetVMConfig().StatelessSelfValidation {
		witness, err = stateless.NewWitness(currentBlock.Header(), s.bc)
		if err != nil {
			return nil, nil, nil, err
		}
		if s.bc.GetVMConfig().EnableWitnessStats {
			witnessStats = stateless.NewWitnessStats()
		}
	}
	statedb.StartPrefetcher("TransactionStreamer", witness, witnessStats)
	defer statedb.StopPrefetcher()

	var runCtx *core.MessageRunContext
	if isMsgForPrefetch {
		runCtx = core.NewMessagePrefetchContext()
	} else {
		runCtx = core.NewMessageCommitContext(s.wasmTargets)
	}
	block, receipts, err := arbos.ProduceBlock(
		msg.Message,
		msg.DelayedMessagesRead,
		currentHeader,
		statedb,
		s.bc,
		isMsgForPrefetch,
		runCtx,
		s.exposeMultiGas,
	)

	return block, statedb, receipts, err
}

// must hold createBlockMutex
func (s *ExecutionEngine) appendBlock(block *types.Block, statedb *state.StateDB, receipts types.Receipts, duration time.Duration) error {
	var logs []*types.Log
	for _, receipt := range receipts {
		logs = append(logs, receipt.Logs...)
	}
	startTime := time.Now()
	if s.bc.GetVMConfig().Tracer != nil {
		// InsertChain is basically WriteBlockAndSetHeadWithTime along with recomputing
		// the entire block which is also traced which works directly for live-tracing
		if _, err := s.bc.InsertChain([]*types.Block{block}); err != nil {
			return err
		}
	} else {
		status, err := s.bc.WriteBlockAndSetHeadWithTime(block, receipts, logs, statedb, true, duration)
		if err != nil {
			return err
		}
		if status == core.SideStatTy { // TODO: This check can be removed as this WriteStatus is never returned when setting head
			return errors.New("geth rejected block as non-canonical")
		}
	}
	blockWriteToDbTimer.Update(time.Since(startTime).Nanoseconds())
	baseFeeGauge.Update(block.BaseFee().Int64())
	txCountHistogram.Update(int64(len(block.Transactions()) - 1))
	var blockGasused uint64
	for i := 1; i < len(receipts); i++ {
		receipt := receipts[i]
		val := arbmath.SaturatingUSub(receipt.GasUsed, receipt.GasUsedForL1)
		txGasUsedHistogram.Update(int64(val))
		blockGasused += val

		if s.exposeMultiGas {
			for kind := range multiGasUsedSinceStartupCounters {
				amount := receipt.MultiGasUsed.Get(multigas.ResourceKind(kind))
				if amount > 0 {
					multiGasUsedSinceStartupCounters[kind].Inc(int64(amount))
				}
			}
			totalMultiGasUsedSinceStartupCounter.Inc(int64(receipt.MultiGasUsed.SingleGas()))
		}
	}
	blockGasUsedHistogram.Update(int64(blockGasused))
	gasUsedSinceStartupCounter.Inc(int64(blockGasused))
	s.updateL1GasPriceEstimateMetric()
	return nil
}

func (s *ExecutionEngine) resultFromHeader(header *types.Header) (*execution.MessageResult, error) {
	if header == nil {
		return nil, ResultNotFound
	}
	info := types.DeserializeHeaderExtraInformation(header)
	return &execution.MessageResult{
		BlockHash: header.Hash(),
		SendRoot:  info.SendRoot,
	}, nil
}

func (s *ExecutionEngine) ResultAtMessageIndex(msgIdx arbutil.MessageIndex) (*execution.MessageResult, error) {
	return s.resultFromHeader(s.bc.GetHeaderByNumber(s.MessageIndexToBlockNumber(msgIdx)))
}

func (s *ExecutionEngine) updateL1GasPriceEstimateMetric() {
	bc := s.bc
	latestHeader := bc.CurrentBlock()
	latestState, err := bc.StateAt(latestHeader.Root)
	if err != nil {
		log.Error("error getting latest statedb while fetching l2 Estimate of L1 GasPrice")
		return
	}
	arbState, err := arbosState.OpenSystemArbosState(latestState, nil, true)
	if err != nil {
		log.Error("error opening system arbos state while fetching l2 Estimate of L1 GasPrice")
		return
	}
	l2EstimateL1GasPrice, err := arbState.L1PricingState().PricePerUnit()
	if err != nil {
		log.Error("error fetching l2 Estimate of L1 GasPrice")
		return
	}
	l1GasPriceEstimateGauge.Update(l2EstimateL1GasPrice.Int64())
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

func (s *ExecutionEngine) cacheL1PriceDataOfMsg(msgIdx arbutil.MessageIndex, block *types.Block, blockBuiltUsingDelayedMessage bool) {
	var callDataUnits uint64
	if !blockBuiltUsingDelayedMessage {
		// s.cachedL1PriceData tracks L1 price data for messages posted by Nitro,
		// so delayed messages should not update cummulative values kept on it.

		for _, tx := range block.Transactions() {
			_, cachedUnits := tx.GetRawCachedCalldataUnits()
			callDataUnits += cachedUnits
		}
	}

	s.cachedL1PriceData.mutex.Lock()
	defer s.cachedL1PriceData.mutex.Unlock()

	resetCache := func() {
		s.cachedL1PriceData.startOfL1PriceDataCache = msgIdx
		s.cachedL1PriceData.endOfL1PriceDataCache = msgIdx
		s.cachedL1PriceData.msgToL1PriceData = []L1PriceDataOfMsg{{
			callDataUnits:            callDataUnits,
			cummulativeCallDataUnits: callDataUnits,
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
	if msgIdx != s.cachedL1PriceData.endOfL1PriceDataCache+1 {
		if msgIdx > s.cachedL1PriceData.endOfL1PriceDataCache+1 {
			log.Info("message position higher then current end of l1 price data cache, resetting cache to this message")
			resetCache()
		} else if msgIdx < s.cachedL1PriceData.startOfL1PriceDataCache {
			log.Info("message position lower than start of l1 price data cache, ignoring")
		} else {
			log.Info("message position already seen in l1 price data cache, ignoring")
		}
	} else {
		cummulativeCallDataUnits := s.cachedL1PriceData.msgToL1PriceData[size-1].cummulativeCallDataUnits
		s.cachedL1PriceData.msgToL1PriceData = append(s.cachedL1PriceData.msgToL1PriceData, L1PriceDataOfMsg{
			callDataUnits:            callDataUnits,
			cummulativeCallDataUnits: cummulativeCallDataUnits + callDataUnits,
		})
		s.cachedL1PriceData.endOfL1PriceDataCache = msgIdx
	}
}

// DigestMessage is used to create a block by executing msg against the latest state and storing it.
// Also, while creating a block by executing msg against the latest state,
// in parallel, creates a block by executing msgForPrefetch (msg+1) against the latest state
// but does not store the block.
// This helps in filling the cache, so that the next block creation is faster.
func (s *ExecutionEngine) DigestMessage(msgIdx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) (*execution.MessageResult, error) {
	if !s.createBlocksMutex.TryLock() {
		return nil, errors.New("createBlock mutex held")
	}
	defer s.createBlocksMutex.Unlock()
	return s.digestMessageWithBlockMutex(msgIdx, msg, msgForPrefetch)
}

func (s *ExecutionEngine) digestMessageWithBlockMutex(msgIdxToDigest arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) (*execution.MessageResult, error) {
	currentHeader, err := s.getCurrentHeader()
	if err != nil {
		return nil, err
	}
	curMsgIdx, err := s.BlockNumberToMessageIndex(currentHeader.Number.Uint64())
	if err != nil {
		return nil, err
	}
	if curMsgIdx+1 != msgIdxToDigest {
		return nil, fmt.Errorf("wrong message number in digest got %d expected %d", msgIdxToDigest, curMsgIdx+1)
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
	blockCalcTime := time.Since(startTime)
	blockExecutionTimer.Update(blockCalcTime.Nanoseconds())

	err = s.appendBlock(block, statedb, receipts, blockCalcTime)
	if err != nil {
		return nil, err
	}
	s.cacheL1PriceDataOfMsg(msgIdxToDigest, block, false)

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
		logLevel := log.Warn
		if timeUntilUpgrade < time.Hour*24 {
			logLevel = log.Error
		}
		if version > params.MaxArbosVersionSupported {
			logLevel(
				"you need to update your node to the latest version before this scheduled ArbOS upgrade",
				"timeUntilUpgrade", timeUntilUpgrade,
				"upgradeScheduledFor", timestamp,
				"maxSupportedArbosVersion", params.MaxArbosVersionSupported,
				"pendingArbosUpgradeVersion", version,
			)
		}
	}

	sharedmetrics.UpdateSequenceNumberInBlockGauge(msgIdxToDigest)
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

func (s *ExecutionEngine) ArbOSVersionForMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	block := s.bc.GetBlockByNumber(s.MessageIndexToBlockNumber(msgIdx))
	if block == nil {
		return containers.NewReadyPromise(uint64(0), fmt.Errorf("couldn't find block for message index %d", msgIdx))
	}
	extra := types.DeserializeHeaderExtraInformation(block.Header())
	return containers.NewReadyPromise(extra.ArbOSFormatVersion, nil)
}

func (s *ExecutionEngine) Start(ctx_in context.Context) {
	s.StopWaiter.Start(ctx_in, s)

	s.LaunchThread(func(ctx context.Context) {
		for {
			if s.syncTillBlock > 0 && s.latestBlock != nil && s.latestBlock.NumberU64() >= s.syncTillBlock {
				log.Info("stopping block creation in execution engine", "syncTillBlock", s.syncTillBlock)
				return
			}
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
	if !s.disableStylusCacheMetricsCollection {
		// periodically update stylus cache metrics
		s.LaunchThread(func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Minute):
					programs.UpdateWasmCacheMetrics()
				}
			}
		})
	}
}

func (s *ExecutionEngine) ShouldTriggerMaintenance(trieLimitBeforeFlushMaintenance time.Duration) bool {
	if s.runningMaintenance.Load() {
		return false
	}

	procTimeBeforeFlush, err := s.bc.ProcTimeBeforeFlush()
	if err != nil {
		log.Error("failed to get time before flush", "err")
		return false
	}

	if procTimeBeforeFlush <= trieLimitBeforeFlushMaintenance/2 {
		log.Warn("Time before flush is too low, maintenance should be triggered soon", "procTimeBeforeFlush", procTimeBeforeFlush)
	}
	return procTimeBeforeFlush <= trieLimitBeforeFlushMaintenance
}

func (s *ExecutionEngine) TriggerMaintenance(capLimit uint64) {
	if s.runningMaintenance.Swap(true) {
		log.Info("Maintenance already running, skipping")
		return
	}

	// Flushing the trie DB can be a long operation, so we run it in a new thread
	s.LaunchThread(func(ctx context.Context) {
		defer s.runningMaintenance.Store(false)

		s.createBlocksMutex.Lock()
		defer s.createBlocksMutex.Unlock()

		log.Info("Flushing trie db through maintenance, it can take a while")
		err := s.bc.FlushTrieDB(common.StorageSize(capLimit))
		if err != nil {
			log.Error("Failed to flush trie db through maintenance", "err", err)
		} else {
			log.Info("Flushed trie db through maintenance completed successfully")
		}
	})
}

func (s *ExecutionEngine) MaintenanceStatus() *execution.MaintenanceStatus {
	return &execution.MaintenanceStatus{
		IsRunning: s.runningMaintenance.Load(),
	}
}
