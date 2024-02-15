// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package threadsafe

import (
	"github.com/ethereum/go-ethereum/common/lru"
	"sync"

	"github.com/ethereum/go-ethereum/metrics"
)

type LruMap[K comparable, V any] struct {
	sync.RWMutex
	items lru.BasicLRU[K, V]
	gauge *metrics.Gauge
}

type LruMapOpt[K comparable, V any] func(*LruMap[K, V])

func LruMapWithMetric[K comparable, V any](name string) LruMapOpt[K, V] {
	return func(m *LruMap[K, V]) {
		gauge := metrics.NewRegisteredGauge("arb/validator/threadsafe_lru_map/"+name, nil)
		m.gauge = &gauge
	}
}

func NewLruMap[K comparable, V any](capacity int, opts ...LruMapOpt[K, V]) *LruMap[K, V] {
	m := &LruMap[K, V]{items: lru.NewBasicLRU[K, V](capacity)}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (s *LruMap[K, V]) IsEmpty() bool {
	s.RLock()
	defer s.RUnlock()
	return s.items.Len() == 0
}

func (s *LruMap[K, V]) Put(k K, v V) {
	s.Lock()
	defer s.Unlock()
	s.items.Add(k, v)
	if s.gauge != nil {
		(*s.gauge).Inc(1)
	}
}

func (s *LruMap[K, V]) Has(k K) bool {
	s.RLock()
	defer s.RUnlock()
	return s.items.Contains(k)
}

func (s *LruMap[K, V]) NumItems() uint64 {
	s.RLock()
	defer s.RUnlock()
	return uint64(s.items.Len())
}

func (s *LruMap[K, V]) TryGet(k K) (V, bool) {
	s.RLock()
	defer s.RUnlock()
	return s.items.Get(k)
}

func (s *LruMap[K, V]) Delete(k K) {
	s.Lock()
	defer s.Unlock()
	s.items.Remove(k)
	if s.gauge != nil {
		(*s.gauge).Dec(1)
	}
}

func (s *LruMap[K, V]) ForEach(fn func(k K, v V) error) error {
	s.RLock()
	defer s.RUnlock()
	for _, k := range s.items.Keys() {
		v, _ := s.items.Get(k)
		if err := fn(k, v); err != nil {
			return err
		}
	}
	return nil
}
