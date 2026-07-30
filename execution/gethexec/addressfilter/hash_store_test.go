// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"encoding/binary"
	"errors"
	"math/rand/v2"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

func storeHashes(t testing.TB, store *HashStore, id uuid.UUID, salt uuid.UUID, scheme HashingScheme, hashes []common.Hash, digest string) {
	t.Helper()
	require.NoError(t, store.Store(digest, len(hashes), func(addHash func(common.Hash)) (*ListMeta, error) {
		for _, h := range hashes {
			addHash(h)
		}
		return &ListMeta{ID: id, Salt: salt, Scheme: scheme}, nil
	}))
}

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

	storeHashes(t, store, uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h1}, "e1")
	require.Same(t, d1, store.data.Load(), "first store should publish buffer 1")
	if r, _ := store.IsRestricted(addr1); !r {
		t.Fatal("addr1 should be restricted after first store")
	}
	if r, _ := store.IsRestricted(addr2); r {
		t.Fatal("addr2 should not be restricted after first store")
	}
	require.Equal(t, 1, store.Size())

	storeHashes(t, store, uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h2}, "e2")
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

	storeHashes(t, store, uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h1}, "e1")
	snap := store.data.Load() // hold the old snapshot

	// The next store reuses the other buffer; it must not mutate snap.
	storeHashes(t, store, uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h2}, "e2")

	_, ok := snap.hashes[snap.hashAddress(addr1)]
	require.True(t, ok, "old snapshot should still contain addr1 after a swap")
}

func TestHashStoreDisabledModeAllocatesNewData(t *testing.T) {
	store := NewHashStore(100) // maxHashes == 0
	require.Equal(t, 0, store.maxHashes)
	salt := uuid.New()
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	h := HashStringInputWithPrefix(GetHashStringInputPrefix(salt), addr)

	storeHashes(t, store, uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h}, "e1")
	d1 := store.data.Load()
	storeHashes(t, store, uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h}, "e2")
	d2 := store.data.Load()
	require.NotSame(t, d1, d2, "disabled mode should allocate a new hashData each Store")
}

// TestHashStorePreallocConcurrentReuseRaceFree is the regression guard for the
// buffer-reuse data race. Readers hammer IsRestricted while Store rapidly
// recycles the ping-pong buffers in place. The per-buffer RWMutex makes Store
// wait for in-flight readers before clearing a buffer, so the concurrent map
// read/write (a fatal, unrecoverable error in Go) never happens. Run under
// -race; with the locking removed this fails.
func TestHashStorePreallocConcurrentReuseRaceFree(t *testing.T) {
	store := newHashStore(100, 1000)
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

	// Recycle the buffers many times in quick succession while readers hammer them.
	for range 1000 {
		storeHashes(t, store, uuid.New(), salt, HashingSchemeStringInput, []common.Hash{h}, "e")
	}
	close(stop)
	wg.Wait()
}

func TestHashStoreStoreFailureKeepsSnapshot(t *testing.T) {
	for _, maxHashes := range []int{0, 1000} {
		name := "prealloc"
		if maxHashes == 0 {
			name = "no prealloc"
		}
		t.Run(name, func(t *testing.T) {
			store := newHashStore(100, maxHashes)
			salt := uuid.New()
			addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
			addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
			h1 := HashStringInputWithPrefix(GetHashStringInputPrefix(salt), addr1)
			h2 := HashStringInputWithPrefix(GetHashStringInputPrefix(salt), addr2)

			id1 := uuid.New()
			storeHashes(t, store, id1, salt, HashingSchemeStringInput, []common.Hash{h1}, "e1")

			// A fill that inserts a hash and then fails must publish nothing.
			fillErr := errors.New("stream broke")
			err := store.Store("e2", 1, func(add func(common.Hash)) (*ListMeta, error) {
				add(h2)
				return nil, fillErr
			})
			require.ErrorIs(t, err, fillErr)

			restricted, gotID := store.IsRestricted(addr1)
			require.True(t, restricted, "old snapshot must survive a failed fill")
			require.Equal(t, id1, gotID)
			if restricted, _ := store.IsRestricted(addr2); restricted {
				t.Fatal("hash from the failed fill must not be visible")
			}
			require.Equal(t, "e1", store.Digest())
			require.Equal(t, 1, store.Size())

			// The next successful store must not resurrect the aborted fill's hashes.
			require.NoError(t, store.Store("e3", 1, func(add func(common.Hash)) (*ListMeta, error) {
				add(h1)
				return &ListMeta{ID: uuid.New(), Salt: salt, Scheme: HashingSchemeStringInput}, nil
			}))
			if restricted, _ := store.IsRestricted(addr2); restricted {
				t.Fatal("residue from the aborted fill leaked into the next snapshot")
			}
			if restricted, _ := store.IsRestricted(addr1); !restricted {
				t.Fatal("addr1 should be restricted after the successful reload")
			}
			require.Equal(t, "e3", store.Digest())
		})
	}
}

// TestHashStoreStoreConcurrentReaders verifies readers keep serving the old
// snapshot, race-free, while a slow streaming fill is in progress (run with -race).
func TestHashStoreStoreConcurrentReaders(t *testing.T) {
	store := newHashStore(100, 1000)
	salt := uuid.New()
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	h := HashStringInputWithPrefix(GetHashStringInputPrefix(salt), addr)
	id1 := uuid.New()
	storeHashes(t, store, id1, salt, HashingSchemeStringInput, []common.Hash{h}, "e1")

	fillStarted := make(chan struct{})
	finishFill := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- store.Store("e2", 1, func(add func(common.Hash)) (*ListMeta, error) {
			close(fillStarted)
			add(h)
			<-finishFill
			return &ListMeta{ID: uuid.New(), Salt: salt, Scheme: HashingSchemeStringInput}, nil
		})
	}()

	<-fillStarted
	// Mid-fill, readers must still see the previous snapshot.
	for range 100 {
		restricted, gotID := store.IsRestricted(addr)
		require.True(t, restricted)
		require.Equal(t, id1, gotID)
		require.Equal(t, "e1", store.Digest())
	}
	close(finishFill)
	require.NoError(t, <-done)
	require.Equal(t, "e2", store.Digest())
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
	var addrSeed uint64
	for i := range addrs {
		addrSeed++
		binary.LittleEndian.PutUint64(addrs[i][:], addrSeed)
		hashes[i] = HashStringInputWithPrefix(prefix, addrs[i])
	}

	store := NewHashStore(cacheSize)
	storeHashes(b, store, uuid.New(), salt, HashingSchemeStringInput, hashes, "bench")

	// Precompute random indices so the timed loop does no RNG work. The length is a
	// power of two far larger than the cache, so masking is cheap and cycling back to
	// the start does not noticeably perturb the hit rate.
	rng := rand.New(rand.NewPCG(1, 2))
	const idxCount = 1 << 20
	idxs := make([]uint32, idxCount)
	for k := range idxs {
		idxs[k] = rng.Uint32N(benchIsRestrictedAddrs)
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
