// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package gethexec

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
)

const recentHeaderPreimageWindow = 256

type wasmKey struct {
	moduleHash common.Hash
	target     rawdb.WasmTarget
}

type chainTipRecording struct {
	record      *execution.RecordResult
	blockNumber uint64
	parentHash  common.Hash
	codeHashes  []common.Hash
	wasmKeys    []wasmKey
}

// ChainTipBlockRecorder serves recordings captured while the chain tip block is
// produced.
type ChainTipBlockRecorder struct {
	execEngine *ExecutionEngine

	lock                sync.Mutex
	lastRecording       *chainTipRecording
	headerPreimages     recentHeaderPreimageCache
	servedTipRecordings atomic.Uint64
}

func NewChainTipBlockRecorder(execEngine *ExecutionEngine) *ChainTipBlockRecorder {
	recorder := &ChainTipBlockRecorder{
		execEngine: execEngine,
	}
	execEngine.SetTipRecorder(recorder)
	return recorder
}

func copyPreimageMap(preimages map[common.Hash][]byte) map[common.Hash][]byte {
	copied := make(map[common.Hash][]byte, len(preimages))
	for hash, preimage := range preimages {
		copied[hash] = common.CopyBytes(preimage)
	}
	return copied
}

func copyCodeHashes(codeHashes []common.Hash) []common.Hash {
	return append([]common.Hash(nil), codeHashes...)
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

func copyRecordResult(record *execution.RecordResult) *execution.RecordResult {
	if record == nil {
		return nil
	}
	return &execution.RecordResult{
		Pos:       record.Pos,
		BlockHash: record.BlockHash,
		Preimages: copyPreimageMap(record.Preimages),
		UserWasms: nil,
	}
}

func copyChainTipRecording(recording *chainTipRecording) *chainTipRecording {
	if recording == nil {
		return nil
	}
	return &chainTipRecording{
		record:      copyRecordResult(recording.record),
		blockNumber: recording.blockNumber,
		parentHash:  recording.parentHash,
		codeHashes:  copyCodeHashes(recording.codeHashes),
		wasmKeys:    append([]wasmKey(nil), recording.wasmKeys...),
	}
}

func (r *ChainTipBlockRecorder) RecordTip(block *types.Block, preimages map[common.Hash][]byte, codeHashes []common.Hash, userWasms state.UserWasms) error {
	if block == nil {
		return nil
	}
	pos, err := r.execEngine.BlockNumberToMessageIndex(block.NumberU64())
	if err != nil {
		return fmt.Errorf("failed to calculate message index for chain-tip recording block %d: %w", block.NumberU64(), err)
	}
	preimages = copyPreimageMap(preimages)

	record := &chainTipRecording{
		record: &execution.RecordResult{
			Pos:       pos,
			BlockHash: block.Hash(),
			Preimages: preimages,
		},
		blockNumber: block.NumberU64(),
		parentHash:  block.ParentHash(),
		codeHashes:  copyCodeHashes(codeHashes),
		wasmKeys:    userWasmKeys(userWasms),
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
		if _, exists := record.Preimages[codeHash]; exists {
			continue
		}
		code := rawdb.ReadCode(disk, codeHash)
		if len(code) == 0 {
			return fmt.Errorf("chain-tip recording missing code preimage for hash %s", codeHash)
		}
		record.Preimages[codeHash] = common.CopyBytes(code)
	}
	return nil
}

func (r *ChainTipBlockRecorder) loadUserWasms(record *execution.RecordResult, keys []wasmKey) error {
	if len(keys) == 0 {
		return nil
	}
	if record.UserWasms == nil {
		record.UserWasms = make(state.UserWasms)
	}
	stateCache := r.execEngine.bc.StateCache()
	for _, key := range keys {
		asmMap := record.UserWasms[key.moduleHash]
		if asmMap == nil {
			asmMap = make(state.ActivatedWasm)
			record.UserWasms[key.moduleHash] = asmMap
		}
		if _, exists := asmMap[key.target]; exists {
			continue
		}
		asm := stateCache.ActivatedAsm(key.target, key.moduleHash)
		if len(asm) == 0 {
			return fmt.Errorf("chain-tip recording missing user wasm for module %s target %s", key.moduleHash, key.target)
		}
		asmMap[key.target] = common.CopyBytes(asm)
	}
	return nil
}

func (r *ChainTipBlockRecorder) Recording(pos arbutil.MessageIndex) (*execution.RecordResult, error) {
	r.lock.Lock()
	recording := copyChainTipRecording(r.lastRecording)
	r.lock.Unlock()
	if recording == nil || recording.record == nil || recording.record.Pos != pos {
		return nil, fmt.Errorf("chain-tip recording unavailable for pos %d", pos)
	}
	if err := r.validateRecording(pos, recording); err != nil {
		return nil, err
	}
	if err := r.loadCodePreimages(recording.record, recording.codeHashes); err != nil {
		return nil, err
	}
	if err := r.loadRecentHeaderPreimages(recording.record, recording.blockNumber, recording.parentHash); err != nil {
		return nil, err
	}
	if err := r.loadUserWasms(recording.record, recording.wasmKeys); err != nil {
		return nil, err
	}
	r.servedTipRecordings.Add(1)
	return recording.record, nil
}

func (r *ChainTipBlockRecorder) validateRecording(pos arbutil.MessageIndex, recording *chainTipRecording) error {
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

func firstRecentHeaderNum(blockNumber uint64) uint64 {
	if blockNumber <= recentHeaderPreimageWindow {
		return 0
	}
	return blockNumber - recentHeaderPreimageWindow
}

func (r *ChainTipBlockRecorder) loadRecentHeaderPreimages(record *execution.RecordResult, blockNumber uint64, parentHash common.Hash) error {
	if blockNumber == 0 {
		return nil
	}
	if record.Preimages == nil {
		record.Preimages = make(map[common.Hash][]byte)
	}
	headerHash := parentHash
	firstHeaderNumber := firstRecentHeaderNum(blockNumber)
	for headerNum := blockNumber; headerNum > firstHeaderNumber; {
		headerNum--
		entry, ok := r.headerPreimages.get(headerNum, headerHash)
		if !ok {
			header := r.execEngine.bc.GetHeader(headerHash, headerNum)
			if header == nil {
				return fmt.Errorf("failed to load recent header %d hash %s for chain-tip recording", headerNum, headerHash)
			}
			var err error
			entry.preimage, err = rlp.EncodeToBytes(header)
			if err != nil {
				return fmt.Errorf("failed to encode recent header %d for chain-tip recording: %w", headerNum, err)
			}
			entry.parentHash = header.ParentHash
			r.headerPreimages.add(headerNum, headerHash, entry)
		}
		if _, exists := record.Preimages[headerHash]; exists {
			headerHash = entry.parentHash
			continue
		}
		record.Preimages[headerHash] = common.CopyBytes(entry.preimage)
		headerHash = entry.parentHash
	}
	return nil
}

type recentHeaderPreimage struct {
	parentHash common.Hash
	preimage   []byte
}

type recentHeaderPreimageSlot struct {
	blockNumber uint64
	occupied    bool
	entries     map[common.Hash]recentHeaderPreimage
}

type recentHeaderPreimageCache struct {
	lock  sync.Mutex
	slots [recentHeaderPreimageWindow]recentHeaderPreimageSlot
}

func (c *recentHeaderPreimageCache) get(blockNumber uint64, hash common.Hash) (recentHeaderPreimage, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	slot := c.slots[blockNumber%recentHeaderPreimageWindow]
	if !slot.occupied || slot.blockNumber != blockNumber {
		return recentHeaderPreimage{}, false
	}
	entry, ok := slot.entries[hash]
	entry.preimage = common.CopyBytes(entry.preimage)
	return entry, ok
}

func (c *recentHeaderPreimageCache) add(blockNumber uint64, hash common.Hash, entry recentHeaderPreimage) {
	c.lock.Lock()
	defer c.lock.Unlock()
	slot := &c.slots[blockNumber%recentHeaderPreimageWindow]
	if !slot.occupied || slot.blockNumber != blockNumber {
		slot.blockNumber = blockNumber
		slot.occupied = true
		slot.entries = make(map[common.Hash]recentHeaderPreimage)
	}
	if _, exists := slot.entries[hash]; exists {
		return
	}
	entry.preimage = common.CopyBytes(entry.preimage)
	slot.entries[hash] = entry
}
