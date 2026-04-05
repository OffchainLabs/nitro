// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package jsonapi

import (
	"math"
	"testing"
)

func TestUint64String_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		value    Uint64String
		expected string
	}{
		{
			name:     "zero value",
			value:    Uint64String(0),
			expected: `"0"`,
		},
		{
			name:     "small positive value",
			value:    Uint64String(123),
			expected: `"123"`,
		},
		{
			name:     "large value",
			value:    Uint64String(9876543210),
			expected: `"9876543210"`,
		},
		{
			name:     "max uint64",
			value:    Uint64String(math.MaxUint64),
			expected: `"18446744073709551615"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.value.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON() unexpected error: %v", err)
			}
			if string(result) != tt.expected {
				t.Errorf("MarshalJSON() = %s, want %s", string(result), tt.expected)
			}
		})
	}
}

func TestUint64String_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  Uint64String
		shouldErr bool
	}{
		{
			name:      "valid zero",
			input:     `"0"`,
			expected:  Uint64String(0),
			shouldErr: false,
		},
		{
			name:      "valid small number",
			input:     `"456"`,
			expected:  Uint64String(456),
			shouldErr: false,
		},
		{
			name:      "valid large number",
			input:     `"9876543210"`,
			expected:  Uint64String(9876543210),
			shouldErr: false,
		},
		{
			name:      "max uint64",
			input:     `"18446744073709551615"`,
			expected:  Uint64String(math.MaxUint64),
			shouldErr: false,
		},
		{
			name:      "null value",
			input:     `null`,
			expected:  Uint64String(0),
			shouldErr: false,
		},
		{
			name:      "invalid - negative number",
			input:     `"-123"`,
			expected:  Uint64String(0),
			shouldErr: true,
		},
		{
			name:      "invalid - not a string",
			input:     `123`,
			expected:  Uint64String(0),
			shouldErr: true,
		},
		{
			name:      "invalid - non-numeric string",
			input:     `"abc"`,
			expected:  Uint64String(0),
			shouldErr: true,
		},
		{
			name:      "invalid - overflow",
			input:     `"18446744073709551616"`,
			expected:  Uint64String(0),
			shouldErr: true,
		},
		{
			name:      "invalid - empty string",
			input:     `""`,
			expected:  Uint64String(0),
			shouldErr: true,
		},
		{
			name:      "invalid - float",
			input:     `"123.45"`,
			expected:  Uint64String(0),
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result Uint64String
			err := result.UnmarshalJSON([]byte(tt.input))

			if tt.shouldErr {
				if err == nil {
					t.Errorf("UnmarshalJSON() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("UnmarshalJSON() unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("UnmarshalJSON() = %d, want %d", result, tt.expected)
				}
			}
		})
	}
}
