package validator

/*
#cgo CFLAGS: -g -Wall
#include "c-api/arbitrator.h"
#include <stdlib.h>

struct CByteArray InboxReaderWrapper(uint64_t inbox_idx, uint64_t seq_num);
*/
import "C"
import (
	"context"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type BlockValidator struct {
	preimageCache      preimageCache
	posToValidate      posToValidateList
	posToValidateMutex sync.Mutex
	posNext            uint64
	batchNrValidated   uint64
	blocksValidated    uint64
	posValidatedMutex  sync.Mutex
	posNextSend        uint64

	baseMachine *C.struct_Machine

	atomicValidationsRunning int32
	concurrentRunsLimit      int32

	sendValidationsChan chan interface{}
	checkProgressChan   chan interface{}
	progressChan        chan uint64
}

type PosInSequencer struct {
	Pos           uint64
	BatchNum      uint64
	PosInSequence uint64
	BatchAfter    uint64
	PosAfter      uint64
}

type BlockValidatorRegistrer interface {
	SetBlockValidator(*BlockValidator)
}

type DelayedMessageReader interface {
	BlockValidatorRegistrer
	GetDelayedMessageBytes(uint64) ([]byte, error)
}

// block validator interacts with c, so some functions don't have specific conext and must use globals
type blockValidatorGlobals struct {
	initialized       bool
	validationEntries sync.Map
	sequencerBatches  sync.Map
	inboxTracker      DelayedMessageReader
}

var validatorStatic blockValidatorGlobals

type validationEntry struct {
	BlockNumber   uint64
	PrevBlockHash common.Hash
	BlockHash     common.Hash
	BlockHeader   *types.Header
	Preimages     []common.Hash
	EndPos        uint64
	Running       bool
	StartBatchNr  uint64
	MsgsAllocated []C.CByteArray
	Valid         bool
}

func newValidationEntry(header *types.Header, preimages []common.Hash, endPos uint64) *validationEntry {
	return &validationEntry{
		BlockNumber:   header.Number.Uint64(),
		BlockHash:     header.Hash(),
		PrevBlockHash: header.ParentHash,
		BlockHeader:   header,
		Preimages:     preimages,
		EndPos:        endPos,
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

func NewBlockValidator(inbox DelayedMessageReader, streamer BlockValidatorRegistrer) *BlockValidator {
	rootPath := "/home/tsahee/src/nitro/prover-env/"
	moduleList := []string{rootPath + "wasi_stub.wasm", rootPath + "soft-float.wasm", rootPath + "go_stub.wasm", rootPath + "host_io.wasm"}
	cModuleList := CreateCStringList(moduleList)
	cBinPath := C.CString(rootPath + "replay.wasm")

	cZeroPreimages := C.CMultipleByteArrays{}
	cZeroPreimages.len = 0
	baseMachine := C.arbitrator_load_machine(cBinPath, cModuleList, C.intptr_t(len(moduleList)), C.GlobalState{}, cZeroPreimages, (*[0]byte)(C.InboxReaderWrapper))
	if validatorStatic.initialized {
		panic("creating block validator when one exists")
	}
	validatorStatic.inboxTracker = inbox
	validatorStatic.initialized = true

	validator := &BlockValidator{
		posNextSend:         0,
		sendValidationsChan: make(chan interface{}),
		checkProgressChan:   make(chan interface{}),
		progressChan:        make(chan uint64),
		baseMachine:         baseMachine,
		concurrentRunsLimit: 8,
	}
	streamer.SetBlockValidator(validator)
	inbox.SetBlockValidator(validator)
	return validator
}

func (v *BlockValidator) prepareBlock(header *types.Header, preimages map[common.Hash][]byte, startPos uint64, endPos uint64) {
	hashlist := v.preimageCache.PourToCache(preimages)
	validatorStatic.validationEntries.Store(startPos, newValidationEntry(header, hashlist, endPos))
	v.sendValidationsChan <- struct{}{}
}

func (v *BlockValidator) NewBlock(block *types.Block, preimages map[common.Hash][]byte, startPos uint64, endPos uint64) {
	go v.prepareBlock(block.Header(), preimages, startPos, endPos)
}

// this func cannot be in a file where the C premable has anything other than declarations
//export InboxReaderFunc
func InboxReaderFunc(c_context C.uint64_t, c_inbox_idx C.uint64_t, c_seq_num C.uint64_t) C.CByteArray {
	index := uint64(c_inbox_idx)
	msgNum := uint64(c_seq_num)
	startPos := uint64(c_context)
	if index == 0 {
		entry, found := validatorStatic.sequencerBatches.Load(msgNum)
		if !found {
			log.Error("didn't find sequencer message", "pos", startPos, "msgNum", msgNum)
			runtime.Goexit()
		}
		cbyte, ok := entry.(C.CByteArray)
		if !ok {
			log.Error("sequencer message bad format", "pos", startPos, "msgNum", msgNum)
			runtime.Goexit()
		}
		return cbyte
	} else if index == 1 {
		entry, found := validatorStatic.validationEntries.Load(startPos)
		if !found {
			log.Error("error while trying to read validation entry", "pos", startPos)
			runtime.Goexit()
		}
		validationEntry := entry.(*validationEntry)
		msg, err := validatorStatic.inboxTracker.GetDelayedMessageBytes(msgNum)
		if err != nil {
			log.Error("error while trying to read delayed msg for proving", "err", err, "seq", msgNum, "pos", startPos)
			runtime.Goexit()
		}
		cByte := CreateCByteArray(msg)
		validationEntry.MsgsAllocated = append(validationEntry.MsgsAllocated, cByte)
		return cByte
	} else {
		log.Error("bad inbox index while proving", "index", index, "pos", startPos)
		runtime.Goexit()
	}
	return C.CByteArray{} //will never get here, parsers don't realise Goexit is dead end
}

func (v *BlockValidator) validate(validationEntry *validationEntry, start, end PosInSequencer) {
	log.Info("starting validation for block", "blockNr", validationEntry.BlockNumber, "start", start, "end", end)
	if !validatorStatic.initialized {
		log.Error("validator: validatorStatic not initialized")
		return
	}
	if validationEntry.EndPos != end.Pos {
		log.Error("validator: validate got bad args", "block.end", validationEntry.EndPos, "end", end.Pos)
		return
	}
	c_preimages, err := v.preimageCache.PrepareMultByteArrays(validationEntry.Preimages)
	if err != nil {
		log.Error("validator: failed prepare arrays", "err", err)
		return
	}
	validationEntry.Running = true
	validationEntry.StartBatchNr = start.BatchNum
	gsStart := CreateGlobalState(start.BatchNum, start.PosInSequence, validationEntry.PrevBlockHash)

	mach := C.arbitrator_clone_machine(v.baseMachine)
	C.arbitrator_add_preimages(mach, c_preimages)
	C.arbitrator_set_inbox_reader_context(mach, C.uint64_t(start.Pos))
	C.arbitrator_set_global_state(mach, gsStart)
	steps := 0
	for !C.arbitrator_is_halted(mach) {
		C.arbitrator_step(mach, C.intptr_t(100000000))
		steps += 100000000
		log.Info("validation", "block", validationEntry.BlockNumber, "steps", steps)
	}
	gsEnd := C.arbitrator_global_state(mach)

	resBatch, resPosInSequence, resHash := ParseGlobalState(gsEnd)

	if (resBatch != end.BatchAfter) || (resPosInSequence != end.PosAfter) || (resHash != validationEntry.BlockHash) {
		log.Error("validation failed", "startPos", start.Pos, "batch_exp", end.BatchAfter, "batch_actual", resBatch, "pos_exp", end.PosAfter, "pos_actual", resPosInSequence, "hash_exp", validationEntry.BlockHash, "hash_actual", resHash)
		log.Error("validation failed", "expHeader", validationEntry.BlockHeader)
		panic("validation failed. quitting..")
	}
	v.preimageCache.RemoveFromCache(validationEntry.Preimages)
	for _, cbyte := range validationEntry.MsgsAllocated {
		DestroyCByteArray(cbyte)
	}
	atomic.AddInt32(&v.atomicValidationsRunning, -1)
	validationEntry.MsgsAllocated = nil
	validationEntry.Preimages = nil
	validationEntry.Valid = true //after that - validation entry could be deleted from map
	log.Info("validation succeeded", "blockNr", validationEntry.BlockNumber)
	v.checkProgressChan <- struct{}{}
	v.sendValidationsChan <- struct{}{}
}

func (v *BlockValidator) sendValidations() {
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
		entry, found := validatorStatic.validationEntries.Load(v.posNextSend)
		if !found {
			return
		}
		validationEntry, ok := entry.(*validationEntry)
		if !ok || (validationEntry == nil) {
			log.Error("bad entry trying to validate batch")
			return
		}
		idx = v.posToValidate.StupidSearchPos(validationEntry.EndPos)
		if len(v.posToValidate) <= idx || v.posToValidate[idx].Pos != validationEntry.EndPos {
			return
		}
		atomic.AddInt32(&v.atomicValidationsRunning, 1)
		go v.validate(validationEntry, v.posToValidate[0], v.posToValidate[idx])
		v.posNextSend = validationEntry.EndPos + 1
		v.posToValidate = v.posToValidate[idx+1:]
	}
}

func (v *BlockValidator) startValidationLoop() {
	go (func() {
		for {
			_, ok := <-v.sendValidationsChan
			if !ok {
				return
			}
			v.sendValidations()
		}
	})()
}

func (v *BlockValidator) ProgressValidated() {
	v.posValidatedMutex.Lock()
	defer v.posValidatedMutex.Unlock()
	for {
		entry, found := validatorStatic.validationEntries.Load(v.posNext)
		if !found {
			log.Info("validator progress: not in db:", "pos", v.posNext) //TEMP
			return
		}
		validationEntry, ok := entry.(*validationEntry)
		if !ok || (validationEntry == nil) {
			log.Error("bad entry trying to advance validated counter")
			return
		}
		if !validationEntry.Valid {
			log.Info("validator progress: not valid:", "pos", v.posNext) //TEMP
			return
		}
		if validationEntry.BlockNumber != v.blocksValidated+1 {
			log.Error("bad block number for validation entry", "expected", v.blocksValidated+1, "found", validationEntry.BlockNumber, "pos", v.posNext)
			return
		}
		validatorStatic.validationEntries.Delete(v.posNext)
		for batch := v.batchNrValidated; batch < validationEntry.StartBatchNr; batch++ {
			entry, found := validatorStatic.sequencerBatches.LoadAndDelete(batch)
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
		}
		v.posNext = validationEntry.EndPos + 1
		v.blocksValidated = validationEntry.BlockNumber + 1
		v.progressChan <- v.blocksValidated
	}
}

func (v *BlockValidator) startProgressLoop() {
	go (func() {
		for {
			_, ok := <-v.checkProgressChan
			if !ok {
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
		validatorStatic.sequencerBatches.Store(batchNr, CreateCByteArray(msg))
	}
	v.posToValidateMutex.Lock()
	v.posToValidate = append(v.posToValidate, posData...)
	v.posToValidateMutex.Unlock()
	v.sendValidationsChan <- struct{}{}
}

func (v *BlockValidator) Start(_ context.Context) {
	v.startProgressLoop()
	v.startValidationLoop()
}

//can only be used from One thread
func (v *BlockValidator) WaitForBlock(blockNumber uint64, timeout time.Duration) bool {
	timeoutChan := time.After(timeout)
	for {
		select {
		case <-timeoutChan:
			return false
		case block := <-v.progressChan:
			if block >= blockNumber {
				return true
			}
		}
	}
}
