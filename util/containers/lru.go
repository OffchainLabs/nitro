// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package containers

import (
	"github.com/hashicorp/golang-lru/v2/simplelru"
)

// Not thread safe!
// Unlike simplelru, a zero or negative size means it has no capacity and is always empty.
type LruCache[K comparable, V any] struct {
	inner   *simplelru.LRU[K, V]
	onEvict func(key K, value V)
}

func NewLruCache[K comparable, V any](size int) *LruCache[K, V] {
	return NewLruCacheWithOnEvict[K, V](size, nil)
}

func NewLruCacheWithOnEvict[K comparable, V any](size int, onEvict func(K, V)) *LruCache[K, V] {
	var inner *simplelru.LRU[K, V]
	if size > 0 {
		// Can't fail because newSize > 0
		inner, _ = simplelru.NewLRU(size, onEvict)
	}
	return &LruCache[K, V]{
		inner:   inner,
		onEvict: onEvict,
	}
}

// Returns true if an item was evicted
func (c *LruCache[K, V]) Add(key K, value V) bool {
	if c.inner == nil {
		return true
	}
	return c.inner.Add(key, value)
}

func (c *LruCache[K, V]) Get(key K) (V, bool) {
	var empty V
	if c.inner == nil {
		return empty, false
	}
	return c.inner.Get(key)
}

func (c *LruCache[K, V]) Contains(key K) bool {
	if c.inner == nil {
		return false
	}
	return c.inner.Contains(key)
}

func (c *LruCache[K, V]) Remove(key K) {
	if c.inner == nil {
		return
	}
	c.inner.Remove(key)
}

func (c *LruCache[K, V]) GetOldest() (K, V, bool) {
	var emptyKey K
	var emptyValue V
	if c.inner == nil {
		return emptyKey, emptyValue, false
	}
	return c.inner.GetOldest()
}

func (c *LruCache[K, V]) RemoveOldest() {
	if c.inner == nil {
		return
	}
	c.inner.RemoveOldest()
}

func (c *LruCache[K, V]) Len() int {
	if c.inner == nil {
		return 0
	}
	return c.inner.Len()
}

func (c *LruCache[K, V]) Clear() {
	if c.inner == nil {
		return
	}
	c.inner.Purge()
}

func (c *LruCache[K, V]) Resize(newSize int) {
	if newSize <= 0 {
		if c.inner != nil && c.onEvict != nil {
			c.inner.Purge() // run the evict functions
		}
		c.inner = nil
	} else if c.inner == nil {
		// Can't fail because newSize > 0
		c.inner, _ = simplelru.NewLRU(newSize, c.onEvict)
	} else {
		c.inner.Resize(newSize)
	}
}
