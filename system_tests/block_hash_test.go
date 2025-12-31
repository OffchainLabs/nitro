// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
)

func testBlockHash(t *testing.T, executionClientMode ExecutionClientMode) {
	ctx := t.Context()

	// Even though we don't use the L1, we need to create this node on L1 to get accurate L1 block numbers
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Faucet", ctx)

	replicaConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	replicaParams := &SecondNodeParams{
		nodeConfig:             replicaConfig,
		useExecutionClientOnly: true,
		executionClientMode:    executionClientMode,
	}

	replicaTestClient, replicaCleanup := builder.Build2ndNode(t, replicaParams)
	defer replicaCleanup()
	replicaClient := replicaTestClient.Client

	// Deploy on primary
	contractAddr, tx, _, err := localgen.DeploySimple(&auth, builder.L2.Client)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Wait for sync to replica
	_, err = WaitForTx(ctx, replicaClient, tx.Hash(), time.Second*15)
	Require(t, err)

	// Test on replica
	simpleOnReplica, err := localgen.NewSimple(contractAddr, replicaClient)
	Require(t, err)
	_, err = simpleOnReplica.CheckBlockHashes(&bind.CallOpts{Context: ctx})
	Require(t, err)
}

func TestBlockHashInternal(t *testing.T) {
	testBlockHash(t, ExecutionClientModeInternal)
}

func TestBlockHashExternal(t *testing.T) {
	testBlockHash(t, ExecutionClientModeExternal)
}

func TestBlockHashComparison(t *testing.T) {
	testBlockHash(t, ExecutionClientModeComparison)
}
