// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package v2

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func init() {
	// Unconstrained L2 tests — support any scheme/engine/version.
	RegisterTest("TestArbStatistics", L2Light(), testRunArbStatistics)
	RegisterTest("TestArbAggregatorGetPreferredAggregator", L2Light(), testRunArbAggregatorGetPreferredAggregator)
	RegisterTest("TestFeeAccounts", L2Light(), testRunFeeAccounts)
	RegisterTest("TestChainOwners", L2Light(), testRunChainOwners)

	// Needs minimum ArbOS version — demonstrates MinArbOSVersion filtering.
	RegisterTest("TestGetBrotliCompressionLevel", L2WithMinArbOS(params.ArbosVersion_20), testRunGetBrotliCompressionLevel)

	// Pinned ArbOS version — demonstrates PinArbOSVersion.
	RegisterTest("TestPurePrecompileMethodCalls", testConfigPinArbOS31, testRunPurePrecompileMethodCalls)

	// Hash-scheme only — demonstrates Schemes constraint.
	RegisterTest("TestL1BaseFeeEstimateInertia", testConfigHashSchemeOnly, testRunL1BaseFeeEstimateInertia)
}

// =========================================================================
// Configs unique to precompile tests
// =========================================================================

// testConfigPinArbOS31: must use exactly ArbOS 31.
func testConfigPinArbOS31() []*BuilderSpec {
	return []*BuilderSpec{{
		Weight:          WeightLight,
		Parallelizable:  true,
		PinArbOSVersion: params.ArbosVersion_31,
	}}
}

// testConfigHashSchemeOnly: only supports hash scheme.
// When CLI says --v2.matrix.state-scheme=hash,path, this runs for hash only.
func testConfigHashSchemeOnly() []*BuilderSpec {
	return []*BuilderSpec{{
		Weight:         WeightLight,
		Parallelizable: true,
		Schemes:        []string{"hash"},
	}}
}

// =========================================================================
// Test implementations
// =========================================================================

func testRunArbStatistics(env *TestEnv) {
	arbStatistics, err := precompilesgen.NewArbStatistics(types.ArbStatisticsAddress, env.L2.Client)
	env.Require(err)
	callOpts := &bind.CallOpts{Context: env.Ctx}
	blockNum, _, _, _, _, _, err := arbStatistics.GetStats(callOpts)
	env.Require(err)
	expectedBlockNum, err := env.L2.Client.BlockNumber(env.Ctx)
	env.Require(err)
	if blockNum.Uint64() != expectedBlockNum {
		env.Fatal("expected block number", expectedBlockNum, "got", blockNum)
	}
}

func testRunArbAggregatorGetPreferredAggregator(env *TestEnv) {
	callOpts := &bind.CallOpts{Context: env.Ctx}
	arbAggregator, err := precompilesgen.NewArbAggregator(types.ArbAggregatorAddress, env.L2.Client)
	env.Require(err)
	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	prefAgg, isDefault, err := arbAggregator.GetPreferredAggregator(callOpts, addr)
	env.Require(err)
	if !isDefault {
		env.Fatal("expected default preferred aggregator")
	}
	if prefAgg != l1pricing.BatchPosterAddress {
		env.Fatal("expected", l1pricing.BatchPosterAddress, "got", prefAgg)
	}
}

func testRunFeeAccounts(env *TestEnv) {
	auth := env.GetDefaultTransactOpts("Owner")
	callOpts := &bind.CallOpts{Context: env.Ctx}
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, env.L2.Client)
	env.Require(err)

	env.L2Info.GenerateAccount("User2")
	addr := env.L2Info.GetAddress("User2")

	tx, err := arbOwner.SetNetworkFeeAccount(&auth, addr)
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	feeAccount, err := arbOwner.GetNetworkFeeAccount(callOpts)
	env.Require(err)
	if feeAccount.Cmp(addr) != 0 {
		env.Fatal("expected fee account", addr, "got", feeAccount)
	}

	tx, err = arbOwner.SetInfraFeeAccount(&auth, addr)
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	feeAccount, err = arbOwner.GetInfraFeeAccount(callOpts)
	env.Require(err)
	if feeAccount.Cmp(addr) != 0 {
		env.Fatal("expected infra fee account", addr, "got", feeAccount)
	}
}

func testRunChainOwners(env *TestEnv) {
	auth := env.GetDefaultTransactOpts("Owner")
	callOpts := &bind.CallOpts{Context: env.Ctx}
	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(types.ArbOwnerPublicAddress, env.L2.Client)
	env.Require(err)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, env.L2.Client)
	env.Require(err)

	env.L2Info.GenerateAccount("Owner2")
	chainOwnerAddr2 := env.L2Info.GetAddress("Owner2")
	tx, err := arbOwner.AddChainOwner(&auth, chainOwnerAddr2)
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	isChainOwner, err := arbOwnerPublic.IsChainOwner(callOpts, chainOwnerAddr2)
	env.Require(err)
	if !isChainOwner {
		env.Fatal("expected owner2 to be a chain owner")
	}

	tx, err = arbOwner.RemoveChainOwner(&auth, chainOwnerAddr2)
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	isChainOwner, err = arbOwnerPublic.IsChainOwner(callOpts, chainOwnerAddr2)
	env.Require(err)
	if isChainOwner {
		env.Fatal("expected owner2 to not be a chain owner")
	}
}

func testRunGetBrotliCompressionLevel(env *TestEnv) {
	auth := env.GetDefaultTransactOpts("Owner")
	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(types.ArbOwnerPublicAddress, env.L2.Client)
	env.Require(err)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, env.L2.Client)
	env.Require(err)

	brotliLevel := uint64(11)
	tx, err := arbOwner.SetBrotliCompressionLevel(&auth, brotliLevel)
	env.Require(err)
	env.EnsureTxSucceeded(tx)

	callOpts := &bind.CallOpts{Context: env.Ctx}
	got, err := arbOwnerPublic.GetBrotliCompressionLevel(callOpts)
	env.Require(err)
	if got != brotliLevel {
		env.Fatal("expected brotli level", brotliLevel, "got", got)
	}
}

func testRunPurePrecompileMethodCalls(env *TestEnv) {
	arbSys, err := precompilesgen.NewArbSys(common.HexToAddress("0x64"), env.L2.Client)
	env.Require(err)
	chainId, err := arbSys.ArbChainID(&bind.CallOpts{})
	env.Require(err)
	if chainId.Uint64() != 412346 {
		env.Fatal("wrong ChainID", chainId.Uint64())
	}

	storageGas, err := arbSys.GetStorageGasAvailable(&bind.CallOpts{})
	env.Require(err)
	if storageGas.Cmp(big.NewInt(0)) != 0 {
		env.Fatal("expected 0 storage gas available, got", storageGas)
	}
}

func testRunL1BaseFeeEstimateInertia(env *TestEnv) {
	auth := env.GetDefaultTransactOpts("Owner")
	arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), env.L2.Client)
	env.Require(err)
	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), env.L2.Client)
	env.Require(err)

	inertia := uint64(11)
	tx, err := arbOwner.SetL1BaseFeeEstimateInertia(&auth, inertia)
	env.Require(err)
	env.EnsureTxSucceeded(tx)

	got, err := arbGasInfo.GetL1BaseFeeEstimateInertia(&bind.CallOpts{Context: env.Ctx})
	env.Require(err)
	if got != inertia {
		env.Fatal("expected inertia", inertia, "got", got)
	}
}
