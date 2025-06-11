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
		wantRemaining []int
	}{
		{
			name:  "find middle element and filter smaller ones",
			input: []int{1, 2, 3, 4, 5},
			compareFunc: func(n int) int {
				if n == 3 {
					return FilterAndFind_Target
				}
				if n < 3 {
					return FilterAndFind_Remove
				}
				return FilterAndFind_Keep
			},
			wantFound:     0,
			wantRemaining: []int{3, 4, 5},
		},
		{
			name:  "no element found",
			input: []int{1, 2, 3, 4, 5},
			compareFunc: func(n int) int {
				if n < 3 {
					return FilterAndFind_Remove
				}
				return FilterAndFind_Keep
			},
			wantFound:     -1,
			wantRemaining: []int{3, 4, 5},
		},
		{
			name:  "empty slice",
			input: []int{},
			compareFunc: func(n int) int {
				return FilterAndFind_Keep
			},
			wantFound:     -1,
			wantRemaining: []int{},
		},
		{
			name:  "remove all elements",
			input: []int{1, 2, 3},
			compareFunc: func(n int) int {
				return FilterAndFind_Remove
			},
			wantFound:     -1,
			wantRemaining: []int{},
		},
		{
			name:  "keep all elements and find one",
			input: []int{1, 2, 3},
			compareFunc: func(n int) int {
				if n == 2 {
					return FilterAndFind_Target
				}
				return FilterAndFind_Keep
			},
			wantFound:     1,
			wantRemaining: []int{1, 2, 3},
		},
		{
			name:  "handle duplicate correctly",
			input: []int{1, 2, 2, 3},
			compareFunc: func(n int) int {
				if n == 2 {
					return FilterAndFind_Target
				}
				return FilterAndFind_Keep
			},
			wantFound:     1,
			wantRemaining: []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of input slice to test against
			input := make([]int, len(tt.input))
			copy(input, tt.input)

			found := FilterAndFind(&input, tt.compareFunc)

			if found != tt.wantFound {
				t.Errorf("FilterAndFind() found = %v, want %v", found, tt.wantFound)
			}

			if !reflect.DeepEqual(input, tt.wantRemaining) {
				t.Errorf("FilterAndFind() remaining = %v, want %v", input, tt.wantRemaining)
			}
		})
	}

	// Test with nil slice
	t.Run("nil slice", func(t *testing.T) {
		var nilSlice []int
		msgIndex := FilterAndFind(&nilSlice, func(n int) int { return 1 })

		if msgIndex != -1 {
			t.Errorf("FilterAndFind() with nil slice should return zero value, got %v", msgIndex)
		}
	})
}

func TestFilterAndFindWithStruct(t *testing.T) {
	type TestStruct struct {
		ID   int
		Name string
	}

	tests := []struct {
		name          string
		input         []TestStruct
		compareFunc   func(TestStruct) int
		wantFound     int
		wantRemaining []TestStruct
	}{
		{
			name:  "find middle element and filter smaller ones",
			input: []TestStruct{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}, {ID: 3, Name: "Charlie"}},
			compareFunc: func(ts TestStruct) int {
				if ts.ID == 2 {
					return FilterAndFind_Target
				}
				return FilterAndFind_Keep
			},
			wantFound:     1,
			wantRemaining: []TestStruct{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}, {ID: 3, Name: "Charlie"}},
		},
		{
			name:  "consume elements in order and filter out duplicates",
			input: []TestStruct{{ID: 2, Name: "Charlie"}, {ID: 1, Name: "Alice"}, {ID: 1, Name: "Bob"}},
			compareFunc: func(ts TestStruct) int {
				if ts.ID == 1 {
					return FilterAndFind_Target
				}
				return FilterAndFind_Keep
			},
			wantFound:     1,
			wantRemaining: []TestStruct{{ID: 2, Name: "Charlie"}, {ID: 1, Name: "Alice"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of input slice to test against
			input := make([]TestStruct, len(tt.input))
			copy(input, tt.input)

			idx := FilterAndFind(&input, tt.compareFunc)

			if idx != tt.wantFound {
				t.Errorf("FilterAndFind() found = %v, want %v", idx, tt.wantFound)
			}

			if !reflect.DeepEqual(input, tt.wantRemaining) {
				t.Errorf("FilterAndFind() remaining = %v, want %v", input, tt.wantRemaining)
			}
		})
	}
}

// TestCountUniqueEntries tests the CountUniqueEntries function in espressostreamer/utils.go
// It tests that the function works for arbitrary types of array inputs, and properly de-duplicates counting the entries in the list.
// in practice the queue that the espressostreamer maintains might be a bit less
func TestCountUniqueEntries(t *testing.T) {
	strList1 := []string{"One", "Two", "Three"}                  // Should return 3
	strList2 := []string{"One", "Two", "Three", "Three"}         // Should return 3
	strList3 := []string{"One", "Two", "Three", "Three", "Four"} // Should return 4
	intList1 := []uint64{1, 2, 3, 4}                             // should return 4
	intList2 := []uint64{1, 2, 3, 3, 4}                          // should return 4
	intList3 := []uint64{1, 2, 3, 4, 2, 3, 5}                    // should return 5

	// get results from all of the inputs.
	result1 := CountUniqueEntries(&strList1)
	result2 := CountUniqueEntries(&strList2)
	result3 := CountUniqueEntries(&strList3)
	result4 := CountUniqueEntries(&intList1)
	result5 := CountUniqueEntries(&intList2)
	result6 := CountUniqueEntries(&intList3)

	if result1 != 3 {
		t.Errorf("Expected result of 3 for , but got %v", result1)
	}
	if result2 != 3 {
		t.Errorf("Expected result of 3 for , but got %v", result1)
	}
	if result3 != 4 {
		t.Errorf("Expected result of 4 for , but got %v", result1)
	}
	if result4 != 4 {
		t.Errorf("Expected result of 4 for , but got %v", result1)
	}
	if result5 != 4 {
		t.Errorf("Expected result of 4 for , but got %v", result1)
	}
	if result6 != 5 {
		t.Errorf("Expected result of 5 for , but got %v", result1)
	}
}
