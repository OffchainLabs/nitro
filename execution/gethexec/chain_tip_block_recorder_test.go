// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
)

func TestChainTipRecorderMissesUnrecordedPositions(t *testing.T) {
	engine := newTestRecorderEngine(t, 3)
	store := newBlockRecordsFreezer(newTestBlockRecordsFreezer(t, ""))
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

	if _, err := recorder.Recording(pos-1, nil); err == nil || !strings.Contains(err.Error(), "below the freezer tail") {
		t.Fatalf("expected a below tail error for pos %d, got err=%v", pos-1, err)
	}
	if _, err := recorder.Recording(pos+1, nil); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("expected unavailable error for pos %d, got err=%v", pos+1, err)
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
	store := newBlockRecordsFreezer(newTestBlockRecordsFreezer(t, ""))
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
	store := newBlockRecordsFreezer(newTestBlockRecordsFreezer(t, ""))
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

func forEachFreezerBackend(t *testing.T, run func(t *testing.T, newFreezer func(t *testing.T) ethdb.ResettableAncientStore)) {
	t.Run("Memory", func(t *testing.T) {
		run(t, func(t *testing.T) ethdb.ResettableAncientStore { return newTestBlockRecordsFreezer(t, "") })
	})
	t.Run("Disk", func(t *testing.T) {
		run(t, func(t *testing.T) ethdb.ResettableAncientStore { return newTestBlockRecordsFreezer(t, t.TempDir()) })
	})
}

func newTestBlockRecordsFreezer(t *testing.T, ancientDir string) ethdb.ResettableAncientStore {
	t.Helper()
	freezer, err := rawdb.NewChainTipBlockRecordsFreezer(ancientDir, false)
	if err != nil {
		t.Fatal(err)
	}
	return freezer
}

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

func testChainTipRecording(pos arbutil.MessageIndex) *chainTipRecording {
	return &chainTipRecording{
		record: &execution.RecordResult{
			Pos: pos,
			// #nosec G115
			BlockHash: testHash(byte(pos + 1)),
			Preimages: map[common.Hash][]byte{
				// #nosec G115
				testHash(1): {byte(pos)},
			},
		},
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

func TestBlockRecordsFreezerRoundTrip(t *testing.T) {
	store := newBlockRecordsFreezer(newTestBlockRecordsFreezer(t, ""))
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

	loaded, err := store.readRecording(pos)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil {
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
	if loaded.firstHeaderNumber != recording.firstHeaderNumber {
		t.Fatalf("unexpected persisted first header number: %d", loaded.firstHeaderNumber)
	}
	if len(loaded.codeHashes) != 1 || loaded.codeHashes[0] != recording.codeHashes[0] {
		t.Fatalf("unexpected code hashes: %v", loaded.codeHashes)
	}
	if len(loaded.wasmKeys) != 1 || loaded.wasmKeys[0] != recording.wasmKeys[0] {
		t.Fatalf("unexpected wasm keys: %v", loaded.wasmKeys)
	}

	loaded.record.Preimages[preimageHash][0] = 8
	loaded, err = store.readRecording(pos)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil {
		t.Fatal("expected persisted recording after caller mutation")
	}
	if !bytes.Equal(loaded.record.Preimages[preimageHash], []byte{1, 2, 3}) {
		t.Fatalf("persisted preimage was mutable, got %v", loaded.record.Preimages[preimageHash])
	}
}

func TestBlockRecordsFreezerWritesStraightToFreezer(t *testing.T) {
	freezer := newTestBlockRecordsFreezer(t, "")
	store := newBlockRecordsFreezer(freezer)
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
	if frozen != uint64(latestPos)+1 {
		t.Fatalf("expected freezer head at %d, got %d", uint64(latestPos)+1, frozen)
	}
	tail, err := freezer.Tail(rawdb.ChainTipBlockRecordsGroup)
	if err != nil {
		t.Fatal(err)
	}
	if tail != uint64(firstPos) {
		t.Fatalf("expected freezer tail at %d, got %d", firstPos, tail)
	}
	for pos := firstPos; pos <= latestPos; pos++ {
		loaded, err := store.readRecording(pos)
		if err != nil {
			t.Fatal(err)
		}
		if loaded == nil || loaded.record.Pos != pos || loaded.record.BlockHash != testChainTipRecording(pos).record.BlockHash {
			t.Fatalf("unexpected frozen recording for pos %d: loaded=%+v", pos, loaded)
		}
	}
}

func TestBlockRecordsFreezerSurvivesUncleanShutdown(t *testing.T) {
	dir := t.TempDir()
	store := newBlockRecordsFreezer(newTestBlockRecordsFreezer(t, dir))
	firstPos := arbutil.MessageIndex(1000)
	latestPos := arbutil.MessageIndex(1004)
	for pos := firstPos; pos <= latestPos; pos++ {
		if err := store.writeRecording(testChainTipRecording(pos)); err != nil {
			t.Fatal(err)
		}
	}

	crashed := filepath.Join(t.TempDir(), "ancient")
	if err := os.CopyFS(crashed, os.DirFS(dir)); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	recovered := newBlockRecordsFreezer(newTestBlockRecordsFreezer(t, crashed))
	defer recovered.Close()
	for pos := firstPos; pos <= latestPos; pos++ {
		loaded, err := recovered.readRecording(pos)
		if err != nil {
			t.Fatal(err)
		}
		if loaded == nil || loaded.record.BlockHash != testChainTipRecording(pos).record.BlockHash {
			t.Fatalf("expected recording for pos %d to survive an unclean shutdown, got loaded=%+v", pos, loaded)
		}
	}
	if err := recovered.writeRecording(testChainTipRecording(latestPos + 1)); err != nil {
		t.Fatal(err)
	}
}

func TestBlockRecordsFreezerReadMissing(t *testing.T) {
	store := newBlockRecordsFreezer(newTestBlockRecordsFreezer(t, ""))
	loaded, err := store.readRecording(arbutil.MessageIndex(42))
	if err != nil {
		t.Fatal(err)
	}
	if loaded != nil {
		t.Fatalf("expected missing recording, got loaded=%+v", loaded)
	}

	if err := store.writeRecording(testChainTipRecording(10)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.readRecording(9); err == nil || !strings.Contains(err.Error(), "below the freezer tail") {
		t.Fatalf("expected a below tail error for pos 9, got err=%v", err)
	}
	loaded, err = store.readRecording(11)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != nil {
		t.Fatalf("expected missing recording for pos 11, got loaded=%+v", loaded)
	}
}

func TestBlockRecordsFreezerReorgTruncatesHead(t *testing.T) {
	newFreezer := func(t *testing.T) ethdb.ResettableAncientStore { return newTestBlockRecordsFreezer(t, t.TempDir()) }
	freezer := newFreezer(t)
	store := newBlockRecordsFreezer(freezer)
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
	if frozen != uint64(reorgPos)+1 {
		t.Fatalf("expected freezer head at %d after reorg, got %d", uint64(reorgPos)+1, frozen)
	}
	loaded, err := store.readRecording(reorgPos)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.record.BlockHash != reorged.record.BlockHash {
		t.Fatalf("expected reorged recording, got loaded=%+v", loaded)
	}
	loaded, err = store.readRecording(reorgPos - 1)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.record.BlockHash != testChainTipRecording(reorgPos-1).record.BlockHash {
		t.Fatalf("expected recording below reorg point to survive, got loaded=%+v", loaded)
	}
	loaded, err = store.readRecording(reorgPos + 1)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != nil {
		t.Fatalf("expected recording above reorg point to be truncated, got loaded=%+v", loaded)
	}

	if err := store.writeRecording(testChainTipRecording(reorgPos + 1)); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.readRecording(reorgPos + 1)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.record.Pos != reorgPos+1 {
		t.Fatalf("expected recording appended after reorg, got loaded=%+v", loaded)
	}
}

type recordingFreezer struct {
	ethdb.ResettableAncientStore
	ops       []string
	syncs     int
	appendErr error
	syncErr   error
}

func (f *recordingFreezer) ModifyAncients(fn func(ethdb.AncientWriteOp) error) (int64, error) {
	f.ops = append(f.ops, "append")
	if f.appendErr != nil {
		return 0, f.appendErr
	}
	return f.ResettableAncientStore.ModifyAncients(fn)
}

func (f *recordingFreezer) TruncateHead(items uint64) (uint64, error) {
	f.ops = append(f.ops, "truncateHead")
	return f.ResettableAncientStore.TruncateHead(items)
}

func (f *recordingFreezer) Reset() error {
	f.ops = append(f.ops, "reset")
	return f.ResettableAncientStore.Reset()
}

func (f *recordingFreezer) SyncAncient() error {
	f.ops = append(f.ops, "sync")
	f.syncs++
	if f.syncErr != nil {
		return f.syncErr
	}
	return f.ResettableAncientStore.SyncAncient()
}

func TestBlockRecordsFreezerReorgToTailRewritesHead(t *testing.T) {
	freezer := &recordingFreezer{ResettableAncientStore: newTestBlockRecordsFreezer(t, "")}
	store := newBlockRecordsFreezer(freezer)
	firstPos := arbutil.MessageIndex(1)
	for pos := firstPos; pos <= 5; pos++ {
		if err := store.writeRecording(testChainTipRecording(pos)); err != nil {
			t.Fatal(err)
		}
	}

	freezer.ops = nil
	reorged := testChainTipRecording(firstPos)
	reorged.record.BlockHash = testHash(250)
	if err := store.writeRecording(reorged); err != nil {
		t.Fatal(err)
	}
	for _, op := range freezer.ops {
		if op == "reset" {
			t.Fatalf("expected a reorg to the tail to rewrite the head, got ops %v", freezer.ops)
		}
	}
	if freezer.ops[0] != "truncateHead" {
		t.Fatalf("expected the head to be rolled back first, got ops %v", freezer.ops)
	}
	loaded, err := store.readRecording(firstPos)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.record.BlockHash != reorged.record.BlockHash {
		t.Fatalf("expected the reorged recording at the tail, got loaded=%+v", loaded)
	}
	if tail, err := freezer.Tail(rawdb.ChainTipBlockRecordsGroup); err != nil || tail != uint64(firstPos) {
		t.Fatalf("expected the tail to stay at %d, got tail=%d err=%v", firstPos, tail, err)
	}
}

func TestBlockRecordsFreezerSyncsEveryWrite(t *testing.T) {
	freezer := &recordingFreezer{ResettableAncientStore: newTestBlockRecordsFreezer(t, "")}
	store := newBlockRecordsFreezer(freezer)
	for pos := arbutil.MessageIndex(1); pos <= 5; pos++ {
		if err := store.writeRecording(testChainTipRecording(pos)); err != nil {
			t.Fatal(err)
		}
	}
	if freezer.syncs != 5 {
		t.Fatalf("expected 5 freezer syncs, got %d", freezer.syncs)
	}
	want := []string{"append", "sync", "append", "sync", "append", "sync", "append", "sync", "append", "sync"}
	if !reflect.DeepEqual(freezer.ops, want) {
		t.Fatalf("expected every append to be followed by a sync, got %v", freezer.ops)
	}
}

func TestBlockRecordsFreezerWriteFailuresPropagate(t *testing.T) {
	appendFailure := errors.New("append failed")
	syncFailure := errors.New("sync failed")
	for _, testCase := range []struct {
		name      string
		appendErr error
		syncErr   error
		want      error
	}{
		{name: "AppendFails", appendErr: appendFailure, want: appendFailure},
		{name: "SyncFails", syncErr: syncFailure, want: syncFailure},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			freezer := &recordingFreezer{
				ResettableAncientStore: newTestBlockRecordsFreezer(t, ""),
				appendErr:              testCase.appendErr,
				syncErr:                testCase.syncErr,
			}
			store := newBlockRecordsFreezer(freezer)
			err := store.writeRecording(testChainTipRecording(1))
			if !errors.Is(err, testCase.want) {
				t.Fatalf("expected the write failure to propagate, got %v", err)
			}

			engine := newTestRecorderEngine(t, 3)
			recorder := NewChainTipBlockRecorder(engine, store)
			block := engine.bc.GetBlockByNumber(2)
			if err := recorder.RecordTip(block, nil, block.NumberU64(), nil, nil); !errors.Is(err, testCase.want) {
				t.Fatalf("expected RecordTip to propagate the write failure, got %v", err)
			}
		})
	}
}

func TestBlockRecordsFreezerRejectsPositionMismatch(t *testing.T) {
	freezer := newTestBlockRecordsFreezer(t, "")
	store := newBlockRecordsFreezer(freezer)
	persisted, err := persistableChainTipRecording(testChainTipRecording(7))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := rlp.EncodeToBytes(persisted)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := freezer.ModifyAncients(func(writer ethdb.AncientWriteOp) error {
		return writer.AppendRaw(rawdb.ChainTipBlockRecordsFreezerTable, 0, encoded)
	}); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.readRecording(0)
	if err == nil || !strings.Contains(err.Error(), "expected 0") {
		t.Fatalf("expected a message index mismatch error, got loaded=%+v err=%v", loaded, err)
	}
}

func TestBlockRecordsFreezerRefusesToRecordOverGap(t *testing.T) {
	newFreezer := func(t *testing.T) ethdb.ResettableAncientStore { return newTestBlockRecordsFreezer(t, t.TempDir()) }
	freezer := newFreezer(t)
	store := newBlockRecordsFreezer(freezer)
	for pos := arbutil.MessageIndex(1000); pos <= 1009; pos++ {
		if err := store.writeRecording(testChainTipRecording(pos)); err != nil {
			t.Fatal(err)
		}
	}

	if err := store.writeRecording(testChainTipRecording(2000)); err == nil {
		t.Fatal("expected recording over a gap to fail")
	}

	loaded, err := store.readRecording(1005)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.record.Pos != 1005 {
		t.Fatalf("expected stored recordings to survive the refused write, got loaded=%+v", loaded)
	}
	if err := store.writeRecording(testChainTipRecording(1010)); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.readRecording(1010)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.record.Pos != 1010 {
		t.Fatalf("expected recording to continue at the head after the refused write, got loaded=%+v", loaded)
	}
}

func TestBlockRecordsFreezerResetsForUnrepresentablePositions(t *testing.T) {
	forEachFreezerBackend(t, testBlockRecordsFreezerResetsForUnrepresentablePositions)
}

func testBlockRecordsFreezerResetsForUnrepresentablePositions(t *testing.T, newFreezer func(t *testing.T) ethdb.ResettableAncientStore) {
	testCases := []struct {
		name     string
		prune    uint64 // freezer tail truncation before the write, if non-zero
		writePos arbutil.MessageIndex
	}{
		{name: "BelowFirstRecording", writePos: 500},
		{name: "BelowPrunedTail", prune: 1005, writePos: 3},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			freezer := newFreezer(t)
			store := newBlockRecordsFreezer(freezer)
			for pos := arbutil.MessageIndex(1000); pos <= 1009; pos++ {
				if err := store.writeRecording(testChainTipRecording(pos)); err != nil {
					t.Fatal(err)
				}
			}
			if testCase.prune > 0 {
				if _, err := freezer.TruncateTail(rawdb.ChainTipBlockRecordsGroup, testCase.prune); err != nil {
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
			if frozen != uint64(testCase.writePos)+1 {
				t.Fatalf("expected freezer head at %d after reset, got %d", uint64(testCase.writePos)+1, frozen)
			}
			tail, err := freezer.Tail(rawdb.ChainTipBlockRecordsGroup)
			if err != nil {
				t.Fatal(err)
			}
			if tail != uint64(testCase.writePos) {
				t.Fatalf("expected freezer tail at %d after reset, got %d", testCase.writePos, tail)
			}
			loaded, err := store.readRecording(1005)
			if err != nil {
				t.Fatal(err)
			}
			if loaded != nil {
				t.Fatalf("expected stale recording to be dropped, got loaded=%+v", loaded)
			}
			loaded, err = store.readRecording(testCase.writePos)
			if err != nil {
				t.Fatal(err)
			}
			if loaded == nil || loaded.record.Pos != testCase.writePos {
				t.Fatalf("expected recording after reset, got loaded=%+v", loaded)
			}
			if err := store.writeRecording(testChainTipRecording(testCase.writePos + 1)); err != nil {
				t.Fatal(err)
			}
			loaded, err = store.readRecording(testCase.writePos + 1)
			if err != nil {
				t.Fatal(err)
			}
			if loaded == nil || loaded.record.Pos != testCase.writePos+1 {
				t.Fatalf("expected recording appended after reset, got loaded=%+v", loaded)
			}
		})
	}
}

func TestBlockRecordsFreezerCorruptContentsAreErrors(t *testing.T) {
	freezer := newTestBlockRecordsFreezer(t, "")
	if _, err := freezer.ModifyAncients(func(writer ethdb.AncientWriteOp) error {
		return writer.AppendRaw(rawdb.ChainTipBlockRecordsFreezerTable, 0, []byte{0xde, 0xad, 0xbe, 0xef})
	}); err != nil {
		t.Fatal(err)
	}
	store := newBlockRecordsFreezer(freezer)

	if _, err := store.readRecording(0); err == nil {
		t.Fatal("expected an undecodable record to be an error, not a miss")
	}
	if err := store.writeRecording(testChainTipRecording(42)); err == nil {
		t.Fatal("expected recording over the gap above corrupt contents to fail")
	}
	// Reading a corrupt record must not stop recording, and must not affect
	// positions that decode.
	if err := store.writeRecording(testChainTipRecording(1)); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.readRecording(1)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.record.Pos != 1 {
		t.Fatalf("expected the readable recording to be served, got loaded=%+v", loaded)
	}
}

func TestBlockRecordsFreezerAdvancesOverGapWhenEmpty(t *testing.T) {
	forEachFreezerBackend(t, testBlockRecordsFreezerAdvancesOverGapWhenEmpty)
}

func testBlockRecordsFreezerAdvancesOverGapWhenEmpty(t *testing.T, newFreezer func(t *testing.T) ethdb.ResettableAncientStore) {
	freezer := newFreezer(t)
	store := newBlockRecordsFreezer(freezer)
	for pos := arbutil.MessageIndex(1000); pos <= 1002; pos++ {
		if err := store.writeRecording(testChainTipRecording(pos)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := freezer.TruncateTail(rawdb.ChainTipBlockRecordsGroup, 1003); err != nil {
		t.Fatal(err)
	}
	pos := arbutil.MessageIndex(2000)
	if err := store.writeRecording(testChainTipRecording(pos)); err != nil {
		t.Fatal(err)
	}
	tail, err := freezer.Tail(rawdb.ChainTipBlockRecordsGroup)
	if err != nil {
		t.Fatal(err)
	}
	if tail != uint64(pos) {
		t.Fatalf("expected freezer tail at %d, got %d", pos, tail)
	}
	loaded, err := store.readRecording(pos)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.record.Pos != pos {
		t.Fatalf("expected recording after gap advance, got loaded=%+v", loaded)
	}
}

func TestBlockRecordsFreezerPrunesRecordings(t *testing.T) {
	for _, firstPos := range []arbutil.MessageIndex{0, 1000} {
		t.Run(fmt.Sprintf("FirstPos%d", firstPos), func(t *testing.T) {
			freezer := newTestBlockRecordsFreezer(t, "")
			store := newBlockRecordsFreezer(freezer)
			latestPos := firstPos + 10
			for pos := firstPos; pos <= latestPos; pos++ {
				if err := store.writeRecording(testChainTipRecording(pos)); err != nil {
					t.Fatal(err)
				}
			}

			if err := store.pruneRecordingsBefore(firstPos); err != nil {
				t.Fatal(err)
			}
			tail, err := freezer.Tail(rawdb.ChainTipBlockRecordsGroup)
			if err != nil {
				t.Fatal(err)
			}
			if tail != uint64(firstPos) {
				t.Fatalf("expected freezer tail %d after no-op pruning, got %d", firstPos, tail)
			}

			if err := store.pruneRecordingsBefore(firstPos + 1); err != nil {
				t.Fatal(err)
			}
			if _, err := store.readRecording(firstPos); err == nil || !strings.Contains(err.Error(), "below the freezer tail") {
				t.Fatalf("expected recording %d to be pruned, got err=%v", firstPos, err)
			}
			loaded, err := store.readRecording(firstPos + 1)
			if err != nil {
				t.Fatal(err)
			}
			if loaded == nil || loaded.record.Pos != firstPos+1 {
				t.Fatalf("expected recording %d to remain, got loaded=%+v", firstPos+1, loaded)
			}
			tail, err = freezer.Tail(rawdb.ChainTipBlockRecordsGroup)
			if err != nil {
				t.Fatal(err)
			}
			if tail != uint64(firstPos)+1 {
				t.Fatalf("expected freezer tail %d, got %d", uint64(firstPos)+1, tail)
			}

			// Pruning beyond the last recording empties the freezer without error.
			if err := store.pruneRecordingsBefore(latestPos + 100); err != nil {
				t.Fatal(err)
			}
			if _, err := store.readRecording(latestPos); err == nil || !strings.Contains(err.Error(), "below the freezer tail") {
				t.Fatalf("expected all recordings pruned, got err=%v", err)
			}
			head, err := freezer.Ancients()
			if err != nil {
				t.Fatal(err)
			}
			tail, err = freezer.Tail(rawdb.ChainTipBlockRecordsGroup)
			if err != nil {
				t.Fatal(err)
			}
			if tail != head || head != uint64(latestPos)+1 {
				t.Fatalf("expected empty freezer at %d, got tail %d head %d", uint64(latestPos)+1, tail, head)
			}
		})
	}
}
func TestChainTipRecorderReadsPersistedRecording(t *testing.T) {
	engine := newTestRecorderEngine(t, 3)
	store := newBlockRecordsFreezer(newTestBlockRecordsFreezer(t, ""))
	recorder := NewChainTipBlockRecorder(engine, store)

	block := engine.bc.GetBlockByNumber(2)
	pos, err := engine.BlockNumberToMessageIndex(block.NumberU64())
	if err != nil {
		t.Fatal(err)
	}
	preimageHash := testHash(1)
	if err := store.writeRecording(&chainTipRecording{
		record: &execution.RecordResult{
			Pos:       pos,
			BlockHash: block.Hash(),
			Preimages: map[common.Hash][]byte{
				preimageHash: {1, 2, 3},
			},
		},
		firstHeaderNumber: block.NumberU64() - 1,
	}); err != nil {
		t.Fatal(err)
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
	parent := engine.bc.GetHeaderByNumber(block.NumberU64() - 1)
	if _, ok := record.Preimages[parent.Hash()]; !ok {
		t.Fatal("expected parent header preimage in served recording")
	}
	if recorder.ServedTipRecordings() != 1 {
		t.Fatalf("expected served count 1, got %d", recorder.ServedTipRecordings())
	}

	stale := engine.bc.GetBlockByNumber(3)
	stalePos, err := engine.BlockNumberToMessageIndex(stale.NumberU64())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.writeRecording(&chainTipRecording{
		record: &execution.RecordResult{
			Pos:       stalePos,
			BlockHash: testHash(99),
		},
		firstHeaderNumber: stale.NumberU64(),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := recorder.Recording(stalePos, nil); err == nil {
		t.Fatal("expected stale recording to be rejected")
	}
}
