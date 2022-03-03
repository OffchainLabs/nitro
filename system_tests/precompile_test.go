//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/solgen/go/precompilesgen"
)

func TestPurePrecompileMethodCalls(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _, client := CreateTestL2(t, ctx)

	arbSys, err := precompilesgen.NewArbSys(common.HexToAddress("0x64"), client)
	Require(t, err, "could not deploy ArbSys contract")
	chainId, err := arbSys.ArbChainID(&bind.CallOpts{})
	Require(t, err, "failed to get the ChainID")
	if chainId.Uint64() != params.ArbitrumDevTestChainConfig().ChainID.Uint64() {
		Fail(t, "Wrong ChainID", chainId.Uint64())
	}
}
