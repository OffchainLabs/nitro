package dataposter

import (
	"context"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/offchainlabs/nitro/arbnode/dataposter/leveldb"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

var ignoreData = cmpopts.IgnoreFields(storage.QueuedTransaction{}, "Data")

func newLevelDBStorage[Item any](t *testing.T) *leveldb.Storage {
	t.Helper()
	db, err := rawdb.NewLevelDBDatabase(path.Join(t.TempDir(), "level.db"), 0, 0, "default", false)
	if err != nil {
		t.Fatalf("NewLevelDBDatabase() unexpected error: %v", err)
	}
	return leveldb.New(db)
}

// func newSliceStorage[Item any]() *slice.Storage[Item] {
// 	return slice.NewStorage[Item]()
// }

// func newRedisStorage[Item any](ctx context.Context, t *testing.T) *redis.Storage[Item] {
// 	t.Helper()
// 	redisUrl := redisutil.CreateTestRedis(ctx, t)
// 	client, err := redisutil.RedisClientFromURL(redisUrl)
// 	if err != nil {
// 		t.Fatalf("RedisClientFromURL(%q) unexpected error: %v", redisUrl, err)
// 	}
// 	s, err := redis.NewStorage[Item](client, "", &signature.TestSimpleHmacConfig)
// 	if err != nil {
// 		t.Fatalf("redis.NewStorage() unexpected error: %v", err)
// 	}
// 	return s
// }

func valueOf(i int) *storage.QueuedTransaction {
	return &storage.QueuedTransaction{
		Meta: []byte{byte(i)},
	}
}

func values(from, to int) []*storage.QueuedTransaction {
	var res []*storage.QueuedTransaction
	for i := from; i <= to; i++ {
		res = append(res, valueOf(i))
	}
	return res
}

// Initializes the QueueStorage. Returns the same object (for convenience).
func initStorage(ctx context.Context, t *testing.T, s QueueStorage) QueueStorage {
	t.Helper()
	for i := 0; i < 20; i++ {
		if err := s.Put(ctx, uint64(i), nil, valueOf(i)); err != nil {
			t.Fatalf("Error putting a key/value: %v", err)
		}
	}
	return s
}

// Returns a map of all empty storages.
func storages(t *testing.T) map[string]QueueStorage {
	t.Helper()
	return map[string]QueueStorage{
		"levelDB": newLevelDBStorage[string](t),
		// "slice":   newSliceStorage[string](),
		// "redis":   newRedisStorage[string](context.Background(), t),
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
				want:       values(5, 7),
			},
			{
				desc:       "corner case of single element",
				startIdx:   0,
				maxResults: 1,
				want:       values(0, 0),
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
				want:       values(9, 11),
			},
			{
				desc:       "max results goes over the last element",
				startIdx:   13,
				maxResults: 10,
				want:       values(13, 19),
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
				val := valueOf(i)
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
		last := valueOf(cnt - 1)
		t.Run(name+"_update_entries", func(t *testing.T) {
			ctx := context.Background()
			for i := 0; i < cnt-1; i++ {
				prev := valueOf(i)
				newVal := valueOf(cnt + i)
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
			want:      values(19, 19),
		},
		{
			desc:      "pruning first element",
			pruneFrom: 1,
			want:      values(1, 19),
		},
		{
			desc:      "pruning first 11 elements",
			pruneFrom: 11,
			want:      values(11, 19),
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
