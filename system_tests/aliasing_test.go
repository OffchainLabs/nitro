// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func TestAliasing(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	user := builder.L1Info.GetDefaultTransactOpts("User", ctx)
	builder.L2.TransferBalanceTo(t, "Owner", util.RemapL1Address(user.From), big.NewInt(1e18), builder.L2Info)

	simpleAddr, simple := builder.L2.DeploySimple(t, auth)
	simpleContract, err := abi.JSON(strings.NewReader(mocksgen.SimpleABI))
	Require(t, err)

	// Test direct calls
	arbsys, err := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L2.Client)
	Require(t, err)
	top, err := arbsys.IsTopLevelCall(nil)
	Require(t, err)
	was, err := arbsys.WasMyCallersAddressAliased(nil)
	Require(t, err)
	alias, err := arbsys.MyCallersAddressWithoutAliasing(nil)
	Require(t, err)
	if !top {
		Fatal(t, "direct call is not top level")
	}
	if was || alias != (common.Address{}) {
		Fatal(t, "direct call has an alias", was, alias)
	}

	testL2Signed := func(top, direct, static, delegate, callcode, call bool) {
		t.Helper()

		// check via L2
		tx, err := simple.CheckCalls(&auth, top, direct, static, delegate, callcode, call)
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)

		// check signed txes via L1
		data, err := simpleContract.Pack("checkCalls", top, direct, static, delegate, callcode, call)
		Require(t, err)
		tx = builder.L2Info.PrepareTxTo("Owner", &simpleAddr, 500000, big.NewInt(0), data)
		builder.L1.SendSignedTx(t, builder.L2.Client, tx, builder.L1Info)
	}

	testUnsigned := func(top, direct, static, delegate, callcode, call bool) {
		t.Helper()

		// check unsigned txes via L1
		data, err := simpleContract.Pack("checkCalls", top, direct, static, delegate, callcode, call)
		Require(t, err)
		tx := builder.L2Info.PrepareTxTo("Owner", &simpleAddr, 500000, big.NewInt(0), data)
		builder.L1.SendUnsignedTx(t, builder.L2.Client, tx, builder.L1Info)
	}

	testL2Signed(true, true, false, false, false, false)
	testL2Signed(false, false, false, false, false, false)
	testUnsigned(true, true, false, false, false, false)
	testUnsigned(false, true, false, true, false, false)
}
