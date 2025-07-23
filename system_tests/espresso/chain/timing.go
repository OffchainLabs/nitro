package chain

import (
	"time"
)

// TimingData holds the timing information for a generic timeline entry.
// It includes the duration of the event as well as the start and end times.
type TimingData struct {
	Duration time.Duration
	Start    time.Time
	End      time.Time
}

// Timing creates a new TimingData instance with the duration between start
// and end times.
func Timing(start, end time.Time) TimingData {
	return TimingData{
		Duration: end.Sub(start),
		Start:    start,
		End:      end,
	}
}
