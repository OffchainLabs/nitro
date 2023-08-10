// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package slice

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

type Storage[Item any] struct {
	firstNonce uint64
	queue      []*Item
}

func NewStorage[Item any]() *Storage[Item] {
	return &Storage[Item]{}
}

func (s *Storage[Item]) FetchContents(_ context.Context, startingIndex uint64, maxResults uint64) ([]*Item, error) {
	ret := s.queue
	if startingIndex >= s.firstNonce+uint64(len(s.queue)) || maxResults == 0 {
		return nil, nil
	}
	if startingIndex > s.firstNonce {
		ret = ret[startingIndex-s.firstNonce:]
	}
	if uint64(len(ret)) > maxResults {
		ret = ret[:maxResults]
	}
	return ret, nil
}

func (s *Storage[Item]) FetchLast(context.Context) (*Item, error) {
	if len(s.queue) == 0 {
		return nil, nil
	}
	return s.queue[len(s.queue)-1], nil
}

func (s *Storage[Item]) Prune(_ context.Context, until uint64) error {
	if until >= s.firstNonce+uint64(len(s.queue)) {
		s.queue = nil
	} else if until >= s.firstNonce {
		s.queue = s.queue[until-s.firstNonce:]
		s.firstNonce = until
	}
	return nil
}

func (s *Storage[Item]) Put(_ context.Context, index uint64, prevItem *Item, newItem *Item) error {
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
		if !reflect.DeepEqual(prevItem, s.queue[queueIdx]) {
			return fmt.Errorf("replacing different item than expected at index: %v: %v %v", index, prevItem, s.queue[queueIdx])
		}
		s.queue[queueIdx] = newItem
	} else {
		return fmt.Errorf("attempted to set too low index %v in queue starting at %v", index, s.firstNonce)
	}
	return nil
}

func (s *Storage[Item]) Length(context.Context) (int, error) {
	return len(s.queue), nil
}

func (s *Storage[Item]) IsPersistent() bool {
	return false
}
