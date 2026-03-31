// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"bytes"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

// TestProgramNativeStackOverflowRecovery exercises the full
// callProgram → retryOnStackOverflow path with a real EVM context.
//
// It deploys a WAT program that recurses deeply with multi-value loops
// (consuming native stack quickly in Singlepass) and configures a small
// initial native stack size so the first call overflows. The Go-side retry
// logic (cranelift recompilation + stack doubling) should recover, and the
// program should terminate normally (out-of-ink or out-of-stack) rather
// than crashing the node.
func TestProgramNativeStackOverflowRecovery(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.execConfig.StylusTarget.NativeStackSize = 64 * 1024 // 64 KB
		b.execConfig.StylusTarget.AllowFallback = true
		b.WithExtraArchs([]string{string(rawdb.LocalTarget())})
	})
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	programAddress := deployWasm(t, ctx, auth, l2client, watFile("stack-overflow"))

	msg := ethereum.CallMsg{
		To:    &programAddress,
		Value: big.NewInt(0),
		Data:  []byte{},
		Gas:   32_000_000,
	}
	_, err := l2client.CallContract(ctx, msg, nil)
	if err != nil {
		t.Logf("call reverted as expected: %v", err)
	} else {
		t.Log("call succeeded (program terminated normally after retry)")
	}

	blockNum, err := l2client.BlockNumber(ctx)
	Require(t, err)
	t.Logf("node still alive, current block: %d", blockNum)
}

// TestProgramNativeStackOverflowNoFallback tests the behavior when
// cranelift fallback is disabled. No retry is attempted and the call
// should revert immediately.
func TestProgramNativeStackOverflowNoFallback(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.execConfig.StylusTarget.NativeStackSize = 64 * 1024
		b.execConfig.StylusTarget.AllowFallback = false
		b.WithExtraArchs([]string{string(rawdb.LocalTarget())})
	})
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	programAddress := deployWasm(t, ctx, auth, l2client, watFile("stack-overflow"))

	msg := ethereum.CallMsg{
		To:    &programAddress,
		Value: big.NewInt(0),
		Data:  []byte{},
		Gas:   32_000_000,
	}
	_, err := l2client.CallContract(ctx, msg, nil)
	if err != nil {
		t.Logf("call reverted as expected (no fallback, no retry): %v", err)
	} else {
		t.Fatal("expected call to fail with no fallback, but it succeeded")
	}

	// Also send an on-chain tx to exercise the on-chain path with fallback disabled.
	tx := builder.L2Info.PrepareTxTo("Owner", &programAddress, builder.L2Info.TransferGas*10, nil, []byte{})
	err = l2client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = WaitForTx(ctx, l2client, tx.Hash(), 60*time.Second)
	Require(t, err)

	// Verify no cranelift ASM was persisted — fallback was disabled so
	// cranelift compilation should never have been attempted.
	wasmDB := builder.L2.ExecNode.Backend.ArbInterface().BlockChain().StateCache().WasmStore()
	asm := readCraneliftAsm(t, wasmDB)
	if len(asm) > 0 {
		t.Fatalf("cranelift ASM should NOT be in wasm store with fallback disabled, found %d bytes", len(asm))
	}

	blockNum, err := l2client.BlockNumber(ctx)
	Require(t, err)
	t.Logf("node still alive, current block: %d", blockNum)
}

// TestProgramNativeStackOverflowViaTransaction exercises the retry path
// through an actual on-chain transaction (not just eth_call). On-chain
// execution uses IsExecutedOnChain()=true, enabling the cranelift fallback
// when AllowFallback=true.
func TestProgramNativeStackOverflowViaTransaction(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.execConfig.StylusTarget.NativeStackSize = 64 * 1024
		b.execConfig.StylusTarget.AllowFallback = true
		b.WithExtraArchs([]string{string(rawdb.LocalTarget())})
	})
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	programAddress := deployWasm(t, ctx, auth, l2client, watFile("stack-overflow"))

	tx := builder.L2Info.PrepareTxTo("Owner", &programAddress, builder.L2Info.TransferGas*10, nil, []byte{})
	err := l2client.SendTransaction(ctx, tx)
	Require(t, err)

	receipt, err := WaitForTx(ctx, l2client, tx.Hash(), 60*time.Second)
	Require(t, err)
	t.Logf("tx included in block %d, status=%d (0=fail, 1=success)", receipt.BlockNumber, receipt.Status)

	blockNum, err := l2client.BlockNumber(ctx)
	Require(t, err)
	t.Logf("node still alive, current block: %d", blockNum)
}

// TestProgramCraneliftPersistenceIntegration verifies the full cranelift
// fallback flow end-to-end:
//  1. Stylus call fails with native stack overflow (singlepass, tiny stack)
//  2. allowFallback=true → cranelift compilation + call succeeds
//  3. Cranelift ASM is persisted to the wasm store
//  4. A second on-chain tx reuses the persisted cranelift ASM (no recompilation)
func TestProgramCraneliftPersistenceIntegration(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.execConfig.StylusTarget.NativeStackSize = 64 * 1024
		b.execConfig.StylusTarget.AllowFallback = true
		b.WithExtraArchs([]string{string(rawdb.LocalTarget())})
	})
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	programAddress := deployWasm(t, ctx, auth, l2client, watFile("stack-overflow"))

	// Verify wasm store has NO cranelift ASM before the overflow-triggering call.
	// Cranelift keys use the {0x00, 'c'} prefix, distinct from singlepass {0x00, 'w'}.
	wasmDB := builder.L2.ExecNode.Backend.ArbInterface().BlockChain().StateCache().WasmStore()
	asmBefore := readCraneliftAsm(t, wasmDB)
	if len(asmBefore) > 0 {
		t.Fatalf("expected no cranelift ASM before overflow trigger, found %d bytes", len(asmBefore))
	}

	// Send an on-chain tx that triggers singlepass overflow → cranelift fallback.
	tx := builder.L2Info.PrepareTxTo("Owner", &programAddress, builder.L2Info.TransferGas*10, nil, []byte{})
	err := l2client.SendTransaction(ctx, tx)
	Require(t, err)
	receipt, err := WaitForTx(ctx, l2client, tx.Hash(), 60*time.Second)
	Require(t, err)
	t.Logf("first tx: block=%d, status=%d", receipt.BlockNumber, receipt.Status)

	// Verify cranelift ASM was persisted to the wasm store.
	asmAfterFirst := readCraneliftAsm(t, wasmDB)
	if len(asmAfterFirst) == 0 {
		t.Fatal("cranelift ASM was NOT persisted to wasm store after overflow + fallback")
	}
	t.Logf("cranelift ASM persisted after first tx: %d bytes", len(asmAfterFirst))

	// Second on-chain tx should reuse the persisted cranelift ASM (no recompilation).
	tx2 := builder.L2Info.PrepareTxTo("Owner", &programAddress, builder.L2Info.TransferGas*10, nil, []byte{})
	err = l2client.SendTransaction(ctx, tx2)
	Require(t, err)
	receipt2, err := WaitForTx(ctx, l2client, tx2.Hash(), 60*time.Second)
	Require(t, err)
	t.Logf("second tx: block=%d, status=%d", receipt2.BlockNumber, receipt2.Status)

	// ASM should be identical (reused from store, not recompiled).
	asmAfterSecond := readCraneliftAsm(t, wasmDB)
	if !bytes.Equal(asmAfterFirst, asmAfterSecond) {
		t.Fatalf("cranelift ASM changed between txs: %d bytes → %d bytes (expected identical reuse)",
			len(asmAfterFirst), len(asmAfterSecond))
	}

	blockNum, err := l2client.BlockNumber(ctx)
	Require(t, err)
	t.Logf("node survived both txs, current block: %d", blockNum)
}

// readCraneliftAsm reads the first cranelift-compiled ASM entry from the
// wasm store by iterating over the {0x00, 'c'} key prefix and returns the
// raw ASM bytes (nil if none found).
// We iterate rather than reading by moduleHash directly because the wasm store
// is keyed by moduleHash (derived during WASM activation), not by codeHash
// (returned by statedb.GetCodeHash). There is no public API to look up a
// contract's moduleHash from a system test without going through ArbOS state.
func readCraneliftAsm(t *testing.T, wasmDB ethdb.KeyValueStore) []byte {
	t.Helper()
	craneliftPrefix := []byte{0x00, 'c'}
	it := wasmDB.NewIterator(craneliftPrefix, nil)
	defer it.Release()
	if !it.Next() {
		return nil
	}
	key := it.Key()
	if len(key) != rawdb.WasmKeyLen {
		t.Fatalf("unexpected cranelift wasm key length: %d, key: %v", len(key), key)
	}
	// Copy value since iterator may reuse the buffer.
	val := it.Value()
	asm := make([]byte, len(val))
	copy(asm, val)
	return asm
}
