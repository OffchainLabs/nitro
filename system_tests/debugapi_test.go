package arbtest

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

type account struct {
	Balance *hexutil.Big                `json:"balance,omitempty"`
	Code    []byte                      `json:"code,omitempty"`
	Nonce   uint64                      `json:"nonce,omitempty"`
	Storage map[common.Hash]common.Hash `json:"storage,omitempty"`
}

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
	tx, err := arbSys.SendTxToL1(&auth, common.Address{}, []byte{})
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	if len(receipt.Logs) != 1 {
		Fatal(t, "Unexpected number of logs", len(receipt.Logs))
	}

	var result json.RawMessage
	flatCallTracer := "flatCallTracer"
	err = l2rpc.CallContext(ctx, &result, "debug_traceTransaction", tx.Hash(), &tracers.TraceConfig{Tracer: &flatCallTracer})
	Require(t, err)
}

func TestPrestateTracerRegistersArbitrumStorage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), builder.L2.Client)
	Require(t, err, "could not bind ArbOwner contract")

	// Schedule a noop upgrade
	tx, err := arbOwner.ScheduleArbOSUpgrade(&auth, 1, 1)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	statedb, err := builder.L2.ExecNode.Backend.ArbInterface().BlockChain().State()
	Require(t, err)
	burner := burn.NewSystemBurner(nil, false)
	arbosSt, err := arbosState.OpenArbosState(statedb, burner)
	Require(t, err)

	l2rpc := builder.L2.Stack.Attach()
	var result map[common.Address]*account
	prestateTracer := "prestateTracer"
	err = l2rpc.CallContext(ctx, &result, "debug_traceTransaction", tx.Hash(), &tracers.TraceConfig{Tracer: &prestateTracer})
	Require(t, err)

	// ArbOSVersion and BrotliCompressionLevel storage slots should be accessed by arbos so we check if the current values of these appear in the prestateTrace
	arbOSVersionHash := common.BigToHash(new(big.Int).SetUint64(arbosSt.ArbOSVersion()))
	bcl, err := arbosSt.BrotliCompressionLevel()
	Require(t, err)
	bclHash := common.BigToHash(new(big.Int).SetUint64(bcl))

	if _, ok := result[types.ArbosStateAddress]; !ok {
		t.Fatal("ArbosStateAddress storage accesses not logged in the prestateTracer's trace")
	}

	found := 0
	for _, val := range result[types.ArbosStateAddress].Storage {
		if val == arbOSVersionHash || val == bclHash {
			found++
		}
	}
	if found != 2 {
		t.Fatal("ArbosStateAddress storage accesses for ArbOSVersion and BrotliCompressionLevel not logged in the prestateTracer's trace")
	}
}
