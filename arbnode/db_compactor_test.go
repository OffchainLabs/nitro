// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"fmt"
	"testing"
	"time"
)

func TestWentPastTimeOfDay(t *testing.T) {
	checkWentPastTimeOfDay := func(before time.Time, after time.Time, timeOfDay string, expected bool) {
		config := DbCompactorConfig{
			TimeOfDay: timeOfDay,
		}
		Require(t, config.Validate(), "Failed to validate sample config")
		have := wentPastTimeOfDay(before, after, config.minutesAfterMidnight)
		if have != expected {
			Fail(t, fmt.Sprintf("Expected wentPastTimeOfDay(%v, %v, \"%v\") to return %v but it returned %v", before, after, timeOfDay, expected, have))
		}
	}

	eleven_pm := time.Date(2000, 1, 1, 23, 0, 0, 0, time.UTC)
	midnight := time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)
	one_am := time.Date(2000, 1, 2, 1, 0, 0, 0, time.UTC)

	checkWentPastTimeOfDay(eleven_pm, eleven_pm, "23:00", false)
	checkWentPastTimeOfDay(midnight, midnight, "00:00", false)
	checkWentPastTimeOfDay(one_am, one_am, "1:00", false)

	checkWentPastTimeOfDay(eleven_pm, midnight, "23:30", true)
	checkWentPastTimeOfDay(eleven_pm, midnight, "00:00", true)
	checkWentPastTimeOfDay(eleven_pm, one_am, "00:00", true)
	checkWentPastTimeOfDay(eleven_pm, one_am, "01:00", true)
	checkWentPastTimeOfDay(eleven_pm, one_am, "02:00", false)
	checkWentPastTimeOfDay(eleven_pm, one_am, "12:00", false)

	checkWentPastTimeOfDay(midnight, one_am, "00:00", false)
	checkWentPastTimeOfDay(midnight, one_am, "00:30", true)
	checkWentPastTimeOfDay(midnight, one_am, "01:00", true)
}
