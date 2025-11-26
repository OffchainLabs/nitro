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

// GasModelToUse returns the active gas-pricing model based on ArbOS version
// and whether the corresponding constraints are currently configured.
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

// GrowBacklog increases the backlog for the active pricing model.
func (ps *L2PricingState) GrowBacklog(usedGas uint64, usedMultiGas multigas.MultiGas) error {
	gasModel, err := ps.GasModelToUse()
	if err != nil {
		return err
	}
	switch gasModel {
	case GasModelLegacy:
		return ps.updateLegacyBacklog(true, usedGas)
	case GasModelSingleGasConstraints:
		return ps.updateSingleGasConstraintsBacklogs(true, usedGas)
	case GasModelMultiGasConstraints:
		return ps.updateMultiGasConstraintsBacklogs(true, usedGas, usedMultiGas)
	default:
		return fmt.Errorf("can not determine gas model")
	}
}

// ShrinkBacklog reduces the backlog for the active pricing model.
func (ps *L2PricingState) ShrinkBacklog(usedGas uint64, usedMultiGas multigas.MultiGas) error {
	gasModel, err := ps.GasModelToUse()
	if err != nil {
		return err
	}
	switch gasModel {
	case GasModelLegacy:
		return ps.updateLegacyBacklog(false, usedGas)
	case GasModelSingleGasConstraints:
		return ps.updateSingleGasConstraintsBacklogs(false, usedGas)
	case GasModelMultiGasConstraints:
		return ps.updateMultiGasConstraintsBacklogs(false, usedGas, usedMultiGas)
	default:
		return fmt.Errorf("can not determine gas model")
	}
}

func (ps *L2PricingState) updateLegacyBacklog(growBacklog bool, usedGas uint64) error {
	backlog, err := ps.GasBacklog()
	if err != nil {
		return err
	}
	backlog = applyGasDelta(backlog, growBacklog, usedGas)
	return ps.SetGasBacklog(backlog)
}

func (ps *L2PricingState) updateSingleGasConstraintsBacklogs(growBacklog bool, usedGas uint64) error {
	constraintsLength, err := ps.gasConstraints.Length()
	if err != nil {
		return err
	}
	for i := range constraintsLength {
		constraint := ps.OpenGasConstraintAt(i)
		backlog, err := constraint.Backlog()
		if err != nil {
			return err
		}
		err = constraint.SetBacklog(applyGasDelta(backlog, growBacklog, usedGas))
		if err != nil {
			return err
		}
	}
	return nil
}

func (ps *L2PricingState) updateMultiGasConstraintsBacklogs(growBacklog bool, usedGas uint64, usedMultiGas multigas.MultiGas) error {
	if usedMultiGas.SingleGas() != usedGas {
		log.Warn("usedGas does not match sum of usedMultiGas", "usedGas", usedGas, "usedMultiGas", usedMultiGas.SingleGas())
	}

	constraintsLength, err := ps.multigasConstraints.Length()
	if err != nil {
		return err
	}
	for i := range constraintsLength {
		constraint := ps.OpenMultiGasConstraintAt(i)
		if growBacklog {
			err = constraint.IncrementBacklog(usedMultiGas)
			if err != nil {
				return err
			}
		} else {
			err = constraint.DecrementBacklog(usedMultiGas)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// applyGasDelta adds delta to backlog if growBacklog=true, otherwise subtracts delta (saturating at zero).
func applyGasDelta(backlog uint64, growBacklog bool, delta uint64) uint64 {
	if growBacklog {
		return arbmath.SaturatingUAdd(backlog, delta)
	} else {
		return arbmath.SaturatingUSub(backlog, delta)
	}
}

// TODO(NIT-4152): eliminate manual gas calculation
// BacklogUpdateCost returns the gas cost for updating the backlog in the active pricing model.
func (ps *L2PricingState) BacklogUpdateCost() uint64 {
	result := uint64(0)

	// Multi-dimensional pricer overhead (ArbOS 60 and later)
	if ps.ArbosVersion >= ArbosMultiGasConstraintsVersion {
		// Read multi-gas constraints length (GasModelToUse)
		// This overhead applies even when no constraints are configured.
		result += storage.StorageReadCost

		// updateMultiGasConstraintsBacklogs costs
		constraintsLength, _ := ps.multigasConstraints.Length()
		if constraintsLength > 0 {
			result += storage.StorageReadCost // read length to traverse

			// DecrementBacklog costs for each multi-dimensional constraint
			result += constraintsLength * uint64(multigas.NumResourceKind) * storage.StorageReadCost
			result += constraintsLength * (storage.StorageReadCost + storage.StorageWriteCost)
			return result
		}
		// No return here, fallthrough to single-constraint costs
	}

	// Single-dimensional constraint pricer costs
	// This overhead applies even when no constraints are configured.
	if ps.ArbosVersion >= ArbosSingleGasConstraintsVersion {
		// Read gas constraints length for "GasModelToUse()"
		result += storage.StorageReadCost
	}

	if ps.ArbosVersion >= params.ArbosVersion_MultiConstraintFix {
		// updateSingleGasConstraintsBacklogs costs (ArbOS 51 and later)
		constraintsLength, _ := ps.gasConstraints.Length()
		if constraintsLength > 0 {
			result += storage.StorageReadCost // read length to traverse
			// Update backlog (read+write) for each constraint
			result += uint64(constraintsLength) * (storage.StorageReadCost + storage.StorageWriteCost)
			return result
		}
		// No return here, fallthrough to legacy costs
	}

	// Legacy pricer costs
	result += storage.StorageReadCost + storage.StorageWriteCost

	return result
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

func (ps *L2PricingState) updatePricingModelLegacy(timePassed uint64) {
	speedLimit, _ := ps.SpeedLimitPerSecond()
	_ = ps.updateLegacyBacklog(false, arbmath.SaturatingUMul(timePassed, speedLimit))
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

// CalcMultiGasConstraintsExponents calculates the exponents for each resource kind
func (ps *L2PricingState) CalcMultiGasConstraintsExponents() ([multigas.NumResourceKind]arbmath.Bips, error) {
	constraintsLength, _ := ps.MultiGasConstraintsLength()
	var exponentPerKind [multigas.NumResourceKind]arbmath.Bips
	for i := range constraintsLength {
		constraint := ps.OpenMultiGasConstraintAt(i)
		target, err := constraint.Target()
		if err != nil {
			return [multigas.NumResourceKind]arbmath.Bips{}, err
		}
		backlog, err := constraint.Backlog()
		if err != nil {
			return [multigas.NumResourceKind]arbmath.Bips{}, err
		}

		if backlog > 0 {
			adjustmentWindow, err := constraint.AdjustmentWindow()
			if err != nil {
				return [multigas.NumResourceKind]arbmath.Bips{}, err
			}
			sumWeights, err := constraint.SumWeights()
			if err != nil {
				return [multigas.NumResourceKind]arbmath.Bips{}, err
			}

			// NOTE: The active divisor follows the multi-dimensional spec:
			//
			//     divisor = A_j * T_j * sum(a_j^i)
			//
			// With this form, the exponent for each resource is scaled by 1/sumWeights
			// compared to the legacy single-gas model. In the compatibility tests we
			// sometimes build a constraint where only ResourceKindComputation has
			// weight=1 (all other weights are 0); in that case sumWeights=1 and this
			// reduces to the legacy exponent. The commented alternative below shows the
			// unnormalized divisor (A_j * T_j) that was used when we wanted to match the
			// old single-gas behavior exactly:
			//
			//     // divisor := arbmath.SaturatingCastToBips(
			//     //     arbmath.SaturatingUMul(uint64(adjustmentWindow), target))
			divisor := arbmath.SaturatingCastToBips(
				arbmath.SaturatingUMul(uint64(adjustmentWindow),
					arbmath.SaturatingUMul(target, sumWeights)))

			usedResources, err := constraint.UsedResources()
			if err != nil {
				return [multigas.NumResourceKind]arbmath.Bips{}, err
			}

			for _, kind := range usedResources {
				weight, err := constraint.ResourceWeight(uint8(kind))
				if err != nil {
					return [multigas.NumResourceKind]arbmath.Bips{}, err
				}

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
