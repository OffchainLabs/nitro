// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"testing"

	"github.com/offchainlabs/nitro/util/s3syncer"
)

func TestConfigNumPreallocatedHashes(t *testing.T) {
	cases := []struct {
		name     string
		prealloc bool
		maxMB    int
		want     int
	}{
		{"disabled", false, 10, 0},
		{"no max size", true, 0, 0},
		{"negative max size", true, -1, 0},
		{"one mb", true, 1, 1024 * 1024 / minBytesPerHashEntry},
		{"ten mb", true, 10, 10 * 1024 * 1024 / minBytesPerHashEntry},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{S3: s3syncer.Config{PreallocateMemory: tt.prealloc, MaxFileSizeMB: tt.maxMB}}
			if got := cfg.numPreallocatedHashes(); got != tt.want {
				t.Errorf("numPreallocatedHashes() = %d, want %d", got, tt.want)
			}
		})
	}
}
