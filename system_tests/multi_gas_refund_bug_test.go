// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

// runMultiGasRefundDrainTest runs a pump-and-drain scenario and asserts that
// the sender's balance delta equals Σ(EffectiveGasPrice × GasUsed). The two
// lambdas configure the pricing pressure: initialSetup creates it, triggerDrain
// releases it so the next block's base fee crashes.
func runMultiGasRefundDrainTest(
	t *testing.T,
	initialSetup func(b *NodeBuilder, arbOwner *precompilesgen.ArbOwner, auth *bind.TransactOpts),
	triggerDrain func(b *NodeBuilder, arbOwner *precompilesgen.ArbOwner, auth *bind.TransactOpts),
) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, false).
		WithArbOSVersion(params.ArbosVersion_61)
	cleanup := builder.Build(t)
	defer cleanup()

	// Raise the fee cap so the pump burst isn't rejected when base fee spikes.
	builder.L2Info.GasPrice = big.NewInt(100 * params.GWei)

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	ownerAuth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	initialSetup(builder, arbOwner, &ownerAuth)

	// Fund user account.
	builder.L2Info.GenerateAccount("User")
	userAddr := builder.L2Info.GetAddress("User")
	hundredEther := new(big.Int).Mul(big.NewInt(params.Ether), big.NewInt(100))
	TransferBalance(t, "Owner", "User", hundredEther, builder.L2Info, builder.L2.Client, ctx)

	startBalance, err := builder.L2.Client.BalanceAt(ctx, userAddr, nil)
	Require(t, err)

	sendUserTx := func(gasTipCap *big.Int) *types.Receipt {
		base := builder.L2Info.PrepareTxTo("User", &userAddr, builder.L2Info.TransferGas, common.Big0, nil)
		signed := builder.L2Info.SignTxAs("User", &types.DynamicFeeTx{
			To:        base.To(),
			Gas:       base.Gas(),
			GasTipCap: gasTipCap,
			GasFeeCap: base.GasFeeCap(),
			Value:     base.Value(),
			Nonce:     base.Nonce(),
		})
		Require(t, builder.L2.Client.SendTransaction(ctx, signed))
		receipt, err := builder.L2.EnsureTxSucceeded(signed)
		Require(t, err)
		return receipt
	}

	const pumpTxs = 100
	const measurementTxs = 10
	receipts := make([]*types.Receipt, 0, pumpTxs+measurementTxs)

	// Pump the backlog. No refund fires here.
	for range pumpTxs {
		receipts = append(receipts, sendUserTx(common.Big0))
	}

	// Release pressure. The next block's internal tx drains the backlog and
	// BaseFeeWei() crashes, while header.BaseFee still holds the pumped value —
	// opening the drain window where the bug fires.
	triggerDrain(builder, arbOwner, &ownerAuth)

	for range measurementTxs {
		receipts = append(receipts, sendUserTx(common.Big0))
	}

	endBalance, err := builder.L2.Client.BalanceAt(ctx, userAddr, nil)
	Require(t, err)

	// Confirm a drain actually happened; otherwise the bug can't reproduce.
	peak := receipts[0].EffectiveGasPrice
	for _, r := range receipts {
		if r.EffectiveGasPrice.Cmp(peak) > 0 {
			peak = r.EffectiveGasPrice
		}
	}
	final := receipts[len(receipts)-1].EffectiveGasPrice
	if peak.Cmp(final) <= 0 {
		Fatal(t, "base fee did not fall during the run; bug cannot reproduce",
			"peak", peak, "final", final)
	}

	expectedPaid := new(big.Int)
	for _, r := range receipts {
		expectedPaid.Add(expectedPaid,
			new(big.Int).Mul(r.EffectiveGasPrice, new(big.Int).SetUint64(r.GasUsed)))
	}
	actualPaid := new(big.Int).Sub(startBalance, endBalance)

	if actualPaid.Cmp(expectedPaid) != 0 {
		overRefund := new(big.Int).Sub(expectedPaid, actualPaid)
		Fatal(t, "balance delta does not match sum(effectiveGasPrice * gasUsed)",
			"expectedPaid", expectedPaid,
			"actualPaid", actualPaid,
			"overRefund", overRefund)
	}
}

// TestMultiGasRefundWithoutConstraints reproduces the bug under the legacy
// pricing model. The v61 workaround (skip refund when GasModelToUse() !=
// GasModelMultiGasConstraints) prevents the bug; this test verifies that.
func TestMultiGasRefundWithoutConstraints(t *testing.T) {
	setLimit := func(limit uint64) func(*NodeBuilder, *precompilesgen.ArbOwner, *bind.TransactOpts) {
		return func(b *NodeBuilder, arbOwner *precompilesgen.ArbOwner, auth *bind.TransactOpts) {
			tx, err := arbOwner.SetSpeedLimit(auth, limit)
			Require(t, err)
			_, err = b.L2.EnsureTxSucceeded(tx)
			Require(t, err)
		}
	}
	runMultiGasRefundDrainTest(t, setLimit(5_000), setLimit(7_000_000))
}

// TestMultiGasRefundDuringDrainWithConstraints reproduces the root-cause bug
// that the v61 workaround does not address: in MultiDimensionalPriceForRefund
// the SingleDim per-kind fee is overridden with BaseFeeWei() (post-update),
// while EndTxHook's totalCost uses header.BaseFee (pre-update). With a
// uniform-weight constraint there is no legitimate per-kind refund, so any
// drift from balance_lost == Σ(EffGasPrice × GasUsed) is the SingleDim bug.
func TestMultiGasRefundDuringDrainWithConstraints(t *testing.T) {
	installConstraint := func(targetPerSec uint64) func(*NodeBuilder, *precompilesgen.ArbOwner, *bind.TransactOpts) {
		return func(b *NodeBuilder, arbOwner *precompilesgen.ArbOwner, auth *bind.TransactOpts) {
			constraint := precompilesgen.ArbMultiGasConstraintsTypesResourceConstraint{
				Resources: []precompilesgen.ArbMultiGasConstraintsTypesWeightedResource{
					{Resource: uint8(multigas.ResourceKindComputation), Weight: 1},
					{Resource: uint8(multigas.ResourceKindHistoryGrowth), Weight: 1},
					{Resource: uint8(multigas.ResourceKindStorageAccessRead), Weight: 1},
					{Resource: uint8(multigas.ResourceKindStorageAccessWrite), Weight: 1},
					{Resource: uint8(multigas.ResourceKindStorageGrowth), Weight: 1},
				},
				AdjustmentWindowSecs: 100,
				TargetPerSec:         targetPerSec,
				Backlog:              0,
			}
			tx, err := arbOwner.SetMultiGasPricingConstraints(auth,
				[]precompilesgen.ArbMultiGasConstraintsTypesResourceConstraint{constraint})
			Require(t, err)
			_, err = b.L2.EnsureTxSucceeded(tx)
			Require(t, err)
		}
	}
	runMultiGasRefundDrainTest(t, installConstraint(5_000), installConstraint(7_000_000))
}
