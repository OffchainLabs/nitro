// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/client"
	"github.com/offchainlabs/nitro/validator/server_api"
)

func TestValidationInputsAtWithWasmTarget(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, false, func(builder *NodeBuilder) {
		builder.WithLegacyBlockRecorder()
	})
	ctx := builder.ctx
	defer cleanup()

	auth.GasLimit = 32000000
	auth.Value = oneEth

	programAddress, moduleHash, wasmExpected := deployAndActivateStorageProgram(t, ctx, auth, builder)

	_, inboxPos := sendStorageWrite(t, ctx, builder, "Owner", programAddress)
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
	testValidationInputsAtExecutesInValidationWorker(t)
}

func TestValidationInputsAtChainTipStorageCacheFlushOutOfGas(t *testing.T) {
	builder, auth, cleanup := setupChainTipValidationInputsTest(t)
	ctx := builder.ctx
	defer cleanup()

	inboxPos := storageCacheFlushOutOfGasInboxPos(t, ctx, builder, auth)
	validateResultAt(t, ctx, builder, inboxPos)
	_, validationInput := validationInputsAtFromTip(t, ctx, builder, inboxPos)
	preimages := validationInput.Preimages[arbutil.Keccak256PreimageType]
	if len(preimages) == 0 {
		t.Fatal("expected validation input to contain keccak preimages")
	}
}

func TestValidationInputsAtServesMultipleChainTipBlocks(t *testing.T) {
	builder, auth, cleanup := setupProgramTestWithScheme(t, false, "", func(builder *NodeBuilder) {
		builder.WithChainTipBlockRecorder()
	})
	ctx := builder.ctx
	defer cleanup()

	auth.GasLimit = 32000000
	auth.Value = oneEth

	programAddress, _, _ := deployAndActivateStorageProgram(t, ctx, auth, builder)

	positions := make([]arbutil.MessageIndex, 0, 3)
	for i := 0; i < 3; i++ {
		_, inboxPos := sendStorageWrite(t, ctx, builder, "Owner", programAddress)
		positions = append(positions, inboxPos)
	}

	valClient, moduleRoot := newValidationClientForBuilder(t, ctx, builder)
	defer valClient.Stop()

	for _, inboxPos := range positions {
		waitForBatchContainingMessage(t, builder.L2.ConsensusNode, inboxPos, 10*time.Second, 250*time.Millisecond)
		inputJson, validationInput := validationInputsAtFromTip(t, ctx, builder, inboxPos)
		if inputJson.ExpectedEndState == nil {
			t.Fatalf("expected validation input json for pos %d to include expected end state", inboxPos)
		}
		if len(validationInput.Preimages[arbutil.Keccak256PreimageType]) == 0 {
			t.Fatalf("expected validation input for pos %d to contain recorded keccak preimages", inboxPos)
		}
		runValidationInput(t, ctx, valClient, validationInput, moduleRoot, inputJson.ExpectedEndState)
	}
}

func testValidationInputsAtExecutesInValidationWorker(t *testing.T) {
	builder, auth, cleanup := setupProgramTestWithScheme(t, false, "", func(builder *NodeBuilder) {
		builder.WithChainTipBlockRecorder()
	})
	ctx := builder.ctx
	defer cleanup()

	auth.GasLimit = 32000000
	auth.Value = oneEth

	programAddress, moduleHash, _ := deployAndActivateStorageProgram(t, ctx, auth, builder)

	recorderUsers := seedValidationRecordingTrieShape(t, ctx, builder, programAddress)

	receipt, inboxPos := sendStorageWrite(t, ctx, builder, recorderUsers[0], programAddress)
	waitForBatchContainingMessage(t, builder.L2.ConsensusNode, inboxPos, 10*time.Second, 250*time.Millisecond)

	inputJson, validationInput := validationInputsAtFromTip(t, ctx, builder, inboxPos)
	preimages := validationInput.Preimages[arbutil.Keccak256PreimageType]
	if len(preimages) == 0 {
		t.Fatal("expected validation input to contain recorded keccak preimages")
	}
	if receipt.BlockNumber.Uint64() < 1 {
		t.Fatal("expected storage transaction after block 0")
	}
	assertHeaderPreimages(t, builder, receipt.BlockNumber.Uint64()-1, receipt.BlockNumber.Uint64(), preimages)
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
	storageValidationInput := validationInput
	storageExpectedEndState := inputJson.ExpectedEndState

	arbSysABI, err := precompilesgen.ArbSysMetaData.GetAbi()
	Require(t, err)
	requestedBlockNumber := receipt.BlockNumber.Uint64() - 1
	arbBlockHashData, err := arbSysABI.Pack("arbBlockHash", new(big.Int).SetUint64(requestedBlockNumber))
	Require(t, err)
	tx := builder.L2Info.PrepareTxTo("Owner", &types.ArbSysAddress, builder.L2Info.TransferGas, nil, arbBlockHashData)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	blockHashReceipt, err := EnsureTxSucceeded(ctx, builder.L2.Client, tx)
	Require(t, err)

	blockHashInboxPos := blockNumberToMessageIndex(t, ctx, builder, blockHashReceipt.BlockNumber.Uint64())
	waitForBatchContainingMessage(t, builder.L2.ConsensusNode, blockHashInboxPos, 10*time.Second, 250*time.Millisecond)
	blockHashInputJson, blockHashValidationInput := validationInputsAtFromTip(t, ctx, builder, blockHashInboxPos)
	if blockHashInputJson.ExpectedEndState == nil {
		t.Fatal("expected blockhash validation input json to include expected end state")
	}

	// Delayed L1 messages take a different path into block production.
	builder.L2Info.GenerateAccount("DelayedUser")
	delayedTx := builder.L2Info.PrepareTx("Owner", "DelayedUser", builder.L2Info.TransferGas, common.Big1, nil)
	delayedReceipt := SendSignedTxViaL1(t, ctx, builder.L1Info, builder.L1.Client, builder.L2.Client, delayedTx)
	delayedInboxPos := blockNumberToMessageIndex(t, ctx, builder, delayedReceipt.BlockNumber.Uint64())
	waitForBatchContainingMessage(t, builder.L2.ConsensusNode, delayedInboxPos, 10*time.Second, 250*time.Millisecond)
	delayedInputJson, delayedValidationInput := validationInputsAtFromTip(t, ctx, builder, delayedInboxPos)
	if delayedInputJson.ExpectedEndState == nil {
		t.Fatal("expected delayed message validation input json to include expected end state")
	}

	valClient, moduleRoot := newValidationClientForBuilder(t, ctx, builder)
	defer valClient.Stop()

	runValidationInput(t, ctx, valClient, storageValidationInput, moduleRoot, storageExpectedEndState)
	runValidationInput(t, ctx, valClient, blockHashValidationInput, moduleRoot, blockHashInputJson.ExpectedEndState)
	runValidationInput(t, ctx, valClient, delayedValidationInput, moduleRoot, delayedInputJson.ExpectedEndState)
}

func deployAndActivateStorageProgram(t *testing.T, ctx context.Context, auth bind.TransactOpts, builder *NodeBuilder) (common.Address, common.Hash, []byte) {
	t.Helper()

	l2client := builder.L2.Client
	wasmToDeploy, wasmExpected := readWasmFile(t, rustFile("storage"))
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
	return programAddress, l.ModuleHash, wasmExpected
}

func sendStorageWrite(t *testing.T, ctx context.Context, builder *NodeBuilder, from string, programAddress common.Address) (*types.Receipt, arbutil.MessageIndex) {
	t.Helper()

	tx := builder.L2Info.PrepareTxTo(from, &programAddress, builder.L2Info.TransferGas, nil, argsForStorageWrite(testhelpers.RandomHash(), testhelpers.RandomHash()))
	err := builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, builder.L2.Client, tx)
	Require(t, err)
	return receipt, blockNumberToMessageIndex(t, ctx, builder, receipt.BlockNumber.Uint64())
}

func newValidationClientForBuilder(t *testing.T, ctx context.Context, builder *NodeBuilder) (*client.ValidationClient, common.Hash) {
	t.Helper()

	validationConfig := builder.nodeConfig.BlockValidator.ValidationServerConfigs[0]
	valClient := client.NewValidationClient(StaticFetcherFrom(t, &validationConfig), nil)
	Require(t, valClient.Start(ctx))
	moduleRoot := builder.L2.ConsensusNode.StatelessBlockValidator.GetLatestWasmModuleRoot()
	return valClient, moduleRoot
}

func seedValidationRecordingTrieShape(t *testing.T, ctx context.Context, builder *NodeBuilder, programAddress common.Address) []string {
	t.Helper()

	l2info := builder.L2Info
	l2client := builder.L2.Client
	recorderUsers := []string{"RecorderUser1", "RecorderUser2", "RecorderUser3"}
	for _, user := range recorderUsers {
		l2info.GenerateAccount(user)
		tx := l2info.PrepareTx("Owner", user, l2info.TransferGas, oneEth, nil)
		err := l2client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
	}

	storageWriters := append([]string{"Owner"}, recorderUsers...)
	for _, writer := range storageWriters {
		tx := l2info.PrepareTxTo(writer, &programAddress, l2info.TransferGas, nil, argsForStorageWrite(testhelpers.RandomHash(), testhelpers.RandomHash()))
		err := l2client.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
	}
	return recorderUsers
}

func setupChainTipValidationInputsTest(t *testing.T) (*NodeBuilder, bind.TransactOpts, func()) {
	t.Helper()
	return setupProgramTest(t, false, func(builder *NodeBuilder) {
		builder.WithChainTipBlockRecorder()
	})
}

func storageCacheFlushOutOfGasInboxPos(t *testing.T, ctx context.Context, builder *NodeBuilder, auth bind.TransactOpts) arbutil.MessageIndex {
	t.Helper()
	testClient2ndNode, cleanup2ndNode := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: arbnode.ConfigDefaultL1NonSequencerTest()})
	t.Cleanup(cleanup2ndNode)

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	ensure := func(tx *types.Transaction, err error) {
		t.Helper()
		Require(t, err)
		_, err = builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}
	ensure(arbOwner.SetL1BaseFeeEstimateInertia(&auth, 0))
	ensure(arbOwner.SetL1PricingEquilibrationUnits(&auth, big.NewInt(0)))
	ensure(arbOwner.SetL1PricingInertia(&auth, 0))
	ensure(arbOwner.SetL1PricingRewardRate(&auth, 0))
	ensure(arbOwner.SetL1PricePerUnit(&auth, big.NewInt(0)))
	ensure(arbOwner.SetPerBatchGasCharge(&auth, 0))
	ensure(arbOwner.SetAmortizedCostCapBips(&auth, 0))

	multicall := deployWasm(t, ctx, auth, builder.L2.Client, rustFile("multicall"))
	argsMulticall := func(numberOfStores int) []byte {
		args := multicallEmptyArgs()
		for i := 0; i < numberOfStores; i++ {
			args = multicallAppendStore(args, testhelpers.RandomHash(), testhelpers.RandomHash(), false, true)
		}
		return args
	}

	tx := builder.L2Info.PrepareTxTo("Owner", &multicall, 2_210_000, nil, argsMulticall(50))
	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	receipt, err := WaitForTx(ctx, testClient2ndNode.Client, tx.Hash(), time.Second*15)
	Require(t, err)
	if receipt.GasUsedForL1 != 0 {
		t.Fatalf("expected 0 gas used, got %d", receipt.GasUsedForL1)
	}

	tx = builder.L2Info.PrepareTxTo("Owner", &multicall, 2_210_000, nil, argsMulticall(200))
	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	receipt, err = builder.L2.EnsureTxSucceeded(tx)
	if err == nil {
		t.Fatal("expected storage cache flush transaction to fail out of gas")
	}
	if receipt == nil || receipt.BlockNumber == nil {
		t.Fatal("expected failed transaction receipt")
	}
	receipt, err = WaitForTx(ctx, testClient2ndNode.Client, tx.Hash(), time.Second*15)
	Require(t, err)
	inboxPos := arbutil.MessageIndex(receipt.BlockNumber.Uint64())
	waitForBatchContainingMessage(t, builder.L2.ConsensusNode, inboxPos, 10*time.Second, 250*time.Millisecond)
	return inboxPos
}

func validationInputsAtFromTip(t *testing.T, ctx context.Context, builder *NodeBuilder, inboxPos arbutil.MessageIndex) (server_api.InputJSON, *validator.ValidationInput) {
	t.Helper()

	var inputJson server_api.InputJSON
	chainTipRecorder, ok := builder.L2.ExecNode.Recorder.(*gethexec.ChainTipBlockRecorder)
	if !ok {
		Fatal(t, "expected chain-tip block recorder")
	}
	servedTipRecordingsBefore := chainTipRecorder.ServedTipRecordings()
	retryUntilFound(t, ctx, 40, 250*time.Millisecond, "ValidationInputsAt", "batch not found on L1", func() error {
		var err error
		inputJson, err = builder.L2.ConsensusNode.StatelessBlockValidator.ValidationInputsAt(ctx, inboxPos, rawdb.LocalTarget(), rawdb.TargetWavm)
		return err
	})
	if chainTipRecorder.ServedTipRecordings() == servedTipRecordingsBefore {
		t.Fatal("expected ValidationInputsAt to serve a chain-tip recording")
	}
	validationInput, err := server_api.ValidationInputFromJson(&inputJson)
	Require(t, err)
	return inputJson, validationInput
}

func validateResultAt(t *testing.T, ctx context.Context, builder *NodeBuilder, inboxPos arbutil.MessageIndex) {
	t.Helper()
	wasmModuleRoot := currentRootModule(t)
	retryUntilFound(t, ctx, 40, 250*time.Millisecond, "ValidateResult", "batch not found on L1", func() error {
		correct, _, err := builder.L2.ConsensusNode.StatelessBlockValidator.ValidateResult(ctx, inboxPos, false, wasmModuleRoot)
		if err != nil {
			return err
		}
		if !correct {
			t.Fatal("validation result mismatch")
		}
		return nil
	})
}

func blockNumberToMessageIndex(t *testing.T, ctx context.Context, builder *NodeBuilder, blockNumber uint64) arbutil.MessageIndex {
	t.Helper()

	pos, err := builder.L2.ExecNode.BlockNumberToMessageIndex(blockNumber).Await(ctx)
	Require(t, err)
	return pos
}

func runValidationInput(
	t *testing.T,
	ctx context.Context,
	valClient *client.ValidationClient,
	validationInput *validator.ValidationInput,
	moduleRoot common.Hash,
	expectedEndState *validator.GoGlobalState,
) {
	t.Helper()

	run := valClient.Launch(validationInput, moduleRoot)
	defer run.Cancel()
	actualEndState, err := run.Await(ctx)
	Require(t, err)
	if actualEndState != *expectedEndState {
		t.Fatalf("validation result mismatch: got %+v, want %+v", actualEndState, *expectedEndState)
	}
}

func assertHeaderPreimages(t *testing.T, builder *NodeBuilder, firstHeaderNumber uint64, blockNumber uint64, preimages map[common.Hash][]byte) {
	t.Helper()
	if blockNumber == 0 || firstHeaderNumber >= blockNumber {
		return
	}
	bc := builder.L2.ExecNode.Backend.ArbInterface().BlockChain()
	for headerNum := firstHeaderNumber; headerNum < blockNumber; headerNum++ {
		header := bc.GetHeaderByNumber(headerNum)
		if header == nil {
			t.Fatalf("expected header %d to exist", headerNum)
		}
		encodedHeader, err := rlp.EncodeToBytes(header)
		Require(t, err)
		preimage, ok := preimages[header.Hash()]
		if !ok {
			t.Fatalf("expected validation input to contain header preimage for block %d (%s)", headerNum, header.Hash())
		}
		if !bytes.Equal(preimage, encodedHeader) {
			t.Fatalf("header preimage mismatch for block %d (%s)", headerNum, header.Hash())
		}
	}
}
