package leveldb

import (
	"context"
	"path"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/google/go-cmp/cmp"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func newStorage[Item any](t *testing.T) *Storage[Item] {
	t.Helper()
	db, err := rawdb.NewLevelDBDatabase(path.Join(t.TempDir(), "level.db"), 0, 0, "default", false)
	if err != nil {
		t.Fatalf("NewLevelDBDatabase() unexpected error: %v", err)
	}
	return New[Item](db)
}

// Returns storage that is already initialized with some elements.
func newInitStorage(ctx context.Context, t *testing.T) *Storage[string] {
	t.Helper()
	s := newStorage[string](t)
	for i := 0; i < 20; i++ {
		val := strconv.Itoa(i)
		if err := s.Put(ctx, uint64(i), nil, &val); err != nil {
			t.Fatalf("Error putting a key/value: %v", err)
		}
	}
	return s
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
	s := newInitStorage(ctx, t)

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
		t.Run(tc.desc, func(t *testing.T) {
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

func TestGetLast(t *testing.T) {
	s := newStorage[string](t)
	ctx := context.Background()
	for i := 0; i < 100; i++ {
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
			pruneFrom: 0,
		},
		{
			desc:      "prune all but one",
			pruneFrom: 1,
			want:      strPtrs([]string{"0"}),
		},
		{
			desc:      "pruning last element",
			pruneFrom: 19,
			want: strPtrs([]string{"0", "1", "2", "3", "4", "5", "6", "7",
				"8", "9", "10", "11", "12", "13", "14", "15", "16", "17", "18"}),
		},
		{
			desc:      "pruning last 11 elements",
			pruneFrom: 9,
			want: strPtrs([]string{"0", "1", "2", "3", "4", "5", "6", "7",
				"8"}),
		},
		{
			desc:      "pruning from higher than biggest index", // should be no op
			pruneFrom: 30,
			want: strPtrs([]string{"0", "1", "2", "3", "4", "5", "6", "7", "8",
				"9", "10", "11", "12", "13", "14", "15", "16", "17", "18", "19"}),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			s := newInitStorage(ctx, t)
			if err := s.Prune(ctx, tc.pruneFrom); err != nil {
				t.Fatalf("Prune(%d) unexpected error: %v", tc.pruneFrom, err)
			}
			got, err := s.GetContents(ctx, 0, 20)
			if err != nil {
				t.Fatalf("GetContents() unexpected error: %v", err)
			}
			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("Prune(%d) unexpected diff:\n%s", tc.pruneFrom, diff)
			}
		})
	}
}

func TestLength(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		desc      string
		pruneFrom uint64
	}{
		{
			desc: "prune all elements",
		},
		{
			desc:      "prune all but one",
			pruneFrom: 1,
		},
		{
			desc:      "pruning last element",
			pruneFrom: 19,
		},
		{
			desc:      "pruning last 11 elements",
			pruneFrom: 9,
		},
		{
			desc:      "pruning from higher than biggest index", // should be no op
			pruneFrom: 30,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			s := newInitStorage(ctx, t)
			if err := s.Prune(ctx, tc.pruneFrom); err != nil {
				t.Fatalf("Prune(%d) unexpected error: %v", tc.pruneFrom, err)
			}
			got, err := s.Length(ctx)
			if err != nil {
				t.Fatalf("Length() unexpected error: %v", err)
			}
			if want := arbmath.MinInt(20, int(tc.pruneFrom)); got != want {
				t.Errorf("Length() = %d want %d", got, want)
			}
		})
	}
}
