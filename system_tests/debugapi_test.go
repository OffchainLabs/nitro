package arbtest

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestDebugAPI(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	testNode := NewNodeBuilder(ctx).SetIsSequencer(true).CreateTestNodeOnL1AndL2(t)
	defer requireClose(t, testNode.L1Stack)
	defer requireClose(t, testNode.L2Stack)

	l2rpc, _ := testNode.L2Stack.Attach()

	var dump state.Dump
	err := l2rpc.CallContext(ctx, &dump, "debug_dumpBlock", rpc.LatestBlockNumber)
	testhelpers.RequireImpl(t, err)
	err = l2rpc.CallContext(ctx, &dump, "debug_dumpBlock", rpc.PendingBlockNumber)
	testhelpers.RequireImpl(t, err)

	var badBlocks []eth.BadBlockArgs
	err = l2rpc.CallContext(ctx, &badBlocks, "debug_getBadBlocks")
	testhelpers.RequireImpl(t, err)

	var dumpIt state.IteratorDump
	err = l2rpc.CallContext(ctx, &dumpIt, "debug_accountRange", rpc.LatestBlockNumber, hexutil.Bytes{}, 10, true, true, false)
	testhelpers.RequireImpl(t, err)
	err = l2rpc.CallContext(ctx, &dumpIt, "debug_accountRange", rpc.PendingBlockNumber, hexutil.Bytes{}, 10, true, true, false)
	testhelpers.RequireImpl(t, err)

}
