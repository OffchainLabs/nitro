package arbtest

import (
	"context"
	"testing"

	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func TestDebugAPI(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	l2rpc := builder.L2.Stack.Attach()

	var dump state.Dump
	err := l2rpc.CallContext(ctx, &dump, "debug_dumpBlock", rpc.LatestBlockNumber)
	Require(t, err)
	err = l2rpc.CallContext(ctx, &dump, "debug_dumpBlock", rpc.PendingBlockNumber)
	Require(t, err)

	var badBlocks []eth.BadBlockArgs
	err = l2rpc.CallContext(ctx, &badBlocks, "debug_getBadBlocks")
	Require(t, err)

	var dumpIt state.IteratorDump
	err = l2rpc.CallContext(ctx, &dumpIt, "debug_accountRange", rpc.LatestBlockNumber, hexutil.Bytes{}, 10, true, true, false)
	Require(t, err)
	err = l2rpc.CallContext(ctx, &dumpIt, "debug_accountRange", rpc.PendingBlockNumber, hexutil.Bytes{}, 10, true, true, false)
	Require(t, err)

	arbSys, err := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L2.Client)
	Require(t, err)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := arbSys.WithdrawEth(&auth, common.Address{})
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	if len(receipt.Logs) != 1 {
		Fatal(t, "Unexpected number of logs", len(receipt.Logs))
	}

	var result json.RawMessage
	err = l2rpc.CallContext(ctx, &result, "debug_traceTransaction", tx.Hash(), "{\"tracer\": \"flatCallTracer\"}")
	Require(t, err)
}
