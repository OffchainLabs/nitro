package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"

	melextraction "github.com/offchainlabs/nitro/arbnode/mel/extraction"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	melreplay "github.com/offchainlabs/nitro/mel-replay"
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
	cleanup := builder.Build(t)
	defer cleanup()

	// Post a blob batch with a bunch of txs
	startBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()
	initialBatchCount := GetBatchCount(t, builder)
	var txs types.Transactions
	for i := 0; i < 20; i++ {
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

	// MEL Validator: create validation entry
	blobReaderRegistry := daprovider.NewDAProviderRegistry()
	Require(t, blobReaderRegistry.SetupBlobReader(daprovider.NewReaderForBlobReader(builder.L1.L1BlobReader)))
	config := func() *staker.MELValidatorConfig { return &staker.DefaultMELValidatorConfig }
	Require(t, config().Validate())
	melValidator, err := staker.NewMELValidator(config, builder.L2.ConsensusNode.ConsensusDB, builder.L1.Client, builder.L1.Stack, builder.L2.ConsensusNode.MessageExtractor, blobReaderRegistry, common.HexToHash("0123"))
	Require(t, err)
	extractedMsgCount, err := builder.L2.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)
	entry, _, err := melValidator.CreateNextValidationEntry(ctx, startBlock, uint64(extractedMsgCount))
	Require(t, err)

	locator, err := server_common.NewMachineLocator(builder.valnodeConfig.Wasm.RootPath, server_common.WithMELEnabled())
	Require(t, err)
	arbConfigFetcher := func() *server_arb.ArbitratorSpawnerConfig {
		defaultCfg := server_arb.DefaultArbitratorSpawnerConfig
		defaultCfg.MachineConfig = server_arb.ArbitratorMachineConfig{
			WavmBinaryPath:       "unified_machine.wavm.br",
			UntilHostIoStatePath: "unified-until-host-io-state.bin",
		}
		return &defaultCfg
	}
	arbSpawner, err := server_arb.NewArbitratorSpawner(locator, arbConfigFetcher)
	Require(t, err)
	Require(t, arbSpawner.Start(ctx))
	execRunPromise := arbSpawner.CreateExecutionRun(
		locator.LatestWasmModuleRoot(),
		&validator.ValidationInput{
			Id:                      1,
			HasDelayedMsg:           entry.HasDelayedMsg,
			DelayedMsgNr:            entry.DelayedMsgNr,
			Preimages:               entry.Preimages,
			UserWasms:               make(map[rawdb.WasmTarget]map[common.Hash][]byte),
			BatchInfo:               entry.BatchInfo,
			DelayedMsg:              entry.DelayedMsg,
			StartState:              entry.Start,
			DebugChain:              false,
			EndParentChainBlockHash: entry.EndParentChainBlockHash,
		},
		true, /* use bold machine */
	)
	result, err := execRunPromise.Await(ctx)
	Require(t, err)
	finalResult, err := result.GetLastStep().Await(ctx)
	Require(t, err)
	if finalResult.GlobalState.MELStateHash != entry.End.MELStateHash {
		t.Fatalf("Expected final mel state hash %s, got %s", entry.End.MELStateHash, finalResult.GlobalState.MELStateHash)
	}
}

func TestMELValidator_Recording_Preimages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.L2Info.GenerateAccount("User2")
	builder.nodeConfig.BatchPoster.Post4844Blobs = true
	builder.nodeConfig.BatchPoster.IgnoreBlobPrice = true
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour     // set high max-delay so we can test the delay buffer
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour // set a high poll interval to avoid continuous polling
	cleanup := builder.Build(t)
	defer cleanup()

	// Post a blob batch with a bunch of txs
	startBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()
	initialBatchCount := GetBatchCount(t, builder)
	var txs types.Transactions
	for i := 0; i < 20; i++ {
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

	// MEL Validator: create validation entry
	blobReaderRegistry := daprovider.NewDAProviderRegistry()
	Require(t, blobReaderRegistry.SetupBlobReader(daprovider.NewReaderForBlobReader(builder.L1.L1BlobReader)))
	config := func() *staker.MELValidatorConfig { return &staker.DefaultMELValidatorConfig }
	Require(t, config().Validate())
	melValidator, err := staker.NewMELValidator(config, builder.L2.ConsensusNode.ConsensusDB, builder.L1.Client, builder.L1.Stack, builder.L2.ConsensusNode.MessageExtractor, blobReaderRegistry, common.MaxHash)
	Require(t, err)
	extractedMsgCount, err := builder.L2.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)
	entry, _, err := melValidator.CreateNextValidationEntry(ctx, startBlock, uint64(extractedMsgCount))
	Require(t, err)

	// Represents running of MEL validation using preimages in wasm mode. TODO: remove this once we have validation wired
	preimageResolver := melreplay.NewTypeBasedPreimageResolver(
		arbutil.Keccak256PreimageType,
		entry.Preimages,
	)
	state, err := builder.L2.ConsensusNode.MessageExtractor.GetState(ctx, startBlock)
	Require(t, err)
	preimagesBasedDelayedDb := melreplay.NewDelayedMessageDatabase(preimageResolver)
	preimagesBasedDapReaders := daprovider.NewDAProviderRegistry()
	blobReader := &blobPreimageReader{
		melreplay.NewTypeBasedPreimageResolver(
			arbutil.EthVersionedHashPreimageType,
			entry.Preimages,
		),
	}
	Require(t, preimagesBasedDapReaders.SetupBlobReader(daprovider.NewReaderForBlobReader(blobReader)))
	for state.MsgCount < uint64(extractedMsgCount) {
		header, err := builder.L1.Client.HeaderByNumber(ctx, new(big.Int).SetUint64(state.ParentChainBlockNumber+1))
		Require(t, err)
		preimagesBasedTxsFetcher := melreplay.NewTransactionFetcher(header, preimageResolver)
		preimagesBasedLogsFetcher := melreplay.NewLogsFetcher(header, preimageResolver)
		postState, _, _, _, err := melextraction.ExtractMessages(ctx, state, header, preimagesBasedDapReaders, preimagesBasedDelayedDb, preimagesBasedTxsFetcher, preimagesBasedLogsFetcher, nil)
		Require(t, err)
		wantState, err := builder.L2.ConsensusNode.MessageExtractor.GetState(ctx, state.ParentChainBlockNumber+1)
		Require(t, err)
		if postState.Hash() != wantState.Hash() {
			t.Fatalf("MEL state mismatch")
		}
		state = postState
	}
}

type blobPreimageReader struct {
	preimageResolver melreplay.PreimageResolver
}

func (b *blobPreimageReader) Initialize(ctx context.Context) error { return nil }

func (b *blobPreimageReader) GetBlobs(
	ctx context.Context,
	batchBlockHash common.Hash,
	versionedHashes []common.Hash,
) ([]kzg4844.Blob, error) {
	var blobs []kzg4844.Blob
	for _, h := range versionedHashes {
		var blob kzg4844.Blob
		preimage, err := b.preimageResolver.ResolveTypedPreimage(arbutil.EthVersionedHashPreimageType, h)
		if err != nil {
			return nil, err
		}
		if len(preimage) != len(blob) {
			return nil, fmt.Errorf("for blob %v got back preimage of length %v but expected blob length %v", h, len(preimage), len(blob))
		}
		copy(blob[:], preimage)
		blobs = append(blobs, blob)
	}
	return blobs, nil
}
