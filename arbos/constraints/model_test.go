// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package constraints

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"

	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbos/storage"
)

func TestConstraintsModelTwoResources(t *testing.T) {
	// Params: target = 5M/sec, Δ = 10s
	var target uint64 = 5_000_000
	const periodSecs = PeriodSecs(10)
	const iterations = 30

	// Setup new constraint-based pricing model with 2 resources
	constraints := NewResourceConstraints()
	resources := EmptyResourceSet().
		WithResources(
			multigas.ResourceKindComputation,
			multigas.ResourceKindStorageAccess,
		)
	constraints.Set(resources, periodSecs, target)

	model := PricingState{constraints: *constraints}

	// Base fee floor
	baseFee := big.NewInt(100_000_000)

	// Phase 1: exceed target to force fee increase
	for i := 0; i < iterations; i++ {
		mg := multigas.MultiGasFromPairs(
			multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 5},
			multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 2},
		)
		model.constraints.Get(resources, periodSecs).AddToBacklog(mg)

		newFee := model.UpdatePricingModel(baseFee, 1)

		// Fee should never fall below baseFee during surge
		require.GreaterOrEqualf(t, newFee.Cmp(baseFee), 0,
			"fee dropped below base fee at iter %d", i)
	}

	// Phase 2: no usage, backlog drains and fee should decay back
	for i := 0; i < iterations*2; i++ {
		newFee := model.UpdatePricingModel(baseFee, 1)

		// Fee must eventually reach the floor
		if i == iterations*2-1 {
			require.Equal(t, 0, newFee.Cmp(baseFee),
				"fee should decay back to base fee")
		}
	}
}

func TestConstraintsModelVersusLegacy(t *testing.T) {
	// Test parameters
	var gasUsedPerSecond int64 = 8_000_000 // >7M target to accumulate backlog
	var iterations int = 50
	var periodSecs = PeriodSecs(12)

	// Initialize L2PricingState with legacy pricing model
	burner := burn.NewSystemBurner(nil, false)
	storage := storage.NewMemoryBacked(burner)
	require.NoError(t, l2pricing.InitializeL2PricingState(storage))
	l2PricingState := l2pricing.OpenL2PricingState(storage)

	// Match new model
	_ = l2PricingState.SetBacklogTolerance(0) // no tolerance
	require.NoError(t, l2PricingState.SetSpeedLimitPerSecond(l2pricing.InitialSpeedLimitPerSecondV6))

	// Setup constraint-based pricing model with a single gas constraint
	constraints := NewResourceConstraints()
	resources := EmptyResourceSet().
		WithResources(
			multigas.ResourceKindComputation,
			multigas.ResourceKindStorageAccess,
			multigas.ResourceKindStorageGrowth,
			multigas.ResourceKindHistoryGrowth,
			multigas.ResourceKindWasmComputation,
		)
	constraints.Set(resources, periodSecs, l2pricing.InitialSpeedLimitPerSecondV6)
	model := PricingState{
		constraints: *constraints,
	}

	minBaseFee, _ := l2PricingState.MinBaseFeeWei()

	for i := 1; i < iterations+1; i++ {
		// L2PricingState model update
		baseFeeLegacy, _ := l2PricingState.BaseFeeWei()
		burner.Restrict(l2PricingState.AddToGasPool(-gasUsedPerSecond)) // negative = gas consumed
		l2PricingState.UpdatePricingModel(baseFeeLegacy, 1, false)
		legacyFee, _ := l2PricingState.BaseFeeWei()

		// Constraint-based model update
		// #nosec G115 -- gasUsedPerSecond is a fixed positive constant for testing
		mg := multigas.ComputationGas(uint64(gasUsedPerSecond))
		model.constraints.Get(resources, periodSecs).AddToBacklog(mg)
		newFee := model.UpdatePricingModel(minBaseFee, 1)

		diff := new(big.Float).Quo(
			new(big.Float).SetInt(legacyFee),
			new(big.Float).SetInt(newFee),
		)
		val, _ := diff.Float64()

		require.InEpsilonf(t, 1.0, val, 0.01, // within 1% tolerance
			"fees differ too much at iteration %d: legacy=%s new=%s",
			i, legacyFee.String(), newFee.String())

		// Uncomment for debug output
		//	fmt.Printf("%-4d %-15s %-15s %-10.4f\n", i, legacyFee.String(), newFee.String(), val)
	}
}
