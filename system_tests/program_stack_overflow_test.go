// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"bytes"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/offchainlabs/nitro/arbos/programs"
)

// stackOverflowWat is a WAT program that recurses deeply with multi-value
// loops, consuming native stack quickly in Singlepass. Used to trigger
// native stack overflow for testing the recovery path.
var stackOverflowWat = `(module
	(memory 0 0)
	(export "memory" (memory 0))
	(func $main (export "user_entrypoint") (param $args_len i32) (result i32)
		i32.const 0
		i32.const 0
		(loop $outer (param i32 i32) (result i32 i32)
			(loop $inner (param i32 i32) (result i32 i32))
		)
		drop
		drop
		i32.const 0
		call $main
	)
)`

var (
	stackOverflowWatPath string
	stackOverflowWatOnce sync.Once
)

// stackOverflowWatFile returns the path to a temp .wat file containing the
// stack-overflow program. The file is created once and reused across tests.
// deployWasm expects a file path, and .wat files are globally gitignored
// so we can't check one in.
func stackOverflowWatFile(t *testing.T) string {
	t.Helper()
	stackOverflowWatOnce.Do(func() {
		dir, err := os.MkdirTemp("", "stylus-test-wat-*")
		Require(t, err)
		stackOverflowWatPath = filepath.Join(dir, "stack-overflow.wat")
		err = os.WriteFile(stackOverflowWatPath, []byte(stackOverflowWat), 0644) //nolint:gosec
		Require(t, err)
	})
	return stackOverflowWatPath
}

// saveAndRestoreNativeStackGlobals saves the current process-wide Wasmer stack
// size and allowFallback flag, restoring both when the test finishes. Tests that
// modify these via StylusTargetConfig must call this to avoid polluting other
// tests running in the same process.
func saveAndRestoreNativeStackGlobals(t *testing.T) {
	t.Helper()
	savedSize := programs.GetNativeStackSize()
	savedFallback := programs.GetAllowFallback()
	t.Cleanup(func() {
		programs.SetInitialNativeStackSize(savedSize)
		programs.DrainStackPool()
		programs.SetAllowFallback(savedFallback)
	})
}

// TestProgramNativeStackOverflowRecovery exercises the on-chain
// callProgram → retryOnStackOverflow → cranelift fallback path and verifies
// that off-chain calls (eth_call) do NOT trigger the retry.
//
// It deploys a WAT program that recurses deeply with multi-value loops
// (consuming native stack quickly in Singlepass) and configures a small
// initial native stack size so the first call overflows. The Go-side retry
// logic (cranelift recompilation + stack doubling) should recover, and the
// tx should be included in a block without crashing the node. The program
// still recurses infinitely so it ultimately reverts (out-of-stack/ink).
func TestProgramNativeStackOverflowRecovery(t *testing.T) {
	saveAndRestoreNativeStackGlobals(t)
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.DontParalellise()                                   // mutates process-wide native stack size
		b.execConfig.StylusTarget.NativeStackSize = 64 * 1024 // 64 KB
		b.execConfig.StylusTarget.AllowFallback = true
		b.WithExtraArchs([]string{string(rawdb.LocalTarget())})
	})
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	programAddress := deployWasm(t, ctx, auth, l2client, stackOverflowWatFile(t))

	// eth_call (off-chain): retryOnStackOverflow skips retry for off-chain
	// execution (IsExecutedOnChain()=false), so the call should fail even
	// though AllowFallback=true.
	msg := ethereum.CallMsg{
		To:    &programAddress,
		Value: big.NewInt(0),
		Data:  []byte{},
		Gas:   32_000_000,
	}
	_, err := l2client.CallContract(ctx, msg, nil)
	if err == nil {
		t.Fatal("expected eth_call to fail (off-chain never retries), but it succeeded")
	}
	t.Logf("eth_call reverted as expected (off-chain, no retry): %v", err)

	// On-chain tx: IsExecutedOnChain()=true enables the cranelift fallback.
	// The program still recurses infinitely so it reverts after recovery.
	tx := builder.L2Info.PrepareTxTo("Owner", &programAddress, builder.L2Info.TransferGas*10, nil, []byte{})
	err = l2client.SendTransaction(ctx, tx)
	Require(t, err)

	receipt, err := WaitForTx(ctx, l2client, tx.Hash(), 60*time.Second)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusFailed {
		t.Fatalf("expected on-chain tx to revert (infinite recursion), got status=%d", receipt.Status)
	}
	t.Logf("tx included in block %d, reverted as expected", receipt.BlockNumber)

	blockNum, err := l2client.BlockNumber(ctx)
	Require(t, err)
	t.Logf("node still alive after overflow+recovery, current block: %d", blockNum)
}

// TestProgramNativeStackOverflowNoFallback tests the behavior when
// cranelift fallback is disabled. No retry is attempted and the call
// should revert with a depth error.
func TestProgramNativeStackOverflowNoFallback(t *testing.T) {
	saveAndRestoreNativeStackGlobals(t)
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.DontParalellise() // mutates process-wide native stack size
		b.execConfig.StylusTarget.NativeStackSize = 64 * 1024
		b.execConfig.StylusTarget.AllowFallback = false
		b.WithExtraArchs([]string{string(rawdb.LocalTarget())})
	})
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	programAddress := deployWasm(t, ctx, auth, l2client, stackOverflowWatFile(t))

	// eth_call: should fail because fallback is disabled and off-chain
	// execution never retries.
	msg := ethereum.CallMsg{
		To:    &programAddress,
		Value: big.NewInt(0),
		Data:  []byte{},
		Gas:   32_000_000,
	}
	_, err := l2client.CallContract(ctx, msg, nil)
	if err == nil {
		t.Fatal("expected call to fail with no fallback, but it succeeded")
	}
	t.Logf("eth_call reverted as expected (no fallback): %v", err)

	// On-chain tx: should also fail (included in block but reverted) because
	// AllowFallback=false means no cranelift retry even on-chain.
	tx := builder.L2Info.PrepareTxTo("Owner", &programAddress, builder.L2Info.TransferGas*10, nil, []byte{})
	err = l2client.SendTransaction(ctx, tx)
	Require(t, err)
	receipt, err := WaitForTx(ctx, l2client, tx.Hash(), 60*time.Second)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusFailed {
		t.Fatalf("expected on-chain tx to revert with fallback disabled, got status=%d", receipt.Status)
	}

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

// TestProgramCraneliftPersistenceIntegration verifies the full cranelift
// fallback flow end-to-end:
//  1. Stylus call fails with native stack overflow (singlepass, tiny stack)
//  2. allowFallback=true → cranelift compilation + call succeeds
//  3. Cranelift ASM is persisted to the wasm store
//  4. A second on-chain tx reuses the persisted cranelift ASM (no recompilation)
func TestProgramCraneliftPersistenceIntegration(t *testing.T) {
	saveAndRestoreNativeStackGlobals(t)
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.DontParalellise() // mutates process-wide native stack size
		b.execConfig.StylusTarget.NativeStackSize = 64 * 1024
		b.execConfig.StylusTarget.AllowFallback = true
		b.WithExtraArchs([]string{string(rawdb.LocalTarget())})
	})
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	programAddress := deployWasm(t, ctx, auth, l2client, stackOverflowWatFile(t))

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
	// The program recurses infinitely — even after cranelift fallback it hits
	// out-of-stack/ink and reverts. Cranelift ASM is persisted during the retry
	// (compilation step), regardless of the final execution outcome.
	if receipt.Status != types.ReceiptStatusFailed {
		t.Fatalf("expected first tx to revert (infinite recursion), got status=%d", receipt.Status)
	}
	t.Logf("first tx: block=%d, reverted as expected", receipt.BlockNumber)

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
	if receipt2.Status != types.ReceiptStatusFailed {
		t.Fatalf("expected second tx to revert (infinite recursion), got status=%d", receipt2.Status)
	}
	t.Logf("second tx: block=%d, reverted as expected", receipt2.BlockNumber)

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
