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

	testNode := NewNodeBuilder(ctx).SetIsSequencer(true).CreateTestNodeOnL1AndL2(t)
	defer requireClose(t, testNode.L1Stack)
	defer testNode.L2Node.StopAndWait()

	auth := testNode.L2Info.GetDefaultTransactOpts("Owner", ctx)
	user := testNode.L1Info.GetDefaultTransactOpts("User", ctx)
	TransferBalanceTo(t, "Owner", util.RemapL1Address(user.From), big.NewInt(1e18), testNode.L2Info, testNode.L2Client, ctx)

	simpleAddr, simple := testNode.DeploySimple(t, auth)
	simpleContract, err := abi.JSON(strings.NewReader(mocksgen.SimpleABI))
	Require(t, err)

	// Test direct calls
	arbsys, err := precompilesgen.NewArbSys(types.ArbSysAddress, testNode.L2Client)
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
		_, err = EnsureTxSucceeded(ctx, testNode.L2Client, tx)
		Require(t, err)

		// check signed txes via L1
		data, err := simpleContract.Pack("checkCalls", top, direct, static, delegate, callcode, call)
		Require(t, err)
		tx = testNode.L2Info.PrepareTxTo("Owner", &simpleAddr, 500000, big.NewInt(0), data)
		SendSignedTxViaL1(t, ctx, testNode.L1Info, testNode.L1Client, testNode.L2Client, tx)
	}

	testUnsigned := func(top, direct, static, delegate, callcode, call bool) {
		t.Helper()

		// check unsigned txes via L1
		data, err := simpleContract.Pack("checkCalls", top, direct, static, delegate, callcode, call)
		Require(t, err)
		tx := testNode.L2Info.PrepareTxTo("Owner", &simpleAddr, 500000, big.NewInt(0), data)
		SendUnsignedTxViaL1(t, ctx, testNode.L1Info, testNode.L1Client, testNode.L2Client, tx)
	}

	testL2Signed(true, true, false, false, false, false)
	testL2Signed(false, false, false, false, false, false)
	testUnsigned(true, true, false, false, false, false)
	testUnsigned(false, true, false, true, false, false)
}
