// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dataposter

import (
	"context"
	"testing"
)

func checkSlicesMatch(t *testing.T, valueSlice []int, ptrSlice []*int) {
	t.Helper()
	if len(valueSlice) != len(ptrSlice) {
		t.Errorf("lengths of slices do not match: %v, %v", valueSlice, ptrSlice)
	}
	// Check values are expected.
	for i := 0; i < len(valueSlice); i++ {
		if valueSlice[i] != *ptrSlice[i] {
			t.Errorf("non-matching value; want %v: got %v", valueSlice[i], *ptrSlice[i])
		}
	}
	// Check pointers are expected.
	for i := 0; i < len(valueSlice); i++ {
		if &valueSlice[i] != ptrSlice[i] {
			t.Errorf("non-matching ptr; want %v: got %v", &valueSlice[i], ptrSlice[i])
		}
	}
}

// Helper function to check that the queue contents after the prune match an
// expected slice.
func checkPruneResults(ctx context.Context, t *testing.T, ss *SliceStorage[int], want []int, nonce uint64) {
	t.Helper()
	if len(ss.queue) != len(want) {
		t.Errorf("unexpected length after prune; want: %c, got: %d", len(want), len(ss.queue))
	}

	got, err := ss.GetContents(ctx, nonce, uint64(len(want)))
	if err != nil {
		t.Errorf("unexpected error in GetContents: %v", err)
	}
	checkSlicesMatch(t, want, got)
}

func TestGetContents(t *testing.T) {
	ctx := context.Background()
	const maxLen uint64 = 4
	tests := map[string]struct {
		items []int
		ss    *SliceStorage[int]
		nonce uint64
	}{
		"zero_nonce": {
			items: []int{42, 43, 44},
			ss:    NewSliceStorage[int](),
			nonce: 0,
		},
		"nonzero_nonce": {
			items: []int{252, 253, 254, 255},
			ss:    NewSliceStorage[int](),
			nonce: 9,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Insert items.
			for i := 0; i < len(tc.items); i++ {
				if err := tc.ss.Put(ctx, tc.nonce+uint64(i), nil, &tc.items[i]); err != nil {
					t.Errorf("unable to put item %v: %v", tc.items[i], err)
				}
			}

			// Get full contents (starting at 0 should return full regardless of
			// the firstNonce).
			got, err := tc.ss.GetContents(ctx, 0, maxLen)
			if err != nil {
				t.Errorf("unexpected error in GetContents: %v", err)
			}

			// Check elements are expected.
			checkSlicesMatch(t, tc.items, got)

			// Try to get a too big index should result in nil.
			got, err = tc.ss.GetContents(ctx, tc.nonce+uint64(len(tc.items)), maxLen)
			if err != nil {
				t.Errorf("unexpected error in GetContents: %v", err)
			}
			if got != nil {
				t.Errorf("unexpected non-nil return GetContents: %v", got)
			}

			// Get all but first of the contents.
			got, err = tc.ss.GetContents(ctx, tc.nonce+1, maxLen)
			if err != nil {
				t.Errorf("unexpected error in GetContents: %v", err)
			}

			// Check elements are expected.
			checkSlicesMatch(t, tc.items[1:], got)

			// Get just the second element.
			got, err = tc.ss.GetContents(ctx, tc.nonce+1, 1)
			if err != nil {
				t.Errorf("unexpected error in GetContents: %v", err)
			}

			// Check elements are expected.
			checkSlicesMatch(t, tc.items[1:2], got)
		})
	}
}

func TestGetLast(t *testing.T) {
	ss := NewSliceStorage[int]()
	ctx := context.Background()

	// GetLast on empty queue returns nil.
	got, err := ss.GetLast(ctx)
	if err != nil {
		t.Errorf("unexpected error in GetLast: %v", err)
	}
	if got != nil {
		t.Errorf("unexpected non-nil result for GetLast: %v", got)
	}

	item1 := 47
	ss.Put(ctx, 0, nil, &item1)

	// GetLast on non-empty queue returns last item.
	got, err = ss.GetLast(ctx)
	if err != nil {
		t.Errorf("unexpected error in GetLast: %v", err)
	}
	if got != &item1 {
		t.Errorf("unexpected value for GetLast: want %v, got %v", &item1, got)
	}
	if item1 != *got {
		t.Errorf("unexpected ptr for GetLast: want %v, got %v", item1, *got)
	}
}

func TestPrune(t *testing.T) {
	ctx := context.Background()
	const maxLen uint64 = 4
	tests := map[string]struct {
		items []int
		ss    *SliceStorage[int]
		nonce uint64
	}{
		"zero_nonce": {
			items: []int{42, 43, 44},
			ss:    NewSliceStorage[int](),
			nonce: 0,
		},
		"nonzero_nonce": {
			items: []int{252, 253, 254, 255},
			ss:    NewSliceStorage[int](),
			nonce: 9,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Insert items.
			for i := 0; i < len(tc.items); i++ {
				if err := tc.ss.Put(ctx, tc.nonce+uint64(i), nil, &tc.items[i]); err != nil {
					t.Errorf("unable to put item %v: %v", tc.items[i], err)
				}
			}

			// Pruning from 0 should leave the contents unchanged.
			if err := tc.ss.Prune(ctx, 0); err != nil {
				t.Errorf("unexpected error in Prune: %v", err)
			}

			// Check elements are expected.
			checkPruneResults(ctx, t, tc.ss, tc.items, tc.nonce)

			// Pruning first 2 elements from the queue.
			if err := tc.ss.Prune(ctx, tc.nonce+2); err != nil {
				t.Errorf("unexpected error in Prune: %v", err)
			}

			// Check elements are expected.
			checkPruneResults(ctx, t, tc.ss, tc.items[2:], tc.nonce)

			// Pruning past the end of queue results in nil queue.
			if err := tc.ss.Prune(ctx, tc.nonce+maxLen); err != nil {
				t.Errorf("unexpected error in Prune: %v", err)
			}

			if tc.ss.queue != nil {
				t.Errorf("unexpected non-nil queue after prune: %v", tc.ss.queue)
			}
		})
	}
}

func TestPut(t *testing.T) {
	ctx := context.Background()
	tests := map[string]struct {
		ss          *SliceStorage[int]
		nonce       uint64
		checkTooLow bool
	}{
		"zero_nonce": {
			ss:          NewSliceStorage[int](),
			nonce:       0,
			checkTooLow: false,
		},
		"nonzero_nonce": {
			ss:          NewSliceStorage[int](),
			nonce:       9,
			checkTooLow: true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Inserting nil should always return an erro
			if err := tc.ss.Put(ctx, tc.nonce, nil, nil); err == nil {
				t.Error("unexpected success when inserting nil item")
			}

			// Some test ints.
			items1 := []int{47, 74}
			items2 := []int{47, 747}

			// Put into an empty queue with non-nil prevItem should return an error.
			if err := tc.ss.Put(ctx, tc.nonce, &items1[0], &items1[1]); err == nil {
				t.Error("unexpected success when inserting non-nil prev item")
			}

			// Successful put should initialize the queue.
			if err := tc.ss.Put(ctx, tc.nonce, nil, &items1[0]); err != nil {
				t.Errorf("unexpected failure when inserting first item: %v", err)
			}

			// Put into the end of the queue, but with non-nil prevItem should return an error.
			if err := tc.ss.Put(ctx, tc.nonce+1, &items1[1], &items1[1]); err == nil {
				t.Error("unexpected success when inserting non-nil prev item at end")
			}

			// Successful put at the end to grow underlying queue.
			if err := tc.ss.Put(ctx, tc.nonce+1, nil, &items1[1]); err != nil {
				t.Errorf("unexpected failure when inserting second item: %v", err)
			}

			// Check contents.
			got, err := tc.ss.GetContents(ctx, tc.nonce, uint64(len(items1)))
			if err != nil {
				t.Errorf("unexpected error in GetContents: %v", err)
			}
			// Check that that contents are expected.
			checkSlicesMatch(t, items1, got)

			// Put into an invalid (too large) index.
			if err := tc.ss.Put(ctx, tc.nonce+2, &items2[1], &items2[1]); err == nil {
				t.Error("unexpected success when inserting at too large index")
			}

			// Successful put to override first element of the queue.
			if err := tc.ss.Put(ctx, tc.nonce, &items1[0], &items2[0]); err != nil {
				t.Errorf("unexpected failure when inserting second item: %v", err)
			}

			// Successful put to override second element of the queue.
			if err := tc.ss.Put(ctx, tc.nonce+1, &items1[1], &items2[1]); err != nil {
				t.Errorf("unexpected failure when inserting second item: %v", err)
			}

			// Check contents.
			got, err = tc.ss.GetContents(ctx, tc.nonce, uint64(len(items2)))
			if err != nil {
				t.Errorf("unexpected error in GetContents: %v", err)
			}
			// Check that that contents are expected.
			checkSlicesMatch(t, items2, got)

			if tc.checkTooLow {
				// Put into an invalid (too small) index.
				if err := tc.ss.Put(ctx, tc.nonce-1, &items2[1], &items2[1]); err == nil {
					t.Error("unexpected success when inserting at too small index")
				}
			}
		})
	}
}
