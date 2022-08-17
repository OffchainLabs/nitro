// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validator

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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type StateTracker interface {
	Initialize(context.Context, *types.Block) error
	LastBlockValidated(context.Context) (uint64, error)
	LastBlockValidatedAndHash(context.Context) (uint64, common.Hash, error)
	BlockAfterLastValidated(context.Context) (uint64, error)
	GetNextValidation(context.Context) (uint64, GlobalStatePosition, error)
	BeginValidation(context.Context, *types.Header, GlobalStatePosition, GlobalStatePosition) (bool, func(bool), error)
	ValidationCompleted(context.Context, *validationEntry) (uint64, GlobalStatePosition, error)
	Reorg(context.Context, uint64, common.Hash, GlobalStatePosition, func(uint64, common.Hash) bool) error
	ForceConfirm(context.Context, uint64, common.Hash, GlobalStatePosition) error
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
	validationPaused  int32 // atomic

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
	Threads                  int                     `koanf:"threads"`
	CurrentModuleRoot        string                  `koanf:"current-module-root"`
	PendingUpgradeModuleRoot string                  `koanf:"pending-upgrade-module-root"`
	StorePreimages           bool                    `koanf:"store-preimages"`
	Redis                    RedisStateTrackerConfig `koanf:"redis"`
}

func BlockValidatorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBlockValidatorConfig.Enable, "enable block validator")
	f.String(prefix+".output-path", DefaultBlockValidatorConfig.OutputPath, "")
	f.Int(prefix+".threads", DefaultBlockValidatorConfig.Threads, "the number of threads to use for block validation (0 = don't validate anything, -1 = use number of cores)")
	f.String(prefix+".current-module-root", DefaultBlockValidatorConfig.CurrentModuleRoot, "current wasm module root ('current' read from chain, 'latest' from machines/latest dir, or provide hash)")
	f.String(prefix+".pending-upgrade-module-root", DefaultBlockValidatorConfig.PendingUpgradeModuleRoot, "pending upgrade wasm module root to additionally validate (hash, 'latest' or empty)")
	f.Bool(prefix+".store-preimages", DefaultBlockValidatorConfig.StorePreimages, "store preimages of running machines (higher memory cost, better debugging, potentially better performance)")
	RedisStateTrackerConfigAddOptions(prefix+".redis", f)
}

var DefaultBlockValidatorConfig = BlockValidatorConfig{
	Enable:                   false,
	OutputPath:               "output",
	Threads:                  -1,
	CurrentModuleRoot:        "current",
	PendingUpgradeModuleRoot: "latest",
	StorePreimages:           false,
}

var TestBlockValidatorConfig = BlockValidatorConfig{
	Enable:                   false,
	OutputPath:               "output",
	Threads:                  -1,
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
	concurrent := config.Threads
	if concurrent < 0 {
		concurrent = runtime.NumCPU()
	}
	statelessVal, err := NewStatelessBlockValidator(
		machineLoader,
		inboxReader,
		inbox,
		streamer,
		blockchain,
		das,
		config.OutputPath,
	)
	if err != nil {
		return nil, err
	}
	var stateTracker StateTracker
	if len(config.Redis.Url) > 0 {
		stateTracker, err = NewRedisStateTracker(config.Redis)
		if err != nil {
			return nil, err
		}
	} else {
		stateTracker, err = NewLocalStateTracker(db)
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
	select {
	case v.sendValidationsChan <- struct{}{}:
	default:
	}
}

var launchTime = time.Now().Format("2006_01_02__15_04")

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

func (v *BlockValidator) ForceConfirm(ctx context.Context, blockNumber uint64, globalState GoGlobalState) error {
	v.cancelMutex.Lock()
	defer v.cancelMutex.Unlock()
	v.reorgMutex.Lock()
	defer v.reorgMutex.Unlock()
	afterLastValidated, err := v.stateTracker.BlockAfterLastValidated(ctx)
	if err != nil {
		return err
	}
	if afterLastValidated > blockNumber {
		return nil
	}
	for b := afterLastValidated; b <= blockNumber && b < v.nextValidationCancelsBlock; b++ {
		entry, found := v.validationCancels.Load(b)
		if !found {
			continue
		}
		v.validationCancels.Delete(b)

		cancel, ok := entry.(func())
		if !ok || cancel == nil {
			log.Error("bad cancel function trying to force confirm block validator")
			continue
		}
		log.Debug("canceling validation due to L1 rollup progress", "block", b)
		cancel()
	}
	globalStatePos := GlobalStatePosition{
		BatchNumber: globalState.Batch,
		PosInBatch:  globalState.PosInBatch,
	}
	return v.stateTracker.ForceConfirm(ctx, blockNumber, globalState.BlockHash, globalStatePos)
}

func (v *BlockValidator) SetValidationPaused(pause bool) {
	var pauseInt int32
	if pause {
		pauseInt = 1
	}
	atomic.StoreInt32(&v.validationPaused, pauseInt)
}

func (v *BlockValidator) validate(ctx context.Context, moduleRoots []common.Hash, prevHeader *types.Header, header *types.Header, msg arbstate.MessageWithMetadata, seqMsg []byte, startPos GlobalStatePosition, endPos GlobalStatePosition) (*validationEntry, error) {
	preimages, readBatchInfo, hasDelayedMessage, delayedMsgToRead, err := BlockDataForValidation(ctx, v.blockchain, v.inboxReader, header, prevHeader, msg, v.config.StorePreimages)
	if err != nil {
		return nil, err
	}
	entry, err := newValidationEntry(prevHeader, header, hasDelayedMessage, delayedMsgToRead, preimages, readBatchInfo, startPos, endPos)
	if err != nil {
		return nil, err
	}
	defer func() {
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
			return nil, fmt.Errorf("validation of block with wasm module root %v failed: %w", moduleRoot, err)
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
			return nil, fmt.Errorf("validation of block with wasm module root %v failed: expected %v with header %v but got %v", moduleRoot, gsExpected, entry.BlockHeader, gsEnd)
		}

		log.Info("validation succeeded", "blockNr", entry.BlockNumber, "blockHash", entry.BlockHash, "moduleRoot", moduleRoot, "time", duration)
	}
	return entry, nil
}

func (v *BlockValidator) sendValidations(ctx context.Context) {
	v.reorgMutex.Lock()
	defer v.reorgMutex.Unlock()
	var batchCount uint64
	for atomic.LoadInt32(&v.reorgsPending) == 0 {
		if atomic.LoadInt32(&v.atomicValidationsRunning) >= v.concurrentRunsLimit {
			return
		}
		if atomic.LoadInt32(&v.validationPaused) != 0 {
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

		seqMsg, ok := seqBatchEntry.([]byte)
		if !ok {
			log.Error("sequencer message bad format", "blockNr", nextBlockToValidate, "msgNum", startPos.BatchNumber)
			return
		}

		acquiredLockout, validationStopped, err := v.stateTracker.BeginValidation(ctx, block.Header(), startPos, endPos)
		if err != nil {
			log.Error("failed to begin validation in state tracker", "err", err)
			return
		}
		if acquiredLockout {
			atomic.AddInt32(&v.atomicValidationsRunning, 1)
			v.LaunchThread(func(ctx context.Context) {
				defer validationStopped(false)
				defer atomic.AddInt32(&v.atomicValidationsRunning, -1)
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
				entry, err := v.validate(validationCtx, v.GetModuleRootsToValidate(), prevHeader, block.Header(), msg, seqMsg, startPos, endPos)
				if err != nil {
					blockHash := block.Hash()
					if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
						log.Info("validation of block canceled", "blockNr", blockNum, "blockHash", blockHash, "err", err)
					} else {
						log.Error("validation of block failed", "blockNr", blockNum, "blockHash", blockHash, "err", err)
					}
					return
				}

				v.validationCancels.Delete(blockNum)
				validationStopped(true)
				v.finishedValidation(ctx, entry)
			})
			if v.GetContext().Err() != nil {
				validationStopped(false)
			}
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

	genesisBlock := v.blockchain.GetBlockByNumber(v.genesisBlockNum)
	if genesisBlock == nil {
		return fmt.Errorf("blockchain missing genesis block number %v", v.genesisBlockNum)
	}
	err := v.stateTracker.Initialize(ctx, genesisBlock)
	if err != nil {
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
