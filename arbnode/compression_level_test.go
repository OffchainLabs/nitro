// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
)

func TestCompressionLevelStepListValidation(t *testing.T) {
	tests := []struct {
		name    string
		list    CompressionLevelStepList
		wantErr string
	}{
		{
			name:    "empty list",
			list:    CompressionLevelStepList{},
			wantErr: "must have at least one entry",
		},
		{
			name: "first entry not backlog 0",
			list: CompressionLevelStepList{
				{Backlog: 10, Level: 11, RecompressionLevel: 11},
			},
			wantErr: "first compression-levels entry must have backlog: 0",
		},
		{
			name: "valid single entry",
			list: CompressionLevelStepList{
				{Backlog: 0, Level: 11, RecompressionLevel: 11},
			},
			wantErr: "",
		},
		{
			name: "valid default config",
			list: CompressionLevelStepList{
				{Backlog: 0, Level: brotli.BestCompression, RecompressionLevel: brotli.BestCompression},
				{Backlog: 21, Level: brotli.DefaultCompression, RecompressionLevel: brotli.BestCompression},
				{Backlog: 41, Level: brotli.DefaultCompression, RecompressionLevel: brotli.DefaultCompression},
				{Backlog: 61, Level: 4, RecompressionLevel: brotli.DefaultCompression},
			},
			wantErr: "",
		},
		{
			name: "level out of range high",
			list: CompressionLevelStepList{
				{Backlog: 0, Level: 12, RecompressionLevel: 11},
			},
			wantErr: "compression-levels[0].level must be 0-11",
		},
		{
			name: "level out of range negative",
			list: CompressionLevelStepList{
				{Backlog: 0, Level: -1, RecompressionLevel: 11},
			},
			wantErr: "compression-levels[0].level must be 0-11",
		},
		{
			name: "recompression level out of range",
			list: CompressionLevelStepList{
				{Backlog: 0, Level: 11, RecompressionLevel: 12},
			},
			wantErr: "compression-levels[0].recompression-level must be 0-11",
		},
		{
			name: "recompression level less than level",
			list: CompressionLevelStepList{
				{Backlog: 0, Level: 8, RecompressionLevel: 6},
			},
			wantErr: "compression-levels[0].recompression-level (6) must be >= level (8)",
		},
		{
			name: "backlog not ascending",
			list: CompressionLevelStepList{
				{Backlog: 0, Level: 11, RecompressionLevel: 11},
				{Backlog: 20, Level: 6, RecompressionLevel: 11},
				{Backlog: 15, Level: 4, RecompressionLevel: 6},
			},
			wantErr: "compression-levels[2].backlog must be > compression-levels[1].backlog",
		},
		{
			name: "backlog equal not allowed",
			list: CompressionLevelStepList{
				{Backlog: 0, Level: 11, RecompressionLevel: 11},
				{Backlog: 20, Level: 6, RecompressionLevel: 11},
				{Backlog: 20, Level: 4, RecompressionLevel: 6},
			},
			wantErr: "compression-levels[2].backlog must be > compression-levels[1].backlog",
		},
		{
			name: "level not weakly descending",
			list: CompressionLevelStepList{
				{Backlog: 0, Level: 6, RecompressionLevel: 11},
				{Backlog: 20, Level: 8, RecompressionLevel: 11},
			},
			wantErr: "compression-levels[1].level must be <= compression-levels[0].level (weakly descending)",
		},
		{
			name: "recompression level not weakly descending",
			list: CompressionLevelStepList{
				{Backlog: 0, Level: 6, RecompressionLevel: 6},
				{Backlog: 20, Level: 6, RecompressionLevel: 8},
			},
			wantErr: "compression-levels[1].recompression-level must be <= compression-levels[0].recompression-level (weakly descending)",
		},
		{
			name: "valid custom config",
			list: CompressionLevelStepList{
				{Backlog: 0, Level: 11, RecompressionLevel: 11},
				{Backlog: 10, Level: 9, RecompressionLevel: 11},
				{Backlog: 50, Level: 6, RecompressionLevel: 9},
				{Backlog: 100, Level: 4, RecompressionLevel: 6},
			},
			wantErr: "",
		},
		{
			name: "valid with same levels (weakly descending allows equal)",
			list: CompressionLevelStepList{
				{Backlog: 0, Level: 11, RecompressionLevel: 11},
				{Backlog: 20, Level: 11, RecompressionLevel: 11},
				{Backlog: 40, Level: 6, RecompressionLevel: 6},
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.list.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Validate() expected error containing %q, got %q", tt.wantErr, err.Error())
				}
			}
		})
	}
}

func TestCompressionLevelStepListJSONRoundTrip(t *testing.T) {
	original := CompressionLevelStepList{
		{Backlog: 0, Level: 11, RecompressionLevel: 11},
		{Backlog: 21, Level: 6, RecompressionLevel: 11},
		{Backlog: 41, Level: 6, RecompressionLevel: 6},
		{Backlog: 61, Level: 4, RecompressionLevel: 6},
	}

	// Test String() and Set() roundtrip
	jsonStr := original.String()
	var parsed CompressionLevelStepList
	err := parsed.Set(jsonStr)
	if err != nil {
		t.Fatalf("Set() failed: %v", err)
	}

	if len(parsed) != len(original) {
		t.Fatalf("expected %d entries, got %d", len(original), len(parsed))
	}

	for i, step := range original {
		if parsed[i].Backlog != step.Backlog {
			t.Errorf("entry %d: expected backlog %d, got %d", i, step.Backlog, parsed[i].Backlog)
		}
		if parsed[i].Level != step.Level {
			t.Errorf("entry %d: expected level %d, got %d", i, step.Level, parsed[i].Level)
		}
		if parsed[i].RecompressionLevel != step.RecompressionLevel {
			t.Errorf("entry %d: expected recompression-level %d, got %d", i, step.RecompressionLevel, parsed[i].RecompressionLevel)
		}
	}
}
