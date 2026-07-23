// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbos

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/system_tests/v2/systest"
	"github.com/offchainlabs/nitro/util/arbmath"
)

var precompileSanityTests = []systest.Scenario{
	systest.Test(testRunViewLogReverts),
	systest.Test(testRunArbDebugPanic),
	systest.Test(testRunArbDebugLegacyError),
	systest.Test(testRunCustomSolidityErrors),
	systest.Test(testRunPrecompileErrorGasLeft),
	systest.Test(testRunPurePrecompileMethodCalls, systest.WithArbOS(params.ArbosVersion_31)),
}

func testRunViewLogReverts(env *systest.Env) {
	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, env.L2.Client)
	env.Require(err, "could not bind ArbDebug")
	err = arbDebug.EventsView(nil)
	env.NotNil(err, "unexpected success from EventsView")
}

func testRunArbDebugPanic(env *systest.Env) {
	auth := env.L2.TransactOpts("Owner")
	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, env.L2.Client)
	env.Require(err)
	_, err = arbDebug.Panic(&auth)
	env.NotNil(err, "unexpected success from Panic")
	env.Equal("method handler crashed", err.Error(), "unexpected Panic error")
}

func testRunArbDebugLegacyError(env *systest.Env) {
	callOpts := &bind.CallOpts{Context: env.Ctx}
	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, env.L2.Client)
	env.Require(err)
	err = arbDebug.LegacyError(callOpts)
	env.NotNil(err, "unexpected success from LegacyError")
}

func testRunCustomSolidityErrors(env *systest.Env) {
	callOpts := &bind.CallOpts{Context: env.Ctx}
	auth := env.L2.TransactOpts("Owner")

	ensure := func(customError error, expectedError, scenario string) {
		env.NotNil(customError, "should have errored, scenario %s", scenario)
		expected := fmt.Sprintf("execution reverted: error %v: %v", expectedError, expectedError)
		env.Equal(expected, customError.Error(), "scenario %s", scenario)
	}

	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, env.L2.Client)
	env.Require(err, "could not bind ArbDebug contract")
	ensure(
		arbDebug.CustomRevert(callOpts, 1024),
		`Custom(1024, This spider family wards off bugs: /\oo/\ //\(oo)//\ /\oo/\, true)`,
		"arbDebug.CustomRevert",
	)

	arbSys, err := precompilesgen.NewArbSys(types.ArbSysAddress, env.L2.Client)
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

func testRunPrecompileErrorGasLeft(env *systest.Env) {
	auth := env.L2.TransactOpts("Faucet")
	_, _, simple, err := localgen.DeploySimple(&auth, env.L2.Client)
	env.Require(err)

	assertNotAllGasConsumed := func(to common.Address, input []byte) {
		gas, err := simple.CheckGasUsed(&bind.CallOpts{Context: env.Ctx}, to, input)
		env.Require(err, "failed to call CheckGasUsed to precompile %v", to)
		maxGas := big.NewInt(100_000)
		env.True(!arbmath.BigGreaterThan(gas, maxGas), "precompile %v used %v gas reverting, greater than max expected %v", to, gas, maxGas)
	}

	arbSysABI, err := precompilesgen.ArbSysMetaData.GetAbi()
	env.Require(err)
	arbBlockHash := arbSysABI.Methods["arbBlockHash"]
	data, err := arbBlockHash.Inputs.Pack(big.NewInt(1e9))
	env.Require(err)
	input := append([]byte{}, arbBlockHash.ID...)
	input = append(input, data...)
	assertNotAllGasConsumed(types.ArbSysAddress, input)

	arbDebugABI, err := precompilesgen.ArbDebugMetaData.GetAbi()
	env.Require(err)
	assertNotAllGasConsumed(types.ArbDebugAddress, arbDebugABI.Methods["legacyError"].ID)
}

func testRunPurePrecompileMethodCalls(env *systest.Env) {
	arbSys, err := precompilesgen.NewArbSys(types.ArbSysAddress, env.L2.Client)
	env.Require(err, "could not deploy ArbSys contract")
	chainId, err := arbSys.ArbChainID(&bind.CallOpts{})
	env.Require(err, "failed to get the ChainID")
	env.Equal(chaininfo.ArbitrumDevTestChainConfig().ChainID.Uint64(), chainId.Uint64(), "wrong ChainID")

	arbosVersion := params.ArbosVersion_31
	expectedArbosVersion := 55 + arbosVersion // Nitro versions start at 56
	arbSysArbosVersion, err := arbSys.ArbOSVersion(&bind.CallOpts{})
	env.Require(err)
	env.Equal(expectedArbosVersion, arbSysArbosVersion.Uint64(), "unexpected ArbOS version")

	storageGasAvailable, err := arbSys.GetStorageGasAvailable(&bind.CallOpts{})
	env.Require(err)
	env.EqualBig(big.NewInt(0), storageGasAvailable, "unexpected storage gas available")
}
