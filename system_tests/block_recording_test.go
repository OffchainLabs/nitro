// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/timeboost/bindings"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// recordBlockSetup creates the common builder, auth, and cleanup used by every
// block-recording test. It mirrors setupProgramTest but is kept separate so the
// recording tests can be run independently.
func recordBlockSetup(t *testing.T) (*NodeBuilder, *bind.TransactOpts, func()) {
	t.Helper()
	builder, auth, cleanup := setupProgramTest(t, true)
	return builder, &auth, cleanup
}

// ensureTx is a small wrapper that sends a transaction and waits for success.
func ensureTx(t *testing.T, builder *NodeBuilder, tx *types.Transaction, err error) *types.Receipt {
	t.Helper()
	Require(t, err)
	receipt, err := EnsureTxSucceeded(builder.ctx, builder.L2.Client, tx)
	Require(t, err)
	return receipt
}

// record is a convenience wrapper around recordBlock.
func record(t *testing.T, blockNum uint64, builder *NodeBuilder) {
	t.Helper()
	recordBlock(t, blockNum, builder, rawdb.TargetWavm, rawdb.LocalTarget())
}

// sequenceInBlock forces a batch of transactions into a single block by calling
// the execution engine directly, bypassing the sequencer's createBlock loop
// (which typically puts each tx in its own block). Returns the block number and
// the receipts for all transactions.
func sequenceInBlock(t *testing.T, builder *NodeBuilder, txs types.Transactions) (uint64, []*types.Receipt) {
	t.Helper()
	ctx := builder.ctx
	l2client := builder.L2.Client

	lastBlock, err := l2client.BlockByNumber(ctx, nil)
	Require(t, err)
	header := &arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_L2Message,
		Poster:      l1pricing.BatchPosterAddress,
		BlockNumber: lastBlock.NumberU64() + 1,
		Timestamp:   arbmath.SaturatingUCast[uint64](time.Now().Unix()),
	}
	hooks := gethexec.MakeZeroTxSizeSequencingHooksForTesting(txs, nil, nil, nil)
	_, err = builder.L2.ExecNode.ExecEngine.SequenceTransactions(header, hooks, nil)
	Require(t, err)

	var blockNum uint64
	receipts := make([]*types.Receipt, len(txs))
	for i, tx := range txs {
		receipts[i], err = EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		if i == 0 {
			blockNum = receipts[0].BlockNumber.Uint64()
		} else if receipts[i].BlockNumber.Uint64() != blockNum {
			t.Fatalf("tx %d in block %d, expected block %d", i, receipts[i].BlockNumber.Uint64(), blockNum)
		}
	}
	return blockNum, receipts
}

// syncOwnerNonce updates the l2info internal nonce counter for "Owner" to match
// the on-chain pending nonce. This is needed after transactions sent via auth
// (TransactOpts) which bypass l2info's counter (e.g. DeploySimple, deployWasm).
func syncOwnerNonce(t *testing.T, builder *NodeBuilder) {
	t.Helper()
	nonce, err := builder.L2.Client.PendingNonceAt(builder.ctx, builder.L2Info.GetAddress("Owner"))
	Require(t, err)
	builder.L2Info.GetInfoWithPrivKey("Owner").Nonce.Store(nonce)
}

// noSendOpts returns a copy of auth with NoSend=true, an explicit nonce, and a
// fixed gas limit. Setting GasLimit skips eth_estimateGas, which is necessary
// when batched txs depend on each other (e.g. transfers before approvals).
func noSendOpts(auth *bind.TransactOpts, nonce uint64) *bind.TransactOpts {
	opts := *auth
	opts.NoSend = true
	opts.Nonce = new(big.Int).SetUint64(nonce)
	opts.GasLimit = 32000000
	return &opts
}

// ---------------------------------------------------------------------------
// 1. Single simple ETH transfer (tiny block)
// ---------------------------------------------------------------------------

func TestRecordBlockSingleTransfer(t *testing.T) {
	builder, _, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	l2info.GenerateAccount("User1")
	tx := l2info.PrepareTx("Owner", "User1", l2info.TransferGas, big.NewInt(1e16), nil)
	receipt := ensureTx(t, builder, tx, l2client.SendTransaction(ctx, tx))

	record(t, receipt.BlockNumber.Uint64(), builder)
}

// ---------------------------------------------------------------------------
// 2. Several simple ETH transfers (small block, multiple txs)
// ---------------------------------------------------------------------------

func TestRecordBlockMultipleTransfers(t *testing.T) {
	builder, _, cleanup := recordBlockSetup(t)
	l2info := builder.L2Info
	defer cleanup()

	const numTxs = 5
	for i := 0; i < numTxs; i++ {
		l2info.GenerateAccount(fmt.Sprintf("User%d", i))
	}

	txs := make(types.Transactions, numTxs)
	for i := 0; i < numTxs; i++ {
		txs[i] = l2info.PrepareTx("Owner", fmt.Sprintf("User%d", i), l2info.TransferGas, big.NewInt(1e16), nil)
	}

	blockNum, _ := sequenceInBlock(t, builder, txs)
	record(t, blockNum, builder)
}

// ---------------------------------------------------------------------------
// 3. Many simple ETH transfers (medium block, batch)
// ---------------------------------------------------------------------------

func TestRecordBlockManyTransfers(t *testing.T) {
	builder, _, cleanup := recordBlockSetup(t)
	l2info := builder.L2Info
	defer cleanup()

	const numTxs = 50
	for i := 0; i < numTxs; i++ {
		l2info.GenerateAccount(fmt.Sprintf("BatchUser%d", i))
	}

	txs := make(types.Transactions, numTxs)
	for i := 0; i < numTxs; i++ {
		txs[i] = l2info.PrepareTx("Owner", fmt.Sprintf("BatchUser%d", i), l2info.TransferGas, big.NewInt(1e12), nil)
	}

	blockNum, _ := sequenceInBlock(t, builder, txs)
	record(t, blockNum, builder)
}

// ---------------------------------------------------------------------------
// 4. Deploy a Solidity contract (Simple) – contract deployment block
// ---------------------------------------------------------------------------

func TestRecordBlockSolidityDeploy(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	_, tx, _, err := localgen.DeploySimple(auth, l2client)
	receipt := ensureTx(t, builder, tx, err)

	record(t, receipt.BlockNumber.Uint64(), builder)

	// Also call a method to generate a second recorded block with a contract call.
	simple, err := localgen.NewSimple(receipt.ContractAddress, l2client)
	Require(t, err)
	tx, err = simple.Increment(auth)
	receipt = ensureTx(t, builder, tx, err)

	record(t, receipt.BlockNumber.Uint64(), builder)

	_ = ctx
}

// ---------------------------------------------------------------------------
// 5. Deploy ERC20 contract – larger contract deployment
// ---------------------------------------------------------------------------

func TestRecordBlockERC20Deploy(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	l2client := builder.L2.Client
	defer cleanup()

	_, tx, _, err := localgen.DeployERC20(auth, l2client, "TestToken", "TT")
	receipt := ensureTx(t, builder, tx, err)

	record(t, receipt.BlockNumber.Uint64(), builder)
}

// ---------------------------------------------------------------------------
// 6. ERC20 transfers – multiple contract calls
// ---------------------------------------------------------------------------

func TestRecordBlockERC20Transfers(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	// Use MockERC20 which has a Mint function
	_, tx, erc20, err := bindings.DeployMockERC20(auth, l2client)
	ensureTx(t, builder, tx, err)

	// Mint tokens to Owner so transfers succeed
	tx, err = erc20.Mint(auth, auth.From, big.NewInt(1e18))
	ensureTx(t, builder, tx, err)

	const numRecipients = 10
	recipients := make([]common.Address, numRecipients)
	for i := 0; i < numRecipients; i++ {
		name := fmt.Sprintf("ERC20User%d", i)
		l2info.GenerateAccount(name)
		recipients[i] = l2info.GetAddress(name)
	}

	// Build all ERC20 transfer txs without sending (explicit nonces)
	nonce, err := l2client.PendingNonceAt(builder.ctx, auth.From)
	Require(t, err)
	txs := make(types.Transactions, numRecipients)
	for i := 0; i < numRecipients; i++ {
		txs[i], err = erc20.Transfer(noSendOpts(auth, nonce+uint64(i)), recipients[i], big.NewInt(1000))
		Require(t, err)
	}

	blockNum, _ := sequenceInBlock(t, builder, txs)
	record(t, blockNum, builder)
}

// ---------------------------------------------------------------------------
// 7. Deploy Stylus WASM program (storage.wasm) – WASM deployment block
// ---------------------------------------------------------------------------

func TestRecordBlockWasmDeploy(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	// deployWasm both deploys and activates – we record the deployment block.
	wasm, _ := readWasmFile(t, rustFile("storage"))
	auth.GasLimit = 32000000
	program := deployContract(t, ctx, *auth, l2client, wasm)

	// Get the block for the deployment
	nonce, err := l2client.NonceAt(ctx, auth.From, nil)
	Require(t, err)
	_ = nonce
	// The deployment just happened; get the latest block
	latestBlock, err := l2client.BlockByNumber(ctx, nil)
	Require(t, err)

	record(t, latestBlock.NumberU64(), builder)

	_ = program
}

// ---------------------------------------------------------------------------
// 8. Single Stylus storage write – small WASM execution
// ---------------------------------------------------------------------------

func TestRecordBlockWasmStorageWrite(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	programAddress := deployWasm(t, ctx, *auth, l2client, rustFile("storage"))

	key := testhelpers.RandomHash()
	value := testhelpers.RandomHash()
	tx := l2info.PrepareTxTo("Owner", &programAddress, l2info.TransferGas, nil, argsForStorageWrite(key, value))
	receipt := ensureTx(t, builder, tx, l2client.SendTransaction(ctx, tx))

	record(t, receipt.BlockNumber.Uint64(), builder)
}

// ---------------------------------------------------------------------------
// 9. Multiple Stylus storage writes – medium WASM execution
// ---------------------------------------------------------------------------

func TestRecordBlockWasmMultipleStorageWrites(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	programAddress := deployWasm(t, ctx, *auth, l2client, rustFile("storage"))

	// Sync l2info nonce (deployWasm via auth bypasses l2info counter)
	syncOwnerNonce(t, builder)

	const numWrites = 20
	txs := make(types.Transactions, numWrites)
	for i := 0; i < numWrites; i++ {
		key := testhelpers.RandomHash()
		value := testhelpers.RandomHash()
		txs[i] = l2info.PrepareTxTo("Owner", &programAddress, l2info.TransferGas, nil, argsForStorageWrite(key, value))
	}

	blockNum, _ := sequenceInBlock(t, builder, txs)
	record(t, blockNum, builder)

	_ = l2client
}

// ---------------------------------------------------------------------------
// 10. Stylus multicall with nested storage ops – complex WASM execution
// ---------------------------------------------------------------------------

func TestRecordBlockWasmMulticallStorageOps(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	storageAddr := deployWasm(t, ctx, *auth, l2client, rustFile("storage"))
	multicallAddr := deployWasm(t, ctx, *auth, l2client, rustFile("multicall"))

	// Build a multicall that does many storage writes
	args := multicallEmptyArgs()
	for i := 0; i < 16; i++ {
		key := testhelpers.RandomHash()
		value := testhelpers.RandomHash()
		args = multicallAppend(args, vm.CALL, storageAddr, argsForStorageWrite(key, value))
	}

	tx := l2info.PrepareTxTo("Owner", &multicallAddr, 1e9, nil, args)
	receipt := ensureTx(t, builder, tx, l2client.SendTransaction(ctx, tx))

	record(t, receipt.BlockNumber.Uint64(), builder)
}

// ---------------------------------------------------------------------------
// 11. Stylus keccak computation – compute-heavy WASM block
// ---------------------------------------------------------------------------

func TestRecordBlockWasmKeccak(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	keccakAddr := deployWasm(t, ctx, *auth, l2client, rustFile("keccak"))

	_, tx, mock, err := localgen.DeployProgramTest(auth, l2client)
	ensureTx(t, builder, tx, err)

	// Keccak a preimage – uses compute
	preimage := []byte("benchmark preimage data for keccak hashing test case in JIT block recording")
	keccakArgs := []byte{0x01}
	keccakArgs = append(keccakArgs, preimage...)

	tx, err = mock.CallKeccak(auth, keccakAddr, keccakArgs)
	receipt := ensureTx(t, builder, tx, err)

	record(t, receipt.BlockNumber.Uint64(), builder)
}

// ---------------------------------------------------------------------------
// 12. Stylus math operations – compute-heavy WASM block
// ---------------------------------------------------------------------------

func TestRecordBlockWasmMath(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	mathAddr := deployWasm(t, ctx, *auth, l2client, rustFile("math"))

	_, tx, mock, err := localgen.DeployProgramTest(auth, l2client)
	ensureTx(t, builder, tx, err)

	tx, err = mock.MathTest(auth, mathAddr)
	receipt := ensureTx(t, builder, tx, err)

	record(t, receipt.BlockNumber.Uint64(), builder)
}

// ---------------------------------------------------------------------------
// 13. Stylus CREATE opcode – WASM creating a contract
// ---------------------------------------------------------------------------

func TestRecordBlockWasmCreate(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	createAddr := deployWasm(t, ctx, *auth, l2client, rustFile("create"))

	deployWasmBytes, _ := readWasmFile(t, rustFile("storage"))
	deployCode := deployContractInitCode(deployWasmBytes, false)
	startValue := testhelpers.RandomCallValue(1e12)

	create1Args := []byte{0x01}
	create1Args = append(create1Args, common.BigToHash(startValue).Bytes()...)
	create1Args = append(create1Args, deployCode...)

	tx := l2info.PrepareTxTo("Owner", &createAddr, 1e9, startValue, create1Args)
	receipt := ensureTx(t, builder, tx, l2client.SendTransaction(ctx, tx))

	record(t, receipt.BlockNumber.Uint64(), builder)
}

// ---------------------------------------------------------------------------
// 14. Stylus log emission – WASM with event logs
// ---------------------------------------------------------------------------

func TestRecordBlockWasmLogs(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	logAddr := deployWasm(t, ctx, *auth, l2client, rustFile("log"))

	// Emit a log with 4 topics and some data
	topics := make([]common.Hash, 4)
	for i := range topics {
		topics[i] = testhelpers.RandomHash()
	}
	data := testhelpers.RandomSlice(128)

	logArgs := []byte{byte(len(topics))}
	for _, topic := range topics {
		logArgs = append(logArgs, topic[:]...)
	}
	logArgs = append(logArgs, data...)

	tx := l2info.PrepareTxTo("Owner", &logAddr, 1e9, nil, logArgs)
	receipt := ensureTx(t, builder, tx, l2client.SendTransaction(ctx, tx))

	record(t, receipt.BlockNumber.Uint64(), builder)
}

// ---------------------------------------------------------------------------
// 15. Solidity contract with repeated state changes – many SSTORE ops
// ---------------------------------------------------------------------------

func TestRecordBlockSolidityRepeatedIncrements(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	l2client := builder.L2.Client
	defer cleanup()

	_, tx, simple, err := localgen.DeploySimple(auth, l2client)
	ensureTx(t, builder, tx, err)

	// Build 20 Increment() txs without sending (explicit nonces)
	nonce, err := l2client.PendingNonceAt(builder.ctx, auth.From)
	Require(t, err)
	const numIncrements = 20
	txs := make(types.Transactions, numIncrements)
	for i := 0; i < numIncrements; i++ {
		txs[i], err = simple.Increment(noSendOpts(auth, nonce+uint64(i)))
		Require(t, err)
	}

	blockNum, _ := sequenceInBlock(t, builder, txs)
	record(t, blockNum, builder)
}

// ---------------------------------------------------------------------------
// 16. Mixed: ETH transfer + Solidity deploy + Solidity call in sequence
// ---------------------------------------------------------------------------

func TestRecordBlockMixedEthAndSolidity(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	// Setup: deploy Simple contract (must happen before we can call it)
	_, tx, simple, err := localgen.DeploySimple(auth, l2client)
	ensureTx(t, builder, tx, err)

	// Sync l2info nonce with on-chain nonce (deploy via auth bypasses l2info counter)
	syncOwnerNonce(t, builder)
	l2info.GenerateAccount("MixedUser")

	// Build batch: 1 transfer (PrepareTx auto-increments nonce) + 2 Solidity calls
	txTransfer := l2info.PrepareTx("Owner", "MixedUser", l2info.TransferGas, big.NewInt(1e18), nil)
	// After PrepareTx, l2info nonce is already incremented; use it for NoSend calls
	nonce := l2info.GetInfoWithPrivKey("Owner").Nonce.Load()
	txIncrement, err := simple.Increment(noSendOpts(auth, nonce))
	Require(t, err)
	txIncrementEmit, err := simple.IncrementEmit(noSendOpts(auth, nonce+1))
	Require(t, err)

	txs := types.Transactions{txTransfer, txIncrement, txIncrementEmit}
	blockNum, _ := sequenceInBlock(t, builder, txs)
	record(t, blockNum, builder)
}

// ---------------------------------------------------------------------------
// 17. Mixed: Solidity + Stylus WASM in the same test
// ---------------------------------------------------------------------------

func TestRecordBlockMixedSolidityAndWasm(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	// Setup: deploy both contracts
	_, tx, simple, err := localgen.DeploySimple(auth, l2client)
	ensureTx(t, builder, tx, err)
	storageAddr := deployWasm(t, ctx, *auth, l2client, rustFile("storage"))

	// Sync l2info nonce (deploys via auth/deployWasm bypass l2info counter)
	syncOwnerNonce(t, builder)

	// Build batch: 1 Solidity call + 1 WASM storage write
	nonce := l2info.GetInfoWithPrivKey("Owner").Nonce.Load()
	txIncrement, err := simple.Increment(noSendOpts(auth, nonce))
	Require(t, err)

	// PrepareTxTo uses l2info nonce counter; sync it past the NoSend tx
	l2info.GetInfoWithPrivKey("Owner").Nonce.Store(nonce + 1)
	key := testhelpers.RandomHash()
	value := testhelpers.RandomHash()
	txStorage := l2info.PrepareTxTo("Owner", &storageAddr, l2info.TransferGas, nil, argsForStorageWrite(key, value))

	txs := types.Transactions{txIncrement, txStorage}
	blockNum, _ := sequenceInBlock(t, builder, txs)
	record(t, blockNum, builder)
}

// ---------------------------------------------------------------------------
// 18. Multiple Solidity contract deployments in sequence
// ---------------------------------------------------------------------------

func TestRecordBlockMultipleSolidityDeploys(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	l2client := builder.L2.Client
	defer cleanup()

	// Deploy several different contracts
	_, tx, _, err := localgen.DeploySimple(auth, l2client)
	ensureTx(t, builder, tx, err)

	_, tx, _, err = localgen.DeployERC20(auth, l2client, "Token1", "T1")
	ensureTx(t, builder, tx, err)

	_, tx, _, err = localgen.DeployProgramTest(auth, l2client)
	ensureTx(t, builder, tx, err)

	_, tx, _, err = localgen.DeployMultiCallTest(auth, l2client)
	receipt := ensureTx(t, builder, tx, err)

	record(t, receipt.BlockNumber.Uint64(), builder)
}

// ---------------------------------------------------------------------------
// 19. Deeply nested WASM multicall – stress nested calls
// ---------------------------------------------------------------------------

func TestRecordBlockWasmDeepMulticall(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	storageAddr := deployWasm(t, ctx, *auth, l2client, rustFile("storage"))
	multicallAddr := deployWasm(t, ctx, *auth, l2client, rustFile("multicall"))

	// Build nested multicall: multicall -> multicall -> ... -> storage write
	key := testhelpers.RandomHash()
	value := testhelpers.RandomHash()
	inner := argsForStorageWrite(key, value)

	// Nest 8 levels deep: each level wraps the previous in a CALL to multicall
	args := argsForMulticall(vm.CALL, storageAddr, nil, inner)
	for i := 0; i < 7; i++ {
		args = argsForMulticall(vm.CALL, multicallAddr, nil, args)
	}

	tx := l2info.PrepareTxTo("Owner", &multicallAddr, 1e9, nil, args)
	receipt := ensureTx(t, builder, tx, l2client.SendTransaction(ctx, tx))

	record(t, receipt.BlockNumber.Uint64(), builder)
}

// ---------------------------------------------------------------------------
// 20. Large multicall with storage writes + reads + logs
// ---------------------------------------------------------------------------

func TestRecordBlockWasmLargeMulticall(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	storageAddr := deployWasm(t, ctx, *auth, l2client, rustFile("storage"))
	multicallAddr := deployWasm(t, ctx, *auth, l2client, rustFile("multicall"))

	// Build a large multicall: write, read, write, read with logging
	args := multicallEmptyArgs()
	for i := 0; i < 32; i++ {
		key := testhelpers.RandomHash()
		value := testhelpers.RandomHash()
		// Alternate between storage write via CALL and storage read via CALL
		args = multicallAppend(args, vm.CALL, storageAddr, argsForStorageWrite(key, value))
		args = multicallAppend(args, vm.CALL, storageAddr, argsForStorageRead(key))
	}

	tx := l2info.PrepareTxTo("Owner", &multicallAddr, 1e9, nil, args)
	receipt := ensureTx(t, builder, tx, l2client.SendTransaction(ctx, tx))

	record(t, receipt.BlockNumber.Uint64(), builder)
}

// ---------------------------------------------------------------------------
// 21. Large raw bytecode contract deployment – big initcode
// ---------------------------------------------------------------------------

func TestRecordBlockLargeContractDeploy(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	// Create a large contract bytecode (24KB of bytecode, near the limit)
	// Fill with STOP opcodes (0x00)
	largeCode := make([]byte, 24000)
	// Put a simple PUSH1 0 PUSH1 0 RETURN at the start so it's valid
	largeCode[0] = byte(vm.PUSH1)
	largeCode[1] = 0
	largeCode[2] = byte(vm.PUSH1)
	largeCode[3] = 0
	largeCode[4] = byte(vm.RETURN)
	// Rest is already 0x00 (STOP opcodes)

	auth.GasLimit = 32000000
	addr := deployContract(t, ctx, *auth, l2client, largeCode)
	_ = addr

	latestBlock, err := l2client.BlockByNumber(ctx, nil)
	Require(t, err)

	record(t, latestBlock.NumberU64(), builder)
}

// ---------------------------------------------------------------------------
// 22. Transfers with calldata – transfers carrying payload
// ---------------------------------------------------------------------------

func TestRecordBlockTransfersWithCalldata(t *testing.T) {
	builder, _, cleanup := recordBlockSetup(t)
	l2info := builder.L2Info
	defer cleanup()

	l2info.GenerateAccount("DataUser")
	addr := l2info.GetAddress("DataUser")

	// Build transfers with increasing calldata sizes
	sizes := []int{32, 256, 1024, 4096}
	txs := make(types.Transactions, len(sizes))
	for i, size := range sizes {
		data := testhelpers.RandomSlice(uint64(size))
		txs[i] = l2info.PrepareTxTo("Owner", &addr, 1e9, big.NewInt(1), data)
	}

	blockNum, _ := sequenceInBlock(t, builder, txs)
	record(t, blockNum, builder)
}

// ---------------------------------------------------------------------------
// 23. Multiple WASM program deployments – several Stylus deploys
// ---------------------------------------------------------------------------

func TestRecordBlockMultipleWasmDeploys(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	programs := []string{"storage", "keccak", "multicall", "math"}
	for _, name := range programs {
		deployWasm(t, ctx, *auth, l2client, rustFile(name))
	}

	latestBlock, err := l2client.BlockByNumber(ctx, nil)
	Require(t, err)

	record(t, latestBlock.NumberU64(), builder)
}

// ---------------------------------------------------------------------------
// 24. Precompile calls – ArbSys, ArbInfo interactions
// ---------------------------------------------------------------------------

func TestRecordBlockPrecompileCalls(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	arbSys, err := precompilesgen.NewArbSys(types.ArbSysAddress, l2client)
	Require(t, err)

	// Call ArbSys.ArbBlockNumber to exercise precompiles
	blockNum, err := arbSys.ArbBlockNumber(nil)
	Require(t, err)
	_ = blockNum

	// Deploy Simple contract and call CheckBlockHashes which exercises BLOCKHASH
	_, tx, simple, err := localgen.DeploySimple(auth, l2client)
	ensureTx(t, builder, tx, err)

	tx, err = simple.StoreDifficulty(auth)
	receipt := ensureTx(t, builder, tx, err)

	record(t, receipt.BlockNumber.Uint64(), builder)

	_ = ctx
}

// ---------------------------------------------------------------------------
// 25. ERC20 deploy + mint + multiple transfers + approvals (complex contract state)
// ---------------------------------------------------------------------------

func TestRecordBlockERC20FullWorkflow(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	// Setup: deploy MockERC20 (has Mint), mint tokens, and fund users with ETH
	_, tx, erc20, err := bindings.DeployMockERC20(auth, l2client)
	ensureTx(t, builder, tx, err)

	tx, err = erc20.Mint(auth, auth.From, big.NewInt(1e18))
	ensureTx(t, builder, tx, err)

	const numUsers = 5
	users := make([]common.Address, numUsers)
	for i := 0; i < numUsers; i++ {
		name := fmt.Sprintf("ERC20WF%d", i)
		l2info.GenerateAccount(name)
		users[i] = l2info.GetAddress(name)
		ethTx := l2info.PrepareTx("Owner", name, l2info.TransferGas, big.NewInt(1e18), nil)
		ensureTx(t, builder, ethTx, l2client.SendTransaction(ctx, ethTx))
	}

	// Build batch: 5 token transfers (Owner) + 4 approvals (different users) + 1 transferFrom
	ownerNonce, err := l2client.PendingNonceAt(ctx, auth.From)
	Require(t, err)

	var txs types.Transactions

	// Owner transfers tokens to each user (explicit sequential nonces)
	for i := 0; i < numUsers; i++ {
		txTransfer, txErr := erc20.Transfer(noSendOpts(auth, ownerNonce+uint64(i)), users[i], big.NewInt(10000))
		Require(t, txErr)
		txs = append(txs, txTransfer)
	}

	// Each user approves the next user (different signers, each at nonce 0)
	for i := 0; i < numUsers-1; i++ {
		name := fmt.Sprintf("ERC20WF%d", i)
		userAuth := builder.L2Info.GetDefaultTransactOpts(name, ctx)
		txApprove, txErr := erc20.Approve(noSendOpts(&userAuth, 0), users[i+1], big.NewInt(5000))
		Require(t, txErr)
		txs = append(txs, txApprove)
	}

	// User1 executes transferFrom on user0's tokens (nonce 1, since Approve above used 0)
	user1Auth := builder.L2Info.GetDefaultTransactOpts("ERC20WF1", ctx)
	txFrom, err := erc20.TransferFrom(noSendOpts(&user1Auth, 1), users[0], users[2], big.NewInt(1000))
	Require(t, err)
	txs = append(txs, txFrom)

	blockNum, _ := sequenceInBlock(t, builder, txs)
	record(t, blockNum, builder)
}

// ---------------------------------------------------------------------------
// 26. Stylus multicall with mixed storage ops and logs
// ---------------------------------------------------------------------------

func TestRecordBlockWasmMulticallStoreAndLog(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	storageAddr := deployWasm(t, ctx, *auth, l2client, rustFile("storage"))
	multicallAddr := deployWasm(t, ctx, *auth, l2client, rustFile("multicall"))

	// Build multicall args using store with log emission
	args := multicallEmptyArgs()
	for i := 0; i < 16; i++ {
		key := testhelpers.RandomHash()
		value := testhelpers.RandomHash()
		args = multicallAppendStore(args, key, value, true, false)
	}
	// Also do some loads with logs
	for i := 0; i < 8; i++ {
		key := testhelpers.RandomHash()
		args = multicallAppendLoad(args, key, true)
	}

	// Wrap in a CALL to storage via multicall
	outerArgs := argsForMulticall(vm.CALL, multicallAddr, nil, args)

	tx := l2info.PrepareTxTo("Owner", &multicallAddr, 1e9, nil, outerArgs)
	receipt := ensureTx(t, builder, tx, l2client.SendTransaction(ctx, tx))

	record(t, receipt.BlockNumber.Uint64(), builder)

	_ = storageAddr
}

// ---------------------------------------------------------------------------
// 27. Multiple contract creations via Stylus CREATE
// ---------------------------------------------------------------------------

func TestRecordBlockWasmMultipleCreates(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	createAddr := deployWasm(t, ctx, *auth, l2client, rustFile("create"))

	// Sync l2info nonce (deployWasm via auth bypasses l2info counter)
	syncOwnerNonce(t, builder)

	deployWasmBytes, _ := readWasmFile(t, rustFile("storage"))
	deployCode := deployContractInitCode(deployWasmBytes, false)

	// Build 3 CREATE operations without sending
	const numCreates = 3
	txs := make(types.Transactions, numCreates)
	for i := 0; i < numCreates; i++ {
		startValue := testhelpers.RandomCallValue(1e12)
		create1Args := []byte{0x01}
		create1Args = append(create1Args, common.BigToHash(startValue).Bytes()...)
		create1Args = append(create1Args, deployCode...)

		txs[i] = l2info.PrepareTxTo("Owner", &createAddr, 1e9, startValue, create1Args)
	}

	blockNum, _ := sequenceInBlock(t, builder, txs)
	record(t, blockNum, builder)

	_ = l2client
}

// ---------------------------------------------------------------------------
// 28. Mixed: transfers + WASM calls + Solidity calls + deploys
// ---------------------------------------------------------------------------

func TestRecordBlockMixedAll(t *testing.T) {
	builder, auth, cleanup := recordBlockSetup(t)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	// Setup: deploy all contracts first
	_, tx, simple, err := localgen.DeploySimple(auth, l2client)
	ensureTx(t, builder, tx, err)

	storageAddr := deployWasm(t, ctx, *auth, l2client, rustFile("storage"))
	keccakAddr := deployWasm(t, ctx, *auth, l2client, rustFile("keccak"))

	_, tx, mock, err := localgen.DeployProgramTest(auth, l2client)
	ensureTx(t, builder, tx, err)

	// Sync l2info nonce (deploys via auth/deployWasm bypass l2info counter)
	syncOwnerNonce(t, builder)
	ownerNonce := l2info.GetInfoWithPrivKey("Owner").Nonce.Load()

	// Build batch: 3 transfers + 1 Solidity call + 1 WASM write + 1 keccak call
	var txs types.Transactions

	// 3 transfers (PrepareTx auto-increments l2info nonce)
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("AllMixUser%d", i)
		l2info.GenerateAccount(name)
		txs = append(txs, l2info.PrepareTx("Owner", name, l2info.TransferGas, big.NewInt(1e18), nil))
	}
	// ownerNonce+3 is the next available nonce

	// Solidity call (explicit nonce)
	txIncrementEmit, err := simple.IncrementEmit(noSendOpts(auth, ownerNonce+3))
	Require(t, err)
	txs = append(txs, txIncrementEmit)

	// Stylus storage write (PrepareTxTo with synced l2info nonce)
	l2info.GetInfoWithPrivKey("Owner").Nonce.Store(ownerNonce + 4)
	key := testhelpers.RandomHash()
	value := testhelpers.RandomHash()
	txStorage := l2info.PrepareTxTo("Owner", &storageAddr, l2info.TransferGas, nil, argsForStorageWrite(key, value))
	txs = append(txs, txStorage)

	// Stylus keccak call (explicit nonce)
	preimage := []byte("mixed test benchmark data")
	keccakArgs := []byte{0x01}
	keccakArgs = append(keccakArgs, preimage...)
	txKeccak, err := mock.CallKeccak(noSendOpts(auth, ownerNonce+5), keccakAddr, keccakArgs)
	Require(t, err)
	txs = append(txs, txKeccak)

	blockNum, _ := sequenceInBlock(t, builder, txs)
	record(t, blockNum, builder)
}
