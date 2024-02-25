// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import "testing"

func TestBoolRing(t *testing.T) {
	b := NewBoolRing(3)
	if !b.Empty() {
		Fail(t, "not empty as expected")
	}

	b.Update(true)
	if !b.Peek() {
		Fail(t, "Peek returned wrong value")
	}
	if b.All(true) {
		Fail(t, "All shouldn't be true if buffer is not full")
	}

	b.Update(true)
	if !b.Peek() {
		Fail(t, "Peek returned wrong value")
	}
	if b.All(true) {
		Fail(t, "All shouldn't be true if buffer is not full")
	}

	b.Update(true)
	if !b.Peek() {
		Fail(t, "Peek returned wrong value")
	}
	if !b.All(true) {
		Fail(t, "Buffer was full of true, All(true) should be true")
	}

	b.Update(false)
	if b.Peek() {
		Fail(t, "Peek returned wrong value")
	}
	if b.All(true) {
		Fail(t, "Buffer is not full of true")
	}

	b.Update(true)
	if !b.Peek() {
		Fail(t, "Peek returned wrong value")
	}
	if b.All(true) {
		Fail(t, "Buffer is not full of true")
	}

	b.Update(true)
	if !b.Peek() {
		Fail(t, "Peek returned wrong value")
	}
	if b.All(true) {
		Fail(t, "Buffer is not full of true")
	}

	b.Update(true)
	if !b.Peek() {
		Fail(t, "Peek returned wrong value")
	}
	if !b.All(true) {
		Fail(t, "Buffer was full of true, All(true) should be true")
	}

}
