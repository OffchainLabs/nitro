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
	validationclient "github.com/offchainlabs/nitro/validator/client"
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
	waitForBatchContainingMessage(t, builder.L2.ConsensusNode, inboxPos, 10*time.Second, 250*time.Millisecond)

	// Retry ValidationInputsAt because the batch may be tracked locally but
	// not yet confirmed on L1 ("batch not found on L1 yet").
	var inputJson server_api.InputJSON
	retryUntilFound(t, ctx, 40, 250*time.Millisecond, "ValidationInputsAt", "batch not found on L1", func() error {
		var err error
		inputJson, err = builder.L2.ConsensusNode.StatelessBlockValidator.ValidationInputsAt(ctx, inboxPos, rawdb.LocalTarget(), rawdb.TargetWasm)
		return err
	})
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

func TestValidationInputsAtExecutesInValidationWorker(t *testing.T) {
	for _, stateScheme := range []string{rawdb.HashScheme, rawdb.PathScheme} {
		stateScheme := stateScheme
		t.Run(stateScheme, func(t *testing.T) {
			testValidationInputsAtExecutesInValidationWorker(t, stateScheme)
		})
	}
}

func testValidationInputsAtExecutesInValidationWorker(t *testing.T, stateScheme string) {
	builder, auth, cleanup := setupProgramTestWithScheme(t, false, stateScheme)
	ctx := builder.ctx
	l2client := builder.L2.Client
	l2info := builder.L2Info
	defer cleanup()

	auth.GasLimit = 32000000
	auth.Value = oneEth

	wasmToDeploy, _ := readWasmFile(t, rustFile("storage"))
	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)
	programAddress := deployContract(t, ctx, auth, l2client, wasmToDeploy)
	tx, err := arbWasm.ActivateProgram(&auth, programAddress)
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)
	if len(receipt.Logs) != 1 {
		Fatal(t, "expected 1 log while activating, got ", len(receipt.Logs))
	}
	l, err := arbWasm.ParseProgramActivated(*receipt.Logs[0])
	Require(t, err)
	moduleHash := l.ModuleHash

	builder.L2.ExecNode.ChainTipRecorder.Enable()
	defer builder.L2.ExecNode.ChainTipRecorder.Disable()
	key := testhelpers.RandomHash()
	value := testhelpers.RandomHash()
	tx = l2info.PrepareTxTo("Owner", &programAddress, l2info.TransferGas, nil, argsForStorageWrite(key, value))
	err = l2client.SendTransaction(ctx, tx)
	Require(t, err)
	receipt, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	inboxPos := arbutil.MessageIndex(receipt.BlockNumber.Uint64())
	waitForBatchContainingMessage(t, builder.L2.ConsensusNode, inboxPos, 10*time.Second, 250*time.Millisecond)

	var inputJson server_api.InputJSON
	servedTipRecordingsBefore := builder.L2.ExecNode.ChainTipRecorder.ServedTipRecordings()
	retryUntilFound(t, ctx, 40, 250*time.Millisecond, "ValidationInputsAt", "batch not found on L1", func() error {
		var err error
		inputJson, err = builder.L2.ConsensusNode.StatelessBlockValidator.ValidationInputsAt(ctx, inboxPos, rawdb.LocalTarget(), rawdb.TargetWavm)
		return err
	})
	if builder.L2.ExecNode.ChainTipRecorder.ServedTipRecordings() == servedTipRecordingsBefore {
		t.Fatal("expected ValidationInputsAt to serve a chain-tip recording")
	}
	validationInput, err := server_api.ValidationInputFromJson(&inputJson)
	Require(t, err)
	preimages := validationInput.Preimages[arbutil.Keccak256PreimageType]
	if len(preimages) == 0 {
		t.Fatal("expected validation input to contain recorded keccak preimages")
	}
	wasmMap, ok := validationInput.UserWasms[rawdb.TargetWavm]
	if !ok {
		t.Fatal("expected TargetWavm in user wasm map")
	}
	wasm, ok := wasmMap[moduleHash]
	if !ok {
		t.Fatal("expected wavm module hash in user wasm map")
	}
	if len(wasm) == 0 {
		t.Fatal("expected non-empty wavm module")
	}
	if inputJson.ExpectedEndState == nil {
		t.Fatal("expected validation input json to include expected end state")
	}

	validationConfig := builder.nodeConfig.BlockValidator.ValidationServerConfigs[0]
	valClient := validationclient.NewValidationClient(StaticFetcherFrom(t, &validationConfig), nil)
	Require(t, valClient.Start(ctx))
	defer valClient.Stop()

	moduleRoot := builder.L2.ConsensusNode.StatelessBlockValidator.GetLatestWasmModuleRoot()
	run := valClient.Launch(validationInput, moduleRoot)
	defer run.Cancel()
	actualEndState, err := run.Await(ctx)
	Require(t, err)
	if actualEndState != *inputJson.ExpectedEndState {
		t.Fatalf("validation result mismatch: got %+v, want %+v", actualEndState, *inputJson.ExpectedEndState)
	}
}
