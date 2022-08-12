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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type StateTracker interface {
	LastBlockValidated(context.Context) (uint64, error)
	LastBlockValidatedAndHash(context.Context) (uint64, common.Hash, error)
	GetNextValidation(context.Context) (uint64, GlobalStatePosition, error)
	BeginValidation(context.Context, *types.Header, GlobalStatePosition, GlobalStatePosition) (bool, error)
	ValidationCompleted(context.Context, *validationEntry) (uint64, GlobalStatePosition, error)
	Reorg(context.Context, uint64, common.Hash, GlobalStatePosition, func(uint64, common.Hash) bool) error
}

type BlockValidator struct {
	stopwaiter.StopWaiter
	*StatelessBlockValidator

	validationCancels sync.Map
	sequencerBatches  sync.Map
	cancelMutex       sync.Mutex
	batchMutex        sync.Mutex
	reorgMutex        sync.Mutex
	reorgsPending     int32 // atomic

	nextValidationCancelsBlock uint64
	earliestBatchKept          uint64
	nextBatchKept              uint64 // 1 + the last batch number kept
	currentWasmModuleRoot      common.Hash
	pendingWasmModuleRoot      common.Hash

	stateTracker StateTracker

	config                   *BlockValidatorConfig
	atomicValidationsRunning int32
	concurrentRunsLimit      int32
	initialReorgBlock        *types.Block

	sendValidationsChan chan struct{}
	progressChan        chan uint64
}

type BlockValidatorConfig struct {
	Enable                   bool                    `koanf:"enable"`
	OutputPath               string                  `koanf:"output-path"`
	ConcurrentRunsLimit      int                     `koanf:"concurrent-runs-limit"`
	CurrentModuleRoot        string                  `koanf:"current-module-root"`
	PendingUpgradeModuleRoot string                  `koanf:"pending-upgrade-module-root"`
	StorePreimages           bool                    `koanf:"store-preimages"`
	Redis                    RedisStateTrackerConfig `koanf:"redis"`
}

func BlockValidatorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBlockValidatorConfig.Enable, "enable block validator")
	f.String(prefix+".output-path", DefaultBlockValidatorConfig.OutputPath, "")
	f.Int(prefix+".concurrent-runs-limit", DefaultBlockValidatorConfig.ConcurrentRunsLimit, "")
	f.String(prefix+".current-module-root", DefaultBlockValidatorConfig.CurrentModuleRoot, "current wasm module root ('current' read from chain, 'latest' from machines/latest dir, or provide hash)")
	f.String(prefix+".pending-upgrade-module-root", DefaultBlockValidatorConfig.PendingUpgradeModuleRoot, "pending upgrade wasm module root to additionally validate (hash, 'latest' or empty)")
	f.Bool(prefix+".store-preimages", DefaultBlockValidatorConfig.StorePreimages, "store preimages of running machines (higher memory cost, better debugging, potentially better performance)")
	RedisStateTrackerConfigAddOptions(prefix+".redis", f)
}

var DefaultBlockValidatorConfig = BlockValidatorConfig{
	Enable:                   false,
	OutputPath:               "./target/output",
	ConcurrentRunsLimit:      0,
	CurrentModuleRoot:        "current",
	PendingUpgradeModuleRoot: "latest",
	StorePreimages:           false,
}

var TestBlockValidatorConfig = BlockValidatorConfig{
	Enable:                   false,
	OutputPath:               "./target/output",
	ConcurrentRunsLimit:      0,
	CurrentModuleRoot:        "latest",
	PendingUpgradeModuleRoot: "latest",
	StorePreimages:           false,
}

func NewBlockValidator(
	inboxReader InboxReaderInterface,
	inbox InboxTrackerInterface,
	streamer TransactionStreamerInterface,
	blockchain *core.BlockChain,
	db ethdb.Database,
	config *BlockValidatorConfig,
	machineLoader *NitroMachineLoader,
	das arbstate.DataAvailabilityReader,
	reorgingToBlock *types.Block,
) (*BlockValidator, error) {
	concurrent := config.ConcurrentRunsLimit
	if concurrent == 0 {
		concurrent = runtime.NumCPU()
	}
	statelessVal, err := NewStatelessBlockValidator(
		machineLoader,
		inboxReader,
		inbox,
		streamer,
		blockchain,
		das,
	)
	if err != nil {
		return nil, err
	}
	genesisBlock := blockchain.GetBlockByNumber(statelessVal.genesisBlockNum)
	if genesisBlock == nil {
		return nil, fmt.Errorf("blockchain missing genesis block number %v", statelessVal.genesisBlockNum)
	}
	var stateTracker StateTracker
	if config.Redis.Enable {
		stateTracker, err = NewRedisStateTracker(config.Redis, "block-validator", genesisBlock)
		if err != nil {
			return nil, err
		}
	} else {
		stateTracker, err = NewLocalStateTracker(db, genesisBlock)
		if err != nil {
			return nil, err
		}
	}
	validator := &BlockValidator{
		StatelessBlockValidator: statelessVal,
		sendValidationsChan:     make(chan struct{}, 1),
		progressChan:            make(chan uint64, 1),
		concurrentRunsLimit:     int32(concurrent),
		config:                  config,
		stateTracker:            stateTracker,
		initialReorgBlock:       reorgingToBlock,
	}
	streamer.SetBlockValidator(validator)
	inbox.SetBlockValidator(validator)
	return validator, nil
}

func (v *BlockValidator) getModuleRootsToValidateLocked() []common.Hash {
	validatingModuleRoots := []common.Hash{v.currentWasmModuleRoot}
	if (v.currentWasmModuleRoot != v.pendingWasmModuleRoot && v.pendingWasmModuleRoot != common.Hash{}) {
		validatingModuleRoots = append(validatingModuleRoots, v.pendingWasmModuleRoot)
	}
	return validatingModuleRoots
}

func (v *BlockValidator) GetModuleRootsToValidate() []common.Hash {
	v.cancelMutex.Lock()
	defer v.cancelMutex.Unlock()
	return v.getModuleRootsToValidateLocked()
}

func (v *BlockValidator) NewBlock(block *types.Block) {
	v.sendValidationsChan <- struct{}{}
}

var launchTime = time.Now().Format("2006_01_02__15_04")

//nolint:gosec
func (v *BlockValidator) writeToFile(validationEntry *validationEntry, moduleRoot common.Hash, start, end GlobalStatePosition, preimages map[common.Hash][]byte, sequencerMsg, delayedMsg []byte) error {
	machConf := v.MachineLoader.GetConfig()
	outDirPath := filepath.Join(machConf.RootPath, v.config.OutputPath, launchTime, fmt.Sprintf("block_%d", validationEntry.BlockNumber))
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
	v.cancelMutex.Lock()
	defer v.cancelMutex.Unlock()

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
	if v.config.CurrentModuleRoot != "current" {
		return nil
	}
	return fmt.Errorf("unexpected wasmModuleRoot! cannot validate! found %v , current %v, pending %v", hash, v.currentWasmModuleRoot, v.pendingWasmModuleRoot)
}

func (v *BlockValidator) validate(ctx context.Context, moduleRoots []common.Hash, prevHeader *types.Header, header *types.Header, msg arbstate.MessageWithMetadata, seqMsg []byte, startPos GlobalStatePosition, endPos GlobalStatePosition) {
	preimages, readBatchInfo, hasDelayedMessage, delayedMsgToRead, err := BlockDataForValidation(ctx, v.blockchain, v.inboxReader, header, prevHeader, msg, v.config.StorePreimages)
	if err != nil {
		log.Error("failed to set up validation", "err", err, "header", header, "prevHeader", prevHeader)
		return
	}
	entry, err := newValidationEntry(prevHeader, header, hasDelayedMessage, delayedMsgToRead, preimages, readBatchInfo, startPos, endPos)
	if err != nil {
		log.Error("failed to create validation entry", "err", err, "header", header, "prevHeader", prevHeader)
		return
	}
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
	log.Info("starting validation for block", "blockNr", entry.BlockNumber)
	for _, moduleRoot := range moduleRoots {
		before := time.Now()
		gsEnd, delayedMsg, err := v.executeBlock(ctx, entry, moduleRoot)
		duration := time.Since(before)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				log.Info("Validation of block canceled", "blockNr", entry.BlockNumber, "blockHash", entry.BlockHash, "err", err)
			} else {
				log.Error("Validation of block failed", "blockNr", entry.BlockNumber, "blockHash", entry.BlockHash, "moduleRoot", moduleRoot, "err", err)
			}
			return
		}
		gsExpected := entry.expectedEnd()
		resultValid := gsEnd == gsExpected

		writeThisBlock := false
		if !resultValid {
			writeThisBlock = true
		}

		if writeThisBlock {
			err = v.writeToFile(entry, moduleRoot, entry.StartPosition, entry.EndPosition, entry.Preimages, seqMsg, delayedMsg)
			if err != nil {
				log.Error("failed to write file", "err", err)
			}
		}

		if !resultValid {
			log.Error("validation failed", "moduleRoot", moduleRoot, "got", gsEnd, "expected", gsExpected, "expHeader", entry.BlockHeader)
			return
		}

		log.Info("validation succeeded", "blockNr", entry.BlockNumber, "blockHash", entry.BlockHash, "moduleRoot", moduleRoot, "time", duration)
	}

	v.validationCancels.Delete(entry.BlockNumber)
	v.finishedValidation(ctx, entry)
}

func (v *BlockValidator) sendValidations(ctx context.Context) {
	v.reorgMutex.Lock()
	defer v.reorgMutex.Unlock()
	var batchCount uint64
	for atomic.LoadInt32(&v.reorgsPending) == 0 {
		if atomic.LoadInt32(&v.atomicValidationsRunning) >= v.concurrentRunsLimit {
			return
		}
		nextBlockToValidate, globalPosNextSend, err := v.stateTracker.GetNextValidation(ctx)
		if err != nil {
			log.Error("validator failed to get next validation", "err", err)
			return
		}
		if batchCount <= globalPosNextSend.BatchNumber {
			var err error
			batchCount, err = v.inboxTracker.GetBatchCount()
			if err != nil {
				log.Error("validator failed to get message count", "err", err)
				return
			}
			if batchCount <= globalPosNextSend.BatchNumber {
				return
			}
		}
		seqBatchEntry, haveBatch := v.sequencerBatches.Load(globalPosNextSend.BatchNumber)
		if !haveBatch {
			seqMsg, err := v.inboxReader.GetSequencerMessageBytes(ctx, globalPosNextSend.BatchNumber)
			if err != nil {
				log.Error("validator failed to read sequencer message", "err", err)
				return
			}
			v.ProcessBatches(globalPosNextSend.BatchNumber, [][]byte{seqMsg})
			seqBatchEntry = seqMsg
		}
		nextMsg := arbutil.BlockNumberToMessageCount(nextBlockToValidate, v.genesisBlockNum) - 1
		block := v.blockchain.GetBlockByNumber(nextBlockToValidate)
		if block == nil {
			// This block hasn't been created yet.
			return
		}
		prevHeader := v.blockchain.GetHeaderByHash(block.ParentHash())
		if prevHeader == nil && block.ParentHash() != (common.Hash{}) {
			log.Warn("failed to get prevHeader in block validator", "num", nextBlockToValidate-1, "hash", block.ParentHash())
			return
		}
		msg, err := v.streamer.GetMessage(nextMsg)
		if err != nil {
			log.Warn("failed to get message in block validator", "err", err)
			return
		}
		startPos, endPos, err := GlobalStatePositionsFor(v.inboxTracker, nextMsg, globalPosNextSend.BatchNumber)
		if err != nil {
			log.Error("failed calculating position for validation", "err", err, "msg", nextMsg, "batch", globalPosNextSend.BatchNumber)
			return
		}
		if startPos != globalPosNextSend {
			log.Error("inconsistent pos mapping", "msg", nextMsg, "expected", globalPosNextSend, "found", startPos)
			return
		}
		atomic.AddInt32(&v.atomicValidationsRunning, 1)

		seqMsg, ok := seqBatchEntry.([]byte)
		if !ok {
			log.Error("sequencer message bad format", "blockNr", nextBlockToValidate, "msgNum", startPos.BatchNumber)
			return
		}

		success, err := v.stateTracker.BeginValidation(ctx, block.Header(), startPos, endPos)
		if err != nil {
			log.Error("failed to begin validation in state tracker", "err", err)
			return
		}
		if success {
			v.LaunchThread(func(ctx context.Context) {
				validationCtx, cancel := context.WithCancel(ctx)
				defer cancel()
				blockNum := block.NumberU64()
				v.cancelMutex.Lock()
				// It's fine to separately load and then store as we have the cancelMutex acquired
				_, present := v.validationCancels.Load(blockNum)
				if present {
					log.Warn("validation somehow double-started?", "block", blockNum)
					v.cancelMutex.Unlock()
					return
				}
				v.validationCancels.Store(blockNum, cancel)
				if v.nextValidationCancelsBlock <= blockNum {
					v.nextValidationCancelsBlock = blockNum + 1
				}
				v.cancelMutex.Unlock()
				v.validate(validationCtx, v.GetModuleRootsToValidate(), prevHeader, block.Header(), msg, seqMsg, startPos, endPos)
			})
		}
	}
}

func (v *BlockValidator) finishedValidation(ctx context.Context, entry *validationEntry) {
	v.reorgMutex.Lock()
	defer v.reorgMutex.Unlock()
	latestValidated, nextGlobalState, err := v.stateTracker.ValidationCompleted(ctx, entry)
	if err != nil {
		log.Error("failed to record completed validation", "block", entry.BlockNumber, "err", err)
		return
	}

	earliestBatchKept := atomic.LoadUint64(&v.earliestBatchKept)
	seqMsgNr := nextGlobalState.BatchNumber
	for earliestBatchKept < seqMsgNr {
		v.sequencerBatches.Delete(earliestBatchKept)
		atomic.CompareAndSwapUint64(&v.earliestBatchKept, earliestBatchKept, earliestBatchKept+1)
		earliestBatchKept = atomic.LoadUint64(&v.earliestBatchKept)
	}

	select {
	case v.progressChan <- latestValidated:
	default:
	}
}

func (v *BlockValidator) LastBlockValidated(ctx context.Context) (uint64, error) {
	block, err := v.stateTracker.LastBlockValidated(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get last block validated: %w", err)
	}
	return block, nil
}

func (v *BlockValidator) LastBlockValidatedAndHash(ctx context.Context) (blockNumber uint64, blockHash common.Hash, wasmModuleRoots []common.Hash, err error) {
	blockNumber, blockHash, err = v.stateTracker.LastBlockValidatedAndHash(ctx)
	if err != nil {
		err = fmt.Errorf("failed to get last block validated and hash: %w", err)
		return
	}

	// things can be removed from, but not added to, moduleRootsToValidate. By taking root hashes fter the block we know result is valid
	wasmModuleRoots = v.GetModuleRootsToValidate()

	return
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

func (v *BlockValidator) ReorgToBlock(ctx context.Context, blockNum uint64, blockHash common.Hash) error {
	v.cancelMutex.Lock()
	defer v.cancelMutex.Unlock()

	atomic.AddInt32(&v.reorgsPending, 1)
	v.reorgMutex.Lock()
	defer v.reorgMutex.Unlock()
	atomic.AddInt32(&v.reorgsPending, -1)

	nextBlockToValidate, _, err := v.stateTracker.GetNextValidation(ctx)
	if err != nil {
		return err
	}
	if blockNum+1 < nextBlockToValidate {
		log.Warn("block validator processing reorg", "blockNum", blockNum)
		err := v.reorgToBlockImpl(ctx, blockNum, blockHash)
		if err != nil {
			return fmt.Errorf("block validator reorg failed: %w", err)
		}
	}

	return nil
}

func (v *BlockValidator) reorgToBlockImpl(ctx context.Context, blockNum uint64, blockHash common.Hash) error {
	for b := blockNum + 1; b < v.nextValidationCancelsBlock; b++ {
		entry, found := v.validationCancels.Load(b)
		if !found {
			continue
		}
		v.validationCancels.Delete(b)

		cancel, ok := entry.(func())
		if !ok || cancel == nil {
			log.Error("bad cancel function trying to reorg block validator")
			continue
		}
		log.Debug("canceling validation due to reorg", "block", b)
		cancel()
	}
	v.nextValidationCancelsBlock = blockNum + 1
	nextBlockToValidate, _, err := v.stateTracker.GetNextValidation(ctx)
	if err != nil {
		return err
	}
	if nextBlockToValidate <= blockNum+1 {
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
	var newNextPosition GlobalStatePosition
	if batch >= batchCount {
		// This reorg is past the latest batch.
		// Attempt to recover by loading a next validation state at the start of the next batch.
		newNextPosition = GlobalStatePosition{
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
		v.nextValidationCancelsBlock = blockNum + 1
	} else {
		_, newNextPosition, err = GlobalStatePositionsFor(v.inboxTracker, msgIndex, batch)
		if err != nil {
			return err
		}
	}

	isBlockValid := func(num uint64, hash common.Hash) bool {
		// If this is before the reorg, it's fine
		if num <= blockNum {
			return true
		}
		bcHash := v.blockchain.GetCanonicalHash(num)
		if bcHash == (common.Hash{}) {
			// For safety, we treat unknown block numbers as invalid.
			return false
		}
		// Counterintuitively, the reorg function is called _before_ the blocks are removed.
		// Thus, a good heuristic is that our blockchain hash is **invalid**.
		return bcHash != hash
	}

	return v.stateTracker.Reorg(ctx, blockNum, blockHash, newNextPosition, isBlockValid)
}

// Must be called after SetCurrentWasmModuleRoot sets the current one
func (v *BlockValidator) Initialize(ctx context.Context) error {
	switch v.config.CurrentModuleRoot {
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
		v.currentWasmModuleRoot = common.HexToHash(v.config.CurrentModuleRoot)
		if (v.currentWasmModuleRoot == common.Hash{}) {
			return errors.New("current-module-root config value illegal")
		}
	}
	if err := v.MachineLoader.CreateMachine(v.currentWasmModuleRoot, true); err != nil {
		return err
	}

	lastBlockValidated, lastBlockHashValidated, err := v.stateTracker.LastBlockValidatedAndHash(ctx)
	if err != nil {
		return err
	}
	reorgingToBlock := v.initialReorgBlock
	if reorgingToBlock != nil && reorgingToBlock.NumberU64() >= lastBlockValidated {
		// Disregard this reorg as it doesn't affect the last validated block
		reorgingToBlock = nil
	}
	if reorgingToBlock == nil {
		expectedHash := v.blockchain.GetCanonicalHash(lastBlockValidated)
		if expectedHash != lastBlockHashValidated {
			return fmt.Errorf("last validated block %v stored with hash %v, but blockchain has hash %v", lastBlockValidated, lastBlockHashValidated, expectedHash)
		}
	}
	if reorgingToBlock != nil {
		err = v.reorgToBlockImpl(ctx, reorgingToBlock.NumberU64(), reorgingToBlock.Hash())
		if err != nil {
			return err
		}
	}
	if v.config.PendingUpgradeModuleRoot != "" {
		if v.config.PendingUpgradeModuleRoot == "latest" {
			latest, err := v.MachineLoader.GetConfig().ReadLatestWasmModuleRoot()
			if err != nil {
				return err
			}
			v.pendingWasmModuleRoot = latest
		} else {
			v.pendingWasmModuleRoot = common.HexToHash(v.config.PendingUpgradeModuleRoot)
			if (v.pendingWasmModuleRoot == common.Hash{}) {
				return errors.New("pending-upgrade-module-root config value illegal")
			}
		}
		if err := v.MachineLoader.CreateMachine(v.pendingWasmModuleRoot, true); err != nil {
			return err
		}
	}
	v.initialReorgBlock = nil

	log.Info("BlockValidator initialized", "current", v.currentWasmModuleRoot, "pending", v.pendingWasmModuleRoot)
	return nil
}

func (v *BlockValidator) Start(ctxIn context.Context) error {
	v.StopWaiter.Start(ctxIn)
	v.LaunchThread(func(ctx context.Context) {
		// `progressValidated` and `sendValidations` should both only do `concurrentRunsLimit` iterations of work,
		// so they won't stomp on each other and prevent the other from running.
		v.sendValidations(ctx)
		for {
			select {
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
		lastValidated, err := v.stateTracker.LastBlockValidated(ctx)
		if err != nil {
			log.Error("failed to get last validated block", "err", err)
		} else if lastValidated >= blockNumber {
			return true
		}
		select {
		case <-timer.C:
			lastValidated, err := v.stateTracker.LastBlockValidated(ctx)
			if err != nil {
				log.Error("failed to get last block validated", "err", err)
				return false
			}
			if lastValidated >= blockNumber {
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
