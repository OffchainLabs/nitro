// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type tipCapFloorTestEnv struct {
	t              *testing.T
	ctx            context.Context
	builder        *NodeBuilder
	arbOwner       *precompilesgen.ArbOwner
	arbOwnerPublic *precompilesgen.ArbOwnerPublic
	networkFeeAddr common.Address
	baseFee        *big.Int
}

func setupTipCapTest(t *testing.T, arbosVersion uint64, withL1 bool) (*tipCapFloorTestEnv, func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())

	builder := NewNodeBuilder(ctx).DefaultConfig(t, withL1).WithArbOSVersion(arbosVersion)
	cleanup := builder.Build(t)

	arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), builder.L2.Client)
	Require(t, err)
	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(common.HexToAddress("0x6b"), builder.L2.Client)
	Require(t, err)

	callOpts := builder.L2Info.GetDefaultCallOpts("Owner", ctx)
	networkFeeAddr, err := arbOwnerPublic.GetNetworkFeeAccount(callOpts)
	Require(t, err)

	baseFee := builder.L2.GetBaseFee(t)
	if baseFee.Sign() == 0 {
		Fatal(t, "base fee should not be 0")
	}

	env := &tipCapFloorTestEnv{
		t:              t,
		ctx:            ctx,
		builder:        builder,
		arbOwner:       arbOwner,
		arbOwnerPublic: arbOwnerPublic,
		networkFeeAddr: networkFeeAddr,
		baseFee:        baseFee,
	}

	fullCleanup := func() {
		cleanup()
		cancel()
	}
	return env, fullCleanup
}

func (env *tipCapFloorTestEnv) setTipCapFloor(floor *big.Int) {
	env.t.Helper()
	ownerAuth := env.builder.L2Info.GetDefaultTransactOpts("Owner", env.ctx)
	tx, err := env.arbOwner.SetTipCapFloor(&ownerAuth, floor)
	Require(env.t, err)
	_, err = env.builder.L2.EnsureTxSucceeded(tx)
	Require(env.t, err)
}

func (env *tipCapFloorTestEnv) getTipCapFloor() *big.Int {
	env.t.Helper()
	callOpts := env.builder.L2Info.GetDefaultCallOpts("Owner", env.ctx)
	floor, err := env.arbOwnerPublic.GetTipCapFloor(callOpts)
	Require(env.t, err)
	return floor
}

// sendTxWithTip sends a simple ETH transfer with the given tip and returns the
// network fee account revenue delta and the receipt.
func (env *tipCapFloorTestEnv) sendTxWithTip(tipCap *big.Int) (*big.Int, *types.Receipt) {
	env.t.Helper()
	networkBefore := env.builder.L2.GetBalance(env.t, env.networkFeeAddr)

	info := env.builder.L2Info.GetInfoWithPrivKey("Faucet")
	gasFeeCap := new(big.Int).Add(env.baseFee, tipCap)
	tx := env.builder.L2Info.SignTxAs("Faucet", &types.DynamicFeeTx{
		To:        &info.Address,
		Gas:       env.builder.L2Info.TransferGas,
		GasTipCap: tipCap,
		GasFeeCap: gasFeeCap,
		Value:     big.NewInt(1),
		Nonce:     info.Nonce.Add(1) - 1,
	})
	err := env.builder.L2.Client.SendTransaction(env.ctx, tx)
	Require(env.t, err)
	receipt, err := env.builder.L2.EnsureTxSucceeded(tx)
	Require(env.t, err)

	networkAfter := env.builder.L2.GetBalance(env.t, env.networkFeeAddr)
	revenue := new(big.Int).Sub(networkAfter, networkBefore)
	return revenue, receipt
}

// expectedRevenue returns the expected network fee account revenue for a receipt
// given the effective gas price: gasPrice * l2GasUsed.
func (env *tipCapFloorTestEnv) expectedRevenue(gasPrice *big.Int, receipt *types.Receipt) *big.Int {
	l2GasUsed := receipt.GasUsed - receipt.GasUsedForL1
	return arbmath.BigMulByUint(gasPrice, l2GasUsed)
}

func (env *tipCapFloorTestEnv) assertTipDropped(tipCap *big.Int, context string) {
	env.t.Helper()
	revenue, receipt := env.sendTxWithTip(tipCap)
	expected := env.expectedRevenue(env.baseFee, receipt)
	if revenue.Cmp(expected) != 0 {
		Fatal(env.t, context+": tip should be dropped", "revenue", revenue, "expected", expected)
	}
}

func (env *tipCapFloorTestEnv) assertTipCollected(tipCap *big.Int, context string) {
	env.t.Helper()
	revenue, receipt := env.sendTxWithTip(tipCap)
	gasPrice := new(big.Int).Add(env.baseFee, tipCap)
	expected := env.expectedRevenue(gasPrice, receipt)
	if revenue.Cmp(expected) != 0 {
		Fatal(env.t, context+": tip should be collected", "revenue", revenue, "expected", expected)
	}
}

// TestTipCapFloorDefault verifies that by default (floor=0), tips are dropped on v60.
func TestTipCapFloorDefault(t *testing.T) {
	env, cleanup := setupTipCapTest(t, params.ArbosVersion_60, false)
	defer cleanup()

	floor := env.getTipCapFloor()
	if floor.Sign() != 0 {
		Fatal(t, "expected default tip cap floor to be 0, got", floor)
	}

	tip := big.NewInt(41)
	env.assertTipDropped(tip, "default floor=0")
}

// TestTipCapFloorCollectTips verifies that setting floor=1 causes all tips to be collected.
func TestTipCapFloorCollectTips(t *testing.T) {
	env, cleanup := setupTipCapTest(t, params.ArbosVersion_60, false)
	defer cleanup()

	env.setTipCapFloor(big.NewInt(1))

	floor := env.getTipCapFloor()
	if floor.Cmp(big.NewInt(1)) != 0 {
		Fatal(t, "expected tip cap floor to be 1, got", floor)
	}

	tip := big.NewInt(2)
	env.assertTipCollected(tip, "floor=1")
}

// TestTipCapBelowFloor verifies that tips below the floor are dropped.
func TestTipCapBelowFloor(t *testing.T) {
	env, cleanup := setupTipCapTest(t, params.ArbosVersion_60, false)
	defer cleanup()

	highFloor := big.NewInt(10)
	env.setTipCapFloor(highFloor)

	smallTip := big.NewInt(3)
	env.assertTipDropped(smallTip, "tip below floor")
}

// TestTipCapAboveFloor verifies that tips at or above the floor are collected.
func TestTipCapAboveFloor(t *testing.T) {
	env, cleanup := setupTipCapTest(t, params.ArbosVersion_60, false)
	defer cleanup()

	env.setTipCapFloor(big.NewInt(5))

	exactTip := big.NewInt(5)
	env.assertTipCollected(exactTip, "tip at floor")

	largeTip := big.NewInt(10)
	env.assertTipCollected(largeTip, "tip above floor")
}

// TestTipCapPreV60 verifies that tips are always dropped on pre-v60 chains.
func TestTipCapPreV60(t *testing.T) {
	env, cleanup := setupTipCapTest(t, params.ArbosVersion_51, false)
	defer cleanup()

	tip := big.NewInt(10)
	env.assertTipDropped(tip, "pre-v60")
}

// TestTipCapResetToZero verifies that setting the floor back to 0 disables tip collection.
func TestTipCapResetToZero(t *testing.T) {
	env, cleanup := setupTipCapTest(t, params.ArbosVersion_60, false)
	defer cleanup()

	env.setTipCapFloor(big.NewInt(1))
	env.setTipCapFloor(big.NewInt(0))

	floor := env.getTipCapFloor()
	if floor.Sign() != 0 {
		Fatal(t, "expected floor to be 0 after reset, got", floor)
	}

	tip := big.NewInt(10)
	env.assertTipDropped(tip, "floor reset to 0")
}

// TestTipCapV9CollectsTips verifies that tips are always collected on v9 chains.
func TestTipCapV9CollectsTips(t *testing.T) {
	env, cleanup := setupTipCapTest(t, params.ArbosVersion_9, false)
	defer cleanup()

	tip := big.NewInt(10)
	env.assertTipCollected(tip, "v9")
}

// TestTipCapZeroTip verifies that a tx with zero tip doesn't collect anything
// even when the floor is set to 1.
func TestTipCapZeroTip(t *testing.T) {
	env, cleanup := setupTipCapTest(t, params.ArbosVersion_60, false)
	defer cleanup()

	env.setTipCapFloor(big.NewInt(1))
	env.assertTipDropped(big.NewInt(0), "zero tip with floor=1")
}

// TestTipCapDelayedInboxDropsTips verifies that delayed inbox messages always drop tips,
// even when the floor is set to collect them.
func TestTipCapDelayedInboxDropsTips(t *testing.T) {
	env, cleanup := setupTipCapTest(t, params.ArbosVersion_60, true)
	defer cleanup()

	// Set floor=1 so direct txs would collect tips
	env.setTipCapFloor(big.NewInt(1))

	// Prepare a delayed tx with a tip
	tipCap := env.baseFee
	// Use a high gasFeeCap to account for baseFee drift between measurement and sequencing
	gasFeeCap := new(big.Int).Mul(env.baseFee, big.NewInt(5))
	info := env.builder.L2Info.GetInfoWithPrivKey("Owner")
	delayedTx := env.builder.L2Info.SignTxAs("Owner", &types.DynamicFeeTx{
		To:        &info.Address,
		Gas:       env.builder.L2Info.TransferGas,
		GasTipCap: tipCap,
		GasFeeCap: gasFeeCap,
		Value:     big.NewInt(1),
		Nonce:     info.Nonce.Add(1) - 1,
	})

	networkBefore := env.builder.L2.GetBalance(t, env.networkFeeAddr)

	// Send via delayed inbox
	sendDelayedTx(t, env.ctx, env.builder, delayedTx)
	advanceL1ForDelayed(t, env.ctx, env.builder)

	// Wait for the delayed tx to land on L2
	receipt, err := WaitForTx(env.ctx, env.builder.L2.Client, delayedTx.Hash(), time.Second*10)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "delayed tx failed")
	}

	networkAfter := env.builder.L2.GetBalance(t, env.networkFeeAddr)
	revenue := new(big.Int).Sub(networkAfter, networkBefore)
	expected := env.expectedRevenue(env.baseFee, receipt)

	if revenue.Cmp(expected) != 0 {
		Fatal(t, "delayed inbox: tip should be dropped", "revenue", revenue, "expected", expected)
	}
}

// TestTipCapGetPaidGasPrice verifies that the GASPRICE opcode (which calls
// GetPaidGasPrice) returns the full gas price when tips are collected and
// only the base fee when tips are dropped.
func TestTipCapGetPaidGasPrice(t *testing.T) {
	env, cleanup := setupTipCapTest(t, params.ArbosVersion_60, false)
	defer cleanup()

	// Deploy a contract that stores tx.gasprice in slot 0: GASPRICE PUSH1(0) SSTORE STOP
	runtimeCode := []byte{byte(vm.GASPRICE), byte(vm.PUSH1), 0, byte(vm.SSTORE), byte(vm.STOP)}
	auth := env.builder.L2Info.GetDefaultTransactOpts("Faucet", env.ctx)
	contractAddr := deployContract(t, env.ctx, auth, env.builder.L2.Client, runtimeCode)

	callContract := func(tipCap *big.Int) *big.Int {
		t.Helper()
		info := env.builder.L2Info.GetInfoWithPrivKey("Faucet")
		gasFeeCap := new(big.Int).Add(env.baseFee, tipCap)
		tx := env.builder.L2Info.SignTxAs("Faucet", &types.DynamicFeeTx{
			To:        &contractAddr,
			Gas:       env.builder.L2Info.TransferGas,
			GasTipCap: tipCap,
			GasFeeCap: gasFeeCap,
			Nonce:     info.Nonce.Add(1) - 1,
		})
		err := env.builder.L2.Client.SendTransaction(env.ctx, tx)
		Require(t, err)
		_, err = env.builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
		slot0, err := env.builder.L2.Client.StorageAt(env.ctx, contractAddr, common.Hash{}, nil)
		Require(t, err)
		return new(big.Int).SetBytes(slot0)
	}

	tip := big.NewInt(50)

	// With floor=0 (default), tips are dropped: GASPRICE should return baseFee
	observedPrice := callContract(tip)
	if observedPrice.Cmp(env.baseFee) != 0 {
		Fatal(t, "floor=0: GASPRICE should equal baseFee", "observed", observedPrice, "baseFee", env.baseFee)
	}

	// With floor=1, tips are collected: GASPRICE should return baseFee + tip
	env.setTipCapFloor(big.NewInt(1))
	observedPrice = callContract(tip)
	expectedPrice := new(big.Int).Add(env.baseFee, tip)
	if observedPrice.Cmp(expectedPrice) != 0 {
		Fatal(t, "floor=1: GASPRICE should equal baseFee+tip", "observed", observedPrice, "expected", expectedPrice)
	}
}

// TestTipCapFloorPrecompileVersionGating verifies that SetTipCapFloor and
// GetTipCapFloor revert on pre-v60 chains.
func TestTipCapFloorPrecompileVersionGating(t *testing.T) {
	env, cleanup := setupTipCapTest(t, params.ArbosVersion_51, false)
	defer cleanup()

	ownerAuth := env.builder.L2Info.GetDefaultTransactOpts("Owner", env.ctx)
	_, err := env.arbOwner.SetTipCapFloor(&ownerAuth, big.NewInt(1))
	if err == nil {
		Fatal(t, "SetTipCapFloor should revert on pre-v60")
	}

	callOpts := env.builder.L2Info.GetDefaultCallOpts("Owner", env.ctx)
	_, err = env.arbOwnerPublic.GetTipCapFloor(callOpts)
	if err == nil {
		Fatal(t, "GetTipCapFloor should revert on pre-v60")
	}
}
