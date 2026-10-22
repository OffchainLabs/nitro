// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package containers

import "sync/atomic"

type AtomicEnum[T ~int32] struct {
	value atomic.Int32
}

func (a *AtomicEnum[T]) Load() T {
	return T(a.value.Load())
}

func (a *AtomicEnum[T]) Store(value T) {
	a.value.Store(int32(value))
}

func (a *AtomicEnum[T]) Swap(value T) T {
	return T(a.value.Swap(int32(value)))
}
