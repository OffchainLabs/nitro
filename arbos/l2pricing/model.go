// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package l2pricing

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

const ArbosSingleGasConstraintsVersion = params.ArbosVersion_50
const ArbosMultiGasConstraintsVersion = params.ArbosVersion_60

const InitialSpeedLimitPerSecondV0 = 1000000
const InitialPerBlockGasLimitV0 uint64 = 20 * 1000000
const InitialSpeedLimitPerSecondV6 = 7000000
const InitialPerBlockGasLimitV6 uint64 = 32 * 1000000
const InitialMinimumBaseFeeWei = params.GWei / 10
const InitialBaseFeeWei = InitialMinimumBaseFeeWei
const InitialPricingInertia = 102
const InitialBacklogTolerance = 10
const InitialPerTxGasLimitV50 uint64 = 32 * 1000000

type GasModel int

const (
	GasModelUnknown GasModel = iota
	GasModelLegacy
	GasModelSingleGasConstraints
	GasModelMultiGasConstraints
)

func (ps *L2PricingState) GasModelToUse() (GasModel, error) {
	if ps.ArbosVersion >= ArbosMultiGasConstraintsVersion {
		constraintsLength, err := ps.MultiGasConstraintsLength()
		if err != nil {
			return GasModelUnknown, err
		}
		if constraintsLength > 0 {
			return GasModelMultiGasConstraints, nil
		}
	}
	if ps.ArbosVersion >= ArbosSingleGasConstraintsVersion {
		constraintsLength, err := ps.GasConstraintsLength()
		if err != nil {
			return GasModelUnknown, err
		}
		if constraintsLength > 0 {
			return GasModelSingleGasConstraints, nil
		}
	}
	return GasModelLegacy, nil
}

// AddToGasPool increases gas amount in backlog(s) for the active gas pricing model:
func (ps *L2PricingState) AddToGasPool(usedGas uint64, usedMultiGas multigas.MultiGas) error {
	gasModel, err := ps.GasModelToUse()
	if err != nil {
		return err
	}
	switch gasModel {
	case GasModelLegacy:
		return ps.addToGasPoolLegacy(true, usedGas)
	case GasModelSingleGasConstraints:
		return ps.addToGasPoolWithSingleGasConstraints(true, usedGas)
	case GasModelMultiGasConstraints:
		return ps.addToGasPoolWithMultiGasConstraints(true, usedGas, usedMultiGas)
	default:
		return fmt.Errorf("can not determine gas model")
	}
}

// SubstractGasFromPool decreases gas amount in backlog(s) for the active gas pricing model:
func (ps *L2PricingState) SubstractGasFromPool(usedGas uint64, usedMultiGas multigas.MultiGas) error {
	gasModel, err := ps.GasModelToUse()
	if err != nil {
		return err
	}
	switch gasModel {
	case GasModelLegacy:
		return ps.addToGasPoolLegacy(false, usedGas)
	case GasModelSingleGasConstraints:
		return ps.addToGasPoolWithSingleGasConstraints(false, usedGas)
	case GasModelMultiGasConstraints:
		return ps.addToGasPoolWithMultiGasConstraints(false, usedGas, usedMultiGas)
	default:
		return fmt.Errorf("can not determine gas model")
	}
}

func (ps *L2PricingState) addToGasPoolLegacy(growBacklog bool, usedGas uint64) error {
	backlog, err := ps.GasBacklog()
	if err != nil {
		return err
	}
	backlog = applyGasDelta(backlog, growBacklog, usedGas)
	return ps.SetGasBacklog(backlog)
}

func (ps *L2PricingState) addToGasPoolWithSingleGasConstraints(growBacklog bool, usedGas uint64) error {
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
		err = constraint.SetBacklog(applyGasDelta(backlog, growBacklog, usedGas))
		if err != nil {
			return fmt.Errorf("failed to set backlog of constraint %v: %w", i, err)
		}
	}
	return nil
}

func (ps *L2PricingState) GasPoolUpdateCost() uint64 {
	result := uint64(0)

	// Multi-dimensional constraint pricer costs
	if ps.ArbosVersion >= ArbosMultiGasConstraintsVersion {
		result += storage.StorageReadCost // read multi gas constraints length for "GasModelToUse()"

		constraintsLength, _ := ps.multigasConstraints.Length()
		if constraintsLength > 0 {
			result += storage.StorageReadCost // read length to traverse
			// updating (read+write) all constraints, first one was already accounted for
			result += uint64(constraintsLength) * (storage.StorageReadCost + storage.StorageWriteCost)
			return result
		}
		// no return here, fallthrough to single-constraint costs
	}

	// Single-dimensional constraint pricer costs
	// This overhead applies even when no constraints are configured.
	if ps.ArbosVersion >= ArbosSingleGasConstraintsVersion {
		result += storage.StorageReadCost // read gas constraints length for "GasModelToUse()"
	}

	if ps.ArbosVersion >= params.ArbosVersion_MultiConstraintFix {
		// addToGasPoolWithGasConstraints costs (ArbOS 51 and later)
		constraintsLength, _ := ps.gasConstraints.Length()
		if constraintsLength > 0 {
			result += storage.StorageReadCost // read length to traverse
			// updating (read+write) all constraints, first one was already accounted for
			result += uint64(constraintsLength) * (storage.StorageReadCost + storage.StorageWriteCost)
			return result
		}
		// no return here, fallthrough to legacy costs
	}

	// Legacy pricer costs
	result += storage.StorageReadCost + storage.StorageWriteCost

	return result
}

func (ps *L2PricingState) addToGasPoolWithMultiGasConstraints(growBacklog bool, usedGas uint64, usedMultiGas multigas.MultiGas) error {
	if usedMultiGas.SingleGas() != usedGas {
		log.Warn("usedGas does not match sum of usedMultiGas", "usedGas", usedGas, "usedMultiGas", usedMultiGas.SingleGas())
	}

	constraintsLength, err := ps.multigasConstraints.Length()
	if err != nil {
		return fmt.Errorf("failed to get number of multi-gas constraints: %w", err)
	}
	for i := range constraintsLength {
		constraint := ps.OpenMultiGasConstraintAt(i)
		if growBacklog {
			err = constraint.IncrementBacklog(usedMultiGas)
			if err != nil {
				return fmt.Errorf("failed to increment backlog of multi-gas constraint %v: %w", i, err)
			}
		} else {
			err = constraint.DecrementBacklog(usedMultiGas)
			if err != nil {
				return fmt.Errorf("failed to decrement backlog of multi-gas constraint %v: %w", i, err)
			}
		}
	}
	return nil
}

// UpdatePricingModel updates the pricing model with info from the last block
func (ps *L2PricingState) UpdatePricingModel(timePassed uint64) {
	gasModel, _ := ps.GasModelToUse()
	switch gasModel {
	case GasModelLegacy:
		ps.updatePricingModelLegacy(timePassed)
	case GasModelSingleGasConstraints:
		ps.updatePricingModelSingleConstraints(timePassed)
	case GasModelMultiGasConstraints:
		ps.updatePricingModelMultiConstraints(timePassed)
	}
}

// applyGasDelta grows the backlog if the gas is negative and pays off  if the gas is positive.
func applyGasDelta(backlog uint64, growBacklog bool, delta uint64) uint64 {
	if growBacklog {
		return arbmath.SaturatingUAdd(backlog, delta)
	} else {
		return arbmath.SaturatingUSub(backlog, delta)
	}
}

func (ps *L2PricingState) updatePricingModelLegacy(timePassed uint64) {
	speedLimit, _ := ps.SpeedLimitPerSecond()
	_ = ps.addToGasPoolLegacy(false, arbmath.SaturatingUMul(timePassed, speedLimit))
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

func (ps *L2PricingState) updatePricingModelSingleConstraints(timePassed uint64) {
	// Compute exponent used in the basefee formula
	totalExponent := arbmath.Bips(0)
	constraintsLength, _ := ps.gasConstraints.Length()
	for i := range constraintsLength {
		constraint := ps.OpenGasConstraintAt(i)
		target, _ := constraint.Target()

		// Pay off backlog
		backlog, _ := constraint.Backlog()
		gas := arbmath.SaturatingUMul(timePassed, target)
		backlog = applyGasDelta(backlog, false, gas)
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
	baseFee, _ := ps.calcBaseFeeFromExponent(totalExponent)
	_ = ps.SetBaseFeeWei(baseFee)
}

func (ps *L2PricingState) updatePricingModelMultiConstraints(timePassed uint64) {
	constraintsLength, _ := ps.MultiGasConstraintsLength()

	// Pay off backlog per constraint
	for i := range constraintsLength {
		constraint := ps.OpenMultiGasConstraintAt(i)
		target, _ := constraint.Target()

		backlog, _ := constraint.Backlog()
		gas := arbmath.SaturatingUMul(timePassed, target)
		backlog = applyGasDelta(backlog, false, gas)
		_ = constraint.SetBacklog(backlog)
	}

	// Calculate exponents per resource kind for all constraints
	exponentPerKind, _ := ps.CalcMultiGasConstraintsExponents()

	// Choose the most congested resource
	maxExponent := arbmath.Bips(0)
	for _, exp := range exponentPerKind {
		if exp > maxExponent {
			maxExponent = exp
		}
	}

	// Compute base fee
	baseFee, _ := ps.calcBaseFeeFromExponent(maxExponent)
	_ = ps.SetBaseFeeWei(baseFee)
}

func (ps *L2PricingState) CalcMultiGasConstraintsExponents() ([multigas.NumResourceKind]arbmath.Bips, error) {
	var exponentPerKind [multigas.NumResourceKind]arbmath.Bips
	constraintsLength, _ := ps.MultiGasConstraintsLength()
	for i := range constraintsLength {
		constraint := ps.OpenMultiGasConstraintAt(i)
		target, err := constraint.Target()
		if err != nil {
			return exponentPerKind, fmt.Errorf("failed to read target from constraint %d: %w", i, err)
		}
		backlog, err := constraint.Backlog()
		if err != nil {
			return exponentPerKind, fmt.Errorf("failed to read backlog from constraint %d: %w", i, err)
		}

		if backlog > 0 {
			adjustmentWindow, _ := constraint.AdjustmentWindow()
			sumWeights, _ := constraint.SumWeights()

			divisor := arbmath.SaturatingCastToBips(
				arbmath.SaturatingUMul(uint64(adjustmentWindow),
					arbmath.SaturatingUMul(target, sumWeights)))

			// NOTE: Code below kept for apprach without weight normalization
			// Keeping sumWeights in the divisor would dilute the
			// exponent by that factor and produce lower prices than the old model. Removing
			// it restores the exact legacy exponent behavior.
			//
			// divisor := arbmath.SaturatingCastToBips(
			// 	arbmath.SaturatingUMul(uint64(adjustmentWindow), target))

			usedResources, _ := constraint.UsedResources()
			for _, kind := range usedResources {
				weight, _ := constraint.ResourceWeight(uint8(kind))

				dividend := arbmath.NaturalToBips(
					arbmath.SaturatingCast[int64](arbmath.SaturatingUMul(backlog, weight)))

				exp := dividend / divisor
				exponentPerKind[kind] = arbmath.SaturatingBipsAdd(exponentPerKind[kind], exp)
			}
		}
	}
	return exponentPerKind, nil
}

func (ps *L2PricingState) calcBaseFeeFromExponent(exponent arbmath.Bips) (*big.Int, error) {
	minBaseFee, err := ps.MinBaseFeeWei()
	if err != nil {
		return nil, err
	}
	if exponent > 0 {
		return arbmath.BigMulByBips(minBaseFee, arbmath.ApproxExpBasisPoints(exponent, 4)), nil
	} else {
		return minBaseFee, nil
	}
}
