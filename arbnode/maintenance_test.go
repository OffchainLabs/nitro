// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package arbnode

import (
	"testing"
	"time"
)

func TestWentPastTimeOfDay(t *testing.T) {
	eleven_pm := time.Date(2000, 1, 1, 23, 0, 0, 0, time.UTC)
	midnight := time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)
	one_am := time.Date(2000, 1, 2, 1, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		before, after time.Time
		timeOfDay     string
		want          bool
	}{
		{before: eleven_pm, after: eleven_pm, timeOfDay: "23:00"},
		{before: midnight, after: midnight, timeOfDay: "00:00"},
		{before: one_am, after: one_am, timeOfDay: "1:00"},
		{before: eleven_pm, after: midnight, timeOfDay: "23:30", want: true},
		{before: eleven_pm, after: midnight, timeOfDay: "00:00", want: true},
		{before: eleven_pm, after: one_am, timeOfDay: "00:00", want: true},
		{before: eleven_pm, after: one_am, timeOfDay: "01:00", want: true},
		{before: eleven_pm, after: one_am, timeOfDay: "02:00"},
		{before: eleven_pm, after: one_am, timeOfDay: "12:00"},
		{before: midnight, after: one_am, timeOfDay: "00:00"},
		{before: midnight, after: one_am, timeOfDay: "00:30", want: true},
		{before: midnight, after: one_am, timeOfDay: "01:00", want: true},
	} {
		config := MaintenanceConfig{TimeOfDay: tc.timeOfDay}
		Require(t, config.Validate(), "Failed to validate sample config")

		if got := wentPastTimeOfDay(tc.before, tc.after, config.minutesAfterMidnight); got != tc.want {
			t.Errorf("wentPastTimeOfDay(%v, %v, %q) = %T want %T", tc.before, tc.after, tc.timeOfDay, got, tc.want)
		}
	}
}
