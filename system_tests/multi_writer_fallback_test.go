// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !race

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/das"
	"github.com/offchainlabs/nitro/daprovider/referenceda"
	"github.com/offchainlabs/nitro/util/signature"
)

// TestMultiWriterFailure_CustomDAShutdownWithAnyTrustAvailable tests that batch posting fails when CustomDA shuts down
// without explicit fallback signal. This verifies the new behavior where DA errors must be
// explicitly signaled via ErrFallbackRequested to trigger fallback. Even though AnyTrust is configured
// and available, it should not be used without an explicit fallback signal.
func TestMultiWriterFailure_CustomDAShutdownWithAnyTrustAvailable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Setup L1 chain and contracts
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.chainConfig = chaininfo.ArbitrumDevTestDASChainConfig()
	builder.parallelise = false

	// Deploy ReferenceDA validator contract
	builder.WithReferenceDAContractsOnly()

	builder.BuildL1(t)

	// 2. Setup CustomDA provider server (ReferenceDA)
	l1info := builder.L1Info
	dataSigner := signature.DataSignerFromPrivateKey(l1info.GetInfoWithPrivKey("Sequencer").PrivateKey)
	validatorAddr := l1info.GetAddress("ReferenceDAProofValidator")
	customDAServer, customDAURL := createReferenceDAProviderServer(t, ctx, builder.L1.Client, validatorAddr, dataSigner, 0)
	defer func() {
		if err := customDAServer.Shutdown(ctx); err != nil {
			t.Logf("Error shutting down CustomDA server: %v", err)
		}
	}()

	t.Logf("CustomDA server running at: %s", customDAURL)

	// 3. Setup AnyTrust/DAS server
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

	t.Logf("AnyTrust DAS server running at: RPC=%s REST=%s", backendConfig.URL, restServerUrl)

	// 4. Configure sequencer node with both CustomDA and AnyTrust
	builder.nodeConfig.DA.ExternalProvider.Enable = true
	builder.nodeConfig.DA.ExternalProvider.RPC.URL = customDAURL
	builder.nodeConfig.DA.ExternalProvider.WithWriter = true

	builder.nodeConfig.DataAvailability.Enable = true
	builder.nodeConfig.DataAvailability.RPCAggregator = aggConfigForBackend(backendConfig)
	builder.nodeConfig.DataAvailability.RestAggregator = das.DefaultRestfulClientAggregatorConfig
	builder.nodeConfig.DataAvailability.RestAggregator.Enable = true
	builder.nodeConfig.DataAvailability.RestAggregator.Urls = []string{restServerUrl}

	// Enable fallback to on-chain
	builder.nodeConfig.BatchPoster.DisableDapFallbackStoreDataOnChain = false

	// 5. Build L2
	builder.L2Info = NewArbTestInfo(t, builder.chainConfig.ChainID)
	builder.L2Info.GenerateAccount("User2")
	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()

	// 6. Setup follower node with same DA config
	nodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfigB.BlockValidator.Enable = false

	// CustomDA config
	nodeConfigB.DA.ExternalProvider.Enable = true
	nodeConfigB.DA.ExternalProvider.RPC.URL = customDAURL

	// AnyTrust config
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

	// Phase 1: Normal CustomDA operation
	t.Log("Phase 1: Testing normal CustomDA operation")
	checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client,
		builder.L1Info, builder.L2Info, big.NewInt(1e12), l2B.Client)

	// Phase 2: Shutdown CustomDA and verify batch posting fails
	t.Log("Phase 2: Shutting down CustomDA, expecting batch posting to fail")
	err := customDAServer.Shutdown(ctx)
	Require(t, err)
	t.Logf("Phase 2: CustomDA server shut down successfully")

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

	t.Log("SUCCESS: Batch posting failed as expected when CustomDA shut down without fallback signal")
}

// TestMultiWriterFailure_CustomDAShutdownNoFallbackAvailable tests that batch posting fails when CustomDA
// shuts down without AnyTrust configured. This verifies failure behavior in a CustomDA-only setup
// where no fallback DA provider is available.
func TestMultiWriterFailure_CustomDAShutdownNoFallbackAvailable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Setup L1 chain and contracts
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	// Use standard dev test config (not DAS) since we're not using AnyTrust
	builder.chainConfig = chaininfo.ArbitrumDevTestChainConfig()
	builder.parallelise = false

	// Deploy ReferenceDA validator contract
	builder.WithReferenceDAContractsOnly()

	builder.BuildL1(t)

	// 2. Setup CustomDA provider server (ReferenceDA)
	l1info := builder.L1Info
	dataSigner := signature.DataSignerFromPrivateKey(l1info.GetInfoWithPrivKey("Sequencer").PrivateKey)
	validatorAddr := l1info.GetAddress("ReferenceDAProofValidator")
	customDAServer, customDAURL := createReferenceDAProviderServer(t, ctx, builder.L1.Client, validatorAddr, dataSigner, 0)
	defer func() {
		if err := customDAServer.Shutdown(ctx); err != nil {
			t.Logf("Error shutting down CustomDA server: %v", err)
		}
	}()

	t.Logf("CustomDA server running at: %s", customDAURL)

	// 3. Configure sequencer node with CustomDA only (no AnyTrust)
	builder.nodeConfig.DA.ExternalProvider.Enable = true
	builder.nodeConfig.DA.ExternalProvider.RPC.URL = customDAURL
	builder.nodeConfig.DA.ExternalProvider.WithWriter = true

	// Disable AnyTrust
	builder.nodeConfig.DataAvailability.Enable = false

	// Enable fallback to on-chain
	builder.nodeConfig.BatchPoster.DisableDapFallbackStoreDataOnChain = false

	// 4. Build L2
	builder.L2Info = NewArbTestInfo(t, builder.chainConfig.ChainID)
	builder.L2Info.GenerateAccount("User2")
	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()

	// 5. Setup follower node with same DA config
	nodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfigB.BlockValidator.Enable = false

	// CustomDA config
	nodeConfigB.DA.ExternalProvider.Enable = true
	nodeConfigB.DA.ExternalProvider.RPC.URL = customDAURL

	// Disable AnyTrust
	nodeConfigB.DataAvailability.Enable = false

	nodeBParams := SecondNodeParams{
		nodeConfig: nodeConfigB,
		initData:   &builder.L2Info.ArbInitData,
	}
	l2B, cleanupB := builder.Build2ndNode(t, &nodeBParams)
	defer cleanupB()

	// Phase 1: Normal CustomDA operation
	t.Log("Phase 1: Testing normal CustomDA operation")
	checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client,
		builder.L1Info, builder.L2Info, big.NewInt(1e12), l2B.Client)

	// Phase 2: Shutdown CustomDA and verify batch posting fails
	t.Log("Phase 2: Shutting down CustomDA, expecting batch posting to fail")
	err := customDAServer.Shutdown(ctx)
	Require(t, err)
	t.Logf("Phase 2: CustomDA server shut down successfully")

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

	t.Log("SUCCESS: Batch posting failed as expected when CustomDA shut down without fallback signal")
}

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

// TestMultiWriterFallback_CustomDAToAnyTrustExplicit tests that explicit fallback signals work correctly.
// When CustomDA returns ErrFallbackRequested, the batch poster should fall back to AnyTrust.
func TestMultiWriterFallback_CustomDAToAnyTrustExplicit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Setup L1 chain and contracts
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.chainConfig = chaininfo.ArbitrumDevTestDASChainConfig()
	builder.parallelise = false

	// Deploy ReferenceDA validator contract
	builder.WithReferenceDAContractsOnly()

	builder.BuildL1(t)

	// 2. Setup CustomDA provider server with control handles
	l1info := builder.L1Info
	dataSigner := signature.DataSignerFromPrivateKey(l1info.GetInfoWithPrivKey("Sequencer").PrivateKey)
	validatorAddr := l1info.GetAddress("ReferenceDAProofValidator")
	customDAServer, customDAURL, writerControl := createReferenceDAProviderServerWithControl(t, ctx, builder.L1.Client, validatorAddr, dataSigner, 0, referenceda.DefaultConfig.MaxBatchSize)
	defer func() {
		if err := customDAServer.Shutdown(ctx); err != nil {
			t.Logf("Error shutting down CustomDA server: %v", err)
		}
	}()

	t.Logf("CustomDA server with control running at: %s", customDAURL)

	// 3. Setup AnyTrust/DAS server
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

	t.Logf("AnyTrust DAS server running at: RPC=%s REST=%s", backendConfig.URL, restServerUrl)

	// 4. Configure sequencer node with both CustomDA and AnyTrust
	builder.nodeConfig.DA.ExternalProvider.Enable = true
	builder.nodeConfig.DA.ExternalProvider.RPC.URL = customDAURL
	builder.nodeConfig.DA.ExternalProvider.WithWriter = true

	builder.nodeConfig.DataAvailability.Enable = true
	builder.nodeConfig.DataAvailability.RPCAggregator = aggConfigForBackend(backendConfig)
	builder.nodeConfig.DataAvailability.RestAggregator = das.DefaultRestfulClientAggregatorConfig
	builder.nodeConfig.DataAvailability.RestAggregator.Enable = true
	builder.nodeConfig.DataAvailability.RestAggregator.Urls = []string{restServerUrl}

	// Enable fallback to on-chain
	builder.nodeConfig.BatchPoster.DisableDapFallbackStoreDataOnChain = false

	// 5. Build L2
	builder.L2Info = NewArbTestInfo(t, builder.chainConfig.ChainID)
	builder.L2Info.GenerateAccount("User2")
	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()

	// 6. Setup follower node with same DA config
	nodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfigB.BlockValidator.Enable = false

	// CustomDA config
	nodeConfigB.DA.ExternalProvider.Enable = true
	nodeConfigB.DA.ExternalProvider.RPC.URL = customDAURL

	// AnyTrust config
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

	// Phase 1: Normal CustomDA operation
	t.Log("Phase 1: Testing normal CustomDA operation")
	checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client,
		builder.L1Info, builder.L2Info, big.NewInt(1e12), l2B.Client)

	// Phase 2: Trigger explicit fallback and verify AnyTrust is used
	t.Log("Phase 2: Triggering explicit fallback from CustomDA to AnyTrust")

	// Trigger fallback by setting control handle directly
	writerControl.SetShouldFallback(true)
	t.Log("Phase 2: Set fallback flag to true")

	// Record L1 block range for Phase 2
	phase2StartBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	// Post a batch that should fall back to AnyTrust
	checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client,
		builder.L1Info, builder.L2Info, big.NewInt(2e12), l2B.Client)

	phase2EndBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	// Verify Phase 2 batch used AnyTrust
	t.Log("Phase 2 Verification: Checking that batch posted during Phase 2 used AnyTrust")
	seqInbox, err := arbnode.NewSequencerInbox(builder.L1.Client, builder.addresses.SequencerInbox, 0)
	Require(t, err)

	phase2Batches, err := seqInbox.LookupBatchesInRange(ctx, new(big.Int).SetUint64(phase2StartBlock), new(big.Int).SetUint64(phase2EndBlock))
	Require(t, err)

	phase2AnyTrustFound := false
	for _, batch := range phase2Batches {
		serializedBatch, err := batch.Serialize(ctx, builder.L1.Client)
		Require(t, err)

		if len(serializedBatch) <= 40 {
			continue
		}

		headerByte := serializedBatch[40]
		if daprovider.IsDASMessageHeaderByte(headerByte) {
			t.Logf("Phase 2: Found AnyTrust batch (header byte: 0x%02x)", headerByte)
			phase2AnyTrustFound = true
			break
		} else {
			t.Logf("Phase 2: Found non-AnyTrust batch (header byte: 0x%02x)", headerByte)
		}
	}

	if !phase2AnyTrustFound {
		t.Fatal("Phase 2: Expected AnyTrust to be used after explicit fallback, but it was not")
	}

	// Phase 3: Reset CustomDA and verify it's used again
	t.Log("Phase 3: Resetting CustomDA, verifying it's used again")

	// Reset by clearing control handles directly
	writerControl.SetShouldFallback(false)
	writerControl.SetCustomError(nil)
	t.Log("Phase 3: Reset fallback flag to false")

	// Record L1 block range for Phase 3
	phase3StartBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	// Post another batch that should use CustomDA again
	checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client,
		builder.L1Info, builder.L2Info, big.NewInt(3e12), l2B.Client)

	phase3EndBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	// Verify Phase 3 batch used CustomDA
	t.Log("Phase 3 Verification: Checking that batch posted during Phase 3 used CustomDA")
	phase3Batches, err := seqInbox.LookupBatchesInRange(ctx, new(big.Int).SetUint64(phase3StartBlock), new(big.Int).SetUint64(phase3EndBlock))
	Require(t, err)

	phase3CustomDAFound := false
	for _, batch := range phase3Batches {
		serializedBatch, err := batch.Serialize(ctx, builder.L1.Client)
		Require(t, err)

		if len(serializedBatch) <= 40 {
			continue
		}

		headerByte := serializedBatch[40]
		if daprovider.IsDACertificateMessageHeaderByte(headerByte) {
			t.Logf("Phase 3: Found CustomDA batch (header byte: 0x%02x)", headerByte)
			phase3CustomDAFound = true
			break
		} else {
			t.Logf("Phase 3: Found non-CustomDA batch (header byte: 0x%02x)", headerByte)
		}
	}

	if !phase3CustomDAFound {
		t.Fatal("Phase 3: Expected CustomDA to be used after reset, but it was not")
	}

	t.Log("SUCCESS: Explicit fallback signal correctly triggered fallback from CustomDA to AnyTrust")
}

// getCustomDAPayloadSize recovers the actual payload size from a CustomDA batch.
// For CustomDA, the sequencer inbox only contains a small certificate (~98 bytes),
// so we need to use the Reader to fetch the actual data from storage.
func getCustomDAPayloadSize(
	t *testing.T,
	ctx context.Context,
	batch *arbnode.SequencerInboxBatch,
	l1Client *ethclient.Client,
	validatorAddr common.Address,
) int {
	t.Helper()

	// Get the serialized batch (contains the certificate)
	serializedBatch, err := batch.Serialize(ctx, l1Client)
	Require(t, err)

	if len(serializedBatch) <= 40 {
		t.Fatal("Batch too small to contain certificate")
	}

	// Create a reader using the same storage as the server
	storage := referenceda.GetInMemoryStorage()
	reader := referenceda.NewReader(storage, l1Client, validatorAddr)

	// Use RecoverPayload to fetch the actual data
	payloadPromise := reader.RecoverPayload(
		batch.SequenceNumber,
		batch.BlockHash,
		serializedBatch,
	)

	// Wait for the promise to resolve
	payloadResult, err := payloadPromise.Await(ctx)
	Require(t, err)

	return len(payloadResult.Payload)
}

// TestMultiWriterFallback_CustomDAToCalldataWithBatchResizing tests batch resizing when falling back to size-constrained EthDA.
// This test verifies that when a large batch built for AltDA (CustomDA) needs to fall back to EthDA,
// the batch poster correctly splits it into multiple smaller batches that fit within the calldata size limit.
func TestMultiWriterFallback_CustomDAToCalldataWithBatchResizing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Setup L1 chain and contracts
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.chainConfig = chaininfo.ArbitrumDevTestChainConfig()
	builder.parallelise = false

	// Deploy ReferenceDA validator contract
	builder.WithReferenceDAContractsOnly()

	builder.BuildL1(t)

	// 2. Setup CustomDA provider server with control handles (ReferenceDA)
	l1info := builder.L1Info
	dataSigner := signature.DataSignerFromPrivateKey(l1info.GetInfoWithPrivKey("Sequencer").PrivateKey)
	validatorAddr := l1info.GetAddress("ReferenceDAProofValidator")
	customDAServer, customDAURL, writerControl := createReferenceDAProviderServerWithControl(t, ctx, builder.L1.Client, validatorAddr, dataSigner, 0, 10_000)
	defer func() {
		if err := customDAServer.Shutdown(ctx); err != nil {
			t.Logf("Error shutting down CustomDA server: %v", err)
		}
	}()

	t.Logf("CustomDA server with control running at: %s", customDAURL)

	// 3. Configure sequencer node with CustomDA and small batch size limits
	builder.nodeConfig.DA.ExternalProvider.Enable = true
	builder.nodeConfig.DA.ExternalProvider.RPC.URL = customDAURL
	builder.nodeConfig.DA.ExternalProvider.WithWriter = true

	// Disable AnyTrust
	builder.nodeConfig.DataAvailability.Enable = false

	// Enable fallback to on-chain
	builder.nodeConfig.BatchPoster.DisableDapFallbackStoreDataOnChain = false

	// Configure small batch size limits for testing
	// AltDA batch size: 10KB - large enough for Phase 1 batch (set via maxMessageSize param above)
	// MaxCalldataBatchSize: 3KB - forces multiple smaller batches in Phase 2
	// MaxDelay: 60s - safety net (should not be hit, we rely on size-based posting)
	builder.nodeConfig.BatchPoster.MaxCalldataBatchSize = 3_000
	builder.nodeConfig.BatchPoster.MaxDelay = 60 * time.Second

	// 4. Build L2
	builder.L2Info = NewArbTestInfo(t, builder.chainConfig.ChainID)
	builder.L2Info.GenerateAccount("User2")
	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()

	// 5. Setup follower node with same DA config
	nodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfigB.BlockValidator.Enable = false

	// CustomDA config
	nodeConfigB.DA.ExternalProvider.Enable = true
	nodeConfigB.DA.ExternalProvider.RPC.URL = customDAURL

	// Disable AnyTrust
	nodeConfigB.DataAvailability.Enable = false

	nodeBParams := SecondNodeParams{
		nodeConfig: nodeConfigB,
		initData:   &builder.L2Info.ArbInitData,
	}
	l2B, cleanupB := builder.Build2ndNode(t, &nodeBParams)
	defer cleanupB()

	// Phase 1: Build large batch for CustomDA
	t.Log("Phase 1: Generating transactions to hit MaxAltDABatchSize (10KB)")

	// Record L1 block before Phase 1
	l1BlockBeforePhase1, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	// Generate enough transactions to hit MaxAltDABatchSize
	// Over-generate to ensure we hit the limit
	for i := 0; i < 250; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User2",
			builder.L2Info.TransferGas, big.NewInt(1e12), nil)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
	}

	t.Log("Phase 1: Generated 250 transactions")

	// Create L1 blocks to process messages and trigger batch posting
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, builder.L1.Client, []*types.Transaction{
			builder.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

	t.Log("Phase 1: Created L1 blocks to trigger batch posting")

	// Brief pause to ensure batch is posted
	time.Sleep(time.Second * 2)

	// Verify follower synced
	_, err = builder.L2.Client.BlockNumber(ctx)
	Require(t, err)
	l2BBlockNum, err := l2B.Client.BlockNumber(ctx)
	Require(t, err)
	if l2BBlockNum == 0 {
		t.Fatal("Phase 1: Follower node did not sync")
	}

	// Record L1 block after Phase 1
	l1BlockAfterPhase1, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	// Check Phase 1 batches
	seqInbox, err := arbnode.NewSequencerInbox(builder.L1.Client, builder.addresses.SequencerInbox, 0)
	Require(t, err)

	// #nosec G115
	phase1Batches, err := seqInbox.LookupBatchesInRange(ctx, big.NewInt(int64(l1BlockBeforePhase1)), big.NewInt(int64(l1BlockAfterPhase1)))
	Require(t, err)

	var phase1CustomDABatches int
	for _, batch := range phase1Batches {
		serializedBatch, err := batch.Serialize(ctx, builder.L1.Client)
		Require(t, err)

		if len(serializedBatch) <= 40 {
			continue
		}

		headerByte := serializedBatch[40]
		if daprovider.IsDACertificateMessageHeaderByte(headerByte) {
			phase1CustomDABatches++

			// For CustomDA, the sequencer inbox only contains a small certificate.
			// Recover the actual payload from storage to check its size.
			payloadSize := getCustomDAPayloadSize(t, ctx, batch, builder.L1.Client, validatorAddr)
			t.Logf("Phase 1: Found CustomDA batch, actual payload size=%d bytes", payloadSize)

			// Verify batch payload is approximately 10KB (8KB-12KB range)
			if payloadSize < 8_000 || payloadSize > 12_000 {
				t.Errorf("Phase 1: CustomDA payload size %d outside expected range 8KB-12KB", payloadSize)
			}
		}
	}

	if phase1CustomDABatches < 1 {
		t.Fatal("Phase 1: Expected at least 1 CustomDA batch")
	}

	t.Logf("Phase 1: Posted %d CustomDA batch(es)", phase1CustomDABatches)

	// Phase 2: Trigger explicit fallback and verify batch resizing
	t.Log("Phase 2: Triggering explicit fallback from CustomDA, testing batch resizing fallback to EthDA")

	// Trigger fallback by setting control handle directly
	writerControl.SetShouldFallback(true)
	t.Log("Phase 2: Set fallback flag to true")

	// Record L1 block before Phase 2
	l1BlockBeforePhase2, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	// Generate more transactions (same count as Phase 1)
	for i := 0; i < 250; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User2",
			builder.L2Info.TransferGas, big.NewInt(1e12), nil)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
	}

	t.Log("Phase 2: Generated 250 transactions")

	// Create L1 blocks to process messages and trigger batch posting
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, builder.L1.Client, []*types.Transaction{
			builder.L1Info.PrepareTx("Faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

	t.Log("Phase 2: Created L1 blocks to trigger batch posting")

	// Longer pause to allow multiple batches to be posted
	time.Sleep(time.Second * 3)

	// Record L1 block after Phase 2
	l1BlockAfterPhase2, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	// Check Phase 2 batches
	// #nosec G115
	phase2Batches, err := seqInbox.LookupBatchesInRange(ctx, big.NewInt(int64(l1BlockBeforePhase2)), big.NewInt(int64(l1BlockAfterPhase2)))
	Require(t, err)

	var phase2CalldataBatches int
	for _, batch := range phase2Batches {
		serializedBatch, err := batch.Serialize(ctx, builder.L1.Client)
		Require(t, err)

		if len(serializedBatch) <= 40 {
			continue
		}

		headerByte := serializedBatch[40]
		if daprovider.IsBrotliMessageHeaderByte(headerByte) {
			phase2CalldataBatches++
			batchSize := len(serializedBatch)
			t.Logf("Phase 2: Found Calldata batch, size=%d bytes", batchSize)

			// Verify batch is small (~3KB, max 4KB)
			if batchSize > 4_000 {
				t.Errorf("Phase 2: Calldata batch size %d exceeds expected max 4KB", batchSize)
			}
		}
	}

	// Expect at least 3 calldata batches for resize (250 txs should split into 4-5 batches)
	if phase2CalldataBatches < 3 {
		t.Fatalf("Phase 2: Expected at least 3 calldata batches for resize, got %d", phase2CalldataBatches)
	}

	t.Logf("Phase 2: Posted %d calldata batch(es)", phase2CalldataBatches)

	// Final verification
	t.Logf("SUCCESS: Test passed - %d CustomDA batch(es) in Phase 1, %d calldata batch(es) in Phase 2",
		phase1CustomDABatches, phase2CalldataBatches)
	t.Log("Batch resizing worked correctly: large batch for CustomDA split into multiple smaller batches for EthDA")
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
