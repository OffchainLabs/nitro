// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build block_recording

package arbtest

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

// fixedRecordGasLimit is used instead of eth_estimateGas for txs that are
// batched or depend on prior unmined txs and therefore cannot be estimated
// independently.
const fixedRecordGasLimit = 32_000_000

// ---------------------------------------------------------------------------
// Pure ETH transfer — no contract interactions
// ---------------------------------------------------------------------------

func TestRecordBlockTransfer(t *testing.T) {
	builder, _, cleanup := setupRecordingTest(t)
	l2info := builder.L2Info
	defer cleanup()

	l2info.GenerateAccount("Receiver")
	tx := l2info.PrepareTx("Owner", "Receiver", l2info.TransferGas, big.NewInt(1e16), nil)
	Require(t, builder.L2.Client.SendTransaction(builder.ctx, tx))
	receipt := requireTxSucceeded(t, builder, tx)

	recordBlockInputs(t, receipt.BlockNumber.Uint64(), builder)
}

// ---------------------------------------------------------------------------
// EVM contract calls — 20 Solidity SSTORE operations, no Stylus
// ---------------------------------------------------------------------------

func TestRecordBlockSolidity(t *testing.T) {
	builder, auth, cleanup := setupRecordingTest(t)
	l2client := builder.L2.Client
	defer cleanup()

	_, tx, simple, err := localgen.DeploySimple(&auth, l2client)
	Require(t, err)
	requireTxSucceeded(t, builder, tx)

	nonce, err := l2client.PendingNonceAt(builder.ctx, auth.From)
	Require(t, err)
	const numIncrements = 20
	txs := make(types.Transactions, numIncrements)
	for i := 0; i < numIncrements; i++ {
		txs[i], err = simple.Increment(noSendOpts(&auth, nonce+uint64(i)))
		Require(t, err)
	}

	blockNum := sequenceInBlock(t, builder, txs)
	recordBlockInputs(t, blockNum, builder)
}

// ---------------------------------------------------------------------------
// Single Stylus call — one WASM storage write
// ---------------------------------------------------------------------------

func TestRecordBlockStylus(t *testing.T) {
	builder, auth, cleanup := setupRecordingTest(t)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	programAddress := deployWasm(t, ctx, auth, l2client, rustFile("storage"))

	key, value := recordingKV(0)
	tx := l2info.PrepareTxTo("Owner", &programAddress, l2info.TransferGas, nil, argsForStorageWrite(key, value))
	Require(t, l2client.SendTransaction(ctx, tx))
	receipt := requireTxSucceeded(t, builder, tx)

	recordBlockInputs(t, receipt.BlockNumber.Uint64(), builder)
}

// ---------------------------------------------------------------------------
// Heavy Stylus — 32 cross-contract read/write pairs via multicall
// ---------------------------------------------------------------------------

func TestRecordBlockStylusHeavy(t *testing.T) {
	builder, auth, cleanup := setupRecordingTest(t)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	storageAddr := deployWasm(t, ctx, auth, l2client, rustFile("storage"))
	multicallAddr := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))

	args := multicallEmptyArgs()
	for i := 0; i < 32; i++ {
		key, value := recordingKV(i)
		args = multicallAppend(args, vm.CALL, storageAddr, argsForStorageWrite(key, value))
		args = multicallAppend(args, vm.CALL, storageAddr, argsForStorageRead(key))
	}

	tx := l2info.PrepareTxTo("Owner", &multicallAddr, 1e9, nil, args)
	Require(t, l2client.SendTransaction(ctx, tx))
	receipt := requireTxSucceeded(t, builder, tx)

	recordBlockInputs(t, receipt.BlockNumber.Uint64(), builder)
}

// ---------------------------------------------------------------------------
// Mixed heavy block — ETH transfers + EVM call + multiple Stylus programs
// ---------------------------------------------------------------------------

func TestRecordBlockMixed(t *testing.T) {
	builder, auth, cleanup := setupRecordingTest(t)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	_, tx, simple, err := localgen.DeploySimple(&auth, l2client)
	Require(t, err)
	requireTxSucceeded(t, builder, tx)

	storageAddr := deployWasm(t, ctx, auth, l2client, rustFile("storage"))
	keccakAddr := deployWasm(t, ctx, auth, l2client, rustFile("keccak"))

	syncOwnerNonce(t, builder)
	ownerNonce := l2info.GetInfoWithPrivKey("Owner").Nonce.Load()

	var txs types.Transactions

	// 3 ETH transfers; PrepareTx auto-increments the l2info nonce.
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("AllMixUser%d", i)
		l2info.GenerateAccount(name)
		txs = append(txs, l2info.PrepareTx("Owner", name, l2info.TransferGas, big.NewInt(1e18), nil))
	}

	txIncrementEmit, err := simple.IncrementEmit(noSendOpts(&auth, ownerNonce+uint64(len(txs))))
	Require(t, err)
	txs = append(txs, txIncrementEmit)

	// The NoSend tx's signer already advanced l2info's counter; defensively
	// re-assert the expected value before handing the counter back to
	// PrepareTxTo.
	l2info.GetInfoWithPrivKey("Owner").Nonce.Store(ownerNonce + uint64(len(txs)))
	txStorage := l2info.PrepareTxTo("Owner", &storageAddr, l2info.TransferGas, nil, argsForStorageWrite(recordingKV(0)))
	txs = append(txs, txStorage)

	// Keccak Stylus program takes 0x01 || preimage directly — no wrapper needed.
	txKeccak := l2info.PrepareTxTo("Owner", &keccakAddr, l2info.TransferGas, nil, append([]byte{0x01}, []byte("mixed test benchmark data")...))
	txs = append(txs, txKeccak)

	blockNum := sequenceInBlock(t, builder, txs)
	recordBlockInputs(t, blockNum, builder)
}

// ---------------------------------------------------------------------------
// Stylus activation — the recorded block contains the
//    ArbWasm.activateProgram tx itself, exercising the activate_v2 hostio.
// ---------------------------------------------------------------------------

func TestRecordBlockStylusActivation(t *testing.T) {
	recordStylusActivation(t, rustFile("storage"))
}

// Same as above but activating a larger program; together the two give
// datapoints for how activation cost scales with program size.
func TestRecordBlockStylusActivationMulticall(t *testing.T) {
	recordStylusActivation(t, rustFile("multicall"))
}

// recordStylusActivation deploys the wasm without activating it (unlike
// deployWasm), then records the block containing the activation tx itself.
func recordStylusActivation(t *testing.T, file string) {
	builder, auth, cleanup := setupRecordingTest(t)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	wasm, _ := readWasmFile(t, file)
	auth.GasLimit = fixedRecordGasLimit
	program := deployContract(t, ctx, auth, l2client, wasm)

	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)
	auth.Value = oneEth
	tx, err := arbWasm.ActivateProgram(&auth, program)
	Require(t, err)
	receipt := requireTxSucceeded(t, builder, tx)

	recordBlockInputs(t, receipt.BlockNumber.Uint64(), builder)
}

// ---------------------------------------------------------------------------
// Signature-heavy block — many ETH transfers in a single block to amplify
//    ECRecover (sender recovery) signal in profile snapshots.
// ---------------------------------------------------------------------------

func TestRecordBlockSignatures(t *testing.T) {
	builder, _, cleanup := setupRecordingTest(t)
	l2info := builder.L2Info
	defer cleanup()

	const numTransfers = 50
	txs := make(types.Transactions, numTransfers)
	for i := 0; i < numTransfers; i++ {
		name := fmt.Sprintf("SigReceiver%d", i)
		l2info.GenerateAccount(name)
		txs[i] = l2info.PrepareTx("Owner", name, l2info.TransferGas, big.NewInt(1e16), nil)
	}

	blockNum := sequenceInBlock(t, builder, txs)
	recordBlockInputs(t, blockNum, builder)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// setupRecordingTest wraps setupProgramTest, pinning the ink price back to
// the chain default: setupProgramTest randomizes it (1..20000), which makes
// Stylus gas costs swing 10,000x between runs — enough to run the heavy
// multicall test out of gas against ArbOS's per-tx gas clamp — and makes
// recordings non-comparable across runs.
func setupRecordingTest(t *testing.T) (*NodeBuilder, bind.TransactOpts, func()) {
	builder, auth, cleanup := setupProgramTest(t, true)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	tx, err := arbOwner.SetInkPrice(&auth, 10_000)
	Require(t, err)
	requireTxSucceeded(t, builder, tx)
	return builder, auth, cleanup
}

// recordingKV returns deterministic storage key/value pairs so recorded
// blocks don't vary across runs.
func recordingKV(i int) (common.Hash, common.Hash) {
	key := crypto.Keccak256Hash([]byte(fmt.Sprintf("recording-key-%d", i)))
	value := crypto.Keccak256Hash([]byte(fmt.Sprintf("recording-value-%d", i)))
	return key, value
}

// requireTxSucceeded waits for a transaction to be included and asserts success.
func requireTxSucceeded(t *testing.T, builder *NodeBuilder, tx *types.Transaction) *types.Receipt {
	t.Helper()
	receipt, err := EnsureTxSucceeded(builder.ctx, builder.L2.Client, tx)
	Require(t, err)
	return receipt
}

// recordBlockInputs dumps block inputs for the given block number.
func recordBlockInputs(t *testing.T, blockNum uint64, builder *NodeBuilder) {
	t.Helper()
	recordBlock(t, blockNum, builder, rawdb.TargetWavm, rawdb.TargetWasm, rawdb.LocalTarget())
}

// sequenceInBlock bypasses the sequencer's one-tx-per-block loop and forces
// all txs into a single block, asserting each succeeded. Mirrors
// sequenceTransactionsInTheSameBlock (common_test.go), expanded here to pin
// the header timestamp to the parent block's so that recorded blocks don't
// depend on wall-clock time.
func sequenceInBlock(t *testing.T, builder *NodeBuilder, txs types.Transactions) uint64 {
	t.Helper()
	ctx := builder.ctx
	l2client := builder.L2.Client

	sequencer := builder.L2.ExecNode.Sequencer
	sequencer.Pause()
	defer sequencer.Activate()

	header, hooks := sequencer.MakeSameBlockSequencingHooksAndHeaderForTest(t, txs)
	lastBlock, err := l2client.BlockByNumber(ctx, nil)
	Require(t, err)
	header.Timestamp = lastBlock.Time()

	block, _ := sequenceTransactions(t, builder, header, hooks)
	sequencer.DispatchPendingFilteredTxReportsForTest(t)

	for i, tx := range txs {
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		if receipt.BlockNumber.Uint64() != block.NumberU64() {
			t.Fatalf("tx %d in block %d, expected block %d", i, receipt.BlockNumber.Uint64(), block.NumberU64())
		}
	}
	return block.NumberU64()
}

// syncOwnerNonce defensively aligns l2info's internal nonce counter for
// "Owner" with the on-chain pending nonce. Auth-signed txs (DeploySimple,
// deployWasm) do advance the counter via the signer callback, so this is a
// re-assert rather than a correction — kept so the nonce bookkeeping in
// TestRecordBlockMixed doesn't silently depend on that implementation
// detail.
func syncOwnerNonce(t *testing.T, builder *NodeBuilder) {
	t.Helper()
	nonce, err := builder.L2.Client.PendingNonceAt(builder.ctx, builder.L2Info.GetAddress("Owner"))
	Require(t, err)
	builder.L2Info.GetInfoWithPrivKey("Owner").Nonce.Store(nonce)
}

// noSendOpts returns auth with NoSend=true, an explicit nonce, and a fixed
// gas limit (see fixedRecordGasLimit).
func noSendOpts(auth *bind.TransactOpts, nonce uint64) *bind.TransactOpts {
	opts := *auth
	opts.NoSend = true
	opts.Nonce = new(big.Int).SetUint64(nonce)
	opts.GasLimit = fixedRecordGasLimit
	return &opts
}
