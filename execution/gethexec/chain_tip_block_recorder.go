// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package gethexec

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
)

// ChainTipBlockRecorder serves recordings captured while the chain tip block is
// produced.
type ChainTipBlockRecorder struct {
	execEngine *ExecutionEngine

	lock                sync.Mutex
	enabled             atomic.Bool
	lastRecording       *execution.RecordResult
	servedTipRecordings atomic.Uint64
}

func NewChainTipBlockRecorder(execEngine *ExecutionEngine) *ChainTipBlockRecorder {
	recorder := &ChainTipBlockRecorder{
		execEngine: execEngine,
	}
	execEngine.SetTipRecorder(recorder)
	return recorder
}

func (r *ChainTipBlockRecorder) Enable() {
	r.enabled.Store(true)
}

func (r *ChainTipBlockRecorder) Disable() {
	r.enabled.Store(false)
}

func (r *ChainTipBlockRecorder) Enabled() bool {
	return r != nil && r.enabled.Load()
}

func copyPreimageMap(preimages map[common.Hash][]byte) map[common.Hash][]byte {
	if preimages == nil {
		return nil
	}
	copied := make(map[common.Hash][]byte, len(preimages))
	for hash, preimage := range preimages {
		copied[hash] = common.CopyBytes(preimage)
	}
	return copied
}

func copyUserWasms(userWasms state.UserWasms) state.UserWasms {
	if userWasms == nil {
		return nil
	}
	copied := make(state.UserWasms, len(userWasms))
	for moduleHash, asmMap := range userWasms {
		copiedAsmMap := make(state.ActivatedWasm, len(asmMap))
		for target, asm := range asmMap {
			copiedAsmMap[target] = common.CopyBytes(asm)
		}
		copied[moduleHash] = copiedAsmMap
	}
	return copied
}

func copyRecordResult(record *execution.RecordResult) *execution.RecordResult {
	if record == nil {
		return nil
	}
	return &execution.RecordResult{
		Pos:       record.Pos,
		BlockHash: record.BlockHash,
		Preimages: copyPreimageMap(record.Preimages),
		UserWasms: copyUserWasms(record.UserWasms),
	}
}

func (r *ChainTipBlockRecorder) RecordTip(block *types.Block, preimages map[common.Hash][]byte, userWasms state.UserWasms) {
	if !r.Enabled() || block == nil {
		return
	}
	pos, err := r.execEngine.BlockNumberToMessageIndex(block.NumberU64())
	if err != nil {
		log.Warn("failed to calculate message index for chain-tip recording", "block", block.NumberU64(), "err", err)
		return
	}
	preimages = copyPreimageMap(preimages)
	if block.NumberU64() > 0 {
		prevHeader := r.execEngine.bc.GetHeader(block.ParentHash(), block.NumberU64()-1)
		if prevHeader == nil {
			log.Warn("failed to load previous header for chain-tip recording", "block", block.NumberU64(), "parentHash", block.ParentHash())
			return
		}
		encodedHeader, err := rlp.EncodeToBytes(prevHeader)
		if err != nil {
			log.Warn("failed to encode previous header for chain-tip recording", "block", block.NumberU64(), "err", err)
			return
		}
		if preimages == nil {
			preimages = make(map[common.Hash][]byte)
		}
		preimages[prevHeader.Hash()] = encodedHeader
	}

	record := &execution.RecordResult{
		Pos:       pos,
		BlockHash: block.Hash(),
		Preimages: preimages,
		UserWasms: copyUserWasms(userWasms),
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	r.lastRecording = record
}

func (r *ChainTipBlockRecorder) Recording(pos arbutil.MessageIndex) (*execution.RecordResult, error) {
	if !r.Enabled() {
		return nil, fmt.Errorf("chain-tip recording unavailable for pos %d", pos)
	}
	r.lock.Lock()
	record := copyRecordResult(r.lastRecording)
	r.lock.Unlock()
	if record == nil || record.Pos != pos {
		return nil, fmt.Errorf("chain-tip recording unavailable for pos %d", pos)
	}
	r.servedTipRecordings.Add(1)
	return record, nil
}

func (r *ChainTipBlockRecorder) ServedTipRecordings() uint64 {
	return r.servedTipRecordings.Load()
}
