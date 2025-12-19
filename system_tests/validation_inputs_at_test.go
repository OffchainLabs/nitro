// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"bytes"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/validator/server_api"
)

func TestValidationInputsAtWithWasmTarget(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, false)
	ctx := builder.ctx
	l2client := builder.L2.Client
	l2info := builder.L2Info
	defer cleanup()

	auth.GasLimit = 32000000
	auth.Value = oneEth

	// deploys contract
	wasmToDeploy, wasmExpected := readWasmFile(t, rustFile("storage"))
	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)
	programAddress := deployContract(t, ctx, auth, l2client, wasmToDeploy)
	tx, err := arbWasm.ActivateProgram(&auth, programAddress)
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	// gets module hash
	if len(receipt.Logs) != 1 {
		Fatal(t, "expected 1 log while activating, got ", len(receipt.Logs))
	}
	l, err := arbWasm.ParseProgramActivated(*receipt.Logs[0])
	Require(t, err)
	moduleHash := l.ModuleHash

	// calls contract
	key := testhelpers.RandomHash()
	value := testhelpers.RandomHash()
	tx = l2info.PrepareTxTo("Owner", &programAddress, l2info.TransferGas, nil, argsForStorageWrite(key, value))
	err = l2client.SendTransaction(ctx, tx)
	Require(t, err)
	receipt, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	inboxPos := arbutil.MessageIndex(receipt.BlockNumber.Uint64())
	for range 40 {
		time.Sleep(250 * time.Millisecond)
		batches, err := builder.L2.ConsensusNode.InboxTracker.GetBatchCount()
		Require(t, err)
		haveMessages, err := builder.L2.ConsensusNode.InboxTracker.GetBatchMessageCount(batches - 1)
		Require(t, err)
		if haveMessages >= inboxPos {
			break
		}
	}

	inputJson, err := builder.L2.ConsensusNode.StatelessBlockValidator.ValidationInputsAt(ctx, inboxPos, rawdb.LocalTarget(), rawdb.TargetWasm)
	Require(t, err)
	validationInput, err := server_api.ValidationInputFromJson(&inputJson)
	Require(t, err)
	wasmMap, ok := validationInput.UserWasms[rawdb.TargetWasm]
	if !ok {
		t.Fatal("expected TargetWasm in user wasm map")
	}
	wasm, ok := wasmMap[moduleHash]
	if !ok {
		t.Fatal("expected wasm module hash in user wasm map")
	}
	if !bytes.Equal(wasm, wasmExpected) {
		t.Fatal("wasm does not match expected wasm")
	}
}
