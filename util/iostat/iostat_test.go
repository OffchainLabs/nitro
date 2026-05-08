// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package iostat

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseStream(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []DeviceStats
	}{
		{
			name: "well formed row",
			input: `
Device r/s w/s await
nvme0n1 1.25 2.50 3.75
`,
			want: []DeviceStats{{
				DeviceName:      "nvme0n1",
				ReadsPerSecond:  1.25,
				WritesPerSecond: 2.50,
				Await:           3.75,
			}},
		},
		{
			name: "header has more columns than data row",
			input: `
Device r/s w/s await
nvme0n1 1.25 2.50
`,
			want: []DeviceStats{{
				DeviceName:      "nvme0n1",
				ReadsPerSecond:  1.25,
				WritesPerSecond: 2.50,
				Await:           0,
			}},
		},
		{
			name: "device header with trailing colon",
			input: `
Device: r/s w/s await
nvme0n1 4.25 5.50 6.75
`,
			want: []DeviceStats{{
				DeviceName:      "nvme0n1",
				ReadsPerSecond:  4.25,
				WritesPerSecond: 5.50,
				Await:           6.75,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver := make(chan DeviceStats)
			go func() {
				parseStream(strings.NewReader(tt.input), receiver)
				close(receiver)
			}()

			var got []DeviceStats
			for stat := range receiver {
				got = append(got, stat)
			}
			require.Equal(t, tt.want, got)
		})
	}
}
