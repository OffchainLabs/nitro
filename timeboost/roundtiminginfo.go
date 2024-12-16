// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package timeboost

import (
	"fmt"
	"time"

	"github.com/offchainlabs/nitro/util/arbmath"
)

// Solgen solidity bindings don't give names to return structs, give it a name for convenience.
type RoundTimingInfo struct {
	OffsetTimestamp          int64
	RoundDurationSeconds     uint64
	AuctionClosingSeconds    uint64
	ReserveSubmissionSeconds uint64
}

// Validate the RoundTimingInfo fields.
// resolutionWaitTime is an additional parameter passed into the auctioneer that it
// needs to validate against the other fields.
func (c *RoundTimingInfo) Validate(resolutionWaitTime *time.Duration) error {
	roundDuration := arbmath.SaturatingCast[time.Duration](c.RoundDurationSeconds) * time.Second
	auctionClosing := arbmath.SaturatingCast[time.Duration](c.AuctionClosingSeconds) * time.Second
	reserveSubmission := arbmath.SaturatingCast[time.Duration](c.ReserveSubmissionSeconds) * time.Second

	// Validate minimum durations
	if roundDuration < time.Second*10 {
		return fmt.Errorf("RoundDurationSeconds (%d) must be at least 10 seconds", c.RoundDurationSeconds)
	}

	if auctionClosing < time.Second*5 {
		return fmt.Errorf("AuctionClosingSeconds (%d) must be at least 5 seconds", c.AuctionClosingSeconds)
	}

	if reserveSubmission < time.Second {
		return fmt.Errorf("ReserveSubmissionSeconds (%d) must be at least 1 second", c.ReserveSubmissionSeconds)
	}

	// Validate combined auction closing and reserve submission against round duration
	combinedClosingTime := auctionClosing + reserveSubmission
	if roundDuration <= combinedClosingTime {
		return fmt.Errorf("RoundDurationSeconds (%d) must be greater than AuctionClosingSeconds (%d) + ReserveSubmissionSeconds (%d) = %d",
			c.RoundDurationSeconds,
			c.AuctionClosingSeconds,
			c.ReserveSubmissionSeconds,
			combinedClosingTime/time.Second)
	}

	// Validate resolution wait time if provided
	if resolutionWaitTime != nil {
		// Resolution wait time shouldn't be more than 50% of auction closing time
		if *resolutionWaitTime > auctionClosing/2 {
			return fmt.Errorf("resolution wait time (%v) must not exceed 50%% of auction closing time (%v)",
				*resolutionWaitTime, auctionClosing)
		}
	}

	return nil
}
