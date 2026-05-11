// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func testHash(b byte) common.Hash {
	return common.BytesToHash(bytes.Repeat([]byte{b}, common.HashLength))
}

func TestRecentHeaderPreimageCacheStoresForksAtSameHeight(t *testing.T) {
	var cache recentHeaderPreimageCache
	headerA := testHash(1)
	headerB := testHash(2)
	parentA := testHash(3)
	parentB := testHash(4)

	cache.add(10, headerA, recentHeaderPreimage{
		parentHash: parentA,
		preimage:   []byte{1, 2, 3},
	})
	cache.add(10, headerB, recentHeaderPreimage{
		parentHash: parentB,
		preimage:   []byte{4, 5, 6},
	})

	entryA, ok := cache.get(10, headerA)
	if !ok {
		t.Fatal("expected cache hit for first fork")
	}
	if entryA.parentHash != parentA || !bytes.Equal(entryA.preimage, []byte{1, 2, 3}) {
		t.Fatalf("unexpected first fork entry: %+v", entryA)
	}

	entryB, ok := cache.get(10, headerB)
	if !ok {
		t.Fatal("expected cache hit for second fork")
	}
	if entryB.parentHash != parentB || !bytes.Equal(entryB.preimage, []byte{4, 5, 6}) {
		t.Fatalf("unexpected second fork entry: %+v", entryB)
	}
}

func TestRecentHeaderPreimageCacheReturnsCopies(t *testing.T) {
	var cache recentHeaderPreimageCache
	headerHash := testHash(1)
	cache.add(10, headerHash, recentHeaderPreimage{
		parentHash: testHash(2),
		preimage:   []byte{1, 2, 3},
	})

	entry, ok := cache.get(10, headerHash)
	if !ok {
		t.Fatal("expected cache hit")
	}
	entry.preimage[0] = 9

	entry, ok = cache.get(10, headerHash)
	if !ok {
		t.Fatal("expected cache hit after caller mutation")
	}
	if !bytes.Equal(entry.preimage, []byte{1, 2, 3}) {
		t.Fatalf("cache returned mutable preimage, got %v", entry.preimage)
	}
}

func TestRecentHeaderPreimageCacheEvictsByHeightSlot(t *testing.T) {
	var cache recentHeaderPreimageCache
	oldBlockNumber := uint64(7)
	newBlockNumber := oldBlockNumber + recentHeaderPreimageWindow
	oldHeaderA := testHash(1)
	oldHeaderB := testHash(2)
	newHeader := testHash(3)

	cache.add(oldBlockNumber, oldHeaderA, recentHeaderPreimage{parentHash: testHash(4), preimage: []byte{1}})
	cache.add(oldBlockNumber, oldHeaderB, recentHeaderPreimage{parentHash: testHash(5), preimage: []byte{2}})
	cache.add(newBlockNumber, newHeader, recentHeaderPreimage{parentHash: testHash(6), preimage: []byte{3}})

	if _, ok := cache.get(oldBlockNumber, oldHeaderA); ok {
		t.Fatal("expected old first fork entry to be evicted")
	}
	if _, ok := cache.get(oldBlockNumber, oldHeaderB); ok {
		t.Fatal("expected old second fork entry to be evicted")
	}
	if _, ok := cache.get(newBlockNumber, newHeader); !ok {
		t.Fatal("expected new slot entry to be cached")
	}
}

func TestRecentHeaderPreimageCacheRetainsOverlappingWindow(t *testing.T) {
	var cache recentHeaderPreimageCache
	for blockNumber := uint64(0); blockNumber < recentHeaderPreimageWindow; blockNumber++ {
		cache.add(blockNumber, testHash(byte(blockNumber)), recentHeaderPreimage{
			parentHash: testHash(byte(blockNumber + 1)),
			preimage:   []byte{byte(blockNumber)},
		})
	}

	cache.add(recentHeaderPreimageWindow, testHash(255), recentHeaderPreimage{
		parentHash: testHash(254),
		preimage:   []byte{255},
	})

	if _, ok := cache.get(0, testHash(0)); ok {
		t.Fatal("expected block 0 to be evicted")
	}
	for blockNumber := uint64(1); blockNumber < recentHeaderPreimageWindow; blockNumber++ {
		if _, ok := cache.get(blockNumber, testHash(byte(blockNumber))); !ok {
			t.Fatalf("expected block %d to remain cached", blockNumber)
		}
	}
	if _, ok := cache.get(recentHeaderPreimageWindow, testHash(255)); !ok {
		t.Fatal("expected newest block to be cached")
	}
}
