// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !race

package arbtest

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/anytrust"
)

// Post-retirement: chain config still requires DAC and DAS is unreachable.
// The dangerous flag must short-circuit AnyTrust wiring so the node never tries to connect.
func TestDangerousAlwaysFallback_BatchPosterPostsToParentChain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.chainConfig = chaininfo.ArbitrumDevTestAnyTrustChainConfig()
	builder.parallelise = false
	builder.BuildL1(t)

	builder.nodeConfig.Dangerous.AlwaysFallbackToParentChainDA = true
	builder.nodeConfig.DA.AnyTrust.Enable = true

	builder.L2Info = NewArbTestInfo(t, builder.chainConfig.ChainID)
	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()

	nodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfigB.BlockValidator.Enable = false
	nodeConfigB.Dangerous.AlwaysFallbackToParentChainDA = true
	nodeConfigB.DA.AnyTrust.Enable = true

	l2B, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{
		nodeConfig: nodeConfigB,
		initData:   &builder.L2Info.ArbInitData,
	})
	defer cleanupB()

	checkBatchPosting(t, ctx, builder, l2B.Client)
	assertNoAnyTrustBatchesOnL1(t, ctx, builder)
}

// Migration step 1: DAS still alive, AnyTrust still enabled in config, but the
// dangerous flag must override and force the batch poster to post to L1.
func TestDangerousAlwaysFallback_OverridesEnabledAnyTrust(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.chainConfig = chaininfo.ArbitrumDevTestAnyTrustChainConfig()
	builder.parallelise = false
	builder.BuildL1(t)

	rpcServer, pubkey, _, restServer, restURL := startLocalAnyTrustServer(t, ctx, t.TempDir(), builder.L1.Client, builder.addresses.SequencerInbox)
	defer func() { _ = rpcServer.Shutdown(ctx) }()
	defer func() { _ = restServer.Shutdown() }()
	authorizeAnyTrustKeyset(t, ctx, pubkey, builder.L1Info, builder.L1.Client)

	builder.nodeConfig.Dangerous.AlwaysFallbackToParentChainDA = true
	builder.nodeConfig.DA.AnyTrust.Enable = true
	builder.nodeConfig.DA.AnyTrust.RestAggregator = anytrust.DefaultRestfulClientAggregatorConfig
	builder.nodeConfig.DA.AnyTrust.RestAggregator.Enable = true
	builder.nodeConfig.DA.AnyTrust.RestAggregator.Urls = []string{restURL}

	builder.L2Info = NewArbTestInfo(t, builder.chainConfig.ChainID)
	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()

	nodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfigB.BlockValidator.Enable = false
	nodeConfigB.Dangerous.AlwaysFallbackToParentChainDA = true
	nodeConfigB.DA.AnyTrust.Enable = true
	nodeConfigB.DA.AnyTrust.RestAggregator = anytrust.DefaultRestfulClientAggregatorConfig
	nodeConfigB.DA.AnyTrust.RestAggregator.Enable = true
	nodeConfigB.DA.AnyTrust.RestAggregator.Urls = []string{restURL}

	l2B, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{
		nodeConfig: nodeConfigB,
		initData:   &builder.L2Info.ArbInitData,
	})
	defer cleanupB()

	checkBatchPosting(t, ctx, builder, l2B.Client)
	assertNoAnyTrustBatchesOnL1(t, ctx, builder)
}

func TestDangerousAlwaysFallback_SyncingFatalsOnAnyTrustBatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.chainConfig = chaininfo.ArbitrumDevTestAnyTrustChainConfig()
	builder.parallelise = false
	builder.BuildL1(t)

	rpcServer, pubkey, backendConfig, restServer, restURL := startLocalAnyTrustServer(t, ctx, t.TempDir(), builder.L1.Client, builder.addresses.SequencerInbox)
	defer func() { _ = rpcServer.Shutdown(ctx) }()
	defer func() { _ = restServer.Shutdown() }()
	authorizeAnyTrustKeyset(t, ctx, pubkey, builder.L1Info, builder.L1.Client)

	builder.nodeConfig.DA.AnyTrust.Enable = true
	builder.nodeConfig.DA.AnyTrust.RPCAggregator = aggConfigForBackend(backendConfig)
	builder.nodeConfig.DA.AnyTrust.RestAggregator = anytrust.DefaultRestfulClientAggregatorConfig
	builder.nodeConfig.DA.AnyTrust.RestAggregator.Enable = true
	builder.nodeConfig.DA.AnyTrust.RestAggregator.Urls = []string{restURL}

	builder.L2Info = NewArbTestInfo(t, builder.chainConfig.ChainID)
	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("Recipient")
	recipient := builder.L2Info.GetAddress("Recipient")
	tx := builder.L2Info.PrepareTxTo("Owner", &recipient, builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	builder.L2.SendWaitTestTransactions(t, []*types.Transaction{tx})
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)

	waitForAnyTrustBatchOnL1(t, ctx, builder, 30*time.Second)

	nodeBFatalErrChan := make(chan error, 10)
	nodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfigB.BlockValidator.Enable = false
	nodeConfigB.Dangerous.AlwaysFallbackToParentChainDA = true
	nodeConfigB.DA.AnyTrust.Enable = true

	_, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{
		nodeConfig:   nodeConfigB,
		initData:     &builder.L2Info.ArbInitData,
		fatalErrChan: nodeBFatalErrChan,
	})
	defer cleanupB()

	select {
	case err := <-nodeBFatalErrChan:
		if !errors.Is(err, daprovider.ErrAnyTrustRequiresFallback) {
			t.Fatalf("expected error to wrap ErrAnyTrustRequiresFallback, got: %v", err)
		}
		var typed *daprovider.AnyTrustRequiresFallbackError
		if !errors.As(err, &typed) {
			t.Fatalf("expected error to wrap *AnyTrustRequiresFallbackError, got: %v", err)
		}
		if !strings.Contains(err.Error(), "inbox reader:") {
			t.Fatalf("expected error to be wrapped with \"inbox reader:\" prefix, got: %v", err)
		}
		t.Logf("Node B fataled on batch %d: %v", typed.BatchNum, err)
	case <-time.After(60 * time.Second):
		t.Fatal("timed out waiting for Node B to fatal on AnyTrust batch")
	}
}

// Phase-out: sequencer (no flag) posts real AnyTrust batches; a syncing node
// with the flag plus rest-aggregator must read them from the committee, not halt.
func TestDangerousAlwaysFallback_SyncingReadsFromCommittee(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.chainConfig = chaininfo.ArbitrumDevTestAnyTrustChainConfig()
	builder.parallelise = false
	builder.BuildL1(t)

	rpcServer, pubkey, backendConfig, restServer, restURL := startLocalAnyTrustServer(t, ctx, t.TempDir(), builder.L1.Client, builder.addresses.SequencerInbox)
	defer func() { _ = rpcServer.Shutdown(ctx) }()
	defer func() { _ = restServer.Shutdown() }()
	authorizeAnyTrustKeyset(t, ctx, pubkey, builder.L1Info, builder.L1.Client)

	builder.nodeConfig.DA.AnyTrust.Enable = true
	builder.nodeConfig.DA.AnyTrust.RPCAggregator = aggConfigForBackend(backendConfig)
	builder.nodeConfig.DA.AnyTrust.RestAggregator = anytrust.DefaultRestfulClientAggregatorConfig
	builder.nodeConfig.DA.AnyTrust.RestAggregator.Enable = true
	builder.nodeConfig.DA.AnyTrust.RestAggregator.Urls = []string{restURL}

	builder.L2Info = NewArbTestInfo(t, builder.chainConfig.ChainID)
	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("Recipient")
	recipient := builder.L2Info.GetAddress("Recipient")
	tx := builder.L2Info.PrepareTxTo("Owner", &recipient, builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	builder.L2.SendWaitTestTransactions(t, []*types.Transaction{tx})
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)

	waitForAnyTrustBatchOnL1(t, ctx, builder, 30*time.Second)

	sequencerBatchCount, err := builder.L2.ConsensusNode.GetParentChainDataSource().GetBatchCount()
	Require(t, err)

	nodeBFatalErrChan := make(chan error, 10)
	nodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfigB.BlockValidator.Enable = false
	nodeConfigB.Dangerous.AlwaysFallbackToParentChainDA = true
	nodeConfigB.DA.AnyTrust.Enable = true
	nodeConfigB.DA.AnyTrust.RestAggregator = anytrust.DefaultRestfulClientAggregatorConfig
	nodeConfigB.DA.AnyTrust.RestAggregator.Enable = true
	nodeConfigB.DA.AnyTrust.RestAggregator.Urls = []string{restURL}

	l2B, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{
		nodeConfig:   nodeConfigB,
		initData:     &builder.L2Info.ArbInitData,
		fatalErrChan: nodeBFatalErrChan,
	})
	defer cleanupB()

	waitForNodeBatchCount(t, l2B.ConsensusNode, sequencerBatchCount, nodeBFatalErrChan, 30*time.Second)

	checkBatchPosting(t, ctx, builder, l2B.Client)

	// Window > inbox reader poll cadence so a trailing fatal can't slip past the check.
	select {
	case err := <-nodeBFatalErrChan:
		t.Fatalf("node B unexpectedly fataled: %v", err)
	case <-time.After(2 * time.Second):
	}
}

func waitForNodeBatchCount(t *testing.T, node *arbnode.Node, target uint64, fatalErrChan <-chan error, timeout time.Duration) {
	t.Helper()
	pcds := node.GetParentChainDataSource()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case err := <-fatalErrChan:
			t.Fatalf("node fataled while waiting for historic batch %d: %v", target, err)
		default:
		}
		count, err := pcds.GetBatchCount()
		Require(t, err)
		if count >= target {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("node did not reach batch count %d within %s", target, timeout)
}

func TestDangerousAlwaysFallback_RejectsMessageExtractionCombo(t *testing.T) {
	cfg := arbnode.ConfigDefaultL1Test()
	cfg.Dangerous.AlwaysFallbackToParentChainDA = true
	cfg.MessageExtraction.Enable = true

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "message-extraction.enable=true") {
		t.Fatalf("expected error to mention message-extraction.enable=true, got: %v", err)
	}
}

func TestDangerousAlwaysFallback_RejectsAnyTrustDisabled(t *testing.T) {
	cfg := arbnode.ConfigDefaultL1Test()
	cfg.Dangerous.AlwaysFallbackToParentChainDA = true
	cfg.DA.AnyTrust.Enable = false

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "da.anytrust.enable=true") {
		t.Fatalf("expected error to mention da.anytrust.enable=true, got: %v", err)
	}
}

func TestDangerousAlwaysFallback_RejectsBlockValidatorWithoutRestAggregator(t *testing.T) {
	cfg := arbnode.ConfigDefaultL1Test()
	cfg.Dangerous.AlwaysFallbackToParentChainDA = true
	cfg.DA.AnyTrust.Enable = true
	cfg.DA.AnyTrust.RestAggregator.Enable = false
	cfg.BlockValidator.Enable = true

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "block-validator.enable=true") {
		t.Fatalf("expected error to mention block-validator.enable=true, got: %v", err)
	}
}

func assertNoAnyTrustBatchesOnL1(t *testing.T, ctx context.Context, builder *NodeBuilder) {
	t.Helper()
	pcds := builder.L2.ConsensusNode.GetParentChainDataSource()
	batchCount, err := pcds.GetBatchCount()
	Require(t, err)
	if batchCount == 0 {
		t.Fatal("no batches posted")
	}
	var fallbackPayloadBatches int
	for seqNum := uint64(0); seqNum < batchCount; seqNum++ {
		batchData, _, err := pcds.GetSequencerMessageBytes(ctx, seqNum)
		Require(t, err)
		if len(batchData) <= 40 {
			continue
		}
		headerByte := batchData[40]
		if daprovider.IsAnyTrustMessageHeaderByte(headerByte) {
			t.Fatalf("batch %d posted with AnyTrust header byte 0x%02x", seqNum, headerByte)
		}
		if daprovider.IsBlobHashesHeaderByte(headerByte) || daprovider.IsBrotliMessageHeaderByte(headerByte) {
			fallbackPayloadBatches++
		}
	}
	if fallbackPayloadBatches == 0 {
		t.Fatal("no batch carried a calldata (brotli) or blob payload; cannot confirm fallback to parent-chain DA happened")
	}
}

func waitForAnyTrustBatchOnL1(t *testing.T, ctx context.Context, builder *NodeBuilder, timeout time.Duration) {
	t.Helper()
	pcds := builder.L2.ConsensusNode.GetParentChainDataSource()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		batchCount, err := pcds.GetBatchCount()
		Require(t, err)
		for seqNum := uint64(0); seqNum < batchCount; seqNum++ {
			batchData, _, err := pcds.GetSequencerMessageBytes(ctx, seqNum)
			Require(t, err)
			if len(batchData) > 40 && daprovider.IsAnyTrustMessageHeaderByte(batchData[40]) {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("no AnyTrust batch posted within timeout")
}
