// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build block_recording

package arbtest

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

// ---------------------------------------------------------------------------
// 1. Pure ETH transfer — no contract interactions
// ---------------------------------------------------------------------------

func TestRecordBlockTransfer(t *testing.T) {
	builder, _, cleanup := setupProgramTest(t, true)
	l2info := builder.L2Info
	defer cleanup()

	l2info.GenerateAccount("Receiver")
	tx := l2info.PrepareTx("Owner", "Receiver", l2info.TransferGas, big.NewInt(1e16), nil)
	Require(t, builder.L2.Client.SendTransaction(builder.ctx, tx))
	receipt := ensureTx(t, builder, tx)

	record(t, receipt.BlockNumber.Uint64(), builder)
}

// ---------------------------------------------------------------------------
// 2. EVM contract calls — 20 Solidity SSTORE operations, no Stylus
// ---------------------------------------------------------------------------

func TestRecordBlockSolidity(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true)
	l2client := builder.L2.Client
	defer cleanup()

	_, tx, simple, err := localgen.DeploySimple(&auth, l2client)
	Require(t, err)
	ensureTx(t, builder, tx)

	nonce, err := l2client.PendingNonceAt(builder.ctx, auth.From)
	Require(t, err)
	const numIncrements = 20
	txs := make(types.Transactions, numIncrements)
	for i := 0; i < numIncrements; i++ {
		txs[i], err = simple.Increment(noSendOpts(&auth, nonce+uint64(i)))
		Require(t, err)
	}

	blockNum, _ := sequenceInBlock(t, builder, txs)
	record(t, blockNum, builder)
}

// ---------------------------------------------------------------------------
// 3. Single Stylus call — one WASM storage write
// ---------------------------------------------------------------------------

func TestRecordBlockStylus(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	programAddress := deployWasm(t, ctx, auth, l2client, rustFile("storage"))

	key := testhelpers.RandomHash()
	value := testhelpers.RandomHash()
	tx := l2info.PrepareTxTo("Owner", &programAddress, l2info.TransferGas, nil, argsForStorageWrite(key, value))
	Require(t, l2client.SendTransaction(ctx, tx))
	receipt := ensureTx(t, builder, tx)

	record(t, receipt.BlockNumber.Uint64(), builder)
}

// ---------------------------------------------------------------------------
// 4. Heavy Stylus — 32 cross-contract read/write pairs via multicall
// ---------------------------------------------------------------------------

func TestRecordBlockStylusHeavy(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	storageAddr := deployWasm(t, ctx, auth, l2client, rustFile("storage"))
	multicallAddr := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))

	args := multicallEmptyArgs()
	for i := 0; i < 32; i++ {
		key := testhelpers.RandomHash()
		value := testhelpers.RandomHash()
		args = multicallAppend(args, vm.CALL, storageAddr, argsForStorageWrite(key, value))
		args = multicallAppend(args, vm.CALL, storageAddr, argsForStorageRead(key))
	}

	tx := l2info.PrepareTxTo("Owner", &multicallAddr, 1e9, nil, args)
	Require(t, l2client.SendTransaction(ctx, tx))
	receipt := ensureTx(t, builder, tx)

	record(t, receipt.BlockNumber.Uint64(), builder)
}

// ---------------------------------------------------------------------------
// 5. Mixed heavy block — ETH transfers + EVM call + multiple Stylus programs
// ---------------------------------------------------------------------------

func TestRecordBlockMixed(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	_, tx, simple, err := localgen.DeploySimple(&auth, l2client)
	Require(t, err)
	ensureTx(t, builder, tx)

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

	txIncrementEmit, err := simple.IncrementEmit(noSendOpts(&auth, ownerNonce+3))
	Require(t, err)
	txs = append(txs, txIncrementEmit)

	// PrepareTxTo uses l2info's nonce counter; sync it past the NoSend tx above.
	l2info.GetInfoWithPrivKey("Owner").Nonce.Store(ownerNonce + 4)
	txStorage := l2info.PrepareTxTo("Owner", &storageAddr, l2info.TransferGas, nil, argsForStorageWrite(testhelpers.RandomHash(), testhelpers.RandomHash()))
	txs = append(txs, txStorage)

	// Keccak Stylus program takes 0x01 || preimage directly — no wrapper needed.
	txKeccak := l2info.PrepareTxTo("Owner", &keccakAddr, l2info.TransferGas, nil, append([]byte{0x01}, []byte("mixed test benchmark data")...))
	txs = append(txs, txKeccak)

	blockNum, _ := sequenceInBlock(t, builder, txs)
	record(t, blockNum, builder)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// ensureTx waits for a transaction to be included and asserts success.
func ensureTx(t *testing.T, builder *NodeBuilder, tx *types.Transaction) *types.Receipt {
	t.Helper()
	receipt, err := EnsureTxSucceeded(builder.ctx, builder.L2.Client, tx)
	Require(t, err)
	return receipt
}

// record dumps block inputs for the given block number.
func record(t *testing.T, blockNum uint64, builder *NodeBuilder) {
	t.Helper()
	recordBlock(t, blockNum, builder, rawdb.TargetWavm, rawdb.TargetWasm, rawdb.LocalTarget())
}

// sequenceInBlock bypasses the sequencer's one-tx-per-block loop and forces all
// txs into a single block via the execution engine directly.
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

// syncOwnerNonce syncs l2info's internal nonce counter for "Owner" with the
// on-chain pending nonce. Needed after txs sent via auth (TransactOpts) that
// bypass l2info's counter, such as DeploySimple or deployWasm.
func syncOwnerNonce(t *testing.T, builder *NodeBuilder) {
	t.Helper()
	nonce, err := builder.L2.Client.PendingNonceAt(builder.ctx, builder.L2Info.GetAddress("Owner"))
	Require(t, err)
	builder.L2Info.GetInfoWithPrivKey("Owner").Nonce.Store(nonce)
}

// noSendOpts returns auth with NoSend=true, an explicit nonce, and a fixed gas
// limit. The fixed gas limit skips eth_estimateGas, required when batched txs
// depend on each other and cannot be estimated independently.
func noSendOpts(auth *bind.TransactOpts, nonce uint64) *bind.TransactOpts {
	opts := *auth
	opts.NoSend = true
	opts.Nonce = new(big.Int).SetUint64(nonce)
	opts.GasLimit = 32000000
	return &opts
}
