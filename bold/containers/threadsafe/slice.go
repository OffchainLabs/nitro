// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

// Package threadsafe defines generic, threadsafe analogues of common data structures
// in Go such as maps, slices, and sets for use in BoLD with an intuitive API.
package threadsafe

import (
	"sync"

	"github.com/offchainlabs/nitro/bold/containers/option"
)

type Slice[V any] struct {
	sync.RWMutex
	items []V
}

func NewSlice[V any]() *Slice[V] {
	return &Slice[V]{items: make([]V, 0)}
}

func (s *Slice[V]) Push(v V) {
	s.Lock()
	defer s.Unlock()
	s.items = append(s.items, v)
}

func (s *Slice[V]) Get(i int) option.Option[V] {
	s.RLock()
	defer s.RUnlock()
	if i >= len(s.items) {
		return option.None[V]()
	}
	return option.Some(s.items[i])
}

func (s *Slice[V]) Len() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.items)
}

func (s *Slice[V]) Find(fn func(idx int, elem V) bool) bool {
	s.RLock()
	defer s.RUnlock()
	for ii, vv := range s.items {
		if fn(ii, vv) {
			return true
		}
	}
	return false
}
