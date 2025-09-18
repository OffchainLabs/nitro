// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package threadsafe

import (
	"sync"

	"github.com/ethereum/go-ethereum/metrics"
)

type Map[K comparable, V any] struct {
	sync.RWMutex
	items map[K]V
	gauge *metrics.Gauge
}

type MapOpt[K comparable, V any] func(*Map[K, V])

func MapWithMetric[K comparable, V any](name string) MapOpt[K, V] {
	return func(m *Map[K, V]) {
		gauge := metrics.NewRegisteredGauge("arb/validator/threadsafe_map/"+name, nil)
		m.gauge = gauge
	}
}

func NewMap[K comparable, V any](opts ...MapOpt[K, V]) *Map[K, V] {
	m := &Map[K, V]{items: make(map[K]V)}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func NewMapFromItems[K comparable, V any](m map[K]V) *Map[K, V] {
	return &Map[K, V]{items: m}
}

func (s *Map[K, V]) IsEmpty() bool {
	s.RLock()
	defer s.RUnlock()
	return len(s.items) == 0
}

func (s *Map[K, V]) Put(k K, v V) {
	s.Lock()
	defer s.Unlock()
	s.items[k] = v
	if s.gauge != nil {
		(*s.gauge).Inc(1)
	}
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
	if s.gauge != nil {
		(*s.gauge).Dec(1)
	}
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
