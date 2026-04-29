// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !race

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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/validator/valnode"
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
		t.Fatalf("native ECRECOVER returned %d bytes for high-S input, expected 32 — divergence not reproduced", len(out))
	}
	t.Logf("native ECRECOVER accepted high-S: recovered address %x", out[12:32])
}

// TestEcrecoverHighSDivergence deploys a contract that triggers the ECRECOVER precompile
// with a high-S signature and verifies consistent behaviour across native and JIT execution:
//
//  1. On-chain storage check: native execution stores 1 (accepted) rather than 2 (rejected).
//
//  2. JIT block re-execution: StatelessBlockValidator.ValidateResult re-executes the trigger
//     block through the k256-based JIT prover. With the high-S fix applied (normalize_s +
//     flip recovery point y-parity), the prover also stores 1 and the state roots match →
//     ValidateResult returns success=true.
func TestEcrecoverHighSDivergence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.RequireScheme(t, rawdb.HashScheme)
	builder.nodeConfig.BlockValidator.Enable = false
	builder.nodeConfig.BatchPoster.Enable = true
	builder.nodeConfig.ParentChainReader.Enable = true
	builder.nodeConfig.ParentChainReader.OldHeaderTimeout = 10 * time.Minute

	valConf := valnode.TestValidationConfig
	valConf.UseJit = true
	_, valStack := createTestValidationNode(t, ctx, &valConf)
	configByValidationNode(builder.nodeConfig, valStack)

	cleanup := builder.Build(t)
	defer cleanup()

	sbv := builder.L2.ConsensusNode.StatelessBlockValidator
	if sbv == nil {
		t.Fatal("StatelessBlockValidator is nil — validation node URL not wired up correctly")
	}

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	_, deployTx, contract, err := localgen.DeployHighSEcrecover(&auth, builder.L2.Client)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(deployTx)
	Require(t, err)

	input := buildHighSInput()
	auth.GasLimit = 1e9
	triggerTx, err := contract.Fallback(&auth, input[:])
	Require(t, err)
	triggerReceipt, err := builder.L2.EnsureTxSucceeded(triggerTx)
	Require(t, err)

	// Part 1: confirm native execution accepted the high-S signature.
	result, err := contract.Result(nil)
	Require(t, err)
	if result.Uint64() != 1 {
		t.Fatalf("result = %d: expected 1 (native accepted high-S); either the bug is fixed or the test input is wrong", result.Uint64())
	}

	// Part 2: confirm JIT re-execution agrees with native (fix: k256 now accepts high-S).
	msgIdx := arbutil.MessageIndex(triggerReceipt.BlockNumber.Uint64())
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)
	doUntil(t, 250*time.Millisecond, 150, func() bool {
		_, found, err := builder.L2.ConsensusNode.GetParentChainDataSource().FindInboxBatchContainingMessage(msgIdx)
		Require(t, err)
		return found
	})

	nativeResult, err := builder.L2.ExecNode.ResultAtMessageIndex(msgIdx).Await(ctx)
	Require(t, err)
	_, nativeEndPos, err := sbv.GlobalStatePositionsAtCount(msgIdx + 1)
	Require(t, err)
	nativeGS := staker.BuildGlobalState(*nativeResult, nativeEndPos)

	moduleRoot := sbv.GetLatestWasmModuleRoot()
	success, jitGS, err := sbv.ValidateResult(ctx, msgIdx, true, moduleRoot)
	Require(t, err)

	t.Logf("native BlockHash: %v", nativeGS.BlockHash)
	t.Logf("JIT    BlockHash: %v", jitGS.BlockHash)

	if !success {
		t.Fatalf("JIT and native state roots diverge — high-S fix not effective: native=%v jit=%v", nativeGS.BlockHash, jitGS.BlockHash)
	}
}
