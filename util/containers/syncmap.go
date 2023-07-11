package containers

import "sync"

type SyncMap[K any, V any] struct {
	internal sync.Map
}

func (m *SyncMap[K, V]) Load(key K) (V, bool) {
	val, found := m.internal.Load(key)
	if !found {
		var empty V
		return empty, false
	}
	return val.(V), true
}

func (m *SyncMap[K, V]) Store(key K, val V) {
	m.internal.Store(key, val)
}

func (m *SyncMap[K, V]) Delete(key K) {
	m.internal.Delete(key)
}
