// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/testhelpers/github"
	"github.com/offchainlabs/nitro/validator/client/redis"
)

func TestStylusStorageFlushValidation_ArbOS40_ReplayWasm(t *testing.T) {
	runFlushValidation(t, params.ArbosVersion_40, "")
}

func TestStylusStorageFlushValidation_ArbOS51_ReplayWasm(t *testing.T) {
	runFlushValidation(t, params.ArbosVersion_51, "")
}

func TestStylusStorageFlushValidation_ArbOS51_ConsensusV51(t *testing.T) {
	cr, err := github.LatestConsensusRelease(t.Context())
	Require(t, err)
	if cr.ArbosVersion < params.ArbosVersion_51 {
		Fatal(t, "LatestConsensusRelease version is below ArbOS 51", "got", cr.ArbosVersion, "want", params.ArbosVersion_51)
	}
	runFlushValidation(t, params.ArbosVersion_51, populateMachineDir(t, cr))
}

func runFlushValidation(t *testing.T, arbosVersion uint64, wasmRootDir string) {
	t.Helper()
	ctx := t.Context()
	builder, valClient := setupValidatedPair(t, ctx, arbosVersion, wasmRootDir)
	blocks := runStylusStorageFlushAndCollectBlocks(t, builder)
	waitValidateUpToCollected(t, ctx, valClient, blocks)
}

func setupValidatedPair(t *testing.T, ctx context.Context, arbosVersion uint64, wasmRootDir string) (*NodeBuilder, *TestClient) {
	t.Helper()

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithArbOSVersion(arbosVersion).
		WithWasmRootDir(wasmRootDir)

	// For now PathDB is not supported when using block validation
	builder.RequireScheme(t, rawdb.HashScheme)

	cleanup := builder.Build(t)
	t.Cleanup(cleanup)

	validatorConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	validatorConfig.BlockValidator.Enable = true
	validatorConfig.DA.AnyTrust = builder.nodeConfig.DA.AnyTrust
	validatorConfig.DA.AnyTrust.RPCAggregator.Enable = false
	validatorConfig.BlockValidator.RedisValidationClientConfig = redis.ValidationClientConfig{}

	AddValNode(t, ctx, validatorConfig, false, "", wasmRootDir)

	valClient, cleanupValClient := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: validatorConfig})
	t.Cleanup(cleanupValClient)

	return builder, valClient
}

func waitValidateUpToCollected(t *testing.T, ctx context.Context, valClient *TestClient, blocks []*types.Block) {
	t.Helper()

	if len(blocks) == 0 {
		Fatal(t, "no blocks collected")
	}

	var maxBlock uint64
	for _, b := range blocks {
		if n := b.NumberU64(); n > maxBlock {
			maxBlock = n
		}
	}

	timeout := getDeadlineTimeout(t, time.Minute*10)
	if !valClient.ConsensusNode.BlockValidator.WaitForPos(t, ctx, arbutil.MessageIndex(maxBlock), timeout) {
		Fatal(t, "did not validate all blocks up to", "block", maxBlock)
	}
}

func runStylusStorageFlushAndCollectBlocks(t *testing.T, builder *NodeBuilder) []*types.Block {
	t.Helper()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", builder.ctx)
	stylusProgram := deployWasm(t, builder.ctx, auth, builder.L2.Client, rustFile("multicall"))

	const storeHostio = "storage_flush_cache"
	const emitLog, notFlush = false, true

	fillHash := func(b byte) common.Hash {
		var h common.Hash
		for i := range h {
			h[i] = b
		}
		return h
	}

	slot := fillHash(0x22)
	buildStore := func(value byte) []byte {
		return multicallAppendStore(multicallEmptyArgs(), slot, fillHash(value), emitLog, notFlush)
	}

	waitReceipt := func(txHash common.Hash) *types.Receipt {
		for range 200 {
			receipt, err := builder.L2.Client.TransactionReceipt(builder.ctx, txHash)
			if err == nil && receipt != nil {
				return receipt
			}
			time.Sleep(50 * time.Millisecond)
		}
		Fatal(t, "no receipt for tx", txHash)
		return nil
	}

	// Warm the slot so SSTORE pricing is deterministic across cases.
	warmTx := builder.L2Info.PrepareTxTo("Owner", &stylusProgram, 32_000_000, nil, buildStore(0x30))
	Require(t, builder.L2.Client.SendTransaction(builder.ctx, warmTx))
	_, err := builder.L2.EnsureTxSucceeded(warmTx)
	Require(t, err)

	cases := []struct {
		name        string
		gasLimit    uint64
		shouldFlush bool
	}{
		{name: "success_1", gasLimit: 200_000, shouldFlush: true},
		{name: "success_2", gasLimit: 200_000, shouldFlush: true},
		{name: "success_3", gasLimit: 200_000, shouldFlush: true},
		{name: "oog_1", gasLimit: 38_000, shouldFlush: false},
		{name: "oog_2", gasLimit: 38_000, shouldFlush: false},
		{name: "oog_3", gasLimit: 38_000, shouldFlush: false},
	}

	blocks := make([]*types.Block, 0, len(cases))
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tx := builder.L2Info.PrepareTxTo("Owner", &stylusProgram, c.gasLimit, nil, buildStore(byte(0x31+i)))
			Require(t, builder.L2.Client.SendTransaction(builder.ctx, tx), "case", c.name)

			receipt := waitReceipt(tx.Hash())

			wantStatus := types.ReceiptStatusFailed
			if c.shouldFlush {
				wantStatus = types.ReceiptStatusSuccessful
			}
			if receipt.Status != wantStatus {
				Fatal(t, "unexpected tx status", "case", c.name, "want", wantStatus, "got", receipt.Status)
			}

			stylusInkUsage, err := stylusHostiosInkUsage(builder.ctx, builder.L2.Client.Client(), tx)
			Require(t, err, "case", c.name)
			_, gotFlush := stylusInkUsage[storeHostio]
			if gotFlush != c.shouldFlush {
				Fatal(t, "unexpected flush presence", "case", c.name, "wantFlush", c.shouldFlush, "gotFlush", gotFlush)
			}

			block, err := builder.L2.Client.BlockByHash(builder.ctx, receipt.BlockHash)
			Require(t, err, "case", c.name)
			blocks = append(blocks, block)
		})
	}
	return blocks
}
