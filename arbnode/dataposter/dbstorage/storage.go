// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dbstorage

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/util/dbutil"
)

// Storage implements db based storage for batch poster.
type Storage struct {
	db     ethdb.Database
	encDec storage.EncoderDecoderF
}

var (
	// Value at this index holds the *index* of last item.
	// Keys that we never want to be accidentally deleted by "Prune()" should be
	// lexicographically less than minimum index (that is "0"), hence the prefix
	// ".".
	lastItemIdxKey = []byte(".last_item_idx_key")
	countKey       = []byte(".count_key")
)

func New(db ethdb.Database, enc storage.EncoderDecoderF) *Storage {
	return &Storage{db: db, encDec: enc}
}

func idxToKey(idx uint64) []byte {
	return []byte(fmt.Sprintf("%020d", idx))
}

func (s *Storage) FetchContents(_ context.Context, startingIndex uint64, maxResults uint64) ([]*storage.QueuedTransaction, error) {
	var res []*storage.QueuedTransaction
	it := s.db.NewIterator([]byte(""), idxToKey(startingIndex))
	defer it.Release()
	for i := uint64(0); i < maxResults; i++ {
		if !it.Next() {
			break
		}
		item, err := s.encDec().Decode(it.Value())
		if err != nil {
			return nil, err
		}
		res = append(res, item)
	}
	return res, it.Error()
}

func (s *Storage) Get(_ context.Context, index uint64) (*storage.QueuedTransaction, error) {
	key := idxToKey(index)
	value, err := s.db.Get(key)
	if err != nil {
		if dbutil.IsErrNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return s.encDec().Decode(value)
}

func (s *Storage) lastItemIdx(context.Context) ([]byte, error) {
	return s.db.Get(lastItemIdxKey)
}

func (s *Storage) FetchLast(ctx context.Context) (*storage.QueuedTransaction, error) {
	size, err := s.Length(ctx)
	if err != nil {
		return nil, err
	}
	if size == 0 {
		return nil, nil
	}
	lastItemIdx, err := s.lastItemIdx(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting last item index: %w", err)
	}
	val, err := s.db.Get(lastItemIdx)
	if err != nil {
		return nil, err
	}
	return s.encDec().Decode(val)
}

func (s *Storage) PruneAll(ctx context.Context) error {
	idx, err := s.lastItemIdx(ctx)
	if err != nil {
		return fmt.Errorf("pruning all keys: %w", err)
	}
	until, err := strconv.ParseUint(string(idx), 10, 64)
	if err != nil {
		return fmt.Errorf("converting last item index bytes to integer: %w", err)
	}
	return s.Prune(ctx, until+1)
}

func (s *Storage) Prune(ctx context.Context, until uint64) error {
	cnt, err := s.Length(ctx)
	if err != nil {
		return err
	}
	end := idxToKey(until)
	it := s.db.NewIterator([]byte{}, idxToKey(0))
	defer it.Release()
	b := s.db.NewBatch()
	for it.Next() {
		if bytes.Compare(it.Key(), end) >= 0 {
			break
		}
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
func (s *Storage) valueAt(_ context.Context, key []byte) ([]byte, error) {
	val, err := s.db.Get(key)
	if err != nil {
		if dbutil.IsErrNotFound(err) {
			return s.encDec().Encode((*storage.QueuedTransaction)(nil))
		}
		return nil, err
	}
	return val, nil
}

func (s *Storage) Put(ctx context.Context, index uint64, prev, new *storage.QueuedTransaction) error {
	key := idxToKey(index)
	stored, err := s.valueAt(ctx, key)
	if err != nil {
		return err
	}
	prevEnc, err := s.encDec().Encode(prev)
	if err != nil {
		return fmt.Errorf("encoding previous item: %w", err)
	}
	if !bytes.Equal(stored, prevEnc) {
		return fmt.Errorf("replacing different item than expected at index: %v, stored: %v, prevEnc: %v", index, hex.EncodeToString(stored), hex.EncodeToString(prevEnc))
	}
	newEnc, err := s.encDec().Encode(new)
	if err != nil {
		return fmt.Errorf("encoding new item: %w", err)
	}
	b := s.db.NewBatch()
	cnt, err := s.Length(ctx)
	if err != nil {
		return err
	}
	if err := b.Put(key, newEnc); err != nil {
		return fmt.Errorf("updating value at: %v: %w", key, err)
	}
	lastItemIdx, err := s.lastItemIdx(ctx)
	if err != nil && !dbutil.IsErrNotFound(err) {
		return err
	}
	if dbutil.IsErrNotFound(err) {
		lastItemIdx = []byte{}
	}
	if cnt == 0 || bytes.Compare(key, lastItemIdx) > 0 {
		if err := b.Put(lastItemIdxKey, key); err != nil {
			return fmt.Errorf("updating last item: %w", err)
		}
		if err := b.Put(countKey, []byte(strconv.Itoa(cnt+1))); err != nil {
			return fmt.Errorf("updating length counter: %w", err)
		}
	}
	return b.Write()
}

func (s *Storage) Length(context.Context) (int, error) {
	val, err := s.db.Get(countKey)
	if err != nil {
		if dbutil.IsErrNotFound(err) {
			return 0, nil
		}
		return 0, err
	}
	return strconv.Atoi(string(val))
}

func (s *Storage) IsPersistent() bool {
	return true
}
