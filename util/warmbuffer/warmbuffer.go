// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package warmbuffer allocates in-memory data structures and faults every page up front so the OS backs them with real
// RAM immediately, rather than lazily on first write.
//
// This matters in a container or Kubernetes environment: the pod runs under a cgroup memory limit, and Linux charges
// anonymous pages against that limit only when they are first written, not when make reserves the address space.
// Without faulting the pages up front, a process could start on a pod that cannot actually back the full allocation and
// then be killed by the cgroup OOM killer later, when a write faults the pages in while the pod is already under load.
// Touching every page now forces that accounting at startup, so an under-provisioned pod fails fast at deploy time
// instead of mid-operation, and the committed pages stay resident (pods usually run without swap) and are reused for
// the process lifetime.
package warmbuffer

import (
	"os"
	"unsafe"
)

// MakeWarmArray allocates a []T of length n and faults its backing memory by writing one element per OS page.
func MakeWarmArray[T any](n int) []T {
	a := make([]T, n)
	var zero T
	elemSize := int(unsafe.Sizeof(zero))
	if elemSize == 0 {
		return a // zero-size elements occupy no memory
	}
	stride := max(os.Getpagesize()/elemSize, 1)
	for i := 0; i < len(a); i += stride {
		a[i] = zero
	}
	return a
}

// MakeWarmBuffer allocates a byte buffer of the given size with every page faulted.
func MakeWarmBuffer(size int) []byte {
	return MakeWarmArray[byte](size)
}

// MakeWarmMap allocates a map presized for capacity entries, inserts that many distinct keys to fault and commit its
// bucket memory, then clears it. The builtin clear empties the map while keeping the buckets allocated, so the returned
// map is empty, usable, and already backed by committed memory. nextKey must return a distinct key on each call.
func MakeWarmMap[K comparable, V any](capacity int, nextKey func() K) map[K]V {
	m := make(map[K]V, capacity)
	var zero V
	for range capacity {
		m[nextKey()] = zero
	}
	clear(m)
	return m
}
