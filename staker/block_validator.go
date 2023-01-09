// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package staker

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
)

type BlockValidator struct {
	stopwaiter.StopWaiter
	*StatelessBlockValidator

	validations      sync.Map
	sequencerBatches sync.Map
	blockMutex       sync.Mutex
	batchMutex       sync.Mutex
	reorgMutex       sync.Mutex
	reorgsPending    int32 // atomic

	lastBlockValidated      uint64      // both atomic and behind lastBlockValidatedMutex
	lastBlockValidatedHash  common.Hash // behind lastBlockValidatedMutex
	lastBlockValidatedMutex sync.Mutex
	earliestBatchKept       uint64
	nextBatchKept           uint64 // 1 + the last batch number kept

	nextBlockToValidate       uint64
	lastValidationEntryBlock  uint64 // used to delete entries in reorg, protected by blockMutex
	lastBlockValidatedUnknown bool
	globalPosNextSend         GlobalStatePosition

	config                   BlockValidatorConfigFetcher
	atomicValidationsRunning int32

	sendValidationsChan chan struct{}
	checkProgressChan   chan struct{}
	progressChan        chan uint64

	lastHeaderForPrepareState *types.Header

	// recentValid holds one recently valid header, to commit it to DB on shutdown
	recentValidMutex   sync.Mutex
	awaitingValidation *types.Header
	validHeader        *types.Header

	fatalErr chan<- error
}

type BlockValidatorConfig struct {
	Enable                   bool                          `koanf:"enable"`
	ArbitratorValidator      bool                          `koanf:"arbitrator-validator"`
	JitValidator             bool                          `koanf:"jit-validator"`
	JitValidatorCranelift    bool                          `koanf:"jit-validator-cranelift"`
	OutputPath               string                        `koanf:"output-path" reload:"hot"`
	ConcurrentRunsLimit      int                           `koanf:"concurrent-runs-limit" reload:"hot"`
	PrerecordedBlocks        uint64                        `koanf:"prerecorded-blocks" reload:"hot"`
	ForwardBlocks            uint64                        `koanf:"forward-blocks" reload:"hot"`
	CurrentModuleRoot        string                        `koanf:"current-module-root"`         // TODO(magic) requires reinitialization on hot reload
	PendingUpgradeModuleRoot string                        `koanf:"pending-upgrade-module-root"` // TODO(magic) requires StatelessBlockValidator recreation on hot reload
	FailureIsFatal           bool                          `koanf:"failure-is-fatal" reload:"hot"`
	Dangerous                BlockValidatorDangerousConfig `koanf:"dangerous"`
}

type BlockValidatorDangerousConfig struct {
	ResetBlockValidation bool `koanf:"reset-block-validation"`
}

type BlockValidatorConfigFetcher func() *BlockValidatorConfig

func BlockValidatorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBlockValidatorConfig.Enable, "enable block-by-block validation")
	f.Bool(prefix+".arbitrator-validator", DefaultBlockValidatorConfig.ArbitratorValidator, "enable the complete, arbitrator block validator")
	f.Bool(prefix+".jit-validator", DefaultBlockValidatorConfig.JitValidator, "enable the faster, jit-accelerated block validator")
	f.Bool(prefix+".jit-validator-cranelift", DefaultBlockValidatorConfig.JitValidatorCranelift, "use Cranelift instead of LLVM when validating blocks using the jit-accelerated block validator")
	f.String(prefix+".output-path", DefaultBlockValidatorConfig.OutputPath, "")
	f.Int(prefix+".concurrent-runs-limit", DefaultBlockValidatorConfig.ConcurrentRunsLimit, "")
	f.Uint64(prefix+".forward-blocks", DefaultBlockValidatorConfig.ForwardBlocks, "prepare entries for up to that many blocks ahead of validation (small footprint)")
	f.Uint64(prefix+".prerecorded-blocks", DefaultBlockValidatorConfig.PrerecordedBlocks, "record that many blocks ahead of validation (larger footprint)")
	f.String(prefix+".current-module-root", DefaultBlockValidatorConfig.CurrentModuleRoot, "current wasm module root ('current' read from chain, 'latest' from machines/latest dir, or provide hash)")
	f.String(prefix+".pending-upgrade-module-root", DefaultBlockValidatorConfig.PendingUpgradeModuleRoot, "pending upgrade wasm module root to additionally validate (hash, 'latest' or empty)")
	f.Bool(prefix+".failure-is-fatal", DefaultBlockValidatorConfig.FailureIsFatal, "failing a validation is treated as a fatal error")
	BlockValidatorDangerousConfigAddOptions(prefix+".dangerous", f)
}

func BlockValidatorDangerousConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".reset-block-validation", DefaultBlockValidatorDangerousConfig.ResetBlockValidation, "resets block-by-block validation, starting again at genesis")
}

var DefaultBlockValidatorConfig = BlockValidatorConfig{
	Enable:                   false,
	ArbitratorValidator:      false,
	JitValidator:             true,
	JitValidatorCranelift:    true,
	OutputPath:               "./target/output",
	ConcurrentRunsLimit:      0,
	ForwardBlocks:            1024,
	PrerecordedBlocks:        128,
	CurrentModuleRoot:        "current",
	PendingUpgradeModuleRoot: "latest",
	FailureIsFatal:           true,
	Dangerous:                DefaultBlockValidatorDangerousConfig,
}

var TestBlockValidatorConfig = BlockValidatorConfig{
	Enable:                   false,
	ArbitratorValidator:      false,
	JitValidator:             false,
	JitValidatorCranelift:    true,
	OutputPath:               "./target/output",
	ConcurrentRunsLimit:      0,
	ForwardBlocks:            128,
	PrerecordedBlocks:        64,
	CurrentModuleRoot:        "latest",
	PendingUpgradeModuleRoot: "latest",
	FailureIsFatal:           true,
	Dangerous:                DefaultBlockValidatorDangerousConfig,
}

var DefaultBlockValidatorDangerousConfig = BlockValidatorDangerousConfig{
	ResetBlockValidation: false,
}

type valStatusField uint32

const (
	Unprepared valStatusField = iota
	RecordSent
	RecordFailed
	Prepared
	Failed
	Valid
)

type validationStatus struct {
	Status      uint32           // atomic: value is one of validationStatus*
	Cancel      func()           // non-atomic: only read/written to with reorg mutex
	Entry       *validationEntry // non-atomic: only read if Status >= validationStatusPrepared
	ModuleRoots []common.Hash    // non-atomic: present from the start
}

func (s *validationStatus) setStatus(val valStatusField) {
	atomic.StoreUint32(&s.Status, uint32(val))
}

func (s *validationStatus) getStatus() valStatusField {
	uintStat := atomic.LoadUint32(&s.Status)
	return valStatusField(uintStat)
}

func (s *validationStatus) replaceStatus(old, new valStatusField) bool {
	return atomic.CompareAndSwapUint32(&s.Status, uint32(old), uint32(new))
}

func NewBlockValidator(
	statelessBlockValidator *StatelessBlockValidator,
	inbox InboxTrackerInterface,
	streamer TransactionStreamerInterface,
	reorgingToBlock *types.Block,
	config BlockValidatorConfigFetcher,
	fatalErr chan<- error,
) (*BlockValidator, error) {
	validator := &BlockValidator{
		StatelessBlockValidator: statelessBlockValidator,
		sendValidationsChan:     make(chan struct{}, 1),
		checkProgressChan:       make(chan struct{}, 1),
		progressChan:            make(chan uint64, 1),
		config:                  config,
		fatalErr:                fatalErr,
	}
	err := validator.readLastBlockValidatedDbInfo(reorgingToBlock)
	if err != nil {
		return nil, err
	}
	streamer.SetBlockValidator(validator)
	inbox.SetBlockValidator(validator)
	return validator, nil
}

func (v *BlockValidator) possiblyFatal(err error) {
	if v.Stopped() {
		return
	}
	if err == nil {
		return
	}
	log.Error("Error during validation", "err", err)
	if v.config().FailureIsFatal {
		select {
		case v.fatalErr <- err:
		default:
		}
	}
}

func (v *BlockValidator) triggerSendValidations() {
	select {
	case v.sendValidationsChan <- struct{}{}:
	default:
	}
}

func (v *BlockValidator) recentlyValid(header *types.Header) {
	v.recentValidMutex.Lock()
	defer v.recentValidMutex.Unlock()
	if v.awaitingValidation == nil {
		return
	}
	if v.awaitingValidation.Number.Cmp(header.Number) > 0 {
		return
	}
	if v.validHeader != nil {
		v.recordingDatabase.Dereference(v.validHeader)
	}
	v.validHeader = v.awaitingValidation
	v.awaitingValidation = nil
}

func (v *BlockValidator) recentStateComputed(header *types.Header) {
	v.recentValidMutex.Lock()
	defer v.recentValidMutex.Unlock()
	if v.awaitingValidation != nil {
		return
	}
	_, err := v.recordingDatabase.StateFor(header)
	if err != nil {
		log.Error("failed to get state for block while validating", "err", err, "blockNum", header.Number, "hash", header.Hash())
		return
	}
	v.awaitingValidation = header
}

func (v *BlockValidator) recentShutdown() error {
	v.recentValidMutex.Lock()
	defer v.recentValidMutex.Unlock()
	if v.validHeader == nil {
		return nil
	}
	err := v.recordingDatabase.WriteStateToDatabase(v.validHeader)
	v.recordingDatabase.Dereference(v.validHeader)
	return err
}

func (v *BlockValidator) readLastBlockValidatedDbInfo(reorgingToBlock *types.Block) error {
	v.lastBlockValidatedMutex.Lock()
	defer v.lastBlockValidatedMutex.Unlock()

	exists, err := v.db.Has(lastBlockValidatedInfoKey)
	if err != nil {
		return err
	}

	if !exists || v.config().Dangerous.ResetBlockValidation {
		// The db contains no validation info; start from the beginning.
		// TODO: this skips validating the genesis block.
		atomic.StoreUint64(&v.lastBlockValidated, v.genesisBlockNum)
		genesisBlock := v.blockchain.GetBlockByNumber(v.genesisBlockNum)
		if genesisBlock == nil {
			return fmt.Errorf("blockchain missing genesis block number %v", v.genesisBlockNum)
		}
		v.lastBlockValidatedHash = genesisBlock.Hash()
		v.nextBlockToValidate = v.genesisBlockNum + 1
		v.globalPosNextSend = GlobalStatePosition{
			BatchNumber: 1,
			PosInBatch:  0,
		}
		return nil
	}

	infoBytes, err := v.db.Get(lastBlockValidatedInfoKey)
	if err != nil {
		return err
	}

	var info lastBlockValidatedDbInfo
	err = rlp.DecodeBytes(infoBytes, &info)
	if err != nil {
		return err
	}

	if reorgingToBlock != nil && reorgingToBlock.NumberU64() >= info.BlockNumber {
		// Disregard this reorg as it doesn't affect the last validated block
		reorgingToBlock = nil
	}

	if reorgingToBlock == nil {
		expectedHash := v.blockchain.GetCanonicalHash(info.BlockNumber)
		if expectedHash != info.BlockHash && (expectedHash != common.Hash{}) {
			return fmt.Errorf("last validated block %v stored with hash %v, but blockchain has hash %v", info.BlockNumber, info.BlockHash, expectedHash)
		}
	}

	atomic.StoreUint64(&v.lastBlockValidated, info.BlockNumber)
	v.lastBlockValidatedHash = info.BlockHash
	v.nextBlockToValidate = v.lastBlockValidated + 1
	v.globalPosNextSend = info.AfterPosition

	if reorgingToBlock != nil {
		err = v.reorgToBlockImpl(reorgingToBlock.NumberU64(), reorgingToBlock.Hash(), true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (v *BlockValidator) sendRecord(s *validationStatus, mustDeref bool) error {
	if !v.Started() {
		// this could only be sent by NewBlock, so mustDeref is not sent
		return nil
	}
	prevHeader := s.Entry.PrevBlockHeader
	if !s.replaceStatus(Unprepared, RecordSent) {
		if mustDeref {
			v.recordingDatabase.Dereference(prevHeader)
		}
		return errors.Errorf("failed status check for send record. Status: %v", s.getStatus())
	}
	v.LaunchThread(func(ctx context.Context) {
		if mustDeref {
			defer v.recordingDatabase.Dereference(prevHeader)
		}
		err := v.ValidationEntryRecord(ctx, s.Entry, true)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			s.replaceStatus(RecordSent, RecordFailed) // after that - could be removed from validations map
			log.Error("Error while recording", "err", err, "status", s.getStatus())
			return
		}
		v.recentStateComputed(prevHeader)
		v.recordingDatabase.Dereference(prevHeader) // removes the reference added by ValidationEntryRecord
		if !s.replaceStatus(RecordSent, Prepared) {
			log.Error("Fault trying to update validation with recording", "entry", s.Entry, "status", s.getStatus())
			return
		}
		v.triggerSendValidations()
	})
	return nil
}

func (v *BlockValidator) newValidationStatus(prevHeader, header *types.Header, msg *arbstate.MessageWithMetadata) (*validationStatus, error) {
	entry, err := newValidationEntry(prevHeader, header, msg)
	if err != nil {
		return nil, err
	}
	status := &validationStatus{
		Status:      uint32(Unprepared),
		Entry:       entry,
		ModuleRoots: v.GetModuleRootsToValidate(),
	}
	return status, nil
}

func (v *BlockValidator) NewBlock(block *types.Block, prevHeader *types.Header, msg arbstate.MessageWithMetadata) {
	v.blockMutex.Lock()
	defer v.blockMutex.Unlock()
	blockNum := block.NumberU64()
	if blockNum < v.lastBlockValidated {
		return
	}
	if v.lastBlockValidatedUnknown {
		if block.Hash() == v.lastBlockValidatedHash {
			v.lastBlockValidated = blockNum
			v.nextBlockToValidate = blockNum + 1
			v.lastBlockValidatedUnknown = false
			log.Info("Block building caught up to staker", "blockNr", v.lastBlockValidated, "blockHash", v.lastBlockValidatedHash)
			// note: this block is already valid
		}
		return
	}
	if v.nextBlockToValidate+v.config().ForwardBlocks <= blockNum {
		return
	}
	status, err := v.newValidationStatus(prevHeader, block.Header(), &msg)
	if err != nil {
		log.Error("failed creating validation status", "err", err)
		return
	}
	// It's fine to separately load and then store as we have the blockMutex acquired
	_, present := v.validations.Load(blockNum)
	if present {
		return
	}
	if v.nextBlockToValidate+v.config().PrerecordedBlocks > blockNum {
		err := v.sendRecord(status, false)
		if err != nil {
			log.Error("failed send recording for new block", "err", err)
		}
	}
	v.validations.Store(blockNum, status)
	if v.lastValidationEntryBlock < blockNum {
		v.lastValidationEntryBlock = blockNum
	}
	v.triggerSendValidations()
}

//nolint:gosec
func (v *BlockValidator) writeToFile(validationEntry *validationEntry, moduleRoot common.Hash, sequencerMsg []byte) error {
	input, err := validationEntry.ToInput()
	if err != nil {
		return err
	}
	expOut, err := validationEntry.expectedEnd()
	if err != nil {
		return err
	}
	return v.validationSpawner.WriteToFile(v.config().OutputPath, input, expOut, moduleRoot, sequencerMsg)
}

func (v *BlockValidator) SetCurrentWasmModuleRoot(hash common.Hash) error {
	v.blockMutex.Lock()
	v.moduleMutex.Lock()
	defer v.blockMutex.Unlock()
	defer v.moduleMutex.Unlock()

	if (hash == common.Hash{}) {
		return errors.New("trying to set zero as wsmModuleRoot")
	}
	if hash == v.currentWasmModuleRoot {
		return nil
	}
	if (v.currentWasmModuleRoot == common.Hash{}) {
		v.currentWasmModuleRoot = hash
		return nil
	}
	if v.pendingWasmModuleRoot == hash {
		log.Info("Block validator: detected progressing to pending machine", "hash", hash)
		v.currentWasmModuleRoot = hash
		return nil
	}
	if v.config().CurrentModuleRoot != "current" {
		return nil
	}
	return fmt.Errorf(
		"unexpected wasmModuleRoot! cannot validate! found %v , current %v, pending %v",
		hash, v.currentWasmModuleRoot, v.pendingWasmModuleRoot,
	)
}

var ErrValidationCanceled = errors.New("validation of block cancelled")

func (v *BlockValidator) validate(ctx context.Context, validationStatus *validationStatus, seqMsg []byte) {
	if currentStatus := validationStatus.getStatus(); currentStatus != Prepared {
		log.Error("attempted to validate unprepared validation entry", "status", currentStatus)
		return
	}
	entry := validationStatus.Entry
	defer func() {
		atomic.AddInt32(&v.atomicValidationsRunning, -1)
		v.triggerSendValidations()
	}()
	entry.BatchInfo = append(entry.BatchInfo, validator.BatchInfo{
		Number: entry.StartPosition.BatchNumber,
		Data:   seqMsg,
	})
	log.Debug(
		"starting validation for block", "blockNr", entry.BlockNumber,
		"blockAge", common.PrettyAge(time.Unix(int64(entry.BlockHeader.Time), 0)),
		"blockDate", time.Unix(int64(entry.BlockHeader.Time), 0))
	for _, moduleRoot := range validationStatus.ModuleRoots {

		type replay = func(context.Context, *validator.ValidationInput, common.Hash) (validator.GoGlobalState, error)

		execValidation := func(replay replay, validationType string) error {
			input, err := entry.ToInput()
			if err != nil {
				return err
			}
			gsEnd, err := replay(ctx, input, moduleRoot)
			if err != nil {
				canceled := ctx.Err() != nil
				if canceled {
					return fmt.Errorf("%w: blockNr: %v, hash: %v, validationType: %v",
						ErrValidationCanceled, entry.BlockNumber, entry.BlockHash, validationType)
				}
				return fmt.Errorf("validation of block failed. blockNr: %v, hash: %v, validationType: %v err: %w",
					entry.BlockNumber, entry.BlockHash, validationType, err)
			}

			var gsExpected validator.GoGlobalState
			gsExpected, err = entry.expectedEnd()
			if err != nil || gsEnd != gsExpected {
				return fmt.Errorf("validation of block failed. moduleRoot: %v got: %v expected: %v expectedHeader: %v, validationType: %v, gsErr: %w",
					moduleRoot, gsEnd, gsExpected, entry.BlockHeader, validationType, err)
			}

			return nil
		}

		before := time.Now()

		config := v.config()
		var valError error
		if config.ArbitratorValidator {
			valError = execValidation(v.validationSpawner.ExecuteArbitrator, "arbitrator")
		}
		if config.JitValidator && valError == nil {
			valError = execValidation(v.validationSpawner.ExecuteJit, "jit")
		}
		if valError != nil {
			if errors.Is(valError, ErrValidationCanceled) {
				log.Info("validation cancelled", "info", valError)
			} else {
				err := v.writeToFile(
					entry, moduleRoot, seqMsg,
				)
				if err != nil {
					log.Error("failed to write file", "err", err)
				}
				v.possiblyFatal(valError)
			}
			validationStatus.setStatus(Failed)
			return
		}

		log.Debug(
			"validation succeeded", "blockNr", entry.BlockNumber,
			"blockAge", common.PrettyAge(time.Unix(int64(entry.BlockHeader.Time), 0)),
			"blockDate", time.Unix(int64(entry.BlockHeader.Time), 0),
			"blockHash", entry.BlockHash, "moduleRoot", moduleRoot, "time", time.Since(before),
		)
	}

	validationStatus.setStatus(Valid) // after that - validation entry could be deleted from map

	select {
	case v.checkProgressChan <- struct{}{}:
	default:
	}
}

func (v *BlockValidator) sendValidations(ctx context.Context) {
	v.reorgMutex.Lock()
	defer v.reorgMutex.Unlock()
	concurrentRunsLimit := (int32)(v.config().ConcurrentRunsLimit)
	if concurrentRunsLimit == 0 {
		concurrentRunsLimit = (int32)(runtime.NumCPU())
	}
	var batchCount uint64
	for atomic.LoadInt32(&v.reorgsPending) == 0 {
		if atomic.LoadInt32(&v.atomicValidationsRunning) >= concurrentRunsLimit {
			return
		}
		if batchCount <= v.globalPosNextSend.BatchNumber {
			var err error
			batchCount, err = v.inboxTracker.GetBatchCount()
			if err != nil {
				log.Error("validator failed to get message count", "err", err)
				return
			}
			if batchCount <= v.globalPosNextSend.BatchNumber {
				return
			}
		}
		seqBatchEntry, haveBatch := v.sequencerBatches.Load(v.globalPosNextSend.BatchNumber)
		if !haveBatch {
			if batchCount == v.globalPosNextSend.BatchNumber+1 {
				// This is the latest batch.
				// To avoid re-querying it unnecessarily, wait for the inbox tracker to provide it to us.
				return
			}
			seqMsg, err := v.inboxReader.GetSequencerMessageBytes(ctx, v.globalPosNextSend.BatchNumber)
			if err != nil {
				log.Error("validator failed to read sequencer message", "err", err)
				return
			}
			v.ProcessBatches(v.globalPosNextSend.BatchNumber, [][]byte{seqMsg})
			seqBatchEntry = seqMsg
		}
		v.blockMutex.Lock()
		if v.lastBlockValidatedUnknown {
			firstMsgInBatch := arbutil.MessageIndex(0)
			if v.globalPosNextSend.BatchNumber > 0 {
				var err error
				firstMsgInBatch, err = v.inboxTracker.GetBatchMessageCount(v.globalPosNextSend.BatchNumber - 1)
				if err != nil {
					v.blockMutex.Unlock()
					log.Error("validator couldnt read message count", "err", err)
					return
				}
			}
			v.lastBlockValidated = uint64(arbutil.MessageCountToBlockNumber(firstMsgInBatch+arbutil.MessageIndex(v.globalPosNextSend.PosInBatch), v.genesisBlockNum))
			v.nextBlockToValidate = v.lastBlockValidated + 1
			v.lastBlockValidatedUnknown = false
			log.Info("Inbox caught up to staker", "blockNr", v.lastBlockValidated, "blockHash", v.lastBlockValidatedHash)
		}
		v.blockMutex.Unlock()
		nextMsg := arbutil.BlockNumberToMessageCount(v.nextBlockToValidate, v.genesisBlockNum) - 1
		// valdationEntries is By blockNumber
		entry, found := v.validations.Load(v.nextBlockToValidate)
		if !found {
			return
		}
		validationStatus, ok := entry.(*validationStatus)
		if !ok || (validationStatus == nil) {
			log.Error("bad entry trying to validate batch")
			return
		}
		if validationStatus.getStatus() < Prepared {
			return
		}
		startPos, endPos, err := GlobalStatePositionsFor(v.inboxTracker, nextMsg, v.globalPosNextSend.BatchNumber)
		if err != nil {
			log.Error("failed calculating position for validation", "err", err, "msg", nextMsg, "batch", v.globalPosNextSend.BatchNumber)
			return
		}
		if startPos != v.globalPosNextSend {
			log.Error("inconsistent pos mapping", "msg", nextMsg, "expected", v.globalPosNextSend, "found", startPos)
			return
		}
		atomic.AddInt32(&v.atomicValidationsRunning, 1)

		seqMsg, ok := seqBatchEntry.([]byte)
		if !ok {
			batchNum := validationStatus.Entry.StartPosition.BatchNumber
			log.Error("sequencer message bad format", "blockNr", v.nextBlockToValidate, "msgNum", batchNum)
			return
		}
		v.LaunchThread(func(ctx context.Context) {
			validationCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			validationStatus.Cancel = cancel
			err := v.ValidationEntryAddSeqMessage(ctx, validationStatus.Entry, startPos, endPos, seqMsg)
			if err != nil && validationCtx.Err() == nil {
				log.Error("error preparing validation", "err", err)
				return
			}
			v.validate(validationCtx, validationStatus, seqMsg)
		})

		v.nextBlockToValidate++
		v.globalPosNextSend = endPos
	}
}

func (v *BlockValidator) sendRecords(ctx context.Context) {
	v.reorgMutex.Lock()
	defer v.reorgMutex.Unlock()
	nextRecord := v.nextBlockToValidate
	for atomic.LoadInt32(&v.reorgsPending) == 0 {
		if nextRecord >= v.nextBlockToValidate+v.config().PrerecordedBlocks {
			return
		}
		entry, found := v.validations.Load(nextRecord)
		if !found {
			header := v.blockchain.GetHeaderByNumber(nextRecord)
			if header == nil {
				// This block hasn't been created yet.
				return
			}
			prevHeader := v.blockchain.GetHeaderByHash(header.ParentHash)
			if prevHeader == nil && header.ParentHash != (common.Hash{}) {
				log.Warn("failed to get prevHeader in block validator", "num", nextRecord-1, "hash", header.ParentHash)
				return
			}
			msgNum := arbutil.BlockNumberToMessageCount(nextRecord, v.genesisBlockNum) - 1
			msg, err := v.streamer.GetMessage(msgNum)
			if err != nil {
				log.Warn("failed to get message in block validator", "err", err)
				return
			}
			status, err := v.newValidationStatus(prevHeader, header, msg)
			if err != nil {
				log.Warn("failed to create validation status", "err", err)
				return
			}
			v.blockMutex.Lock()
			entry, found = v.validations.Load(nextRecord)
			if !found {
				v.validations.Store(nextRecord, status)
				entry = status
			}
			v.blockMutex.Unlock()
		}
		validationStatus, ok := entry.(*validationStatus)
		if !ok || (validationStatus == nil) {
			log.Error("bad entry trying to send recordings")
			return
		}
		currentStatus := validationStatus.getStatus()
		if currentStatus == RecordFailed {
			// retry
			v.validations.Delete(nextRecord)
			v.triggerSendValidations()
			return
		}
		if currentStatus == Unprepared {
			prevHeader := validationStatus.Entry.PrevBlockHeader
			if prevHeader != nil {
				_, err := v.recordingDatabase.GetOrRecreateState(ctx, prevHeader, stateLogFunc)
				if err != nil {
					log.Error("error trying to prepare state for recording", "err", err)
				}
				// add another reference that will be released by the record thread
				_, err = v.recordingDatabase.StateFor(prevHeader)
				if err != nil {
					log.Error("error trying re-reference state for recording", "err", err)
				}
				if v.lastHeaderForPrepareState != nil {
					v.recordingDatabase.Dereference(v.lastHeaderForPrepareState)
				}
				v.lastHeaderForPrepareState = prevHeader
			}
			err := v.sendRecord(validationStatus, true)
			if err != nil {
				log.Error("error trying to send preimage recording", "err", err)
			}
		}
		nextRecord++
	}
}

func (v *BlockValidator) writeLastValidatedToDb(blockNumber uint64, blockHash common.Hash, endPos GlobalStatePosition) error {
	info := lastBlockValidatedDbInfo{
		BlockNumber:   blockNumber,
		BlockHash:     blockHash,
		AfterPosition: endPos,
	}
	encodedInfo, err := rlp.EncodeToBytes(info)
	if err != nil {
		return err
	}
	err = v.db.Put(lastBlockValidatedInfoKey, encodedInfo)
	if err != nil {
		return err
	}
	return nil
}

func (v *BlockValidator) progressValidated() {
	v.reorgMutex.Lock()
	defer v.reorgMutex.Unlock()
	for atomic.LoadInt32(&v.reorgsPending) == 0 {
		// Reads from blocksValidated can be non-atomic as all writes hold reorgMutex
		checkingBlock := v.lastBlockValidated + 1
		entry, found := v.validations.Load(checkingBlock)
		if !found {
			return
		}
		validationStatus, ok := entry.(*validationStatus)
		if !ok || (validationStatus == nil) {
			log.Error("bad entry trying to advance validated counter")
			return
		}
		if validationStatus.getStatus() < Valid {
			return
		}
		validationEntry := validationStatus.Entry
		if validationEntry.BlockNumber != checkingBlock {
			log.Error("bad block number for validation entry", "expected", checkingBlock, "found", validationEntry.BlockNumber)
			return
		}
		// It's safe to read lastBlockValidatedHash without the lastBlockValidatedMutex as we have the reorgMutex
		if v.lastBlockValidatedHash != validationEntry.PrevBlockHash {
			log.Error("lastBlockValidatedHash is %v but validationEntry has prevBlockHash %v for block number %v", v.lastBlockValidatedHash, validationEntry.PrevBlockHash, v.lastBlockValidated)
			return
		}
		earliestBatchKept := atomic.LoadUint64(&v.earliestBatchKept)
		seqMsgNr := validationEntry.StartPosition.BatchNumber
		if earliestBatchKept < seqMsgNr {
			for batch := earliestBatchKept; batch < seqMsgNr; batch++ {
				v.sequencerBatches.Delete(batch)
			}
			atomic.StoreUint64(&v.earliestBatchKept, seqMsgNr)
		}

		v.lastBlockValidatedMutex.Lock()
		atomic.StoreUint64(&v.lastBlockValidated, checkingBlock)
		v.lastBlockValidatedHash = validationEntry.BlockHash
		v.lastBlockValidatedMutex.Unlock()
		v.recentlyValid(validationEntry.BlockHeader)

		v.validations.Delete(checkingBlock)
		select {
		case v.progressChan <- checkingBlock:
		default:
		}
		err := v.writeLastValidatedToDb(validationEntry.BlockNumber, validationEntry.BlockHash, validationEntry.EndPosition)
		if err != nil {
			log.Error("failed to write validated entry to database", "err", err)
		}
	}
}

func (v *BlockValidator) AssumeValid(globalState validator.GoGlobalState) error {
	if v.Started() {
		return errors.Errorf("cannot handle AssumeValid while running")
	}
	v.lastBlockValidatedMutex.Lock()
	defer v.lastBlockValidatedMutex.Unlock()

	// don't do anything if we already validated past that
	if v.globalPosNextSend.BatchNumber > globalState.Batch {
		return nil
	}
	if v.globalPosNextSend.BatchNumber == globalState.Batch && v.globalPosNextSend.PosInBatch > globalState.PosInBatch {
		return nil
	}

	block := v.blockchain.GetBlockByHash(globalState.BlockHash)
	if block == nil {
		v.lastBlockValidatedUnknown = true
	} else {
		v.lastBlockValidated = block.NumberU64()
		v.nextBlockToValidate = v.lastBlockValidated + 1
	}
	v.lastBlockValidatedHash = globalState.BlockHash
	v.globalPosNextSend = GlobalStatePosition{
		BatchNumber: globalState.Batch,
		PosInBatch:  globalState.PosInBatch,
	}
	return nil
}

func (v *BlockValidator) LastBlockValidated() uint64 {
	return atomic.LoadUint64(&v.lastBlockValidated)
}

func (v *BlockValidator) LastBlockValidatedAndHash() (blockNumber uint64, blockHash common.Hash, wasmModuleRoots []common.Hash) {
	v.lastBlockValidatedMutex.Lock()
	blockValidated := v.lastBlockValidated
	blockValidatedHash := v.lastBlockValidatedHash
	v.lastBlockValidatedMutex.Unlock()

	// things can be removed from, but not added to, moduleRootsToValidate. By taking root hashes fter the block we know result is valid
	moduleRootsValidated := v.GetModuleRootsToValidate()

	return blockValidated, blockValidatedHash, moduleRootsValidated
}

// Because batches and blocks are handled at separate layers in the node,
// and because block generation from messages is asynchronous,
// this call is different than ReorgToBlock, which is currently called later.
func (v *BlockValidator) ReorgToBatchCount(count uint64) {
	v.batchMutex.Lock()
	defer v.batchMutex.Unlock()
	v.reorgToBatchCountImpl(count)
}

func (v *BlockValidator) reorgToBatchCountImpl(count uint64) {
	localBatchCount := v.nextBatchKept
	if localBatchCount < count {
		return
	}
	for i := count; i < localBatchCount; i++ {
		v.sequencerBatches.Delete(i)
	}
	v.nextBatchKept = count
}

func (v *BlockValidator) ProcessBatches(pos uint64, batches [][]byte) {
	v.batchMutex.Lock()
	defer v.batchMutex.Unlock()

	v.reorgToBatchCountImpl(pos)

	// Attempt to fill in earliestBatchKept if it's empty
	atomic.CompareAndSwapUint64(&v.earliestBatchKept, 0, pos)

	for i, msg := range batches {
		v.sequencerBatches.Store(pos+uint64(i), msg)
	}
	v.nextBatchKept = pos + uint64(len(batches))
	v.triggerSendValidations()
}

func (v *BlockValidator) ReorgToBlock(blockNum uint64, blockHash common.Hash) error {
	v.blockMutex.Lock()
	defer v.blockMutex.Unlock()

	atomic.AddInt32(&v.reorgsPending, 1)
	v.reorgMutex.Lock()
	defer v.reorgMutex.Unlock()
	atomic.AddInt32(&v.reorgsPending, -1)

	if blockNum < v.lastValidationEntryBlock {
		log.Warn("block validator processing reorg", "blockNum", blockNum)
		err := v.reorgToBlockImpl(blockNum, blockHash, false)
		if err != nil {
			return fmt.Errorf("block validator reorg failed: %w", err)
		}
	}

	return nil
}

func (v *BlockValidator) reorgToBlockImpl(blockNum uint64, blockHash common.Hash, hasLastValidatedMutex bool) error {
	for b := blockNum + 1; b <= v.lastValidationEntryBlock; b++ {
		entry, found := v.validations.Load(b)
		if !found {
			continue
		}
		v.validations.Delete(b)

		validationStatus, ok := entry.(*validationStatus)
		if !ok || (validationStatus == nil) {
			log.Error("bad entry trying to reorg block validator")
			continue
		}
		log.Debug("canceling validation due to reorg", "block", b)
		if validationStatus.Cancel != nil {
			validationStatus.Cancel()
		}
	}
	v.lastValidationEntryBlock = blockNum
	if v.nextBlockToValidate <= blockNum+1 {
		return nil
	}
	msgIndex := arbutil.BlockNumberToMessageCount(blockNum, v.genesisBlockNum) - 1
	batchCount, err := v.inboxTracker.GetBatchCount()
	if err != nil {
		return err
	}
	batch, err := FindBatchContainingMessageIndex(v.inboxTracker, msgIndex, batchCount)
	if err != nil {
		return err
	}
	if batch >= batchCount {
		// This reorg is past the latest batch.
		// Attempt to recover by loading a next validation state at the start of the next batch.
		v.globalPosNextSend = GlobalStatePosition{
			BatchNumber: batch,
			PosInBatch:  0,
		}
		msgCount, err := v.inboxTracker.GetBatchMessageCount(batch - 1)
		if err != nil {
			return err
		}
		nextBlockSigned := arbutil.MessageCountToBlockNumber(msgCount, v.genesisBlockNum) + 1
		if nextBlockSigned <= 0 {
			return errors.New("reorg past genesis block")
		}
		blockNum = uint64(nextBlockSigned) - 1
		block := v.blockchain.GetBlockByNumber(blockNum)
		if block == nil {
			return fmt.Errorf("failed to get end of batch block %v", blockNum)
		}
		blockHash = block.Hash()
		v.lastValidationEntryBlock = blockNum
	} else {
		_, v.globalPosNextSend, err = GlobalStatePositionsFor(v.inboxTracker, msgIndex, batch)
		if err != nil {
			return err
		}
	}
	if v.nextBlockToValidate > blockNum+1 {
		v.nextBlockToValidate = blockNum + 1
	}

	if v.lastBlockValidated > blockNum {
		if !hasLastValidatedMutex {
			v.lastBlockValidatedMutex.Lock()
		}
		atomic.StoreUint64(&v.lastBlockValidated, blockNum)
		v.lastBlockValidatedHash = blockHash
		if !hasLastValidatedMutex {
			v.lastBlockValidatedMutex.Unlock()
		}

		err = v.writeLastValidatedToDb(blockNum, blockHash, v.globalPosNextSend)
		if err != nil {
			return err
		}
	}

	return nil
}

// Initialize must be called after SetCurrentWasmModuleRoot sets the current one
func (v *BlockValidator) Initialize() error {
	config := v.config()
	currentModuleRoot := config.CurrentModuleRoot
	switch currentModuleRoot {
	case "latest":
		latest, err := v.validationSpawner.LatestWasmModuleRoot()
		if err != nil {
			return err
		}
		v.currentWasmModuleRoot = latest
	case "current":
		if (v.currentWasmModuleRoot == common.Hash{}) {
			return errors.New("wasmModuleRoot set to 'current' - but info not set from chain")
		}
	default:
		v.currentWasmModuleRoot = common.HexToHash(currentModuleRoot)
		if (v.currentWasmModuleRoot == common.Hash{}) {
			return errors.New("current-module-root config value illegal")
		}
	}
	log.Info("BlockValidator initialized", "current", v.currentWasmModuleRoot, "pending", v.pendingWasmModuleRoot)
	return nil
}

func (v *BlockValidator) Start(ctxIn context.Context) error {
	v.StopWaiter.Start(ctxIn, v)
	v.LaunchThread(func(ctx context.Context) {
		// `progressValidated` and `sendValidations` should both only do `concurrentRunsLimit` iterations of work,
		// so they won't stomp on each other and prevent the other from running.
		v.sendRecords(ctx)
		v.sendValidations(ctx)
		for {
			select {
			case _, ok := <-v.checkProgressChan:
				if !ok {
					return
				}
				v.progressValidated()
			case _, ok := <-v.sendValidationsChan:
				if !ok {
					return
				}
				v.sendRecords(ctx)
				v.sendValidations(ctx)
			case <-ctx.Done():
				return
			}
		}
	})
	lastValid := uint64(0)
	v.CallIteratively(func(ctx context.Context) time.Duration {
		newValid, validHash, wasmModuleRoots := v.LastBlockValidatedAndHash()
		if newValid != lastValid {
			validHeader := v.blockchain.GetHeader(validHash, newValid)
			if validHeader == nil {
				foundHeader := v.blockchain.GetHeaderByNumber(newValid)
				foundHash := common.Hash{}
				if foundHeader != nil {
					foundHash = foundHeader.Hash()
				}
				log.Warn("last valid block not in blockchain", "blockNum", newValid, "validatedBlockHash", validHash, "found-hash", foundHash)
			} else {
				validTimestamp := time.Unix(int64(validHeader.Time), 0)
				log.Info("Validated blocks", "blockNum", newValid, "hash", validHash,
					"timestamp", validTimestamp, "age", time.Since(validTimestamp), "wasm", wasmModuleRoots)
			}
			lastValid = newValid
		}
		return time.Second
	})
	return nil
}

func (v *BlockValidator) StopAndWait() {
	v.StopWaiter.StopAndWait()
	err := v.recentShutdown()
	if err != nil {
		log.Error("error storing valid state", "err", err)
	}
}

// WaitForBlock can only be used from One thread
func (v *BlockValidator) WaitForBlock(ctx context.Context, blockNumber uint64, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		if atomic.LoadUint64(&v.lastBlockValidated) >= blockNumber {
			return true
		}
		select {
		case <-timer.C:
			if atomic.LoadUint64(&v.lastBlockValidated) >= blockNumber {
				return true
			}
			return false
		case block, ok := <-v.progressChan:
			if block >= blockNumber {
				return true
			}
			if !ok {
				return false
			}
		case <-ctx.Done():
			return false
		}
	}
}
