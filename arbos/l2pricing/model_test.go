// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package l2pricing

import (
	"fmt"
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func toGwei(wei *big.Int) string {
	gweiDivisor := big.NewInt(params.GWei)
	weiRat := new(big.Rat).SetInt(wei)
	gweiDivisorRat := new(big.Rat).SetInt(gweiDivisor)
	gweiRat := new(big.Rat).Quo(weiRat, gweiDivisorRat)
	return gweiRat.FloatString(3)
}

func TestCompareLegacyPricingModelWithSingleGasConstraints(t *testing.T) {
	pricing := PricingForTest(t)

	// In this test, we don't check for storage set errors because they won't happen and they
	// are not the focus of the test.

	// Set the speed limit
	_ = pricing.SetSpeedLimitPerSecond(InitialSpeedLimitPerSecondV6)

	// Compare the basefee for both models with different backlogs
	var backlogs = []uint64{0}
	for i := range uint64(9) {
		backlogs = append(backlogs, 1_000_000*(1+i))
		backlogs = append(backlogs, 10_000_000*(1+i))
		backlogs = append(backlogs, 100_000_000*(1+i))
		backlogs = append(backlogs, 1_000_000_000*(1+i))
		backlogs = append(backlogs, 10_000_000_000*(1+i))
	}

	slices.Sort(backlogs)
	for timePassed := range uint64(100) {
		for _, backlog := range backlogs {
			_ = pricing.gasBacklog.Set(backlog)

			// Initialize with a single constraint based on the legacy model
			_ = pricing.setGasConstraintsFromLegacy()

			pricing.updatePricingModelLegacy(timePassed)
			legacyPrice, _ := pricing.baseFeeWei.Get()

			pricing.updatePricingModelSingleConstraints(timePassed)
			multiPrice, _ := pricing.baseFeeWei.Get()

			if timePassed == 0 {
				fmt.Printf("backlog=%vM\tlegacy=%v gwei\tmultiConstraints=%v gwei\ttimePassed=%v\n",
					backlog/1_000_000, toGwei(legacyPrice), toGwei(multiPrice), timePassed)
			}

			if multiPrice.Cmp(legacyPrice) != 0 {
				t.Errorf("wrong result: backlog=%v, timePassed=%v, multiPrice=%v, legacyPrice=%v",
					backlog, timePassed, multiPrice, legacyPrice)
			}
		}
	}
}

func TestCompareSingleGasConstraintsPricingModelWithMultiGasConstraints(t *testing.T) {
	pricing := PricingForTest(t)

	// Configure base parameters (same as single-constraint test)
	_ = pricing.SetSpeedLimitPerSecond(InitialSpeedLimitPerSecondV6)
	inertia, _ := pricing.PricingInertia()
	target, _ := pricing.SpeedLimitPerSecond()

	// Build a set of backlogs to test
	var backlogs = []uint64{0}
	for i := range uint64(9) {
		backlogs = append(backlogs, 1_000_000*(1+i))
		backlogs = append(backlogs, 10_000_000*(1+i))
		backlogs = append(backlogs, 100_000_000*(1+i))
		backlogs = append(backlogs, 1_000_000_000*(1+i))
		backlogs = append(backlogs, 10_000_000_000*(1+i))
	}

	fmt.Printf("target = %v, inertia = %v\n", target, inertia)

	slices.Sort(backlogs)
	for timePassed := range uint64(1) {
		for _, backlog := range backlogs {
			// Clear any existing constraints
			Require(t, pricing.ClearGasConstraints())
			Require(t, pricing.ClearMultiGasConstraints())

			// Manually create a single-gas constraint:
			// this is the "single-dimensional" model: one constraint with (target, inertia, backlog).
			Require(t, pricing.AddGasConstraint(target, inertia, backlog))

			// Transfer single-gas constraint to multi-gas constraint
			Require(t, pricing.setMultiGasConstraintsFromSingleGasConstraints())

			// Trigger single-constraint pricing update
			pricing.updatePricingModelSingleConstraints(timePassed)
			singlePrice, err := pricing.baseFeeWei.Get()
			Require(t, err)

			// Trigger multi-gas pricing update
			pricing.updatePricingModelMultiConstraints(timePassed)
			multiPrice, err := pricing.baseFeeWei.Get()
			Require(t, err)

			if timePassed == 0 {
				fmt.Printf("backlog=%vM\tlegacy=%v gwei\tmultiConstraints=%v gwei\ttimePassed=%v\n",
					backlog/1_000_000, toGwei(singlePrice), toGwei(multiPrice), timePassed)
			}

			if multiPrice.Cmp(singlePrice) != 0 {
				t.Errorf(
					"mismatch: backlog=%v timePassed=%v single=%v multi=%v",
					backlog, timePassed, singlePrice, multiPrice,
				)
			}
		}
	}
}

func TestCalcMultiGasConstraintsExponents(t *testing.T) {
	pricing := PricingForTest(t)
	pricing.ArbosVersion = params.ArbosVersion_MultiGasConstraintsVersion

	Require(t, pricing.AddMultiGasConstraint(
		100000,
		10,
		20000,
		map[uint8]uint64{
			uint8(multigas.ResourceKindComputation):   1,
			uint8(multigas.ResourceKindStorageAccess): 2,
		},
	))
	Require(t, pricing.AddMultiGasConstraint(
		50000,
		5,
		15000,
		map[uint8]uint64{
			uint8(multigas.ResourceKindStorageGrowth): 1,
		},
	))

	exponents, err := pricing.CalcMultiGasConstraintsExponents()
	Require(t, err)

	if got, want := exponents[multigas.ResourceKindComputation], arbmath.Bips(100); got != want {
		t.Errorf("unexpected computation exponent: got %v, want %v", got, want)
	}
	if got, want := exponents[multigas.ResourceKindStorageAccess], arbmath.Bips(200); got != want {
		t.Errorf("unexpected storage-access exponent: got %v, want %v", got, want)
	}

	if got, want := exponents[multigas.ResourceKindStorageGrowth], arbmath.Bips(600); got != want {
		t.Errorf("unexpected storage-growth exponent: got %v, want %v", got, want)
	}

	// All other kinds should be zero
	if got := exponents[multigas.ResourceKindHistoryGrowth]; got != 0 {
		t.Errorf("expected zero history-growth exponent, got %v", got)
	}
	if got := exponents[multigas.ResourceKindL1Calldata]; got != 0 {
		t.Errorf("expected zero L1 calldata exponent, got %v", got)
	}
	if got := exponents[multigas.ResourceKindL2Calldata]; got != 0 {
		t.Errorf("expected zero L2 calldata exponent, got %v", got)
	}
	if got := exponents[multigas.ResourceKindWasmComputation]; got != 0 {
		t.Errorf("expected zero wasm computation exponent, got %v", got)
	}
}

func TestMultiDimensionalPriceForRefund(t *testing.T) {
	pricing := PricingForTest(t)

	minPrice, err := pricing.MinBaseFeeWei()
	Require(t, err)

	multiGas := multigas.MultiGasFromPairs(
		multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 50000},
		multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 15000},
	)
	// #nosec G115
	singleGas := big.NewInt(int64(multiGas.SingleGas()))
	// Initial price should match minBaseFeeWei * singleGas
	singlePrice := minPrice.Mul(minPrice, singleGas)
	Require(t, err)

	pricing.ArbosVersion = params.ArbosVersion_MultiGasConstraintsVersion

	// Initial price check
	price, err := pricing.MultiDimensionalPriceForRefund(multiGas)
	Require(t, err)
	if price.Cmp(singlePrice) != 0 {
		t.Errorf("Unexpected initial price: got %v, want %v", price, singlePrice)
	}

	// updatePricingModelMultiConstraints() should set multi gas base fees
	Require(t, pricing.AddMultiGasConstraint(
		100000,
		10,
		20000,
		map[uint8]uint64{
			uint8(multigas.ResourceKindComputation):   1,
			uint8(multigas.ResourceKindStorageAccess): 2,
		},
	))
	Require(t, pricing.AddMultiGasConstraint(
		50000,
		5,
		15000,
		map[uint8]uint64{
			uint8(multigas.ResourceKindComputation):   2,
			uint8(multigas.ResourceKindStorageAccess): 1,
		},
	))
	usedMultiGas := multigas.MultiGasFromPairs(
		multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: 500000},
		multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: 1500000},
	)
	err = pricing.GrowBacklog(usedMultiGas.SingleGas(), usedMultiGas)
	Require(t, err)

	pricing.updatePricingModelMultiConstraints(10)

	// Compute the expected price
	err = pricing.CommitMultiGasFees()
	Require(t, err)

	exponentPerKind, err := pricing.CalcMultiGasConstraintsExponents()
	Require(t, err)

	baseFeeWei, err := pricing.BaseFeeWei()
	Require(t, err)

	expectedPrice := new(big.Int)
	for k := 0; k < int(multigas.NumResourceKind); k++ {
		// #nosec G115
		kind := multigas.ResourceKind(k)

		baseFeeKind, err := pricing.calcBaseFeeFromExponent(exponentPerKind[kind])
		Require(t, err)

		// Same override logic as MultiDimensionalPriceForRefund
		if kind == multigas.ResourceKindL1Calldata || baseFeeKind.Sign() == 0 {
			baseFeeKind = baseFeeWei
		}

		amount := multiGas.Get(kind)
		if amount == 0 {
			continue
		}

		part := new(big.Int).Mul(new(big.Int).SetUint64(amount), baseFeeKind)
		expectedPrice.Add(expectedPrice, part)
	}

	price, err = pricing.MultiDimensionalPriceForRefund(multiGas)
	Require(t, err)
	if price.Cmp(expectedPrice) != 0 {
		t.Errorf("Price did not increase after backlog growth: got %v, want > %v", price, expectedPrice)
	}
}

// Use huge backlog and zero timePassed to ensure the updatePricingModel re-calculates base-fee
const benchmarkBacklog = 1_000_000_000
const benchmarkTimePassed = 0

func setupBenchmarkPricingModel(b *testing.B) *L2PricingState {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	if err := InitializeL2PricingState(sto); err != nil {
		b.Fatal(err)
	}
	return OpenL2PricingState(sto, params.ArbosVersion_MultiGasConstraintsVersion)
}

func allResourceWeights() map[uint8]uint64 {
	weights := map[uint8]uint64{}
	for resource := multigas.ResourceKindComputation; resource < multigas.NumResourceKind; resource++ {
		weights[uint8(resource)] = 1
	}
	return weights
}

func BenchmarkUpdatePricing_Legacy(b *testing.B) {
	pricing := setupBenchmarkPricingModel(b)

	if err := pricing.SetSpeedLimitPerSecond(InitialSpeedLimitPerSecondV6); err != nil {
		b.Fatal(err)
	}
	if err := pricing.gasBacklog.Set(benchmarkBacklog); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for b.Loop() {
		pricing.UpdatePricingModel(benchmarkTimePassed)
	}
}

func BenchmarkUpdatePricing_SingleGas_1Constraint(b *testing.B) {
	pricing := setupBenchmarkPricingModel(b)

	if err := pricing.AddGasConstraint(7_000_000, 12, benchmarkBacklog); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for b.Loop() {
		pricing.UpdatePricingModel(benchmarkTimePassed)
	}
}

func BenchmarkUpdatePricing_SingleGas_6Constraints(b *testing.B) {
	pricing := setupBenchmarkPricingModel(b)

	targets := []uint64{60_000_000, 41_000_000, 29_000_000, 20_000_000, 14_000_000, 10_000_000}
	windows := []uint64{9, 52, 329, 2_105, 13_485, 86_400}
	for i := range targets {
		if err := pricing.AddGasConstraint(targets[i], windows[i], benchmarkBacklog); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for b.Loop() {
		pricing.UpdatePricingModel(benchmarkTimePassed)
	}
}

func BenchmarkUpdatePricing_MultiGas_1Constraint(b *testing.B) {
	pricing := setupBenchmarkPricingModel(b)

	weights := allResourceWeights()
	if err := pricing.AddMultiGasConstraint(7_000_000, 12, benchmarkBacklog, weights); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for b.Loop() {
		pricing.UpdatePricingModel(benchmarkTimePassed)
	}
}

func BenchmarkUpdatePricing_MultiGas_6Constraint(b *testing.B) {
	pricing := setupBenchmarkPricingModel(b)

	weights := allResourceWeights()
	targets := []uint64{60_000_000, 41_000_000, 29_000_000, 20_000_000, 14_000_000, 10_000_000}
	windows := []uint32{9, 52, 329, 2_105, 13_485, 86_400}
	for i := range targets {
		if err := pricing.AddMultiGasConstraint(targets[i], windows[i], benchmarkBacklog, weights); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for b.Loop() {
		pricing.UpdatePricingModel(benchmarkTimePassed)
	}
}

func BenchmarkUpdatePricing_MultiGas_42Constraint(b *testing.B) {
	pricing := setupBenchmarkPricingModel(b)

	targets := []uint64{60_000_000, 41_000_000, 29_000_000, 20_000_000, 14_000_000, 10_000_000}
	windows := []uint32{9, 52, 329, 2_105, 13_485, 86_400}
	for resource := multigas.ResourceKindComputation; resource < multigas.NumResourceKind; resource++ {
		weights := map[uint8]uint64{
			uint8(resource): 1,
		}
		for i := range targets {
			if err := pricing.AddMultiGasConstraint(targets[i], windows[i], benchmarkBacklog, weights); err != nil {
				b.Fatal(err)
			}
		}
	}

	b.ResetTimer()
	for b.Loop() {
		pricing.UpdatePricingModel(benchmarkTimePassed)
	}
}
