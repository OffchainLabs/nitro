// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package containers

import (
	"github.com/hashicorp/golang-lru/simplelru"
)

// Not thread safe!
// A zero or negative size means it has no capacity instead of unlimited.
type LruCache[K comparable, V any] struct {
	inner     *simplelru.LRU
	onEvicted func(key, value interface{})
}

func NewLruCache[K comparable, V any](size int) *LruCache[K, V] {
	return NewLruCacheWithOnEvict[K, V](size, nil)
}

func NewLruCacheWithOnEvict[K comparable, V any](size int, onEvict func(K, V)) *LruCache[K, V] {
	var untypedOnEvict func(key, value interface{})
	if onEvict != nil {
		untypedOnEvict = func(key, value interface{}) {
			castedKey, ok := key.(K)
			if !ok {
				panic("LRU cache has key of wrong type")
			}
			castedValue, ok := value.(V)
			if !ok {
				panic("LRU cache has value of wrong type")
			}
			onEvict(castedKey, castedValue)
		}
	}
	var inner *simplelru.LRU
	if size > 0 {
		// Can't fail because newSize > 0
		inner, _ = simplelru.NewLRU(size, untypedOnEvict)
	}
	return &LruCache[K, V]{
		inner:     inner,
		onEvicted: untypedOnEvict,
	}
}

func (c *LruCache[K, V]) Add(key K, value V) {
	if c.inner == nil {
		return
	}
	c.inner.Add(key, value)
}

func (c *LruCache[K, V]) Get(key K) (V, bool) {
	var empty V
	if c.inner == nil {
		return empty, false
	}
	value, ok := c.inner.Get(key)
	if !ok {
		return empty, false
	}
	casted, ok := value.(V)
	if !ok {
		panic("LRU cache has value of wrong type")
	}
	return casted, true
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
	key, value, ok := c.inner.GetOldest()
	if !ok {
		return emptyKey, emptyValue, false
	}
	castedKey, ok := key.(K)
	if !ok {
		panic("LRU cache has key of wrong type")
	}
	castedValue, ok := value.(V)
	if !ok {
		panic("LRU cache has value of wrong type")
	}
	return castedKey, castedValue, true
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
		if c.inner != nil && c.onEvicted != nil {
			c.inner.Purge() // run the evict functions
		}
		c.inner = nil
	} else if c.inner == nil {
		// Can't fail because newSize > 0
		c.inner, _ = simplelru.NewLRU(newSize, c.onEvicted)
	} else {
		c.inner.Resize(newSize)
	}
}
