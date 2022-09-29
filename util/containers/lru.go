// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package containers

import "github.com/golang/groupcache/lru"

// Not thread safe!
// In contrast to lru.Cache, as zero size means it has no capacity instead of unlimited.
type LruCache[K comparable, V any] struct {
	inner lru.Cache
}

func NewLruCache[K comparable, V any](size int) *LruCache[K, V] {
	inner := lru.New(size)
	return &LruCache[K, V]{inner: *inner}
}

func (c *LruCache[K, V]) Add(key K, value V) {
	if c.inner.MaxEntries <= 0 {
		return
	}
	c.inner.Add(key, value)
}

func (c *LruCache[K, V]) Get(key K) (V, bool) {
	value, ok := c.inner.Get(key)
	if !ok {
		var empty V
		return empty, false
	}
	casted, ok := value.(V)
	if !ok {
		panic("LRU cache has value of wrong type")
	}
	return casted, true
}

func (c *LruCache[K, V]) Remove(key K) {
	c.inner.Remove(key)
}

func (c *LruCache[K, V]) RemoveOldest() {
	c.inner.RemoveOldest()
}

func (c *LruCache[K, V]) Len() int {
	return c.inner.Len()
}

func (c *LruCache[K, V]) Clear() {
	c.inner.Clear()
}

func (c *LruCache[K, V]) GetSize() int {
	return c.inner.MaxEntries
}

func (c *LruCache[K, V]) Resize(newSize int) {
	c.inner.MaxEntries = newSize
	if newSize <= 0 {
		c.Clear()
	} else {
		for c.Len() > newSize {
			c.RemoveOldest()
		}
	}
}
