package espressostreamer

import (
	"reflect"
	"testing"
)

func TestFilterAndFind(t *testing.T) {
	tests := []struct {
		name          string
		input         []int
		compareFunc   func(int) int
		wantFound     int
		wantExists    bool
		wantRemaining []int
	}{
		{
			name:  "find middle element and filter smaller ones",
			input: []int{1, 2, 3, 4, 5},
			compareFunc: func(n int) int {
				if n == 3 {
					return 0
				}
				if n < 3 {
					return -1
				}
				return 1
			},
			wantFound:     3,
			wantExists:    true,
			wantRemaining: []int{4, 5},
		},
		{
			name:  "no element found",
			input: []int{1, 2, 3, 4, 5},
			compareFunc: func(n int) int {
				if n < 3 {
					return -1
				}
				return 1
			},
			wantFound:     0, // zero value for int
			wantExists:    false,
			wantRemaining: []int{3, 4, 5},
		},
		{
			name:  "empty slice",
			input: []int{},
			compareFunc: func(n int) int {
				return 1
			},
			wantFound:     0,
			wantExists:    false,
			wantRemaining: []int{},
		},
		{
			name:  "remove all elements",
			input: []int{1, 2, 3},
			compareFunc: func(n int) int {
				return -1
			},
			wantFound:     0,
			wantExists:    false,
			wantRemaining: []int{},
		},
		{
			name:  "keep all elements and find one",
			input: []int{1, 2, 3},
			compareFunc: func(n int) int {
				if n == 2 {
					return 0
				}
				return 1
			},
			wantFound:     2,
			wantExists:    true,
			wantRemaining: []int{1, 3},
		},
		{
			name:  "handle duplicate correctly",
			input: []int{1, 2, 2, 3},
			compareFunc: func(n int) int {
				if n == 2 {
					return 0
				}
				return 1
			},
			wantFound:     2,
			wantExists:    true,
			wantRemaining: []int{1, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of input slice to test against
			input := make([]int, len(tt.input))
			copy(input, tt.input)

			found, exists := FilterAndFind(&input, tt.compareFunc)

			if found != tt.wantFound {
				t.Errorf("FilterAndFind() found = %v, want %v", found, tt.wantFound)
			}

			if exists != tt.wantExists {
				t.Errorf("FilterAndFind() exists = %v, want %v", exists, tt.wantExists)
			}

			if !reflect.DeepEqual(input, tt.wantRemaining) {
				t.Errorf("FilterAndFind() remaining = %v, want %v", input, tt.wantRemaining)
			}
		})
	}

	// Test with nil slice
	t.Run("nil slice", func(t *testing.T) {
		var nilSlice []int
		found, exists := FilterAndFind(&nilSlice, func(n int) int { return 1 })

		if exists {
			t.Error("FilterAndFind() with nil slice should return exists = false")
		}

		if found != 0 {
			t.Errorf("FilterAndFind() with nil slice should return zero value, got %v", found)
		}
	})
}
