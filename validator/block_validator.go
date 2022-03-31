//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

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
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/util"
)

type BlockValidator struct {
	util.StopWaiter
	*StatelessBlockValidator

	validationEntries sync.Map
	sequencerBatches  sync.Map
	preimageCache     preimageCache
	blockMutex        sync.Mutex
	batchMutex        sync.Mutex
	reorgMutex        sync.Mutex
	reorgsPending     int32 // atomic

	lastBlockValidated      uint64      // both atomic and behind lastBlockValidatedMutex
	lastBlockValidatedHash  common.Hash // behind lastBlockValidatedMutex
	lastBlockValidatedMutex sync.Mutex
	earliestBatchKept       uint64
	nextBatchKept           uint64 // 1 + the last batch number kept

	nextBlockToValidate      uint64
	nextValidationEntryBlock uint64
	globalPosNextSend        GlobalStatePosition

	config                   *BlockValidatorConfig
	atomicValidationsRunning int32
	concurrentRunsLimit      int32

	sendValidationsChan chan struct{}
	checkProgressChan   chan struct{}
	progressChan        chan uint64
}

type BlockValidatorConfig struct {
	Enable              bool   `koanf:"enable"`
	OutputPath          string `koanf:"output-path"`
	ConcurrentRunsLimit int    `koanf:"concurrent-runs-limit"`
}

func BlockValidatorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBlockValidatorConfig.Enable, "enable block validator")
	f.String(prefix+".output-path", DefaultBlockValidatorConfig.OutputPath, "")
	f.Int(prefix+".concurrent-runs-limit", DefaultBlockValidatorConfig.ConcurrentRunsLimit, "")
}

var DefaultBlockValidatorConfig = BlockValidatorConfig{
	Enable:              false,
	OutputPath:          "./target/output",
	ConcurrentRunsLimit: 0,
}

const validationStatusUnprepared uint32 = 0 // waiting for validationEntry to be populated
const validationStatusPrepared uint32 = 1   // ready to undergo validation
const validationStatusValid uint32 = 2      // validation succeeded

type validationStatus struct {
	Status    uint32           // atomic: value is one of validationStatus*
	Cancel    func()           // non-atomic: only read/written to with reorg mutex
	Entry     *validationEntry // non-atomic: only read if Status >= validationStatusPrepared
	Preimages []common.Hash    // non-atomic: only read if Status >= validationStatusPrepared
}

func NewBlockValidator(inboxReader InboxReaderInterface, inbox InboxTrackerInterface, streamer TransactionStreamerInterface, blockchain *core.BlockChain, db ethdb.Database, config *BlockValidatorConfig, das das.DataAvailabilityService) (*BlockValidator, error) {
	concurrent := config.ConcurrentRunsLimit
	if concurrent == 0 {
		concurrent = runtime.NumCPU()
	}
	statelessVal, err := NewStatelessBlockValidator(
		inboxReader,
		inbox,
		streamer,
		blockchain,
		db,
		das,
	)
	if err != nil {
		return nil, err
	}
	validator := &BlockValidator{
		StatelessBlockValidator: statelessVal,
		sendValidationsChan:     make(chan struct{}, 1),
		checkProgressChan:       make(chan struct{}, 1),
		progressChan:            make(chan uint64, 1),
		concurrentRunsLimit:     int32(concurrent),
		config:                  config,
	}
	err = validator.readLastBlockValidatedDbInfo()
	if err != nil {
		return nil, err
	}
	streamer.SetBlockValidator(validator)
	inbox.SetBlockValidator(validator)
	return validator, nil
}

func (v *BlockValidator) readLastBlockValidatedDbInfo() error {
	v.lastBlockValidatedMutex.Lock()
	defer v.lastBlockValidatedMutex.Unlock()

	exists, err := v.db.Has(lastBlockValidatedInfoKey)
	if err != nil {
		return err
	}

	if !exists {
		// The db contains no validation info; start from the beginning.
		// TODO: this skips validating the genesis block.
		v.lastBlockValidated = v.genesisBlockNum
		v.lastBlockValidatedHash = v.blockchain.Genesis().Hash()
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

	expectedHash := v.blockchain.GetCanonicalHash(info.BlockNumber)
	if expectedHash != info.BlockHash {
		return fmt.Errorf("last validated block %v stored with hash %v, but blockchain has hash %v", info.BlockNumber, info.BlockHash, expectedHash)
	}

	v.lastBlockValidated = info.BlockNumber
	v.lastBlockValidatedHash = info.BlockHash
	v.nextBlockToValidate = v.lastBlockValidated + 1
	v.globalPosNextSend = info.AfterPosition

	return nil
}

func (v *BlockValidator) prepareBlock(header *types.Header, prevHeader *types.Header, msg arbstate.MessageWithMetadata, validationStatus *validationStatus) {
	preimages, hasDelayedMessage, delayedMsgToRead, err := BlockDataForValidation(v.blockchain, header, prevHeader, msg)
	if err != nil {
		log.Error("failed to set up validation", "err", err, "header", header, "prevHeader", prevHeader)
		return
	}
	hashlist := v.preimageCache.PourToCache(preimages)
	validationEntry, err := newValidationEntry(prevHeader, header, hasDelayedMessage, delayedMsgToRead)
	if err != nil {
		log.Error("failed to create validation entry", "err", err, "header", header, "prevHeader", prevHeader)
		return
	}
	validationStatus.Entry = validationEntry
	validationStatus.Preimages = hashlist
	atomic.StoreUint32(&validationStatus.Status, validationStatusPrepared)
	v.sendValidationsChan <- struct{}{}
}

func (v *BlockValidator) NewBlock(block *types.Block, prevHeader *types.Header, msg arbstate.MessageWithMetadata) {
	v.blockMutex.Lock()
	defer v.blockMutex.Unlock()
	status := &validationStatus{
		Status: validationStatusUnprepared,
		Entry:  nil,
	}
	blockNum := block.NumberU64()
	v.validationEntries.Store(blockNum, status)
	if v.nextValidationEntryBlock <= blockNum {
		v.nextValidationEntryBlock = blockNum + 1
	}
	v.LaunchUntrackedThread(func() { v.prepareBlock(block.Header(), prevHeader, msg, status) })
}

var launchTime = time.Now().Format("2006_01_02__15_04")

//nolint:gosec
func (v *BlockValidator) writeToFile(validationEntry *validationEntry, start, end GlobalStatePosition, preimages map[common.Hash][]byte, sequencerMsg, delayedMsg []byte) error {
	outDirPath := filepath.Join(StaticNitroMachineConfig.RootPath, v.config.OutputPath, launchTime, fmt.Sprintf("block_%d", validationEntry.BlockNumber))
	err := os.MkdirAll(outDirPath, 0777)
	if err != nil {
		return err
	}

	cmdFile, err := os.Create(filepath.Join(outDirPath, "run-prover.sh"))
	if err != nil {
		return err
	}
	defer cmdFile.Close()
	_, err = cmdFile.WriteString("#!/bin/bash\n" +
		fmt.Sprintf("# expected output: batch %d, postion %d, hash %s\n", end.BatchNumber, end.PosInBatch, validationEntry.BlockHash) +
		"ROOTPATH=\"" + StaticNitroMachineConfig.RootPath + "\"\n" +
		"if (( $# > 1 )); then\n" +
		"	if [[ $1 == \"-r\" ]]; then\n" +
		"		ROOTPATH=$2\n" +
		"		shift\n" +
		"		shift\n" +
		"	fi\n" +
		"fi\n" +
		"${ROOTPATH}/bin/prover ${ROOTPATH}/" + StaticNitroMachineConfig.ProverBinPath)
	if err != nil {
		return err
	}

	for _, module := range StaticNitroMachineConfig.ModulePaths {
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
	err = cmdFile.Chmod(0777)
	if err != nil {
		return err
	}
	return nil
}

func (v *BlockValidator) validate(ctx context.Context, validationStatus *validationStatus, seqMsg []byte) {
	if atomic.LoadUint32(&validationStatus.Status) < validationStatusPrepared {
		log.Error("attempted to validate unprepared validation entry")
		return
	}
	entry := validationStatus.Entry
	log.Info("starting validation for block", "blockNr", entry.BlockNumber)
	preimages, err := v.preimageCache.FillHashedValues(validationStatus.Preimages)
	if err != nil {
		log.Error("validator: failed prepare arrays", "err", err)
		return
	}
	defer (func() {
		atomic.AddInt32(&v.atomicValidationsRunning, -1)
		v.sendValidationsChan <- struct{}{}
		err := v.preimageCache.RemoveFromCache(validationStatus.Preimages)
		if err != nil {
			log.Error("validator failed to remove from cache", "err", err)
		}
	})()
	log.Info("starting validation for block", "blockNr", entry.BlockNumber)
	gsEnd, delayedMsg, err := v.executeBlock(ctx, entry, preimages, seqMsg)
	if err != nil {
		log.Error("Validation of block failed", "err", err)
		return
	}
	gsExpected := entry.expectedEnd()
	resultValid := gsEnd == gsExpected

	writeThisBlock := false
	if !resultValid {
		writeThisBlock = true
	}

	if writeThisBlock {
		err = v.writeToFile(entry, entry.StartPosition, entry.EndPosition, preimages, seqMsg, delayedMsg)
		if err != nil {
			log.Error("failed to write file", "err", err)
		}
	}

	if !resultValid {
		log.Error("validation failed", "got", gsEnd, "expected", gsExpected, "expHeader", entry.BlockHeader)
		return
	}

	atomic.StoreUint32(&validationStatus.Status, validationStatusValid) // after that - validation entry could be deleted from map
	log.Info("validation succeeded", "blockNr", entry.BlockNumber)
	v.checkProgressChan <- struct{}{}
}

func (v *BlockValidator) sendValidations(ctx context.Context) {
	v.reorgMutex.Lock()
	defer v.reorgMutex.Unlock()
	var batchCount uint64
	for atomic.LoadInt32(&v.reorgsPending) == 0 {
		if atomic.LoadInt32(&v.atomicValidationsRunning) >= v.concurrentRunsLimit {
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
			return
		}
		// valdationEntries is By blockNumber
		entry, found := v.validationEntries.Load(v.nextBlockToValidate)
		if !found {
			return
		}
		validationStatus, ok := entry.(*validationStatus)
		if !ok || (validationStatus == nil) {
			log.Error("bad entry trying to validate batch")
			return
		}
		if atomic.LoadUint32(&validationStatus.Status) == 0 {
			return
		}
		nextMsg := arbutil.BlockNumberToMessageCount(v.nextBlockToValidate, v.genesisBlockNum) - 1
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
		validationCtx, cancel := context.WithCancel(ctx)
		validationStatus.Cancel = cancel

		batchNum := validationStatus.Entry.StartPosition.BatchNumber
		seqMsg, ok := seqBatchEntry.([]byte)
		if !ok {
			log.Error("sequencer message bad format", "blockNr", v.nextBlockToValidate, "msgNum", batchNum)
			return
		}

		// validation can take long time. Don't wait for it when shutting down
		v.LaunchUntrackedThread(func() {
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

func (v *BlockValidator) LastBlockValidated() uint64 {
	return atomic.LoadUint64(&v.lastBlockValidated)
}

func (v *BlockValidator) LastBlockValidatedAndHash() (uint64, common.Hash) {
	v.lastBlockValidatedMutex.Lock()
	defer v.lastBlockValidatedMutex.Unlock()

	return v.lastBlockValidated, v.lastBlockValidatedHash
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
		err := v.reorgToBlockImpl(blockNum, blockHash)
		if err != nil {
			return fmt.Errorf("block validator reorg failed: %w", err)
		}
	}

	return nil
}

func (v *BlockValidator) reorgToBlockImpl(blockNum uint64, blockHash common.Hash) error {
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
		v.lastBlockValidatedMutex.Lock()
		atomic.StoreUint64(&v.lastBlockValidated, blockNum)
		v.lastBlockValidatedHash = blockHash
		v.lastBlockValidatedMutex.Unlock()

		err = v.writeLastValidatedToDb(blockNum, blockHash, v.globalPosNextSend)
		if err != nil {
			return err
		}
	}

	return nil
}

func (v *BlockValidator) Start(ctxIn context.Context) error {
	v.StopWaiter.Start(ctxIn)
	v.LaunchThread(func(ctx context.Context) {
		// `progressValidated` and `sendValidations` should both only do `concurrentRunsLimit` iterations of work,
		// so they won't stomp on each other and prevent the other from running.
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

// can only be used from One thread
func (v *BlockValidator) WaitForBlock(ctx context.Context, blockNumber uint64) bool {
	for {
		if atomic.LoadUint64(&v.lastBlockValidated) >= blockNumber {
			return true
		}
		select {
		case <-ctx.Done():
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
		}
	}
}
