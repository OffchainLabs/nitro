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
	"github.com/ethereum/go-ethereum/log"
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
	genesisBlockNum uint64

	validationEntries sync.Map
	sequencerBatches  sync.Map
	preimageCache     preimageCache

	blocksValidated   uint64
	earliestBatchKept uint64

	posNextSend       arbutil.MessageIndex
	globalPosNextSend GlobalStatePosition

	config                   *BlockValidatorConfig
	atomicValidationsRunning int32
	concurrentRunsLimit      int32

	das das.DataAvailabilityService

	sendValidationsChan chan interface{}
	checkProgressChan   chan interface{}
	progressChan        chan uint64
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

func GlobalStatePositionsFor(reader InboxTrackerInterface, pos arbutil.MessageIndex, batch uint64) (GlobalStatePosition, GlobalStatePosition, error) {
	msgCountInBatch, err := reader.GetBatchMessageCount(batch)
	if err != nil {
		return GlobalStatePosition{}, GlobalStatePosition{}, err
	}
	var firstInBatch arbutil.MessageIndex
	if batch > 0 {
		firstInBatch, err = reader.GetBatchMessageCount(batch - 1)
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
	Valid         uint32 // Atomic, either 0 or 1
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

func NewBlockValidator(inbox InboxTrackerInterface, streamer TransactionStreamerInterface, blockchain *core.BlockChain, config *BlockValidatorConfig, das das.DataAvailabilityService) (*BlockValidator, error) {
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
		inboxTracker:        inbox,
		blockchain:          blockchain,
		sendValidationsChan: make(chan interface{}),
		checkProgressChan:   make(chan interface{}),
		progressChan:        make(chan uint64),
		concurrentRunsLimit: int32(concurrent),
		config:              config,
		das:                 das,
		genesisBlockNum:     genesisBlockNum,
		// TODO: this skips validating the genesis block
		posNextSend: 1,
		globalPosNextSend: GlobalStatePosition{
			BatchNumber: 1,
		},
	}
	streamer.SetBlockValidator(validator)
	inbox.SetBlockValidator(validator)
	return validator, nil
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

func (v *BlockValidator) prepareBlock(header *types.Header, prevHeader *types.Header, msg arbstate.MessageWithMetadata) {
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
	v.validationEntries.Store(header.Number.Uint64(), validationEntry)
	v.sendValidationsChan <- struct{}{}
}

func (v *BlockValidator) NewBlock(block *types.Block, prevHeader *types.Header, msg arbstate.MessageWithMetadata) {
	v.LaunchUntrackedThread(func() { v.prepareBlock(block.Header(), prevHeader, msg) })
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

func (v *BlockValidator) validate(ctx context.Context, validationEntry *validationEntry, start, end GlobalStatePosition) {
	log.Info("starting validation for block", "blockNr", validationEntry.BlockNumber)
	preimages, err := v.preimageCache.FillHashedValues(validationEntry.Preimages)
	if err != nil {
		log.Error("validator: failed prepare arrays", "err", err)
		return
	}
	gsStart := GoGlobalState{
		Batch:      start.BatchNumber,
		PosInBatch: start.PosInBatch,
		BlockHash:  validationEntry.PrevBlockHash,
		SendRoot:   validationEntry.PrevSendRoot,
	}

	seqEntry, found := v.sequencerBatches.Load(start.BatchNumber)
	if !found {
		log.Error("didn't find sequencer message", "blockNr", validationEntry.BlockNumber, "msgNum", start.BatchNumber)
		return
	}
	seqMsg, ok := seqEntry.([]byte)
	if !ok {
		log.Error("sequencer message bad format", "blockNr", validationEntry.BlockNumber, "msgNum", start.BatchNumber)
		return
	}

	if arbstate.IsDASMessageHeaderByte(seqMsg[40]) {
		if v.das == nil {
			log.Error("No DAS configured, but sequencer message found with DAS header")
			return
		}
		cert, _, err := arbstate.DeserializeDASCertFrom(seqMsg[40:])
		if err != nil {
			log.Error("Failed to deserialize DAS message", "err", err)
			return
		} else {
			preimages[common.BytesToHash(cert.DataHash[:])], err = v.das.Retrieve(ctx, seqMsg[40:])
			if err != nil {
				log.Error("Couldn't retrieve message from DAS", "err", err)
				return
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
		log.Error("error while trying to add sequencer msg for proving", "err", err, "seq", start.BatchNumber, "blockNr", validationEntry.BlockNumber)
		return
	}
	var delayedMsg []byte
	if validationEntry.HasDelayedMsg {
		delayedMsg, err = v.inboxTracker.GetDelayedMessageBytes(validationEntry.DelayedMsgNr)
		if err != nil {
			log.Error("error while trying to read delayed msg for proving", "err", err, "seq", validationEntry.DelayedMsgNr, "blockNr", validationEntry.BlockNumber)
			return
		}
		err = mach.AddDelayedInboxMessage(validationEntry.DelayedMsgNr, delayedMsg)
		if err != nil {
			log.Error("error while trying to add delayed msg for proving", "err", err, "seq", validationEntry.DelayedMsgNr, "blockNr", validationEntry.BlockNumber)
			return
		}
	}

	var steps uint64
	for mach.IsRunning() {
		var count uint64 = 500000000
		err = mach.Step(ctx, count)
		if steps > 0 {
			log.Info("validation", "block", validationEntry.BlockNumber, "steps", steps)
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
		// TODO: remove this panic by making this function fallible
		panic("Machine entered errored state during attempted validation")
	}
	gsEnd := mach.GetGlobalState()

	gsExpected := GoGlobalState{Batch: end.BatchNumber, PosInBatch: end.PosInBatch, BlockHash: validationEntry.BlockHash, SendRoot: validationEntry.SendRoot}

	writeThisBlock := false

	resultValid := (gsEnd == gsExpected)

	if !resultValid {
		writeThisBlock = true
	}
	// stupid search for now, assuming the list will always be empty or very mall
	for _, blockNr := range v.config.BlocksToRecord {
		if blockNr > validationEntry.BlockNumber {
			break
		}
		if blockNr == validationEntry.BlockNumber {
			writeThisBlock = true
			break
		}
	}

	if writeThisBlock {
		err = v.writeToFile(validationEntry, start, end, preimages, seqMsg, delayedMsg)
		if err != nil {
			log.Error("failed to write file", "err", err)
		}
	}

	err = v.preimageCache.RemoveFromCache(validationEntry.Preimages)
	if err != nil {
		log.Error("validator failed to remove from cache", "err", err)
	}
	atomic.AddInt32(&v.atomicValidationsRunning, -1)
	validationEntry.Preimages = nil

	if !resultValid {
		log.Error("validation failed", "got", gsEnd, "expected", gsExpected, "expHeader", validationEntry.BlockHeader)
		return
	}

	atomic.StoreUint32(&validationEntry.Valid, 1) // after that - validation entry could be deleted from map
	log.Info("validation succeeded", "blockNr", validationEntry.BlockNumber)
	v.checkProgressChan <- struct{}{}
	v.sendValidationsChan <- struct{}{}
}

func (v *BlockValidator) sendValidations(ctx context.Context) {
	var batchCount uint64
	for {
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
		_, haveBatch := v.sequencerBatches.Load(v.globalPosNextSend.BatchNumber)
		if !haveBatch {
			return
		}
		// valdationEntries is By blockNumber
		nextBlockNum := uint64(arbutil.MessageCountToBlockNumber(v.posNextSend, v.genesisBlockNum) + 1)
		entry, found := v.validationEntries.Load(nextBlockNum)
		if !found {
			return
		}
		validationEntry, ok := entry.(*validationEntry)
		if !ok || (validationEntry == nil) {
			log.Error("bad entry trying to validate batch")
			return
		}
		startPos, endPos, err := GlobalStatePositionsFor(v.inboxTracker, v.posNextSend, v.globalPosNextSend.BatchNumber)
		if err != nil {
			log.Error("failed calculating position for validation", "err", err, "pos", v.posNextSend, "batch", v.globalPosNextSend.BatchNumber)
			return
		}
		if startPos != v.globalPosNextSend {
			log.Error("inconsistent pos mapping", "uint_pos", v.posNextSend, "expected", v.globalPosNextSend, "found", startPos)
			return
		}
		atomic.AddInt32(&v.atomicValidationsRunning, 1)
		validationEntry.SeqMsgNr = startPos.BatchNumber
		// validation can take long time. Don't wait for it when shutting down
		v.LaunchUntrackedThread(func() { v.validate(v.GetContext(), validationEntry, startPos, endPos) })
		v.posNextSend += 1
		v.globalPosNextSend = endPos
	}
}

func (v *BlockValidator) startValidationLoop() {
	v.LaunchThread(func(ctx context.Context) {
		for {
			select {
			case _, ok := <-v.sendValidationsChan:
				if !ok {
					return
				}
			case <-ctx.Done():
				return
			}
			v.sendValidations(ctx)
		}
	})
}

func (v *BlockValidator) progressValidated() {
	for {
		// Reads from blocksValidated can be non-atomic as this goroutine is the only writer
		entry, found := v.validationEntries.Load(v.blocksValidated + 1)
		if !found {
			return
		}
		validationEntry, ok := entry.(*validationEntry)
		if !ok || (validationEntry == nil) {
			log.Error("bad entry trying to advance validated counter")
			return
		}
		if atomic.LoadUint32(&validationEntry.Valid) == 0 {
			return
		}
		if validationEntry.BlockNumber != v.blocksValidated+1 {
			log.Error("bad block number for validation entry", "expected", v.blocksValidated+1, "found", validationEntry.BlockNumber)
			return
		}
		if v.earliestBatchKept < validationEntry.SeqMsgNr {
			for batch := v.earliestBatchKept; batch < validationEntry.SeqMsgNr; batch++ {
				v.sequencerBatches.Delete(batch)
			}
			v.earliestBatchKept = validationEntry.SeqMsgNr
		}
		atomic.AddUint64(&v.blocksValidated, 1)
		v.validationEntries.Delete(v.blocksValidated)
		select {
		case v.progressChan <- v.blocksValidated:
		default:
		}
	}
}

func (v *BlockValidator) startProgressLoop() {
	v.LaunchThread(func(ctx context.Context) {
		for {
			select {
			case _, ok := <-v.checkProgressChan:
				if !ok {
					return
				}
			case <-ctx.Done():
				return
			}
			v.progressValidated()
		}
	})
}

func (v *BlockValidator) BlocksValidated() uint64 {
	return atomic.LoadUint64(&v.blocksValidated)
}

func (v *BlockValidator) ProcessBatches(batches map[uint64][]byte) {
	for batchNr, msg := range batches {
		v.sequencerBatches.Store(batchNr, msg)
	}
	select {
	case v.sendValidationsChan <- struct{}{}:
	default:
	}
}

func (v *BlockValidator) Start(ctxIn context.Context) error {
	v.StopWaiter.Start(ctxIn)
	v.startProgressLoop()
	v.startValidationLoop()
	return nil
}

// can only be used from One thread
func (v *BlockValidator) WaitForBlock(blockNumber uint64, timeout time.Duration) bool {
	timeoutChan := time.After(timeout)
	for {
		if atomic.LoadUint64(&v.blocksValidated) >= blockNumber {
			return true
		}
		select {
		case <-timeoutChan:
			if atomic.LoadUint64(&v.blocksValidated) >= blockNumber {
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
