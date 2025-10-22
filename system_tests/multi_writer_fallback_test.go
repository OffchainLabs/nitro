// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !race

package arbtest

import (
	"context"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"strings"
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

// TestMultiWriterFallback_CustomDAToAnyTrust tests the full fallback chain:
// CustomDA → AnyTrust → EthDA (calldata/4844) → CustomDA recovery
func TestMultiWriterFallback_CustomDAToAnyTrust(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Setup L1 chain and contracts
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.chainConfig = chaininfo.ArbitrumDevTestDASChainConfig()
	builder.parallelise = false

	// Deploy ReferenceDA validator contract
	builder.WithReferenceDA()

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
	builder.nodeConfig.DataAvailability.ParentChainNodeURL = "none"

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
	nodeConfigB.DataAvailability.ParentChainNodeURL = "none"

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

	// Phase 2: CustomDA failure → AnyTrust fallback
	t.Log("Phase 2: Shutting down CustomDA, testing fallback to AnyTrust")
	err := customDAServer.Shutdown(ctx)
	Require(t, err)
	t.Logf("Phase 2: CustomDA server shut down successfully")
	t.Logf("Phase 2: AnyTrust DAS RPC server should still be running at: %s", backendConfig.URL)

	checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client,
		builder.L1Info, builder.L2Info, big.NewInt(2e12), l2B.Client)

	// Phase 3: AnyTrust failure → EthDA fallback
	t.Log("Phase 3: Shutting down AnyTrust, testing fallback to EthDA")
	err = dasRpcServer.Shutdown(ctx)
	Require(t, err)
	t.Logf("Phase 3: AnyTrust DAS RPC server shut down successfully")
	err = restServer.Shutdown()
	Require(t, err)
	t.Logf("Phase 3: AnyTrust DAS REST server shut down successfully")

	checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client,
		builder.L1Info, builder.L2Info, big.NewInt(3e12), l2B.Client)

	// Phase 4: CustomDA recovery
	t.Log("Phase 4: Restarting CustomDA, testing recovery")

	// Extract port from original CustomDA URL to restart on same port
	customDAAddr := strings.TrimPrefix(customDAURL, "http://")
	customDAAddr = strings.TrimPrefix(customDAAddr, "https://")
	_, portStr, err := net.SplitHostPort(customDAAddr)
	Require(t, err)
	customDAPort, err := strconv.Atoi(portStr)
	Require(t, err)
	t.Logf("Phase 4: Restarting CustomDA on same port %d", customDAPort)

	// Restart on same port with retry for port reuse issues
	var customDAServer2 *http.Server
	var customDAURL2 string
	for i := 0; i < 5; i++ {
		customDAServer2, customDAURL2 = createReferenceDAProviderServer(t, ctx,
			builder.L1.Client, validatorAddr, dataSigner, customDAPort)
		if customDAServer2 != nil {
			break
		}
		t.Logf("Phase 4: Port not yet available, retrying... (attempt %d/5)", i+1)
		time.Sleep(time.Millisecond * 100)
	}
	if customDAServer2 == nil {
		t.Fatal("Phase 4: Failed to restart CustomDA server after 5 attempts")
	}
	defer func() {
		if err := customDAServer2.Shutdown(ctx); err != nil {
			t.Logf("Error shutting down CustomDA server 2: %v", err)
		}
	}()

	t.Logf("CustomDA server restarted at: %s (same port as before)", customDAURL2)

	// Give batch poster time to reconnect
	time.Sleep(time.Second * 2)

	// Track L1 block range for Phase 4 to verify CustomDA was used
	phase4StartBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client,
		builder.L1Info, builder.L2Info, big.NewInt(4e12), l2B.Client)

	phase4EndBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	// Verify Phase 4 batch used CustomDA
	t.Log("Phase 4 Verification: Checking that batch posted during Phase 4 used CustomDA")
	seqInbox, err := arbnode.NewSequencerInbox(builder.L1.Client, builder.addresses.SequencerInbox, 0)
	Require(t, err)

	phase4Batches, err := seqInbox.LookupBatchesInRange(ctx, new(big.Int).SetUint64(phase4StartBlock), new(big.Int).SetUint64(phase4EndBlock))
	Require(t, err)

	phase4CustomDAFound := false
	for _, batch := range phase4Batches {
		serializedBatch, err := batch.Serialize(ctx, builder.L1.Client)
		Require(t, err)

		if len(serializedBatch) <= 40 {
			continue
		}

		headerByte := serializedBatch[40]
		if daprovider.IsDACertificateMessageHeaderByte(headerByte) {
			t.Logf("Phase 4: Found CustomDA batch (header byte: 0x%02x)", headerByte)
			phase4CustomDAFound = true
			break
		} else {
			t.Logf("Phase 4: Found non-CustomDA batch (header byte: 0x%02x)", headerByte)
		}
	}

	if !phase4CustomDAFound {
		t.Fatal("Phase 4: Expected CustomDA to be used after restart, but it was not")
	}

	// Verification: Check that sequencer inbox contains batches from all three DA types
	t.Log("Verification: Checking sequencer inbox for all DA types")

	latestBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	batches, err := seqInbox.LookupBatchesInRange(ctx, big.NewInt(0), new(big.Int).SetUint64(latestBlock))
	Require(t, err)

	var customDASeen, anyTrustSeen, ethDASeen bool

	for _, batch := range batches {
		serializedBatch, err := batch.Serialize(ctx, builder.L1.Client)
		Require(t, err)

		if len(serializedBatch) <= 40 {
			continue
		}

		headerByte := serializedBatch[40]

		if daprovider.IsDACertificateMessageHeaderByte(headerByte) {
			t.Logf("Found CustomDA batch (header byte: 0x%02x)", headerByte)
			customDASeen = true
		} else if daprovider.IsDASMessageHeaderByte(headerByte) {
			t.Logf("Found AnyTrust batch (header byte: 0x%02x)", headerByte)
			anyTrustSeen = true
		} else if daprovider.IsBrotliMessageHeaderByte(headerByte) {
			t.Logf("Found EthDA/Calldata batch (header byte: 0x%02x)", headerByte)
			ethDASeen = true
		}
	}

	if !customDASeen {
		t.Error("Expected to see CustomDA batches in sequencer inbox")
	}
	if !anyTrustSeen {
		t.Error("Expected to see AnyTrust batches in sequencer inbox")
	}
	if !ethDASeen {
		t.Error("Expected to see EthDA batches in sequencer inbox")
	}

	if !customDASeen || !anyTrustSeen || !ethDASeen {
		t.Fatal("Expected batches from all three DA types")
	}

	t.Log("SUCCESS: All three DA types were used successfully")
}

// TestMultiWriterFallback_CustomDAToCalldata tests the two-way fallback chain:
// CustomDA → EthDA (calldata/4844) without AnyTrust
func TestMultiWriterFallback_CustomDAToCalldata(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Setup L1 chain and contracts
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	// Use standard dev test config (not DAS) since we're not using AnyTrust
	builder.chainConfig = chaininfo.ArbitrumDevTestChainConfig()
	builder.parallelise = false

	// Deploy ReferenceDA validator contract
	builder.WithReferenceDA()

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

	// Phase 2: CustomDA failure → EthDA fallback
	t.Log("Phase 2: Shutting down CustomDA, testing fallback to EthDA")
	err := customDAServer.Shutdown(ctx)
	Require(t, err)
	t.Logf("Phase 2: CustomDA server shut down successfully")

	checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client,
		builder.L1Info, builder.L2Info, big.NewInt(2e12), l2B.Client)

	// Phase 3: CustomDA recovery
	t.Log("Phase 3: Restarting CustomDA, testing recovery")

	// Extract port from original CustomDA URL to restart on same port
	customDAAddr := strings.TrimPrefix(customDAURL, "http://")
	customDAAddr = strings.TrimPrefix(customDAAddr, "https://")
	_, portStr, err := net.SplitHostPort(customDAAddr)
	Require(t, err)
	customDAPort, err := strconv.Atoi(portStr)
	Require(t, err)
	t.Logf("Phase 3: Restarting CustomDA on same port %d", customDAPort)

	// Restart on same port with retry for port reuse issues
	var customDAServer2 *http.Server
	var customDAURL2 string
	for i := 0; i < 5; i++ {
		customDAServer2, customDAURL2 = createReferenceDAProviderServer(t, ctx,
			builder.L1.Client, validatorAddr, dataSigner, customDAPort)
		if customDAServer2 != nil {
			break
		}
		t.Logf("Phase 3: Port not yet available, retrying... (attempt %d/5)", i+1)
		time.Sleep(time.Millisecond * 100)
	}
	if customDAServer2 == nil {
		t.Fatal("Phase 3: Failed to restart CustomDA server after 5 attempts")
	}
	defer func() {
		if err := customDAServer2.Shutdown(ctx); err != nil {
			t.Logf("Error shutting down CustomDA server 2: %v", err)
		}
	}()

	t.Logf("CustomDA server restarted at: %s (same port as before)", customDAURL2)

	// Give batch poster time to reconnect
	time.Sleep(time.Second * 2)

	// Track L1 block range for Phase 3 to verify CustomDA was used
	phase3StartBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client,
		builder.L1Info, builder.L2Info, big.NewInt(3e12), l2B.Client)

	phase3EndBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	// Verify Phase 3 batch used CustomDA
	t.Log("Phase 3 Verification: Checking that batch posted during Phase 3 used CustomDA")
	seqInbox, err := arbnode.NewSequencerInbox(builder.L1.Client, builder.addresses.SequencerInbox, 0)
	Require(t, err)

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
		t.Fatal("Phase 3: Expected CustomDA to be used after restart, but it was not")
	}

	// Verification: Check that sequencer inbox contains batches from both DA types
	t.Log("Verification: Checking sequencer inbox for both CustomDA and EthDA")

	latestBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	batches, err := seqInbox.LookupBatchesInRange(ctx, big.NewInt(0), new(big.Int).SetUint64(latestBlock))
	Require(t, err)

	var customDASeen, ethDASeen bool

	for _, batch := range batches {
		serializedBatch, err := batch.Serialize(ctx, builder.L1.Client)
		Require(t, err)

		if len(serializedBatch) <= 40 {
			continue
		}

		headerByte := serializedBatch[40]

		if daprovider.IsDACertificateMessageHeaderByte(headerByte) {
			t.Logf("Found CustomDA batch (header byte: 0x%02x)", headerByte)
			customDASeen = true
		} else if daprovider.IsBrotliMessageHeaderByte(headerByte) {
			t.Logf("Found EthDA/Calldata batch (header byte: 0x%02x)", headerByte)
			ethDASeen = true
		}
	}

	if !customDASeen {
		t.Error("Expected to see CustomDA batches in sequencer inbox")
	}
	if !ethDASeen {
		t.Error("Expected to see EthDA batches in sequencer inbox")
	}

	if !customDASeen || !ethDASeen {
		t.Fatal("Expected batches from both CustomDA and EthDA")
	}

	t.Log("SUCCESS: Both CustomDA and EthDA were used successfully")
}

// TestMultiWriterFallback_AnyTrustToCalldata tests the two-way fallback chain:
// AnyTrust → EthDA (calldata/4844) without CustomDA
func TestMultiWriterFallback_AnyTrustToCalldata(t *testing.T) {
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
	builder.nodeConfig.DataAvailability.ParentChainNodeURL = "none"

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

	// Disable CustomDA
	nodeConfigB.DA.ExternalProvider.Enable = false

	// Enable AnyTrust
	nodeConfigB.DataAvailability.Enable = true
	nodeConfigB.DataAvailability.RestAggregator = das.DefaultRestfulClientAggregatorConfig
	nodeConfigB.DataAvailability.RestAggregator.Enable = true
	nodeConfigB.DataAvailability.RestAggregator.Urls = []string{restServerUrl}
	nodeConfigB.DataAvailability.ParentChainNodeURL = "none"

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

	// Phase 2: AnyTrust failure → EthDA fallback
	t.Log("Phase 2: Shutting down AnyTrust, testing fallback to EthDA")
	err := dasRpcServer.Shutdown(ctx)
	Require(t, err)
	t.Logf("Phase 2: AnyTrust DAS RPC server shut down successfully")
	err = restServer.Shutdown()
	Require(t, err)
	t.Logf("Phase 2: AnyTrust DAS REST server shut down successfully")

	checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client,
		builder.L1Info, builder.L2Info, big.NewInt(2e12), l2B.Client)

	// Verification: Check that sequencer inbox contains batches from both DA types
	t.Log("Verification: Checking sequencer inbox for both AnyTrust and EthDA")
	seqInbox, err := arbnode.NewSequencerInbox(builder.L1.Client, builder.addresses.SequencerInbox, 0)
	Require(t, err)

	latestBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)

	batches, err := seqInbox.LookupBatchesInRange(ctx, big.NewInt(0), new(big.Int).SetUint64(latestBlock))
	Require(t, err)

	var anyTrustSeen, ethDASeen bool

	for _, batch := range batches {
		serializedBatch, err := batch.Serialize(ctx, builder.L1.Client)
		Require(t, err)

		if len(serializedBatch) <= 40 {
			continue
		}

		headerByte := serializedBatch[40]

		if daprovider.IsDASMessageHeaderByte(headerByte) {
			t.Logf("Found AnyTrust batch (header byte: 0x%02x)", headerByte)
			anyTrustSeen = true
		} else if daprovider.IsBrotliMessageHeaderByte(headerByte) {
			t.Logf("Found EthDA/Calldata batch (header byte: 0x%02x)", headerByte)
			ethDASeen = true
		}
	}

	if !anyTrustSeen {
		t.Error("Expected to see AnyTrust batches in sequencer inbox")
	}
	if !ethDASeen {
		t.Error("Expected to see EthDA batches in sequencer inbox")
	}

	if !anyTrustSeen || !ethDASeen {
		t.Fatal("Expected batches from both AnyTrust and EthDA")
	}

	t.Log("SUCCESS: Both AnyTrust and EthDA were used successfully")
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

// TestMultiWriterFallback_BatchResizing tests batch resizing when falling back to size-constrained EthDA.
// This test verifies that when a large batch built for AltDA (CustomDA) needs to fall back to EthDA,
// the batch poster correctly splits it into multiple smaller batches that fit within the calldata size limit.
func TestMultiWriterFallback_BatchResizing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Setup L1 chain and contracts
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.chainConfig = chaininfo.ArbitrumDevTestChainConfig()
	builder.parallelise = false

	// Deploy ReferenceDA validator contract
	builder.WithReferenceDA()

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

	// 3. Configure sequencer node with CustomDA and small batch size limits
	builder.nodeConfig.DA.ExternalProvider.Enable = true
	builder.nodeConfig.DA.ExternalProvider.RPC.URL = customDAURL
	builder.nodeConfig.DA.ExternalProvider.WithWriter = true

	// Disable AnyTrust
	builder.nodeConfig.DataAvailability.Enable = false

	// Enable fallback to on-chain
	builder.nodeConfig.BatchPoster.DisableDapFallbackStoreDataOnChain = false

	// Configure small batch size limits for testing
	// MaxAltDABatchSize: 10KB - large enough for Phase 1 batch
	// MaxSize: 3KB - forces multiple smaller batches in Phase 2
	// MaxDelay: 60s - safety net (should not be hit, we rely on size-based posting)
	builder.nodeConfig.BatchPoster.MaxAltDABatchSize = 10_000
	builder.nodeConfig.BatchPoster.MaxSize = 3_000
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

	// Phase 2: Force fallback and verify resize
	t.Log("Phase 2: Shutting down CustomDA, testing batch resizing fallback to EthDA")

	err = customDAServer.Shutdown(ctx)
	Require(t, err)
	t.Logf("Phase 2: CustomDA server shut down successfully")

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
