package arbtest

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbos/programs"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	pgen "github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type account struct {
	Balance         *hexutil.Big                `json:"balance,omitempty"`
	Code            hexutil.Bytes               `json:"code,omitempty"`
	Nonce           uint64                      `json:"nonce,omitempty"`
	Storage         map[common.Hash]common.Hash `json:"storage,omitempty"`
	ArbitrumStorage map[common.Hash]common.Hash `json:"arbitrumStorage,omitempty"`
}
type prestateTrace struct {
	Post map[common.Address]*account `json:"post"`
	Pre  map[common.Address]*account `json:"pre"`
}

func TestPrestateTracerArbitrumStorage(t *testing.T) {
	builder, ownerAuth, cleanup := setupProgramTest(t, true)
	ctx := builder.ctx
	l2client := builder.L2.Client
	l2info := builder.L2Info
	defer cleanup()

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}
	assert := func(cond bool, err error, msg ...interface{}) {
		t.Helper()
		Require(t, err)
		if !cond {
			Fatal(t, msg...)
		}
	}

	// precompiles we plan to use
	arbWasm, err := pgen.NewArbWasm(types.ArbWasmAddress, builder.L2.Client)
	Require(t, err)
	arbWasmCache, err := pgen.NewArbWasmCache(types.ArbWasmCacheAddress, builder.L2.Client)
	Require(t, err)
	arbOwner, err := pgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	ensure(arbOwner.SetInkPrice(&ownerAuth, 10_000))
	parseLog := logParser[pgen.ArbWasmCacheUpdateProgramCache](t, pgen.ArbWasmCacheABI, "UpdateProgramCache")

	// fund a user account we'll use to probe access-restricted methods
	l2info.GenerateAccount("Anyone")
	userAuth := l2info.GetDefaultTransactOpts("Anyone", ctx)
	userAuth.GasLimit = 3e6
	TransferBalance(t, "Owner", "Anyone", arbmath.BigMulByUint(oneEth, 32), l2info, l2client, ctx)

	// deploy without activating a wasm
	wasm, _ := readWasmFile(t, rustFile("keccak"))
	program := deployContract(t, ctx, userAuth, l2client, wasm)
	codehash := crypto.Keccak256Hash(wasm)

	// athorize the manager
	manager, tx, mock, err := mocksgen.DeploySimpleCacheManager(&ownerAuth, l2client)
	ensure(tx, err)
	isManager, err := arbWasmCache.IsCacheManager(nil, manager)
	assert(!isManager, err)
	ensure(arbOwner.AddWasmCacheManager(&ownerAuth, manager))
	assert(arbWasmCache.IsCacheManager(nil, manager))
	all, err := arbWasmCache.AllCacheManagers(nil)
	assert(len(all) == 1 && all[0] == manager, err)

	// cache the active program
	activateWasm(t, ctx, userAuth, l2client, program, "keccak")
	cacheTx, err := mock.CacheProgram(&userAuth, program)
	ensure(cacheTx, err)
	assert(arbWasmCache.CodehashIsCached(nil, codehash))

	l2rpc := builder.L2.Stack.Attach()

	var result prestateTrace
	traceConfig := map[string]interface{}{
		"tracer": "prestateTracer",
		"tracerConfig": map[string]interface{}{
			"diffMode": true,
		},
	}
	err = l2rpc.CallContext(ctx, &result, "debug_traceTransaction", cacheTx.Hash(), traceConfig)
	Require(t, err)

	// Validate trace result
	_, ok := result.Pre[manager]
	assert(ok, nil, "manager address not found in pre section of trace")
	assert(result.Pre[manager].ArbitrumStorage != nil, nil, "changes to arbitrum storage not picked up by prestate tracer")
	_, ok = result.Pre[manager].ArbitrumStorage[codehash]
	assert(ok, nil, "activated program's codehash key not found in the arbitrum storage trace entry for manager address in Pre")
	preData := result.Pre[manager].ArbitrumStorage[codehash]

	_, ok = result.Post[manager]
	assert(ok, nil, "manager address not found in post section oftrace")
	assert(result.Post[manager].ArbitrumStorage != nil, nil, "changes to arbitrum storage not picked up by prestate tracer")
	_, ok = result.Post[manager].ArbitrumStorage[codehash]
	assert(ok, nil, "activated program's codehash key not found in the arbitrum storage trace entry for manager address in Post")
	postData := result.Post[manager].ArbitrumStorage[codehash]

	// since we are just caching the program the only thing that should differ between the pre and post values is the cached byte
	assert(!(preData == postData), nil, "preData and postData shouldnt be equal")
	assert(bytes.Equal(preData[:14], postData[:14]), nil, "preData and postData should only differ in cached byte")
	assert(bytes.Equal(preData[15:], postData[15:]), nil, "preData and postData should only differ in cached byte")
	assert(!arbmath.BytesToBool(preData[14:15]), nil, "cached byte of preData should be false")
	assert(arbmath.BytesToBool(postData[14:15]), nil, "cached byte of postData should be true")

	version, err := arbWasm.StylusVersion(nil)
	assert(arbmath.BytesToUint16(postData[:2]) == version, err, "stylus version mismatch")

	programMemoryFootprint, err := arbWasm.ProgramMemoryFootprint(nil, program)
	assert(arbmath.BytesToUint16(postData[6:8]) == programMemoryFootprint, err, "programMemoryFootprint mismatch")

	codehashAsmSize, err := arbWasm.CodehashAsmSize(nil, codehash)
	codehashAsmSizeFromTrace := arbmath.SaturatingUMul(arbmath.BytesToUint24(postData[11:14]).ToUint32(), 1024)
	assert(codehashAsmSizeFromTrace == codehashAsmSize, err, "codehashAsmSize mismatch")

	hourNow := (time.Now().Unix() - programs.ArbitrumStartTime) / 3600
	hourActivatedFromTrace := arbmath.BytesToUint24(postData[8:11])
	// #nosec G115
	assert(uint64(hourActivatedFromTrace) == uint64(hourNow), nil, "wrong activated time in trace")

	// compare gas costs
	keccak := func() uint64 {
		tx := l2info.PrepareTxTo("Owner", &program, 1e9, nil, []byte{0x00})
		return ensure(tx, l2client.SendTransaction(ctx, tx)).GasUsedForL2()
	}
	ensure(mock.EvictProgram(&userAuth, program))
	miss := keccak()
	ensure(mock.CacheProgram(&userAuth, program))
	hits := keccak()
	cost, err := arbWasm.ProgramInitGas(nil, program)
	assert(hits-cost.GasWhenCached == miss-cost.Gas, err)
	empty := len(ensure(mock.CacheProgram(&userAuth, program)).Logs)
	evict := parseLog(ensure(mock.EvictProgram(&userAuth, program)).Logs[0])
	cache := parseLog(ensure(mock.CacheProgram(&userAuth, program)).Logs[0])
	assert(empty == 0 && evict.Manager == manager && !evict.Cached && cache.Codehash == codehash && cache.Cached, nil)
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
