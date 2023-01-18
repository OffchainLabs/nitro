// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
)

func TestKeccakProgram(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rand.Seed(time.Now().UTC().UnixNano())

	chainConfig := params.ArbitrumDevTestChainConfig()
	l2config := arbnode.ConfigDefaultL1Test()
	l2config.BlockValidator.ArbitratorValidator = true
	l2config.BlockValidator.JitValidator = false
	l2config.BatchPoster.Enable = true
	l2config.L1Reader.Enable = true

	l2info, node, l2client, _, _, _, l1stack := createTestNodeOnL1WithConfig(t, ctx, true, l2config, chainConfig, nil)
	defer requireClose(t, l1stack)
	defer node.StopAndWait()

	auth := l2info.GetDefaultTransactOpts("Owner", ctx)
	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, l2client)
	Require(t, err)
	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, l2client)
	Require(t, err)

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}

	// set non-zero costs
	wasmGasPrice := uint64(rand.Intn(200) * rand.Intn(2))
	wasmHostioCost := uint64(rand.Intn(2000))
	ensure(arbDebug.BecomeChainOwner(&auth))
	ensure(arbOwner.SetWasmGasPrice(&auth, wasmGasPrice))
	ensure(arbOwner.SetWasmHostioCost(&auth, wasmHostioCost))
	colors.PrintBlue("Wasm pricing ", wasmGasPrice, wasmHostioCost)

	file := "../arbitrator/stylus/tests/keccak/target/wasm32-unknown-unknown/release/keccak.wasm"
	wasmSource, err := os.ReadFile(file)
	Require(t, err)
	wasm, err := arbcompress.CompressWell(wasmSource)
	Require(t, err)

	toKb := func(data []byte) float64 { return float64(len(data)) / 1024.0 }
	colors.PrintMint(fmt.Sprintf("WASM len %.2fK vs %.2fK", toKb(wasm), toKb(wasmSource)))

	timed := func(message string, lambda func()) {
		t.Helper()
		now := time.Now()
		lambda()
		passed := time.Since(now)
		colors.PrintBlue("Time to ", message, ": ", passed.String())
	}

	programAddress := deployContract(t, ctx, auth, l2client, wasm)
	colors.PrintBlue("program deployed to ", programAddress.Hex())

	timed("compile", func() {
		ensure(arbWasm.CompileProgram(&auth, programAddress))
	})

	preimage := []byte("°º¤ø,¸,ø¤°º¤ø,¸,ø¤°º¤ø,¸ nyan nyan ~=[,,_,,]:3 nyan nyan")
	correct := crypto.Keccak256Hash(preimage)

	args := []byte{0x01} // keccak the preimage once
	args = append(args, preimage...)

	timed("execute", func() {
		result, err := arbWasm.CallProgram(&bind.CallOpts{}, programAddress, args)
		Require(t, err)

		if len(result) != 32 {
			Fail(t, "unexpected return result: ", "result", result)
		}

		hash := common.BytesToHash(result)
		if hash != correct {
			Fail(t, "computed hash mismatch", hash, correct)
		}
		colors.PrintGrey("keccak(x) = ", hash)
	})

	// do a mutating call for proving's sake
	_, tx, mock, err := mocksgen.DeployProgramTest(&auth, l2client)
	ensure(tx, err)
	ensure(mock.CallKeccak(&auth, programAddress, args))

	doUntil(t, 20*time.Millisecond, 50, func() bool {
		batchCount, err := node.InboxTracker.GetBatchCount()
		Require(t, err)
		meta, err := node.InboxTracker.GetBatchMetadata(batchCount - 1)
		Require(t, err)
		messageCount, err := node.ArbInterface.TransactionStreamer().GetMessageCount()
		Require(t, err)
		return meta.MessageCount == messageCount
	})

	blockHeight, err := l2client.BlockNumber(ctx)
	Require(t, err)

	success := true
	for block := uint64(1); block <= blockHeight; block++ {
		header, err := l2client.HeaderByNumber(ctx, arbmath.UintToBig(block))
		Require(t, err)

		now := time.Now()
		correct, err := node.StatelessBlockValidator.ValidateBlock(ctx, header, true, common.Hash{})
		Require(t, err, "block", block)
		passed := time.Since(now).String()
		if correct {
			colors.PrintMint("yay!! we validated block ", block, " in ", passed)
		} else {
			colors.PrintRed("failed to validate block ", block, " in ", passed)
		}
		success = success && correct
	}
	if !success {
		Fail(t)
	}
}
