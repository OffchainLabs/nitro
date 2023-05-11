// Package threadsafe includes generic utilities for maps and sets that can
// be safely used concurrently for type-safety at compile time with the
// bare minimum methods needed in this repository.
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

func (s *Map[K, V]) ForEach(fn func(k K, v V)) {
	s.RLock()
	defer s.RUnlock()
	for k, v := range s.items {
		fn(k, v)
	}
}

type Set[T comparable] struct {
	sync.RWMutex
	items map[T]bool
}

func NewSet[T comparable]() *Set[T] {
	return &Set[T]{
		items: make(map[T]bool),
	}
}

func NewSetFromItems[T comparable](items []T) *Set[T] {
	s := &Set[T]{
		items: make(map[T]bool),
	}
	for _, item := range items {
		s.items[item] = true
	}
	return s
}

func (s *Set[T]) Insert(t T) {
	s.Lock()
	defer s.Unlock()
	s.items[t] = true
}

func (s *Set[T]) NumItems() uint64 {
	s.RLock()
	defer s.RUnlock()
	return uint64(len(s.items))
}

func (s *Set[T]) Has(t T) bool {
	s.RLock()
	defer s.RUnlock()
	return s.items[t]
}

func (s *Set[T]) Delete(t T) {
	s.Lock()
	defer s.Unlock()
	delete(s.items, t)
}

func (s *Set[T]) ForEach(fn func(elem T)) {
	s.RLock()
	defer s.RUnlock()
	for elem := range s.items {
		fn(elem)
	}
}
