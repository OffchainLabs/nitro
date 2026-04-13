// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"testing"
)

func TestStylusTargetConfigValidateNativeStackSize(t *testing.T) {
	tests := []struct {
		name    string
		size    uint64
		wantErr bool
	}{
		{"zero means default", 0, false},
		{"minimum boundary", 8 * 1024, false},
		{"valid 1MB", 1024 * 1024, false},
		{"maximum boundary", 100 * 1024 * 1024, false},
		{"below minimum", 4 * 1024, true},
		{"above maximum", 200 * 1024 * 1024, true},
		{"one byte", 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := DefaultStylusTargetConfig
			c.NativeStackSize = tt.size
			err := c.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("expected error for NativeStackSize=%d, got nil", tt.size)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for NativeStackSize=%d: %v", tt.size, err)
			}
		})
	}
}
