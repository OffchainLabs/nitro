// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

type GasSimulator struct {
	CommonConfig
	minGasTarget               uint64
	prevUnnormSequencerBacklog uint64
}

func NewGasSimulator(config CommonConfig, minGasTarget uint64) *GasSimulator {
	return &GasSimulator{config, minGasTarget, 0}
}

func (s *GasSimulator) compute(timestamp uint64, prevBaseFee *big.Int) uint64 {
	// the surge increases linearly over 10 seconds, reaches its peak, and then decreases linearly
	var unnormalizedDemand float64
	if timestamp < s.SurgeRamp {
		unnormalizedDemand = float64(timestamp*s.SurgeDemand) / float64(s.SurgeRamp)
	} else if timestamp < s.SurgeRamp+s.SurgeDuration {
		unnormalizedDemand = float64(s.SurgeDemand)
	} else if timestamp < 2*s.SurgeRamp+s.SurgeDuration {
		unnormalizedDemand = float64((2*s.SurgeRamp+s.SurgeDuration-timestamp-1)*s.SurgeDemand) / float64(s.SurgeRamp)
	}
	unnormalizedDemand = max(unnormalizedDemand, float64(s.minGasTarget))

	// Compute all in float
	prevUnnormSequencerBacklog := float64(s.prevUnnormSequencerBacklog)
	unnormSeqBacklogAfterArrivals := prevUnnormSequencerBacklog + unnormalizedDemand
	normalization := gweiToFloat(prevBaseFee)
	unnormSeqThroughput := float64(s.SequencerThroughput) * normalization
	unnormSeqOutput := min(unnormSeqBacklogAfterArrivals, unnormSeqThroughput)
	unnormSeqBacklog := max(0, prevUnnormSequencerBacklog+unnormalizedDemand-unnormSeqThroughput)

	s.prevUnnormSequencerBacklog = uint64(unnormSeqBacklog / normalization)
	gasUsed := uint64(unnormSeqOutput / normalization)

	return gasUsed
}

func gweiToFloat(value *big.Int) float64 {
	return float64(value.Int64()) / params.GWei
}
