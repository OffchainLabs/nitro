package containers

import (
	"sync"
)

// thread safe version of containers.LruCache
type SafeLruCache[K comparable, V any] struct {
	inner *LruCache[K, V]
	mutex sync.RWMutex
}

func NewSafeLruCache[K comparable, V any](size int) *SafeLruCache[K, V] {
	return NewSafeLruCacheWithOnEvict[K, V](size, nil)
}

func NewSafeLruCacheWithOnEvict[K comparable, V any](size int, onEvict func(K, V)) *SafeLruCache[K, V] {
	return &SafeLruCache[K, V]{
		inner: NewLruCacheWithOnEvict(size, onEvict),
	}
}

// Returns true if an item was evicted
func (c *SafeLruCache[K, V]) Add(key K, value V) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.inner.Add(key, value)
}

func (c *SafeLruCache[K, V]) Get(key K) (V, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.inner.Get(key)
}

func (c *SafeLruCache[K, V]) Contains(key K) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.inner.Contains(key)
}

func (c *SafeLruCache[K, V]) Remove(key K) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.inner.Remove(key)
}

func (c *SafeLruCache[K, V]) GetOldest() (K, V, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.inner.GetOldest()
}

func (c *SafeLruCache[K, V]) RemoveOldest() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.inner.RemoveOldest()
}

func (c *SafeLruCache[K, V]) Len() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.inner.Len()
}

func (c *SafeLruCache[K, V]) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.inner.Size()
}

func (c *SafeLruCache[K, V]) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.inner.Clear()
}

func (c *SafeLruCache[K, V]) Resize(newSize int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.inner.Resize(newSize)
}
