// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode/parent"
	"github.com/offchainlabs/nitro/util/headerreader"
)

func TestParentChainEthConfigPolling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	// Create a header reader connected to the L1
	l1Client := builder.L1.Client
	l1HeaderReader, err := headerreader.New(
		ctx,
		l1Client,
		func() *headerreader.Config { return &headerreader.TestConfig },
		nil,
	)
	Require(t, err)
	l1HeaderReader.Start(ctx)
	defer l1HeaderReader.StopAndWait()

	// Create a ParentChain with a short poll interval for testing
	testConfig := parent.Config{ConfigPollInterval: 100 * time.Millisecond}
	pc := parent.NewParentChainWithConfig(
		ctx,
		big.NewInt(1337), // test L1 chain ID
		l1HeaderReader,
		func() *parent.Config { return &testConfig },
	)
	pc.Start(ctx)
	defer pc.StopAndWait()

	// Wait for the poller to fetch the config
	var blobConfig *params.BlobConfig
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		blobConfig = pc.CachedBlobConfig()
		if blobConfig != nil {
			break
		}
	}

	if blobConfig == nil {
		t.Fatal("ParentChain did not fetch blob config from L1 eth_config within timeout")
	}

	// The test L1 uses AllDevChainProtocolChanges which has the Osaka blob config
	expectedBlobConfig := params.DefaultOsakaBlobConfig
	if blobConfig.Target != expectedBlobConfig.Target {
		t.Errorf("blob config target mismatch: got %d, want %d", blobConfig.Target, expectedBlobConfig.Target)
	}
	if blobConfig.Max != expectedBlobConfig.Max {
		t.Errorf("blob config max mismatch: got %d, want %d", blobConfig.Max, expectedBlobConfig.Max)
	}
	if blobConfig.UpdateFraction != expectedBlobConfig.UpdateFraction {
		t.Errorf("blob config update fraction mismatch: got %d, want %d", blobConfig.UpdateFraction, expectedBlobConfig.UpdateFraction)
	}

	// Verify that MaxBlobGasPerBlock uses the cached config
	maxBlobGas, err := pc.MaxBlobGasPerBlock(ctx, nil)
	Require(t, err)
	// #nosec G115
	expectedMaxBlobGas := uint64(expectedBlobConfig.Max) * params.BlobTxBlobGasPerBlob
	if maxBlobGas != expectedMaxBlobGas {
		t.Errorf("MaxBlobGasPerBlock mismatch: got %d, want %d", maxBlobGas, expectedMaxBlobGas)
	}
}

// TestParentChainEthConfigForkTransition verifies that the ParentChain poller
// detects when the parent chain transitions through a fork that changes the
// blob schedule (e.g., Osaka -> BPO1).
func TestParentChainEthConfigForkTransition(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a custom L1 chain config: all forks active at genesis except BPO1
	// far in the future. The pointer is shared with the geth node's config so we
	// can mutate it later to simulate a fork activation without restarting the node
	// #nosec G115
	farFuture := uint64(time.Now().Unix()) + 60
	l1ChainConfig := *params.AllDevChainProtocolChanges
	l1ChainConfig.BPO1Time = &farFuture
	l1ChainConfig.BlobScheduleConfig = &params.BlobScheduleConfig{
		Cancun: params.DefaultCancunBlobConfig,
		Prague: params.DefaultPragueBlobConfig,
		Osaka:  params.DefaultOsakaBlobConfig,
		BPO1:   params.DefaultBPO1BlobConfig,
	}

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).WithL1ChainConfig(&l1ChainConfig)
	cleanup := builder.Build(t)
	defer cleanup()

	// Create a header reader connected to the L1
	l1Client := builder.L1.Client
	l1HeaderReader, err := headerreader.New(
		ctx,
		l1Client,
		func() *headerreader.Config { return &headerreader.TestConfig },
		nil,
	)
	Require(t, err)
	l1HeaderReader.Start(ctx)
	defer l1HeaderReader.StopAndWait()

	// Create a ParentChain with fast polling
	testConfig := parent.Config{ConfigPollInterval: 200 * time.Millisecond}
	pc := parent.NewParentChainWithConfig(
		ctx,
		l1ChainConfig.ChainID,
		l1HeaderReader,
		func() *parent.Config { return &testConfig },
	)
	pc.Start(ctx)
	defer pc.StopAndWait()

	// Phase 1: Verify initial config is Osaka (BPO1 is far in the future)
	var blobConfigPhase1 *params.BlobConfig
	for i := 0; i < 50; i++ {
		time.Sleep(200 * time.Millisecond)
		blobConfigPhase1 = pc.CachedBlobConfig()
		if blobConfigPhase1 != nil {
			break
		}
	}
	if blobConfigPhase1 == nil {
		t.Fatal("ParentChain did not fetch initial blob config within timeout")
	}

	// Phase 2: Activate BPO1 by advancing L1
	go keepChainMoving(t, 100*time.Millisecond, ctx, builder.L1Info, builder.L1.Client)

	t.Logf("Phase 1: got initial blob config target=%d max=%d (expecting Osaka: target=%d max=%d)",
		blobConfigPhase1.Target, blobConfigPhase1.Max,
		params.DefaultOsakaBlobConfig.Target, params.DefaultOsakaBlobConfig.Max)

	if blobConfigPhase1.Target != params.DefaultOsakaBlobConfig.Target ||
		blobConfigPhase1.Max != params.DefaultOsakaBlobConfig.Max {
		t.Fatalf("initial blob config should be Osaka, got target=%d max=%d",
			blobConfigPhase1.Target, blobConfigPhase1.Max)
	}

	var blobConfigPhase2 *params.BlobConfig
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		blobConfigPhase2 = pc.CachedBlobConfig()
		if blobConfigPhase2 != nil && blobConfigPhase2.Target == params.DefaultBPO1BlobConfig.Target {
			break
		}
	}

	// Now we verify the activated fork was indeed BPO1
	if blobConfigPhase2 == nil ||
		blobConfigPhase2.Target != params.DefaultBPO1BlobConfig.Target ||
		blobConfigPhase2.Max != params.DefaultBPO1BlobConfig.Max {
		t.Fatalf("blob config did not transition to BPO1: got target=%d max=%d, want target=%d max=%d",
			blobConfigPhase2.Target, blobConfigPhase2.Max,
			params.DefaultBPO1BlobConfig.Target, params.DefaultBPO1BlobConfig.Max)
	}

	t.Logf("Phase 2: blob config transitioned to BPO1 target=%d max=%d updateFraction=%d",
		blobConfigPhase2.Target, blobConfigPhase1.Max, blobConfigPhase2.UpdateFraction)

	// Also verify MaxBlobGasPerBlock reflects the new config
	maxBlobGas, err := pc.MaxBlobGasPerBlock(ctx, nil)
	Require(t, err)
	// #nosec G115
	expectedMaxBlobGas := uint64(params.DefaultBPO1BlobConfig.Max) * params.BlobTxBlobGasPerBlob
	if maxBlobGas != expectedMaxBlobGas {
		t.Errorf("MaxBlobGasPerBlock after BPO1: got %d, want %d", maxBlobGas, expectedMaxBlobGas)
	}
}
