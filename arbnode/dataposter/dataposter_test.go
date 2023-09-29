package dataposter

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestParseReplacementTimes(t *testing.T) {
	for _, tc := range []struct {
		desc, replacementTimes string
		want                   []time.Duration
		wantErr                bool
	}{
		{
			desc:             "valid case",
			replacementTimes: "1s,2s,1m,5m",
			want: []time.Duration{
				time.Duration(time.Second),
				time.Duration(2 * time.Second),
				time.Duration(time.Minute),
				time.Duration(5 * time.Minute),
				time.Duration(time.Hour * 24 * 365 * 10),
			},
		},
		{
			desc:             "non-increasing replacement times",
			replacementTimes: "1s,2s,1m,5m,1s",
			wantErr:          true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := parseReplacementTimes(tc.replacementTimes)
			if gotErr := (err != nil); gotErr != tc.wantErr {
				t.Fatalf("Got error: %t, want: %t", gotErr, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("parseReplacementTimes(%s) unexpected diff:\n%s", tc.replacementTimes, diff)
			}
		})
	}
}
