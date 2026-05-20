// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/util/containers"
)

func testHash(b byte) common.Hash {
	return common.BytesToHash(bytes.Repeat([]byte{b}, common.HashLength))
}

func newTestChainTipRecorderWithHeaderCache(size int) *ChainTipBlockRecorder {
	return &ChainTipBlockRecorder{
		headerPreimages: containers.NewLruCache[recentHeaderPreimageKey, arbitrum.RecordedHeaderPreimage](size),
	}
}

func TestRecentHeaderPreimageCacheStoresForksAtSameHeight(t *testing.T) {
	recorder := newTestChainTipRecorderWithHeaderCache(recentHeaderPreimageCacheSlots)
	headerA := testHash(1)
	headerB := testHash(2)
	parentA := testHash(3)
	parentB := testHash(4)

	recorder.AddRecordedHeaderPreimage(10, headerA, arbitrum.RecordedHeaderPreimage{
		ParentHash: parentA,
		Preimage:   []byte{1, 2, 3},
	})
	recorder.AddRecordedHeaderPreimage(10, headerB, arbitrum.RecordedHeaderPreimage{
		ParentHash: parentB,
		Preimage:   []byte{4, 5, 6},
	})

	entryA, ok := recorder.GetRecordedHeaderPreimage(10, headerA)
	if !ok {
		t.Fatal("expected cache hit for first fork")
	}
	if entryA.ParentHash != parentA || !bytes.Equal(entryA.Preimage, []byte{1, 2, 3}) {
		t.Fatalf("unexpected first fork entry: %+v", entryA)
	}

	entryB, ok := recorder.GetRecordedHeaderPreimage(10, headerB)
	if !ok {
		t.Fatal("expected cache hit for second fork")
	}
	if entryB.ParentHash != parentB || !bytes.Equal(entryB.Preimage, []byte{4, 5, 6}) {
		t.Fatalf("unexpected second fork entry: %+v", entryB)
	}
}

func TestRecentHeaderPreimageCacheReturnsCopies(t *testing.T) {
	recorder := newTestChainTipRecorderWithHeaderCache(recentHeaderPreimageCacheSlots)
	headerHash := testHash(1)
	recorder.AddRecordedHeaderPreimage(10, headerHash, arbitrum.RecordedHeaderPreimage{
		ParentHash: testHash(2),
		Preimage:   []byte{1, 2, 3},
	})

	entry, ok := recorder.GetRecordedHeaderPreimage(10, headerHash)
	if !ok {
		t.Fatal("expected cache hit")
	}
	entry.Preimage[0] = 9

	entry, ok = recorder.GetRecordedHeaderPreimage(10, headerHash)
	if !ok {
		t.Fatal("expected cache hit after caller mutation")
	}
	if !bytes.Equal(entry.Preimage, []byte{1, 2, 3}) {
		t.Fatalf("cache returned mutable preimage, got %v", entry.Preimage)
	}
}

func TestRecentHeaderPreimageCacheEvictsLeastRecentlyUsed(t *testing.T) {
	recorder := newTestChainTipRecorderWithHeaderCache(2)
	oldHeader := testHash(1)
	middleHeader := testHash(2)
	newHeader := testHash(3)

	recorder.AddRecordedHeaderPreimage(1, oldHeader, arbitrum.RecordedHeaderPreimage{ParentHash: testHash(4), Preimage: []byte{1}})
	recorder.AddRecordedHeaderPreimage(2, middleHeader, arbitrum.RecordedHeaderPreimage{ParentHash: testHash(5), Preimage: []byte{2}})
	recorder.AddRecordedHeaderPreimage(3, newHeader, arbitrum.RecordedHeaderPreimage{ParentHash: testHash(6), Preimage: []byte{3}})

	if _, ok := recorder.GetRecordedHeaderPreimage(1, oldHeader); ok {
		t.Fatal("expected oldest entry to be evicted")
	}
	if _, ok := recorder.GetRecordedHeaderPreimage(2, middleHeader); !ok {
		t.Fatal("expected middle entry to remain cached")
	}
	if _, ok := recorder.GetRecordedHeaderPreimage(3, newHeader); !ok {
		t.Fatal("expected newest entry to be cached")
	}
}

func TestRecentHeaderPreimageCacheRetainsRecentEntries(t *testing.T) {
	recorder := newTestChainTipRecorderWithHeaderCache(recentHeaderPreimageCacheSlots)
	for blockNumber := uint64(0); blockNumber < recentHeaderPreimageCacheSlots; blockNumber++ {
		recorder.AddRecordedHeaderPreimage(blockNumber, testHash(byte(blockNumber)), arbitrum.RecordedHeaderPreimage{
			ParentHash: testHash(byte(blockNumber + 1)),
			Preimage:   []byte{byte(blockNumber)},
		})
	}

	recorder.AddRecordedHeaderPreimage(recentHeaderPreimageCacheSlots, testHash(255), arbitrum.RecordedHeaderPreimage{
		ParentHash: testHash(254),
		Preimage:   []byte{255},
	})

	if _, ok := recorder.GetRecordedHeaderPreimage(0, testHash(0)); ok {
		t.Fatal("expected block 0 to be evicted")
	}
	for blockNumber := uint64(1); blockNumber < recentHeaderPreimageCacheSlots; blockNumber++ {
		if _, ok := recorder.GetRecordedHeaderPreimage(blockNumber, testHash(byte(blockNumber))); !ok {
			t.Fatalf("expected block %d to remain cached", blockNumber)
		}
	}
	if _, ok := recorder.GetRecordedHeaderPreimage(recentHeaderPreimageCacheSlots, testHash(255)); !ok {
		t.Fatal("expected newest block to be cached")
	}
}
