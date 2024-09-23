// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func TestPurePrecompileMethodCalls(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	arbSys, err := precompilesgen.NewArbSys(common.HexToAddress("0x64"), builder.L2.Client)
	Require(t, err, "could not deploy ArbSys contract")
	chainId, err := arbSys.ArbChainID(&bind.CallOpts{})
	Require(t, err, "failed to get the ChainID")
	if chainId.Uint64() != params.ArbitrumDevTestChainConfig().ChainID.Uint64() {
		Fatal(t, "Wrong ChainID", chainId.Uint64())
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

func TestCustomSolidityErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	callOpts := &bind.CallOpts{Context: ctx}
	arbDebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), builder.L2.Client)
	Require(t, err, "could not bind ArbDebug contract")
	customError := arbDebug.CustomRevert(callOpts, 1024)
	if customError == nil {
		Fatal(t, "customRevert call should have errored")
	}
	observedMessage := customError.Error()
	expectedError := "Custom(1024, This spider family wards off bugs: /\\oo/\\ //\\(oo)//\\ /\\oo/\\, true)"
	// The first error is server side. The second error is client side ABI decoding.
	expectedMessage := fmt.Sprintf("execution reverted: error %v: %v", expectedError, expectedError)
	if observedMessage != expectedMessage {
		Fatal(t, observedMessage)
	}

	arbSys, err := precompilesgen.NewArbSys(arbos.ArbSysAddress, builder.L2.Client)
	Require(t, err, "could not bind ArbSys contract")
	_, customError = arbSys.ArbBlockHash(callOpts, big.NewInt(1e9))
	if customError == nil {
		Fatal(t, "out of range ArbBlockHash call should have errored")
	}
	observedMessage = customError.Error()
	expectedError = "InvalidBlockNumber(1000000000, 1)"
	expectedMessage = fmt.Sprintf("execution reverted: error %v: %v", expectedError, expectedError)
	if observedMessage != expectedMessage {
		Fatal(t, observedMessage)
	}
}

func TestPrecompileErrorGasLeft(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)
	_, _, simple, err := mocksgen.DeploySimple(&auth, builder.L2.Client)
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

func TestPrecompileEmulatedRevert(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)
	_, _, simple, err := mocksgen.DeploySimple(&auth, builder.L2.Client)
	Require(t, err)

	arbDebug, err := precompilesgen.ArbDebugMetaData.GetAbi()
	Require(t, err)

	gasEmulRevertPack, err := simple.CheckGasUsed(&bind.CallOpts{Context: ctx}, common.HexToAddress("0xff"), arbDebug.Methods["emulateRevertPackingOutput"].ID)
	Require(t, err)

	gasRevertPack, err := simple.CheckGasUsed(&bind.CallOpts{Context: ctx}, common.HexToAddress("0xff"), arbDebug.Methods["revertPackingOutput"].ID)
	Require(t, err)

	if gasRevertPack.Cmp(gasEmulRevertPack) != 0 {
		Fatal(t, "gasRevert: ", gasRevertPack, " emulated: ", gasEmulRevertPack)
	}

	_, _, multiCaller, err := mocksgen.DeployMultiCallTest(&auth, builder.L2.Client)
	Require(t, err)

	checkDebugFuncReverts := func(methodName string) {
		funcId := arbDebug.Methods[methodName].ID
		args := argsForMulticall(vm.CALL, common.HexToAddress("0xff"), nil, funcId)
		// emit event and allow revert
		args[5] = args[5] | 0xC
		tx, err := multiCaller.Fallback(&auth, args)
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, builder.L2.Client, tx)
		Require(t, err)
		if len(receipt.Logs) != 1 {
			Fatal(t, methodName, " calling from multi got wrong num of logs")
		}
		calledEvt, err := multiCaller.ParseCalled(*receipt.Logs[0])
		Require(t, err)
		if calledEvt.Success {
			Fatal(t, methodName, "did not revert")
		}
	}
	checkDebugFuncReverts("revertPackingOutput")
	checkDebugFuncReverts("emulateRevertPackingOutput")
}

func TestScheduleArbosUpgrade(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(common.HexToAddress("0x6b"), builder.L2.Client)
	Require(t, err, "could not bind ArbOwner contract")

	arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), builder.L2.Client)
	Require(t, err, "could not bind ArbOwner contract")

	callOpts := &bind.CallOpts{Context: ctx}
	scheduled, err := arbOwnerPublic.GetScheduledUpgrade(callOpts)
	Require(t, err, "failed to call GetScheduledUpgrade before scheduling upgrade")
	if scheduled.ArbosVersion != 0 || scheduled.ScheduledForTimestamp != 0 {
		t.Errorf("expected no upgrade to be scheduled, got version %v timestamp %v", scheduled.ArbosVersion, scheduled.ScheduledForTimestamp)
	}

	// Schedule a noop upgrade, which should test GetScheduledUpgrade in the same way an already completed upgrade would.
	tx, err := arbOwner.ScheduleArbOSUpgrade(&auth, 1, 1)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	scheduled, err = arbOwnerPublic.GetScheduledUpgrade(callOpts)
	Require(t, err, "failed to call GetScheduledUpgrade after scheduling noop upgrade")
	if scheduled.ArbosVersion != 0 || scheduled.ScheduledForTimestamp != 0 {
		t.Errorf("expected completed scheduled upgrade to be ignored, got version %v timestamp %v", scheduled.ArbosVersion, scheduled.ScheduledForTimestamp)
	}

	// TODO: Once we have an ArbOS 30, test a real upgrade with it
	// We can't test 11 -> 20 because 11 doesn't have the GetScheduledUpgrade method we want to test
	var testVersion uint64 = 100
	var testTimestamp uint64 = 1 << 62
	tx, err = arbOwner.ScheduleArbOSUpgrade(&auth, 100, 1<<62)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	scheduled, err = arbOwnerPublic.GetScheduledUpgrade(callOpts)
	Require(t, err, "failed to call GetScheduledUpgrade after scheduling upgrade")
	if scheduled.ArbosVersion != testVersion || scheduled.ScheduledForTimestamp != testTimestamp {
		t.Errorf("expected upgrade to be scheduled for version %v timestamp %v, got version %v timestamp %v", testVersion, testTimestamp, scheduled.ArbosVersion, scheduled.ScheduledForTimestamp)
	}
}
