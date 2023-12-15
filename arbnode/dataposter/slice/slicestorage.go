// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package slice

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
)

type Storage struct {
	firstNonce uint64
	queue      [][]byte
	encDec     func() storage.EncoderDecoderInterface
}

func NewStorage(encDec func() storage.EncoderDecoderInterface) *Storage {
	return &Storage{encDec: encDec}
}

func (s *Storage) FetchContents(_ context.Context, startingIndex uint64, maxResults uint64) ([]*storage.QueuedTransaction, error) {
	txs := s.queue
	if startingIndex >= s.firstNonce+uint64(len(s.queue)) || maxResults == 0 {
		return nil, nil
	}
	if startingIndex > s.firstNonce {
		txs = txs[startingIndex-s.firstNonce:]
	}
	if uint64(len(txs)) > maxResults {
		txs = txs[:maxResults]
	}
	var res []*storage.QueuedTransaction
	for _, r := range txs {
		item, err := s.encDec().Decode(r)
		if err != nil {
			return nil, err
		}
		res = append(res, item)
	}
	return res, nil
}

func (s *Storage) FetchLast(context.Context) (*storage.QueuedTransaction, error) {
	if len(s.queue) == 0 {
		return nil, nil
	}
	return s.encDec().Decode(s.queue[len(s.queue)-1])
}

func (s *Storage) Prune(_ context.Context, until uint64) error {
	if until >= s.firstNonce+uint64(len(s.queue)) {
		s.queue = nil
	} else if until >= s.firstNonce {
		s.queue = s.queue[until-s.firstNonce:]
		s.firstNonce = until
	}
	return nil
}

func (s *Storage) Put(_ context.Context, index uint64, prev, new *storage.QueuedTransaction) error {
	if new == nil {
		return fmt.Errorf("tried to insert nil item at index %v", index)
	}
	newEnc, err := s.encDec().Encode(new)
	if err != nil {
		return fmt.Errorf("encoding new item: %w", err)
	}
	if len(s.queue) == 0 {
		if prev != nil {
			return errors.New("prevItem isn't nil but queue is empty")
		}
		s.queue = append(s.queue, newEnc)
		s.firstNonce = index
	} else if index == s.firstNonce+uint64(len(s.queue)) {
		if prev != nil {
			return errors.New("prevItem isn't nil but item is just after end of queue")
		}
		s.queue = append(s.queue, newEnc)
	} else if index >= s.firstNonce {
		queueIdx := int(index - s.firstNonce)
		if queueIdx > len(s.queue) {
			return fmt.Errorf("attempted to set out-of-bounds index %v in queue starting at %v of length %v", index, s.firstNonce, len(s.queue))
		}
		prevEnc, err := s.encDec().Encode(prev)
		if err != nil {
			return fmt.Errorf("encoding previous item: %w", err)
		}
		if !bytes.Equal(prevEnc, s.queue[queueIdx]) {
			return fmt.Errorf("replacing different item than expected at index: %v, stored: %v, prevEnc: %v", index, hex.EncodeToString(s.queue[queueIdx]), hex.EncodeToString(prevEnc))
		}
		s.queue[queueIdx] = newEnc
	} else {
		return fmt.Errorf("attempted to set too low index %v in queue starting at %v", index, s.firstNonce)
	}
	return nil
}

func (s *Storage) Length(context.Context) (int, error) {
	return len(s.queue), nil
}

func (s *Storage) IsPersistent() bool {
	return false
}
