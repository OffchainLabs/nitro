// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dataposter

import (
	"context"
	"errors"
	"fmt"
)

type SliceStorage[Item any] struct {
	firstNonce uint64
	queue      []*Item
}

func NewSliceStorage[Item any]() *SliceStorage[Item] {
	return &SliceStorage[Item]{}
}

func (s *SliceStorage[Item]) GetContents(ctx context.Context, startingIndex uint64, maxResults uint64) ([]*Item, error) {
	ret := s.queue
	if startingIndex >= s.firstNonce+uint64(len(s.queue)) {
		ret = nil
	} else if startingIndex > s.firstNonce {
		ret = ret[startingIndex-s.firstNonce:]
	}
	if uint64(len(ret)) > maxResults {
		ret = ret[:maxResults]
	}
	return ret, nil
}

func (s *SliceStorage[Item]) GetLast(ctx context.Context) (*Item, error) {
	if len(s.queue) > 0 {
		return s.queue[len(s.queue)-1], nil
	} else {
		return nil, nil
	}
}

func (s *SliceStorage[Item]) Prune(ctx context.Context, keepStartingAt uint64) error {
	if keepStartingAt >= s.firstNonce+uint64(len(s.queue)) {
		s.queue = nil
	} else if keepStartingAt >= s.firstNonce {
		s.queue = s.queue[keepStartingAt-s.firstNonce:]
		s.firstNonce = keepStartingAt
	}
	return nil
}

func (s *SliceStorage[Item]) Put(ctx context.Context, index uint64, prevItem *Item, newItem *Item) error {
	if newItem == nil {
		return fmt.Errorf("tried to insert nil item at index %v", index)
	}
	if len(s.queue) == 0 {
		if prevItem != nil {
			return errors.New("prevItem isn't nil but queue is empty")
		}
		s.queue = append(s.queue, newItem)
		s.firstNonce = index
	} else if index == s.firstNonce+uint64(len(s.queue)) {
		if prevItem != nil {
			return errors.New("prevItem isn't nil but item is just after end of queue")
		}
		s.queue = append(s.queue, newItem)
	} else if index >= s.firstNonce {
		queueIdx := int(index - s.firstNonce)
		if queueIdx > len(s.queue) {
			return fmt.Errorf("attempted to set out-of-bounds index %v in queue starting at %v of length %v", index, s.firstNonce, len(s.queue))
		}
		if prevItem != s.queue[queueIdx] {
			return errors.New("prevItem isn't nil but item is just after end of queue")
		}
		s.queue[queueIdx] = newItem
	} else {
		return fmt.Errorf("attempted to set too low index %v in queue starting at %v", index, s.firstNonce)
	}
	return nil
}
