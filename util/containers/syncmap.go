package containers

import (
	"fmt"
	"sync"
)

type SyncMap[K any, V any] struct {
	internal sync.Map
}

func (m *SyncMap[K, V]) Load(key K) (V, bool) {
	val, found := m.internal.Load(key)
	if !found {
		var empty V
		return empty, false
	}
	vVal, ok := val.(V)
	if !ok {
		panic(fmt.Sprintf("type assertion failed on %s", val))
	}
	return vVal, true
}

func (m *SyncMap[K, V]) Store(key K, val V) {
	m.internal.Store(key, val)
}

func (m *SyncMap[K, V]) Delete(key K) {
	m.internal.Delete(key)
}

// Only used for testing
func (m *SyncMap[K, V]) Keys() []K {
	s := make([]K, 0)
	m.internal.Range(func(k, v interface{}) bool {
		kKey, ok := k.(K)
		if !ok {
			panic(fmt.Sprintf("type assertion failed on %s", k))
		}
		s = append(s, kKey)
		return true
	})
	return s
}
