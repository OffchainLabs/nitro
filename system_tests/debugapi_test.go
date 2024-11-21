package arbtest

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/colors"
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

	var dumpIt state.Dump
	err = l2rpc.CallContext(ctx, &dumpIt, "debug_accountRange", rpc.LatestBlockNumber, hexutil.Bytes{}, 10, true, true, false)
	Require(t, err)
	err = l2rpc.CallContext(ctx, &dumpIt, "debug_accountRange", rpc.PendingBlockNumber, hexutil.Bytes{}, 10, true, true, false)
	Require(t, err)

	arbSys, err := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L2.Client)
	Require(t, err)
	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	withdrawalValue := big.NewInt(1000000000)
	auth.Value = withdrawalValue
	tx, err := arbSys.SendTxToL1(&auth, common.Address{}, []byte{})
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	if len(receipt.Logs) != 1 {
		Fatal(t, "Unexpected number of logs", len(receipt.Logs))
	}

	// Use JS tracer
	js := `{
		"onBalanceChange": function(balanceChange) { 
			if (!this.balanceChanges) {
				this.balanceChanges = [];
			}
			this.balanceChanges.push({
				addr: balanceChange.addr,
				prev: balanceChange.prev,
				new: balanceChange.new,
				reason: balanceChange.reason
        	});
		},
		"result": function() { return this.balanceChanges || []; },
		"fault":  function() { return this.names; },
		names: []
	}`
	type balanceChangeJS struct {
		Addr   common.Address `json:"addr"`
		Prev   big.Int        `json:"prev"`
		New    big.Int        `json:"new"`
		Reason string         `json:"reason"`
	}
	var jsTrace []balanceChangeJS
	err = l2rpc.CallContext(ctx, &jsTrace, "debug_traceTransaction", tx.Hash(), &tracers.TraceConfig{Tracer: &js})
	Require(t, err)
	found := false
	for _, balChange := range jsTrace {
		if balChange.Reason == tracing.BalanceDecreaseWithdrawToL1.String() &&
			balChange.Addr == types.ArbSysAddress &&
			balChange.Prev.Cmp(withdrawalValue) == 0 &&
			balChange.New.Cmp(common.Big0) == 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("balanceChanges in tracing via js tracer didn't register withdrawal of funds to L1")
	}

	var result json.RawMessage
	err = l2rpc.CallContext(ctx, &result, "debug_traceTransaction", tx.Hash(), &tracers.TraceConfig{Tracer: &js})
	Require(t, err)
	colors.PrintGrey("balance changes: ", string(result))
}
