// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

// Package threadsafe includes generic utilities for maps and sets that can
// be safely used concurrently for type-safety at compile time with the
// bare minimum methods needed in this repository.
package threadsafe

import (
	"sync"

	"github.com/ethereum/go-ethereum/metrics"
)

type Set[T comparable] struct {
	sync.RWMutex
	items map[T]bool
	gauge *metrics.Gauge
}

type SetOpt[T comparable] func(*Set[T])

func SetWithMetric[T comparable](name string) SetOpt[T] {
	return func(s *Set[T]) {
		gauge := metrics.NewRegisteredGauge("arb/validator/threadsafe_set/"+name, nil)
		s.gauge = &gauge
	}
}

func NewSet[T comparable](opts ...SetOpt[T]) *Set[T] {
	s := &Set[T]{
		items: make(map[T]bool),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Set[T]) Insert(t T) {
	s.Lock()
	defer s.Unlock()
	s.items[t] = true
	if s.gauge != nil {
		(*s.gauge).Inc(1)
	}
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
	if s.gauge != nil {
		(*s.gauge).Dec(1)
	}
}

func (s *Set[T]) ForEach(fn func(elem T)) {
	s.RLock()
	defer s.RUnlock()
	for elem := range s.items {
		fn(elem)
	}
}
