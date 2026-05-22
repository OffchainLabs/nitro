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
	"github.com/ethereum/go-ethereum/node"
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

func TestChainTipRecorderMissesUnrecordedPositions(t *testing.T) {
	engine := newTestRecorderEngine(t, 3)
	store := newBlockRecordsDatabase(rawdb.NewMemoryDatabase())
	recorder := NewChainTipBlockRecorder(engine, store)

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

func TestChainTipRecorderLoadFailuresAreErrors(t *testing.T) {
	engine := newTestRecorderEngine(t, 3)
	store := newBlockRecordsDatabase(rawdb.NewMemoryDatabase())
	recorder := NewChainTipBlockRecorder(engine, store)

	block := engine.bc.GetBlockByNumber(2)
	pos, err := engine.BlockNumberToMessageIndex(block.NumberU64())
	if err != nil {
		t.Fatal(err)
	}
	fabricate := func(codeHashes []common.Hash, wasmKeys []wasmKey) {
		t.Helper()
		if err := store.writeRecording(&chainTipRecording{
			record: &execution.RecordResult{
				Pos:       pos,
				BlockHash: block.Hash(),
			},
			blockNumber:       block.NumberU64(),
			parentHash:        block.ParentHash(),
			firstHeaderNumber: block.NumberU64(),
			codeHashes:        codeHashes,
			wasmKeys:          wasmKeys,
		}); err != nil {
			t.Fatal(err)
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
	store := newBlockRecordsDatabase(rawdb.NewMemoryDatabase())
	recorder := NewChainTipBlockRecorder(engine, store)

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

func TestBlockRecordsDatabaseRoundTrip(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	store := newBlockRecordsDatabase(db)
	pos := arbutil.MessageIndex(42)
	preimageHash := testHash(1)
	preimage := []byte{1, 2, 3}
	recording := &chainTipRecording{
		record: &execution.RecordResult{
			Pos:       pos,
			BlockHash: testHash(2),
			Preimages: map[common.Hash][]byte{
				preimageHash: preimage,
			},
		},
		blockNumber:       100,
		parentHash:        testHash(3),
		firstHeaderNumber: 99,
		codeHashes:        []common.Hash{testHash(4)},
		wasmKeys: []wasmKey{{
			moduleHash: testHash(5),
			target:     rawdb.TargetWavm,
		}},
	}

	if err := store.writeRecording(recording); err != nil {
		t.Fatal(err)
	}
	preimage[0] = 9

	loaded, ok, err := store.readRecording(pos)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected persisted recording")
	}
	if loaded.record.Pos != pos {
		t.Fatalf("unexpected pos: %d", loaded.record.Pos)
	}
	if loaded.record.BlockHash != recording.record.BlockHash {
		t.Fatalf("unexpected block hash: %s", loaded.record.BlockHash)
	}
	if !bytes.Equal(loaded.record.Preimages[preimageHash], []byte{1, 2, 3}) {
		t.Fatalf("unexpected preimage: %v", loaded.record.Preimages[preimageHash])
	}
	if loaded.blockNumber != 0 || loaded.parentHash != (common.Hash{}) || loaded.firstHeaderNumber != recording.firstHeaderNumber {
		t.Fatalf("unexpected persisted metadata: block %d parent %s first header %d", loaded.blockNumber, loaded.parentHash, loaded.firstHeaderNumber)
	}
	if len(loaded.codeHashes) != 1 || loaded.codeHashes[0] != recording.codeHashes[0] {
		t.Fatalf("unexpected code hashes: %v", loaded.codeHashes)
	}
	if len(loaded.wasmKeys) != 1 || loaded.wasmKeys[0] != recording.wasmKeys[0] {
		t.Fatalf("unexpected wasm keys: %v", loaded.wasmKeys)
	}

	loaded.record.Preimages[preimageHash][0] = 8
	loaded, ok, err = store.readRecording(pos)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected persisted recording after caller mutation")
	}
	if !bytes.Equal(loaded.record.Preimages[preimageHash], []byte{1, 2, 3}) {
		t.Fatalf("persisted preimage was mutable, got %v", loaded.record.Preimages[preimageHash])
	}
}

func TestBlockRecordsDatabasePersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	db, err := node.NewPebbleDBDatabase(dir, 0, 0, "", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	store := newBlockRecordsDatabase(db)
	pos := arbutil.MessageIndex(42)
	preimageHash := testHash(1)
	if err := store.writeRecording(&chainTipRecording{
		record: &execution.RecordResult{
			Pos:       pos,
			BlockHash: testHash(2),
			Preimages: map[common.Hash][]byte{
				preimageHash: {1, 2, 3},
			},
		},
		blockNumber:       100,
		parentHash:        testHash(3),
		firstHeaderNumber: 99,
		codeHashes:        []common.Hash{testHash(4)},
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = node.NewPebbleDBDatabase(dir, 0, 0, "", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store = newBlockRecordsDatabase(db)
	loaded, ok, err := store.readRecording(pos)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected persisted recording after reopening database")
	}
	if loaded.record.Pos != pos || loaded.record.BlockHash != testHash(2) {
		t.Fatalf("unexpected loaded record: %+v", loaded.record)
	}
	if !bytes.Equal(loaded.record.Preimages[preimageHash], []byte{1, 2, 3}) {
		t.Fatalf("unexpected preimage after reopening database: %v", loaded.record.Preimages[preimageHash])
	}
	if loaded.blockNumber != 0 || loaded.parentHash != (common.Hash{}) || loaded.firstHeaderNumber != 99 {
		t.Fatalf("unexpected persisted metadata after reopening database: block %d parent %s first header %d", loaded.blockNumber, loaded.parentHash, loaded.firstHeaderNumber)
	}
	if len(loaded.codeHashes) != 1 || loaded.codeHashes[0] != testHash(4) {
		t.Fatalf("unexpected code hashes after reopening database: %v", loaded.codeHashes)
	}
}

func TestBlockRecordsDatabaseReadMissing(t *testing.T) {
	store := newBlockRecordsDatabase(rawdb.NewMemoryDatabase())
	loaded, ok, err := store.readRecording(arbutil.MessageIndex(42))
	if err != nil {
		t.Fatal(err)
	}
	if ok || loaded != nil {
		t.Fatalf("expected missing recording, got ok=%t loaded=%+v", ok, loaded)
	}
}

func TestChainTipRecorderReadsPersistedRecording(t *testing.T) {
	store := newBlockRecordsDatabase(rawdb.NewMemoryDatabase())
	pos := arbutil.MessageIndex(42)
	preimageHash := testHash(1)
	if err := store.writeRecording(&chainTipRecording{
		record: &execution.RecordResult{
			Pos:       pos,
			BlockHash: testHash(2),
			Preimages: map[common.Hash][]byte{
				preimageHash: {1, 2, 3},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	recorder := &ChainTipBlockRecorder{
		recordsDatabase: store,
	}

	record, err := recorder.Recording(pos, nil)
	if err != nil {
		t.Fatal(err)
	}
	if record.Pos != pos {
		t.Fatalf("unexpected pos: %d", record.Pos)
	}
	if !bytes.Equal(record.Preimages[preimageHash], []byte{1, 2, 3}) {
		t.Fatalf("unexpected preimage: %v", record.Preimages[preimageHash])
	}
	if recorder.ServedTipRecordings() != 1 {
		t.Fatalf("expected served count 1, got %d", recorder.ServedTipRecordings())
	}
}

func TestChainTipRecorderReadsMultiplePersistedRecordings(t *testing.T) {
	store := newBlockRecordsDatabase(rawdb.NewMemoryDatabase())
	preimageHash := testHash(1)
	positions := []arbutil.MessageIndex{40, 41, 42}
	for _, pos := range positions {
		if err := store.writeRecording(&chainTipRecording{
			record: &execution.RecordResult{
				Pos:       pos,
				BlockHash: testHash(byte(pos)),
				Preimages: map[common.Hash][]byte{
					preimageHash: {byte(pos)},
				},
			},
		}); err != nil {
			t.Fatal(err)
		}
	}
	recorder := &ChainTipBlockRecorder{
		recordsDatabase: store,
	}

	for _, pos := range positions {
		record, err := recorder.Recording(pos, nil)
		if err != nil {
			t.Fatal(err)
		}
		if record.Pos != pos {
			t.Fatalf("unexpected pos: %d", record.Pos)
		}
		if record.BlockHash != testHash(byte(pos)) {
			t.Fatalf("unexpected block hash: %s", record.BlockHash)
		}
		if !bytes.Equal(record.Preimages[preimageHash], []byte{byte(pos)}) {
			t.Fatalf("unexpected preimage for pos %d: %v", pos, record.Preimages[preimageHash])
		}
	}
	if recorder.ServedTipRecordings() != uint64(len(positions)) {
		t.Fatalf("expected served count %d, got %d", len(positions), recorder.ServedTipRecordings())
	}
}
