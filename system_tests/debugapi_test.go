package arbtest

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/gasestimator"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
)

var (
	txsSeenByTracer    map[common.Hash]struct{}
	blocksSeenByTracer uint64
)

type testTracer struct{}

func newTestTracer(_ json.RawMessage) (*tracing.Hooks, error) {
	t := &testTracer{}
	return &tracing.Hooks{
		OnTxStart:    t.OnTxStart,
		OnBlockStart: t.OnBlockStart,
		OnBlockEnd:   t.OnBlockEnd,
	}, nil
}

func (t *testTracer) OnTxStart(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
	if from != types.ArbosAddress {
		txsSeenByTracer[tx.Hash()] = struct{}{}
	}
	log.Info("TestTracerLogging", "txHash", tx.Hash(), "from", from)
}

func (t *testTracer) OnBlockStart(ev tracing.BlockEvent) {
	blocksSeenByTracer++
	log.Info("TestTracerLogging OnBlockStart")
}

func (t *testTracer) OnBlockEnd(err error) {
	log.Info("TestTracerLogging OnBlockEnd")
}

// TestLiveTracingInNode: currently live tracing is only available when building a 2nd node
func TestLiveTracingInNode(t *testing.T) {
	txsSeenByTracer = make(map[common.Hash]struct{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).WithTakeOwnership(false)
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User")
	user := builder.L2Info.GetDefaultTransactOpts("User", ctx)

	// Create transactions to progress blockchain
	var err error
	var lastTx *types.Transaction
	numTxs := 20
	txs := make(map[common.Hash]struct{})
	for i := 0; i < numTxs; i++ {
		lastTx, _ = builder.L2.TransferBalanceTo(t, "Owner", util.RemapL1Address(user.From), big.NewInt(1e18), builder.L2Info)
		txs[lastTx.Hash()] = struct{}{}
	}

	// Start second node with live tracing
	execConfig := builder.execConfig
	execConfig.VmTrace = gethexec.LiveTracingConfig{TracerName: "testTracer"}
	tracers.LiveDirectory.Register("testTracer", newTestTracer)
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{execConfig: execConfig})
	defer cleanupB()

	// Wait for second node to catchup
	_, err = WaitForTx(ctx, testClientB.Client, lastTx.Hash(), time.Second*5)
	Require(t, err)
	if len(txsSeenByTracer) != numTxs {
		t.Fatalf("unexpected number of txs seen by testTracer. Want: %d, Got: %d", numTxs, len(txsSeenByTracer))
	}
	for txHash := range txs {
		if _, ok := txsSeenByTracer[txHash]; !ok {
			t.Fatalf("transaction: %s not seen by testTracer", txHash.String())
		}
	}

	totalBlocks, err := testClientB.Client.BlockNumber(ctx)
	Require(t, err)
	if blocksSeenByTracer != totalBlocks {
		t.Fatalf("unexpected number of txs seen by testTracer. Want: %d, Got: %d", totalBlocks, blocksSeenByTracer)
	}
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
		if balChange.Reason == tracing.BalanceDecreaseWithdrawToL1.Str() &&
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

type account struct {
	Balance *hexutil.Big                `json:"balance,omitempty"`
	Code    []byte                      `json:"code,omitempty"`
	Nonce   uint64                      `json:"nonce,omitempty"`
	Storage map[common.Hash]common.Hash `json:"storage,omitempty"`
}
type prestateTrace struct {
	Post map[common.Address]*account `json:"post"`
	Pre  map[common.Address]*account `json:"pre"`
}

func TestPrestateTracingSimple(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")
	sender := builder.L2Info.GetAddress("Owner")
	receiver := builder.L2Info.GetAddress("User2")
	ownerOldBalance, err := builder.L2.Client.BalanceAt(ctx, sender, nil)
	Require(t, err)

	value := big.NewInt(1e6)
	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, value, nil)
	Require(t, builder.L2.Client.SendTransaction(ctx, tx))
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	l2rpc := builder.L2.Stack.Attach()

	var result prestateTrace
	traceConfig := map[string]interface{}{
		"tracer": "prestateTracer",
		"tracerConfig": map[string]interface{}{
			"diffMode": true,
		},
	}
	err = l2rpc.CallContext(ctx, &result, "debug_traceTransaction", tx.Hash(), traceConfig)
	Require(t, err)

	if !arbmath.BigEquals(result.Pre[sender].Balance.ToInt(), ownerOldBalance) {
		Fatal(t, "Unexpected initial balance of sender")
	}
	if _, ok := result.Pre[receiver]; ok {
		Fatal(t, "receiver account found in the result of prestate tracer, its initial balance is zero so it shouldn't have been")
	}
	expBalance := arbmath.BigSub(ownerOldBalance, value)
	gas := arbmath.BigMulByUint(receipt.EffectiveGasPrice, receipt.GasUsed)
	expBalance = arbmath.BigSub(expBalance, gas)
	if !arbmath.BigEquals(result.Post[sender].Balance.ToInt(), expBalance) {
		Fatal(t, "Unexpected final balance of sender")
	}
	onchain, err := builder.L2.Client.BalanceAt(ctx, sender, receipt.BlockNumber)
	Require(t, err)
	if !arbmath.BigEquals(result.Post[sender].Balance.ToInt(), onchain) {
		Fatal(t, "Final balance of sender does not fit chain")
	}
	if !arbmath.BigEquals(result.Post[receiver].Balance.ToInt(), value) {
		Fatal(t, "Unexpected final balance of receiver")
	}
	if result.Post[sender].Nonce != result.Pre[sender].Nonce+1 {
		Fatal(t, "sender nonce increment wasn't registered")
	}
}

func TestArbTxTypesTracingPrestateTracerAndCallTracer(t *testing.T) {
	builder, delayedInbox, lookupL2Tx, ctx, teardown := retryableSetup(t)
	defer teardown()

	// Test prestate tracing of a ArbitrumDepositTx type tx
	faucetAddr := builder.L1Info.GetAddress("Faucet")
	oldBalance, err := builder.L2.Client.BalanceAt(ctx, faucetAddr, nil)
	Require(t, err)

	txOpts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	txOpts.Value = big.NewInt(13)

	l1tx, err := delayedInbox.DepositEth439370b1(&txOpts)
	Require(t, err)

	l1Receipt, err := builder.L1.EnsureTxSucceeded(l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		t.Errorf("Got transaction status: %v, want: %v", l1Receipt.Status, types.ReceiptStatusSuccessful)
	}
	waitForL1DelayBlocks(t, builder)

	l2Tx := lookupL2Tx(l1Receipt)
	l2Receipt, err := builder.L2.EnsureTxSucceeded(l2Tx)
	Require(t, err)
	newBalance, err := builder.L2.Client.BalanceAt(ctx, faucetAddr, l2Receipt.BlockNumber)
	Require(t, err)
	if got := new(big.Int); got.Sub(newBalance, oldBalance).Cmp(txOpts.Value) != 0 {
		t.Errorf("Got transferred: %v, want: %v", got, txOpts.Value)
	}

	l2rpc := builder.L2.Stack.Attach()
	var result prestateTrace
	traceConfig := map[string]interface{}{
		"tracer": "prestateTracer",
		"tracerConfig": map[string]interface{}{
			"diffMode": true,
		},
	}
	err = l2rpc.CallContext(ctx, &result, "debug_traceTransaction", l2Tx.Hash(), traceConfig)
	Require(t, err)

	if _, ok := result.Pre[faucetAddr]; !ok {
		Fatal(t, "Faucet account not found in the result of prestate tracer")
	}
	// Nonce shouldn't exist (in this case defaults to 0) in the Post map of the trace in DiffMode
	if l2Tx.SkipNonceChecks() && result.Post[faucetAddr].Nonce != 0 {
		Fatal(t, "Faucet account's nonce should remain unchanged ")
	}
	if !arbmath.BigEquals(result.Pre[faucetAddr].Balance.ToInt(), oldBalance) {
		Fatal(t, "Unexpected initial balance of Faucet")
	}
	if !arbmath.BigEquals(result.Post[faucetAddr].Balance.ToInt(), arbmath.BigAdd(oldBalance, txOpts.Value)) {
		Fatal(t, "Unexpected final balance of Faucet")
	}

	var blockTrace json.RawMessage
	blockTraceConfig := map[string]interface{}{"tracer": "callTracer"}

	blockTraceConfig["tracerConfig"] = map[string]interface{}{"onlyTopCall": false}
	err = l2rpc.CallContext(ctx, &blockTrace, "debug_traceBlockByNumber", rpc.BlockNumber(l2Receipt.BlockNumber.Int64()), blockTraceConfig)
	Require(t, err)

	blockTraceConfig["tracerConfig"] = map[string]interface{}{"onlyTopCall": true}
	err = l2rpc.CallContext(ctx, &blockTrace, "debug_traceBlockByNumber", rpc.BlockNumber(l2Receipt.BlockNumber.Int64()), blockTraceConfig)
	Require(t, err)

	// Test prestate tracing of a ArbitrumSubmitRetryableTx type tx
	user2Address := builder.L2Info.GetAddress("User2")
	beneficiaryAddress := builder.L2Info.GetAddress("Beneficiary")

	deposit := arbmath.BigMul(big.NewInt(1e12), big.NewInt(1e12))
	callValue := big.NewInt(1e6)

	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, builder.L2.Client)
	Require(t, err, "failed to deploy NodeInterface")

	// estimate the gas needed to auto redeem the retryable
	usertxoptsL2 := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)
	usertxoptsL2.NoSend = true
	usertxoptsL2.GasMargin = 0
	tx, err := nodeInterface.EstimateRetryableTicket(
		&usertxoptsL2,
		usertxoptsL2.From,
		deposit,
		user2Address,
		callValue,
		beneficiaryAddress,
		beneficiaryAddress,
		[]byte{0x32, 0x42, 0x32, 0x88}, // increase the cost to beyond that of params.TxGas
	)
	Require(t, err, "failed to estimate retryable submission")
	estimate := tx.Gas()
	expectedEstimate := params.TxGas + params.TxDataNonZeroGasEIP2028*4
	if float64(estimate) > float64(expectedEstimate)*(1+gasestimator.EstimateGasErrorRatio) {
		t.Errorf("estimated retryable ticket at %v gas but expected %v, with error margin of %v",
			estimate,
			expectedEstimate,
			gasestimator.EstimateGasErrorRatio,
		)
	}

	// submit & auto redeem the retryable using the gas estimate
	usertxoptsL1 := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	usertxoptsL1.Value = deposit
	l1tx, err = delayedInbox.CreateRetryableTicket(
		&usertxoptsL1,
		user2Address,
		callValue,
		big.NewInt(1e16),
		beneficiaryAddress,
		beneficiaryAddress,
		arbmath.UintToBig(estimate),
		big.NewInt(l2pricing.InitialBaseFeeWei*2),
		[]byte{0x32, 0x42, 0x32, 0x88},
	)
	Require(t, err)

	l1Receipt, err = builder.L1.EnsureTxSucceeded(l1tx)
	Require(t, err)
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t, "l1Receipt indicated failure")
	}

	waitForL1DelayBlocks(t, builder)

	l2Tx = lookupL2Tx(l1Receipt)
	receipt, err := builder.L2.EnsureTxSucceeded(l2Tx)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		Fatal(t)
	}

	l2balance, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), nil)
	Require(t, err)
	if !arbmath.BigEquals(l2balance, callValue) {
		Fatal(t, "Unexpected balance:", l2balance)
	}

	ticketId := receipt.Logs[0].Topics[1]
	firstRetryTxId := receipt.Logs[1].Topics[2]
	fmt.Println("submitretryable txid ", ticketId)
	fmt.Println("auto redeem txid ", firstRetryTxId)

	// Trace ArbitrumSubmitRetryableTx
	result = prestateTrace{}
	err = l2rpc.CallContext(ctx, &result, "debug_traceTransaction", l2Tx.Hash(), traceConfig)
	Require(t, err)

	escrowAddr := retryables.RetryableEscrowAddress(ticketId)
	if _, ok := result.Pre[escrowAddr]; ok {
		Fatal(t, "Escrow account found in the result of prestate tracer for a ArbitrumSubmitRetryableTx transaction, its initial balance is zero so it shouldn't have been")
	}

	if !arbmath.BigEquals(result.Post[escrowAddr].Balance.ToInt(), callValue) {
		Fatal(t, "Unexpected final balance of Escrow")
	}

	blockTraceConfig["tracerConfig"] = map[string]interface{}{"onlyTopCall": false}
	err = l2rpc.CallContext(ctx, &blockTrace, "debug_traceBlockByNumber", rpc.BlockNumber(receipt.BlockNumber.Int64()), blockTraceConfig)
	Require(t, err)
	fmt.Println(string(blockTrace))

	blockTraceConfig["tracerConfig"] = map[string]interface{}{"onlyTopCall": true}
	err = l2rpc.CallContext(ctx, &blockTrace, "debug_traceBlockByNumber", rpc.BlockNumber(receipt.BlockNumber.Int64()), blockTraceConfig)
	Require(t, err)
	fmt.Println(string(blockTrace))

	// Trace ArbitrumRetryTx
	result = prestateTrace{}
	err = l2rpc.CallContext(ctx, &result, "debug_traceTransaction", firstRetryTxId, traceConfig)
	Require(t, err)

	if _, ok := result.Pre[user2Address]; ok {
		Fatal(t, "User2 account found in the result of prestate tracer, its initial balance is zero so it shouldn't have been")
	}
	if !arbmath.BigEquals(result.Post[user2Address].Balance.ToInt(), callValue) {
		Fatal(t, "Unexpected final balance of User2")
	}
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
