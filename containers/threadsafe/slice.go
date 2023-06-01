package threadsafe

import "sync"

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
