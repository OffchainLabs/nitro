// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/staker"
)

func TestBlockValidatorRejectsInvalidWasmModuleRoot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	// Send a transaction so there are blocks to validate
	builder.L2Info.GenerateAccount("User2")
	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err := builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	_, valStack := createMockValidationNode(t, ctx, nil)
	blockValidatorConfig := staker.TestBlockValidatorConfig

	stateless, err := staker.NewStatelessBlockValidator(
		builder.L2.ConsensusNode.InboxReader,
		builder.L2.ConsensusNode.InboxTracker,
		builder.L2.ConsensusNode.TxStreamer,
		builder.L2.ExecNode,
		builder.L2.ConsensusNode.ConsensusDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
		mockWasmModuleRoots[0],
	)
	Require(t, err)
	err = stateless.Start(ctx)
	Require(t, err)

	blockValidator, err := staker.NewBlockValidator(
		stateless,
		builder.L2.ConsensusNode.InboxTracker,
		builder.L2.ConsensusNode.TxStreamer,
		StaticFetcherFrom(t, &blockValidatorConfig),
		nil,
	)
	Require(t, err)

	// Set an initial WasmModuleRoot (simulates normal startup where the on-chain
	// root matches one of the validator's known roots)
	err = blockValidator.SetCurrentWasmModuleRoot(mockWasmModuleRoots[0])
	Require(t, err)

	// Now try to set a completely different, unexpected WasmModuleRoot.
	// This simulates the on-chain rollup's WasmModuleRoot changing to a value
	// the validator doesn't recognize as either current or pending.
	invalidRoot := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	err = blockValidator.SetCurrentWasmModuleRoot(invalidRoot)
	if err == nil {
		Fatal(t, "expected error when setting unexpected WasmModuleRoot, but got nil")
	}
	t.Logf("Got expected error for mismatched WasmModuleRoot: %v", err)
}

func TestBlockValidatorRejectsZeroWasmModuleRoot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	_, valStack := createMockValidationNode(t, ctx, nil)
	blockValidatorConfig := staker.TestBlockValidatorConfig

	stateless, err := staker.NewStatelessBlockValidator(
		builder.L2.ConsensusNode.InboxReader,
		builder.L2.ConsensusNode.InboxTracker,
		builder.L2.ConsensusNode.TxStreamer,
		builder.L2.ExecNode,
		builder.L2.ConsensusNode.ConsensusDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
		mockWasmModuleRoots[0],
	)
	Require(t, err)
	err = stateless.Start(ctx)
	Require(t, err)

	blockValidator, err := staker.NewBlockValidator(
		stateless,
		builder.L2.ConsensusNode.InboxTracker,
		builder.L2.ConsensusNode.TxStreamer,
		StaticFetcherFrom(t, &blockValidatorConfig),
		nil,
	)
	Require(t, err)

	// Setting a zero WasmModuleRoot should always be rejected
	err = blockValidator.SetCurrentWasmModuleRoot(common.Hash{})
	if err == nil {
		Fatal(t, "expected error when setting zero WasmModuleRoot, but got nil")
	}
	t.Logf("Got expected error for zero WasmModuleRoot: %v", err)
}

func TestStatelessBlockValidatorRejectsZeroWasmModuleRoot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	_, valStack := createMockValidationNode(t, ctx, nil)
	blockValidatorConfig := staker.TestBlockValidatorConfig

	// Creating a StatelessBlockValidator with a zero WasmModuleRoot should
	// fail at construction time.
	_, err := staker.NewStatelessBlockValidator(
		builder.L2.ConsensusNode.InboxReader,
		builder.L2.ConsensusNode.InboxTracker,
		builder.L2.ConsensusNode.TxStreamer,
		builder.L2.ExecNode,
		builder.L2.ConsensusNode.ConsensusDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
		common.Hash{}, // zero WasmModuleRoot
	)
	if err == nil {
		Fatal(t, "expected error when creating StatelessBlockValidator with zero WasmModuleRoot, but got nil")
	}
	t.Logf("Got expected error for zero WasmModuleRoot at construction: %v", err)
}
