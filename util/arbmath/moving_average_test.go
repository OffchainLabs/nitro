// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbmath

import "testing"

func TestMovingAverage(t *testing.T) {
	ma := NewMovingAverage[int](5)
	if ma.Average() != 0 {
		t.Errorf("moving average should be 0 at start, got %v", ma.Average())
	}
	ma.Update(2)
	if ma.Average() != 2 {
		t.Errorf("moving average should be 2, got %v", ma.Average())
	}
	ma.Update(4)
	if ma.Average() != 3 {
		t.Errorf("moving average should be 3, got %v", ma.Average())
	}

	for i := 0; i < 5; i++ {
		ma.Update(10)
	}
	if ma.Average() != 10 {
		t.Errorf("moving average should be 10, got %v", ma.Average())
	}

	for i := 0; i < 5; i++ {
		ma.Update(0)
	}
	if ma.Average() != 0 {
		t.Errorf("moving average should be 0, got %v", ma.Average())
	}
}
