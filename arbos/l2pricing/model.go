// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package l2pricing

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

const ArbosMultiConstraintsVersion = params.ArbosVersion_50

const InitialSpeedLimitPerSecondV0 = 1000000
const InitialPerBlockGasLimitV0 uint64 = 20 * 1000000
const InitialSpeedLimitPerSecondV6 = 7000000
const InitialPerBlockGasLimitV6 uint64 = 32 * 1000000
const InitialMinimumBaseFeeWei = params.GWei / 10
const InitialBaseFeeWei = InitialMinimumBaseFeeWei
const InitialPricingInertia = 102
const InitialBacklogTolerance = 10
const InitialPerTxGasLimitV50 uint64 = 32 * 1000000

func (ps *L2PricingState) ShouldUseGasConstraints(arbosVersion uint64) (bool, error) {
	if arbosVersion >= ArbosMultiConstraintsVersion {
		constraintsLength, err := ps.GasConstraintsLength()
		if err != nil {
			return false, err
		}
		return constraintsLength > 0, nil
	}
	return false, nil
}

func (ps *L2PricingState) AddToGasPool(gas int64, arbosVersion uint64) error {
	shouldUseGasConstraints, err := ps.ShouldUseGasConstraints(arbosVersion)
	if err != nil {
		return err
	}
	if shouldUseGasConstraints {
		return ps.addToGasPoolWithGasConstraints(gas)
	}
	return ps.addToGasPoolLegacy(gas)
}

func (ps *L2PricingState) addToGasPoolLegacy(gas int64) error {
	backlog, err := ps.GasBacklog()
	if err != nil {
		return err
	}
	backlog = applyGasDelta(backlog, gas)
	return ps.SetGasBacklog(backlog)
}

func (ps *L2PricingState) addToGasPoolWithGasConstraints(gas int64) error {
	constraintsLength, err := ps.gasConstraints.Length()
	if err != nil {
		return fmt.Errorf("failed to get number of constraints: %w", err)
	}
	for i := range constraintsLength {
		constraint := ps.OpenGasConstraintAt(i)
		backlog, err := constraint.Backlog()
		if err != nil {
			return fmt.Errorf("failed to get backlog of constraint %v: %w", i, err)
		}
		err = constraint.SetBacklog(applyGasDelta(backlog, gas))
		if err != nil {
			return fmt.Errorf("failed to set backlog of constraint %v: %w", i, err)
		}
	}
	return nil
}

func (ps *L2PricingState) GasPoolUpdateCost(arbosVersion uint64) uint64 {
	var result uint64

	fallBackToLegacy := true
	if arbosVersion >= ArbosMultiConstraintsVersion {
		// Extra read for gas constraints length
		result += storage.StorageReadCost

		// addToGasPoolWithGasConstraints costs
		constraintsLength, _ := ps.gasConstraints.Length()
		if constraintsLength > 0 {
			// reading length one more time + read and write each constraint's Backlog value
			result += storage.StorageReadCost + uint64(constraintsLength)*(storage.StorageReadCost+storage.StorageWriteCost)
			fallBackToLegacy = false
		}
	}

	// addToGasPoolLegacy costs
	if fallBackToLegacy {
		// read and write Backlog value
		result += storage.StorageReadCost + storage.StorageWriteCost
	}
	return result
}

// UpdatePricingModel updates the pricing model with info from the last block
func (ps *L2PricingState) UpdatePricingModel(timePassed uint64, arbosVersion uint64) {
	shouldUseGasConstraints, _ := ps.ShouldUseGasConstraints(arbosVersion)
	if shouldUseGasConstraints {
		ps.updatePricingModelGasConstraints(timePassed)
	} else {
		ps.updatePricingModelLegacy(timePassed)
	}
}

// applyGasDelta grows the backlog if the gas is negative and pays off  if the gas is positive.
func applyGasDelta(backlog uint64, gas int64) uint64 {
	if gas > 0 {
		return arbmath.SaturatingUSub(backlog, uint64(gas))
	} else {
		return arbmath.SaturatingUAdd(backlog, uint64(-gas))
	}
}

func (ps *L2PricingState) updatePricingModelLegacy(timePassed uint64) {
	speedLimit, _ := ps.SpeedLimitPerSecond()
	_ = ps.addToGasPoolLegacy(arbmath.SaturatingCast[int64](arbmath.SaturatingUMul(timePassed, speedLimit)))
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

func (ps *L2PricingState) updatePricingModelGasConstraints(timePassed uint64) {
	// Compute exponent used in the basefee formula
	totalExponent := arbmath.Bips(0)
	constraintsLength, _ := ps.gasConstraints.Length()
	for i := range constraintsLength {
		constraint := ps.OpenGasConstraintAt(i)
		target, _ := constraint.Target()

		// Pay off backlog
		backlog, _ := constraint.Backlog()
		gas := arbmath.SaturatingCast[int64](arbmath.SaturatingUMul(timePassed, target))
		backlog = applyGasDelta(backlog, gas)
		_ = constraint.SetBacklog(backlog)

		// Calculate exponent with the formula backlog/divisor
		if backlog > 0 {
			inertia, _ := constraint.AdjustmentWindow()
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
