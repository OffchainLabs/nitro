//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package validator

/*
#cgo CFLAGS: -g -Wall -I../arbitrator/target/env/include/
#include "arbitrator.h"
#include <stdlib.h>
*/
import "C"
import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/pkg/errors"
)

type BlockValidator struct {
	inboxTracker MessageReader
	blockchain   *core.BlockChain

	validationEntries  sync.Map
	sequencerBatches   sync.Map
	preimageCache      preimageCache
	posToValidate      posToValidateList
	posToValidateMutex sync.Mutex

	blocksValidated      uint64
	blocksValidatedMutex sync.Mutex
	earliestBatchKept    uint64
	posNextSend          uint64

	config                   *BlockValidatorConfig
	atomicValidationsRunning int32
	concurrentRunsLimit      int32

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
	OutputPath:          "./arbitrator/target/env/output",
	ConcurrentRunsLimit: 0,
	BlocksToRecord:      []uint64{},
}

type PosInSequencer struct {
	Pos        uint64
	BatchNum   uint64
	PosInBatch uint64
	BatchAfter uint64
	PosAfter   uint64
}

type BlockValidatorRegistrer interface {
	SetBlockValidator(*BlockValidator)
}

type MessageReader interface {
	BlockValidatorRegistrer
	GetDelayedMessageBytes(uint64) ([]byte, error)
	GetBatchMetadata(seqNum uint64) (common.Hash, uint64, uint64, error)
}

type validationEntry struct {
	BlockNumber   uint64
	PrevBlockHash common.Hash
	BlockHash     common.Hash
	BlockHeader   *types.Header
	Preimages     []common.Hash
	HasDelayedMsg bool
	DelayedMsgNr  uint64
	SeqMsgNr      uint64
	Running       bool
	Valid         bool
}

func newValidationEntry(header *types.Header, hasDelayed bool, delayedMsgNr uint64, preimages []common.Hash) *validationEntry {
	return &validationEntry{
		BlockNumber:   header.Number.Uint64(),
		BlockHash:     header.Hash(),
		PrevBlockHash: header.ParentHash,
		BlockHeader:   header,
		Preimages:     preimages,
		HasDelayedMsg: hasDelayed,
		DelayedMsgNr:  delayedMsgNr,
	}
}

type posToValidateList []PosInSequencer

func (l posToValidateList) Len() int {
	return len(l)
}

func (l posToValidateList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l posToValidateList) Less(i, j int) bool {
	return l[i].Pos < l[j].Pos
}

// we search for pos that should be close to start - so stupid is best
func (l posToValidateList) StupidSearchPos(pos uint64) int {
	idx := 0
	for (idx < len(l)) && (l[idx].Pos < pos) {
		idx++
	}
	return idx
}

func NewBlockValidator(inbox MessageReader, streamer BlockValidatorRegistrer, blockchain *core.BlockChain, config *BlockValidatorConfig) *BlockValidator {
	CreateHostIoMachine()
	concurrent := config.ConcurrentRunsLimit
	if concurrent == 0 {
		concurrent = runtime.NumCPU()
	}
	validator := &BlockValidator{
		inboxTracker:        inbox,
		blockchain:          blockchain,
		posNextSend:         0,
		sendValidationsChan: make(chan interface{}),
		checkProgressChan:   make(chan interface{}),
		progressChan:        make(chan uint64),
		concurrentRunsLimit: int32(concurrent),
		config:              config,
	}
	streamer.SetBlockValidator(validator)
	inbox.SetBlockValidator(validator)
	return validator
}

func RecordBlockCreation(blockchain *core.BlockChain, prevHeader *types.Header, msg arbstate.MessageWithMetadata) (common.Hash, map[common.Hash][]byte, error) {
	recordingdb, chaincontext, recordingKV, err := arbitrum.PrepareRecording(blockchain, prevHeader)
	if err != nil {
		return common.Hash{}, nil, err
	}

	block, _ := arbos.ProduceBlock(
		msg.Message,
		msg.DelayedMessagesRead,
		prevHeader,
		recordingdb,
		chaincontext,
	)

	preimages, err := arbitrum.PreimagesFromRecording(chaincontext, recordingKV)

	return block.Hash(), preimages, err
}

func (v *BlockValidator) prepareBlock(header *types.Header, prevHeader *types.Header, msg arbstate.MessageWithMetadata) {
	blockhash, preimages, err := RecordBlockCreation(v.blockchain, prevHeader, msg)
	if err != nil {
		log.Error("failed getting records", "err", err, "blocknum", header.Number)
		return
	}
	if blockhash != header.Hash() {
		log.Error("wrong hash while recording", "expected", header.Hash(), "got", blockhash, "blocknum", header.Number)
		return
	}

	hashlist := v.preimageCache.PourToCache(preimages)
	var delayedMsgToRead uint64
	var hasDelayedMessage bool
	if header.ParentHash != prevHeader.Hash() {
		log.Error("prepareBlock: wrong headers", "num", header.Number, "parenthash", header.ParentHash, "prevhash", prevHeader.Hash())
		return
	}
	if header.Nonce != prevHeader.Nonce {
		hasDelayedMessage = true
		delayedMsgToRead = prevHeader.Nonce.Uint64()
	}
	v.validationEntries.Store(header.Number.Uint64(), newValidationEntry(header, hasDelayedMessage, delayedMsgToRead, hashlist))
	v.sendValidationsChan <- struct{}{}
}

func (v *BlockValidator) NewBlock(block *types.Block, prevHeader *types.Header, msg arbstate.MessageWithMetadata) {
	go v.prepareBlock(block.Header(), prevHeader, msg)
}

var launchTime = time.Now().Format("2006_01_02__15_04")

func (v *BlockValidator) writeToFile(validationEntry *validationEntry, pos PosInSequencer, c_preimages C.CMultipleByteArrays, sequencerCByte C.CByteArray, delayedCByte C.CByteArray) error {
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
		fmt.Sprintf("# expected output: batch %d, postion %d, hash %s\n", pos.BatchAfter, pos.PosAfter, validationEntry.BlockHash) +
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
	_, err = cmdFile.WriteString(fmt.Sprintf(" --inbox-position %d --position-within-message %d --last-block-hash %s", validationEntry.SeqMsgNr, pos.PosInBatch, validationEntry.PrevBlockHash))
	if err != nil {
		return err
	}

	sequencerFileName := fmt.Sprintf("sequencer_%d.bin", validationEntry.SeqMsgNr)
	err = CByteToFile(sequencerCByte, filepath.Join(outDirPath, sequencerFileName))
	if err != nil {
		return err
	}
	_, err = cmdFile.WriteString(" --inbox " + sequencerFileName)
	if err != nil {
		return err
	}

	err = CMultipleByteArrayToFile(c_preimages, filepath.Join(outDirPath, "preimages.bin"))
	if err != nil {
		return err
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
		err = CByteToFile(delayedCByte, filepath.Join(outDirPath, filename))
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

func (v *BlockValidator) validate(ctx context.Context, validationEntry *validationEntry, pos PosInSequencer) {
	if validationEntry.BlockNumber != pos.Pos+1 {
		log.Error("bad validation", "blockNr", validationEntry.BlockNumber, "pos", pos.Pos)
		return
	}
	log.Info("starting validation for block", "blockNr", validationEntry.BlockNumber)
	c_preimages, err := v.preimageCache.PrepareMultByteArrays(validationEntry.Preimages)
	defer C.free(unsafe.Pointer(c_preimages.ptr))
	if err != nil {
		log.Error("validator: failed prepare arrays", "err", err)
		return
	}
	validationEntry.SeqMsgNr = pos.BatchNum
	validationEntry.Running = true
	gsStart := CreateGlobalState(pos.BatchNum, pos.PosInBatch, validationEntry.PrevBlockHash)

	seqEntry, found := v.sequencerBatches.Load(pos.BatchNum)
	if !found {
		log.Error("didn't find sequencer message", "blockNr", validationEntry.BlockNumber, "msgNum", validationEntry.SeqMsgNr)
		return
	}
	seqCByte, ok := seqEntry.(C.CByteArray)
	if !ok {
		log.Error("sequencer message bad format", "blockNr", validationEntry.BlockNumber, "msgNum", validationEntry.SeqMsgNr)
		return
	}

	basemachine, err := GetHostIoMachine(ctx)
	if err != nil {
		return
	}
	mach := basemachine.Clone()
	C.arbitrator_add_preimages(mach.ptr, c_preimages)
	err = mach.SetGlobalState(gsStart)
	if err != nil {
		log.Error("error while tsetting global state for proving", "err", err, "gsStart", gsStart)
		return
	}
	err = mach.AddSequencerInboxMessage(validationEntry.SeqMsgNr, seqCByte)
	if err != nil {
		log.Error("error while trying to add sequencer msg for proving", "err", err, "seq", validationEntry.SeqMsgNr, "blockNr", validationEntry.BlockNumber)
		return
	}
	var delayedByte C.CByteArray
	if validationEntry.HasDelayedMsg {
		msg, err := v.inboxTracker.GetDelayedMessageBytes(validationEntry.DelayedMsgNr)
		if err != nil {
			log.Error("error while trying to read delayed msg for proving", "err", err, "seq", validationEntry.DelayedMsgNr, "blockNr", validationEntry.BlockNumber)
			return
		}
		delayedByte = CreateCByteArray(msg)
		defer DestroyCByteArray(delayedByte)
		err = mach.AddDelayedInboxMessage(validationEntry.DelayedMsgNr, delayedByte)
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
	gsEnd := mach.GetGlobalState()

	resBatch, resPosInBatch, resHash := ParseGlobalState(gsEnd)

	writeThisBlock := false

	resultValid := (resBatch == pos.BatchAfter) && (resPosInBatch == pos.PosAfter) && (resHash == validationEntry.BlockHash)

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
		err = v.writeToFile(validationEntry, pos, c_preimages, seqCByte, delayedByte)
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
		log.Error("validation failed", "blockNr", validationEntry.BlockNumber, "batch_exp", pos.BatchAfter, "batch_actual", resBatch, "pos_exp", pos.PosAfter, "pos_actual", resPosInBatch, "hash_exp", validationEntry.BlockHash, "hash_actual", resHash)
		log.Error("validation failed", "expHeader", validationEntry.BlockHeader)
		return
	}

	validationEntry.Valid = true // after that - validation entry could be deleted from map
	log.Info("validation succeeded", "blockNr", validationEntry.BlockNumber)
	v.checkProgressChan <- struct{}{}
	v.sendValidationsChan <- struct{}{}
}

func (v *BlockValidator) sendValidations(ctx context.Context) {
	v.posToValidateMutex.Lock()
	defer v.posToValidateMutex.Unlock()
	sort.Sort(v.posToValidate)

	idx := v.posToValidate.StupidSearchPos(v.posNextSend)
	v.posToValidate = v.posToValidate[idx:]

	for {
		if atomic.LoadInt32(&v.atomicValidationsRunning) >= v.concurrentRunsLimit {
			return
		}
		if len(v.posToValidate) == 0 || v.posToValidate[0].Pos != v.posNextSend {
			return
		}
		entry, found := v.validationEntries.Load(v.posNextSend + 1)
		if !found {
			return
		}
		validationEntry, ok := entry.(*validationEntry)
		if !ok || (validationEntry == nil) {
			log.Error("bad entry trying to validate batch")
			return
		}
		atomic.AddInt32(&v.atomicValidationsRunning, 1)
		go v.validate(ctx, validationEntry, v.posToValidate[0])
		v.posNextSend += 1
		v.posToValidate = v.posToValidate[1:]
	}
}

func (v *BlockValidator) startValidationLoop(ctx context.Context) {
	go (func() {
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
	})()
}

func (v *BlockValidator) ProgressValidated() {
	v.blocksValidatedMutex.Lock()
	defer v.blocksValidatedMutex.Unlock()
	for {
		entry, found := v.validationEntries.Load(v.blocksValidated + 1)
		if !found {
			return
		}
		validationEntry, ok := entry.(*validationEntry)
		if !ok || (validationEntry == nil) {
			log.Error("bad entry trying to advance validated counter")
			return
		}
		if !validationEntry.Valid {
			return
		}
		if validationEntry.BlockNumber != v.blocksValidated+1 {
			log.Error("bad block number for validation entry", "expected", v.blocksValidated+1, "found", validationEntry.BlockNumber)
			return
		}
		if v.earliestBatchKept < validationEntry.SeqMsgNr {
			for batch := v.earliestBatchKept; batch < validationEntry.SeqMsgNr; batch++ {
				entry, found := v.sequencerBatches.Load(batch)
				if !found {
					log.Warn("didn't find sequencer batch", "number", batch)
					continue
				}
				cbyte, ok := entry.(C.CByteArray)
				if !ok {
					log.Error("bad entry trying to delete batch", "number", batch)
					continue
				}
				DestroyCByteArray(cbyte)
				v.sequencerBatches.Delete(batch)
			}
			v.earliestBatchKept = validationEntry.SeqMsgNr
		}
		v.blocksValidated += 1
		v.validationEntries.Delete(v.blocksValidated)
		select {
		case v.progressChan <- v.blocksValidated:
		default:
		}
	}
}

func (v *BlockValidator) startProgressLoop(ctx context.Context) {
	go (func() {
		for {
			select {
			case _, ok := <-v.checkProgressChan:
				if !ok {
					return
				}
			case <-ctx.Done():
				return
			}
			v.ProgressValidated()
		}
	})()
}

func (v *BlockValidator) BlocksValidated() uint64 {
	return v.blocksValidated
}

func (v *BlockValidator) ProcessBatches(batches map[uint64][]byte, posData []PosInSequencer) {
	for batchNr, msg := range batches {
		v.sequencerBatches.Store(batchNr, CreateCByteArray(msg))
	}
	v.posToValidateMutex.Lock()
	v.posToValidate = append(v.posToValidate, posData...)
	v.posToValidateMutex.Unlock()
	select {
	case v.sendValidationsChan <- struct{}{}:
	default:
	}
}

func (v *BlockValidator) Start(ctx context.Context) error {
	v.startProgressLoop(ctx)
	v.startValidationLoop(ctx)
	return nil
}

// can only be used from One thread
func (v *BlockValidator) WaitForBlock(blockNumber uint64, timeout time.Duration) bool {
	timeoutChan := time.After(timeout)
	for {
		if v.blocksValidated >= blockNumber {
			return true
		}
		select {
		case <-timeoutChan:
			if v.blocksValidated >= blockNumber {
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
