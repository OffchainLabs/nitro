// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"context"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/cmd/nitro/config"
	"github.com/offchainlabs/nitro/cmd/nitro/init"
	"github.com/offchainlabs/nitro/execution/gethexec"
)

func TestGetParsedInitMsgFromParentChain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// We need L1 to get builder.initMessage
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	initMessage, err := nitroinit.GetParsedInitMsgFromParentChain(ctx, builder.chainConfig.ChainID, builder.L1.Client, builder.addresses)
	Require(t, err)

	if success := reflect.DeepEqual(initMessage, builder.initMessage); !success {
		t.Fatalf("diff found in initMessage %v and builder.initMessage: %v", initMessage, builder.initMessage)
	}
}

func TestOpenExistingExecutionDB(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.l2StackConfig.DBEngine = rawdb.DBPebble
	builder.l2StackConfig.Name = "arb-init-test-l2"
	builder.execConfig.Caching.StateScheme = rawdb.PathScheme
	_ = builder.Build(t)

	builder.L2Info.GenerateAccount("User2")
	var txs []*types.Transaction
	for i := uint64(0); i < 51; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, common.Big1, nil)
		txs = append(txs, tx)
	}
	receipts := builder.L2.SendWaitTestTransactions(t, txs)

	builder.L2.cleanup()
	t.Log("stopped L2 node")

	stack, err := node.New(builder.l2StackConfig)
	Require(t, err)
	defer func() {
		cancel()
		builder.L1.cleanup()
		builder.ctxCancel()
		stack.Close()
	}()

	nodeConfig := config.NodeConfigDefault
	nodeConfig.Execution.Caching.StateScheme = rawdb.PathScheme
	nodeConfig.Chain.ID = builder.chainConfig.ChainID.Uint64()
	nodeConfig.Node = *builder.nodeConfig

	executionDB, _, _, _, err := nitroinit.OpenExistingExecutionDB(
		stack,
		&nodeConfig,
		new(big.Int).SetUint64(nodeConfig.Chain.ID),
		gethexec.DefaultCacheConfigFor(&nodeConfig.Execution.Caching),
		nil,
		&nodeConfig.Persistent,
	)
	Require(t, err)

	// Get a receipt from an arbitrary transaction to make sure executionDB contains the correct information
	targetReceipt := receipts[40]
	blockHash := rawdb.ReadCanonicalHash(executionDB, targetReceipt.BlockNumber.Uint64())
	if blockHash != targetReceipt.BlockHash {
		t.Fatalf("Expected block hash: %s does not match canonical rawdb block hash: %s", targetReceipt.BlockHash.Hex(), blockHash.Hex())
	}
}
