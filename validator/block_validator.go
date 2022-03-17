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

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/util"
	"github.com/pkg/errors"
)

type BlockValidator struct {
	util.StopWaiter
	inboxTracker    InboxTrackerInterface
	blockchain      *core.BlockChain
	db              ethdb.Database
	genesisBlockNum uint64

	validationEntries sync.Map
	sequencerBatches  sync.Map
	preimageCache     preimageCache
	blockMutex        sync.Mutex

	lastBlockValidated uint64
	earliestBatchKept  uint64
	nextBatchKept      uint64 // 1 + the last batch number kept

	nextBlockToValidate uint64
	globalPosNextSend   GlobalStatePosition

	config                   *BlockValidatorConfig
	atomicValidationsRunning int32
	concurrentRunsLimit      int32

	das das.DataAvailabilityService

	sendValidationsChan     chan struct{}
	checkProgressChan       chan struct{}
	reorgToBlockChan        chan uint64
	blockReorgCompletedChan chan struct{}
	progressChan            chan uint64
}

type BlockValidatorConfig struct {
	OutputPath          string
	ConcurrentRunsLimit int // 0 - default (CPU#)
	BlocksToRecord      []uint64
}

var DefaultBlockValidatorConfig = BlockValidatorConfig{
	OutputPath:          "./target/output",
	ConcurrentRunsLimit: 0,
	BlocksToRecord:      []uint64{},
}

type BlockValidatorRegistrer interface {
	SetBlockValidator(*BlockValidator)
}

type InboxTrackerInterface interface {
	BlockValidatorRegistrer
	GetDelayedMessageBytes(uint64) ([]byte, error)
	GetBatchMessageCount(seqNum uint64) (arbutil.MessageIndex, error)
	GetBatchAcc(seqNum uint64) (common.Hash, error)
	GetBatchCount() (uint64, error)
}

type TransactionStreamerInterface interface {
	BlockValidatorRegistrer
	GetMessage(seqNum arbutil.MessageIndex) (arbstate.MessageWithMetadata, error)
	GetGenesisBlockNumber() (uint64, error)
}

type InboxReaderInterface interface {
	GetSequencerMessageBytes(ctx context.Context, seqNum uint64) ([]byte, error)
}

type GlobalStatePosition struct {
	BatchNumber uint64
	PosInBatch  uint64
}

func GlobalStatePositionsFor(tracker InboxTrackerInterface, pos arbutil.MessageIndex, batch uint64) (GlobalStatePosition, GlobalStatePosition, error) {
	msgCountInBatch, err := tracker.GetBatchMessageCount(batch)
	if err != nil {
		return GlobalStatePosition{}, GlobalStatePosition{}, err
	}
	var firstInBatch arbutil.MessageIndex
	if batch > 0 {
		firstInBatch, err = tracker.GetBatchMessageCount(batch - 1)
		if err != nil {
			return GlobalStatePosition{}, GlobalStatePosition{}, err
		}
	}
	if msgCountInBatch <= pos {
		return GlobalStatePosition{}, GlobalStatePosition{}, fmt.Errorf("batch %d has up to message %d, failed getting for %d", batch, msgCountInBatch-1, pos)
	}
	if firstInBatch > pos {
		return GlobalStatePosition{}, GlobalStatePosition{}, fmt.Errorf("batch %d starts from %d, failed getting for %d", batch, firstInBatch, pos)
	}
	startPos := GlobalStatePosition{batch, uint64(pos - firstInBatch)}
	if msgCountInBatch == pos+1 {
		return startPos, GlobalStatePosition{batch + 1, 0}, nil
	}
	return startPos, GlobalStatePosition{batch, uint64(pos + 1 - firstInBatch)}, nil
}

func FindBatchContainingMessageIndex(tracker InboxTrackerInterface, pos arbutil.MessageIndex, high uint64) (uint64, error) {
	var low uint64
	// Iteration preconditions:
	// - high >= low
	// - msgCount(low - 1) <= pos implies low <= target
	// - msgCount(high) > pos implies high >= target
	// Therefore, if low == high, then low == high == target
	for high > low {
		// Due to integer rounding, mid >= low && mid < high
		mid := (low + high) / 2
		count, err := tracker.GetBatchMessageCount(mid)
		if err != nil {
			return 0, err
		}
		if count < pos {
			// Must narrow as mid >= low, therefore mid + 1 > low, therefore newLow > oldLow
			// Keeps low precondition as msgCount(mid) < pos
			low = mid + 1
		} else if count == pos {
			return mid + 1, nil
		} else if count == pos+1 || mid == low { // implied: count > pos
			return mid, nil
		} else { // implied: count > pos + 1
			// Must narrow as mid < high, therefore newHigh < lowHigh
			// Keeps high precondition as msgCount(mid) > pos
			high = mid
		}
	}
	return low, nil
}

type validationEntry struct {
	BlockNumber   uint64
	PrevBlockHash common.Hash
	BlockHash     common.Hash
	SendRoot      common.Hash
	PrevSendRoot  common.Hash
	BlockHeader   *types.Header
	Preimages     []common.Hash
	HasDelayedMsg bool
	DelayedMsgNr  uint64
	SeqMsgNr      uint64
	StartPosition GlobalStatePosition
	EndPosition   GlobalStatePosition
}

const validationStatusUnprepared uint32 = 0 // waiting for validationEntry to be populated
const validationStatusPrepared uint32 = 1   // ready to undergo validation
const validationStatusValid uint32 = 2      // validation succeeded

type validationStatus struct {
	Status uint32           // atomic: value is one of validationStatus*
	Cancel func()           // non-atomic: only read/written to by main block validator thread
	Entry  *validationEntry // non-atomic: only read if Status >= validationStatusPrepared
}

func newValidationEntry(prevHeader *types.Header, header *types.Header, hasDelayed bool, delayedMsgNr uint64, preimages []common.Hash) (*validationEntry, error) {
	extraInfo, err := types.DeserializeHeaderExtraInformation(header)
	if err != nil {
		return nil, err
	}
	prevExtraInfo, err := types.DeserializeHeaderExtraInformation(prevHeader)
	if err != nil {
		return nil, err
	}
	return &validationEntry{
		BlockNumber:   header.Number.Uint64(),
		BlockHash:     header.Hash(),
		SendRoot:      extraInfo.SendRoot,
		PrevSendRoot:  prevExtraInfo.SendRoot,
		PrevBlockHash: header.ParentHash,
		BlockHeader:   header,
		Preimages:     preimages,
		HasDelayedMsg: hasDelayed,
		DelayedMsgNr:  delayedMsgNr,
	}, nil
}

func NewBlockValidator(inbox InboxTrackerInterface, streamer TransactionStreamerInterface, blockchain *core.BlockChain, db ethdb.Database, config *BlockValidatorConfig, das das.DataAvailabilityService) (*BlockValidator, error) {
	CreateHostIoMachine()
	concurrent := config.ConcurrentRunsLimit
	if concurrent == 0 {
		concurrent = runtime.NumCPU()
	}
	genesisBlockNum, err := streamer.GetGenesisBlockNumber()
	if err != nil {
		return nil, err
	}
	validator := &BlockValidator{
		inboxTracker:            inbox,
		blockchain:              blockchain,
		db:                      db,
		sendValidationsChan:     make(chan struct{}, 1),
		checkProgressChan:       make(chan struct{}, 1),
		progressChan:            make(chan uint64, 1),
		reorgToBlockChan:        make(chan uint64, 1),
		blockReorgCompletedChan: make(chan struct{}),
		concurrentRunsLimit:     int32(concurrent),
		config:                  config,
		das:                     das,
		genesisBlockNum:         genesisBlockNum,
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
	exists, err := v.db.Has(lastBlockValidatedInfoKey)
	if err != nil {
		return err
	}

	if !exists {
		// The db contains no validation info; start from the beginning.
		// TODO: this skips validating the genesis block.
		v.lastBlockValidated = v.genesisBlockNum
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

	v.lastBlockValidated = info.BlockNumber
	v.nextBlockToValidate = v.lastBlockValidated + 1
	v.globalPosNextSend = info.AfterPosition

	return nil
}

// If msg is nil, this will record block creation up to the point where message would be accessed (for a "too far" proof)
func RecordBlockCreation(blockchain *core.BlockChain, prevHeader *types.Header, msg *arbstate.MessageWithMetadata) (common.Hash, map[common.Hash][]byte, error) {
	recordingdb, chaincontext, recordingKV, err := arbitrum.PrepareRecording(blockchain, prevHeader)
	if err != nil {
		return common.Hash{}, nil, err
	}

	chainConfig := blockchain.Config()

	// Get the chain ID, both to validate and because the replay binary also gets the chain ID,
	// so we need to populate the recordingdb with preimages for retrieving the chain ID.
	if prevHeader != nil {
		initialArbosState, err := arbosState.OpenSystemArbosState(recordingdb, true)
		if err != nil {
			return common.Hash{}, nil, fmt.Errorf("Error opening initial ArbOS state: %w", err)
		}
		chainId, err := initialArbosState.ChainId()
		if err != nil {
			return common.Hash{}, nil, fmt.Errorf("Error getting chain ID from initial ArbOS state: %w", err)
		}
		if chainId.Cmp(chainConfig.ChainID) != 0 {
			return common.Hash{}, nil, fmt.Errorf("Unexpected chain ID %v in ArbOS state, expected %v", chainId, chainConfig.ChainID)
		}
	}

	var blockHash common.Hash
	if msg != nil {
		block, _ := arbos.ProduceBlock(
			msg.Message,
			msg.DelayedMessagesRead,
			prevHeader,
			recordingdb,
			chaincontext,
			chainConfig,
		)
		blockHash = block.Hash()
	}

	preimages, err := arbitrum.PreimagesFromRecording(chaincontext, recordingKV)

	return blockHash, preimages, err
}

func BlockDataForValidation(blockchain *core.BlockChain, header, prevHeader *types.Header, msg arbstate.MessageWithMetadata) (preimages map[common.Hash][]byte, hasDelayedMessage bool, delayedMsgNr uint64, err error) {
	var prevHash common.Hash
	if prevHeader != nil {
		prevHash = prevHeader.Hash()
	}
	if header.ParentHash != prevHash {
		err = fmt.Errorf("bad arguments: prev does not match")
		return
	}

	var blockhash common.Hash
	blockhash, preimages, err = RecordBlockCreation(blockchain, prevHeader, &msg)
	if err != nil {
		return
	}
	if blockhash != header.Hash() {
		err = fmt.Errorf("wrong hash expected %s got %s", header.Hash(), blockhash)
		return
	}
	if prevHeader == nil || header.Nonce != prevHeader.Nonce {
		hasDelayedMessage = true
		if prevHeader != nil {
			delayedMsgNr = prevHeader.Nonce.Uint64()
		}
	}
	return
}

func (v *BlockValidator) prepareBlock(header *types.Header, prevHeader *types.Header, msg arbstate.MessageWithMetadata, validationStatus *validationStatus) {
	preimages, hasDelayedMessage, delayedMsgToRead, err := BlockDataForValidation(v.blockchain, header, prevHeader, msg)
	if err != nil {
		log.Error("failed to set up validation", "err", err, "header", header, "prevHeader", prevHeader)
		return
	}
	hashlist := v.preimageCache.PourToCache(preimages)
	validationEntry, err := newValidationEntry(prevHeader, header, hasDelayedMessage, delayedMsgToRead, hashlist)
	if err != nil {
		log.Error("failed to create validation entry", "err", err, "header", header, "prevHeader", prevHeader)
		return
	}
	validationStatus.Entry = validationEntry
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
	v.validationEntries.Store(block.NumberU64(), status)
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
		panic("attempted to validate unprepared validation entry")
	}
	entry := validationStatus.Entry
	log.Info("starting validation for block", "blockNr", entry.BlockNumber)
	preimages, err := v.preimageCache.FillHashedValues(entry.Preimages)
	if err != nil {
		log.Error("validator: failed prepare arrays", "err", err)
		return
	}
	defer (func() {
		err := v.preimageCache.RemoveFromCache(entry.Preimages)
		if err != nil {
			log.Error("validator failed to remove from cache", "err", err)
		}
	})()
	start := entry.StartPosition
	end := entry.EndPosition
	gsStart := GoGlobalState{
		Batch:      start.BatchNumber,
		PosInBatch: start.PosInBatch,
		BlockHash:  entry.PrevBlockHash,
		SendRoot:   entry.PrevSendRoot,
	}

	if arbstate.IsDASMessageHeaderByte(seqMsg[40]) {
		if v.das == nil {
			log.Error("No DAS configured, but sequencer message found with DAS header")
		} else {
			cert, _, err := arbstate.DeserializeDASCertFrom(seqMsg[40:])
			if err != nil {
				log.Error("Failed to deserialize DAS message", "err", err)
			} else {
				preimages[common.BytesToHash(cert.DataHash[:])], err = v.das.Retrieve(ctx, seqMsg[40:])
				if err != nil {
					log.Error("Couldn't retrieve message from DAS", "err", err)
					return
				}
			}
		}
	}

	basemachine, err := GetHostIoMachine(ctx)
	if err != nil {
		return
	}
	mach := basemachine.Clone()
	err = mach.AddPreimages(preimages)
	if err != nil {
		log.Error("error while adding preimage for proving", "err", err, "gsStart", gsStart)
		return
	}
	err = mach.SetGlobalState(gsStart)
	if err != nil {
		log.Error("error while setting global state for proving", "err", err, "gsStart", gsStart)
		return
	}
	err = mach.AddSequencerInboxMessage(start.BatchNumber, seqMsg)
	if err != nil {
		log.Error("error while trying to add sequencer msg for proving", "err", err, "seq", start.BatchNumber, "blockNr", entry.BlockNumber)
		return
	}
	var delayedMsg []byte
	if entry.HasDelayedMsg {
		delayedMsg, err = v.inboxTracker.GetDelayedMessageBytes(entry.DelayedMsgNr)
		if err != nil {
			log.Error("error while trying to read delayed msg for proving", "err", err, "seq", entry.DelayedMsgNr, "blockNr", entry.BlockNumber)
			return
		}
		err = mach.AddDelayedInboxMessage(entry.DelayedMsgNr, delayedMsg)
		if err != nil {
			log.Error("error while trying to add delayed msg for proving", "err", err, "seq", entry.DelayedMsgNr, "blockNr", entry.BlockNumber)
			return
		}
	}

	var steps uint64
	for mach.IsRunning() {
		var count uint64 = 500000000
		err = mach.Step(ctx, count)
		if steps > 0 {
			log.Info("validation", "block", entry.BlockNumber, "steps", steps)
		}
		if err != nil {
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				log.Error("running machine failed", "err", err)
				panic("Failed to run machine: " + err.Error())
			}
			return
		}
		steps += count
	}
	if mach.IsErrored() {
		log.Error("machine entered errored state during attempted validation", "block", entry.BlockNumber)
		return
	}
	gsEnd := mach.GetGlobalState()

	gsExpected := GoGlobalState{Batch: end.BatchNumber, PosInBatch: end.PosInBatch, BlockHash: entry.BlockHash, SendRoot: entry.SendRoot}

	writeThisBlock := false

	resultValid := (gsEnd == gsExpected)

	if !resultValid {
		writeThisBlock = true
	}
	// stupid search for now, assuming the list will always be empty or very mall
	for _, blockNr := range v.config.BlocksToRecord {
		if blockNr > entry.BlockNumber {
			break
		}
		if blockNr == entry.BlockNumber {
			writeThisBlock = true
			break
		}
	}

	if writeThisBlock {
		err = v.writeToFile(entry, start, end, preimages, seqMsg, delayedMsg)
		if err != nil {
			log.Error("failed to write file", "err", err)
		}
	}

	atomic.AddInt32(&v.atomicValidationsRunning, -1)
	entry.Preimages = nil

	if !resultValid {
		log.Error("validation failed", "got", gsEnd, "expected", gsExpected, "expHeader", entry.BlockHeader)
		return
	}

	atomic.StoreUint32(&validationStatus.Status, validationStatusValid) // after that - validation entry could be deleted from map
	log.Info("validation succeeded", "blockNr", entry.BlockNumber)
	v.checkProgressChan <- struct{}{}
	v.sendValidationsChan <- struct{}{}
}

func (v *BlockValidator) sendValidations(ctx context.Context) {
	var batchCount uint64
	for len(v.reorgToBlockChan) == 0 {
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
		validationStatus.Entry.SeqMsgNr = startPos.BatchNumber
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

func (v *BlockValidator) writeCompletedValidationToDb(entry *validationEntry) error {
	info := lastBlockValidatedDbInfo{
		BlockNumber:   entry.BlockNumber,
		AfterPosition: entry.EndPosition,
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
	for len(v.reorgToBlockChan) == 0 {
		// Reads from blocksValidated can be non-atomic as this goroutine is the only writer
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
		earliestBatchKept := atomic.LoadUint64(&v.earliestBatchKept)
		if earliestBatchKept < validationEntry.SeqMsgNr {
			for batch := earliestBatchKept; batch < validationEntry.SeqMsgNr; batch++ {
				v.sequencerBatches.Delete(batch)
			}
			atomic.StoreUint64(&v.earliestBatchKept, validationEntry.SeqMsgNr)
		}
		v.lastBlockValidated = checkingBlock
		v.validationEntries.Delete(checkingBlock)
		select {
		case v.progressChan <- checkingBlock:
		default:
		}
		err := v.writeCompletedValidationToDb(validationEntry)
		if err != nil {
			log.Error("failed to write validated entry to database", "err", err)
		}
	}
}

func (v *BlockValidator) LastBlockValidated() uint64 {
	return atomic.LoadUint64(&v.lastBlockValidated)
}

// Must not be called concurrently with ProcessBatches
func (v *BlockValidator) ReorgToBatchCount(count uint64) {
	localBatchCount := v.nextBatchKept
	if localBatchCount >= count {
		return
	}
	for i := count; i < localBatchCount; i++ {
		v.sequencerBatches.Delete(i)
	}
	v.nextBatchKept = count
}

// Must not be called concurrently with ReorgToBatchCount
func (v *BlockValidator) ProcessBatches(batches map[uint64][]byte, first uint64, next uint64) {
	v.ReorgToBatchCount(first)

	// Attempt to fill in earliestBatchKept if it's empty
	atomic.CompareAndSwapUint64(&v.earliestBatchKept, 0, first)

	for batchNr, msg := range batches {
		v.sequencerBatches.Store(batchNr, msg)
	}
	v.nextBatchKept = next

	select {
	case v.sendValidationsChan <- struct{}{}:
	default:
	}
}

func (v *BlockValidator) ReorgToBlock(blockNum uint64) {
	v.blockMutex.Lock()
	defer v.blockMutex.Unlock()
	v.reorgToBlockChan <- blockNum
	<-v.blockReorgCompletedChan
}

func (v *BlockValidator) reorgToBlockImpl(blockNum uint64) error {
	for b := blockNum + 1; b < v.nextBlockToValidate; b++ {
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
		validationStatus.Cancel()
	}
	v.nextBlockToValidate = blockNum + 1
	msgIndex := arbutil.BlockNumberToMessageCount(blockNum, v.genesisBlockNum) - 1
	batchCount, err := v.inboxTracker.GetBatchCount()
	if err != nil {
		return err
	}
	batch, err := FindBatchContainingMessageIndex(v.inboxTracker, msgIndex, batchCount)
	if err != nil {
		return err
	}
	if batch == batchCount {
		v.globalPosNextSend = GlobalStatePosition{
			BatchNumber: batch,
			PosInBatch:  0,
		}
		msgCount, err := v.inboxTracker.GetBatchMessageCount(batch - 1)
		if err != nil {
			return err
		}
		nextBlockSigned := arbutil.MessageCountToBlockNumber(msgCount, v.genesisBlockNum)
		if nextBlockSigned <= 0 {
			return fmt.Errorf("reorg to impossible block to validate %v", nextBlockSigned)
		}
		v.nextBlockToValidate = uint64(nextBlockSigned)
		if v.lastBlockValidated >= v.nextBlockToValidate {
			v.lastBlockValidated = v.nextBlockToValidate - 1
		}
	} else {
		_, v.globalPosNextSend, err = GlobalStatePositionsFor(v.inboxTracker, msgIndex, batch)
		if err != nil {
			return err
		}
	}
	if v.lastBlockValidated > blockNum {
		v.lastBlockValidated = blockNum
	}
	return nil
}

func (v *BlockValidator) Start(ctxIn context.Context) error {
	v.StopWaiter.Start(ctxIn)
	v.LaunchThread(func(ctx context.Context) {
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
			case blockNum, ok := <-v.reorgToBlockChan:
				if !ok {
					return
				}
				if blockNum+1 < v.nextBlockToValidate {
					log.Warn("block validator canceling validations to reorg", "blockNum", blockNum)
					for {
						err := v.reorgToBlockImpl(blockNum)
						if err != nil {
							log.Error("block validator reorg failed", "err", err)
						} else {
							break
						}
					}
				}
				v.blockReorgCompletedChan <- struct{}{}
			case <-ctx.Done():
				return
			}
		}
	})
	return nil
}

// can only be used from One thread
func (v *BlockValidator) WaitForBlock(blockNumber uint64, timeout time.Duration) bool {
	timeoutChan := time.After(timeout)
	for {
		if atomic.LoadUint64(&v.lastBlockValidated) >= blockNumber {
			return true
		}
		select {
		case <-timeoutChan:
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
