// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func TestSetAndGetGasPricingConstraints(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{Context: ctx}

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)
	arbGasInfo, err := precompilesgen.NewArbGasInfo(types.ArbGasInfoAddress, builder.L2.Client)
	require.NoError(t, err)

	// Set constraints
	constraints := [][3]uint64{
		{30_000_000, 102, 800_000},   // short-term
		{15_000_000, 600, 1_600_000}, // long-term
	}
	tx, err := arbOwner.SetGasPricingConstraints(&auth, constraints)
	require.NoError(t, err)
	require.NotNil(t, tx)

	// Get and check values
	constraints, err = arbGasInfo.GetGasPricingConstraints(callOpts)
	require.NoError(t, err)
	require.Equal(t, 2, len(constraints))
	require.Equal(t, uint64(30_000_000), constraints[0][0])
	require.Equal(t, uint64(102), constraints[0][1])
	require.GreaterOrEqual(t, constraints[0][2], uint64(800_000))
	require.Equal(t, uint64(15_000_000), constraints[1][0])
	require.Equal(t, uint64(600), constraints[1][1])
	require.GreaterOrEqual(t, constraints[1][2], uint64(1_600_000))
}

func TestSetAndGetMultiGasPricingConstraints(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, false).
		WithArbOSVersion(l2pricing.ArbosMultiGasConstraintsVersion)

	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{Context: ctx}

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)

	arbGasInfo, err := precompilesgen.NewArbGasInfo(types.ArbGasInfoAddress, builder.L2.Client)
	require.NoError(t, err)

	constraint0 := precompilesgen.ArbMultiGasConstraintsTypesResourceConstraint{
		Resources: []precompilesgen.ArbMultiGasConstraintsTypesWeightedResource{
			{Resource: uint8(multigas.ResourceKindComputation), Weight: 3},
			{Resource: uint8(multigas.ResourceKindHistoryGrowth), Weight: 2},
			{Resource: uint8(multigas.ResourceKindStorageAccess), Weight: 1},
			{Resource: uint8(multigas.ResourceKindStorageGrowth), Weight: 4},
			{Resource: uint8(multigas.ResourceKindL1Calldata), Weight: 5},
		},
		AdjustmentWindowSecs: 102,
		TargetPerSec:         30_000_000,
		Backlog:              800_000,
	}

	constraint1 := precompilesgen.ArbMultiGasConstraintsTypesResourceConstraint{
		Resources: []precompilesgen.ArbMultiGasConstraintsTypesWeightedResource{
			{Resource: uint8(multigas.ResourceKindStorageAccess), Weight: 7},
			{Resource: uint8(multigas.ResourceKindL1Calldata), Weight: 9},
			{Resource: uint8(multigas.ResourceKindHistoryGrowth), Weight: 11},
		},
		AdjustmentWindowSecs: 600,
		TargetPerSec:         15_000_000,
		Backlog:              1_600_000,
	}

	constraints := []precompilesgen.ArbMultiGasConstraintsTypesResourceConstraint{
		constraint0,
		constraint1,
	}

	tx, err := arbOwner.SetMultiGasPricingConstraints(&auth, constraints)
	require.NoError(t, err)
	require.NotNil(t, tx)

	readBack, err := arbGasInfo.GetMultiGasPricingConstraints(callOpts)
	require.NoError(t, err)
	require.Equal(t, 2, len(readBack))

	toMap := func(list []precompilesgen.ArbMultiGasConstraintsTypesWeightedResource) map[uint8]uint64 {
		m := make(map[uint8]uint64, len(list))
		for _, r := range list {
			m[r.Resource] = r.Weight
		}
		return m
	}

	want0 := toMap(constraint0.Resources)
	want1 := toMap(constraint1.Resources)

	require.Equal(t, uint64(30_000_000), readBack[0].TargetPerSec)
	require.Equal(t, uint32(102), readBack[0].AdjustmentWindowSecs)
	require.GreaterOrEqual(t, readBack[0].Backlog, uint64(800_000))

	got0 := toMap(readBack[0].Resources)
	require.Equal(t, want0, got0)

	require.Equal(t, uint64(15_000_000), readBack[1].TargetPerSec)
	require.Equal(t, uint32(600), readBack[1].AdjustmentWindowSecs)
	require.GreaterOrEqual(t, readBack[1].Backlog, uint64(1_600_000))

	got1 := toMap(readBack[1].Resources)
	require.Equal(t, want1, got1)
}

func TestMultiGasRefundForNormalTx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, false).
		WithArbOSVersion(l2pricing.ArbosMultiGasConstraintsVersion)

	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	owner := auth.From

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)

	// Set multi-gas constraints with heavy-constrained storage growth
	constraint := precompilesgen.ArbMultiGasConstraintsTypesResourceConstraint{
		Resources: []precompilesgen.ArbMultiGasConstraintsTypesWeightedResource{
			{Resource: uint8(multigas.ResourceKindStorageGrowth), Weight: 1},
		},
		AdjustmentWindowSecs: 10,
		TargetPerSec:         50_000,
		Backlog:              200_000,
	}
	tx, err := arbOwner.SetMultiGasPricingConstraints(&auth, []precompilesgen.ArbMultiGasConstraintsTypesResourceConstraint{
		constraint,
	})
	require.NoError(t, err)
	require.NotNil(t, tx)

	// First transaction: spin the pricing model, should use InitialBaseFeeWei
	tx = builder.L2Info.PrepareTx(
		"Owner", "Owner",
		builder.L2Info.TransferGas,
		big.NewInt(1),
		nil,
	)
	require.NoError(t, builder.L2.Client.SendTransaction(ctx, tx))
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	require.Equal(t, uint64(l2pricing.InitialBaseFeeWei), receipt.EffectiveGasPrice.Uint64())

	// Second transaction: does not use storage growth, should get a positive refund
	balanceBefore, err := builder.L2.Client.BalanceAt(ctx, owner, nil)

	require.NoError(t, err)
	tx = builder.L2Info.PrepareTx(
		"Owner", "Owner",
		builder.L2Info.TransferGas,
		big.NewInt(1),
		nil,
	)
	require.NoError(t, builder.L2.Client.SendTransaction(ctx, tx))
	receipt, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// Ensure base fee is greater than initial (due to the constrained storage growth)
	require.Greater(t, receipt.EffectiveGasPrice.Uint64(), uint64(l2pricing.InitialBaseFeeWei))

	balanceAfter, err := builder.L2.Client.BalanceAt(ctx, owner, nil)
	require.NoError(t, err)

	// Single cost: what the user would pay without multi-gas refund
	gasUsed := receipt.GasUsed
	singleCost := new(big.Int).Mul(
		new(big.Int).SetUint64(gasUsed),
		receipt.EffectiveGasPrice,
	)

	// Expect positive refund
	actualCost := new(big.Int).Sub(balanceBefore, balanceAfter)
	require.Less(t, actualCost.Cmp(singleCost), 0, "expected actual cost < single cost")

	refund := new(big.Int).Sub(singleCost, actualCost)
	require.True(t, refund.Sign() > 0, "expected positive refund, got %v", refund)
}

func TestMultiGasRefundForRetryableTx(t *testing.T) {
	builder, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t, func(b *NodeBuilder) {
		b.WithArbOSVersion(l2pricing.ArbosMultiGasConstraintsVersion)
	})
	defer teardown()

	_, networkFeeAddr := setupFeeAddresses(t, ctx, builder)

	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	simpleAddr, simple := builder.L2.DeploySimple(t, ownerTxOpts)
	simpleABI, err := localgen.SimpleMetaData.GetAbi()
	require.NoError(t, err)

	// Enable multi-gas constraints with heavy-constrained storage growth.
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)

	constraint := precompilesgen.ArbMultiGasConstraintsTypesResourceConstraint{
		Resources: []precompilesgen.ArbMultiGasConstraintsTypesWeightedResource{
			{Resource: uint8(multigas.ResourceKindStorageGrowth), Weight: 1},
		},
		AdjustmentWindowSecs: 10,
		TargetPerSec:         50_000,
		Backlog:              200_000,
	}
	tx, err := arbOwner.SetMultiGasPricingConstraints(&ownerTxOpts, []precompilesgen.ArbMultiGasConstraintsTypesResourceConstraint{
		constraint,
	})
	require.NoError(t, err)
	require.NotNil(t, tx)

	elevateL2Basefee(t, ctx, builder)

	beneficiaryAddress := builder.L2Info.GetAddress("Beneficiary")
	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	callValue := common.Big0

	usertxoptsL1 := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	usertxoptsL1.Value = deposit

	baseFee := builder.L2.GetBaseFee(t)

	l1tx, err := delayedInbox.CreateRetryableTicket(
		&usertxoptsL1,
		simpleAddr,
		callValue,
		big.NewInt(1e16),
		beneficiaryAddress,
		beneficiaryAddress,
		big.NewInt(int64(params.TxGas+params.TxDataNonZeroGasEIP2028*4)),
		big.NewInt(baseFee.Int64()*2),
		simpleABI.Methods["incrementRedeem"].ID,
	)
	require.NoError(t, err)

	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, l1Receipt.Status)

	elevateL2Basefee(t, ctx, builder)
	waitForL1DelayBlocks(t, builder)
	elevateL2Basefee(t, ctx, builder)

	submissionTxOuter := lookupL2Tx(l1Receipt)
	submissionReceipt, err := builder.L2.EnsureTxSucceeded(submissionTxOuter)
	require.NoError(t, err)
	require.Len(t, submissionReceipt.Logs, 2)

	ticketId := submissionReceipt.Logs[0].Topics[1]
	firstRetryTxId := submissionReceipt.Logs[1].Topics[2]

	// Auto-redeem should fail.
	autoRedeemReceipt, err := WaitForTx(ctx, builder.L2.Client, firstRetryTxId, time.Second*5)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusFailed, autoRedeemReceipt.Status)

	usertxoptsL2 := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)
	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), builder.L2.Client)
	require.NoError(t, err)

	tx, err = arbRetryableTx.Redeem(&usertxoptsL2, ticketId)
	require.NoError(t, err)

	redeemReceipt, err := builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)
	retryTxId := redeemReceipt.Logs[0].Topics[2]

	// Get the retry receipt and ensure success.
	retryReceipt, err := WaitForTx(ctx, builder.L2.Client, retryTxId, time.Second*1)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, retryReceipt.Status)

	// Sanity: counter incremented, caller/redeemer as expected.
	counter, err := simple.Counter(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, uint64(1), counter)

	require.Len(t, retryReceipt.Logs, 1)
	parsed, err := simple.ParseRedeemedEvent(*retryReceipt.Logs[0])
	require.NoError(t, err)

	aliasedSender := util.RemapL1Address(usertxoptsL1.From)
	require.Equal(t, aliasedSender, parsed.Caller)
	require.Equal(t, usertxoptsL2.From, parsed.Redeemer)

	// Multi-gas refund check (retryable path)
	// Network fee actually collected for the redeem+retry block.
	networkRedeemFee, err := builder.L2.BalanceDifferenceAtBlock(networkFeeAddr, retryReceipt.BlockNumber)
	require.NoError(t, err)

	// For comparison: naive single-gas network fee for redeem+retry.
	retryTxOuter, _, err := builder.L2.Client.TransactionByHash(ctx, retryTxId)
	require.NoError(t, err)

	retryTx, ok := retryTxOuter.GetInner().(*types.ArbitrumRetryTx)
	require.True(t, ok, "inner tx isn't ArbitrumRetryTx")

	redeemBaseFee := builder.L2.GetBaseFeeAt(t, redeemReceipt.BlockNumber)

	// Same redeem gas accounting as existing retryable fee test.
	redeemGasUsed := redeemReceipt.GasUsed - redeemReceipt.GasUsedForL1 - retryTx.Gas + retryReceipt.GasUsed
	singleGasRedeemFee := arbmath.BigMulByUint(redeemBaseFee, redeemGasUsed)

	// With multi-gas pricing, network fee must not exceed the single-gas cost.
	require.True(
		t,
		networkRedeemFee.Cmp(singleGasRedeemFee) <= 0,
		"expected networkRedeemFee <= single-gas network fee, want <= %v have %v",
		singleGasRedeemFee,
		networkRedeemFee,
	)
}
