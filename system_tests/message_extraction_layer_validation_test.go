package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/validator/server_arb"
	"github.com/offchainlabs/nitro/validator/server_common"
)

func TestMELValidator_Recording_RunsUnifiedReplayBinary(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.L2Info.GenerateAccount("User2")
	builder.nodeConfig.BatchPoster.Post4844Blobs = true
	builder.nodeConfig.BatchPoster.IgnoreBlobPrice = true
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour     // set high max-delay so we can test the delay buffer
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour // set a high poll interval to avoid continuous polling
	builder.nodeConfig.MELValidator.Enable = true
	builder.nodeConfig.BlockValidator.Enable = false
	cleanup := builder.Build(t)
	defer cleanup()

	// Post a blob batch with a bunch of txs
	startBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()
	initialBatchCount := GetBatchCount(t, builder)
	var txs types.Transactions
	for range 20 {
		tx, _ := builder.L2.TransferBalance(t, "Faucet", "User2", big.NewInt(1e12), builder.L2Info)
		txs = append(txs, tx)
	}
	builder.nodeConfig.BatchPoster.MaxDelay = 0
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
	_, err = builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
	Require(t, err)
	for _, tx := range txs {
		_, err := testClientB.EnsureTxSucceeded(tx)
		Require(t, err, "tx not found on second node")
	}
	CheckBatchCount(t, builder, initialBatchCount+1)

	// Post delayed messages
	forceDelayedBatchPosting(t, ctx, builder, testClientB, 10, 0)

	// MEL Validator
	extractedMsgCountToValidate, err := builder.L2.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)
	locator, err := server_common.NewMachineLocator(builder.valnodeConfig.Wasm.RootPath, server_common.WithMELEnabled()) // to get unified-module-root
	Require(t, err)
	blobReaderRegistry := daprovider.NewDAProviderRegistry()
	Require(t, blobReaderRegistry.SetupBlobReader(daprovider.NewReaderForBlobReader(builder.L1.L1BlobReader)))

	config := func() *staker.MELValidatorConfig { return &builder.nodeConfig.MELValidator }
	melValidator, err := staker.NewMELValidator(config, builder.L2.ConsensusNode.ConsensusDB, builder.L1.Client, builder.L1.Stack, builder.L2.ConsensusNode.MessageExtractor, blobReaderRegistry, locator.LatestWasmModuleRoot())
	Require(t, err)
	Require(t, melValidator.Initialize(ctx))
	entry, endMELState, err := melValidator.CreateNextValidationEntry(ctx, startBlock, uint64(extractedMsgCountToValidate))
	Require(t, err)
	doneEntry, err := melValidator.SendValidationEntry(ctx, entry)
	Require(t, err)
	if !doneEntry.Success {
		t.Fatal("failed mel validation")
	}
	Require(t, melValidator.AdvanceValidations(ctx, doneEntry))

	mockMElV := &mockMELValidator{
		realValidator:        melValidator,
		latestValidatedState: endMELState,
	}
	entryCreator := staker.NewMELEnabledValidationEntryCreator(
		mockMElV, builder.L2.ConsensusNode.TxStreamer, builder.L2.ConsensusNode.MessageExtractor,
	)
	Require(t, err)

	// Create a machine loader for the unified replay binary.
	arbSpawnerCfgFetcher := func() *server_arb.ArbitratorSpawnerConfig {
		cfg := server_arb.DefaultArbitratorSpawnerConfig
		cfg.MachineConfig.UntilHostIoStatePath = "unified-until-host-io-state.bin"
		cfg.MachineConfig.WavmBinaryPath = "unified_machine.wavm.br"
		return &cfg
	}
	spawner, err := server_arb.NewArbitratorSpawner(locator, arbSpawnerCfgFetcher)
	Require(t, err)
	Require(t, spawner.Start(ctx))

	sbv := builder.L2.ConsensusNode.StatelessBlockValidator
	computedGlobalState := doneEntry.End

	// While the computed global state's msg hash is non-empty, we will run MEL validation
	// until we validate all the blocks corresponding to messages extracted by MEL.
	// This is because when MEL extraction runs, it may extract N new messages. Then, we validate block production
	// for messages 0 to N-1. At that point, a new message extraction must occur to fetch brand new messages beyond that.
	for computedGlobalState.MELMsgHash != (common.Hash{}) {
		blockValidatorEntry, created, err := entryCreator.CreateBlockValidationEntry(
			ctx,
			computedGlobalState,
			arbutil.MessageIndex(computedGlobalState.PosInBatch),
		)
		Require(t, err)
		if !created {
			t.Fatal("validation entry not created")
		}

		l2Header, err := builder.L2.ExecNode.Backend.APIBackend().HeaderByHash(ctx, blockValidatorEntry.End.BlockHash)
		Require(t, err)
		prevL2Header, err := builder.L2.ExecNode.Backend.APIBackend().HeaderByHash(ctx, l2Header.ParentHash)
		Require(t, err)

		// We run recording over the execution of the block validator entry.
		err = sbv.ValidationEntryRecord(ctx, blockValidatorEntry)
		Require(t, err)

		// We add the previous block header to the preimages map.
		rlpEncodedHeader, err := rlp.EncodeToBytes(prevL2Header)
		Require(t, err)
		blockValidatorEntry.Preimages[arbutil.Keccak256PreimageType][l2Header.ParentHash] = rlpEncodedHeader

		// Launch an execution run with the entry.
		input, err := blockValidatorEntry.ToInput(spawner.StylusArchs())
		Require(t, err)
		execRun := spawner.CreateExecutionRun(locator.LatestWasmModuleRoot(), input, true /* use bold machinery */)

		// Verify the final global state matches the block hash of the native execution of that message.
		createdRun, err := execRun.Await(ctx)
		Require(t, err)
		lastStep, err := createdRun.GetLastStep().Await(ctx)
		Require(t, err)
		if lastStep.GlobalState.BlockHash != blockValidatorEntry.End.BlockHash {
			t.Fatalf("Expected to compute %s block hash but computed %s", blockValidatorEntry.End.BlockHash, lastStep.GlobalState.BlockHash)
		}
		t.Logf("Validated block execution of message index %+v\n", lastStep.GlobalState)

		// Update the computed global state to the one just computed by Arbitrator.
		computedGlobalState = lastStep.GlobalState
	}

	// Finally, we want to verify that the ending global state has executed all messages extracted by MEL
	// and that it also contains the proper MEL state hash field corresponding to the extraction of such messages.
	// This puts everything together and verifies we can validate both extraction and execution correctly, in lock-step.
	if computedGlobalState.MELMsgHash != (common.Hash{}) {
		t.Fatalf("Expected to compute MEL msg hash %s but computed %s", common.Hash{}, computedGlobalState.MELMsgHash)
	}
	if computedGlobalState.PosInBatch != endMELState.MsgCount {
		t.Fatalf("Expected to validate execution of %d messages, but got %d", endMELState.MsgCount, computedGlobalState.PosInBatch)
	}
}

type mockMELValidator struct {
	realValidator        *staker.MELValidator
	latestValidatedState *mel.State
}

func (m *mockMELValidator) LatestValidatedMELState(ctx context.Context) (*mel.State, error) {
	return m.latestValidatedState, nil
}

func (m *mockMELValidator) FetchMsgPreimages(ctx context.Context, l2BlockNum, parentChainBlockNumber uint64) (daprovider.PreimagesMap, error) {
	return m.realValidator.FetchMsgPreimages(ctx, l2BlockNum, parentChainBlockNumber)
}

func (m *mockMELValidator) ClearValidatedMsgPreimages(lastValidatedL2BlockParentChainBlockNumber uint64) {

}
