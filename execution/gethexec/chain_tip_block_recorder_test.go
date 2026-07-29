// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
)

func newTestRecorderEngine(t *testing.T, blocks int) *ExecutionEngine {
	t.Helper()
	gspec := &core.Genesis{Config: params.TestChainConfig}
	_, generated, _ := core.GenerateChainWithGenesis(gspec, ethash.NewFaker(), blocks, nil)
	bc, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), nil, gspec, ethash.NewFaker(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { bc.Stop() })
	if _, err := bc.InsertChain(generated); err != nil {
		t.Fatal(err)
	}
	return &ExecutionEngine{bc: bc}
}

func TestChainTipRecorderMissesNonTipPositions(t *testing.T) {
	engine := newTestRecorderEngine(t, 3)
	recorder := NewChainTipBlockRecorder(engine)

	if _, err := recorder.Recording(2, nil); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("expected unavailable error before any recording, got err=%v", err)
	}

	block := engine.bc.GetBlockByNumber(2)
	if err := recorder.RecordTip(block, nil, block.NumberU64(), nil, nil); err != nil {
		t.Fatal(err)
	}
	pos, err := engine.BlockNumberToMessageIndex(block.NumberU64())
	if err != nil {
		t.Fatal(err)
	}

	for _, missPos := range []arbutil.MessageIndex{pos - 1, pos + 1} {
		if _, err := recorder.Recording(missPos, nil); err == nil || !strings.Contains(err.Error(), "unavailable") {
			t.Fatalf("expected unavailable error for pos %d, got err=%v", missPos, err)
		}
	}
	if recorder.ServedTipRecordings() != 0 {
		t.Fatalf("expected served count 0 after misses, got %d", recorder.ServedTipRecordings())
	}

	record, err := recorder.Recording(pos, nil)
	if err != nil {
		t.Fatal(err)
	}
	if record.BlockHash != block.Hash() {
		t.Fatalf("unexpected served block hash: %s", record.BlockHash)
	}
}

func TestChainTipRecorderIgnoresOlderRecordings(t *testing.T) {
	engine := newTestRecorderEngine(t, 3)
	recorder := NewChainTipBlockRecorder(engine)

	record := func(block *types.Block) {
		t.Helper()
		if err := recorder.RecordTip(block, nil, block.NumberU64(), nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	posOf := func(number uint64) arbutil.MessageIndex {
		t.Helper()
		pos, err := engine.BlockNumberToMessageIndex(number)
		if err != nil {
			t.Fatal(err)
		}
		return pos
	}

	record(engine.bc.GetBlockByNumber(2))

	record(engine.bc.GetBlockByNumber(1))
	served, err := recorder.Recording(posOf(2), nil)
	if err != nil {
		t.Fatal(err)
	}
	if served.BlockHash != engine.bc.GetBlockByNumber(2).Hash() {
		t.Fatalf("expected retained tip to survive older write, got %s", served.BlockHash)
	}

	record(engine.bc.GetBlockByNumber(3))
	if _, err := recorder.Recording(posOf(2), nil); err == nil {
		t.Fatal("expected previous tip to be replaced by newer recording")
	}
	served, err = recorder.Recording(posOf(3), nil)
	if err != nil {
		t.Fatal(err)
	}
	if served.BlockHash != engine.bc.GetBlockByNumber(3).Hash() {
		t.Fatalf("expected newer recording to serve, got %s", served.BlockHash)
	}

	header := engine.bc.GetBlockByNumber(3).Header()
	header.Extra = []byte("same-height fork")
	record(types.NewBlock(header, &types.Body{}, nil, trie.NewStackTrie(nil)))
	if _, err := recorder.Recording(posOf(3), nil); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("expected same-position overwrite to replace the tip, got err=%v", err)
	}
}

func TestChainTipRecorderLoadFailuresAreErrors(t *testing.T) {
	engine := newTestRecorderEngine(t, 3)
	recorder := NewChainTipBlockRecorder(engine)

	block := engine.bc.GetBlockByNumber(2)
	pos, err := engine.BlockNumberToMessageIndex(block.NumberU64())
	if err != nil {
		t.Fatal(err)
	}
	fabricate := func(codeHashes []common.Hash, wasmKeys []wasmKey) {
		recorder.lastRecording = &chainTipRecording{
			record: &execution.RecordResult{
				Pos:       pos,
				BlockHash: block.Hash(),
			},
			blockNumber:       block.NumberU64(),
			parentHash:        block.ParentHash(),
			firstHeaderNumber: block.NumberU64(),
			codeHashes:        codeHashes,
			wasmKeys:          wasmKeys,
		}
	}

	fabricate(nil, []wasmKey{{moduleHash: testHash(1), target: rawdb.TargetWavm}})
	if _, err := recorder.Recording(pos, []rawdb.WasmTarget{rawdb.TargetArm64}); err == nil || !strings.Contains(err.Error(), "missing requested target") {
		t.Fatalf("expected missing requested target error, got err=%v", err)
	}

	fabricate(nil, []wasmKey{{moduleHash: testHash(1), target: rawdb.TargetWavm}})
	if _, err := recorder.Recording(pos, []rawdb.WasmTarget{rawdb.TargetWavm}); err == nil || !strings.Contains(err.Error(), "missing user wasm") {
		t.Fatalf("expected missing user wasm error, got err=%v", err)
	}

	fabricate([]common.Hash{testHash(2)}, nil)
	if _, err := recorder.Recording(pos, nil); err == nil || !strings.Contains(err.Error(), "missing code preimage") {
		t.Fatalf("expected missing code preimage error, got err=%v", err)
	}

	if recorder.ServedTipRecordings() != 0 {
		t.Fatalf("expected served count 0 after load failures, got %d", recorder.ServedTipRecordings())
	}
}

func TestChainTipRecorderRejectsStaleRecording(t *testing.T) {
	engine := newTestRecorderEngine(t, 3)
	recorder := NewChainTipBlockRecorder(engine)

	header := engine.bc.GetBlockByNumber(2).Header()
	header.Extra = []byte("orphaned fork")
	orphaned := types.NewBlock(header, &types.Body{}, nil, trie.NewStackTrie(nil))
	if err := recorder.RecordTip(orphaned, nil, header.Number.Uint64(), nil, nil); err != nil {
		t.Fatal(err)
	}
	pos, err := engine.BlockNumberToMessageIndex(header.Number.Uint64())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := recorder.Recording(pos, nil); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("expected stale recording to be rejected, got err=%v", err)
	}
	if recorder.ServedTipRecordings() != 0 {
		t.Fatalf("expected served count 0, got %d", recorder.ServedTipRecordings())
	}

	recorder.lastRecording = &chainTipRecording{
		record: &execution.RecordResult{
			Pos:       pos,
			BlockHash: engine.bc.GetBlockByNumber(2).Hash(),
		},
		blockNumber: 3,
	}
	if _, err := recorder.Recording(pos, nil); err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("expected block number mismatch to be rejected, got err=%v", err)
	}
	if recorder.ServedTipRecordings() != 0 {
		t.Fatalf("expected served count 0, got %d", recorder.ServedTipRecordings())
	}

	canonical := engine.bc.GetBlockByNumber(2)
	if err := recorder.RecordTip(canonical, nil, canonical.NumberU64(), nil, nil); err != nil {
		t.Fatal(err)
	}
	record, err := recorder.Recording(pos, nil)
	if err != nil {
		t.Fatal(err)
	}
	if record.BlockHash != canonical.Hash() {
		t.Fatalf("unexpected served block hash: %s", record.BlockHash)
	}
	if recorder.ServedTipRecordings() != 1 {
		t.Fatalf("expected served count 1, got %d", recorder.ServedTipRecordings())
	}
}

func testHash(b byte) common.Hash {
	return common.BytesToHash(bytes.Repeat([]byte{b}, common.HashLength))
}

func newTestChainTipRecorderWithHeaderCache(size int) *ChainTipBlockRecorder {
	return &ChainTipBlockRecorder{
		headerPreimages: containers.NewLruCache[common.Hash, arbitrum.RecordedHeaderPreimage](size),
	}
}

func TestRecentHeaderPreimageCacheStoresForksAtSameHeight(t *testing.T) {
	recorder := newTestChainTipRecorderWithHeaderCache(recentHeaderPreimageCacheSlots)
	headerA := testHash(1)
	headerB := testHash(2)
	parentA := testHash(3)
	parentB := testHash(4)

	recorder.AddRecordedHeaderPreimage(headerA, arbitrum.RecordedHeaderPreimage{
		ParentHash: parentA,
		Preimage:   []byte{1, 2, 3},
	})
	recorder.AddRecordedHeaderPreimage(headerB, arbitrum.RecordedHeaderPreimage{
		ParentHash: parentB,
		Preimage:   []byte{4, 5, 6},
	})

	entryA, ok := recorder.GetRecordedHeaderPreimage(headerA)
	if !ok {
		t.Fatal("expected cache hit for first fork")
	}
	if entryA.ParentHash != parentA || !bytes.Equal(entryA.Preimage, []byte{1, 2, 3}) {
		t.Fatalf("unexpected first fork entry: %+v", entryA)
	}

	entryB, ok := recorder.GetRecordedHeaderPreimage(headerB)
	if !ok {
		t.Fatal("expected cache hit for second fork")
	}
	if entryB.ParentHash != parentB || !bytes.Equal(entryB.Preimage, []byte{4, 5, 6}) {
		t.Fatalf("unexpected second fork entry: %+v", entryB)
	}
}

func TestRecentHeaderPreimageCacheEvictsLeastRecentlyUsed(t *testing.T) {
	recorder := newTestChainTipRecorderWithHeaderCache(2)
	oldHeader := testHash(1)
	middleHeader := testHash(2)
	newHeader := testHash(3)

	recorder.AddRecordedHeaderPreimage(oldHeader, arbitrum.RecordedHeaderPreimage{ParentHash: testHash(4), Preimage: []byte{1}})
	recorder.AddRecordedHeaderPreimage(middleHeader, arbitrum.RecordedHeaderPreimage{ParentHash: testHash(5), Preimage: []byte{2}})
	recorder.AddRecordedHeaderPreimage(newHeader, arbitrum.RecordedHeaderPreimage{ParentHash: testHash(6), Preimage: []byte{3}})

	if _, ok := recorder.GetRecordedHeaderPreimage(oldHeader); ok {
		t.Fatal("expected oldest entry to be evicted")
	}
	if _, ok := recorder.GetRecordedHeaderPreimage(middleHeader); !ok {
		t.Fatal("expected middle entry to remain cached")
	}
	if _, ok := recorder.GetRecordedHeaderPreimage(newHeader); !ok {
		t.Fatal("expected newest entry to be cached")
	}
}

func TestRecentHeaderPreimageCacheRetainsRecentEntries(t *testing.T) {
	recorder := newTestChainTipRecorderWithHeaderCache(recentHeaderPreimageCacheSlots)
	newestHeader := common.BytesToHash([]byte("newest header"))
	for blockNumber := uint64(0); blockNumber < recentHeaderPreimageCacheSlots; blockNumber++ {
		recorder.AddRecordedHeaderPreimage(testHash(byte(blockNumber)), arbitrum.RecordedHeaderPreimage{
			ParentHash: testHash(byte(blockNumber + 1)),
			Preimage:   []byte{byte(blockNumber)},
		})
	}

	recorder.AddRecordedHeaderPreimage(newestHeader, arbitrum.RecordedHeaderPreimage{
		ParentHash: testHash(254),
		Preimage:   []byte{255},
	})

	if _, ok := recorder.GetRecordedHeaderPreimage(testHash(0)); ok {
		t.Fatal("expected block 0 to be evicted")
	}
	for blockNumber := uint64(1); blockNumber < recentHeaderPreimageCacheSlots; blockNumber++ {
		if _, ok := recorder.GetRecordedHeaderPreimage(testHash(byte(blockNumber))); !ok {
			t.Fatalf("expected block %d to remain cached", blockNumber)
		}
	}
	if _, ok := recorder.GetRecordedHeaderPreimage(newestHeader); !ok {
		t.Fatal("expected newest block to be cached")
	}
}
