// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func TestPurePrecompileMethodCalls(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _, client, l2stack := CreateTestL2(t, ctx)
	defer requireClose(t, l2stack)

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

	_, _, client, l2stack := CreateTestL2(t, ctx)
	defer requireClose(t, l2stack)

	arbDebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), client)
	Require(t, err, "could not deploy ArbDebug contract")
	customError := arbDebug.CustomRevert(&bind.CallOpts{}, 1024)
	if customError == nil {
		Fail(t, "should have errored")
	}
	observedMessage := customError.Error()
	expectedMessage := "execution reverted: error Custom(1024, This spider family wards off bugs: /\\oo/\\ //\\(oo)/\\ /\\oo/\\, true)"
	if observedMessage != expectedMessage {
		Fail(t, observedMessage)
	}
}
