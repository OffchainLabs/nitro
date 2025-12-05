// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !race

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/das"
)

// TestMultiWriterFailure_AnyTrustShutdownFallbackDisabled tests that batch posting fails when AnyTrust shuts down
// and the operator has disabled automatic fallback to calldata. Even though AnyTrust returns
// ErrFallbackRequested (explicit fallback signal), the operator's DisableDapFallbackStoreDataOnChain
// setting takes precedence, causing batch posting to fail as intended.
func TestMultiWriterFailure_AnyTrustShutdownFallbackDisabled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Setup L1 chain and contracts
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.chainConfig = chaininfo.ArbitrumDevTestDASChainConfig()
	builder.parallelise = false

	builder.BuildL1(t)

	// 2. Setup AnyTrust/DAS server
	dasDataDir := t.TempDir()
	dasRpcServer, pubkey, backendConfig, restServer, restServerUrl := startLocalDASServer(
		t, ctx, dasDataDir, builder.L1.Client, builder.addresses.SequencerInbox)
	defer func() {
		if err := dasRpcServer.Shutdown(ctx); err != nil {
			t.Logf("Error shutting down DAS RPC server: %v", err)
		}
	}()
	defer func() {
		if err := restServer.Shutdown(); err != nil {
			t.Logf("Error shutting down REST server: %v", err)
		}
	}()

	authorizeDASKeyset(t, ctx, pubkey, builder.L1Info, builder.L1.Client)

	// Mine L1 blocks to ensure keyset logs are queryable.
	// The keyset fetcher queries from blockNum to blockNum+1, so we need
	// at least one more block after the keyset transaction.
	TransferBalance(t, "Faucet", "User", big.NewInt(1), builder.L1Info, builder.L1.Client, ctx)

	t.Logf("AnyTrust DAS server running at: RPC=%s REST=%s", backendConfig.URL, restServerUrl)

	// 3. Configure sequencer node with AnyTrust only (no CustomDA)
	// Disable CustomDA
	builder.nodeConfig.DA.ExternalProvider.Enable = false

	// Enable AnyTrust
	builder.nodeConfig.DataAvailability.Enable = true
	builder.nodeConfig.DataAvailability.RPCAggregator = aggConfigForBackend(backendConfig)
	builder.nodeConfig.DataAvailability.RestAggregator = das.DefaultRestfulClientAggregatorConfig
	builder.nodeConfig.DataAvailability.RestAggregator.Enable = true
	builder.nodeConfig.DataAvailability.RestAggregator.Urls = []string{restServerUrl}

	// Disable fallback to on-chain (operator choice to prevent automatic expensive fallback)
	builder.nodeConfig.BatchPoster.DisableDapFallbackStoreDataOnChain = true

	// 4. Build L2
	builder.L2Info = NewArbTestInfo(t, builder.chainConfig.ChainID)
	builder.L2Info.GenerateAccount("User2")
	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()

	// 5. Setup follower node with same DA config
	nodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfigB.BlockValidator.Enable = false

	// Disable CustomDA
	nodeConfigB.DA.ExternalProvider.Enable = false

	// Enable AnyTrust
	nodeConfigB.DataAvailability.Enable = true
	nodeConfigB.DataAvailability.RestAggregator = das.DefaultRestfulClientAggregatorConfig
	nodeConfigB.DataAvailability.RestAggregator.Enable = true
	nodeConfigB.DataAvailability.RestAggregator.Urls = []string{restServerUrl}

	nodeBParams := SecondNodeParams{
		nodeConfig: nodeConfigB,
		initData:   &builder.L2Info.ArbInitData,
	}
	l2B, cleanupB := builder.Build2ndNode(t, &nodeBParams)
	defer cleanupB()

	// Phase 1: Normal AnyTrust operation
	t.Log("Phase 1: Testing normal AnyTrust operation")
	checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client,
		builder.L1Info, builder.L2Info, big.NewInt(1e12), l2B.Client)

	// Phase 2: Shutdown AnyTrust and verify batch posting fails
	t.Log("Phase 2: Shutting down AnyTrust, expecting batch posting to fail")
	err := dasRpcServer.Shutdown(ctx)
	Require(t, err)
	t.Logf("Phase 2: AnyTrust DAS RPC server shut down successfully")
	err = restServer.Shutdown()
	Require(t, err)
	t.Logf("Phase 2: AnyTrust DAS REST server shut down successfully")

	// Record the follower's current block before generating transactions
	followerBlockBefore, err := l2B.Client.BlockNumber(ctx)
	Require(t, err)
	t.Logf("Phase 2: Follower block before shutdown: %d", followerBlockBefore)

	// Generate transactions that would normally trigger a batch
	for i := 0; i < 10; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User2",
			builder.L2Info.TransferGas, big.NewInt(1e12), nil)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
	}
	t.Log("Phase 2: Generated transactions on sequencer")

	// Mine L1 blocks to give batch poster opportunities to post (and fail)
	for i := 0; i < 10; i++ {
		SendWaitTestTransactions(t, ctx, builder.L1.Client, []*types.Transaction{
			builder.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}
	t.Log("Phase 2: Mined L1 blocks")

	// Wait for batch poster to attempt and fail
	time.Sleep(5 * time.Second)

	// Verify follower did NOT sync (batch was not posted)
	followerBlockAfter, err := l2B.Client.BlockNumber(ctx)
	Require(t, err)
	t.Logf("Phase 2: Follower block after waiting: %d", followerBlockAfter)

	if followerBlockAfter > followerBlockBefore {
		t.Fatalf("Phase 2: Follower synced when it should not have (before=%d, after=%d)",
			followerBlockBefore, followerBlockAfter)
	}

	t.Log("SUCCESS: Batch posting failed as expected when operator disabled fallback to calldata")
}

// TestMultiWriterFallback_AnyTrustToCalldataOnBackendFailure tests that when AnyTrust aggregator
// fails due to insufficient backends, it triggers explicit fallback to the next DA provider (Calldata).
func TestMultiWriterFallback_AnyTrustToCalldataOnBackendFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Setup L1 chain and contracts
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.chainConfig = chaininfo.ArbitrumDevTestDASChainConfig()
	builder.parallelise = false

	builder.BuildL1(t)

	// 2. Setup AnyTrust/DAS server
	dasDataDir := t.TempDir()
	dasRpcServer, pubkey, backendConfig, restServer, restServerUrl := startLocalDASServer(
		t, ctx, dasDataDir, builder.L1.Client, builder.addresses.SequencerInbox)
	defer func() {
		if err := dasRpcServer.Shutdown(ctx); err != nil {
			t.Logf("Error shutting down DAS RPC server: %v", err)
		}
	}()
	defer func() {
		if err := restServer.Shutdown(); err != nil {
			t.Logf("Error shutting down REST server: %v", err)
		}
	}()

	authorizeDASKeyset(t, ctx, pubkey, builder.L1Info, builder.L1.Client)

	// Mine L1 blocks to ensure keyset logs are queryable
	TransferBalance(t, "Faucet", "User", big.NewInt(1), builder.L1Info, builder.L1.Client, ctx)

	t.Logf("AnyTrust DAS server running at: RPC=%s REST=%s", backendConfig.URL, restServerUrl)

	// 3. Configure sequencer node with AnyTrust → Calldata fallback
	// Disable CustomDA
	builder.nodeConfig.DA.ExternalProvider.Enable = false

	// Enable AnyTrust
	builder.nodeConfig.DataAvailability.Enable = true
	builder.nodeConfig.DataAvailability.RPCAggregator = aggConfigForBackend(backendConfig)
	builder.nodeConfig.DataAvailability.RestAggregator = das.DefaultRestfulClientAggregatorConfig
	builder.nodeConfig.DataAvailability.RestAggregator.Enable = true
	builder.nodeConfig.DataAvailability.RestAggregator.Urls = []string{restServerUrl}

	// Enable fallback to Calldata when AnyTrust fails
	builder.nodeConfig.BatchPoster.DisableDapFallbackStoreDataOnChain = false

	// 4. Build L2
	builder.L2Info = NewArbTestInfo(t, builder.chainConfig.ChainID)
	builder.L2Info.GenerateAccount("User2")
	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()

	// 5. Setup follower node with same DA config
	nodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfigB.BlockValidator.Enable = false

	// Disable CustomDA
	nodeConfigB.DA.ExternalProvider.Enable = false

	// Enable AnyTrust so follower can read Phase 1 batches
	nodeConfigB.DataAvailability.Enable = true
	nodeConfigB.DataAvailability.RestAggregator = das.DefaultRestfulClientAggregatorConfig
	nodeConfigB.DataAvailability.RestAggregator.Enable = true
	nodeConfigB.DataAvailability.RestAggregator.Urls = []string{restServerUrl}

	nodeBParams := SecondNodeParams{
		nodeConfig: nodeConfigB,
		initData:   &builder.L2Info.ArbInitData,
	}
	l2B, cleanupB := builder.Build2ndNode(t, &nodeBParams)
	defer cleanupB()

	// Phase 1: Normal AnyTrust operation
	t.Log("Phase 1: Testing normal AnyTrust operation")

	phase1StartBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client,
		builder.L1Info, builder.L2Info, big.NewInt(1e12), l2B.Client)

	phase1EndBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	// Verify Phase 1 used AnyTrust (0x88 header)
	seqInbox, err := arbnode.NewSequencerInbox(builder.L1.Client, builder.addresses.SequencerInbox, 0)
	Require(t, err)

	phase1Batches, err := seqInbox.LookupBatchesInRange(ctx, new(big.Int).SetUint64(phase1StartBlock), new(big.Int).SetUint64(phase1EndBlock))
	Require(t, err)

	phase1AnyTrustBatches := 0
	for _, batch := range phase1Batches {
		serializedBatch, err := batch.Serialize(ctx, builder.L1.Client)
		Require(t, err)

		if len(serializedBatch) <= 40 {
			continue
		}

		headerByte := serializedBatch[40]
		if daprovider.IsDASMessageHeaderByte(headerByte) {
			phase1AnyTrustBatches++
			t.Logf("Phase 1: Found AnyTrust batch (header=0x%02x)", headerByte)
		}
	}

	if phase1AnyTrustBatches == 0 {
		t.Fatal("Phase 1: Expected at least one AnyTrust batch, found none")
	}
	t.Logf("Phase 1: Posted %d AnyTrust batch(es)", phase1AnyTrustBatches)

	// Phase 2: Shut down AnyTrust backends and verify fallback to Calldata
	t.Log("Phase 2: Shutting down AnyTrust backends, expecting fallback to Calldata")

	err = dasRpcServer.Shutdown(ctx)
	Require(t, err)
	t.Logf("Phase 2: AnyTrust DAS RPC server shut down")
	err = restServer.Shutdown()
	Require(t, err)
	t.Logf("Phase 2: AnyTrust DAS REST server shut down")

	// Record L1 block range for Phase 2
	phase2StartBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	// Post a batch that should fall back to Calldata
	checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client,
		builder.L1Info, builder.L2Info, big.NewInt(2e12), l2B.Client)

	phase2EndBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	// Verify fallback to Calldata occurred
	phase2Batches, err := seqInbox.LookupBatchesInRange(ctx, new(big.Int).SetUint64(phase2StartBlock), new(big.Int).SetUint64(phase2EndBlock))
	Require(t, err)

	phase2CalldataBatches := 0
	for _, batch := range phase2Batches {
		serializedBatch, err := batch.Serialize(ctx, builder.L1.Client)
		Require(t, err)

		if len(serializedBatch) <= 40 {
			continue
		}

		headerByte := serializedBatch[40]
		if daprovider.IsBrotliMessageHeaderByte(headerByte) {
			phase2CalldataBatches++
			t.Logf("Phase 2: Found Calldata batch (header=0x%02x)", headerByte)
		}
	}

	if phase2CalldataBatches == 0 {
		t.Fatal("Phase 2: Expected fallback to Calldata, but found no calldata batches")
	}

	t.Logf("Phase 2: Posted %d calldata batch(es)", phase2CalldataBatches)
	t.Logf("SUCCESS: Phase 1 posted %d AnyTrust batch(es), Phase 2 fell back to %d Calldata batch(es)",
		phase1AnyTrustBatches, phase2CalldataBatches)
	t.Log("AnyTrust backend failure correctly triggered fallback to Calldata")
}
