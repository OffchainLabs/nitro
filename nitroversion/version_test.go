// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package nitroversion

import (
	"errors"
	"strings"
	"testing"
)

func TestNitroVersionString(t *testing.T) {
	const timestamp = "2026-01-02T04:04:05+01:00"
	for _, tc := range []struct {
		name       string
		provenance Provenance
		want       string
	}{
		{
			name:       "tagged release",
			provenance: Provenance{Tag: "v3.9.9", Commit: "26b4b9b", Timestamp: timestamp},
			want:       "v3.9.9+26b4b9b-20260102T030405Z",
		},
		{
			name:       "tagged prerelease",
			provenance: Provenance{Tag: "v3.9.9-rc.1", Commit: "26b4b9b", Timestamp: timestamp},
			want:       "v3.9.9-rc.1+26b4b9b-20260102T030405Z",
		},
		{
			name:       "branch build",
			provenance: Provenance{Branch: "branch.name", Commit: "26b4b9b", Timestamp: timestamp},
			want:       "branch.name+26b4b9b-20260102T030405Z",
		},
		{
			name:       "modified build",
			provenance: Provenance{Branch: "dev", Commit: "26b4b9b", Timestamp: timestamp, Modified: true},
			want:       "dev+26b4b9b-20260102T030405Z-modified",
		},
		{
			name:       "local build with unknown provenance",
			provenance: Provenance{Commit: "unknown", Timestamp: "unknown"},
			want:       "local+unknown-unknown",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			version, err := New(tc.provenance)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			if got := version.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNewValidatesProvenance(t *testing.T) {
	for _, tc := range []struct {
		name        string
		provenance  Provenance
		wantErr     error
		wantMessage string
	}{
		{
			name:        "noncanonical tag",
			provenance:  Provenance{Tag: "3.9.9", Timestamp: "2026-01-02T03:04:05Z"},
			wantErr:     ErrInvalidCanonicalVersion,
			wantMessage: "tag",
		},
		{
			name:        "malformed timestamp",
			provenance:  Provenance{Branch: "dev", Timestamp: "not-a-timestamp"},
			wantMessage: `commit timestamp "not-a-timestamp"`,
		},
		{
			name:        "tagged build without timestamp",
			provenance:  Provenance{Tag: "v3.9.9"},
			wantMessage: "tagged version v3.9.9 has no commit timestamp",
		},
		{
			name:        "tagged build with unknown timestamp",
			provenance:  Provenance{Tag: "v3.9.9", Timestamp: "unknown"},
			wantMessage: "tagged version v3.9.9 has no commit timestamp",
		},
		{
			name:        "tagged build with zero timestamp",
			provenance:  Provenance{Tag: "v3.9.9", Timestamp: "0001-01-01T00:00:00Z"},
			wantMessage: "tagged version v3.9.9 has no commit timestamp",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(tc.provenance)
			if err == nil {
				t.Fatal("New() expected an error")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("New() error = %v, want error matching %v", err, tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantMessage) {
				t.Errorf("New() error = %q, want it to contain %q", err, tc.wantMessage)
			}
		})
	}
}
