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

type tipCollectionTestEnv struct {
	t              *testing.T
	ctx            context.Context
	builder        *NodeBuilder
	arbOwner       *precompilesgen.ArbOwner
	arbOwnerPublic *precompilesgen.ArbOwnerPublic
	networkFeeAddr common.Address
	baseFee        *big.Int
}

func setupTipCollectionTest(t *testing.T, arbosVersion uint64, withL1 bool) (*tipCollectionTestEnv, func()) {
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

	env := &tipCollectionTestEnv{
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

func (env *tipCollectionTestEnv) setCollectTips(collect bool) {
	env.t.Helper()
	ownerAuth := env.builder.L2Info.GetDefaultTransactOpts("Owner", env.ctx)
	tx, err := env.arbOwner.SetCollectTips(&ownerAuth, collect)
	Require(env.t, err)
	_, err = env.builder.L2.EnsureTxSucceeded(tx)
	Require(env.t, err)
}

func (env *tipCollectionTestEnv) getCollectTips() bool {
	env.t.Helper()
	callOpts := env.builder.L2Info.GetDefaultCallOpts("Owner", env.ctx)
	collect, err := env.arbOwnerPublic.GetCollectTips(callOpts)
	Require(env.t, err)
	return collect
}

// sendTxWithTip sends a simple ETH transfer with the given tip and returns the
// network fee account revenue delta and the receipt.
func (env *tipCollectionTestEnv) sendTxWithTip(tip *big.Int) (*big.Int, *types.Receipt) {
	env.t.Helper()
	networkBefore := env.builder.L2.GetBalance(env.t, env.networkFeeAddr)

	info := env.builder.L2Info.GetInfoWithPrivKey("Faucet")
	gasFeeCap := new(big.Int).Add(env.baseFee, tip)
	tx := env.builder.L2Info.SignTxAs("Faucet", &types.DynamicFeeTx{
		To:        &info.Address,
		Gas:       env.builder.L2Info.TransferGas,
		GasTipCap: tip,
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
func (env *tipCollectionTestEnv) expectedRevenue(gasPrice *big.Int, receipt *types.Receipt) *big.Int {
	l2GasUsed := receipt.GasUsed - receipt.GasUsedForL1
	return arbmath.BigMulByUint(gasPrice, l2GasUsed)
}

func (env *tipCollectionTestEnv) assertTipDropped(tip *big.Int, context string) {
	env.t.Helper()
	revenue, receipt := env.sendTxWithTip(tip)
	expected := env.expectedRevenue(env.baseFee, receipt)
	if env.baseFee.Cmp(receipt.EffectiveGasPrice) != 0 {
		Fatal(env.t, context+": incorrect receipt.EffectiveGasPrice", "want", env.baseFee, "got", receipt.EffectiveGasPrice)
	}
	if revenue.Cmp(expected) != 0 {
		Fatal(env.t, context+": tip should be dropped", "revenue", revenue, "expected", expected)
	}
}

func (env *tipCollectionTestEnv) assertTipCollected(tip *big.Int, context string) {
	env.t.Helper()
	revenue, receipt := env.sendTxWithTip(tip)
	gasPrice := new(big.Int).Add(env.baseFee, tip)
	expected := env.expectedRevenue(gasPrice, receipt)
	if gasPrice.Cmp(receipt.EffectiveGasPrice) != 0 {
		Fatal(env.t, context+": incorrect receipt.EffectiveGasPrice", "want", gasPrice, "got", receipt.EffectiveGasPrice)
	}
	if revenue.Cmp(expected) != 0 {
		Fatal(env.t, context+": tip should be collected", "revenue", revenue, "expected", expected)
	}
}

// TestTipCollectionDefault verifies that by default (collect=false), tips are dropped on v60.
func TestTipCollectionDefault(t *testing.T) {
	env, cleanup := setupTipCollectionTest(t, params.ArbosVersion_60, false)
	defer cleanup()

	env.setCollectTips(false)

	if env.getCollectTips() {
		Fatal(t, "expected collect tips to be false")
	}

	tip := big.NewInt(41)
	env.assertTipDropped(tip, "collect=false")
}

// TestTipCollectionEnabled verifies that enabling tip collection causes tips to be collected.
func TestTipCollectionEnabled(t *testing.T) {
	env, cleanup := setupTipCollectionTest(t, params.ArbosVersion_60, false)
	defer cleanup()

	env.setCollectTips(true)

	if !env.getCollectTips() {
		Fatal(t, "expected collect tips to be true")
	}

	tip := big.NewInt(2)
	env.assertTipCollected(tip, "collect=true")
}

// TestTipCollectionVariousTips verifies that various tip amounts are collected when enabled.
func TestTipCollectionVariousTips(t *testing.T) {
	env, cleanup := setupTipCollectionTest(t, params.ArbosVersion_60, false)
	defer cleanup()

	env.setCollectTips(true)

	smallTip := big.NewInt(5)
	env.assertTipCollected(smallTip, "small tip")

	largeTip := big.NewInt(10)
	env.assertTipCollected(largeTip, "large tip")
}

// TestTipCollectionPreV60 verifies that tips are always dropped on pre-v60 chains.
func TestTipCollectionPreV60(t *testing.T) {
	env, cleanup := setupTipCollectionTest(t, params.ArbosVersion_51, false)
	defer cleanup()

	tip := big.NewInt(10)
	env.assertTipDropped(tip, "pre-v60")
}

// TestTipCollectionDisableAfterEnable verifies that disabling tip collection stops collecting tips.
func TestTipCollectionDisableAfterEnable(t *testing.T) {
	env, cleanup := setupTipCollectionTest(t, params.ArbosVersion_60, false)
	defer cleanup()

	env.setCollectTips(true)
	env.setCollectTips(false)

	if env.getCollectTips() {
		Fatal(t, "expected collect tips to be false after disable")
	}

	tip := big.NewInt(10)
	env.assertTipDropped(tip, "collect disabled")
}

// TestTipCollectionV9 verifies that tips are always collected on v9 chains.
func TestTipCollectionV9(t *testing.T) {
	env, cleanup := setupTipCollectionTest(t, params.ArbosVersion_9, false)
	defer cleanup()

	tip := big.NewInt(10)
	env.assertTipCollected(tip, "v9")
}

// TestTipCollectionZeroTip verifies that a tx with zero tip doesn't collect anything
// even when tip collection is enabled.
func TestTipCollectionZeroTip(t *testing.T) {
	env, cleanup := setupTipCollectionTest(t, params.ArbosVersion_60, false)
	defer cleanup()

	env.setCollectTips(true)
	env.assertTipDropped(big.NewInt(0), "zero tip with collect=true")
}

// TestTipCollectionDelayedInboxDropsTips verifies that delayed inbox messages always drop tips,
// even when tip collection is enabled.
func TestTipCollectionDelayedInboxDropsTips(t *testing.T) {
	env, cleanup := setupTipCollectionTest(t, params.ArbosVersion_60, true)
	defer cleanup()

	// Enable tip collection so direct txs would collect tips
	env.setCollectTips(true)

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

	// Verify the receipt's EffectiveGasPrice equals baseFee (not baseFee+tip).
	// This confirms that the block-level CollectTips header flag is false for delayed blocks,
	// which is needed for correct receipt re-derivation.
	blockBaseFee := env.builder.L2.GetBaseFeeAt(t, receipt.BlockNumber)
	if receipt.EffectiveGasPrice.Cmp(blockBaseFee) != 0 {
		Fatal(t, "delayed inbox: EffectiveGasPrice should equal baseFee",
			"effectiveGasPrice", receipt.EffectiveGasPrice, "baseFee", blockBaseFee)
	}
}

// TestTipCollectionGetPaidGasPrice verifies that the GASPRICE opcode (which calls
// GetPaidGasPrice) returns the full gas price when tips are collected and
// only the base fee when tips are dropped.
func TestTipCollectionGetPaidGasPrice(t *testing.T) {
	env, cleanup := setupTipCollectionTest(t, params.ArbosVersion_60, false)
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

	// With collect=false, tips are dropped: GASPRICE should return baseFee
	env.setCollectTips(false)
	observedPrice := callContract(tip)
	if observedPrice.Cmp(env.baseFee) != 0 {
		Fatal(t, "collect=false: GASPRICE should equal baseFee", "observed", observedPrice, "baseFee", env.baseFee)
	}

	// With collect=true, tips are collected: GASPRICE should return baseFee + tip
	env.setCollectTips(true)
	observedPrice = callContract(tip)
	expectedPrice := new(big.Int).Add(env.baseFee, tip)
	if observedPrice.Cmp(expectedPrice) != 0 {
		Fatal(t, "collect=true: GASPRICE should equal baseFee+tip", "observed", observedPrice, "expected", expectedPrice)
	}
}

// TestTipCollectionPrecompileVersionGating verifies that SetCollectTips and
// GetCollectTips revert on pre-v60 chains.
func TestTipCollectionPrecompileVersionGating(t *testing.T) {
	env, cleanup := setupTipCollectionTest(t, params.ArbosVersion_51, false)
	defer cleanup()

	ownerAuth := env.builder.L2Info.GetDefaultTransactOpts("Owner", env.ctx)
	_, err := env.arbOwner.SetCollectTips(&ownerAuth, true)
	if err == nil {
		Fatal(t, "SetCollectTips should revert on pre-v60")
	}

	callOpts := env.builder.L2Info.GetDefaultCallOpts("Owner", env.ctx)
	_, err = env.arbOwnerPublic.GetCollectTips(callOpts)
	if err == nil {
		Fatal(t, "GetCollectTips should revert on pre-v60")
	}
}
