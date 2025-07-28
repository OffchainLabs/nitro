package chain_test

import (
	"testing"
	"time"

	"github.com/offchainlabs/nitro/system_tests/espresso/chain"
)

// TestTiming tests the Timing function to ensure it correctly calculates
// the duration between start and end times, and that it captures the start
// and end times accurately.
func TestTiming(t *testing.T) {
	start := time.Now()
	time.Sleep(100 * time.Millisecond)
	end := time.Now()

	timingData := chain.Timing(start, end)

	if have, want := timingData.Duration, 100*time.Millisecond; have <= want {
		t.Errorf("process should have slept for at least the expected duration:\nhave:\n\t%q\nwant:\n\t%q", have, want)
	}

	if have, want := timingData.Start, start; have != want {
		t.Errorf("start time should match expectation:\nhave:\n\t%q\nwant:\n\t%q", have, want)
	}

	if have, want := timingData.End, end; have != want {
		t.Errorf("end time should match expectation:\nhave:\n\t%q\nwant:\n\t%q", have, want)
	}
}
