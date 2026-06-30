// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"context"
	"encoding/binary"
	"math/rand/v2"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

func TestHashStorePingPongReuse(t *testing.T) {
	store := newHashStore(100, 1000)
	store.reuseGrace = 0 // reuse buffers immediately; deterministic ping-pong
	ctx := context.Background()
	salt := uuid.New()
	addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	h1 := HashStringInputWithPrefix(GetHashStringInputPrefix(salt), addr1)
	h2 := HashStringInputWithPrefix(GetHashStringInputPrefix(salt), addr2)

	d0 := store.buffers[0]
	d1 := store.buffers[1]
	require.NotNil(t, d0)
	require.NotNil(t, d1)

	require.NoError(t, store.Store(ctx, uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h1}, "e1"))
	require.Same(t, d1, store.data.Load(), "first store should publish buffer 1")
	if r, _ := store.IsRestricted(addr1); !r {
		t.Fatal("addr1 should be restricted after first store")
	}
	if r, _ := store.IsRestricted(addr2); r {
		t.Fatal("addr2 should not be restricted after first store")
	}
	require.Equal(t, 1, store.Size())

	require.NoError(t, store.Store(ctx, uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h2}, "e2"))
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
	store.reuseGrace = 0
	ctx := context.Background()
	salt := uuid.New()
	addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	h1 := HashStringInputWithPrefix(GetHashStringInputPrefix(salt), addr1)
	h2 := HashStringInputWithPrefix(GetHashStringInputPrefix(salt), addr2)

	require.NoError(t, store.Store(ctx, uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h1}, "e1"))
	snap := store.data.Load() // hold the old snapshot

	// The next store reuses the other buffer; it must not mutate snap.
	require.NoError(t, store.Store(ctx, uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h2}, "e2"))

	_, ok := snap.hashes[snap.hashAddress(addr1)]
	require.True(t, ok, "old snapshot should still contain addr1 after a swap")
}

func TestHashStoreDisabledModeAllocatesNewData(t *testing.T) {
	store := NewHashStore(100) // maxHashes == 0
	require.Equal(t, 0, store.maxHashes)
	ctx := context.Background()
	salt := uuid.New()
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	h := HashStringInputWithPrefix(GetHashStringInputPrefix(salt), addr)

	require.NoError(t, store.Store(ctx, uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h}, "e1"))
	d1 := store.data.Load()
	require.NoError(t, store.Store(ctx, uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h}, "e2"))
	d2 := store.data.Load()
	require.NotSame(t, d1, d2, "disabled mode should allocate a new hashData each Store")
}

// TestHashStorePreallocConcurrentReuseRaceFree is the regression guard for the
// buffer-reuse data race. Run under -race: each Store waits out the grace before
// recycling a buffer, so no concurrent reader can still hold it. Without the
// grace wait this fails under -race. The grace is set well above any plausible
// reader deschedule so the test is not timing-flaky; with the wait removed the
// race fires regardless of the grace value.
func TestHashStorePreallocConcurrentReuseRaceFree(t *testing.T) {
	store := newHashStore(100, 1000)
	store.reuseGrace = 400 * time.Millisecond
	ctx := context.Background()
	salt := uuid.New()
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	h := HashStringInputWithPrefix(GetHashStringInputPrefix(salt), addr)

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for range 4 {
		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
					store.IsRestricted(addr)
				}
			}
		})
	}

	// Each store after the first waits out the grace, so a buffer is recycled
	// only long after it was retired, while readers keep hammering it.
	for range 5 {
		require.NoError(t, store.Store(ctx, uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h}, "e"))
	}
	close(stop)
	wg.Wait()
}

// benchIsRestrictedAddrs is the number of restricted addresses each IsRestricted
// benchmark loads and queries.
const benchIsRestrictedAddrs = 10000

// benchmarkIsRestricted measures IsRestricted against benchIsRestrictedAddrs restricted
// addresses, querying them in random order so the LRU hit rate tracks the cache capacity:
// a cacheSize of 0, half, or all of the addresses gives roughly 0%, 50%, or 100% hits.
func benchmarkIsRestricted(b *testing.B, cacheSize int) {
	salt := uuid.New()
	prefix := GetHashStringInputPrefix(salt)
	addrs := make([]common.Address, benchIsRestrictedAddrs)
	hashes := make([]common.Hash, benchIsRestrictedAddrs)
	for i := range addrs {
		binary.LittleEndian.PutUint64(addrs[i][:], uint64(i)+1)
		hashes[i] = HashStringInputWithPrefix(prefix, addrs[i])
	}

	store := NewHashStore(cacheSize)
	require.NoError(b, store.Store(context.Background(), uuid.New(), salt, HashingSchemeStringInput, hashes, "bench"))

	// Precompute random indices so the timed loop does no RNG work. The length is a
	// power of two far larger than the cache, so masking is cheap and cycling back to
	// the start does not noticeably perturb the hit rate.
	rng := rand.New(rand.NewPCG(1, 2))
	const idxCount = 1 << 20
	idxs := make([]uint32, idxCount)
	for k := range idxs {
		idxs[k] = uint32(rng.IntN(benchIsRestrictedAddrs))
	}

	// b.Loop resets the timer on first call, so the setup above is not measured.
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		store.IsRestricted(addrs[idxs[i&(idxCount-1)]])
		i++
	}
}

func BenchmarkIsRestrictedNoCache(b *testing.B) {
	benchmarkIsRestricted(b, 0)
}

func BenchmarkIsRestrictedHalfCache(b *testing.B) {
	benchmarkIsRestricted(b, benchIsRestrictedAddrs/2)
}

func BenchmarkIsRestrictedFullCache(b *testing.B) {
	benchmarkIsRestricted(b, benchIsRestrictedAddrs)
}
