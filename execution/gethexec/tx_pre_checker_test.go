// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import "testing"

func TestTxPreCheckerConfigValidate(t *testing.T) {
	tests := []struct {
		name       string
		maxPending int
		wantErr    bool
	}{
		{"disabled", 0, false},
		{"positive", 100, false},
		{"negative", -1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := DefaultTxPreCheckerConfig
			c.MaxPendingConditionalTxs = tt.maxPending
			err := c.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}
