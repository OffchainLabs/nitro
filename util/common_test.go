// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package util

import "testing"

type stringer interface {
	String() string
}

type myString struct{ s string }

func (m *myString) String() string { return m.s }

func TestIsNil(t *testing.T) {
	var typedNilPtr *int
	nonNilPtr := new(int)

	var nilIface stringer
	var typedNilIface stringer = (*myString)(nil)
	var nonNilIface stringer = &myString{"hi"}

	tests := []struct {
		name string
		v    any
		want bool
	}{
		{"untyped nil", nil, true},
		{"typed-nil pointer", typedNilPtr, true},
		{"non-nil pointer", nonNilPtr, false},
		{"non-pointer value", 42, false},
		{"nil interface", nilIface, true},
		{"typed-nil interface", typedNilIface, true},
		{"non-nil interface", nonNilIface, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNil(tt.v); got != tt.want {
				t.Errorf("IsNil(%v) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}
