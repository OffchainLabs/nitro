// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validator

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
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
)

type BlockValidator struct {
	stopwaiter.StopWaiter
	*StatelessBlockValidator

	validationEntries sync.Map
	sequencerBatches  sync.Map
	blockMutex        sync.Mutex
	batchMutex        sync.Mutex
	reorgMutex        sync.Mutex
	reorgsPending     int32 // atomic

	lastBlockValidated      uint64      // both atomic and behind lastBlockValidatedMutex
	lastBlockValidatedHash  common.Hash // behind lastBlockValidatedMutex
	lastBlockValidatedMutex sync.Mutex
	earliestBatchKept       uint64
	nextBatchKept           uint64 // 1 + the last batch number kept

	nextBlockToValidate       uint64
	nextValidationEntryBlock  uint64
	lastBlockValidatedUnknown bool
	globalPosNextSend         GlobalStatePosition

	config                   BlockValidatorConfigFetcher
	atomicValidationsRunning int32

	sendValidationsChan chan struct{}
	checkProgressChan   chan struct{}
	progressChan        chan uint64
}

type BlockValidatorConfig struct {
	Enable                   bool                          `koanf:"enable"`
	ArbitratorValidator      bool                          `koanf:"arbitrator-validator"`
	JitValidator             bool                          `koanf:"jit-validator"`
	JitValidatorCranelift    bool                          `koanf:"jit-validator-cranelift"`
	OutputPath               string                        `koanf:"output-path" reload:"hot"`
	ConcurrentRunsLimit      int                           `koanf:"concurrent-runs-limit" reload:"hot"`
	CurrentModuleRoot        string                        `koanf:"current-module-root"`          // TODO(magic) requires reinitialization on hot reload
	PendingUpgradeModuleRoot string                        `koanf:"pending-upgrade-module-root"`  // TODO(magic) requires StatelessBlockValidator recreation on hot reload
	StorePreimages           bool                          `koanf:"store-preimages" reload:"hot"` // TODO verify if hot reloading is safe
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
	f.String(prefix+".current-module-root", DefaultBlockValidatorConfig.CurrentModuleRoot, "current wasm module root ('current' read from chain, 'latest' from machines/latest dir, or provide hash)")
	f.String(prefix+".pending-upgrade-module-root", DefaultBlockValidatorConfig.PendingUpgradeModuleRoot, "pending upgrade wasm module root to additionally validate (hash, 'latest' or empty)")
	f.Bool(prefix+".store-preimages", DefaultBlockValidatorConfig.StorePreimages, "store preimages of running machines (higher memory cost, better debugging, potentially better performance)")
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
	CurrentModuleRoot:        "current",
	PendingUpgradeModuleRoot: "latest",
	StorePreimages:           false,
	Dangerous:                DefaultBlockValidatorDangerousConfig,
}

var TestBlockValidatorConfig = BlockValidatorConfig{
	Enable:                   false,
	ArbitratorValidator:      false,
	JitValidator:             false,
	JitValidatorCranelift:    true,
	OutputPath:               "./target/output",
	ConcurrentRunsLimit:      0,
	CurrentModuleRoot:        "latest",
	PendingUpgradeModuleRoot: "latest",
	StorePreimages:           false,
	Dangerous:                DefaultBlockValidatorDangerousConfig,
}

var DefaultBlockValidatorDangerousConfig = BlockValidatorDangerousConfig{
	ResetBlockValidation: false,
}

const validationStatusUnprepared uint32 = 0 // waiting for validationEntry to be populated
const validationStatusPrepared uint32 = 1   // ready to undergo validation
const validationStatusFailed uint32 = 2     // validation failed
const validationStatusValid uint32 = 3      // validation succeeded

type validationStatus struct {
	Status      uint32           // atomic: value is one of validationStatus*
	Cancel      func()           // non-atomic: only read/written to with reorg mutex
	Entry       *validationEntry // non-atomic: only read if Status >= validationStatusPrepared
	ModuleRoots []common.Hash    // non-atomic: present from the start
}

func NewBlockValidator(
	statelessBlockValidator *StatelessBlockValidator,
	inbox InboxTrackerInterface,
	streamer TransactionStreamerInterface,
	machineLoader *NitroMachineLoader,
	reorgingToBlock *types.Block,
	config BlockValidatorConfigFetcher,
) (*BlockValidator, error) {
	validator := &BlockValidator{
		StatelessBlockValidator: statelessBlockValidator,
		sendValidationsChan:     make(chan struct{}, 1),
		checkProgressChan:       make(chan struct{}, 1),
		progressChan:            make(chan uint64, 1),
		config:                  config,
	}
	err := validator.readLastBlockValidatedDbInfo(reorgingToBlock)
	if err != nil {
		return nil, err
	}
	streamer.SetBlockValidator(validator)
	inbox.SetBlockValidator(validator)
	return validator, nil
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
		if expectedHash != info.BlockHash {
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

func (v *BlockValidator) prepareBlock(ctx context.Context, header *types.Header, prevHeader *types.Header, msg arbstate.MessageWithMetadata, validationStatus *validationStatus) {
	preimages, readBatchInfo, hasDelayedMessage, delayedMsgToRead, err := BlockDataForValidation(ctx, v.blockchain, v.inboxReader, header, prevHeader, msg, v.config().StorePreimages)
	if err != nil {
		log.Error("failed to set up validation", "err", err, "header", header, "prevHeader", prevHeader)
		return
	}
	validationEntry, err := newValidationEntry(prevHeader, header, hasDelayedMessage, delayedMsgToRead, preimages, readBatchInfo)
	if err != nil {
		log.Error("failed to create validation entry", "err", err, "header", header, "prevHeader", prevHeader)
		return
	}
	validationStatus.Entry = validationEntry
	atomic.StoreUint32(&validationStatus.Status, validationStatusPrepared)
	select {
	case v.sendValidationsChan <- struct{}{}:
	default:
	}
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
	status := &validationStatus{
		Status:      validationStatusUnprepared,
		Entry:       nil,
		ModuleRoots: v.GetModuleRootsToValidate(),
	}
	// It's fine to separately load and then store as we have the blockMutex acquired
	_, present := v.validationEntries.Load(blockNum)
	if present {
		return
	}
	v.validationEntries.Store(blockNum, status)
	if v.nextValidationEntryBlock <= blockNum {
		v.nextValidationEntryBlock = blockNum + 1
	}
	v.LaunchUntrackedThread(func() { v.prepareBlock(context.Background(), block.Header(), prevHeader, msg, status) })
}

var launchTime = time.Now().Format("2006_01_02__15_04")

//nolint:gosec
func (v *BlockValidator) writeToFile(validationEntry *validationEntry, moduleRoot common.Hash, start, end GlobalStatePosition, preimages map[common.Hash][]byte, sequencerMsg, delayedMsg []byte) error {
	machConf := v.MachineLoader.GetConfig()
	outDirPath := filepath.Join(machConf.RootPath, v.config().OutputPath, launchTime, fmt.Sprintf("block_%d", validationEntry.BlockNumber))
	err := os.MkdirAll(outDirPath, 0755)
	if err != nil {
		return err
	}

	cmdFile, err := os.OpenFile(filepath.Join(outDirPath, "run-prover.sh"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer cmdFile.Close()
	_, err = cmdFile.WriteString("#!/bin/bash\n" +
		fmt.Sprintf("# expected output: batch %d, postion %d, hash %s\n", end.BatchNumber, end.PosInBatch, validationEntry.BlockHash) +
		"MACHPATH=\"" + machConf.getMachinePath(moduleRoot) + "\"\n" +
		"if (( $# > 1 )); then\n" +
		"	if [[ $1 == \"-m\" ]]; then\n" +
		"		MACHPATH=$2\n" +
		"		shift\n" +
		"		shift\n" +
		"	fi\n" +
		"fi\n" +
		"${ROOTPATH}/bin/prover ${MACHPATH}/" + machConf.ProverBinPath)
	if err != nil {
		return err
	}

	for _, module := range machConf.LibraryPaths {
		_, err = cmdFile.WriteString(" -l " + "${ROOTPATH}/" + module)
		if err != nil {
			return err
		}
	}
	_, err = cmdFile.WriteString(fmt.Sprintf(" --inbox-position %d --position-within-message %d --last-block-hash %s", start.BatchNumber, start.PosInBatch, validationEntry.PrevBlockHash))
	if err != nil {
		return err
	}

	sequencerFileName := fmt.Sprintf("sequencer_%d.bin", start.BatchNumber)
	err = os.WriteFile(filepath.Join(outDirPath, sequencerFileName), sequencerMsg, 0644)
	if err != nil {
		return err
	}
	_, err = cmdFile.WriteString(" --inbox " + sequencerFileName)
	if err != nil {
		return err
	}

	preimageFile, err := os.Create(filepath.Join(outDirPath, "preimages.bin"))
	if err != nil {
		return err
	}
	defer preimageFile.Close()
	for _, data := range preimages {
		lenbytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(lenbytes, uint64(len(data)))
		_, err := preimageFile.Write(lenbytes)
		if err != nil {
			return err
		}
		_, err = preimageFile.Write(data)
		if err != nil {
			return err
		}
	}

	_, err = cmdFile.WriteString(" --preimages preimages.bin")
	if err != nil {
		return err
	}

	if validationEntry.HasDelayedMsg {
		_, err = cmdFile.WriteString(fmt.Sprintf(" --delayed-inbox-position %d", validationEntry.DelayedMsgNr))
		if err != nil {
			return err
		}
		filename := fmt.Sprintf("delayed_%d.bin", validationEntry.DelayedMsgNr)
		err = os.WriteFile(filepath.Join(outDirPath, filename), delayedMsg, 0644)
		if err != nil {
			return err
		}
		_, err = cmdFile.WriteString(fmt.Sprintf(" --delayed-inbox %s", filename))
		if err != nil {
			return err
		}
	}

	_, err = cmdFile.WriteString(" \"$@\"\n")
	if err != nil {
		return err
	}
	return nil
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

func (v *BlockValidator) validate(ctx context.Context, validationStatus *validationStatus, seqMsg []byte) {
	if currentStatus := atomic.LoadUint32(&validationStatus.Status); currentStatus != validationStatusPrepared {
		log.Error("attempted to validate unprepared validation entry", "status", currentStatus)
		return
	}
	entry := validationStatus.Entry
	defer func() {
		atomic.AddInt32(&v.atomicValidationsRunning, -1)
		select {
		case v.sendValidationsChan <- struct{}{}:
		default:
		}
	}()
	entry.BatchInfo = append(entry.BatchInfo, BatchInfo{
		Number: entry.StartPosition.BatchNumber,
		Data:   seqMsg,
	})
	log.Info(
		"starting validation for block", "blockNr", entry.BlockNumber,
		"blockDate", common.PrettyAge(time.Unix(int64(entry.BlockHeader.Time), 0)))
	for _, moduleRoot := range validationStatus.ModuleRoots {

		type replay = func(context.Context, *validationEntry, common.Hash) (GoGlobalState, []byte, error)
		var delayedMsg []byte

		execValidation := func(replay replay, name string) (bool, bool) {
			gsEnd, delayed, err := replay(ctx, entry, moduleRoot)
			delayedMsg = delayed

			if err != nil {
				canceled := ctx.Err() != nil
				if canceled {
					log.Info(
						"Validation of block canceled", "blockNr", entry.BlockNumber,
						"blockHash", entry.BlockHash, "name", name, "err", err,
					)
				} else {
					log.Error(
						"Validation of block failed", "blockNr", entry.BlockNumber,
						"blockHash", entry.BlockHash, "moduleRoot", moduleRoot,
						"name", name, "err", err,
					)
				}
				return false, !canceled
			}

			gsExpected := entry.expectedEnd()
			resultValid := gsEnd == gsExpected

			if !resultValid {
				log.Error(
					"validation failed", "moduleRoot", moduleRoot, "got", gsEnd,
					"expected", gsExpected, "expHeader", entry.BlockHeader, "name", name,
				)
			}
			return resultValid, !resultValid
		}

		before := time.Now()
		writeBlock := false // we write the block if either fail

		config := v.config()
		valid := true
		if config.ArbitratorValidator {
			thisValid, thisWriteBlock := execValidation(v.executeBlock, "arbitrator")
			valid = valid && thisValid
			writeBlock = writeBlock || thisWriteBlock
		}
		if config.JitValidator {
			thisValid, thisWriteBlock := execValidation(v.jitBlock, "jit")
			valid = valid && thisValid
			writeBlock = writeBlock || thisWriteBlock
		}

		if writeBlock {
			err := v.writeToFile(
				entry, moduleRoot, entry.StartPosition, entry.EndPosition,
				entry.Preimages, seqMsg, delayedMsg,
			)
			if err != nil {
				log.Error("failed to write file", "err", err)
			}
		}

		if !valid {
			atomic.StoreUint32(&validationStatus.Status, validationStatusFailed)
			return
		}

		log.Info(
			"validation succeeded", "blockNr", entry.BlockNumber,
			"blockDate", common.PrettyAge(time.Unix(int64(entry.BlockHeader.Time), 0)),
			"blockHash", entry.BlockHash, "moduleRoot", moduleRoot, "time", time.Since(before),
		)
	}

	atomic.StoreUint32(&validationStatus.Status, validationStatusValid) // after that - validation entry could be deleted from map
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
		entry, found := v.validationEntries.Load(v.nextBlockToValidate)
		if !found {
			block := v.blockchain.GetBlockByNumber(v.nextBlockToValidate)
			if block == nil {
				// This block hasn't been created yet.
				return
			}
			prevHeader := v.blockchain.GetHeaderByHash(block.ParentHash())
			if prevHeader == nil && block.ParentHash() != (common.Hash{}) {
				log.Warn("failed to get prevHeader in block validator", "num", v.nextBlockToValidate-1, "hash", block.ParentHash())
				return
			}
			msg, err := v.streamer.GetMessage(nextMsg)
			if err != nil {
				log.Warn("failed to get message in block validator", "err", err)
				return
			}
			v.NewBlock(block, prevHeader, *msg)
			return
		}
		validationStatus, ok := entry.(*validationStatus)
		if !ok || (validationStatus == nil) {
			log.Error("bad entry trying to validate batch")
			return
		}
		if atomic.LoadUint32(&validationStatus.Status) == validationStatusUnprepared {
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
		validationStatus.Entry.StartPosition = startPos
		validationStatus.Entry.EndPosition = endPos

		batchNum := validationStatus.Entry.StartPosition.BatchNumber
		seqMsg, ok := seqBatchEntry.([]byte)
		if !ok {
			log.Error("sequencer message bad format", "blockNr", v.nextBlockToValidate, "msgNum", batchNum)
			return
		}

		v.LaunchThread(func(ctx context.Context) {
			validationCtx, cancel := context.WithCancel(ctx)
			validationStatus.Cancel = cancel
			v.validate(validationCtx, validationStatus, seqMsg)
			cancel()
		})

		v.nextBlockToValidate++
		v.globalPosNextSend = endPos
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
		entry, found := v.validationEntries.Load(checkingBlock)
		if !found {
			return
		}
		validationStatus, ok := entry.(*validationStatus)
		if !ok || (validationStatus == nil) {
			log.Error("bad entry trying to advance validated counter")
			return
		}
		if atomic.LoadUint32(&validationStatus.Status) < validationStatusValid {
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

		v.validationEntries.Delete(checkingBlock)
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

func (v *BlockValidator) AssumeValid(globalState GoGlobalState) error {
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

	select {
	case v.sendValidationsChan <- struct{}{}:
	default:
	}
}

func (v *BlockValidator) ReorgToBlock(blockNum uint64, blockHash common.Hash) error {
	v.blockMutex.Lock()
	defer v.blockMutex.Unlock()

	atomic.AddInt32(&v.reorgsPending, 1)
	v.reorgMutex.Lock()
	defer v.reorgMutex.Unlock()
	atomic.AddInt32(&v.reorgsPending, -1)

	if blockNum+1 < v.nextValidationEntryBlock {
		log.Warn("block validator processing reorg", "blockNum", blockNum)
		err := v.reorgToBlockImpl(blockNum, blockHash, false)
		if err != nil {
			return fmt.Errorf("block validator reorg failed: %w", err)
		}
	}

	return nil
}

func (v *BlockValidator) reorgToBlockImpl(blockNum uint64, blockHash common.Hash, hasLastValidatedMutex bool) error {
	for b := blockNum + 1; b < v.nextValidationEntryBlock; b++ {
		entry, found := v.validationEntries.Load(b)
		if !found {
			continue
		}
		v.validationEntries.Delete(b)

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
	v.nextValidationEntryBlock = blockNum + 1
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
		v.nextValidationEntryBlock = blockNum + 1
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

// Must be called after SetCurrentWasmModuleRoot sets the current one
func (v *BlockValidator) Initialize() error {
	config := v.config()
	currentModuleRoot := config.CurrentModuleRoot
	switch currentModuleRoot {
	case "latest":
		latest, err := v.MachineLoader.GetConfig().ReadLatestWasmModuleRoot()
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
	if config.ArbitratorValidator {
		if err := v.MachineLoader.CreateMachine(v.currentWasmModuleRoot, true, false); err != nil {
			return err
		}
	}
	if config.JitValidator {
		if err := v.MachineLoader.CreateMachine(v.currentWasmModuleRoot, true, true); err != nil {
			return err
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
				v.sendValidations(ctx)
			case <-ctx.Done():
				return
			}
		}
	})
	return nil
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
