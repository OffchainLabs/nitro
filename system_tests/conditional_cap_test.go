// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/testhelpers/env"
)

const conditionalCap = 2

func TestSequencerConditionalCap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.execConfig.TxPreChecker.MaxPendingConditionalTxs = conditionalCap
	cleanup := builder.Build(t)
	defer cleanup()

	testConditionalCap(t, ctx, builder, builder.L2)
}

func TestPrecheckerConditionalCap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ipcPath := tmpPath(t, "test.ipc")
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.BatchPoster.Enable = false
	builder.l2StackConfig.IPCPath = ipcPath
	cleanup := builder.Build(t)
	defer cleanup()

	nodeConfig := arbnode.ConfigDefaultL1Test()
	nodeConfig.Sequencer = false
	nodeConfig.DelayedSequencer.Enable = false
	nodeConfig.BatchPoster.Enable = false
	execConfig := ExecConfigDefaultTest(t, env.GetTestStateScheme())
	execConfig.Sequencer.Enable = false
	execConfig.Forwarder.RedisUrl = ""
	execConfig.ForwardingTarget = ipcPath
	execConfig.TxPreChecker.MaxPendingConditionalTxs = conditionalCap
	prechecker, cleanupPrechecker := builder.Build2ndNode(t, &SecondNodeParams{
		nodeConfig: nodeConfig,
		execConfig: execConfig,
	})
	defer cleanupPrechecker()

	testConditionalCap(t, ctx, builder, prechecker)
}

func testConditionalCap(t *testing.T, ctx context.Context, builder *NodeBuilder, target *TestClient) {
	t.Helper()
	builder.L2Info.GenerateAccount("User")
	prepare := func() *types.Transaction {
		return builder.L2Info.PrepareTx("Owner", "User", builder.L2Info.TransferGas, common.Big1, nil)
	}
	var held []*types.Transaction
	for range conditionalCap {
		held = append(held, prepare())
	}
	plain := prepare()
	rejected := prepare()
	options := &arbitrum_types.ConditionalOptions{}
	sendConditional := func(tx *types.Transaction) error {
		return arbitrum.SendConditionalTransactionRPC(ctx, target.ConsensusNode.Stack.Attach(), tx, options)
	}

	builder.L2.ExecNode.Pause()
	results := make(chan error, len(held)+1)
	for _, tx := range held {
		go func() { results <- sendConditional(tx) }()
	}
	waitForSequencerQueue(t, builder, conditionalCap)

	err := sendConditional(rejected)
	var want rpc.Error
	if !errors.As(gethexec.ErrTooManyPendingConditionalTxs, &want) {
		Fatal(t, "ErrTooManyPendingConditionalTxs does not carry an rpc error code")
	}
	var got rpc.Error
	if !errors.As(err, &got) {
		Fatal(t, "cap rejection is not an rpc error:", err)
	}
	if got.ErrorCode() != want.ErrorCode() {
		Fatal(t, "unexpected cap rejection code, have:", got.ErrorCode(), "want:", want.ErrorCode())
	}
	if got.Error() != want.Error() {
		Fatal(t, "unexpected cap rejection message, have:", got.Error(), "want:", want.Error())
	}

	go func() { results <- target.Client.SendTransaction(ctx, plain) }()
	waitForSequencerQueue(t, builder, conditionalCap+1)

	builder.L2.ExecNode.Activate()
	for range cap(results) {
		Require(t, <-results)
	}
	for _, tx := range held {
		_, err := builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}
	_, err = builder.L2.EnsureTxSucceeded(plain)
	Require(t, err)
	requireNoReceipt(t, ctx, builder, rejected)
}
