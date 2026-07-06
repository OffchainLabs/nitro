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
	"github.com/ethereum/go-ethereum/ethdb"
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
	store := newBlockRecordsDatabase(newTestBlockRecordsFreezer(t, ""))
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
	store := newBlockRecordsDatabase(newTestBlockRecordsFreezer(t, ""))
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
	store := newBlockRecordsDatabase(newTestBlockRecordsFreezer(t, ""))
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

func newTestBlockRecordsFreezer(t *testing.T, ancientDir string) ethdb.ResettableAncientStore {
	t.Helper()
	freezer, err := rawdb.NewChainTipBlockRecordsFreezer(ancientDir, false)
	if err != nil {
		t.Fatal(err)
	}
	return freezer
}

func testChainTipRecording(pos arbutil.MessageIndex) *chainTipRecording {
	return &chainTipRecording{
		record: &execution.RecordResult{
			Pos:       pos,
			BlockHash: testHash(byte(pos + 1)),
			Preimages: map[common.Hash][]byte{
				testHash(1): {byte(pos)},
			},
		},
		blockNumber:       uint64(pos + 100),
		parentHash:        testHash(byte(pos + 2)),
		firstHeaderNumber: uint64(pos + 99),
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
	store := newBlockRecordsDatabase(newTestBlockRecordsFreezer(t, ""))
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

func TestBlockRecordsDatabaseWritesStraightToFreezer(t *testing.T) {
	freezer := newTestBlockRecordsFreezer(t, "")
	store := newBlockRecordsDatabase(freezer)
	firstPos := arbutil.MessageIndex(1)
	latestPos := arbutil.MessageIndex(10)
	for pos := firstPos; pos <= latestPos; pos++ {
		if err := store.writeRecording(testChainTipRecording(pos)); err != nil {
			t.Fatal(err)
		}
	}

	frozen, err := freezer.Ancients()
	if err != nil {
		t.Fatal(err)
	}
	if frozen != uint64(latestPos-firstPos+1) {
		t.Fatalf("expected %d frozen recordings, got %d", latestPos-firstPos+1, frozen)
	}
	for pos := firstPos; pos <= latestPos; pos++ {
		loaded, ok, err := store.readRecording(pos)
		if err != nil {
			t.Fatal(err)
		}
		if !ok || loaded.record.Pos != pos || loaded.record.BlockHash != testChainTipRecording(pos).record.BlockHash {
			t.Fatalf("unexpected frozen recording for pos %d: ok=%t loaded=%+v", pos, ok, loaded)
		}
	}
}

func TestBlockRecordsDatabasePersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	store := newBlockRecordsDatabase(newTestBlockRecordsFreezer(t, dir))
	firstPos := arbutil.MessageIndex(1000)
	latestPos := arbutil.MessageIndex(1009)
	for pos := firstPos; pos <= latestPos; pos++ {
		if err := store.writeRecording(testChainTipRecording(pos)); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	store = newBlockRecordsDatabase(newTestBlockRecordsFreezer(t, dir))
	defer store.Close()
	for _, pos := range []arbutil.MessageIndex{firstPos, latestPos} {
		loaded, ok, err := store.readRecording(pos)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatalf("expected persisted recording for pos %d after reopening database", pos)
		}
		if loaded.record.Pos != pos || loaded.record.BlockHash != testChainTipRecording(pos).record.BlockHash {
			t.Fatalf("unexpected recording after reopening database: %+v", loaded.record)
		}
	}
	if err := store.writeRecording(testChainTipRecording(latestPos + 1)); err != nil {
		t.Fatal(err)
	}
	loaded, ok, err := store.readRecording(latestPos + 1)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || loaded.record.Pos != latestPos+1 {
		t.Fatalf("expected recording appended after reopen, got ok=%t loaded=%+v", ok, loaded)
	}
}

func TestBlockRecordsDatabaseReadMissing(t *testing.T) {
	store := newBlockRecordsDatabase(newTestBlockRecordsFreezer(t, ""))
	loaded, ok, err := store.readRecording(arbutil.MessageIndex(42))
	if err != nil {
		t.Fatal(err)
	}
	if ok || loaded != nil {
		t.Fatalf("expected missing recording, got ok=%t loaded=%+v", ok, loaded)
	}

	if err := store.writeRecording(testChainTipRecording(10)); err != nil {
		t.Fatal(err)
	}
	for _, pos := range []arbutil.MessageIndex{9, 11} {
		loaded, ok, err = store.readRecording(pos)
		if err != nil {
			t.Fatal(err)
		}
		if ok || loaded != nil {
			t.Fatalf("expected missing recording for pos %d, got ok=%t loaded=%+v", pos, ok, loaded)
		}
	}
}

func TestBlockRecordsDatabaseReorgTruncatesHead(t *testing.T) {
	freezer := newTestBlockRecordsFreezer(t, "")
	store := newBlockRecordsDatabase(freezer)
	firstPos := arbutil.MessageIndex(1)
	latestPos := arbutil.MessageIndex(10)
	for pos := firstPos; pos <= latestPos; pos++ {
		if err := store.writeRecording(testChainTipRecording(pos)); err != nil {
			t.Fatal(err)
		}
	}

	reorgPos := arbutil.MessageIndex(6)
	reorged := testChainTipRecording(reorgPos)
	reorged.record.BlockHash = testHash(250)
	if err := store.writeRecording(reorged); err != nil {
		t.Fatal(err)
	}

	frozen, err := freezer.Ancients()
	if err != nil {
		t.Fatal(err)
	}
	if frozen != uint64(reorgPos-firstPos+1) {
		t.Fatalf("expected freezer head at %d after reorg, got %d", reorgPos-firstPos+1, frozen)
	}
	loaded, ok, err := store.readRecording(reorgPos)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || loaded.record.BlockHash != reorged.record.BlockHash {
		t.Fatalf("expected reorged recording, got ok=%t loaded=%+v", ok, loaded)
	}
	loaded, ok, err = store.readRecording(reorgPos - 1)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || loaded.record.BlockHash != testChainTipRecording(reorgPos-1).record.BlockHash {
		t.Fatalf("expected recording below reorg point to survive, got ok=%t loaded=%+v", ok, loaded)
	}
	loaded, ok, err = store.readRecording(reorgPos + 1)
	if err != nil {
		t.Fatal(err)
	}
	if ok || loaded != nil {
		t.Fatalf("expected recording above reorg point to be truncated, got ok=%t loaded=%+v", ok, loaded)
	}

	if err := store.writeRecording(testChainTipRecording(reorgPos + 1)); err != nil {
		t.Fatal(err)
	}
	loaded, ok, err = store.readRecording(reorgPos + 1)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || loaded.record.Pos != reorgPos+1 {
		t.Fatalf("expected recording appended after reorg, got ok=%t loaded=%+v", ok, loaded)
	}
}

type syncCountingFreezer struct {
	ethdb.ResettableAncientStore
	syncs int
}

func (f *syncCountingFreezer) SyncAncient() error {
	f.syncs++
	return f.ResettableAncientStore.SyncAncient()
}

func TestBlockRecordsDatabaseSyncsEveryWrite(t *testing.T) {
	freezer := &syncCountingFreezer{ResettableAncientStore: newTestBlockRecordsFreezer(t, "")}
	store := newBlockRecordsDatabase(freezer)
	for pos := arbutil.MessageIndex(1); pos <= 5; pos++ {
		if err := store.writeRecording(testChainTipRecording(pos)); err != nil {
			t.Fatal(err)
		}
	}
	if freezer.syncs != 5 {
		t.Fatalf("expected 5 freezer syncs, got %d", freezer.syncs)
	}
}

func TestBlockRecordsDatabaseResetsForUnrepresentablePositions(t *testing.T) {
	// Positions the stored recordings cannot precede contiguously drop the
	// stored history and restart recording from the new position.
	testCases := []struct {
		name     string
		prune    uint64 // freezer tail truncation before the write, if non-zero
		writePos arbutil.MessageIndex
	}{
		// The recorder was disabled for a while, leaving a gap that can never
		// be backfilled.
		{name: "AheadOfHead", writePos: 2000},
		// A reorg below the first stored recording makes everything stored
		// stale.
		{name: "BelowFirstRecording", writePos: 500},
		// A reorg below the pruned tail is not representable in the freezer
		// at all and must not halt recording.
		{name: "BelowPrunedTail", prune: 5, writePos: 3},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freezer := newTestBlockRecordsFreezer(t, "")
			store := newBlockRecordsDatabase(freezer)
			for pos := arbutil.MessageIndex(1000); pos <= 1009; pos++ {
				if err := store.writeRecording(testChainTipRecording(pos)); err != nil {
					t.Fatal(err)
				}
			}
			if testCase.prune > 0 {
				if _, err := freezer.TruncateTail(testCase.prune); err != nil {
					t.Fatal(err)
				}
			}

			if err := store.writeRecording(testChainTipRecording(testCase.writePos)); err != nil {
				t.Fatal(err)
			}

			frozen, err := freezer.Ancients()
			if err != nil {
				t.Fatal(err)
			}
			if frozen != 1 {
				t.Fatalf("expected 1 recording after freezer reset, got %d", frozen)
			}
			tail, err := freezer.Tail()
			if err != nil {
				t.Fatal(err)
			}
			if tail != 0 {
				t.Fatalf("expected freezer tail to reset to 0, got %d", tail)
			}
			loaded, ok, err := store.readRecording(1005)
			if err != nil {
				t.Fatal(err)
			}
			if ok || loaded != nil {
				t.Fatalf("expected stale recording to be dropped, got ok=%t loaded=%+v", ok, loaded)
			}
			loaded, ok, err = store.readRecording(testCase.writePos)
			if err != nil {
				t.Fatal(err)
			}
			if !ok || loaded.record.Pos != testCase.writePos {
				t.Fatalf("expected recording after reset, got ok=%t loaded=%+v", ok, loaded)
			}
			// Recording continues contiguously from the new position.
			if err := store.writeRecording(testChainTipRecording(testCase.writePos + 1)); err != nil {
				t.Fatal(err)
			}
			loaded, ok, err = store.readRecording(testCase.writePos + 1)
			if err != nil {
				t.Fatal(err)
			}
			if !ok || loaded.record.Pos != testCase.writePos+1 {
				t.Fatalf("expected recording appended after reset, got ok=%t loaded=%+v", ok, loaded)
			}
		})
	}
}

func TestBlockRecordsDatabaseRecoversFromFullyCorruptContents(t *testing.T) {
	// Corruption surviving the freezer's own repair (e.g. torn writes after a
	// power loss) must not permanently halt recording: with no decodable
	// recording left, the store drops the junk and restarts.
	freezer := newTestBlockRecordsFreezer(t, "")
	if _, err := freezer.ModifyAncients(func(writer ethdb.AncientWriteOp) error {
		return writer.AppendRaw(rawdb.ChainTipBlockRecordsFreezerTable, 0, []byte{0xde, 0xad, 0xbe, 0xef})
	}); err != nil {
		t.Fatal(err)
	}
	store := newBlockRecordsDatabase(freezer)

	loaded, ok, err := store.readRecording(0)
	if err != nil {
		t.Fatal(err)
	}
	if ok || loaded != nil {
		t.Fatalf("expected corrupt recording to read as missing, got ok=%t loaded=%+v", ok, loaded)
	}
	pos := arbutil.MessageIndex(42)
	if err := store.writeRecording(testChainTipRecording(pos)); err != nil {
		t.Fatal(err)
	}
	frozen, err := freezer.Ancients()
	if err != nil {
		t.Fatal(err)
	}
	if frozen != 1 {
		t.Fatalf("expected corrupt contents to be dropped, got %d freezer items", frozen)
	}
	loaded, ok, err = store.readRecording(pos)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || loaded.record.Pos != pos {
		t.Fatalf("expected recording after corruption recovery, got ok=%t loaded=%+v", ok, loaded)
	}
}

func TestBlockRecordsDatabaseCorruptRecordingReadsAsMissing(t *testing.T) {
	freezer := newTestBlockRecordsFreezer(t, "")
	store := newBlockRecordsDatabase(freezer)
	if err := store.writeRecording(testChainTipRecording(1)); err != nil {
		t.Fatal(err)
	}
	if err := store.writeRecording(testChainTipRecording(2)); err != nil {
		t.Fatal(err)
	}
	// Corrupt item for position 3 behind the decodable recordings.
	if _, err := freezer.ModifyAncients(func(writer ethdb.AncientWriteOp) error {
		return writer.AppendRaw(rawdb.ChainTipBlockRecordsFreezerTable, 2, []byte{0xde, 0xad, 0xbe, 0xef})
	}); err != nil {
		t.Fatal(err)
	}
	store = newBlockRecordsDatabase(freezer)

	loaded, ok, err := store.readRecording(3)
	if err != nil {
		t.Fatal(err)
	}
	if ok || loaded != nil {
		t.Fatalf("expected corrupt recording to read as missing, got ok=%t loaded=%+v", ok, loaded)
	}
	if err := store.writeRecording(testChainTipRecording(4)); err != nil {
		t.Fatal(err)
	}
	for _, pos := range []arbutil.MessageIndex{1, 2, 4} {
		loaded, ok, err = store.readRecording(pos)
		if err != nil {
			t.Fatal(err)
		}
		if !ok || loaded.record.Pos != pos {
			t.Fatalf("expected recording for pos %d to survive corruption, got ok=%t loaded=%+v", pos, ok, loaded)
		}
	}
}

func TestBlockRecordsDatabaseTailPruning(t *testing.T) {
	dir := t.TempDir()
	freezer := newTestBlockRecordsFreezer(t, dir)
	store := newBlockRecordsDatabase(freezer)
	firstPos := arbutil.MessageIndex(1000)
	latestPos := arbutil.MessageIndex(1009)
	for pos := firstPos; pos <= latestPos; pos++ {
		if err := store.writeRecording(testChainTipRecording(pos)); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := freezer.TruncateTail(3); err != nil {
		t.Fatal(err)
	}
	loaded, ok, err := store.readRecording(firstPos + 2)
	if err != nil {
		t.Fatal(err)
	}
	if ok || loaded != nil {
		t.Fatalf("expected pruned recording to be missing, got ok=%t loaded=%+v", ok, loaded)
	}
	loaded, ok, err = store.readRecording(firstPos + 3)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || loaded.record.Pos != firstPos+3 {
		t.Fatalf("expected earliest retained recording, got ok=%t loaded=%+v", ok, loaded)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	// The message index of freezer item 0 must still be recoverable from the
	// earliest retained recording after a restart.
	store = newBlockRecordsDatabase(newTestBlockRecordsFreezer(t, dir))
	defer store.Close()
	loaded, ok, err = store.readRecording(firstPos + 3)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || loaded.record.Pos != firstPos+3 {
		t.Fatalf("expected earliest retained recording after reopen, got ok=%t loaded=%+v", ok, loaded)
	}
	loaded, ok, err = store.readRecording(firstPos + 2)
	if err != nil {
		t.Fatal(err)
	}
	if ok || loaded != nil {
		t.Fatalf("expected pruned recording to stay missing after reopen, got ok=%t loaded=%+v", ok, loaded)
	}
	if err := store.writeRecording(testChainTipRecording(latestPos + 1)); err != nil {
		t.Fatal(err)
	}
	loaded, ok, err = store.readRecording(latestPos + 1)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || loaded.record.Pos != latestPos+1 {
		t.Fatalf("expected recording appended after pruning, got ok=%t loaded=%+v", ok, loaded)
	}
}

func TestChainTipRecorderReadsPersistedRecording(t *testing.T) {
	store := newBlockRecordsDatabase(newTestBlockRecordsFreezer(t, ""))
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
	store := newBlockRecordsDatabase(newTestBlockRecordsFreezer(t, ""))
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
