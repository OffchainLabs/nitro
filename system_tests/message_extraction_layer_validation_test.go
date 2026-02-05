package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/validator"
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
	entry, _, err := melValidator.CreateNextValidationEntry(ctx, startBlock, uint64(extractedMsgCountToValidate))
	Require(t, err)
	doneEntry, err := melValidator.SendValidationEntry(ctx, entry)
	Require(t, err)
	if !doneEntry.Success {
		t.Fatal("failed mel validation")
	}
	Require(t, melValidator.AdvanceValidations(ctx, doneEntry))

	// Check if we have executed all messages.

	// Create block validator.
	errChan := make(chan error, 1)
	cfgFetcher := func() *staker.BlockValidatorConfig {
		cfg := builder.nodeConfig.BlockValidator
		cfg.Enable = true
		cfg.EnableMEL = true
		return &cfg
	}
	blockValidator, err := staker.NewBlockValidator(
		builder.L2.ConsensusNode.StatelessBlockValidator,
		builder.L2.ConsensusNode.InboxTracker,
		builder.L2.ConsensusNode.TxStreamer,
		cfgFetcher,
		errChan,
		builder.L2.ConsensusNode.MessageExtractor,
		melValidator,
	)
	Require(t, err)

	// Create an entry.
	entryCreator := staker.NewMELEnabledValidationEntryCreator(
		melValidator, builder.L2.ConsensusNode.TxStreamer, builder.L2.ConsensusNode.MessageExtractor,
	)
	Require(t, err)

	// Use the block recorder over the entry.
	blockValidatorEntry, created, err := entryCreator.CreateBlockValidationEntry(ctx, entry.End, arbutil.MessageIndex(entry.End.PosInBatch))
	Require(t, err)
	if !created {
		t.Fatal("validation entry not created")
	}
	sbv := builder.L2.ConsensusNode.StatelessBlockValidator
	Require(t, sbv.ValidationEntryRecord(ctx, blockValidatorEntry))

	// Create a machine loader.
	spawner, err := server_arb.NewArbitratorSpawner(nil, locator)
	Require(t, err)

	// Launch an execution run with the entry and await GetLastStep() of execution run.
	execRun := spawner.CreateExecutionRun(locator.LatestWasmModuleRoot(), &validator.ValidationInput{}, true)

	// Verify that the final global state matches the block hash of the native node at that message.
	createdRun, err := execRun.Await(ctx)
	Require(t, err)
	lastStep, err := createdRun.GetLastStep().Await(ctx)
	Require(t, err)
	_ = lastStep
	// TODO: Verify the block hash against the native execution of the node.
	// Check the global MEL fields remain the same, but that we changed the blockhash and pos in batch.
}
