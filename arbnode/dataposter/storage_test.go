// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dataposter

import (
	"context"
	"math/big"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/offchainlabs/nitro/arbnode/dataposter/dbstorage"
	"github.com/offchainlabs/nitro/arbnode/dataposter/redis"
	"github.com/offchainlabs/nitro/arbnode/dataposter/slice"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/signature"
)

var ignoreData = cmp.Options{
	cmpopts.IgnoreUnexported(
		types.Transaction{},
		types.DynamicFeeTx{},
		big.Int{},
	),
	cmpopts.IgnoreFields(types.Transaction{}, "hash", "size", "from"),
}

func newLevelDBStorage(t *testing.T, encF storage.EncoderDecoderF) *dbstorage.Storage {
	t.Helper()
	db, err := rawdb.NewLevelDBDatabase(path.Join(t.TempDir(), "level.db"), 0, 0, "default", false)
	if err != nil {
		t.Fatalf("NewLevelDBDatabase() unexpected error: %v", err)
	}
	return dbstorage.New(db, encF)
}

func newPebbleDBStorage(t *testing.T, encF storage.EncoderDecoderF) *dbstorage.Storage {
	t.Helper()
	db, err := rawdb.NewPebbleDBDatabase(path.Join(t.TempDir(), "pebble.db"), 0, 0, "default", false)
	if err != nil {
		t.Fatalf("NewPebbleDBDatabase() unexpected error: %v", err)
	}
	return dbstorage.New(db, encF)
}

func newSliceStorage(encF storage.EncoderDecoderF) *slice.Storage {
	return slice.NewStorage(encF)
}

func newRedisStorage(ctx context.Context, t *testing.T, encF storage.EncoderDecoderF) *redis.Storage {
	t.Helper()
	redisUrl := redisutil.CreateTestRedis(ctx, t)
	client, err := redisutil.RedisClientFromURL(redisUrl)
	if err != nil {
		t.Fatalf("RedisClientFromURL(%q) unexpected error: %v", redisUrl, err)
	}
	s, err := redis.NewStorage(client, "", &signature.TestSimpleHmacConfig, encF)
	if err != nil {
		t.Fatalf("redis.NewStorage() unexpected error: %v", err)
	}
	return s
}

func valueOf(t *testing.T, i int) *storage.QueuedTransaction {
	t.Helper()
	meta, err := rlp.EncodeToBytes(storage.BatchPosterPosition{DelayedMessageCount: uint64(i)})
	if err != nil {
		t.Fatalf("Encoding batch poster position, error: %v", err)
	}
	return &storage.QueuedTransaction{
		FullTx: types.NewTransaction(
			uint64(i),
			common.Address{},
			big.NewInt(int64(i)),
			uint64(i),
			big.NewInt(int64(i)),
			[]byte{byte(i)}),
		Meta: meta,
		Data: types.DynamicFeeTx{
			ChainID:    big.NewInt(int64(i)),
			Nonce:      uint64(i),
			GasTipCap:  big.NewInt(int64(i)),
			GasFeeCap:  big.NewInt(int64(i)),
			Gas:        uint64(i),
			Value:      big.NewInt(int64(i)),
			Data:       []byte{byte(i % 8)},
			AccessList: types.AccessList{},
			V:          big.NewInt(int64(i)),
			R:          big.NewInt(int64(i)),
			S:          big.NewInt(int64(i)),
		},
	}
}

func values(t *testing.T, from, to int) []*storage.QueuedTransaction {
	var res []*storage.QueuedTransaction
	for i := from; i <= to; i++ {
		res = append(res, valueOf(t, i))
	}
	return res
}

// Initializes the QueueStorage. Returns the same object (for convenience).
func initStorage(ctx context.Context, t *testing.T, s QueueStorage) QueueStorage {
	t.Helper()
	for i := 0; i < 20; i++ {
		if err := s.Put(ctx, uint64(i), nil, valueOf(t, i)); err != nil {
			t.Fatalf("Error putting a key/value: %v", err)
		}
	}
	return s
}

// Returns a map of all empty storages.
func storages(t *testing.T) map[string]QueueStorage {
	t.Helper()
	f := func(enc storage.EncoderDecoderInterface) storage.EncoderDecoderF {
		return func() storage.EncoderDecoderInterface {
			return enc
		}
	}
	return map[string]QueueStorage{
		"levelDBLegacy": newLevelDBStorage(t, f(&storage.LegacyEncoderDecoder{})),
		"sliceLegacy":   newSliceStorage(f(&storage.LegacyEncoderDecoder{})),
		"redisLegacy":   newRedisStorage(context.Background(), t, f(&storage.LegacyEncoderDecoder{})),
		"levelDB":       newLevelDBStorage(t, f(&storage.EncoderDecoder{})),
		"pebbleDB":      newPebbleDBStorage(t, f(&storage.EncoderDecoder{})),
		"slice":         newSliceStorage(f(&storage.EncoderDecoder{})),
		"redis":         newRedisStorage(context.Background(), t, f(&storage.EncoderDecoder{})),
	}
}

// Returns a map of all initialized storages.
func initStorages(ctx context.Context, t *testing.T) map[string]QueueStorage {
	t.Helper()
	m := map[string]QueueStorage{}
	for k, v := range storages(t) {
		m[k] = initStorage(ctx, t, v)
	}
	return m
}

func TestPruneAll(t *testing.T) {
	s := newLevelDBStorage(t, func() storage.EncoderDecoderInterface { return &storage.EncoderDecoder{} })
	ctx := context.Background()
	for i := 0; i < 20; i++ {
		if err := s.Put(ctx, uint64(i), nil, valueOf(t, i)); err != nil {
			t.Fatalf("Error putting a key/value: %v", err)
		}
	}
	size, err := s.Length(ctx)
	if err != nil {
		t.Fatalf("Length() unexpected error %v", err)
	}
	if size != 20 {
		t.Errorf("Length()=%v want 20", size)
	}
	if err := s.PruneAll(ctx); err != nil {
		t.Fatalf("PruneAll() unexpected error: %v", err)
	}
	size, err = s.Length(ctx)
	if err != nil {
		t.Fatalf("Length() unexpected error %v", err)
	}
	if size != 0 {
		t.Errorf("Length()=%v want 0", size)
	}
}

func TestFetchContents(t *testing.T) {
	ctx := context.Background()
	for name, s := range initStorages(ctx, t) {
		for _, tc := range []struct {
			desc       string
			startIdx   uint64
			maxResults uint64
			want       []*storage.QueuedTransaction
		}{
			{
				desc:       "sequence with single digits",
				startIdx:   5,
				maxResults: 3,
				want:       values(t, 5, 7),
			},
			{
				desc:       "corner case of single element",
				startIdx:   0,
				maxResults: 1,
				want:       values(t, 0, 0),
			},
			{
				desc:       "no elements",
				startIdx:   3,
				maxResults: 0,
			},
			{
				// Making sure it's correctly ordered lexicographically.
				desc:       "sequence with variable number of digits",
				startIdx:   9,
				maxResults: 3,
				want:       values(t, 9, 11),
			},
			{
				desc:       "max results goes over the last element",
				startIdx:   13,
				maxResults: 10,
				want:       values(t, 13, 19),
			},
		} {
			t.Run(name+"_"+tc.desc, func(t *testing.T) {
				values, err := s.FetchContents(ctx, tc.startIdx, tc.maxResults)
				if err != nil {
					t.Fatalf("FetchContents(%d, %d) unexpected error: %v", tc.startIdx, tc.maxResults, err)
				}
				if diff := cmp.Diff(tc.want, values, ignoreData); diff != "" {
					t.Errorf("FetchContents(%d, %d) unexpected diff:\n%s", tc.startIdx, tc.maxResults, diff)
				}
			})
		}
	}
}

func TestLast(t *testing.T) {
	cnt := 100
	for name, s := range storages(t) {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			for i := 0; i < cnt; i++ {
				val := valueOf(t, i)
				if err := s.Put(ctx, uint64(i), nil, val); err != nil {
					t.Fatalf("Error putting a key/value: %v", err)
				}
				got, err := s.FetchLast(ctx)
				if err != nil {
					t.Fatalf("Error getting a last element: %v", err)
				}
				if diff := cmp.Diff(val, got, ignoreData); diff != "" {
					t.Errorf("FetchLast() unexpected diff:\n%s", diff)
				}

			}
		})
		last := valueOf(t, cnt-1)
		t.Run(name+"_update_entries", func(t *testing.T) {
			ctx := context.Background()
			for i := 0; i < cnt-1; i++ {
				prev := valueOf(t, i)
				newVal := valueOf(t, cnt+i)
				if err := s.Put(ctx, uint64(i), prev, newVal); err != nil {
					t.Fatalf("Error putting a key/value: %v, prev: %v, new: %v", err, prev, newVal)
				}
				got, err := s.FetchLast(ctx)
				if err != nil {
					t.Fatalf("Error getting a last element: %v", err)
				}
				if diff := cmp.Diff(last, got, ignoreData); diff != "" {
					t.Errorf("FetchLast() unexpected diff:\n%s", diff)
				}
				gotCnt, err := s.Length(ctx)
				if err != nil {
					t.Fatalf("Length() unexpected error: %v", err)
				}
				if gotCnt != cnt {
					t.Errorf("Length() = %d want %d", gotCnt, cnt)
				}
			}
		})
	}
}

func TestPrune(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		desc      string
		pruneFrom uint64
		want      []*storage.QueuedTransaction
	}{
		{
			desc:      "prune all elements",
			pruneFrom: 20,
		},
		{
			desc:      "prune all but one",
			pruneFrom: 19,
			want:      values(t, 19, 19),
		},
		{
			desc:      "pruning first element",
			pruneFrom: 1,
			want:      values(t, 1, 19),
		},
		{
			desc:      "pruning first 11 elements",
			pruneFrom: 11,
			want:      values(t, 11, 19),
		},
		{
			desc:      "pruning from higher than biggest index",
			pruneFrom: 30,
		},
	} {
		// Storages must be re-initialized in each test-case.
		for name, s := range initStorages(ctx, t) {
			t.Run(name+"_"+tc.desc, func(t *testing.T) {
				if err := s.Prune(ctx, tc.pruneFrom); err != nil {
					t.Fatalf("Prune(%d) unexpected error: %v", tc.pruneFrom, err)
				}
				got, err := s.FetchContents(ctx, 0, 20)
				if err != nil {
					t.Fatalf("FetchContents() unexpected error: %v", err)
				}
				if diff := cmp.Diff(tc.want, got, ignoreData); diff != "" {
					t.Errorf("Prune(%d) unexpected diff:\n%s", tc.pruneFrom, diff)
				}
			})
		}
	}
}

func TestLength(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		desc      string
		pruneFrom uint64
	}{
		{
			desc: "not prune any elements",
		},
		{
			desc:      "prune all but one",
			pruneFrom: 19,
		},
		{
			desc:      "pruning first element",
			pruneFrom: 1,
		},
		{
			desc:      "pruning first 11 elements",
			pruneFrom: 11,
		},
		{
			desc:      "pruning from higher than biggest index",
			pruneFrom: 30,
		},
	} {
		// Storages must be re-initialized in each test-case.
		for name, s := range initStorages(ctx, t) {
			t.Run(name+"_"+tc.desc, func(t *testing.T) {
				if err := s.Prune(ctx, tc.pruneFrom); err != nil {
					t.Fatalf("Prune(%d) unexpected error: %v", tc.pruneFrom, err)
				}
				got, err := s.Length(ctx)
				if err != nil {
					t.Fatalf("Length() unexpected error: %v", err)
				}
				if want := arbmath.MaxInt(0, 20-int(tc.pruneFrom)); got != want {
					t.Errorf("Length() = %d want %d", got, want)
				}
			})
		}

	}
}
