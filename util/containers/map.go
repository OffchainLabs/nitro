package containers

import "sync"

// Map implements thread-safe generic map.
type Map[K comparable, V any] struct {
	m sync.Map
}

// Delete deletes the key. If it doesn't exist it's a no-op.
func (m *Map[K, V]) Delete(k K) {
	m.m.Delete(k)
}

// Load retrieves the key from the map.
func (m *Map[K, V]) Load(k K) (val V, found bool) {
	v, ok := m.m.Load(k)
	if !ok {
		return val, false
	}
	return v.(V), true
}

// Store stores key/value in the map.
func (m *Map[K, V]) Store(k K, v V) {
	m.m.Store(k, v)
}
