// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package v2

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func init() {
	// Simple read-only / error tests
	RegisterTest("TestViewLogReverts", testConfigL2Light, testRunViewLogReverts)
	RegisterTest("TestArbDebugPanic", testConfigL2Light, testRunArbDebugPanic)
	RegisterTest("TestArbDebugLegacyError", testConfigL2Light, testRunArbDebugLegacyError)
	RegisterTest("TestCustomSolidityErrors", testConfigL2Light, testRunCustomSolidityErrors)
	RegisterTest("TestPrecompileErrorGasLeft", testConfigL2Light, testRunPrecompileErrorGasLeft)
	RegisterTest("TestArbStatistics", testConfigL2Light, testRunArbStatistics)
	RegisterTest("TestCurrentTxL1GasFees", testConfigL2Light, testRunCurrentTxL1GasFees)
	RegisterTest("TestArbAggregatorGetPreferredAggregator", testConfigL2Light, testRunArbAggregatorGetPreferredAggregator)

	// Tests that send transactions
	RegisterTest("TestGetBrotliCompressionLevel", testConfigMinArbOS20, testRunGetBrotliCompressionLevel)
	RegisterTest("TestArbosFeatures", testConfigMinArbOS40, testRunArbosFeatures)
	RegisterTest("TestArbFunctionTable", testConfigL2Light, testRunArbFunctionTable)
	RegisterTest("TestArbAggregatorBaseFee", testConfigL2Light, testRunArbAggregatorBaseFee)
	RegisterTest("TestFeeAccounts", testConfigL2Light, testRunFeeAccounts)
	RegisterTest("TestChainOwners", testConfigL2Light, testRunChainOwners)
	RegisterTest("TestArbAggregatorBatchPosters", testConfigL2Light, testRunArbAggregatorBatchPosters)
	RegisterTest("TestArbDebugOverwriteContractCode", testConfigL2Light, testRunArbDebugOverwriteContractCode)

	// Tests that need a specific ArbOS version
	RegisterTest("TestPurePrecompileMethodCalls", testConfigPurePrecompileMethodCalls, testRunPurePrecompileMethodCalls)
	RegisterTest("TestNativeTokenManagementNotAvailableBeforeArbos41", testConfigNativeTokenNotAvailable, testRunNativeTokenManagementNotAvailableBeforeArbos41)

	// setupArbOwnerAndArbGasInfo cluster (all share the same config)
	RegisterTest("TestL1BaseFeeEstimateInertia", testConfigL2Light, testRunL1BaseFeeEstimateInertia)
	RegisterTest("TestL1PricingInertia", testConfigL2Light, testRunL1PricingInertia)
	RegisterTest("TestL1PricingRewardRate", testConfigL2Light, testRunL1PricingRewardRate)
	RegisterTest("TestL1PricingRewardRecipient", testConfigL2Light, testRunL1PricingRewardRecipient)
	RegisterTest("TestL2GasPricingInertia", testConfigL2Light, testRunL2GasPricingInertia)
	RegisterTest("TestL2GasBacklogTolerance", testConfigL2Light, testRunL2GasBacklogTolerance)
	RegisterTest("TestPerBatchGasCharge", testConfigL2Light, testRunPerBatchGasCharge)
	RegisterTest("TestL1PricingEquilibrationUnits", testConfigL2Light, testRunL1PricingEquilibrationUnits)
	RegisterTest("TestGasAccountingParams", testConfigMinArbOS50, testRunGasAccountingParams)
}

// --- Shared configs ---

// testConfigL2Light returns a single lightweight L2-only spec.
// It has no version constraints — compatible with any TestParams.
func testConfigL2Light(_ TestParams) []*BuilderSpec {
	return []*BuilderSpec{{
		Weight:         WeightLight,
		Parallelizable: true,
	}}
}

// testConfigMinArbOS20 is like testConfigL2Light but requires ArbOS >= 20.
func testConfigMinArbOS20(_ TestParams) []*BuilderSpec {
	return []*BuilderSpec{{
		Weight:          WeightLight,
		Parallelizable:  true,
		MinArbOSVersion: params.ArbosVersion_20,
	}}
}

// testConfigMinArbOS40 is like testConfigL2Light but requires ArbOS >= 40.
func testConfigMinArbOS40(_ TestParams) []*BuilderSpec {
	return []*BuilderSpec{{
		Weight:          WeightLight,
		Parallelizable:  true,
		MinArbOSVersion: params.ArbosVersion_40,
	}}
}

// testConfigMinArbOS50 is like testConfigL2Light but requires ArbOS >= 50.
func testConfigMinArbOS50(_ TestParams) []*BuilderSpec {
	return []*BuilderSpec{{
		Weight:          WeightLight,
		Parallelizable:  true,
		MinArbOSVersion: params.ArbosVersion_50,
	}}
}

// --- Shared helpers ---

// setupArbOwnerAndArbGasInfo binds ArbOwner + ArbGasInfo and returns an owner auth.
func setupArbOwnerAndArbGasInfo(env *TestEnv) (bind.TransactOpts, *precompilesgen.ArbOwner, *precompilesgen.ArbGasInfo) {
	auth := env.GetDefaultTransactOpts("Owner")
	arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), env.L2.Client)
	env.Require(err)
	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), env.L2.Client)
	env.Require(err)
	return auth, arbOwner, arbGasInfo
}

// ============================================================================
// Simple read-only / error tests
// ============================================================================

func testRunViewLogReverts(env *TestEnv) {
	arbDebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), env.L2.Client)
	env.Require(err, "could not bind ArbDebug")
	err = arbDebug.EventsView(nil)
	if err == nil {
		env.Fatal("unexpected success from EventsView")
	}
}

func testRunArbDebugPanic(env *TestEnv) {
	auth := env.GetDefaultTransactOpts("Owner")
	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, env.L2.Client)
	env.Require(err)
	_, err = arbDebug.Panic(&auth)
	if err == nil {
		env.Fatal("unexpected success from Panic")
	}
	if err.Error() != "method handler crashed" {
		env.Fatal("expected 'method handler crashed', got:", err)
	}
}

func testRunArbDebugLegacyError(env *TestEnv) {
	callOpts := &bind.CallOpts{Context: env.Ctx}
	arbDebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), env.L2.Client)
	env.Require(err)
	err = arbDebug.LegacyError(callOpts)
	if err == nil {
		env.Fatal("unexpected success from LegacyError")
	}
}

func testRunCustomSolidityErrors(env *TestEnv) {
	t := env.T
	callOpts := &bind.CallOpts{Context: env.Ctx}
	auth := env.GetDefaultTransactOpts("Owner")

	ensure := func(customError error, expectedError, scenario string) {
		t.Helper()
		if customError == nil {
			t.Fatal("should have errored", "scenario", scenario)
		}
		observed := customError.Error()
		expected := fmt.Sprintf("execution reverted: error %v: %v", expectedError, expectedError)
		if observed != expected {
			t.Fatal(observed, "scenario", scenario)
		}
	}

	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, env.L2.Client)
	env.Require(err, "could not bind ArbDebug contract")
	ensure(
		arbDebug.CustomRevert(callOpts, 1024),
		"Custom(1024, This spider family wards off bugs: /\\oo/\\ //\\(oo)//\\ /\\oo/\\, true)",
		"arbDebug.CustomRevert",
	)

	arbSys, err := precompilesgen.NewArbSys(arbos.ArbSysAddress, env.L2.Client)
	env.Require(err, "could not bind ArbSys contract")
	_, customError := arbSys.ArbBlockHash(callOpts, big.NewInt(1e9))
	ensure(customError, "InvalidBlockNumber(1000000000, 1)", "arbSys.ArbBlockHash")

	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(types.ArbRetryableTxAddress, env.L2.Client)
	env.Require(err)
	_, customError = arbRetryableTx.SubmitRetryable(
		&auth, [32]byte{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0),
		0, big.NewInt(0), common.Address{}, common.Address{}, common.Address{}, []byte{},
	)
	ensure(customError, "NotCallable()", "arbRetryableTx.SubmitRetryable")

	arbosActs, err := precompilesgen.NewArbosActs(types.ArbosAddress, env.L2.Client)
	env.Require(err)
	_, customError = arbosActs.StartBlock(&auth, big.NewInt(0), 0, 0, 0)
	ensure(customError, "CallerNotArbOS()", "arbosActs.StartBlock")
	_, customError = arbosActs.BatchPostingReport(&auth, big.NewInt(0), common.Address{}, 0, 0, big.NewInt(0))
	ensure(customError, "CallerNotArbOS()", "arbosActs.BatchPostingReport")
	_, customError = arbosActs.BatchPostingReportV2(&auth, big.NewInt(0), common.Address{}, 0, 0, 0, 0, big.NewInt(0))
	ensure(customError, "CallerNotArbOS()", "arbosActs.BatchPostingReportV2")
}

func testRunPrecompileErrorGasLeft(env *TestEnv) {
	auth := env.GetDefaultTransactOpts("Faucet")
	_, _, simple, err := localgen.DeploySimple(&auth, env.L2.Client)
	env.Require(err)

	assertNotAllGasConsumed := func(to common.Address, input []byte) {
		gas, err := simple.CheckGasUsed(&bind.CallOpts{Context: env.Ctx}, to, input)
		env.Require(err, "Failed to call CheckGasUsed to precompile", to)
		maxGas := big.NewInt(100_000)
		if arbmath.BigGreaterThan(gas, maxGas) {
			env.Fatal("Precompile", to, "used", gas, "gas reverting, greater than max expected", maxGas)
		}
	}

	arbSysABI, err := precompilesgen.ArbSysMetaData.GetAbi()
	env.Require(err)
	arbBlockHash := arbSysABI.Methods["arbBlockHash"]
	data, err := arbBlockHash.Inputs.Pack(big.NewInt(1e9))
	env.Require(err)
	input := append([]byte{}, arbBlockHash.ID...)
	input = append(input, data...)
	assertNotAllGasConsumed(arbos.ArbSysAddress, input)

	arbDebugABI, err := precompilesgen.ArbDebugMetaData.GetAbi()
	env.Require(err)
	assertNotAllGasConsumed(common.HexToAddress("0xff"), arbDebugABI.Methods["legacyError"].ID)
}

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

func testRunCurrentTxL1GasFees(env *TestEnv) {
	arbGasInfo, err := precompilesgen.NewArbGasInfo(types.ArbGasInfoAddress, env.L2.Client)
	env.Require(err)
	currTxL1GasFees, err := arbGasInfo.GetCurrentTxL1GasFees(&bind.CallOpts{Context: env.Ctx})
	env.Require(err)
	if currTxL1GasFees == nil {
		env.Fatal("currTxL1GasFees is nil")
	}
	if currTxL1GasFees.Cmp(big.NewInt(0)) != 1 {
		env.Fatal("expected currTxL1GasFees to be greater than 0, got", currTxL1GasFees)
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
		env.Fatal("expected default preferred aggregator to be", l1pricing.BatchPosterAddress, "got", prefAgg)
	}
	prefAgg, err = arbAggregator.GetDefaultAggregator(callOpts)
	env.Require(err)
	if prefAgg != l1pricing.BatchPosterAddress {
		env.Fatal("expected default preferred aggregator to be", l1pricing.BatchPosterAddress, "got", prefAgg)
	}
}

// ============================================================================
// Tests that send transactions
// ============================================================================

func testRunGetBrotliCompressionLevel(env *TestEnv) {
	auth := env.GetDefaultTransactOpts("Owner")
	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(types.ArbOwnerPublicAddress, env.L2.Client)
	env.Require(err)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, env.L2.Client)
	env.Require(err)

	brotliCompressionLevel := uint64(11)
	tx, err := arbOwner.SetBrotliCompressionLevel(&auth, brotliCompressionLevel)
	env.Require(err)
	env.EnsureTxSucceeded(tx)

	callOpts := &bind.CallOpts{Context: env.Ctx}
	retrieved, err := arbOwnerPublic.GetBrotliCompressionLevel(callOpts)
	env.Require(err)
	if retrieved != brotliCompressionLevel {
		env.Fatal("expected brotli compression level to be", brotliCompressionLevel, "got", retrieved)
	}
}

func testRunArbosFeatures(env *TestEnv) {
	auth := env.GetDefaultTransactOpts("Owner")
	callOpts := &bind.CallOpts{Context: env.Ctx}
	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(types.ArbOwnerPublicAddress, env.L2.Client)
	env.Require(err)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, env.L2.Client)
	env.Require(err)

	isCDPIEnabled, err := arbOwnerPublic.IsCalldataPriceIncreaseEnabled(callOpts)
	env.Require(err)
	if isCDPIEnabled {
		env.Fatal("expected calldata price increase to be disabled")
	}

	tx, err := arbOwner.SetCalldataPriceIncrease(&auth, true)
	env.Require(err)
	env.EnsureTxSucceeded(tx)

	isCDPIEnabled, err = arbOwnerPublic.IsCalldataPriceIncreaseEnabled(callOpts)
	env.Require(err)
	if !isCDPIEnabled {
		env.Fatal("expected calldata price increase to be enabled")
	}
}

func testRunArbFunctionTable(env *TestEnv) {
	auth := env.GetDefaultTransactOpts("Owner")
	callOpts := &bind.CallOpts{Context: env.Ctx}
	arbFunctionTable, err := precompilesgen.NewArbFunctionTable(types.ArbFunctionTableAddress, env.L2.Client)
	env.Require(err)

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	// should be a noop
	tx, err := arbFunctionTable.Upload(&auth, []byte{0, 0, 0, 0})
	env.Require(err)
	env.EnsureTxSucceeded(tx)

	size, err := arbFunctionTable.Size(callOpts, addr)
	env.Require(err)
	if size.Cmp(big.NewInt(0)) != 0 {
		env.Fatal("Size should be 0")
	}

	_, _, _, err = arbFunctionTable.Get(callOpts, addr, big.NewInt(10))
	if err == nil {
		env.Fatal("Should error")
	}
}

func testRunArbAggregatorBaseFee(env *TestEnv) {
	auth := env.GetDefaultTransactOpts("Owner")
	callOpts := &bind.CallOpts{Context: env.Ctx}
	arbAggregator, err := precompilesgen.NewArbAggregator(types.ArbAggregatorAddress, env.L2.Client)
	env.Require(err)

	tx, err := arbAggregator.SetTxBaseFee(&auth, common.Address{}, big.NewInt(1))
	env.Require(err)
	env.EnsureTxSucceeded(tx)

	fee, err := arbAggregator.GetTxBaseFee(callOpts, common.Address{})
	env.Require(err)
	if fee.Cmp(big.NewInt(0)) != 0 {
		env.Fatal("expected fee to be 0, got", fee)
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
		env.Fatal("expected fee account to be", addr, "got", feeAccount)
	}

	tx, err = arbOwner.SetInfraFeeAccount(&auth, addr)
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	feeAccount, err = arbOwner.GetInfraFeeAccount(callOpts)
	env.Require(err)
	if feeAccount.Cmp(addr) != 0 {
		env.Fatal("expected fee account to be", addr, "got", feeAccount)
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

	chainOwnersPublic, err := arbOwnerPublic.GetAllChainOwners(callOpts)
	env.Require(err)
	chainOwnersOwner, err := arbOwner.GetAllChainOwners(callOpts)
	env.Require(err)
	if len(chainOwnersPublic) != len(chainOwnersOwner) {
		env.Fatal("expected chain owners to be the same length")
	}
	sort.Slice(chainOwnersPublic, func(i, j int) bool {
		return chainOwnersPublic[i].Cmp(chainOwnersPublic[j]) < 0
	})
	for i := range chainOwnersPublic {
		if chainOwnersPublic[i].Cmp(chainOwnersOwner[i]) != 0 {
			env.Fatal("expected chain owners to be the same")
		}
	}
	chainOwnerAddr := env.L2Info.GetAddress("Owner")
	found := false
	for _, co := range chainOwnersOwner {
		if co.Cmp(chainOwnerAddr) == 0 {
			found = true
		}
	}
	if !found {
		env.Fatal("expected owner to be in chain owners")
	}

	tx, err = arbOwner.RemoveChainOwner(&auth, chainOwnerAddr2)
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	isChainOwner, err = arbOwnerPublic.IsChainOwner(callOpts, chainOwnerAddr2)
	env.Require(err)
	if isChainOwner {
		env.Fatal("expected owner2 to not be a chain owner")
	}

	_, err = arbOwnerPublic.RectifyChainOwner(&auth, chainOwnerAddr)
	if err == nil || err.Error() != "execution reverted" {
		env.Fatal("expected rectify chain owner to revert since it is already an owner")
	}
}

func testRunArbAggregatorBatchPosters(env *TestEnv) {
	auth := env.GetDefaultTransactOpts("Owner")
	callOpts := &bind.CallOpts{Context: env.Ctx}
	arbAggregator, err := precompilesgen.NewArbAggregator(types.ArbAggregatorAddress, env.L2.Client)
	env.Require(err)
	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, env.L2.Client)
	env.Require(err)

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	bps, err := arbAggregator.GetBatchPosters(callOpts)
	env.Require(err)
	if len(bps) != 1 {
		env.Fatal("expected one batch poster")
	}

	tx, err := arbDebug.BecomeChainOwner(&auth)
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	tx, err = arbAggregator.AddBatchPoster(&auth, addr)
	env.Require(err)
	env.EnsureTxSucceeded(tx)

	bps, err = arbAggregator.GetBatchPosters(callOpts)
	env.Require(err)
	if len(bps) != 2 {
		env.Fatal("expected two batch posters")
	}
	if bps[0] != addr && bps[1] != addr {
		env.Fatal("expected addr to be a batch poster")
	}
}

func testRunArbDebugOverwriteContractCode(env *TestEnv) {
	auth := env.GetDefaultTransactOpts("Owner")
	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, env.L2.Client)
	env.Require(err)
	tx, err := arbDebug.BecomeChainOwner(&auth)
	env.Require(err)
	env.EnsureTxSucceeded(tx)

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	code, err := env.L2.Client.CodeAt(env.Ctx, addr, nil)
	env.Require(err)
	if len(code) != 0 {
		env.Fatal("expected code to be empty")
	}

	testCodeA := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	tx, err = arbDebug.OverwriteContractCode(&auth, addr, testCodeA)
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	code, err = env.L2.Client.CodeAt(env.Ctx, addr, nil)
	env.Require(err)
	if !bytes.Equal(code, testCodeA) {
		env.Fatal("expected code A to be", testCodeA, "got", code)
	}

	testCodeB := []byte{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	tx, err = arbDebug.OverwriteContractCode(&auth, addr, testCodeB)
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	code, err = env.L2.Client.CodeAt(env.Ctx, addr, nil)
	env.Require(err)
	if !bytes.Equal(code, testCodeB) {
		env.Fatal("expected code B to be", testCodeB, "got", code)
	}
}

// ============================================================================
// Tests that need a specific ArbOS version
// ============================================================================

func testConfigPurePrecompileMethodCalls(p TestParams) []*BuilderSpec {
	// This test needs ArbOS 31 specifically. If params pin a different version, skip.
	if p.ArbOSVersion != nil && *p.ArbOSVersion != params.ArbosVersion_31 {
		return nil
	}
	return []*BuilderSpec{{
		Weight:         WeightLight,
		Parallelizable: true,
		ArbOSVersion:   params.ArbosVersion_31,
	}}
}

func testRunPurePrecompileMethodCalls(env *TestEnv) {
	arbSys, err := precompilesgen.NewArbSys(common.HexToAddress("0x64"), env.L2.Client)
	env.Require(err, "could not deploy ArbSys contract")
	chainId, err := arbSys.ArbChainID(&bind.CallOpts{})
	env.Require(err, "failed to get the ChainID")
	if chainId.Uint64() != chaininfo.ArbitrumDevTestChainConfig().ChainID.Uint64() {
		env.Fatal("Wrong ChainID", chainId.Uint64())
	}

	arbosVersion := params.ArbosVersion_31
	expectedArbosVersion := 55 + arbosVersion // Nitro versions start at 56
	arbSysArbosVersion, err := arbSys.ArbOSVersion(&bind.CallOpts{})
	env.Require(err)
	if arbSysArbosVersion.Uint64() != expectedArbosVersion {
		env.Fatal("Expected ArbOS version", expectedArbosVersion, "got", arbSysArbosVersion)
	}

	storageGasAvailable, err := arbSys.GetStorageGasAvailable(&bind.CallOpts{})
	env.Require(err)
	if storageGasAvailable.Cmp(big.NewInt(0)) != 0 {
		env.Fatal("Expected 0 storage gas available, got", storageGasAvailable)
	}
}

func testConfigNativeTokenNotAvailable(p TestParams) []*BuilderSpec {
	// This test needs exactly ArbOS 40 (pre-41). If params pin a different version, skip.
	if p.ArbOSVersion != nil && *p.ArbOSVersion != params.ArbosVersion_40 {
		return nil
	}
	return []*BuilderSpec{{
		Weight:         WeightLight,
		Parallelizable: true,
		ArbOSVersion:   params.ArbosVersion_40,
	}}
}

func testRunNativeTokenManagementNotAvailableBeforeArbos41(env *TestEnv) {
	auth := env.GetDefaultTransactOpts("Owner")
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, env.L2.Client)
	env.Require(err)

	env.L2Info.GenerateAccount("User2")
	accountAddr := env.L2Info.GetAddress("User2")
	_, err = arbOwner.AddNativeTokenOwner(&auth, accountAddr)
	if err == nil || err.Error() != "execution reverted" {
		env.Fatal("expected adding native token owner to fail")
	}

	balanceBefore, err := env.L2.Client.BalanceAt(env.Ctx, accountAddr, nil)
	env.Require(err)
	arbNativeTokenManager, err := precompilesgen.NewArbNativeTokenManager(types.ArbNativeTokenManagerAddress, env.L2.Client)
	env.Require(err)
	tx, err := arbNativeTokenManager.MintNativeToken(&auth, big.NewInt(100))
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	balanceAfter, err := env.L2.Client.BalanceAt(env.Ctx, accountAddr, nil)
	env.Require(err)
	if balanceBefore.Cmp(balanceAfter) != 0 {
		env.Fatal("expected balance to be the same before and after minting")
	}
}

// ============================================================================
// setupArbOwnerAndArbGasInfo cluster
// ============================================================================

func testRunL1BaseFeeEstimateInertia(env *TestEnv) {
	auth, arbOwner, arbGasInfo := setupArbOwnerAndArbGasInfo(env)
	inertia := uint64(11)
	tx, err := arbOwner.SetL1BaseFeeEstimateInertia(&auth, inertia)
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	got, err := arbGasInfo.GetL1BaseFeeEstimateInertia(&bind.CallOpts{Context: env.Ctx})
	env.Require(err)
	if got != inertia {
		env.Fatal("expected inertia to be", inertia, "got", got)
	}
}

func testRunL1PricingInertia(env *TestEnv) {
	auth, arbOwner, arbGasInfo := setupArbOwnerAndArbGasInfo(env)
	inertia := uint64(12)
	tx, err := arbOwner.SetL1PricingInertia(&auth, inertia)
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	got, err := arbGasInfo.GetL1BaseFeeEstimateInertia(&bind.CallOpts{Context: env.Ctx})
	env.Require(err)
	if got != inertia {
		env.Fatal("expected inertia to be", inertia, "got", got)
	}
}

func testRunL1PricingRewardRate(env *TestEnv) {
	auth, arbOwner, arbGasInfo := setupArbOwnerAndArbGasInfo(env)
	perUnitReward := uint64(13)
	tx, err := arbOwner.SetL1PricingRewardRate(&auth, perUnitReward)
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	got, err := arbGasInfo.GetL1RewardRate(&bind.CallOpts{Context: env.Ctx})
	env.Require(err)
	if got != perUnitReward {
		env.Fatal("expected per unit reward to be", perUnitReward, "got", got)
	}
}

func testRunL1PricingRewardRecipient(env *TestEnv) {
	auth, arbOwner, arbGasInfo := setupArbOwnerAndArbGasInfo(env)
	rewardRecipient := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	tx, err := arbOwner.SetL1PricingRewardRecipient(&auth, rewardRecipient)
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	got, err := arbGasInfo.GetL1RewardRecipient(&bind.CallOpts{Context: env.Ctx})
	env.Require(err)
	if got.Cmp(rewardRecipient) != 0 {
		env.Fatal("expected reward recipient to be", rewardRecipient, "got", got)
	}
}

func testRunL2GasPricingInertia(env *TestEnv) {
	auth, arbOwner, arbGasInfo := setupArbOwnerAndArbGasInfo(env)
	inertia := uint64(14)
	tx, err := arbOwner.SetL2GasPricingInertia(&auth, inertia)
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	got, err := arbGasInfo.GetPricingInertia(&bind.CallOpts{Context: env.Ctx})
	env.Require(err)
	if got != inertia {
		env.Fatal("expected inertia to be", inertia, "got", got)
	}
}

func testRunL2GasBacklogTolerance(env *TestEnv) {
	auth, arbOwner, arbGasInfo := setupArbOwnerAndArbGasInfo(env)
	gasTolerance := uint64(15)
	tx, err := arbOwner.SetL2GasBacklogTolerance(&auth, gasTolerance)
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	got, err := arbGasInfo.GetGasBacklogTolerance(&bind.CallOpts{Context: env.Ctx})
	env.Require(err)
	if got != gasTolerance {
		env.Fatal("expected gas tolerance to be", gasTolerance, "got", got)
	}
}

func testRunPerBatchGasCharge(env *TestEnv) {
	auth, arbOwner, arbGasInfo := setupArbOwnerAndArbGasInfo(env)
	perBatchGasCharge := int64(16)
	tx, err := arbOwner.SetPerBatchGasCharge(&auth, perBatchGasCharge)
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	got, err := arbGasInfo.GetPerBatchGasCharge(&bind.CallOpts{Context: env.Ctx})
	env.Require(err)
	if got != perBatchGasCharge {
		env.Fatal("expected per batch gas charge to be", perBatchGasCharge, "got", got)
	}
}

func testRunL1PricingEquilibrationUnits(env *TestEnv) {
	auth, arbOwner, arbGasInfo := setupArbOwnerAndArbGasInfo(env)
	equilUnits := big.NewInt(17)
	tx, err := arbOwner.SetL1PricingEquilibrationUnits(&auth, equilUnits)
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	got, err := arbGasInfo.GetL1PricingEquilibrationUnits(&bind.CallOpts{Context: env.Ctx})
	env.Require(err)
	if got.Cmp(equilUnits) != 0 {
		env.Fatal("expected equilibration units to be", equilUnits, "got", got)
	}
}

func testRunGasAccountingParams(env *TestEnv) {
	auth, arbOwner, arbGasInfo := setupArbOwnerAndArbGasInfo(env)
	speedLimit := uint64(18)
	blockGasLimit := uint64(19)
	tx, err := arbOwner.SetSpeedLimit(&auth, speedLimit)
	env.Require(err)
	env.EnsureTxSucceeded(tx)
	tx, err = arbOwner.SetMaxBlockGasLimit(&auth, blockGasLimit)
	env.Require(err)
	env.EnsureTxSucceeded(tx)

	gotSpeedLimit, gotPoolSize, gotTxGasLimit, err := arbGasInfo.GetGasAccountingParams(&bind.CallOpts{Context: env.Ctx})
	env.Require(err)
	// #nosec G115
	if gotSpeedLimit.Cmp(big.NewInt(int64(speedLimit))) != 0 {
		env.Fatal("expected speed limit to be", speedLimit, "got", gotSpeedLimit)
	}
	// #nosec G115
	if gotPoolSize.Cmp(big.NewInt(int64(blockGasLimit))) != 0 {
		env.Fatal("expected pool size to be", blockGasLimit, "got", gotPoolSize)
	}
	// #nosec G115
	if gotTxGasLimit.Cmp(big.NewInt(int64(blockGasLimit))) != 0 {
		env.Fatal("expected tx gas limit to be", blockGasLimit, "got", gotTxGasLimit)
	}
}
