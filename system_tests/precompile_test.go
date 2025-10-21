// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func TestPurePrecompileMethodCalls(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	arbosVersion := params.ArbosVersion_31
	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, false).
		WithArbOSVersion(arbosVersion)
	cleanup := builder.Build(t)
	defer cleanup()

	arbSys, err := precompilesgen.NewArbSys(common.HexToAddress("0x64"), builder.L2.Client)
	Require(t, err, "could not deploy ArbSys contract")
	chainId, err := arbSys.ArbChainID(&bind.CallOpts{})
	Require(t, err, "failed to get the ChainID")
	if chainId.Uint64() != chaininfo.ArbitrumDevTestChainConfig().ChainID.Uint64() {
		Fatal(t, "Wrong ChainID", chainId.Uint64())
	}

	expectedArbosVersion := 55 + arbosVersion // Nitro versions start at 56
	arbSysArbosVersion, err := arbSys.ArbOSVersion(&bind.CallOpts{})
	Require(t, err)
	if arbSysArbosVersion.Uint64() != expectedArbosVersion {
		Fatal(t, "Expected ArbOS version", expectedArbosVersion, "got", arbSysArbosVersion)
	}

	storageGasAvailable, err := arbSys.GetStorageGasAvailable(&bind.CallOpts{})
	Require(t, err)
	if storageGasAvailable.Cmp(big.NewInt(0)) != 0 {
		Fatal(t, "Expected 0 storage gas available, got", storageGasAvailable)
	}
}

func TestViewLogReverts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	arbDebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), builder.L2.Client)
	Require(t, err, "could not deploy ArbSys contract")

	err = arbDebug.EventsView(nil)
	if err == nil {
		Fatal(t, "unexpected success")
	}
}

func TestArbDebugPanic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, builder.L2.Client)
	Require(t, err)

	_, err = arbDebug.Panic(&auth)
	if err == nil {
		Fatal(t, "unexpected success")
	}
	if err.Error() != "method handler crashed" {
		Fatal(t, "expected method handler to crash")
	}
}

func TestArbDebugLegacyError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	callOpts := &bind.CallOpts{Context: ctx}

	arbDebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), builder.L2.Client)
	Require(t, err)

	err = arbDebug.LegacyError(callOpts)
	if err == nil {
		Fatal(t, "unexpected success")
	}
}

func TestCustomSolidityErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	callOpts := &bind.CallOpts{Context: ctx}
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	ensure := func(
		customError error,
		expectedError string,
		scenario string,
	) {
		if customError == nil {
			Fatal(t, "should have errored", "scenario", scenario)
		}
		observedMessage := customError.Error()
		// The first error is server side. The second error is client side ABI decoding.
		expectedMessage := fmt.Sprintf("execution reverted: error %v: %v", expectedError, expectedError)
		if observedMessage != expectedMessage {
			Fatal(t, observedMessage, "scenario", scenario)
		}
	}

	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, builder.L2.Client)
	Require(t, err, "could not bind ArbDebug contract")
	ensure(
		arbDebug.CustomRevert(callOpts, 1024),
		"Custom(1024, This spider family wards off bugs: /\\oo/\\ //\\(oo)//\\ /\\oo/\\, true)",
		"arbDebug.CustomRevert",
	)

	arbSys, err := precompilesgen.NewArbSys(arbos.ArbSysAddress, builder.L2.Client)
	Require(t, err, "could not bind ArbSys contract")
	_, customError := arbSys.ArbBlockHash(callOpts, big.NewInt(1e9))
	ensure(
		customError,
		"InvalidBlockNumber(1000000000, 1)",
		"arbSys.ArbBlockHash",
	)

	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(types.ArbRetryableTxAddress, builder.L2.Client)
	Require(t, err)
	_, customError = arbRetryableTx.SubmitRetryable(
		&auth,
		[32]byte{},
		big.NewInt(0),
		big.NewInt(0),
		big.NewInt(0),
		big.NewInt(0),
		0,
		big.NewInt(0),
		common.Address{},
		common.Address{},
		common.Address{},
		[]byte{},
	)
	ensure(
		customError,
		"NotCallable()",
		"arbRetryableTx.SubmitRetryable",
	)

	arbosActs, err := precompilesgen.NewArbosActs(types.ArbosAddress, builder.L2.Client)
	Require(t, err)
	_, customError = arbosActs.StartBlock(&auth, big.NewInt(0), 0, 0, 0)
	ensure(
		customError,
		"CallerNotArbOS()",
		"arbosActs.StartBlock",
	)

	_, customError = arbosActs.BatchPostingReport(&auth, big.NewInt(0), common.Address{}, 0, 0, big.NewInt(0))
	ensure(
		customError,
		"CallerNotArbOS()",
		"arbosActs.BatchPostingReport",
	)

	_, customError = arbosActs.BatchPostingReportV2(&auth, big.NewInt(0), common.Address{}, 0, 0, 0, 0, big.NewInt(0))
	ensure(
		customError,
		"CallerNotArbOS()",
		"arbosActs.BatchPostingReportV2",
	)
}

func TestPrecompileErrorGasLeft(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)
	_, _, simple, err := localgen.DeploySimple(&auth, builder.L2.Client)
	Require(t, err)

	assertNotAllGasConsumed := func(to common.Address, input []byte) {
		gas, err := simple.CheckGasUsed(&bind.CallOpts{Context: ctx}, to, input)
		Require(t, err, "Failed to call CheckGasUsed to precompile", to)
		maxGas := big.NewInt(100_000)
		if arbmath.BigGreaterThan(gas, maxGas) {
			Fatal(t, "Precompile", to, "used", gas, "gas reverting, greater than max expected", maxGas)
		}
	}

	arbSys, err := precompilesgen.ArbSysMetaData.GetAbi()
	Require(t, err)

	arbBlockHash := arbSys.Methods["arbBlockHash"]
	data, err := arbBlockHash.Inputs.Pack(big.NewInt(1e9))
	Require(t, err)
	input := append([]byte{}, arbBlockHash.ID...)
	input = append(input, data...)
	assertNotAllGasConsumed(arbos.ArbSysAddress, input)

	arbDebug, err := precompilesgen.ArbDebugMetaData.GetAbi()
	Require(t, err)
	assertNotAllGasConsumed(common.HexToAddress("0xff"), arbDebug.Methods["legacyError"].ID)
}

func setupArbOwnerAndArbGasInfo(
	t *testing.T,
) (
	*NodeBuilder,
	func(),
	bind.TransactOpts,
	*precompilesgen.ArbOwner,
	*precompilesgen.ArbGasInfo,
) {
	ctx, cancel := context.WithCancel(context.Background())

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builderCleanup := builder.Build(t)

	cleanup := func() {
		builderCleanup()
		cancel()
	}

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), builder.L2.Client)
	Require(t, err)
	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), builder.L2.Client)
	Require(t, err)

	return builder, cleanup, auth, arbOwner, arbGasInfo
}

func TestL1BaseFeeEstimateInertia(t *testing.T) {
	builder, cleanup, auth, arbOwner, arbGasInfo := setupArbOwnerAndArbGasInfo(t)
	defer cleanup()
	ctx := builder.ctx

	inertia := uint64(11)
	tx, err := arbOwner.SetL1BaseFeeEstimateInertia(&auth, inertia)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	arbGasInfoInertia, err := arbGasInfo.GetL1BaseFeeEstimateInertia(&bind.CallOpts{Context: ctx})
	Require(t, err)
	if arbGasInfoInertia != inertia {
		Fatal(t, "expected inertia to be", inertia, "got", arbGasInfoInertia)
	}
}

// Similar to TestL1BaseFeeEstimateInertia, but now using a different setter from ArbOwner
func TestL1PricingInertia(t *testing.T) {
	builder, cleanup, auth, arbOwner, arbGasInfo := setupArbOwnerAndArbGasInfo(t)
	defer cleanup()
	ctx := builder.ctx

	inertia := uint64(12)
	tx, err := arbOwner.SetL1PricingInertia(&auth, inertia)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	arbGasInfoInertia, err := arbGasInfo.GetL1BaseFeeEstimateInertia(&bind.CallOpts{Context: ctx})
	Require(t, err)
	if arbGasInfoInertia != inertia {
		Fatal(t, "expected inertia to be", inertia, "got", arbGasInfoInertia)
	}
}

func TestL1PricingRewardRate(t *testing.T) {
	builder, cleanup, auth, arbOwner, arbGasInfo := setupArbOwnerAndArbGasInfo(t)
	defer cleanup()
	ctx := builder.ctx

	perUnitReward := uint64(13)
	tx, err := arbOwner.SetL1PricingRewardRate(&auth, perUnitReward)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	arbGasInfoPerUnitReward, err := arbGasInfo.GetL1RewardRate(&bind.CallOpts{Context: ctx})
	Require(t, err)
	if arbGasInfoPerUnitReward != perUnitReward {
		Fatal(t, "expected per unit reward to be", perUnitReward, "got", arbGasInfoPerUnitReward)
	}
}

func TestL1PricingRewardRecipient(t *testing.T) {
	builder, cleanup, auth, arbOwner, arbGasInfo := setupArbOwnerAndArbGasInfo(t)
	defer cleanup()
	ctx := builder.ctx

	rewardRecipient := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	tx, err := arbOwner.SetL1PricingRewardRecipient(&auth, rewardRecipient)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	arbGasInfoRewardRecipient, err := arbGasInfo.GetL1RewardRecipient(&bind.CallOpts{Context: ctx})
	Require(t, err)
	if arbGasInfoRewardRecipient.Cmp(rewardRecipient) != 0 {
		Fatal(t, "expected reward recipient to be", rewardRecipient, "got", arbGasInfoRewardRecipient)
	}
}

func TestL2GasPricingInertia(t *testing.T) {
	builder, cleanup, auth, arbOwner, arbGasInfo := setupArbOwnerAndArbGasInfo(t)
	defer cleanup()
	ctx := builder.ctx

	inertia := uint64(14)
	tx, err := arbOwner.SetL2GasPricingInertia(&auth, inertia)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	arbGasInfoInertia, err := arbGasInfo.GetPricingInertia(&bind.CallOpts{Context: ctx})
	Require(t, err)
	if arbGasInfoInertia != inertia {
		Fatal(t, "expected inertia to be", inertia, "got", arbGasInfoInertia)
	}
}

func TestL2GasBacklogTolerance(t *testing.T) {
	builder, cleanup, auth, arbOwner, arbGasInfo := setupArbOwnerAndArbGasInfo(t)
	defer cleanup()
	ctx := builder.ctx

	gasTolerance := uint64(15)
	tx, err := arbOwner.SetL2GasBacklogTolerance(&auth, gasTolerance)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	arbGasInfoGasTolerance, err := arbGasInfo.GetGasBacklogTolerance(&bind.CallOpts{Context: ctx})
	Require(t, err)
	if arbGasInfoGasTolerance != gasTolerance {
		Fatal(t, "expected gas tolerance to be", gasTolerance, "got", arbGasInfoGasTolerance)
	}
}

func TestPerBatchGasCharge(t *testing.T) {
	builder, cleanup, auth, arbOwner, arbGasInfo := setupArbOwnerAndArbGasInfo(t)
	defer cleanup()
	ctx := builder.ctx

	perBatchGasCharge := int64(16)
	tx, err := arbOwner.SetPerBatchGasCharge(&auth, perBatchGasCharge)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	arbGasInfoPerBatchGasCharge, err := arbGasInfo.GetPerBatchGasCharge(&bind.CallOpts{Context: ctx})
	Require(t, err)
	if arbGasInfoPerBatchGasCharge != perBatchGasCharge {
		Fatal(t, "expected per batch gas charge to be", perBatchGasCharge, "got", arbGasInfoPerBatchGasCharge)
	}
}

func TestL1PricingEquilibrationUnits(t *testing.T) {
	builder, cleanup, auth, arbOwner, arbGasInfo := setupArbOwnerAndArbGasInfo(t)
	defer cleanup()
	ctx := builder.ctx

	equilUnits := big.NewInt(17)
	tx, err := arbOwner.SetL1PricingEquilibrationUnits(&auth, equilUnits)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	arbGasInfoEquilUnits, err := arbGasInfo.GetL1PricingEquilibrationUnits(&bind.CallOpts{Context: ctx})
	Require(t, err)
	if arbGasInfoEquilUnits.Cmp(equilUnits) != 0 {
		Fatal(t, "expected equilibration units to be", equilUnits, "got", arbGasInfoEquilUnits)
	}
}

func TestGasAccountingParams(t *testing.T) {
	builder, cleanup, auth, arbOwner, arbGasInfo := setupArbOwnerAndArbGasInfo(t)
	defer cleanup()
	ctx := builder.ctx

	speedLimit := uint64(18)
	blockGasLimit := uint64(19)
	tx, err := arbOwner.SetSpeedLimit(&auth, speedLimit)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	tx, err = arbOwner.SetMaxBlockGasLimit(&auth, blockGasLimit)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	arbGasInfoSpeedLimit, arbGasInfoPoolSize, arbGasInfoTxGasLimit, err := arbGasInfo.GetGasAccountingParams(&bind.CallOpts{Context: ctx})
	Require(t, err)
	// #nosec G115
	if arbGasInfoSpeedLimit.Cmp(big.NewInt(int64(speedLimit))) != 0 {
		Fatal(t, "expected speed limit to be", speedLimit, "got", arbGasInfoSpeedLimit)
	}
	// #nosec G115
	if arbGasInfoPoolSize.Cmp(big.NewInt(int64(blockGasLimit))) != 0 {
		Fatal(t, "expected pool size to be", blockGasLimit, "got", arbGasInfoPoolSize)
	}
	// #nosec G115
	if arbGasInfoTxGasLimit.Cmp(big.NewInt(int64(blockGasLimit))) != 0 {
		Fatal(t, "expected tx gas limit to be", blockGasLimit, "got", arbGasInfoTxGasLimit)
	}
}

func TestCurrentTxL1GasFees(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	arbGasInfo, err := precompilesgen.NewArbGasInfo(types.ArbGasInfoAddress, builder.L2.Client)
	Require(t, err)

	currTxL1GasFees, err := arbGasInfo.GetCurrentTxL1GasFees(&bind.CallOpts{Context: ctx})
	Require(t, err)
	if currTxL1GasFees == nil {
		Fatal(t, "currTxL1GasFees is nil")
	}
	if currTxL1GasFees.Cmp(big.NewInt(0)) != 1 {
		Fatal(t, "expected currTxL1GasFees to be greater than 0, got", currTxL1GasFees)
	}
}

func TestArbOwnerMaxTxAndBlockGasLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithArbOSVersion(params.ArbosVersion_50)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), builder.L2.Client)
	Require(t, err)
	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), builder.L2.Client)
	Require(t, err)

	wantTxGasLimit := uint64(3000000)
	wantBlockGasLimit := uint64(4000000)
	txGasLimitTx, err := arbOwner.SetMaxTxGasLimit(&auth, wantTxGasLimit)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, builder.L2.Client, txGasLimitTx)
	Require(t, err)
	blockGasLimitTx, err := arbOwner.SetMaxBlockGasLimit(&auth, wantBlockGasLimit)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, builder.L2.Client, blockGasLimitTx)
	Require(t, err)

	statedb, err := builder.L2.ExecNode.Backend.ArbInterface().BlockChain().State()
	Require(t, err)
	burner := burn.NewSystemBurner(nil, false)
	arbosSt, err := arbosState.OpenArbosState(statedb, burner)
	Require(t, err)

	haveTxGasLimit, err := arbosSt.L2PricingState().PerTxGasLimit()
	Require(t, err)
	if haveTxGasLimit != wantTxGasLimit {
		t.Fatalf("txGasLimit mismatch. have: %d want: %d", haveTxGasLimit, wantTxGasLimit)
	}
	haveBlockGasLimit, err := arbosSt.L2PricingState().PerBlockGasLimit()
	Require(t, err)
	if haveBlockGasLimit != wantBlockGasLimit {
		t.Fatalf("blockGasLimit mismatch. have: %d want: %d", haveBlockGasLimit, wantBlockGasLimit)
	}

	haveTxGasLimitArbGasInfo, err := arbGasInfo.GetMaxTxGasLimit(&bind.CallOpts{Context: ctx})
	Require(t, err)
	if haveTxGasLimitArbGasInfo.Uint64() != wantTxGasLimit {
		t.Fatalf("arbGasInfo txGasLimit mismatch. have: %d want: %d", haveTxGasLimitArbGasInfo.Uint64(), wantTxGasLimit)
	}
	_, _, haveBlockGasLimitArbGasInfo, err := arbGasInfo.GetGasAccountingParams(&bind.CallOpts{Context: ctx})
	Require(t, err)
	if haveBlockGasLimitArbGasInfo.Uint64() != wantBlockGasLimit {
		t.Fatalf("arbGasInfo blockGasLimit mismatch. have: %d want: %d", haveBlockGasLimitArbGasInfo.Uint64(), wantBlockGasLimit)
	}
}

func TestArbNativeTokenManagerThroughSolidityContract(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	arbOSInit := &params.ArbOSInit{
		NativeTokenSupplyManagementEnabled: true,
	}
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithArbOSInit(arbOSInit).WithArbOSVersion(params.ArbosVersion_50)
	cleanup := builder.Build(t)
	defer cleanup()

	authOwner := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	authOwner.GasLimit = 32000000

	// deploys test contract
	contractAddr, tx, contract, err := localgen.DeployArbNativeTokenManagerTest(&authOwner, builder.L2.Client)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, builder.L2.Client, tx)
	Require(t, err)

	// adds native token owner
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	tx, err = arbOwner.AddNativeTokenOwner(&authOwner, contractAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// mints
	toMint := big.NewInt(100)
	tx, err = contract.Mint(&authOwner, toMint)
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// checks minting
	arbNativeTokenManager, err := precompilesgen.NewArbNativeTokenManager(types.ArbNativeTokenManagerAddress, builder.L2.Client)
	Require(t, err)
	nativeTokenOwnerABI, err := precompilesgen.ArbNativeTokenManagerMetaData.GetAbi()
	Require(t, err)
	mintTopic := nativeTokenOwnerABI.Events["NativeTokenMinted"].ID
	mintLogged := false
	for _, log := range receipt.Logs {
		if log.Topics[0] == mintTopic {
			mintLogged = true
			parsedLog, err := arbNativeTokenManager.ParseNativeTokenMinted(*log)
			Require(t, err)
			if parsedLog.To != contractAddr {
				t.Fatal("expected mint to be to", contractAddr, "got", parsedLog.To)
			}
			if parsedLog.Amount.Cmp(toMint) != 0 {
				t.Fatal("expected mint amount to be", toMint, "got", parsedLog.Amount)
			}
		}
	}
	if !mintLogged {
		t.Fatal("expected mint event to be logged")
	}
}

func TestArbNativeTokenManager(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The chain being tested will have the feature enabled.
	arbOSInit := &params.ArbOSInit{
		NativeTokenSupplyManagementEnabled: true,
	}

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithArbOSInit(arbOSInit).WithArbOSVersion(params.ArbosVersion_50)
	cleanup := builder.Build(t)
	defer cleanup()

	authOwner := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	ownerAddr := builder.L2Info.GetAddress("Owner")

	callOpts := &bind.CallOpts{Context: ctx}

	// first tests native token owner management

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	arbOwnerPub, err := precompilesgen.NewArbOwnerPublic(types.ArbOwnerPublicAddress, builder.L2.Client)
	Require(t, err)

	nativeTokenOwnerName := "NativeTokenOwner"
	builder.L2Info.GenerateAccount(nativeTokenOwnerName)
	nativeTokenOwnerAddr := builder.L2Info.GetAddress(nativeTokenOwnerName)

	// checks that no native token owners are set
	isNativeTokenOwner, err := arbOwner.IsNativeTokenOwner(callOpts, nativeTokenOwnerAddr)
	Require(t, err)
	if isNativeTokenOwner {
		t.Fatal("expected native token owner to not be set")
	}
	nativeTokenOwners, err := arbOwner.GetAllNativeTokenOwners(callOpts)
	Require(t, err)
	if len(nativeTokenOwners) != 0 {
		t.Fatal("expected no native token owners")
	}
	// same checks to exercise the public interface
	isNativeTokenOwner, err = arbOwnerPub.IsNativeTokenOwner(callOpts, nativeTokenOwnerAddr)
	Require(t, err)
	if isNativeTokenOwner {
		t.Fatal("expected native token owner to not be set")
	}
	nativeTokenOwners, err = arbOwnerPub.GetAllNativeTokenOwners(callOpts)
	Require(t, err)
	if len(nativeTokenOwners) != 0 {
		t.Fatal("expected no native token owners")
	}

	// adds native token owners
	tx, err := arbOwner.AddNativeTokenOwner(&authOwner, nativeTokenOwnerAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	tx, err = arbOwner.AddNativeTokenOwner(&authOwner, ownerAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// checks that the native token owners are set
	isNativeTokenOwner, err = arbOwner.IsNativeTokenOwner(callOpts, nativeTokenOwnerAddr)
	Require(t, err)
	if !isNativeTokenOwner {
		t.Fatal("expected native token owner to be set")
	}
	expectedNativeTokenOwners := []common.Address{nativeTokenOwnerAddr, ownerAddr}
	addrSorter := func(a, b common.Address) int {
		return a.Cmp(b)
	}
	slices.SortFunc(expectedNativeTokenOwners, addrSorter)
	nativeTokenOwners, err = arbOwner.GetAllNativeTokenOwners(callOpts)
	Require(t, err)
	slices.SortFunc(nativeTokenOwners, addrSorter)
	if diff := cmp.Diff(nativeTokenOwners, expectedNativeTokenOwners); diff != "" {
		t.Errorf("native token owners differ: %s", diff)
	}
	// same checks to exercise the public interface
	isNativeTokenOwner, err = arbOwnerPub.IsNativeTokenOwner(callOpts, nativeTokenOwnerAddr)
	Require(t, err)
	if !isNativeTokenOwner {
		t.Fatal("expected native token owner to be set")
	}
	nativeTokenOwners, err = arbOwnerPub.GetAllNativeTokenOwners(callOpts)
	slices.SortFunc(nativeTokenOwners, addrSorter)
	Require(t, err)
	if diff := cmp.Diff(nativeTokenOwners, expectedNativeTokenOwners); diff != "" {
		t.Errorf("native token owners differ: %s", diff)
	}

	// removes native token owner
	tx, err = arbOwner.RemoveNativeTokenOwner(&authOwner, ownerAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	isNativeTokenOwner, err = arbOwner.IsNativeTokenOwner(callOpts, ownerAddr)
	Require(t, err)
	if isNativeTokenOwner {
		t.Fatal("expected native token owner to not be set")
	}
	enabledTime, err := arbOwnerPub.GetNativeTokenManagementFrom(callOpts)
	Require(t, err)
	if enabledTime != 1 {
		t.Fatalf("enabledTime: want %d, got %d", 1, enabledTime)
	}
	nativeTokenOwners, err = arbOwner.GetAllNativeTokenOwners(callOpts)
	Require(t, err)
	if len(nativeTokenOwners) != 1 {
		t.Fatal("expected one native token owner")
	}
	if nativeTokenOwners[0].Cmp(nativeTokenOwnerAddr) != 0 {
		t.Fatal("expected native token owner to be", nativeTokenOwnerAddr, "got", nativeTokenOwners[0])
	}

	// tests minting and burning native tokens

	nativeTokenOwnerABI, err := precompilesgen.ArbNativeTokenManagerMetaData.GetAbi()
	Require(t, err)
	mintTopic := nativeTokenOwnerABI.Events["NativeTokenMinted"].ID
	burnTopic := nativeTokenOwnerABI.Events["NativeTokenBurned"].ID

	arbNativeTokenManager, err := precompilesgen.NewArbNativeTokenManager(types.ArbNativeTokenManagerAddress, builder.L2.Client)
	Require(t, err)

	// tries to mint and burn without being a native token owner
	_, err = arbNativeTokenManager.MintNativeToken(&authOwner, big.NewInt(100))
	if err == nil || err.Error() != "execution reverted" {
		t.Fatal("expected minting to fail")
	}
	_, err = arbNativeTokenManager.BurnNativeToken(&authOwner, big.NewInt(100))
	if err == nil || err.Error() != "execution reverted" {
		t.Fatal("expected burning to fail")
	}

	// funds the native token owner
	tx = builder.L2Info.PrepareTx("Owner", nativeTokenOwnerName, builder.L2Info.TransferGas, big.NewInt(500000000000000000), nil)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	authNativeTokenOwner := builder.L2Info.GetDefaultTransactOpts(nativeTokenOwnerName, ctx)
	authNativeTokenOwner.GasLimit = 32000000

	getGasUsed := func(receipt *types.Receipt) *big.Int {
		gasUsed := new(big.Int).SetUint64(receipt.GasUsed)
		gasUsed.Mul(gasUsed, receipt.EffectiveGasPrice)
		return gasUsed
	}

	// checks minting
	toMint := big.NewInt(100)
	balanceBeforeMinting, err := builder.L2.Client.BalanceAt(ctx, nativeTokenOwnerAddr, nil)
	Require(t, err)
	tx, err = arbNativeTokenManager.MintNativeToken(&authNativeTokenOwner, toMint)
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	mintLogged := false
	for _, log := range receipt.Logs {
		if log.Topics[0] == mintTopic {
			mintLogged = true
			parsedLog, err := arbNativeTokenManager.ParseNativeTokenMinted(*log)
			Require(t, err)
			if parsedLog.To != nativeTokenOwnerAddr {
				t.Fatal("expected mint to be to", nativeTokenOwnerAddr, "got", parsedLog.To)
			}
			if parsedLog.Amount.Cmp(toMint) != 0 {
				t.Fatal("expected mint amount to be", toMint, "got", parsedLog.Amount)
			}
		}
	}
	if !mintLogged {
		t.Fatal("expected mint event to be logged")
	}
	balanceAfterMinting, err := builder.L2.Client.BalanceAt(ctx, nativeTokenOwnerAddr, nil)
	Require(t, err)
	gasUsed := getGasUsed(receipt)
	expectedBalance := new(big.Int).Sub(balanceBeforeMinting, gasUsed)
	expectedBalance = expectedBalance.Add(expectedBalance, toMint)
	if balanceAfterMinting.Cmp(expectedBalance) != 0 {
		t.Fatal("expected balance to be", expectedBalance, "got", balanceAfterMinting)
	}

	// checks burning
	toBurn := big.NewInt(50)
	tx, err = arbNativeTokenManager.BurnNativeToken(&authNativeTokenOwner, toBurn)
	Require(t, err)
	receipt, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	burnLogged := false
	for _, log := range receipt.Logs {
		if log.Topics[0] == burnTopic {
			burnLogged = true
			parsedLog, err := arbNativeTokenManager.ParseNativeTokenBurned(*log)
			Require(t, err)
			if parsedLog.From != nativeTokenOwnerAddr {
				t.Fatal("expected mint to be from", nativeTokenOwnerAddr, "got", parsedLog.From)
			}
			if parsedLog.Amount.Cmp(toBurn) != 0 {
				t.Fatal("expected mint amount to be", toBurn, "got", parsedLog.Amount)
			}
		}
	}
	if !burnLogged {
		t.Fatal("expected burn event to be logged")
	}
	balanceAfterBurning, err := builder.L2.Client.BalanceAt(ctx, nativeTokenOwnerAddr, nil)
	Require(t, err)
	gasUsed = getGasUsed(receipt)
	expectedBalance = new(big.Int).Sub(balanceAfterMinting, gasUsed)
	expectedBalance = expectedBalance.Sub(expectedBalance, toBurn)
	if balanceAfterBurning.Cmp(expectedBalance) != 0 {
		t.Fatal("expected balance to be", expectedBalance, "got", balanceAfterBurning)
	}

	// checks sending L2 to L1 value is disabled while native token owners exist
	arbSys, err := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L2.Client)
	Require(t, err)
	authNativeTokenOwner.Value = big.NewInt(100)
	_, err = arbSys.SendTxToL1(&authNativeTokenOwner, common.Address{}, []byte{})
	if err == nil || err.Error() != "execution reverted" {
		t.Fatal("expected sending L2 to L1 value to fail")
	}

	// After clearning the native token owners, sending L2 to L1 value should
	// work again.
	tx, err = arbOwner.RemoveNativeTokenOwner(&authOwner, nativeTokenOwnerAddr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	authNativeTokenOwner.Value = big.NewInt(100)
	_, err = arbSys.SendTxToL1(&authNativeTokenOwner, common.Address{}, []byte{})
	if err != nil {
		t.Fatal("expected sending L2 to L1 value to succeed")
	}
}

func TestNativeTokenManagementDisabledByDefault(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithArbOSVersion(params.ArbosVersion_50)
	cleanup := builder.Build(t)
	defer cleanup()

	authOwner := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{Context: ctx}

	// tests that native token owner management is disabled by default
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)

	nativeTokenOwnerName := "NativeTokenOwner"
	builder.L2Info.GenerateAccount(nativeTokenOwnerName)
	nativeTokenOwnerAddr := builder.L2Info.GetAddress(nativeTokenOwnerName)

	// checks that no native token owners are set
	isNativeTokenOwner, err := arbOwner.IsNativeTokenOwner(callOpts, nativeTokenOwnerAddr)
	Require(t, err)
	if isNativeTokenOwner {
		t.Fatal("expected native token owner to not be set")
	}
	nativeTokenOwners, err := arbOwner.GetAllNativeTokenOwners(callOpts)
	Require(t, err)
	if len(nativeTokenOwners) != 0 {
		t.Fatal("expected no native token owners")
	}

	// attempts to add native token owners before the feature is enabled
	_, err = arbOwner.AddNativeTokenOwner(&authOwner, nativeTokenOwnerAddr)
	if err == nil || err.Error() != "execution reverted" {
		t.Error("expected adding native token owner to fail")
	}

	now := time.Now()
	// #nosec G115
	sixDaysFromNow := uint64(now.Add(24 * 6 * time.Hour).Unix())
	// #nosec G115
	sevenAndAHalfDaysFromNow := uint64(now.Add(24*7*time.Hour + 12*time.Hour).Unix())
	// #nosec G115
	eightDaysFromNow := uint64(now.Add(24 * 8 * time.Hour).Unix())

	// attempts to enable the feature too early (6 days from now, instead of 7)
	_, err = arbOwner.SetNativeTokenManagementFrom(&authOwner, sixDaysFromNow)
	if err == nil || err.Error() != "execution reverted" {
		t.Error("expected enabling native token management to fail")
	}

	// succeeds to enable the feature enough in the future (8 days from now)
	tx, err := arbOwner.SetNativeTokenManagementFrom(&authOwner, eightDaysFromNow)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// succeeds to shorten the time to enable the feature as long as it is still
	// far enough in the future (7.5 days from now)
	tx, err = arbOwner.SetNativeTokenManagementFrom(&authOwner, sevenAndAHalfDaysFromNow)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// fails to shorten the time to enable the feature if it is too close to
	// the current time (6 days from now)
	_, err = arbOwner.SetNativeTokenManagementFrom(&authOwner, sixDaysFromNow)
	if err == nil || err.Error() != "execution reverted" {
		t.Error("expected enabling native token management to fail")
	}

	// About to test some very specific time-sensitive boundaries. Setting
	// a new value for now.
	now = time.Now()

	// succeeds to shorten the time to enable the feature to just 5 seconds more
	// than 7 days from now.
	// #nosec G115
	sevenDaysFiveSecondsFromNow := uint64(now.Add(24*7*time.Hour + 5*time.Second).Unix())
	tx, err = arbOwner.SetNativeTokenManagementFrom(&authOwner, sevenDaysFiveSecondsFromNow)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Sleep for 15 seconds to ensure that that the time the feature is will be
	// enabled is <= 7 days from now.
	time.Sleep(15 * time.Second)
	// Time to Enable ~ 6.23:59:50
	// Resetting now after the sleep
	now = time.Now()

	// Now is should be okay to set the time to enable the feature to some time
	// greater than 6 days, 23 hours, 59 minutes and 50 seconds from now, but
	// less than 7 days from now. ~ 6.23:59:55
	// #nosec G115
	almostSevenDaysFromNow := uint64(now.Add(24*7*time.Hour - 5*time.Second).Unix())
	tx, err = arbOwner.SetNativeTokenManagementFrom(&authOwner, almostSevenDaysFromNow)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// It should not, however, be okay to set the time to an even earlier time.
	// ~ 6.23:59:40
	// #nosec G115
	tooFarFromSevenDaysFromNow := uint64(now.Add(24*7*time.Hour - 20*time.Second).Unix())
	_, err = arbOwner.SetNativeTokenManagementFrom(&authOwner, tooFarFromSevenDaysFromNow)
	if err == nil || err.Error() != "execution reverted" {
		t.Error("expected enabling native token management to fail")
	}
}

func TestNativeTokenManagementNotAvailableBeforeArbos41(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithArbOSVersion(params.ArbosVersion_40)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	accountName := "User2"
	builder.L2Info.GenerateAccount(accountName)
	accountAddr := builder.L2Info.GetAddress(accountName)
	_, err = arbOwner.AddNativeTokenOwner(&auth, accountAddr)
	if err == nil || err.Error() != "execution reverted" {
		t.Fatal("expected adding native token owner to fail")
	}

	// checks that minting doesn't work
	balanceBeforeMinting, err := builder.L2.Client.BalanceAt(ctx, accountAddr, nil)
	Require(t, err)
	arbNativeTokenManager, err := precompilesgen.NewArbNativeTokenManager(types.ArbNativeTokenManagerAddress, builder.L2.Client)
	Require(t, err)
	tx, err := arbNativeTokenManager.MintNativeToken(&auth, big.NewInt(100))
	// It doesn't fail because it is handled as ArbNativeTokenManager doesn't exist
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	balanceAfterMinting, err := builder.L2.Client.BalanceAt(ctx, accountAddr, nil)
	Require(t, err)
	if balanceBeforeMinting.Cmp(balanceAfterMinting) != 0 {
		t.Fatal("expected balance to be the same before and after minting")
	}
}

func TestGetBrotliCompressionLevel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(types.ArbOwnerPublicAddress, builder.L2.Client)
	Require(t, err)

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)

	brotliCompressionLevel := uint64(11)

	// sets brotli compression level
	tx, err := arbOwner.SetBrotliCompressionLevel(&auth, brotliCompressionLevel)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// retrieves brotli compression level
	callOpts := &bind.CallOpts{Context: ctx}
	retrievedBrotliCompressionLevel, err := arbOwnerPublic.GetBrotliCompressionLevel(callOpts)
	Require(t, err)
	if retrievedBrotliCompressionLevel != brotliCompressionLevel {
		Fatal(t, "expected brotli compression level to be", brotliCompressionLevel, "got", retrievedBrotliCompressionLevel)
	}
}

func TestArbStatistics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	arbStatistics, err := precompilesgen.NewArbStatistics(types.ArbStatisticsAddress, builder.L2.Client)
	Require(t, err)

	callOpts := &bind.CallOpts{Context: ctx}
	blockNum, _, _, _, _, _, err := arbStatistics.GetStats(callOpts)
	Require(t, err)

	expectedBlockNum, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)

	if blockNum.Uint64() != expectedBlockNum {
		Fatal(t, "expected block number to be", expectedBlockNum, "got", blockNum)
	}
}

func TestArbosFeatures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{Context: ctx}

	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(types.ArbOwnerPublicAddress, builder.L2.Client)
	Require(t, err)

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)

	// check that the feature is disabled by default
	isCDPIEnabled, err := arbOwnerPublic.IsCalldataPriceIncreaseEnabled(callOpts)
	Require(t, err)
	if isCDPIEnabled {
		Fatal(t, "expected calldata price increase to be disabled")
	}

	// enable the feature
	tx, err := arbOwner.SetCalldataPriceIncrease(&auth, true)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// check that the feature is enabled
	isCDPIEnabled, err = arbOwnerPublic.IsCalldataPriceIncreaseEnabled(callOpts)
	Require(t, err)
	if !isCDPIEnabled {
		Fatal(t, "expected calldata price increase to be enabled")
	}
}

func TestArbFunctionTable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{Context: ctx}

	arbFunctionTable, err := precompilesgen.NewArbFunctionTable(types.ArbFunctionTableAddress, builder.L2.Client)
	Require(t, err)

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	// should be a noop
	tx, err := arbFunctionTable.Upload(&auth, []byte{0, 0, 0, 0})
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	size, err := arbFunctionTable.Size(callOpts, addr)
	Require(t, err)
	if size.Cmp(big.NewInt(0)) != 0 {
		t.Fatal("Size should be 0")
	}

	_, _, _, err = arbFunctionTable.Get(callOpts, addr, big.NewInt(10))
	if err == nil {
		t.Fatal("Should error")
	}
}

func TestArbAggregatorBaseFee(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{Context: ctx}

	arbAggregator, err := precompilesgen.NewArbAggregator(types.ArbAggregatorAddress, builder.L2.Client)
	Require(t, err)

	tx, err := arbAggregator.SetTxBaseFee(&auth, common.Address{}, big.NewInt(1))
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	fee, err := arbAggregator.GetTxBaseFee(callOpts, common.Address{})
	Require(t, err)
	if fee.Cmp(big.NewInt(0)) != 0 {
		Fatal(t, "expected fee to be 0, got", fee)
	}
}

func TestFeeAccounts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{Context: ctx}

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)

	builder.L2Info.GenerateAccount("User2")
	addr := builder.L2Info.GetAddress("User2")

	tx, err := arbOwner.SetNetworkFeeAccount(&auth, addr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	feeAccount, err := arbOwner.GetNetworkFeeAccount(callOpts)
	Require(t, err)
	if feeAccount.Cmp(addr) != 0 {
		Fatal(t, "expected fee account to be", addr, "got", feeAccount)
	}

	tx, err = arbOwner.SetInfraFeeAccount(&auth, addr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	feeAccount, err = arbOwner.GetInfraFeeAccount(callOpts)
	Require(t, err)
	if feeAccount.Cmp(addr) != 0 {
		Fatal(t, "expected fee account to be", addr, "got", feeAccount)
	}
}

func TestChainOwners(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{Context: ctx}

	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(types.ArbOwnerPublicAddress, builder.L2.Client)
	Require(t, err)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)

	builder.L2Info.GenerateAccount("Owner2")
	chainOwnerAddr2 := builder.L2Info.GetAddress("Owner2")
	tx, err := arbOwner.AddChainOwner(&auth, chainOwnerAddr2)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	isChainOwner, err := arbOwnerPublic.IsChainOwner(callOpts, chainOwnerAddr2)
	Require(t, err)
	if !isChainOwner {
		Fatal(t, "expected owner2 to be a chain owner")
	}

	// check that the chain owners retrieved from arbOwnerPublic and arbOwner are the same
	chainOwnersArbOwnerPublic, err := arbOwnerPublic.GetAllChainOwners(callOpts)
	Require(t, err)
	chainOwnersArbOwner, err := arbOwner.GetAllChainOwners(callOpts)
	Require(t, err)
	if len(chainOwnersArbOwnerPublic) != len(chainOwnersArbOwner) {
		Fatal(t, "expected chain owners to be the same length")
	}
	// sort the chain owners to ensure they are in the same order
	sort.Slice(chainOwnersArbOwnerPublic, func(i, j int) bool {
		return chainOwnersArbOwnerPublic[i].Cmp(chainOwnersArbOwnerPublic[j]) < 0
	})
	for i := 0; i < len(chainOwnersArbOwnerPublic); i += 1 {
		if chainOwnersArbOwnerPublic[i].Cmp(chainOwnersArbOwner[i]) != 0 {
			Fatal(t, "expected chain owners to be the same")
		}
	}
	chainOwnerAddr := builder.L2Info.GetAddress("Owner")
	chainOwnerInChainOwners := false
	for _, chainOwner := range chainOwnersArbOwner {
		if chainOwner.Cmp(chainOwnerAddr) == 0 {
			chainOwnerInChainOwners = true
		}
	}
	if !chainOwnerInChainOwners {
		Fatal(t, "expected owner to be in chain owners")
	}

	// remove chain owner 2
	tx, err = arbOwner.RemoveChainOwner(&auth, chainOwnerAddr2)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	isChainOwner, err = arbOwnerPublic.IsChainOwner(callOpts, chainOwnerAddr2)
	Require(t, err)
	if isChainOwner {
		Fatal(t, "expected owner2 to not be a chain owner")
	}

	_, err = arbOwnerPublic.RectifyChainOwner(&auth, chainOwnerAddr)
	if (err == nil) || (err.Error() != "execution reverted") {
		Fatal(t, "expected rectify chain owner to revert since it is already an owner")
	}
}

func TestArbAggregatorBatchPosters(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{Context: ctx}

	arbAggregator, err := precompilesgen.NewArbAggregator(types.ArbAggregatorAddress, builder.L2.Client)
	Require(t, err)

	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, builder.L2.Client)
	Require(t, err)

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	// initially should have one batch poster
	bps, err := arbAggregator.GetBatchPosters(callOpts)
	Require(t, err)
	if len(bps) != 1 {
		Fatal(t, "expected one batch poster")
	}

	// add addr as a batch poster
	tx, err := arbDebug.BecomeChainOwner(&auth)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	tx, err = arbAggregator.AddBatchPoster(&auth, addr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// there should now be two batch posters, and addr should be one of them
	bps, err = arbAggregator.GetBatchPosters(callOpts)
	Require(t, err)
	if len(bps) != 2 {
		Fatal(t, "expected two batch posters")
	}
	if bps[0] != addr && bps[1] != addr {
		Fatal(t, "expected addr to be a batch poster")
	}
}

func TestArbAggregatorGetPreferredAggregator(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	callOpts := &bind.CallOpts{Context: ctx}

	arbAggregator, err := precompilesgen.NewArbAggregator(types.ArbAggregatorAddress, builder.L2.Client)
	Require(t, err)

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	prefAgg, isDefault, err := arbAggregator.GetPreferredAggregator(callOpts, addr)
	Require(t, err)
	if !isDefault {
		Fatal(t, "expected default preferred aggregator")
	}
	if prefAgg != l1pricing.BatchPosterAddress {
		Fatal(t, "expected default preferred aggregator to be", l1pricing.BatchPosterAddress, "got", prefAgg)
	}

	prefAgg, err = arbAggregator.GetDefaultAggregator(callOpts)
	Require(t, err)
	if prefAgg != l1pricing.BatchPosterAddress {
		Fatal(t, "expected default preferred aggregator to be", l1pricing.BatchPosterAddress, "got", prefAgg)
	}
}

func TestArbDebugOverwriteContractCode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	// Become chain owner
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, builder.L2.Client)
	Require(t, err)
	tx, err := arbDebug.BecomeChainOwner(&auth)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// create EOA to test against
	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	// test that code is empty
	code, err := builder.L2.Client.CodeAt(ctx, addr, nil)
	Require(t, err)
	if len(code) != 0 {
		t.Fatal("expected code to be empty")
	}

	// overwrite with some code
	testCodeA := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	tx, err = arbDebug.OverwriteContractCode(&auth, addr, testCodeA)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	code, err = builder.L2.Client.CodeAt(ctx, addr, nil)
	Require(t, err)
	if !bytes.Equal(code, testCodeA) {
		t.Fatal("expected code A to be", testCodeA, "got", code)
	}

	// overwrite with some other code
	testCodeB := []byte{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	tx, err = arbDebug.OverwriteContractCode(&auth, addr, testCodeB)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	code, err = builder.L2.Client.CodeAt(ctx, addr, nil)
	Require(t, err)
	if !bytes.Equal(code, testCodeB) {
		t.Fatal("expected code B to be", testCodeB, "got", code)
	}
}
