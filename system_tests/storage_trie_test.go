// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func TestStorageTrie(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var withL1 = true
	builder := NewNodeBuilder(ctx).DefaultConfig(t, withL1)

	// This test tests validates blocks at the end.
	// For now, validation only works with HashScheme set.
	builder.RequireScheme(t, rawdb.HashScheme)
	builder.nodeConfig.BlockValidator.Enable = false
	builder.nodeConfig.Staker.Enable = true
	builder.nodeConfig.BatchPoster.Enable = true
	builder.nodeConfig.ParentChainReader.Enable = true
	builder.nodeConfig.ParentChainReader.OldHeaderTimeout = 10 * time.Minute

	valConf := valnode.TestValidationConfig
	valConf.UseJit = true
	_, valStack := createTestValidationNode(t, ctx, &valConf)
	configByValidationNode(builder.nodeConfig, valStack)

	cleanup := builder.Build(t)
	defer cleanup()

	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	_, bigMap := builder.L2.DeployBigMap(t, ownerTxOpts)

	// Store enough values to use just under 32M gas
	toAdd := big.NewInt(1420)
	// In the first transaction, don't clear any values.
	toClear := big.NewInt(0)

	userTxOpts := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)
	tx, err := bigMap.ClearAndAddValues(&userTxOpts, toClear, toAdd)
	Require(t, err)

	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	tx1BlockNum := receipt.BlockNumber.Uint64()

	want := uint64(31_900_000)
	got := receipt.GasUsedForL2()
	if got < want {
		t.Errorf("Want at least GasUsed: %d: got: %d", want, got)
	}

	// Clear about 75% of them, and add another 10%
	toClear = arbmath.BigDiv(arbmath.BigMul(toAdd, big.NewInt(75)), big.NewInt(100))
	toAdd = arbmath.BigDiv(arbmath.BigMul(toAdd, big.NewInt(10)), big.NewInt(100))

	tx, err = bigMap.ClearAndAddValues(&userTxOpts, toClear, toAdd)
	Require(t, err)

	receipt, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	tx2BlockNum := receipt.BlockNumber.Uint64()

	if tx2BlockNum <= tx1BlockNum {
		t.Errorf("Expected tx2BlockNum > tx1BlockNum: %d <= %d", tx2BlockNum, tx1BlockNum)
	} else {
		t.Logf("tx2BlockNum > tx1BlockNum: %d > %d", tx2BlockNum, tx1BlockNum)
	}

	// Ensures that the validator gets the same results as the executor
	validateBlockRange(t, []uint64{receipt.BlockNumber.Uint64()}, true, builder)
}
