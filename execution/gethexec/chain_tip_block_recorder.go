// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package gethexec

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
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
	firstHeaderNumber uint64
	codeHashes        []common.Hash
	wasmKeys          []wasmKey
}

// ChainTipBlockRecorder serves recordings captured while the chain tip block is
// produced.
type ChainTipBlockRecorder struct {
	execEngine *ExecutionEngine

	headerPreimageLock  sync.Mutex
	headerPreimages     *containers.LruCache[common.Hash, arbitrum.RecordedHeaderPreimage]
	recordsFreezer      *blockRecordsFreezer
	servedTipRecordings atomic.Uint64
}

func NewChainTipBlockRecorder(execEngine *ExecutionEngine, recordsFreezer *blockRecordsFreezer) *ChainTipBlockRecorder {
	recorder := &ChainTipBlockRecorder{
		execEngine:      execEngine,
		headerPreimages: containers.NewLruCache[common.Hash, arbitrum.RecordedHeaderPreimage](recentHeaderPreimageCacheSlots),
		recordsFreezer:  recordsFreezer,
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
		firstHeaderNumber: firstHeaderNumber,
		codeHashes:        codeHashes,
		wasmKeys:          userWasmKeys(userWasms),
	}
	if err := r.recordsFreezer.writeRecording(record); err != nil {
		return fmt.Errorf("failed to persist chain-tip recording for pos %d: %w", pos, err)
	}
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
	recording, err := r.recordsFreezer.readRecording(pos)
	if err != nil {
		return nil, err
	}
	if recording == nil {
		return nil, fmt.Errorf("chain-tip recording unavailable for pos %d", pos)
	}
	blockNumber, parentHash, err := r.recordingBlockMetadata(recording)
	if err != nil {
		return nil, err
	}
	if err := r.loadCodePreimages(recording.record, recording.codeHashes); err != nil {
		return nil, err
	}
	if err := r.loadRecentHeaderPreimages(recording.record, blockNumber, parentHash, recording.firstHeaderNumber); err != nil {
		return nil, err
	}
	if err := r.loadUserWasms(recording.record, recording.wasmKeys, wasmTargets); err != nil {
		return nil, err
	}
	r.servedTipRecordings.Add(1)
	return recording.record, nil
}

func (r *ChainTipBlockRecorder) RecordBlockCreation(_ context.Context, pos arbutil.MessageIndex, _ *arbostypes.MessageWithMetadata, wasmTargets []rawdb.WasmTarget) (*execution.RecordResult, error) {
	return r.Recording(pos, wasmTargets)
}

func (r *ChainTipBlockRecorder) PrepareForRecord(context.Context, arbutil.MessageIndex, arbutil.MessageIndex) error {
	return nil
}

func (r *ChainTipBlockRecorder) MarkValid(arbutil.MessageIndex, common.Hash) {}

func (r *ChainTipBlockRecorder) OrderlyShutdown() {}

func (r *ChainTipBlockRecorder) recordingBlockMetadata(recording *chainTipRecording) (uint64, common.Hash, error) {
	pos := recording.record.Pos
	blockNumber := r.execEngine.MessageIndexToBlockNumber(pos)
	header := r.execEngine.bc.GetHeaderByNumber(blockNumber)
	if header == nil {
		return 0, common.Hash{}, fmt.Errorf("chain-tip recording block %d unavailable for pos %d", blockNumber, pos)
	}
	if header.Hash() != recording.record.BlockHash {
		return 0, common.Hash{}, fmt.Errorf("chain-tip recording stale for pos %d block %d: got hash %s canonical %s", pos, blockNumber, recording.record.BlockHash, header.Hash())
	}
	return blockNumber, header.ParentHash, nil
}

func (r *ChainTipBlockRecorder) ServedTipRecordings() uint64 {
	return r.servedTipRecordings.Load()
}

func (r *ChainTipBlockRecorder) PruneRecordingsBefore(pos arbutil.MessageIndex) error {
	if r.recordsFreezer == nil {
		return nil
	}
	return r.recordsFreezer.pruneRecordingsBefore(pos)
}

func (r *ChainTipBlockRecorder) Close() error {
	return r.recordsFreezer.Close()
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
