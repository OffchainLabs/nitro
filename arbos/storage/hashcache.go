// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package storage

import (
	"hash/maphash"
	"math/bits"

	"github.com/ethereum/go-ethereum/common/lru"
)

// storageKeyCacheSize is the capacity of each shard, so the cache holds
// storageKeyCacheShards times this many entries.
const storageKeyCacheSize = 1024

// storageKeyCacheShards is rounded up to a power of two by newShardedHashCache.
const storageKeyCacheShards = 64

// shardedHashCache spreads the keccak cache across independently locked shards.
//
// geth's lru.Cache guards one BasicLRU with a single sync.Mutex, and Get takes
// that mutex in write mode because a lookup reorders recency. mapAddress
// consults this cache once per storage slot, and every eth_call opens ArbOS
// state before it executes an opcode, so an unsharded cache is a process-wide
// serialisation point shared by RPC and block execution.
type shardedHashCache struct {
	seed   maphash.Seed
	mask   uint64
	shards []*lru.Cache[string, []byte]
}

func newShardedHashCache(shards, sizePerShard int) *shardedHashCache {
	if shards < 1 {
		shards = 1
	}
	// Round up to a power of two so shardFor can mask instead of dividing.
	if bits.OnesCount(uint(shards)) != 1 {
		shards = 1 << bits.Len(uint(shards))
	}
	c := &shardedHashCache{
		seed:   maphash.MakeSeed(),
		mask:   uint64(shards - 1),
		shards: make([]*lru.Cache[string, []byte], shards),
	}
	for i := range c.shards {
		c.shards[i] = lru.NewCache[string, []byte](sizePerShard)
	}
	return c
}

// shardFor hashes the key rather than slicing it: keys are storageKey followed
// by a slot, so a prefix is shared by every slot under the same subspace and
// would land them all on one shard.
func (c *shardedHashCache) shardFor(key string) *lru.Cache[string, []byte] {
	return c.shards[maphash.String(c.seed, key)&c.mask]
}

func (c *shardedHashCache) Get(key string) ([]byte, bool) {
	return c.shardFor(key).Get(key)
}

func (c *shardedHashCache) Add(key string, value []byte) bool {
	return c.shardFor(key).Add(key, value)
}
