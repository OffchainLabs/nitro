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

// stackOverflowWat is a WAT program that recurses 1,000 times using a
// counter in WASM memory. At a small native stack size (e.g., 64KB) this
// overflows the native stack before completing. At a larger stack (e.g., 1MB+)
// it completes and returns 0. Used to trigger native stack overflow for
// testing the recovery path.
var stackOverflowWat = `(module
	(memory 1 1)
	(export "memory" (memory 0))
	(func $main (export "user_entrypoint") (param $args_len i32) (result i32)
		;; Load counter from memory[0]
		i32.const 0
		i32.load
		;; If counter >= 1000, return 0
		i32.const 1000
		i32.ge_u
		if (result i32)
			i32.const 0
		else
			;; Increment counter in memory[0]
			i32.const 0
			i32.const 0
			i32.load
			i32.const 1
			i32.add
			i32.store
			;; Recurse
			i32.const 0
			call $main
		end
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

// TestProgramNativeStackOverflowRecovery exercises the on-chain stack overflow
// recovery path and verifies that:
//  1. Off-chain calls (eth_call) do NOT trigger any retry.
//  2. The first on-chain tx succeeds (overflow → cranelift retry, possibly
//     with stack doubling if cranelift also overflows at the original size).
//
// It deploys a WAT program that recurses 1,000 times and configures a small
// initial native stack size (64KB) so the first call overflows.
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

	// eth_call (off-chain): handleNativeStackOverflow skips all retries
	// for off-chain execution, so the call should fail.
	msg := ethereum.CallMsg{
		To:    &programAddress,
		Value: big.NewInt(0),
		Data:  []byte{},
		Gas:   32_000_000,
	}
	_, err := l2client.CallContract(ctx, msg, nil)
	if err == nil {
		t.Fatal("expected eth_call to fail (off-chain, no stack doubling), but it succeeded")
	}
	t.Logf("eth_call failed as expected (off-chain): %v", err)

	// On-chain tx: overflows → cranelift retry (may also double stack) → succeeds.
	tx := builder.L2Info.PrepareTxTo("Owner", &programAddress, builder.L2Info.TransferGas*10, nil, []byte{})
	err = l2client.SendTransaction(ctx, tx)
	Require(t, err)

	receipt, err := WaitForTx(ctx, l2client, tx.Hash(), 60*time.Second)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("expected on-chain tx to succeed after cranelift retry, got status=%d", receipt.Status)
	}
	t.Logf("tx succeeded after cranelift retry, block=%d", receipt.BlockNumber)

	blockNum, err := l2client.BlockNumber(ctx)
	Require(t, err)
	t.Logf("node still alive after overflow+recovery, current block: %d", blockNum)
}

// TestProgramNativeStackOverflowNoFallback tests the behavior when
// fallback is disabled. No cranelift retry or stack doubling is attempted
// and the call panics with a native stack overflow error.
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
	t.Logf("eth_call failed as expected (no fallback): %v", err)

	// On-chain tx: the sequencer panics during block creation (native stack
	// overflow with fallback disabled). The panic is caught by the sequencer's
	// recover handler, so the node survives but the tx is dropped.
	tx := builder.L2Info.PrepareTxTo("Owner", &programAddress, builder.L2Info.TransferGas*10, nil, []byte{})
	err = l2client.SendTransaction(ctx, tx)
	if err == nil {
		// Tx was accepted into the mempool. It will panic during sequencing
		// and be dropped. Verify the node is still alive.
		blockNum, err := l2client.BlockNumber(ctx)
		Require(t, err)
		t.Logf("node still alive after sequencer panic, current block: %d", blockNum)
	} else {
		t.Logf("tx rejected as expected: %v", err)
	}
}

// TestProgramCraneliftPersistenceIntegration verifies the full cranelift
// overflow recovery flow end-to-end:
//  1. Stylus call fails with native stack overflow (small stack)
//  2. allowFallback=true → cranelift retry (possibly with stack doubling) succeeds
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
	wasmDB := builder.L2.ExecNode.Backend.ArbInterface().BlockChain().StateCache().WasmStore()
	asmBefore := readCraneliftAsm(t, wasmDB)
	if len(asmBefore) > 0 {
		t.Fatalf("expected no cranelift ASM before overflow trigger, found %d bytes", len(asmBefore))
	}

	// Send an on-chain tx that triggers overflow → cranelift retry (may also double stack).
	tx := builder.L2Info.PrepareTxTo("Owner", &programAddress, builder.L2Info.TransferGas*10, nil, []byte{})
	err := l2client.SendTransaction(ctx, tx)
	Require(t, err)
	receipt, err := WaitForTx(ctx, l2client, tx.Hash(), 60*time.Second)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("expected first tx to succeed after cranelift retry, got status=%d", receipt.Status)
	}
	t.Logf("first tx: block=%d, succeeded after cranelift retry", receipt.BlockNumber)

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
	if receipt2.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("expected second tx to succeed (reusing cached cranelift), got status=%d", receipt2.Status)
	}
	t.Logf("second tx: block=%d, succeeded (reusing cached cranelift)", receipt2.BlockNumber)

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
// wasm store by iterating over the {0x00, 'c'} key prefix.
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
	val := it.Value()
	asm := make([]byte, len(val))
	copy(asm, val)
	return asm
}
