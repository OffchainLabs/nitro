// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package metricsutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanonicalizeMetricName(t *testing.T) {
	tests := []struct {
		name   string
		metric string
		want   string
	}{
		{name: "hyphenated LVM device vg0-swap", metric: "vg0-swap", want: "vg0_swap"},
		{name: "hyphenated LVM device vg0-root", metric: "vg0-root", want: "vg0_root"},
		{name: "already valid device name nvme0n1", metric: "nvme0n1", want: "nvme0n1"},
		{name: "already valid device name loop0", metric: "loop0", want: "loop0"},
		{name: "dotted numeric token", metric: "99.99", want: "99_99"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, CanonicalizeMetricName(tt.metric))
		})
	}
}
