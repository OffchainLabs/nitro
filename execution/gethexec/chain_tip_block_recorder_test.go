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
