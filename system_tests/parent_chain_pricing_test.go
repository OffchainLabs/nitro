// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
)

// pricerResult captures the outcome of running a pricer scenario.
type pricerResult struct {
	finalEstimate *big.Int
	actualBaseFee *big.Int
	error         *big.Int
}

// pricerRealisticResult captures cumulative error over time with batch posting active.
type pricerRealisticResult struct {
	cumulativeError *big.Int // sum of |estimate - actual L1 baseFee| at each sample
	finalError      *big.Int
	samples         int
}

// runPricerScenario sets up a chain, inflates pricePerUnit to 10x the actual
// L1 base fee, advances L1 blocks without batch posting, and returns how far
// the estimate drifted toward the actual base fee.
func runPricerScenario(
	t *testing.T,
	arbosVersion uint64,
	enableMEL bool,
	label string,
) pricerResult {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		DontParalellise().
		WithArbOSVersion(arbosVersion)
	if enableMEL {
		builder.nodeConfig.MessageExtraction.Enable = true
	}
	builder.nodeConfig.BatchPoster.Enable = false
	builder.nodeConfig.DelayedSequencer.FinalizeDistance = 1
	cleanup := builder.Build(t)
	defer cleanup()

	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), builder.L2.Client)
	Require(t, err)
	arbDebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), builder.L2.Client)
	Require(t, err)

	ownerAuth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := arbDebug.BecomeChainOwner(&ownerAuth)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	l1Header, err := builder.L1.Client.HeaderByNumber(ctx, nil)
	Require(t, err)
	actualL1BaseFee := l1Header.BaseFee
	colors.PrintBlue(label, " actual L1 base fee: ", actualL1BaseFee)

	inflatedPrice := new(big.Int).Mul(actualL1BaseFee, big.NewInt(10))
	arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), builder.L2.Client)
	Require(t, err)
	tx, err = arbOwner.SetL1PricePerUnit(&ownerAuth, inflatedPrice)
	Require(t, err)
	_, err = WaitForTx(ctx, builder.L2.Client, tx.Hash(), time.Second*5)
	Require(t, err)

	estimate, err := arbGasInfo.GetL1BaseFeeEstimate(&bind.CallOpts{Context: ctx})
	Require(t, err)
	colors.PrintBlue(label, " initial pricePerUnit: ", estimate)

	// Advance L1 by 128 blocks (4 epochs of 32) without batch posting.
	const totalL1Blocks = 128
	for i := 0; i < totalL1Blocks; i++ {
		builder.L1.TransferBalance(t, "Faucet", "Faucet", common.Big1, builder.L1Info)
		if i%8 == 7 {
			builder.L2.TransferBalance(t, "Owner", "Owner", common.Big0, builder.L2Info)
			time.Sleep(100 * time.Millisecond)
		}
	}

	time.Sleep(2 * time.Second)
	builder.L2.TransferBalance(t, "Owner", "Owner", common.Big0, builder.L2Info)
	time.Sleep(500 * time.Millisecond)

	callOpts := &bind.CallOpts{Context: ctx}
	finalEstimate, err := arbGasInfo.GetL1BaseFeeEstimate(callOpts)
	Require(t, err)

	estError := arbmath.BigAbs(arbmath.BigSub(finalEstimate, actualL1BaseFee))
	colors.PrintMint(label, " final pricePerUnit: ", finalEstimate, " error: ", estError)

	// For the new chain, also verify the precompile getters work.
	if enableMEL && arbosVersion >= params.ArbosVersion_ParentChainPricing {
		parentL1BaseFee, err := arbGasInfo.GetParentChainL1BaseFee(callOpts)
		Require(t, err)
		parentBlobBaseFee, err := arbGasInfo.GetParentChainBlobBaseFee(callOpts)
		Require(t, err)
		parentBlockHash, err := arbGasInfo.GetParentChainBlockHash(callOpts)
		Require(t, err)

		colors.PrintBlue(label, " parent chain L1 base fee from precompile: ", parentL1BaseFee)
		colors.PrintBlue(label, " parent chain blob base fee from precompile: ", parentBlobBaseFee)
		colors.PrintBlue(label, " parent chain block hash from precompile: ", parentBlockHash)

		if parentL1BaseFee.Sign() == 0 {
			t.Error("GetParentChainL1BaseFee returned zero — epoch pricing data not stored")
		}
		if parentBlobBaseFee.Sign() < 0 {
			t.Error("GetParentChainBlobBaseFee returned negative value")
		}
		if parentBlockHash == [32]byte{} {
			t.Error("GetParentChainBlockHash returned zero hash — epoch pricing data not stored")
		}
	}

	return pricerResult{
		finalEstimate: finalEstimate,
		actualBaseFee: actualL1BaseFee,
		error:         estError,
	}
}

// TestParentChainPricingStrictlyBetter demonstrates that the new epoch-based
// parent chain pricing mechanism (ArbOS v70+) converges the L1 fee estimate
// to the actual L1 base fee strictly faster than the legacy batch-posting-report
// mechanism.
//
// Both subtests start with pricePerUnit set to 10x the actual L1 base fee and
// advance 128 L1 blocks with batch posting disabled.
//   - Old (v60): pricePerUnit stays stale — no batch posting reports means no updates.
//   - New (v70+MEL): pricePerUnit converges toward the actual L1 base fee via
//     epoch-based EMA updates every 32 blocks.
func TestParentChainPricingStrictlyBetter(t *testing.T) {
	var resultOld, resultNew pricerResult

	t.Run("OldPricer", func(t *testing.T) {
		resultOld = runPricerScenario(t, params.ArbosVersion_60, false, "[OLD]")
	})
	t.Run("NewPricer", func(t *testing.T) {
		resultNew = runPricerScenario(t, params.ArbosVersion_70, true, "[NEW]")
	})

	// ---- Assertion 1: Old pricer should NOT have moved ----
	inflatedPriceOld := new(big.Int).Mul(resultOld.actualBaseFee, big.NewInt(10))
	drift := arbmath.BigAbs(arbmath.BigSub(resultOld.finalEstimate, inflatedPriceOld))
	maxDrift := arbmath.BigDivByUint(inflatedPriceOld, 100)
	if arbmath.BigGreaterThan(drift, maxDrift) {
		t.Fatalf("Old pricer moved more than 1%% without batch posting reports: drift=%v, max=%v", drift, maxDrift)
	}
	colors.PrintGrey("[OLD] pricer stayed stale as expected (no batch posting reports)")

	// ---- Assertion 2: New pricer should have converged ----
	initialGap := new(big.Int).Mul(resultNew.actualBaseFee, big.NewInt(9))
	if !arbmath.BigGreaterThan(initialGap, resultNew.error) {
		t.Fatalf("New pricer did not converge: initialGap=%v, finalError=%v", initialGap, resultNew.error)
	}
	colors.PrintGrey("[NEW] pricer converged toward actual L1 base fee")

	// ---- Assertion 3: New pricer error is strictly less than old pricer error ----
	if !arbmath.BigGreaterThan(resultOld.error, resultNew.error) {
		t.Fatalf("New pricer is NOT strictly better: errorOld=%v, errorNew=%v", resultOld.error, resultNew.error)
	}
	colors.PrintGrey("New pricer error (", resultNew.error, ") < old pricer error (", resultOld.error, ") — strictly better")
}

// runRealisticPricerScenario sets up a chain with batch posting enabled and
// active L2 traffic. It inflates pricePerUnit, then runs a workload loop
// that generates L2 txs and L1 blocks, allowing batch posting reports to flow.
// It samples the pricing error at each iteration and returns cumulative error.
func runRealisticPricerScenario(
	t *testing.T,
	arbosVersion uint64,
	enableMEL bool,
	initialMultiplier int64,
	label string,
) pricerRealisticResult {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		DontParalellise().
		WithArbOSVersion(arbosVersion)
	if enableMEL {
		builder.nodeConfig.MessageExtraction.Enable = true
	}
	builder.nodeConfig.DelayedSequencer.FinalizeDistance = 1
	cleanup := builder.Build(t)
	defer cleanup()

	// SimulatedBeacon in OnDemand mode produces blocks in the future, so
	// set a negative MaxDelay to prevent the batch poster from skipping.
	builder.nodeConfig.BatchPoster.MaxDelay = -time.Hour

	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), builder.L2.Client)
	Require(t, err)
	arbDebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), builder.L2.Client)
	Require(t, err)

	ownerAuth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := arbDebug.BecomeChainOwner(&ownerAuth)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	l1Header, err := builder.L1.Client.HeaderByNumber(ctx, nil)
	Require(t, err)
	actualL1BaseFee := l1Header.BaseFee
	colors.PrintBlue(label, " actual L1 base fee: ", actualL1BaseFee)

	// Inflate pricePerUnit.
	inflatedPrice := new(big.Int).Mul(actualL1BaseFee, big.NewInt(initialMultiplier))
	arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), builder.L2.Client)
	Require(t, err)
	tx, err = arbOwner.SetL1PricePerUnit(&ownerAuth, inflatedPrice)
	Require(t, err)
	_, err = WaitForTx(ctx, builder.L2.Client, tx.Hash(), time.Second*5)
	Require(t, err)

	estimate, err := arbGasInfo.GetL1BaseFeeEstimate(&bind.CallOpts{Context: ctx})
	Require(t, err)
	colors.PrintBlue(label, " initial pricePerUnit: ", estimate)

	// Track batch posting progress.
	var lastBatchCount uint64
	if builder.L2.ConsensusNode.MessageExtractor != nil {
		lastBatchCount, err = builder.L2.ConsensusNode.MessageExtractor.GetBatchCount()
	} else {
		lastBatchCount, err = builder.L2.ConsensusNode.InboxTracker.GetBatchCount()
	}
	Require(t, err)

	cumulativeError := new(big.Int)
	samples := 0
	lastEstimate := new(big.Int).Set(estimate)

	// Workload: 160 iterations of L2 txs + L1 blocks with batch posting active.
	const iterations = 160
	for i := 0; i < iterations; i++ {
		// Generate L2 activity (drives batch posting).
		builder.L2.TransferBalance(t, "Owner", "Owner", common.Big1, builder.L2Info)
		// Generate L1 blocks.
		builder.L1.TransferBalance(t, "Faucet", "Faucet", common.Big1, builder.L1Info)

		// Sample the current estimate and accumulate error.
		curEstimate, err := arbGasInfo.GetL1BaseFeeEstimate(&bind.CallOpts{Context: ctx})
		Require(t, err)
		l1Header, err = builder.L1.Client.HeaderByNumber(ctx, nil)
		Require(t, err)

		sampleError := arbmath.BigAbs(arbmath.BigSub(curEstimate, l1Header.BaseFee))
		cumulativeError.Add(cumulativeError, sampleError)
		samples++

		if !arbmath.BigEquals(lastEstimate, curEstimate) {
			lastEstimate = curEstimate
		}

		// Periodically wait for a new batch to be posted.
		if i%16 == 0 {
			for j := 50; j > 0; j-- {
				var newBatchCount uint64
				if builder.L2.ConsensusNode.MessageExtractor != nil {
					newBatchCount, err = builder.L2.ConsensusNode.MessageExtractor.GetBatchCount()
				} else {
					newBatchCount, err = builder.L2.ConsensusNode.InboxTracker.GetBatchCount()
				}
				Require(t, err)
				if newBatchCount > lastBatchCount {
					lastBatchCount = newBatchCount
					break
				}
				if j == 1 {
					// Don't fatal — the batch poster may legitimately be slow.
					colors.PrintGrey(label, " batch count stalled at ", lastBatchCount, " after iteration ", i)
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

	finalEstimate, err := arbGasInfo.GetL1BaseFeeEstimate(&bind.CallOpts{Context: ctx})
	Require(t, err)
	l1Header, err = builder.L1.Client.HeaderByNumber(ctx, nil)
	Require(t, err)
	finalError := arbmath.BigAbs(arbmath.BigSub(finalEstimate, l1Header.BaseFee))

	colors.PrintMint(label, " final estimate: ", finalEstimate, " actual: ", l1Header.BaseFee, " finalError: ", finalError)
	colors.PrintMint(label, " cumulative error: ", cumulativeError, " over ", samples, " samples")

	return pricerRealisticResult{
		cumulativeError: cumulativeError,
		finalError:      finalError,
		samples:         samples,
	}
}

// TestParentChainPricingRealisticBatchPosting shows that even with batch posting
// reports flowing (the old mechanism working), the new epoch-based pricing still
// produces strictly less cumulative pricing error over time.
//
// Both chains start with pricePerUnit at 10x the actual L1 base fee and run
// 160 iterations of L2 transactions with batch posting active.
// The old chain corrects only via the surplus/deficit feedback loop in
// UpdateForBatchPosterSpending. The new chain also receives direct L1 base fee
// data every epoch, so it converges faster and accumulates less total error.
func TestParentChainPricingRealisticBatchPosting(t *testing.T) {
	const multiplier = int64(10)
	var resultOld, resultNew pricerRealisticResult

	t.Run("OldPricer", func(t *testing.T) {
		resultOld = runRealisticPricerScenario(t, params.ArbosVersion_60, false, multiplier, "[OLD]")
	})
	t.Run("NewPricer", func(t *testing.T) {
		resultNew = runRealisticPricerScenario(t, params.ArbosVersion_70, true, multiplier, "[NEW]")
	})

	colors.PrintMint("Cumulative error — old: ", resultOld.cumulativeError, " new: ", resultNew.cumulativeError)

	// ---- Assertion: New pricer has strictly less cumulative error ----
	if !arbmath.BigGreaterThan(resultOld.cumulativeError, resultNew.cumulativeError) {
		t.Fatalf(
			"New pricer did NOT accumulate less error than old pricer: old=%v, new=%v",
			resultOld.cumulativeError, resultNew.cumulativeError,
		)
	}
	colors.PrintGrey("New pricer cumulative error (", resultNew.cumulativeError,
		") < old pricer (", resultOld.cumulativeError, ") — strictly better with batch posting active")
}
