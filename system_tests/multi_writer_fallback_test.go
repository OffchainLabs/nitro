// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !race

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/das"
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
	customDAServer, customDAURL := createReferenceDAProviderServer(t, ctx, builder.L1.Client, validatorAddr, dataSigner)
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
	customDAServer2, customDAURL2 := createReferenceDAProviderServer(t, ctx,
		builder.L1.Client, validatorAddr, dataSigner)
	defer func() {
		if err := customDAServer2.Shutdown(ctx); err != nil {
			t.Logf("Error shutting down CustomDA server 2: %v", err)
		}
	}()

	// Update node config (hot reload)
	builder.nodeConfig.DA.ExternalProvider.RPC.URL = customDAURL2
	nodeConfigB.DA.ExternalProvider.RPC.URL = customDAURL2

	t.Logf("CustomDA server restarted at: %s", customDAURL2)

	// Give batch poster time to pick up new config
	time.Sleep(time.Second * 2)

	checkBatchPosting(t, ctx, builder.L1.Client, builder.L2.Client,
		builder.L1Info, builder.L2Info, big.NewInt(4e12), l2B.Client)

	// Verification: Check that sequencer inbox contains batches from all three DA types
	t.Log("Verification: Checking sequencer inbox for all DA types")
	seqInbox, err := arbnode.NewSequencerInbox(builder.L1.Client, builder.addresses.SequencerInbox, 0)
	Require(t, err)

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
