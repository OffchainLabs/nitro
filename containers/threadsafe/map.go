// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package threadsafe

import "sync"

type Map[K comparable, V any] struct {
	sync.RWMutex
	items map[K]V
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{items: make(map[K]V)}
}

func NewMapFromItems[K comparable, V any](m map[K]V) *Map[K, V] {
	return &Map[K, V]{items: m}
}

func (s *Map[K, V]) Put(k K, v V) {
	s.Lock()
	defer s.Unlock()
	s.items[k] = v
}

func (s *Map[K, V]) Has(k K) bool {
	s.RLock()
	defer s.RUnlock()
	_, ok := s.items[k]
	return ok
}

func (s *Map[K, V]) NumItems() uint64 {
	s.RLock()
	defer s.RUnlock()
	return uint64(len(s.items))
}

func (s *Map[K, V]) TryGet(k K) (V, bool) {
	s.RLock()
	defer s.RUnlock()
	item, ok := s.items[k]
	return item, ok
}

func (s *Map[K, V]) Get(k K) V {
	s.RLock()
	defer s.RUnlock()
	return s.items[k]
}

func (s *Map[K, V]) Delete(k K) {
	s.Lock()
	defer s.Unlock()
	delete(s.items, k)
}

func (s *Map[K, V]) ForEach(fn func(k K, v V) error) error {
	s.RLock()
	defer s.RUnlock()
	for k, v := range s.items {
		if err := fn(k, v); err != nil {
			return err
		}
	}
	return nil
}
