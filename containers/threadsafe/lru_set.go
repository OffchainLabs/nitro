// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package threadsafe

import (
	"sync"

	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/metrics"
)

type LruSet[T comparable] struct {
	sync.RWMutex
	items lru.BasicLRU[T, bool]
	gauge *metrics.Gauge
}

type LruSetOpt[T comparable] func(*LruSet[T])

func LruSetWithMetric[T comparable](name string) LruSetOpt[T] {
	return func(s *LruSet[T]) {
		gauge := metrics.NewRegisteredGauge("arb/validator/threadsafe_lru_set/"+name, nil)
		s.gauge = &gauge
	}
}

func NewLruSet[T comparable](capacity int, opts ...LruSetOpt[T]) *LruSet[T] {
	s := &LruSet[T]{
		items: lru.NewBasicLRU[T, bool](capacity),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *LruSet[T]) Insert(t T) {
	s.Lock()
	defer s.Unlock()
	s.items.Add(t, true)
	if s.gauge != nil {
		(*s.gauge).Inc(1)
	}
}

func (s *LruSet[T]) NumItems() uint64 {
	s.RLock()
	defer s.RUnlock()
	return uint64(s.items.Len())
}

func (s *LruSet[T]) Has(t T) bool {
	s.RLock()
	defer s.RUnlock()
	return s.items.Contains(t)
}

func (s *LruSet[T]) Delete(t T) {
	s.Lock()
	defer s.Unlock()
	s.items.Remove(t)
	if s.gauge != nil {
		(*s.gauge).Dec(1)
	}
}

func (s *LruSet[T]) ForEach(fn func(elem T)) {
	s.RLock()
	defer s.RUnlock()
	for _, elem := range s.items.Keys() {
		fn(elem)
	}
}
