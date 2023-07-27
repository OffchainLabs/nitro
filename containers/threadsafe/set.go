// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

// Package threadsafe includes generic utilities for maps and sets that can
// be safely used concurrently for type-safety at compile time with the
// bare minimum methods needed in this repository.
package threadsafe

import "sync"

type Set[T comparable] struct {
	sync.RWMutex
	items map[T]bool
}

func NewSet[T comparable]() *Set[T] {
	return &Set[T]{
		items: make(map[T]bool),
	}
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
