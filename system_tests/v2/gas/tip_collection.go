// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gas

import (
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/system_tests/v2/systest"
	"github.com/offchainlabs/nitro/util/arbmath"
)

var tipCollectionTests = []systest.Scenario{
	systest.Test(testRunTipCollectionDefault, systest.WithArbOS(params.ArbosVersion_60)),
	systest.Test(testRunTipCollectionEnabled, systest.WithArbOS(params.ArbosVersion_60)),
	systest.Test(testRunTipCollectionPreV60, systest.MatrixArbOS(params.ArbosVersion_9, params.ArbosVersion_51)),
	systest.Test(testRunTipCollectionDisableAfterEnable, systest.WithArbOS(params.ArbosVersion_60)),
	systest.Test(testRunTipCollectionDelayedInboxDropsTips, systest.WithArbOS(params.ArbosVersion_60), systest.WithL1()),
	systest.Test(testRunTipCollectionGetPaidGasPrice, systest.WithArbOS(params.ArbosVersion_60)),
	systest.Test(testRunTipCollectionPrecompileVersionGating, systest.WithArbOS(params.ArbosVersion_51)),
}

// testRunTipCollectionDefault verifies that by default (collect=false), tips are dropped on v60.
func testRunTipCollectionDefault(env *systest.Env) {
	baseFee := chainBaseFee(env)

	setCollectTips(env, false)
	env.True(!collectTipsEnabled(env), "expected collect tips to be false")

	tip := big.NewInt(41)
	assertTipDropped(env, baseFee, tip, "collect=false")
}

// testRunTipCollectionEnabled verifies that enabling tip collection collects tips of any size.
func testRunTipCollectionEnabled(env *systest.Env) {
	baseFee := chainBaseFee(env)

	setCollectTips(env, true)
	env.True(collectTipsEnabled(env), "expected collect tips to be true")

	// zero tip collects nothing; for tip 0 collected == dropped
	for _, tip := range []int64{0, 2, 5, 10} {
		assertTipCollected(env, baseFee, big.NewInt(tip), "tip "+big.NewInt(tip).String())
	}
}

// testRunTipCollectionPreV60 verifies fixed pre-v60 tip handling: v9 collects, later versions drop.
func testRunTipCollectionPreV60(env *systest.Env) {
	baseFee := chainBaseFee(env)

	tip := big.NewInt(10)
	if env.Spec.ArbOSVersion.Unwrap() <= params.ArbosVersion_9 {
		assertTipCollected(env, baseFee, tip, "v9")
	} else {
		assertTipDropped(env, baseFee, tip, "pre-v60")
	}
}

// testRunTipCollectionDisableAfterEnable verifies that disabling tip collection stops collecting tips.
func testRunTipCollectionDisableAfterEnable(env *systest.Env) {
	baseFee := chainBaseFee(env)

	setCollectTips(env, true)
	setCollectTips(env, false)
	env.True(!collectTipsEnabled(env), "expected collect tips to be false after disable")

	tip := big.NewInt(10)
	assertTipDropped(env, baseFee, tip, "collect disabled")
}

// testRunTipCollectionDelayedInboxDropsTips verifies that delayed inbox messages always drop tips,
// even when tip collection is enabled.
func testRunTipCollectionDelayedInboxDropsTips(env *systest.Env) {
	baseFee := chainBaseFee(env)
	networkFeeAddr := networkFeeAccount(env)

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, env.L2.Client)
	env.Require(err)
	ownerAuth := env.L2.TransactOpts("Owner")
	// Zero the reward rate so L1 pricing rewards can't skew the network-fee delta
	rewardTx, err := arbOwner.SetL1PricingRewardRate(&ownerAuth, 0)
	env.Require(err, "SetL1PricingRewardRate")
	env.L2.EnsureTxSucceeded(rewardTx)

	// Enable tip collection so direct txs would collect tips
	setCollectTips(env, true)

	// Prepare a delayed tx with a tip
	tipCap := baseFee
	// Use a high gasFeeCap to account for baseFee drift between measurement and sequencing
	gasFeeCap := new(big.Int).Mul(baseFee, big.NewInt(5))
	info := env.L2.Info.GetInfoWithPrivKey("Owner")
	delayedTx := env.L2.Info.SignTxAs("Owner", &types.DynamicFeeTx{
		To:        &info.Address,
		Gas:       env.L2.Info.TransferGas,
		GasTipCap: tipCap,
		GasFeeCap: gasFeeCap,
		Value:     big.NewInt(1),
		Nonce:     info.Nonce.Add(1) - 1,
	})

	networkBefore := env.L2.BalanceAt(networkFeeAddr)
	receipt := env.SendSignedTxViaL1(delayedTx)

	networkAfter := env.L2.BalanceAt(networkFeeAddr)
	revenue := new(big.Int).Sub(networkAfter, networkBefore)
	expected := expectedRevenue(baseFee, receipt)
	env.EqualBig(expected, revenue, "delayed inbox: tip should be dropped")

	// Verify the receipt's EffectiveGasPrice equals baseFee (not baseFee+tip).
	// This confirms that the block-level CollectTips header flag is false for delayed blocks,
	// which is needed for correct receipt re-derivation.
	blockBaseFee := env.L2.HeaderByNumber(receipt.BlockNumber).BaseFee
	env.EqualBig(blockBaseFee, receipt.EffectiveGasPrice, "delayed inbox: EffectiveGasPrice should equal baseFee")
}

// testRunTipCollectionGetPaidGasPrice verifies that the GASPRICE opcode (which calls
// GetPaidGasPrice) returns the full gas price when tips are collected and
// only the base fee when tips are dropped.
func testRunTipCollectionGetPaidGasPrice(env *systest.Env) {
	baseFee := chainBaseFee(env)

	// Deploy a contract that stores tx.gasprice in slot 0: GASPRICE PUSH1(0) SSTORE STOP
	runtimeCode := []byte{byte(vm.GASPRICE), byte(vm.PUSH1), 0, byte(vm.SSTORE), byte(vm.STOP)}
	deployCode := program.New().ReturnViaCodeCopy(runtimeCode).Bytes()
	deployGas := env.L2.EstimateGas(ethereum.CallMsg{From: env.L2.Info.GetAddress("Faucet"), Data: deployCode})
	deployTx := env.L2.Info.PrepareTxTo("Faucet", nil, deployGas, nil, deployCode)
	env.L2.SendTx(deployTx)
	contractAddr := env.L2.EnsureTxSucceeded(deployTx).ContractAddress

	callContract := func(tipCap *big.Int) *big.Int {
		info := env.L2.Info.GetInfoWithPrivKey("Faucet")
		gasFeeCap := new(big.Int).Add(baseFee, tipCap)
		tx := env.L2.Info.SignTxAs("Faucet", &types.DynamicFeeTx{
			To:        &contractAddr,
			Gas:       env.L2.Info.TransferGas,
			GasTipCap: tipCap,
			GasFeeCap: gasFeeCap,
			Nonce:     info.Nonce.Add(1) - 1,
		})
		env.L2.SendTx(tx)
		env.L2.EnsureTxSucceeded(tx)
		slot0, err := env.L2.Client.StorageAt(env.Ctx, contractAddr, common.Hash{}, nil)
		env.Require(err, "StorageAt")
		return new(big.Int).SetBytes(slot0)
	}

	tip := big.NewInt(50)

	// With collect=false, tips are dropped: GASPRICE should return baseFee
	setCollectTips(env, false)
	env.EqualBig(baseFee, callContract(tip), "collect=false: GASPRICE should equal baseFee")

	// With collect=true, tips are collected: GASPRICE should return baseFee + tip
	setCollectTips(env, true)
	expectedPrice := new(big.Int).Add(baseFee, tip)
	env.EqualBig(expectedPrice, callContract(tip), "collect=true: GASPRICE should equal baseFee+tip")
}

// testRunTipCollectionPrecompileVersionGating verifies that SetCollectTips and
// GetCollectTips revert on pre-v60 chains.
func testRunTipCollectionPrecompileVersionGating(env *systest.Env) {
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, env.L2.Client)
	env.Require(err)
	auth := env.L2.TransactOpts("Owner")
	_, err = arbOwner.SetCollectTips(&auth, true)
	env.NotNil(err, "SetCollectTips should revert on pre-v60")

	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(types.ArbOwnerPublicAddress, env.L2.Client)
	env.Require(err)
	_, err = arbOwnerPublic.GetCollectTips(&bind.CallOpts{Context: env.Ctx})
	env.NotNil(err, "GetCollectTips should revert on pre-v60")
}

func chainBaseFee(env *systest.Env) *big.Int {
	baseFee := env.L2.HeaderByNumber(nil).BaseFee
	env.True(baseFee.Sign() > 0, "base fee should not be 0")
	return baseFee
}

func networkFeeAccount(env *systest.Env) common.Address {
	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(types.ArbOwnerPublicAddress, env.L2.Client)
	env.Require(err)
	addr, err := arbOwnerPublic.GetNetworkFeeAccount(&bind.CallOpts{Context: env.Ctx})
	env.Require(err, "GetNetworkFeeAccount")
	return addr
}

func setCollectTips(env *systest.Env, collect bool) {
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, env.L2.Client)
	env.Require(err)
	auth := env.L2.TransactOpts("Owner")
	tx, err := arbOwner.SetCollectTips(&auth, collect)
	env.Require(err, "SetCollectTips")
	env.L2.EnsureTxSucceeded(tx)
}

func collectTipsEnabled(env *systest.Env) bool {
	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(types.ArbOwnerPublicAddress, env.L2.Client)
	env.Require(err)
	collect, err := arbOwnerPublic.GetCollectTips(&bind.CallOpts{Context: env.Ctx})
	env.Require(err, "GetCollectTips")
	return collect
}

// sendTxWithTip sends a simple ETH transfer with the given tip and returns the
// network fee account revenue delta and the receipt.
func sendTxWithTip(env *systest.Env, baseFee, tip *big.Int) (*big.Int, *types.Receipt) {
	networkFeeAddr := networkFeeAccount(env)
	networkBefore := env.L2.BalanceAt(networkFeeAddr)

	info := env.L2.Info.GetInfoWithPrivKey("Faucet")
	gasFeeCap := new(big.Int).Add(baseFee, tip)
	tx := env.L2.Info.SignTxAs("Faucet", &types.DynamicFeeTx{
		To:        &info.Address,
		Gas:       env.L2.Info.TransferGas,
		GasTipCap: tip,
		GasFeeCap: gasFeeCap,
		Value:     big.NewInt(1),
		Nonce:     info.Nonce.Add(1) - 1,
	})
	env.L2.SendTx(tx)
	receipt := env.L2.EnsureTxSucceeded(tx)

	networkAfter := env.L2.BalanceAt(networkFeeAddr)
	revenue := new(big.Int).Sub(networkAfter, networkBefore)
	return revenue, receipt
}

// expectedRevenue returns the expected network fee account revenue for a receipt
// given the effective gas price: gasPrice * l2GasUsed.
func expectedRevenue(gasPrice *big.Int, receipt *types.Receipt) *big.Int {
	l2GasUsed := receipt.GasUsed - receipt.GasUsedForL1
	return arbmath.BigMulByUint(gasPrice, l2GasUsed)
}

func assertTipDropped(env *systest.Env, baseFee, tip *big.Int, context string) {
	revenue, receipt := sendTxWithTip(env, baseFee, tip)
	expected := expectedRevenue(baseFee, receipt)
	env.EqualBig(baseFee, receipt.EffectiveGasPrice, "%s: incorrect receipt.EffectiveGasPrice", context)
	env.EqualBig(expected, revenue, "%s: tip should be dropped", context)
}

func assertTipCollected(env *systest.Env, baseFee, tip *big.Int, context string) {
	revenue, receipt := sendTxWithTip(env, baseFee, tip)
	gasPrice := new(big.Int).Add(baseFee, tip)
	expected := expectedRevenue(gasPrice, receipt)
	env.EqualBig(gasPrice, receipt.EffectiveGasPrice, "%s: incorrect receipt.EffectiveGasPrice", context)
	env.EqualBig(expected, revenue, "%s: tip should be collected", context)
}
