package arbtest

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func testDebugBlockInjection(t *testing.T, production bool) {
	t.Run("with-other-tx", func(t *testing.T) {
		testDebugBlockInjectionImpl(t, production, true)
	})
	t.Run("without-other-tx", func(t *testing.T) {
		testDebugBlockInjectionImpl(t, production, false)
	})
}

func testDebugBlockInjectionImpl(t *testing.T, production bool, withOtherTx bool) {
	expectInject := !production
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	startBlock, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)

	// send a transaction to advance the chain
	builder.L2Info.GenerateAccount("SomeUser")
	tx := builder.L2Info.PrepareTx("Owner", "SomeUser", builder.L2Info.TransferGas, common.Big1, nil)
	builder.L2.SendWaitTestTransactions(t, types.Transactions{tx})

	// make sure that DebugUser can't send a tx just yet
	builder.L2Info.GenerateAccount("DebugUser")
	debugUserTx := builder.L2Info.PrepareTx("DebugUser", "SomeUser", builder.L2Info.TransferGas, common.Big1, nil)
	err = builder.L2.Client.SendTransaction(ctx, debugUserTx)
	if err == nil {
		t.Fatal("debugUserTx shouldn't have succeeded before prefunding DebugUser account")
	}

	// make sure the chain advanced
	lastBlock := startBlock
	advanced := pollWithDeadlineDefault(t, func() bool {
		var err error
		lastBlock, err = builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
		return lastBlock > startBlock
	})
	if !advanced {
		t.Fatal("failed to advance chain: timeout exceeded")
	}

	builder.L2.cleanup()
	builder.L2.cleanup = func() {}
	t.Log("l2 node stopped")

	// configure debug block injection
	debugBlockNum := lastBlock + 1
	builder.execConfig.Dangerous.DebugBlock.OverwriteChainConfig = true
	builder.execConfig.Dangerous.DebugBlock.DebugBlockNum = debugBlockNum
	builder.execConfig.Dangerous.DebugBlock.DebugAddress = builder.L2Info.GetInfoWithPrivKey("DebugUser").Address.String()

	if production {
		err := builder.execConfig.Validate()
		if err == nil {
			t.Fatal("execConfig validation should have failed in production build when chain config overwrite is specified")
		} else if !strings.Contains(err.Error(), "debug block injection is not supported") {
			t.Fatal("execConfig validation failed with unexpected error, err:", err)
		}
		// ignore execConfig validation failure during restart, to test that the dangerous configs don't have effect in production
		builder.IgnoreExecConfigValidationError()
	}

	builder.RestartL2Node(t)
	t.Log("restarted l2 node")

	if withOtherTx {
		tx := builder.L2Info.PrepareTx("Owner", "SomeUser", builder.L2Info.TransferGas, common.Big1, nil)
		builder.L2.SendWaitTestTransactions(t, types.Transactions{tx})
	}

	interval := 25 * time.Millisecond
	timeout := 5 * time.Second
	if !expectInject && !withOtherTx {
		// shorter deadline for expected timeout
		timeout = 100 * time.Millisecond
	}

	debugBlockReached := pollWithDeadline(t, interval, timeout, func() bool {
		current, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
		t.Log("current block:", current, "debug block:", debugBlockNum)
		return current >= debugBlockNum
	})

	if expectInject {
		if !debugBlockReached {
			t.Fatalf("debug block number not reached: %v timeout exceeded", timeout)
		}
		// make sure that DebugUser can send a tx now
		builder.L2.SendWaitTestTransactions(t, types.Transactions{debugUserTx})
	} else {
		if debugBlockReached && !withOtherTx {
			t.Error("debug block number reached with no other txes to advance chain")
		}
		// make sure that DebugUser still can't send a tx
		err = builder.L2.Client.SendTransaction(ctx, debugUserTx)
		if err == nil {
			t.Fatal("debugUserTx shouldn't have succeeded in production build")
		}
	}

}
