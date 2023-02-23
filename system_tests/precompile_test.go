// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func TestPurePrecompileMethodCalls(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, node, client := CreateTestL2(t, ctx)
	defer node.StopAndWait()

	arbSys, err := precompilesgen.NewArbSys(common.HexToAddress("0x64"), client)
	Require(t, err, "could not deploy ArbSys contract")
	chainId, err := arbSys.ArbChainID(&bind.CallOpts{})
	Require(t, err, "failed to get the ChainID")
	if chainId.Uint64() != params.ArbitrumDevTestChainConfig().ChainID.Uint64() {
		Fail(t, "Wrong ChainID", chainId.Uint64())
	}
}

func TestCustomSolidityErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, node, client := CreateTestL2(t, ctx)
	defer node.StopAndWait()

	callOpts := &bind.CallOpts{Context: ctx}
	arbDebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), client)
	Require(t, err, "could not bind ArbDebug contract")
	customError := arbDebug.CustomRevert(callOpts, 1024)
	if customError == nil {
		Fail(t, "customRevert call should have errored")
	}
	observedMessage := customError.Error()
	expectedMessage := "execution reverted: error Custom(1024, This spider family wards off bugs: /\\oo/\\ //\\(oo)/\\ /\\oo/\\, true)"
	if observedMessage != expectedMessage {
		Fail(t, observedMessage)
	}

	arbSys, err := precompilesgen.NewArbSys(arbos.ArbSysAddress, client)
	Require(t, err, "could not bind ArbSys contract")
	_, customError = arbSys.ArbBlockHash(callOpts, big.NewInt(1e9))
	if customError == nil {
		Fail(t, "out of range ArbBlockHash call should have errored")
	}
	observedMessage = customError.Error()
	expectedMessage = "execution reverted: error InvalidBlockNumber(1000000000, 1)"
	if observedMessage != expectedMessage {
		Fail(t, observedMessage)
	}
}

func TestPrecompileErrorGasLeft(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	info, node, client := CreateTestL2(t, ctx)
	defer node.StopAndWait()

	auth := info.GetDefaultTransactOpts("Faucet", ctx)
	_, _, simple, err := mocksgen.DeploySimple(&auth, client)
	Require(t, err)

	assertNotAllGasConsumed := func(to common.Address, input []byte) {
		gas, err := simple.CheckGasUsed(&bind.CallOpts{Context: ctx}, to, input)
		Require(t, err, "Failed to call CheckGasUsed to precompile", to)
		maxGas := big.NewInt(100_000)
		if arbmath.BigGreaterThan(gas, maxGas) {
			Fail(t, "Precompile", to, "used", gas, "gas reverting, greater than max expected", maxGas)
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
