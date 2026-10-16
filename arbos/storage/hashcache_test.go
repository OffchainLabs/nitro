// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package storage

import (
	"fmt"
	"testing"
)

func testKeys(n int) []string {
	keys := make([]string, n)
	for i := range keys {
		// Shaped like the real thing: a shared subspace prefix then a slot.
		keys[i] = fmt.Sprintf("arbos-subspace-prefix\x00%032d", i)
	}
	return keys
}

func TestShardedHashCacheRoundTrip(t *testing.T) {
	c := newShardedHashCache(storageKeyCacheShards, storageKeyCacheSize)
	keys := testKeys(1000)
	for i, k := range keys {
		c.Add(k, []byte{byte(i), byte(i >> 8)})
	}
	for i, k := range keys {
		got, ok := c.Get(k)
		if !ok {
			t.Fatalf("key %d missing", i)
		}
		if want := []byte{byte(i), byte(i >> 8)}; string(got) != string(want) {
			t.Fatalf("key %d: got %x want %x", i, got, want)
		}
	}
	if _, ok := c.Get("absent"); ok {
		t.Fatal("absent key reported as cached")
	}
}

func TestShardedHashCacheShardCountIsPowerOfTwo(t *testing.T) {
	for _, in := range []int{0, 1, 3, 64, 100} {
		c := newShardedHashCache(in, 8)
		n := len(c.shards)
		if n&(n-1) != 0 {
			t.Fatalf("newShardedHashCache(%d) gave %d shards, not a power of two", in, n)
		}
		if n < in {
			t.Fatalf("newShardedHashCache(%d) gave %d shards, fewer than requested", in, n)
		}
		if uint64(n-1) != c.mask {
			t.Fatalf("mask %d does not match %d shards", c.mask, n)
		}
	}
}

func TestShardedHashCacheSpreadsKeys(t *testing.T) {
	c := newShardedHashCache(storageKeyCacheShards, storageKeyCacheSize)
	counts := make([]int, len(c.shards))
	for _, k := range testKeys(10000) {
		counts[maphash.String(c.seed, k)&c.mask]++
	}
	// Keys share a long prefix; a prefix-sliced shard function would put them
	// all in one shard. Require every shard to see something.
	for i, n := range counts {
		if n == 0 {
			t.Fatalf("shard %d got no keys of 10000", i)
		}
	}
}

// BenchmarkHashCacheGetParallel contrasts the current single-mutex cache
// (shards=1) with the sharded one. mapAddress calls Get once per storage slot,
// so this is the hot path for eth_call and for block execution.
func BenchmarkHashCacheGetParallel(b *testing.B) {
	for _, shards := range []int{1, 8, 64} {
		b.Run(fmt.Sprintf("shards=%d", shards), func(b *testing.B) {
			c := newShardedHashCache(shards, storageKeyCacheSize)
			keys := testKeys(1024)
			val := make([]byte, 32)
			for _, k := range keys {
				c.Add(k, val)
			}
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					c.Get(keys[i&(len(keys)-1)])
					i++
				}
			})
		})
	}
}
