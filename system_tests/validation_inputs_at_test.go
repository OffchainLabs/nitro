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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/client"
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
	testValidationInputsAtExecutesInValidationWorker(t)
}

func TestValidationInputsAtChainTipIncludesReferencedTrieChildPreimages(t *testing.T) {
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
	assertSomeReferencedTrieChildPreimage(t, preimages)
}

func testValidationInputsAtExecutesInValidationWorker(t *testing.T) {
	builder, auth, cleanup := setupProgramTestWithScheme(t, false, "", func(builder *NodeBuilder) {
		builder.WithChainTipBlockRecorder()
	})
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

	recorderUsers := seedValidationRecordingTrieShape(t, ctx, builder, programAddress)

	key := testhelpers.RandomHash()
	value := testhelpers.RandomHash()
	tx = l2info.PrepareTxTo(recorderUsers[0], &programAddress, l2info.TransferGas, nil, argsForStorageWrite(key, value))
	err = l2client.SendTransaction(ctx, tx)
	Require(t, err)
	receipt, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	inboxPos := arbutil.MessageIndex(receipt.BlockNumber.Uint64())
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
	tx = l2info.PrepareTxTo("Owner", &types.ArbSysAddress, l2info.TransferGas, nil, arbBlockHashData)
	err = l2client.SendTransaction(ctx, tx)
	Require(t, err)
	blockHashReceipt, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	blockHashInboxPos := arbutil.MessageIndex(blockHashReceipt.BlockNumber.Uint64())
	waitForBatchContainingMessage(t, builder.L2.ConsensusNode, blockHashInboxPos, 10*time.Second, 250*time.Millisecond)
	blockHashInputJson, blockHashValidationInput := validationInputsAtFromTip(t, ctx, builder, blockHashInboxPos)
	if blockHashInputJson.ExpectedEndState == nil {
		t.Fatal("expected blockhash validation input json to include expected end state")
	}

	validationConfig := builder.nodeConfig.BlockValidator.ValidationServerConfigs[0]
	valClient := client.NewValidationClient(StaticFetcherFrom(t, &validationConfig), nil)
	Require(t, valClient.Start(ctx))
	defer valClient.Stop()

	moduleRoot := builder.L2.ConsensusNode.StatelessBlockValidator.GetLatestWasmModuleRoot()
	runValidationInput(t, ctx, valClient, storageValidationInput, moduleRoot, storageExpectedEndState)
	runValidationInput(t, ctx, valClient, blockHashValidationInput, moduleRoot, blockHashInputJson.ExpectedEndState)
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

func assertSomeReferencedTrieChildPreimage(t *testing.T, preimages map[common.Hash][]byte) {
	t.Helper()
	references := 0
	recordedReferences := 0
	for hash, preimage := range preimages {
		if crypto.Keccak256Hash(preimage) != hash {
			continue
		}
		var elems []rlp.RawValue
		if err := rlp.DecodeBytes(preimage, &elems); err != nil {
			continue
		}
		switch len(elems) {
		case 17:
			for i := 0; i < 16; i++ {
				countTrieChildPreimageIfHash(preimages, elems[i], &references, &recordedReferences)
			}
		case 2:
			var path []byte
			if err := rlp.DecodeBytes(elems[0], &path); err != nil || len(path) == 0 {
				continue
			}
			if path[0]>>4&2 != 0 {
				continue // leaf node, second element is the value
			}
			countTrieChildPreimageIfHash(preimages, elems[1], &references, &recordedReferences)
		}
	}
	if references == 0 {
		t.Fatal("expected trie node preimages to reference at least one hashed child")
	}
	if recordedReferences == 0 {
		t.Fatal("expected at least one referenced trie child preimage to be recorded")
	}
}

func countTrieChildPreimageIfHash(preimages map[common.Hash][]byte, raw rlp.RawValue, references *int, recordedReferences *int) {
	var child []byte
	if err := rlp.DecodeBytes(raw, &child); err != nil || len(child) != common.HashLength {
		return
	}
	childHash := common.BytesToHash(child)
	*references = *references + 1
	if _, ok := preimages[childHash]; ok {
		*recordedReferences = *recordedReferences + 1
	}
}
