// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dataposter

import (
	"context"
	"errors"
	"fmt"
)

// SliceStorage is a queue storing pointers to generic types. `firstNonce` is a
// virtual index that is used by the API to access the underlying queue.
type SliceStorage[Item any] struct {
	firstNonce uint64
	queue      []*Item
}

// NewSliceStorage is a factory method to construct a SliceStorage.
func NewSliceStorage[Item any]() *SliceStorage[Item] {
	return &SliceStorage[Item]{}
}

// Get Contents fetches items from `startingindex`.
//
// There are two possible paths:
// 1) `startingIndex` > length queue + firstNonce => nil is returned (invalid)
// 2) `startingIndex` > `firstNonce` => return the underlying queue sliced based on the offset (valid)
//
// In either case, if the length of the result exceeds `maxResults`, the slice
// is trimmed before returning.
func (s *SliceStorage[Item]) GetContents(ctx context.Context, startingIndex uint64, maxResults uint64) ([]*Item, error) {
	ret := s.queue
	// Invalid because startIndex exceeds the valid portion of the slice.
	if startingIndex >= s.firstNonce+uint64(len(s.queue)) {
		ret = nil
	} else if startingIndex > s.firstNonce {
		// Slice based on the offset of `firstNonce`.
		ret = ret[startingIndex-s.firstNonce:]
	}
	if uint64(len(ret)) > maxResults {
		ret = ret[:maxResults]
	}
	return ret, nil
}

// GetLast returns the last item in the queue or nil if the queue is empty.
func (s *SliceStorage[Item]) GetLast(ctx context.Context) (*Item, error) {
	if len(s.queue) > 0 {
		return s.queue[len(s.queue)-1], nil
	}
	return nil, nil
}

// Prune trims `keepStartingAt - firstNonce` items from the front of the queue.
func (s *SliceStorage[Item]) Prune(ctx context.Context, keepStartingAt uint64) error {
	if keepStartingAt >= s.firstNonce+uint64(len(s.queue)) {
		s.queue = nil
	} else if keepStartingAt >= s.firstNonce {
		s.queue = s.queue[keepStartingAt-s.firstNonce:]
		s.firstNonce = keepStartingAt
	}
	return nil
}

// Put inserts an element into a index offset by the `firstNonce`. If it
// overwrites an existing value, it requires that you pass in a pointer to the
// item that is getting overwritten. If you are just adding to the queue,
// prevItem must be nil.
//
// There are a 3 successful paths through this:
// 1) the queue is empty => append the item and initialize the `firstNonce`.
// 2) you are adding to the end of the queue => append the item.
// 3) you are overwritting an existing value => replace the item at index.
func (s *SliceStorage[Item]) Put(ctx context.Context, index uint64, prevItem *Item, newItem *Item) error {
	if newItem == nil {
		return fmt.Errorf("tried to insert nil item at index %v", index)
	}
	if len(s.queue) == 0 {
		if prevItem != nil {
			return errors.New("prevItem isn't nil but queue is empty")
		}
		s.queue = append(s.queue, newItem)
		// Initialize the nonce index.
		s.firstNonce = index
	} else if index == s.firstNonce+uint64(len(s.queue)) {
		if prevItem != nil {
			return errors.New("prevItem isn't nil but item is just after end of queue")
		}
		// Add item to end of the list.
		s.queue = append(s.queue, newItem)
	} else if index >= s.firstNonce {
		queueIdx := int(index - s.firstNonce)
		// Confirm that the queue index is valid.
		if queueIdx > len(s.queue) {
			return fmt.Errorf("attempted to set out-of-bounds index %v in queue starting at %v of length %v", index, s.firstNonce, len(s.queue))
		}
		// Assert that the prevItem that was passed matches what is in the queue.
		if prevItem != s.queue[queueIdx] {
			return fmt.Errorf("prevItem %v doesn't equal existing queue item %v", *prevItem, *s.queue[queueIdx])
		}
		// Reassign the item at queueIdx.
		s.queue[queueIdx] = newItem
	} else { // index < s.firstNonce.
		return fmt.Errorf("attempted to set too low index %v in queue starting at %v", index, s.firstNonce)
	}
	return nil
}
