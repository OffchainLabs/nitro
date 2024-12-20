// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package timeboost

import (
	"fmt"
	"time"

	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// Validate the express_lane_auctiongen.RoundTimingInfo fields.
// Returns errors in terms of the solidity field names to ease debugging.
func validateRoundTimingInfo(c *express_lane_auctiongen.RoundTimingInfo) error {
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

	return nil
}

// RoundTimingInfo holds the information from the Solidity type of the same name,
// validated and converted into higher level time types, with helpful methods
// for calculating round number, if a round is closed, and time til close.
type RoundTimingInfo struct {
	Offset            time.Time
	Round             time.Duration
	AuctionClosing    time.Duration
	ReserveSubmission time.Duration
}

// Convert from solgen bindings to domain type
func NewRoundTimingInfo(c express_lane_auctiongen.RoundTimingInfo) (*RoundTimingInfo, error) {
	if err := validateRoundTimingInfo(&c); err != nil {
		return nil, err
	}

	return &RoundTimingInfo{
		Offset:            time.Unix(c.OffsetTimestamp, 0),
		Round:             arbmath.SaturatingCast[time.Duration](c.RoundDurationSeconds) * time.Second,
		AuctionClosing:    arbmath.SaturatingCast[time.Duration](c.AuctionClosingSeconds) * time.Second,
		ReserveSubmission: arbmath.SaturatingCast[time.Duration](c.ReserveSubmissionSeconds) * time.Second,
	}, nil
}

// resolutionWaitTime is an additional parameter that the Auctioneer
// needs to validate against other timing fields.
func (info *RoundTimingInfo) ValidateResolutionWaitTime(resolutionWaitTime time.Duration) error {
	// Resolution wait time shouldn't be more than 50% of auction closing time
	if resolutionWaitTime > info.AuctionClosing/2 {
		return fmt.Errorf("resolution wait time (%v) must not exceed 50%% of auction closing time (%v)",
			resolutionWaitTime, info.AuctionClosing)
	}
	return nil
}

// RoundNumber returns the round number as of now.
func (info *RoundTimingInfo) RoundNumber() uint64 {
	return info.RoundNumberAt(time.Now())
}

// RoundNumberAt returns the round number as of some timestamp.
func (info *RoundTimingInfo) RoundNumberAt(currentTime time.Time) uint64 {
	return arbmath.SaturatingUCast[uint64](currentTime.Sub(info.Offset) / info.Round)
	// info.Round has already been validated to be nonzero during construction.
}

// TimeTilNextRound returns the time til the next round as of now.
func (info *RoundTimingInfo) TimeTilNextRound() time.Duration {
	return info.TimeTilNextRoundAt(time.Now())
}

// TimeTilNextRoundAt returns the time til the next round,
// where the next round is determined from the timestamp passed in.
func (info *RoundTimingInfo) TimeTilNextRoundAt(currentTime time.Time) time.Duration {
	return info.TimeOfNextRoundAt(currentTime).Sub(currentTime)
}

func (info *RoundTimingInfo) TimeOfNextRound() time.Time {
	return info.TimeOfNextRoundAt(time.Now())
}

func (info *RoundTimingInfo) TimeOfNextRoundAt(currentTime time.Time) time.Time {
	roundNum := info.RoundNumberAt(currentTime)
	return info.Offset.Add(info.Round * arbmath.SaturatingCast[time.Duration](roundNum+1))
}

func (info *RoundTimingInfo) durationIntoRound(timestamp time.Time) time.Duration {
	secondsSinceOffset := uint64(timestamp.Sub(info.Offset).Seconds())
	roundDurationSeconds := uint64(info.Round.Seconds())
	return arbmath.SaturatingCast[time.Duration](secondsSinceOffset % roundDurationSeconds)
}

func (info *RoundTimingInfo) isAuctionRoundClosed() bool {
	return info.isAuctionRoundClosedAt(time.Now())
}

func (info *RoundTimingInfo) isAuctionRoundClosedAt(currentTime time.Time) bool {
	if currentTime.Before(info.Offset) {
		return false
	}

	return info.durationIntoRound(currentTime)*time.Second >= info.Round-info.AuctionClosing
}

func (info *RoundTimingInfo) IsWithinAuctionCloseWindow(timestamp time.Time) bool {
	return info.TimeTilNextRoundAt(timestamp) <= info.AuctionClosing
}
