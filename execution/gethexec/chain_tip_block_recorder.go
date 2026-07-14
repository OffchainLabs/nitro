// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package gethexec

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
)

const recentHeaderPreimageCacheSlots = 256

type wasmKey struct {
	moduleHash common.Hash
	target     rawdb.WasmTarget
}

type chainTipRecording struct {
	record            *execution.RecordResult
	blockNumber       uint64
	parentHash        common.Hash
	firstHeaderNumber uint64
	codeHashes        []common.Hash
	wasmKeys          []wasmKey
}

// ChainTipBlockRecorder serves recordings captured while the chain tip block is
// produced.
type ChainTipBlockRecorder struct {
	execEngine *ExecutionEngine

	lock sync.Mutex
	// lastRecording is temporary in-memory storage until we implement a Key-Value backend for chain-tip recordings
	lastRecording       *chainTipRecording
	headerPreimageLock  sync.Mutex
	headerPreimages     *containers.LruCache[common.Hash, arbitrum.RecordedHeaderPreimage]
	servedTipRecordings atomic.Uint64
}

func NewChainTipBlockRecorder(execEngine *ExecutionEngine) *ChainTipBlockRecorder {
	recorder := &ChainTipBlockRecorder{
		execEngine:      execEngine,
		headerPreimages: containers.NewLruCache[common.Hash, arbitrum.RecordedHeaderPreimage](recentHeaderPreimageCacheSlots),
	}
	execEngine.SetTipRecorder(recorder)
	return recorder
}

func userWasmKeys(userWasms state.UserWasms) []wasmKey {
	keys := make([]wasmKey, 0, len(userWasms))
	for moduleHash, asmMap := range userWasms {
		for target := range asmMap {
			keys = append(keys, wasmKey{
				moduleHash: moduleHash,
				target:     target,
			})
		}
	}
	return keys
}
func (r *ChainTipBlockRecorder) RecordTip(block *types.Block, preimages map[common.Hash][]byte, firstHeaderNumber uint64, codeHashes []common.Hash, userWasms state.UserWasms) error {
	if block == nil {
		return nil
	}
	pos, err := r.execEngine.BlockNumberToMessageIndex(block.NumberU64())
	if err != nil {
		return fmt.Errorf("failed to calculate message index for chain-tip recording block %d: %w", block.NumberU64(), err)
	}
	record := &chainTipRecording{
		record: &execution.RecordResult{
			Pos:       pos,
			BlockHash: block.Hash(),
			Preimages: preimages,
		},
		blockNumber:       block.NumberU64(),
		parentHash:        block.ParentHash(),
		firstHeaderNumber: firstHeaderNumber,
		codeHashes:        codeHashes,
		wasmKeys:          userWasmKeys(userWasms),
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.lastRecording != nil && r.lastRecording.record != nil && pos < r.lastRecording.record.Pos {
		log.Warn("ignoring older chain-tip recording", "pos", pos, "lastPos", r.lastRecording.record.Pos)
		return nil
	}
	r.lastRecording = record
	return nil
}

func (r *ChainTipBlockRecorder) loadCodePreimages(record *execution.RecordResult, codeHashes []common.Hash) error {
	if len(codeHashes) == 0 {
		return nil
	}
	if record.Preimages == nil {
		record.Preimages = make(map[common.Hash][]byte, len(codeHashes))
	}
	disk := r.execEngine.bc.StateCache().TrieDB().Disk()
	for _, codeHash := range codeHashes {
		code := rawdb.ReadCode(disk, codeHash)
		if len(code) == 0 {
			return fmt.Errorf("chain-tip recording missing code preimage for hash %s", codeHash)
		}
		record.Preimages[codeHash] = code
	}
	return nil
}

func (r *ChainTipBlockRecorder) loadUserWasms(record *execution.RecordResult, keys []wasmKey, wasmTargets []rawdb.WasmTarget) error {
	if len(keys) == 0 || len(wasmTargets) == 0 {
		return nil
	}
	recordedTargets := make(map[common.Hash]map[rawdb.WasmTarget]struct{}, len(keys))
	for _, key := range keys {
		targets := recordedTargets[key.moduleHash]
		if targets == nil {
			targets = make(map[rawdb.WasmTarget]struct{})
			recordedTargets[key.moduleHash] = targets
		}
		targets[key.target] = struct{}{}
	}
	userWasms := make(state.UserWasms, len(recordedTargets))
	stateCache := r.execEngine.bc.StateCache()
	for moduleHash, recorded := range recordedTargets {
		asmMap := make(state.ActivatedWasm, len(wasmTargets))
		for _, target := range wasmTargets {
			if _, ok := recorded[target]; !ok {
				return fmt.Errorf("chain-tip recording for module %s missing requested target %s", moduleHash, target)
			}
			asm := stateCache.ActivatedAsm(target, moduleHash)
			if len(asm) == 0 {
				return fmt.Errorf("chain-tip recording missing user wasm for module %s target %s", moduleHash, target)
			}
			asmMap[target] = asm
		}
		userWasms[moduleHash] = asmMap
	}
	record.UserWasms = userWasms
	return nil
}

func (r *ChainTipBlockRecorder) Recording(pos arbutil.MessageIndex, wasmTargets []rawdb.WasmTarget) (*execution.RecordResult, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	recording := r.lastRecording
	if recording == nil || recording.record == nil || recording.record.Pos != pos {
		return nil, fmt.Errorf("chain-tip recording unavailable for pos %d", pos)
	}
	if err := r.validateRecording(recording); err != nil {
		return nil, err
	}
	if err := r.loadCodePreimages(recording.record, recording.codeHashes); err != nil {
		return nil, err
	}
	if err := r.loadRecentHeaderPreimages(recording.record, recording.blockNumber, recording.parentHash, recording.firstHeaderNumber); err != nil {
		return nil, err
	}
	if err := r.loadUserWasms(recording.record, recording.wasmKeys, wasmTargets); err != nil {
		return nil, err
	}
	r.servedTipRecordings.Add(1)
	return recording.record, nil
}

func (r *ChainTipBlockRecorder) validateRecording(recording *chainTipRecording) error {
	pos := recording.record.Pos
	expectedBlockNumber := r.execEngine.MessageIndexToBlockNumber(pos)
	if recording.blockNumber != expectedBlockNumber {
		return fmt.Errorf("chain-tip recording block number mismatch for pos %d: got %d expected %d", pos, recording.blockNumber, expectedBlockNumber)
	}
	canonicalHash := r.execEngine.bc.GetCanonicalHash(recording.blockNumber)
	if canonicalHash != recording.record.BlockHash {
		return fmt.Errorf("chain-tip recording stale for pos %d block %d: got hash %s canonical %s", pos, recording.blockNumber, recording.record.BlockHash, canonicalHash)
	}
	return nil
}

func (r *ChainTipBlockRecorder) ServedTipRecordings() uint64 {
	return r.servedTipRecordings.Load()
}

func (r *ChainTipBlockRecorder) loadRecentHeaderPreimages(record *execution.RecordResult, blockNumber uint64, parentHash common.Hash, firstHeaderNumber uint64) error {
	if blockNumber == 0 {
		return nil
	}
	if record.Preimages == nil {
		record.Preimages = make(map[common.Hash][]byte)
	}
	return arbitrum.AddRecordedHeaderPreimagesWithCache(record.Preimages, r.execEngine.bc, parentHash, blockNumber-1, firstHeaderNumber, r)
}

func (r *ChainTipBlockRecorder) GetRecordedHeaderPreimage(hash common.Hash) (arbitrum.RecordedHeaderPreimage, bool) {
	if r.headerPreimages == nil {
		return arbitrum.RecordedHeaderPreimage{}, false
	}
	r.headerPreimageLock.Lock()
	defer r.headerPreimageLock.Unlock()
	entry, ok := r.headerPreimages.Get(hash)
	return entry, ok
}

func (r *ChainTipBlockRecorder) AddRecordedHeaderPreimage(hash common.Hash, entry arbitrum.RecordedHeaderPreimage) {
	if r.headerPreimages == nil {
		return
	}
	r.headerPreimageLock.Lock()
	defer r.headerPreimageLock.Unlock()
	r.headerPreimages.Add(hash, entry)
}
