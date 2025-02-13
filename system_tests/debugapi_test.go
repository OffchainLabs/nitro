package arbtest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/gasestimator"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/google/go-cmp/cmp"

	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
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

type account struct {
	Balance *hexutil.Big                `json:"balance,omitempty"`
	Code    []byte                      `json:"code,omitempty"`
	Nonce   *uint64                     `json:"nonce,omitempty"`
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
	user2OldBalance, err := builder.L2.Client.BalanceAt(ctx, receiver, nil)
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
	if !arbmath.BigEquals(result.Pre[receiver].Balance.ToInt(), user2OldBalance) {
		Fatal(t, "Unexpected initial balance of receiver")
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
	if *result.Post[sender].Nonce != *result.Pre[sender].Nonce+1 {
		Fatal(t, "sender nonce increment wasn't registered")
	}
	if *result.Post[receiver].Nonce != *result.Pre[receiver].Nonce {
		Fatal(t, "receiver nonce shouldn't change")
	}
}

func TestPrestateTracingComplex(t *testing.T) {
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
	// Nonce shouldn't exist in the Post map of the trace in DiffMode
	if l2Tx.SkipAccountChecks() && result.Post[faucetAddr].Nonce != nil {
		Fatal(t, "Faucet account's nonce should remain unchanged ")
	}
	if !arbmath.BigEquals(result.Pre[faucetAddr].Balance.ToInt(), oldBalance) {
		Fatal(t, "Unexpected initial balance of Faucet")
	}
	if !arbmath.BigEquals(result.Post[faucetAddr].Balance.ToInt(), arbmath.BigAdd(oldBalance, txOpts.Value)) {
		Fatal(t, "Unexpected final balance of Faucet")
	}

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
	if _, ok := result.Pre[escrowAddr]; !ok {
		Fatal(t, "Escrow account not found in the result of prestate tracer for a ArbitrumSubmitRetryableTx transaction")
	}

	if !arbmath.BigEquals(result.Pre[escrowAddr].Balance.ToInt(), common.Big0) {
		Fatal(t, "Unexpected initial balance of Escrow")
	}
	if !arbmath.BigEquals(result.Post[escrowAddr].Balance.ToInt(), callValue) {
		Fatal(t, "Unexpected final balance of Escrow")
	}

	// Trace ArbitrumRetryTx
	result = prestateTrace{}
	err = l2rpc.CallContext(ctx, &result, "debug_traceTransaction", firstRetryTxId, traceConfig)
	Require(t, err)

	if !arbmath.BigEquals(result.Pre[user2Address].Balance.ToInt(), common.Big0) {
		Fatal(t, "Unexpected initial balance of User2")
	}
	if !arbmath.BigEquals(result.Post[user2Address].Balance.ToInt(), callValue) {
		Fatal(t, "Unexpected final balance of User2")
	}

	AutomatedPrestateTracerTest(t, builder.L2)
}

type accountDump struct {
	Balance       *big.Int
	Nonce         uint64
	Code          []byte
	HashedStorage map[common.Hash]common.Hash
}

type stateDump struct {
	HashedAccounts map[common.Hash]*accountDump
}

// This uses the trie iterator to dump the state at a given block number.
// Ideally we'd use debug_dumpBlock, but it has a limit of 256 accounts,
// and we don't support configuring preimage storage which is necessary for it.
func dumpState(t *testing.T, client *TestClient, blockNumber uint64) *stateDump {
	bc := client.ExecNode.Backend.BlockChain()
	block := bc.GetBlockByNumber(blockNumber)
	sdb, err := bc.StateAt(block.Root())
	Require(t, err)
	trieIt, err := sdb.GetTrie().NodeIterator(nil)
	Require(t, err)
	it := trie.NewIterator(trieIt)
	dump := &stateDump{
		HashedAccounts: make(map[common.Hash]*accountDump),
	}
	for it.Next() {
		var data types.StateAccount
		err = rlp.DecodeBytes(it.Value, &data)
		Require(t, err)
		account := &accountDump{
			Balance:       data.Balance.ToBig(),
			Nonce:         data.Nonce,
			HashedStorage: make(map[common.Hash]common.Hash),
		}
		addrHash := common.BytesToHash(it.Key)
		dump.HashedAccounts[addrHash] = account
		if len(data.CodeHash) > 0 {
			codeHash := common.BytesToHash(data.CodeHash)
			if codeHash != types.EmptyCodeHash {
				account.Code, err = sdb.Database().ContractCode(common.Address{}, codeHash)
				Require(t, err)
			}
		}
		if data.Root != types.EmptyRootHash {
			storageTrie, err := trie.NewStateTrie(trie.StorageTrieID(block.Root(), addrHash, data.Root), sdb.Database().TrieDB())
			Require(t, err)
			storageIt, err := storageTrie.NodeIterator(nil)
			Require(t, err)
			storageIterator := trie.NewIterator(storageIt)
			for storageIterator.Next() {
				key := common.BytesToHash(storageIterator.Key)
				_, value, _, err := rlp.Split(storageIterator.Value)
				Require(t, err)
				account.HashedStorage[key] = common.BytesToHash(value)
			}
		}
	}
	return dump
}

func AutomatedPrestateTracerTest(t *testing.T, client *TestClient) {
	blockHeight, err := client.Client.BlockNumber(client.ctx)
	Require(t, err)
	runningState := dumpState(t, client, 1)
	for block := uint64(2); block <= blockHeight; block++ {
		var trace []prestateTrace
		traceConfig := map[string]interface{}{
			"tracer": "prestateTracer",
			"tracerConfig": map[string]interface{}{
				"diffMode": true,
			},
		}
		err = client.Client.Client().CallContext(client.ctx, &trace, "debug_traceBlockByNumber", hexutil.Uint64(block), traceConfig)
		Require(t, err)
		for _, trace := range trace {
			for addr, contents := range trace.Pre {
				hashedAddr := crypto.Keccak256Hash(addr.Bytes())
				runningAccount := runningState.HashedAccounts[hashedAddr]
				if runningAccount == nil {
					Fatal(t, "Account ", addr, " not found in previous state for prestate tracer test")
				}
				if contents.Balance == nil {
					Fatal(t, "Balance of account ", addr, " was nil in prestate tracer")
				}
				if !arbmath.BigEquals(contents.Balance.ToInt(), runningAccount.Balance) {
					Fatal(t, "Balance of account ", addr, " was ", runningAccount.Balance, " but tracer shows ", contents.Balance.ToInt())
				}
				if contents.Nonce == nil {
					Fatal(t, "Nonce of account ", addr, " was nil in prestate tracer")
				}
				if *contents.Nonce != runningAccount.Nonce {
					Fatal(t, "Nonce of account ", addr, " was ", runningAccount.Nonce, " but tracer shows ", contents.Nonce)
				}
				if (len(contents.Code) != 0) != (len(runningAccount.Code) != 0) {
					Fatal(t, "Code presence of account ", addr, " was ", len(runningAccount.Code) != 0, " but tracer shows ", len(contents.Code) != 0)
				}
				if !bytes.Equal(contents.Code, runningAccount.Code) {
					Fatal(t, "Code of account ", addr, " was incorrect in prestate tracer")
				}
				accountPostTrace, accountInPost := trace.Post[addr]
				for key, val := range contents.Storage {
					hashedKey := crypto.Keccak256Hash(key.Bytes())
					previousVal := runningAccount.HashedStorage[hashedKey]
					if val != previousVal {
						Fatal(t, "Account ", addr, " storage key ", key, " was ", previousVal, " but tracer shows ", val)
					}
					if accountInPost {
						_, storageInPost := accountPostTrace.Storage[key]
						if !storageInPost {
							// This slot was deleted
							delete(runningAccount.HashedStorage, hashedKey)
						}
					}
				}
				if !accountInPost {
					// This account was deleted
					delete(runningState.HashedAccounts, hashedAddr)
				}
			}
			for addr, contents := range trace.Post {
				hashedAddr := crypto.Keccak256Hash(addr.Bytes())
				runningAccount, hadAccount := runningState.HashedAccounts[hashedAddr]
				preTrace, inPre := trace.Pre[addr]
				if !hadAccount {
					runningAccount = &accountDump{
						HashedStorage: make(map[common.Hash]common.Hash),
					}
					runningState.HashedAccounts[hashedAddr] = runningAccount
				} else if !inPre {
					Fatal(t, "Account ", addr, " was not in tracer prestate but was in state")
				}
				if contents.Balance != nil {
					runningAccount.Balance = contents.Balance.ToInt()
				}
				if contents.Nonce != nil {
					runningAccount.Nonce = *contents.Nonce
				}
				if len(contents.Code) != 0 {
					runningAccount.Code = contents.Code
				}
				for key, val := range contents.Storage {
					hashedKey := crypto.Keccak256Hash(key.Bytes())
					if inPre {
						_, hadPreStorage := preTrace.Storage[key]
						if !hadPreStorage && runningAccount.HashedStorage[hashedKey] != (common.Hash{}) {
							Fatal(t, "Account ", addr, " storage key ", key, " was in state but not in prestate tracer")
						}
					}
					runningAccount.HashedStorage[hashedKey] = val
				}
			}
		}
		expectedState := dumpState(t, client, block)
		diff := cmp.Diff(expectedState, runningState, cmp.Comparer(func(x, y *big.Int) bool {
			return x.Cmp(y) == 0
		}))
		if diff != "" {
			Fatal(t, "State mismatch at block ", block, ":\n", diff)
		}
	}
}
