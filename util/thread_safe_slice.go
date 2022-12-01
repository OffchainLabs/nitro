package util

import "sync"

// ThreadSafeSlice defines a slice with common methods such as length, reading values at an index,
// and empty checks. It uses an RWMutex to protect the underlying slice from concurrent access and
// uses option types in return values.
type ThreadSafeSlice[T any] struct {
	items []T
	lock  sync.RWMutex
}

func NewThreadSafeSlice[T any]() *ThreadSafeSlice[T] {
	return &ThreadSafeSlice[T]{items: make([]T, 0)}
}

func (s *ThreadSafeSlice[T]) Append(item T) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.items = append(s.items, item)
}

func (s *ThreadSafeSlice[T]) Empty() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.items) == 0
}

func (s *ThreadSafeSlice[T]) Len() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.items)
}

func (s *ThreadSafeSlice[T]) Last() Option[T] {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if len(s.items) == 0 {
		return None[T]()
	}
	return Some[T](s.items[len(s.items)-1])
}

func (s *ThreadSafeSlice[T]) Get(i int) Option[T] {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if i >= len(s.items) {
		return None[T]()
	}
	return Some[T](s.items[i])
}
