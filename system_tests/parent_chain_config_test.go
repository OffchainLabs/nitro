// Copyright 2026, Offchain Labs, Inc.
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

	// Pick a BPO1 activation far enough in the future that Build()'s setup
	// blocks won't already cross it, and drive L1 past it explicitly later.
	// The pointer is set once before Build() and never mutated again, so we
	// don't depend on geth retaining the same struct pointer we passed in.
	// #nosec G115
	bpo1Time := uint64(time.Now().Unix()) + 600
	l1ChainConfig := *params.AllDevChainProtocolChanges
	l1ChainConfig.BPO1Time = &bpo1Time
	l1ChainConfig.BlobScheduleConfig = &params.BlobScheduleConfig{
		Cancun: params.DefaultCancunBlobConfig,
		Prague: params.DefaultPragueBlobConfig,
		Osaka:  params.DefaultOsakaBlobConfig,
		BPO1:   params.DefaultBPO1BlobConfig,
	}

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).WithL1ChainConfig(&l1ChainConfig)
	cleanup := builder.Build(t)
	defer cleanup()

	l1HeaderReader, err := headerreader.New(
		ctx,
		builder.L1.Client,
		func() *headerreader.Config { return &headerreader.TestConfig },
		nil,
	)
	Require(t, err)
	l1HeaderReader.Start(ctx)
	defer l1HeaderReader.StopAndWait()

	testConfig := parent.Config{ConfigPollInterval: 200 * time.Millisecond}
	pc := parent.NewParentChainWithConfig(
		ctx,
		l1ChainConfig.ChainID,
		l1HeaderReader,
		func() *parent.Config { return &testConfig },
	)
	pc.Start(ctx)
	defer pc.StopAndWait()

	logBlobConfig := func(phase string, cfg, expect *params.BlobConfig) {
		if cfg == nil {
			t.Logf("%s: cached blob config is nil (expected target=%d max=%d)", phase, expect.Target, expect.Max)
			return
		}
		t.Logf("%s: cached blob config target=%d max=%d updateFraction=%d (expected target=%d max=%d)",
			phase, cfg.Target, cfg.Max, cfg.UpdateFraction, expect.Target, expect.Max)
	}

	var lastPhase1Cfg *params.BlobConfig
	defer func() {
		logBlobConfig("Phase 1 (Osaka)", lastPhase1Cfg, params.DefaultOsakaBlobConfig)
	}()
	pollUntil(t, ctx, 15*time.Second, 200*time.Millisecond, "initial Osaka blob config", func() bool {
		lastPhase1Cfg = pc.CachedBlobConfig()
		return lastPhase1Cfg != nil &&
			lastPhase1Cfg.Target == params.DefaultOsakaBlobConfig.Target &&
			lastPhase1Cfg.Max == params.DefaultOsakaBlobConfig.Max
	})

	pollUntil(t, ctx, 60*time.Second, 100*time.Millisecond, "L1 timestamp past BPO1Time", func() bool {
		head, err := builder.L1.Client.BlockByNumber(ctx, nil)
		if err != nil {
			t.Logf("BlockByNumber failed: %v", err)
			return false
		}
		if head.Time() >= bpo1Time {
			return true
		}
		AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 1)
		return false
	})

	var lastPhase2Cfg *params.BlobConfig
	defer func() {
		logBlobConfig("Phase 2 (BPO1)", lastPhase2Cfg, params.DefaultBPO1BlobConfig)
	}()
	pollUntil(t, ctx, 30*time.Second, 200*time.Millisecond, "blob config transition to BPO1", func() bool {
		lastPhase2Cfg = pc.CachedBlobConfig()
		return lastPhase2Cfg != nil &&
			lastPhase2Cfg.Target == params.DefaultBPO1BlobConfig.Target &&
			lastPhase2Cfg.Max == params.DefaultBPO1BlobConfig.Max
	})

	maxBlobGas, err := pc.MaxBlobGasPerBlock(ctx, nil)
	Require(t, err)
	// #nosec G115
	expectedMaxBlobGas := uint64(params.DefaultBPO1BlobConfig.Max) * params.BlobTxBlobGasPerBlob
	if maxBlobGas != expectedMaxBlobGas {
		t.Errorf("MaxBlobGasPerBlock after BPO1: got %d, want %d", maxBlobGas, expectedMaxBlobGas)
	}
}
