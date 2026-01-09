package arbtest

import (
	"bytes"
	"context"
	"encoding/binary"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
)

func testHistoricalBlockHash(t *testing.T, executionClientMode ExecutionClientMode) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	// Build replica with specified execution mode
	replicaConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	replicaParams := &SecondNodeParams{
		nodeConfig:             replicaConfig,
		useExecutionClientOnly: true,
		executionClientMode:    executionClientMode,
	}

	replicaTestClient, replicaCleanup := builder.Build2ndNode(t, replicaParams)
	defer replicaCleanup()
	replicaClient := replicaTestClient.Client

	// Wait for replica to initialize
	time.Sleep(time.Second * 2)

	// Generate 300+ blocks on primary
	for {
		builder.L2.TransferBalance(t, "Faucet", "Faucet", common.Big1, builder.L2Info)
		number, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
		if number > 300 {
			break
		}
	}

	// Get current block from primary
	block, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)

	// Wait for replica to catch up to primary
	for i := 0; i < 60; i++ {
		replicaBlock, err := replicaClient.BlockNumber(ctx)
		Require(t, err)
		if replicaBlock >= block {
			break
		}
		time.Sleep(time.Second)
	}

	// Verify replica caught up
	replicaBlock, err := replicaClient.BlockNumber(ctx)
	Require(t, err)
	if replicaBlock < block {
		t.Fatalf("Replica failed to sync: primary at block %d, replica at block %d", block, replicaBlock)
	}

	// Test historical block hashes on replica
	for i := uint64(0); i < replicaBlock; i++ {
		var key common.Hash
		binary.BigEndian.PutUint64(key[24:], i)
		expectedBlock, err := replicaClient.BlockByNumber(ctx, new(big.Int).SetUint64(i))
		Require(t, err)
		blockHash := sendContractCall(t, ctx, params.HistoryStorageAddress, replicaClient, key.Bytes())
		if !bytes.Equal(blockHash, expectedBlock.Hash().Bytes()) {
			t.Fatalf("Expected block hash %s, got %s for block %d", expectedBlock.Hash(), blockHash, i)
		}
	}
}

func TestHistoricalBlockHashInternal(t *testing.T) {
	testHistoricalBlockHash(t, ExecutionClientModeInternal)
}

func TestHistoricalBlockHashExternal(t *testing.T) {
	testHistoricalBlockHash(t, ExecutionClientModeExternal)
}

func TestHistoricalBlockHashComparison(t *testing.T) {
	testHistoricalBlockHash(t, ExecutionClientModeComparison)
}
