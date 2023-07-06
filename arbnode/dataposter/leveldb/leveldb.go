// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package leveldb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/syndtr/goleveldb/leveldb"
)

// Storage implements leveldb based storage for batch poster.
type Storage[Item any] struct {

	// Lock is used for using iterator and WriteBatch.
	// https://fuchsia.googlesource.com/third_party/leveldb/+/HEAD/doc/index.md#concurrency
	// Not necessary since there should be a single batchposter active at any
	// point in time.
	lock sync.Mutex
	db   ethdb.Database
}

var (
	// Keys that we never want to be accidentally deleted by "Prune()" should be
	// lexicographically less than minimum index (that is "0"), hence the prefix
	// ".".
	lastItemKey = []byte(".last_item_key")
	countKey    = []byte(".count_key")
)

const DataPosterPrefix string = "d" // the prefix for all data poster keys

func New[Item any](db ethdb.Database) *Storage[Item] {
	return &Storage[Item]{db: db}
}

func (s *Storage[Item]) decodeItem(data []byte) (*Item, error) {
	var item Item
	if err := rlp.DecodeBytes(data, &item); err != nil {
		return nil, fmt.Errorf("decoding item: %w", err)
	}
	return &item, nil
}

func idxToKey(idx uint64) []byte {
	return []byte(fmt.Sprintf("%019d", idx))
}

func (s *Storage[Item]) GetContents(_ context.Context, startingIndex uint64, maxResults uint64) ([]*Item, error) {

	s.lock.Lock()
	defer s.lock.Unlock()

	var res []*Item
	it := s.db.NewIterator([]byte(""), idxToKey(startingIndex))
	for i := 0; i < int(maxResults); i++ {
		if !it.Next() {
			break
		}
		item, err := s.decodeItem(it.Value())
		if err != nil {
			return nil, err
		}
		res = append(res, item)
	}
	return res, nil
}

func (s *Storage[Item]) GetLast(ctx context.Context) (*Item, error) {
	val, err := s.db.Get(lastItemKey)
	if err != nil {
		return nil, err
	}
	return s.decodeItem(val)
}

func (s *Storage[Item]) Prune(ctx context.Context, keepStartingAt uint64) error {
	cnt, err := s.Length(ctx)
	if err != nil {
		return err
	}
	it := s.db.NewIterator([]byte{}, idxToKey(keepStartingAt))
	b := s.db.NewBatch()
	for it.Next() {
		if err := b.Delete(it.Key()); err != nil {
			return fmt.Errorf("deleting key: %w", err)
		}
		cnt--
	}
	if err := b.Put(countKey, []byte(strconv.Itoa(cnt))); err != nil {
		return fmt.Errorf("updating length counter: %w", err)
	}
	return b.Write()
}

// valueAt gets returns the value at key. If it doesn't exist then it returns
// encoded bytes of nil.
func (s *Storage[Item]) valueAt(ctx context.Context, key []byte) ([]byte, error) {
	val, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return rlp.EncodeToBytes((*Item)(nil))
		}
		return nil, err
	}
	return val, nil
}

func (s *Storage[Item]) Put(ctx context.Context, index uint64, prev *Item, new *Item) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	key := idxToKey(index)
	stored, err := s.valueAt(ctx, key)
	if err != nil {
		return err
	}
	prevEnc, err := rlp.EncodeToBytes(prev)
	if err != nil {
		return fmt.Errorf("encoding previous item: %w", err)
	}
	if !bytes.Equal(stored, prevEnc) {
		return fmt.Errorf("replacing different item than expected at index %v %v %v", index, stored, prevEnc)
	}
	newEnc, err := rlp.EncodeToBytes(new)
	if err != nil {
		return fmt.Errorf("encoding new item: %w", err)
	}
	b := s.db.NewBatch()
	cnt, err := s.Length(ctx)
	if err != nil {
		return err
	}
	if err := b.Put(key, newEnc); err != nil {
		return fmt.Errorf("updating value at: %v:  %w", key, err)
	}
	if err := b.Put(lastItemKey, newEnc); err != nil {
		return fmt.Errorf("updating last item: %w", err)
	}
	if err := b.Put(countKey, []byte(strconv.Itoa(cnt+1))); err != nil {
		return fmt.Errorf("updating length counter: %w", err)
	}

	return b.Write()
}

func (s *Storage[Item]) Length(ctx context.Context) (int, error) {
	val, err := s.db.Get(countKey)
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return strconv.Atoi(string(val))
}

func (s *Storage[Item]) IsPersistent() bool {
	return true
}
