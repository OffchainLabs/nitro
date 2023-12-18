// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbmath

import "testing"

func TestMovingAverage(t *testing.T) {
	_, err := NewMovingAverage[int](0)
	if err == nil {
		t.Error("Expected error when creating moving average of period 0")
	}
	_, err = NewMovingAverage[int](-1)
	if err == nil {
		t.Error("Expected error when creating moving average of period -1")
	}

	ma, err := NewMovingAverage[int](5)
	if err != nil {
		t.Fatalf("Error creating moving average of period 5: %v", err)
	}
	if ma.Average() != 0 {
		t.Errorf("Average() = %v, want 0", ma.Average())
	}
	ma.Update(2)
	if ma.Average() != 2 {
		t.Errorf("Average() = %v, want 2", ma.Average())
	}
	ma.Update(4)
	if ma.Average() != 3 {
		t.Errorf("Average() = %v, want 3", ma.Average())
	}

	for i := 0; i < 5; i++ {
		ma.Update(10)
	}
	if ma.Average() != 10 {
		t.Errorf("Average() = %v, want 10", ma.Average())
	}

	for i := 0; i < 5; i++ {
		ma.Update(0)
	}
	if ma.Average() != 0 {
		t.Errorf("Average() = %v, want 0", ma.Average())
	}
}
