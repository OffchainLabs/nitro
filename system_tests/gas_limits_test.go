// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func TestBlockGasLimit(t *testing.T) {
	ctx := t.Context()

	noL1 := false
	b := NewNodeBuilder(ctx).DefaultConfig(t, noL1)
	b.takeOwnership = true

	cleanup := b.Build(t)
	defer cleanup()

	auth := b.L2Info.GetDefaultTransactOpts("Owner", ctx)

	_, bigMap := b.L2.DeployBigMap(t, auth)

	// It is crucial to set the gas limit to avoid estimating gas and having
	// the gas estimation fail when it runs out of gas. This test uses the
	// receipt from the execution of the transaction to check the gas used.
	auth.GasLimit = uint64(50_000_000)
	// Store enough values to use just over 32M gas
	toAdd := big.NewInt(1423)
	toClear := big.NewInt(0)

	overboundTx, err := bigMap.ClearAndAddValues(&auth, toClear, toAdd)
	Require(t, err)
	r := EnsureTxFailed(t, ctx, b.L2.Client, overboundTx)
	// Should run out of gas at the transaction limit
	got := r.GasUsedForL2()
	// This should be exactly the transaction gas limit as of ArbOS 50.
	want := l2pricing.InitialPerTxGasLimitV50
	if got != want {
		t.Fatalf("want: %d gas used, got: %d", want, got)
	}

	// set block gas-limit to 1.5 times the transaction limit
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, b.L2.Client)
	Require(t, err)
	ownerTx, err := arbOwner.SetMaxBlockGasLimit(&auth, l2pricing.InitialPerTxGasLimitV50*3/2)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, b.L2.Client, ownerTx)
	Require(t, err)

	// create a successful transaction that consumes a little less than 32M gas
	toAddSuccesfull := big.NewInt(1420)
	succesfullTx, err := bigMap.ClearAndAddValues(&auth, toClear, toAddSuccesfull)
	Require(t, err)
	lastReceipt, err := EnsureTxSucceeded(ctx, b.L2.Client, succesfullTx)
	Require(t, err)

	// send 3 transactions to the sequencer to be sequenced in the same block, each almost consuming the tx limit
	txes := types.Transactions{}
	for i := 0; i < 3; i++ {
		tx := b.L2Info.PrepareTxTo("Owner", succesfullTx.To(), 50_000_000, big.NewInt(0), succesfullTx.Data())
		txes = append(txes, tx)
	}
	header := &arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_L2Message,
		Poster:      l1pricing.BatchPosterAddress,
		BlockNumber: lastReceipt.BlockNumber.Uint64() + 1,
		Timestamp:   arbmath.SaturatingUCast[uint64](time.Now().Unix()),
		RequestId:   nil,
		L1BaseFee:   nil,
	}
	hooks := gethexec.MakeZeroTxSizeSequencingHooksForTesting(txes, nil, nil, nil)
	_, err = b.L2.ExecNode.ExecEngine.SequenceTransactions(header, hooks, nil)
	Require(t, err)

	// as block gas-limit is 1.5txs, and it's a soft limit - first two transactions should pass
	// 3rd tx will never be included because the block is over the soft limit before reaching it
	receipt0, err := EnsureTxSucceeded(ctx, b.L2.Client, txes[0])
	Require(t, err)
	receipt1, err := EnsureTxSucceeded(ctx, b.L2.Client, txes[1])
	Require(t, err)
	if receipt0.BlockNumber.Uint64() != receipt1.BlockNumber.Uint64() {
		t.Error("two transactions should have been in the same block")
	}
	_, err = WaitForTx(ctx, b.L2.Client, txes[2].Hash(), time.Second)
	if err == nil {
		t.Error("got 3rd tx which should not be there ")
	}
}
