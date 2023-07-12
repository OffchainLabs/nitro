package dataposter

import (
	"context"
	"path"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/google/go-cmp/cmp"
	"github.com/offchainlabs/nitro/arbnode/dataposter/leveldb"
	"github.com/offchainlabs/nitro/arbnode/dataposter/redis"
	"github.com/offchainlabs/nitro/arbnode/dataposter/slice"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/signature"
)

func newLevelDBStorage[Item any](t *testing.T) *leveldb.Storage[Item] {
	t.Helper()
	db, err := rawdb.NewLevelDBDatabase(path.Join(t.TempDir(), "level.db"), 0, 0, "default", false)
	if err != nil {
		t.Fatalf("NewLevelDBDatabase() unexpected error: %v", err)
	}
	return leveldb.New[Item](db)
}

func newSliceStorage[Item any]() *slice.Storage[Item] {
	return slice.NewStorage[Item]()
}

func newRedisStorage[Item any](ctx context.Context, t *testing.T) *redis.Storage[Item] {
	t.Helper()
	redisUrl := redisutil.CreateTestRedis(ctx, t)
	client, err := redisutil.RedisClientFromURL(redisUrl)
	if err != nil {
		t.Fatalf("RedisClientFromURL(%q) unexpected error: %v", redisUrl, err)
	}
	s, err := redis.NewStorage[Item](client, "", &signature.TestSimpleHmacConfig)
	if err != nil {
		t.Fatalf("redis.NewStorage() unexpected error: %v", err)
	}
	return s
}

// Initializes the QueueStorage. Returns the same object (for convenience).
func initStorage(ctx context.Context, t *testing.T, s QueueStorage[string]) QueueStorage[string] {
	t.Helper()
	for i := 0; i < 20; i++ {
		val := strconv.Itoa(i)
		if err := s.Put(ctx, uint64(i), nil, &val); err != nil {
			t.Fatalf("Error putting a key/value: %v", err)
		}
	}
	return s
}

// Returns a map of all empty storages.
func storages(t *testing.T) map[string]QueueStorage[string] {
	t.Helper()
	return map[string]QueueStorage[string]{
		"levelDB": newLevelDBStorage[string](t),
		"slice":   newSliceStorage[string](),
		"redis":   newRedisStorage[string](context.Background(), t),
	}
}

// Returns a map of all initialized storages.
func initStorages(ctx context.Context, t *testing.T) map[string]QueueStorage[string] {
	t.Helper()
	m := map[string]QueueStorage[string]{}
	for k, v := range storages(t) {
		m[k] = initStorage(ctx, t, v)
	}
	return m
}

func strPtrs(values []string) []*string {
	var res []*string
	for _, val := range values {
		v := val
		res = append(res, &v)
	}
	return res
}

func TestGetContents(t *testing.T) {
	ctx := context.Background()
	for name, s := range initStorages(ctx, t) {
		for _, tc := range []struct {
			desc       string
			startIdx   uint64
			maxResults uint64
			want       []*string
		}{
			{
				desc:       "sequence with single digits",
				startIdx:   5,
				maxResults: 3,
				want:       strPtrs([]string{"5", "6", "7"}),
			},
			{
				desc:       "corner case of single element",
				startIdx:   0,
				maxResults: 1,
				want:       strPtrs([]string{"0"}),
			},
			{
				desc:       "no elements",
				startIdx:   3,
				maxResults: 0,
				want:       strPtrs([]string{}),
			},
			{
				// Making sure it's correctly ordered lexicographically.
				desc:       "sequence with variable number of digits",
				startIdx:   9,
				maxResults: 3,
				want:       strPtrs([]string{"9", "10", "11"}),
			},
			{
				desc:       "max results goes over the last element",
				startIdx:   13,
				maxResults: 10,
				want:       strPtrs([]string{"13", "14", "15", "16", "17", "18", "19"}),
			},
		} {
			t.Run(name+"_"+tc.desc, func(t *testing.T) {
				values, err := s.GetContents(ctx, tc.startIdx, tc.maxResults)
				if err != nil {
					t.Fatalf("GetContents(%d, %d) unexpected error: %v", tc.startIdx, tc.maxResults, err)
				}
				if diff := cmp.Diff(tc.want, values); diff != "" {
					t.Errorf("GetContext(%d, %d) unexpected diff:\n%s", tc.startIdx, tc.maxResults, diff)
				}
			})
		}
	}
}

func TestGetLast(t *testing.T) {
	cnt := 100
	for name, s := range storages(t) {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			for i := 0; i < cnt; i++ {
				val := strconv.Itoa(i)
				if err := s.Put(ctx, uint64(i), nil, &val); err != nil {
					t.Fatalf("Error putting a key/value: %v", err)
				}
				got, err := s.GetLast(ctx)
				if err != nil {
					t.Fatalf("Error getting a last element: %v", err)
				}
				if *got != val {
					t.Errorf("GetLast() = %q want %q", *got, val)
				}

			}
		})
		last := strconv.Itoa(cnt - 1)
		t.Run(name+"_update_entries", func(t *testing.T) {
			ctx := context.Background()
			for i := 0; i < cnt-1; i++ {
				prev := strconv.Itoa(i)
				newVal := strconv.Itoa(cnt + i)
				if err := s.Put(ctx, uint64(i), &prev, &newVal); err != nil {
					t.Fatalf("Error putting a key/value: %v, prev: %v, new: %v", err, prev, newVal)
				}
				got, err := s.GetLast(ctx)
				if err != nil {
					t.Fatalf("Error getting a last element: %v", err)
				}
				if *got != last {
					t.Errorf("GetLast() = %q want %q", *got, last)
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
		want      []*string
	}{
		{
			desc:      "prune all elements",
			pruneFrom: 20,
		},
		{
			desc:      "prune all but one",
			pruneFrom: 19,
			want:      strPtrs([]string{"19"}),
		},
		{
			desc:      "pruning first element",
			pruneFrom: 1,
			want: strPtrs([]string{"1", "2", "3", "4", "5", "6", "7",
				"8", "9", "10", "11", "12", "13", "14", "15", "16", "17", "18", "19"}),
		},
		{
			desc:      "pruning first 11 elements",
			pruneFrom: 11,
			want:      strPtrs([]string{"11", "12", "13", "14", "15", "16", "17", "18", "19"}),
		},
		{
			desc:      "pruning from higher than biggest index",
			pruneFrom: 30,
			want:      strPtrs([]string{}),
		},
	} {
		// Storages must be re-initialized in each test-case.
		for name, s := range initStorages(ctx, t) {
			t.Run(name+"_"+tc.desc, func(t *testing.T) {
				if err := s.Prune(ctx, tc.pruneFrom); err != nil {
					t.Fatalf("Prune(%d) unexpected error: %v", tc.pruneFrom, err)
				}
				got, err := s.GetContents(ctx, 0, 20)
				if err != nil {
					t.Fatalf("GetContents() unexpected error: %v", err)
				}
				if diff := cmp.Diff(tc.want, got); diff != "" {
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
