//go:build debugblock

package arbtest

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func TestDebugBlockInjection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	// send a transaction to advance the chain
	builder.L2Info.GenerateAccount("SomeUser")
	tx := builder.L2Info.PrepareTx("Owner", "SomeUser", builder.L2Info.TransferGas, common.Big1, nil)
	err := builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// make sure that DebugUser can't send a tx just yet
	builder.L2Info.GenerateAccount("DebugUser")
	debugUsertx := builder.L2Info.PrepareTx("DebugUser", "SomeUser", builder.L2Info.TransferGas, common.Big1, nil)
	err = builder.L2.Client.SendTransaction(ctx, debugUsertx)
	if err == nil {
		t.Fatal("debugUserTx shouldn't have succeeded before prefunding DebugUser account")
	}

	lastBlock, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)

	builder.L2.cleanup()
	builder.L2.cleanup = func() {}
	t.Log("l2 node stopped")

	// configure debug block injection
	debugBlockNum := lastBlock + 1
	builder.execConfig.Dangerous.DebugBlock.OverwriteChainConfig = true
	builder.execConfig.Dangerous.DebugBlock.DebugBlockNum = debugBlockNum
	builder.execConfig.Dangerous.DebugBlock.DebugAddress = builder.L2Info.GetInfoWithPrivKey("DebugUser").Address.String()

	builder.RestartL2Node(t)
	t.Log("restarted l2 node")

	current, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)
	for current < debugBlockNum {
		<-time.After(100 * time.Millisecond)
		current, err = builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
	}

	// make sure that DebugUser can send a tx now
	err = builder.L2.Client.SendTransaction(ctx, debugUsertx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
}
