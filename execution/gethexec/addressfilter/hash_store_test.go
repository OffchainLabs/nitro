// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

func TestHashStorePingPongReuse(t *testing.T) {
	store := newHashStore(100, 1000)
	salt := uuid.New()
	addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	h1 := HashStringInputWithPrefix(GetHashStringInputPrefix(salt), addr1)
	h2 := HashStringInputWithPrefix(GetHashStringInputPrefix(salt), addr2)

	d0 := store.buffers[0]
	d1 := store.buffers[1]
	require.NotNil(t, d0)
	require.NotNil(t, d1)

	store.Store(uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h1}, "e1")
	require.Same(t, d1, store.data.Load(), "first store should publish buffer 1")
	if r, _ := store.IsRestricted(addr1); !r {
		t.Fatal("addr1 should be restricted after first store")
	}
	if r, _ := store.IsRestricted(addr2); r {
		t.Fatal("addr2 should not be restricted after first store")
	}
	require.Equal(t, 1, store.Size())

	store.Store(uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h2}, "e2")
	require.Same(t, d0, store.data.Load(), "second store should publish buffer 0")
	if r, _ := store.IsRestricted(addr2); !r {
		t.Fatal("addr2 should be restricted after second store")
	}
	if r, _ := store.IsRestricted(addr1); r {
		t.Fatal("addr1 should no longer be restricted after second store")
	}
	require.Equal(t, "e2", store.Digest())

	// Buffers are reused, never reallocated.
	require.Same(t, d0, store.buffers[0])
	require.Same(t, d1, store.buffers[1])
}

func TestHashStoreSnapshotStableAcrossSwap(t *testing.T) {
	store := newHashStore(100, 1000)
	salt := uuid.New()
	addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	h1 := HashStringInputWithPrefix(GetHashStringInputPrefix(salt), addr1)
	h2 := HashStringInputWithPrefix(GetHashStringInputPrefix(salt), addr2)

	store.Store(uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h1}, "e1")
	snap := store.data.Load() // hold the old snapshot

	// The next store reuses the other buffer; it must not mutate snap.
	store.Store(uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h2}, "e2")

	_, ok := snap.hashes[snap.hashAddress(addr1)]
	require.True(t, ok, "old snapshot should still contain addr1 after a swap")
}

func TestHashStoreDisabledModeAllocatesNewData(t *testing.T) {
	store := NewHashStore(100) // maxHashes == 0
	require.Equal(t, 0, store.maxHashes)
	salt := uuid.New()
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	h := HashStringInputWithPrefix(GetHashStringInputPrefix(salt), addr)

	store.Store(uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h}, "e1")
	d1 := store.data.Load()
	store.Store(uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h}, "e2")
	d2 := store.data.Load()
	require.NotSame(t, d1, d2, "disabled mode should allocate a new hashData each Store")
}
