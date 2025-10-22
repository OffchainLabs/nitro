// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package l2pricing

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/util/arbmath"
)

const InitialSpeedLimitPerSecondV0 = 1000000
const InitialPerBlockGasLimitV0 uint64 = 20 * 1000000
const InitialSpeedLimitPerSecondV6 = 7000000
const InitialPerBlockGasLimitV6 uint64 = 32 * 1000000
const InitialMinimumBaseFeeWei = params.GWei / 10
const InitialBaseFeeWei = InitialMinimumBaseFeeWei
const InitialPricingInertia = 102
const InitialBacklogTolerance = 10
const InitialPerTxGasLimitV50 uint64 = 32 * 1000000

const ConstraintDivisorMultiplier = 30

func (ps *L2PricingState) AddToGasPool(gas int64) error {
	backlog, err := ps.GasBacklog()
	if err != nil {
		return err
	}
	backlog = applyGasDelta(backlog, gas)
	return ps.SetGasBacklog(backlog)
}

func (ps *L2PricingState) AddToGasPoolMultiConstraints(gas int64) error {
	constraintsLength, err := ps.constraints.Length()
	if err != nil {
		return fmt.Errorf("failed to get number of constraints: %w", err)
	}
	for i := range constraintsLength {
		constraint := ps.OpenConstraintAt(i)
		backlog, err := constraint.backlog.Get()
		if err != nil {
			return fmt.Errorf("failed to get backlog of constraint %v: %w", i, err)
		}
		err = constraint.backlog.Set(applyGasDelta(backlog, gas))
		if err != nil {
			return fmt.Errorf("failed to set backlog of constraint %v: %w", i, err)
		}
	}
	return nil
}

// applyGasDelta grows the backlog if the gas is negative and pays off  if the gas is positive.
func applyGasDelta(backlog uint64, gas int64) uint64 {
	if gas > 0 {
		return arbmath.SaturatingUSub(backlog, uint64(gas))
	} else {
		return arbmath.SaturatingUAdd(backlog, uint64(-gas))
	}
}

// UpdatePricingModel updates the pricing model with info from the last block
func (ps *L2PricingState) UpdatePricingModel(l2BaseFee *big.Int, timePassed uint64, debug bool) {
	speedLimit, _ := ps.SpeedLimitPerSecond()
	_ = ps.AddToGasPool(arbmath.SaturatingCast[int64](arbmath.SaturatingUMul(timePassed, speedLimit)))
	inertia, _ := ps.PricingInertia()
	tolerance, _ := ps.BacklogTolerance()
	backlog, _ := ps.GasBacklog()
	minBaseFee, _ := ps.MinBaseFeeWei()
	baseFee := minBaseFee
	if backlog > tolerance*speedLimit {
		excess := arbmath.SaturatingCast[int64](backlog - tolerance*speedLimit)
		exponentBips := arbmath.NaturalToBips(excess) / arbmath.SaturatingCast[arbmath.Bips](arbmath.SaturatingUMul(inertia, speedLimit))
		baseFee = arbmath.BigMulByBips(minBaseFee, arbmath.ApproxExpBasisPoints(exponentBips, 4))
	}
	_ = ps.SetBaseFeeWei(baseFee)
}

func (ps *L2PricingState) UpdatePricingModelMultiConstraints(timePassed uint64) {
	// Compute exponent used in the basefee formula
	totalExponent := arbmath.Bips(0)
	constraintsLength, _ := ps.constraints.Length()
	for i := range constraintsLength {
		constraint := ps.OpenConstraintAt(i)
		target, _ := constraint.target.Get()

		// Pay off backlog
		backlog, _ := constraint.backlog.Get()
		gas := arbmath.SaturatingCast[int64](arbmath.SaturatingUMul(timePassed, target))
		backlog = applyGasDelta(backlog, gas)
		_ = constraint.backlog.Set(backlog)

		// Calculate exponent with the formula backlog/divisor
		if backlog > 0 {
			inertia, _ := constraint.inertia.Get()
			divisor := arbmath.SaturatingCastToBips(arbmath.SaturatingUMul(inertia, target))
			exponent := arbmath.NaturalToBips(arbmath.SaturatingCast[int64](backlog)) / divisor
			totalExponent = arbmath.SaturatingBipsAdd(totalExponent, exponent)
		}
	}

	// Compute base fee
	minBaseFee, _ := ps.MinBaseFeeWei()
	var baseFee *big.Int
	if totalExponent > 0 {
		baseFee = arbmath.BigMulByBips(minBaseFee, arbmath.ApproxExpBasisPoints(totalExponent, 4))
	} else {
		baseFee = minBaseFee
	}
	_ = ps.SetBaseFeeWei(baseFee)
}
