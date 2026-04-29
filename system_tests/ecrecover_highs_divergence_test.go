// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

// Tests that document the high-S ECRECOVER divergence between native Go execution and the
// WASM prover (k256 Rust library):
//
//   - Native (go-ethereum / decred secp256k1): EIP-2 high-S rejection applies only to
//     transaction signatures, not to the ECRECOVER precompile.  The precompile calls
//     ValidateSignatureValues(v, r, s, homestead=false), so s in the upper half of N is
//     accepted → returns a 32-byte address.
//
//   - Prover (WASM / k256 via go:wasmimport arbcrypto ecrecovery): k256 always rejects
//     high-S, returning an error → precompile returns 0 bytes.
//
// A transaction that triggers ECRECOVER with a high-S signature therefore produces
// different state roots on the sequencer and the validator, which is the divergence bug.

import (
	"context"
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// secp256k1 curve order N
var highSTestN = new(big.Int).SetBytes(common.Hex2Bytes("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141"))

// buildHighSInput returns the 128-byte ECRECOVER precompile input with a canonical high-S signature:
//
//	[  0: 32] hash = SHA256("high-s divergence")
//	[ 32: 64] v    = 27  (right-aligned, recovery id 0)
//	[ 64: 96] r    =  1  (right-aligned)
//	[ 96:128] s    = N-1 (right-aligned, high-S)
func buildHighSInput() [128]byte {
	hash := sha256.Sum256([]byte("high-s divergence"))
	sVal := new(big.Int).Sub(highSTestN, big.NewInt(1)) // N-1

	var input [128]byte
	copy(input[0:32], hash[:])
	input[63] = 27 // v = 27 in the least-significant byte of the 32-byte v field
	input[95] = 1  // r = 1 in the least-significant byte of the 32-byte r field
	sBytes := sVal.Bytes()
	copy(input[128-len(sBytes):128], sBytes)
	return input
}

// TestEcrecoverHighSDivergencePrecompile verifies directly that the native Go ECRECOVER
// precompile (address 0x1) accepts a high-S signature and returns a 32-byte address.
// The WASM prover uses k256 which rejects high-S and would return 0 bytes, causing a
// state-root mismatch.
func TestEcrecoverHighSDivergencePrecompile(t *testing.T) {
	precompile := vm.PrecompiledContractsOsaka[common.BytesToAddress([]byte{1})]
	if precompile == nil {
		t.Fatal("ecrecover precompile not found at address 0x1")
	}

	input := buildHighSInput()
	out, err := precompile.Run(input[:])
	if err != nil {
		t.Fatalf("native ECRECOVER returned an error for high-S input: %v", err)
	}
	if len(out) != 32 {
		t.Fatalf("native ECRECOVER returned %d bytes for high-S input, expected 32 — divergence not reproduced (high-S may have been rejected natively too)", len(out))
	}
	t.Logf("native ECRECOVER accepted high-S: recovered address %x", out[12:32])
	t.Log("the WASM prover (k256) returns 0 bytes for the same input → state-root divergence")
}

// TestEcrecoverHighSDivergenceOnChain deploys an EVM contract that triggers the ECRECOVER
// precompile with a high-S signature and records whether native execution accepted it:
//
//	storage[0] == 1  →  ECRECOVER returned 32 bytes (accepted high-S)
//	storage[0] == 2  →  ECRECOVER returned 0 bytes  (rejected)
//
// Under native L2 execution storage[0] must equal 1, confirming the divergence with the
// WASM prover which always produces storage[0] == 2 for the same transaction.
//
// Runtime bytecode layout (59 bytes):
//
//	calldatasize < 128  → VIEW mode:    SLOAD(0); return 32 bytes
//	calldatasize >= 128 → TRIGGER mode: CALLDATACOPY(0,0,128)
//	                                    STATICCALL(gas, 0x1, 0, 128, 128, 32)
//	                                    if RETURNDATASIZE == 32 → SSTORE(0, 1) ; STOP
//	                                    else                    → SSTORE(0, 2) ; STOP
func TestEcrecoverHighSDivergenceOnChain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	// Assembled runtime bytecode (59 bytes).  Offsets:
	//   0x28 (40) = JUMPDEST for native path (RETURNDATASIZE == 32)
	//   0x2f (47) = JUMPDEST for view path   (calldatasize < 128)
	runtime := common.Hex2Bytes(
		"36" + // CALLDATASIZE
			"6080" + // PUSH1 128
			"10" + // LT            (calldatasize < 128?)
			"602f" + // PUSH1 0x2f   (VIEW_OFFSET)
			"57" + // JUMPI
			// — trigger path: copy calldata[0:128] → mem[0:128] —
			"6080" + // PUSH1 128    (size)
			"6000" + // PUSH1 0      (destOffset)
			"6000" + // PUSH1 0      (dataOffset)
			"37" + // CALLDATACOPY
			// STATICCALL(gas, 0x1, argsOffset=0, argsSize=128, retOffset=128, retSize=32)
			"6020" + // PUSH1 32     (retSize)
			"6080" + // PUSH1 128    (retOffset)
			"6080" + // PUSH1 128    (argsSize)
			"6000" + // PUSH1 0      (argsOffset)
			"6001" + // PUSH1 1      (addr = ECRECOVER)
			"5a" + // GAS
			"fa" + // STATICCALL
			"50" + // POP           (discard success flag)
			"3d" + // RETURNDATASIZE
			"6020" + // PUSH1 32
			"14" + // EQ            (returndatasize == 32?)
			"6028" + // PUSH1 0x28   (NATIVE_OFFSET)
			"57" + // JUMPI
			// — prover path: k256 rejected high-S —
			"6002" + // PUSH1 2
			"6000" + // PUSH1 0
			"55" + // SSTORE(0, 2)
			"00" + // STOP
			// 0x28 JUMPDEST — native path: go-ethereum accepted high-S —
			"5b" + // JUMPDEST
			"6001" + // PUSH1 1
			"6000" + // PUSH1 0
			"55" + // SSTORE(0, 1)
			"00" + // STOP
			// 0x2f JUMPDEST — view path: return SLOAD(0) —
			"5b" + // JUMPDEST
			"6000" + // PUSH1 0
			"54" + // SLOAD(0)
			"6000" + // PUSH1 0
			"52" + // MSTORE(0, value)
			"6020" + // PUSH1 32
			"6000" + // PUSH1 0
			"f3", // RETURN(0, 32)
	)
	if len(runtime) != 59 {
		t.Fatalf("runtime bytecode length mismatch: got %d, want 59", len(runtime))
	}

	// Deploy code: copies runtime to memory and returns it.
	// Layout: PUSH1 runtimeLen, PUSH1 12 (deployOffset), PUSH1 0, CODECOPY, PUSH1 runtimeLen, PUSH1 0, RETURN
	deployCode := make([]byte, 0, 12+len(runtime))
	deployCode = append(deployCode,
		0x60, byte(len(runtime)), // PUSH1 runtimeLen
		0x60, 12, // PUSH1 deployOffset
		0x60, 0x00, // PUSH1 0
		0x39,                     // CODECOPY
		0x60, byte(len(runtime)), // PUSH1 runtimeLen
		0x60, 0x00, // PUSH1 0
		0xF3, // RETURN
	)
	deployCode = append(deployCode, runtime...)

	ownerAddr := builder.L2Info.GetAddress("Owner")
	deployGas, err := builder.L2.Client.EstimateGas(ctx, ethereum.CallMsg{
		From: ownerAddr,
		Data: deployCode,
	})
	Require(t, err)
	deployTx := builder.L2Info.PrepareTxTo("Owner", nil, deployGas, common.Big0, deployCode)
	Require(t, builder.L2.Client.SendTransaction(ctx, deployTx))
	deployReceipt, err := builder.L2.EnsureTxSucceeded(deployTx)
	Require(t, err)
	contractAddr := deployReceipt.ContractAddress

	input := buildHighSInput()

	triggerTx := builder.L2Info.PrepareTxTo("Owner", &contractAddr, 1e9, common.Big0, input[:])
	Require(t, builder.L2.Client.SendTransaction(ctx, triggerTx))
	_, err = builder.L2.EnsureTxSucceeded(triggerTx)
	Require(t, err)

	slot0, err := builder.L2.Client.StorageAt(ctx, contractAddr, common.Hash{}, nil)
	Require(t, err)
	slotVal := new(big.Int).SetBytes(slot0).Uint64()

	switch slotVal {
	case 1:
		// Native ECRECOVER accepted the high-S signature.
		// The WASM prover (k256) would produce slotVal == 2, causing a state-root divergence.
		t.Logf("storage[0] = 1: native ECRECOVER accepted high-S (prover would store 2 → state-root divergence confirmed)")
	case 2:
		t.Fatal("storage[0] = 2: native ECRECOVER also rejected high-S — divergence not reproduced; either the bug is fixed or the test input is invalid")
	default:
		t.Fatalf("storage[0] = %d: unexpected value, contract was not triggered correctly", slotVal)
	}
}
