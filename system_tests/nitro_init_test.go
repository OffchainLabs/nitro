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

	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/nitro/config"
	"github.com/offchainlabs/nitro/cmd/nitro/init"
	"github.com/offchainlabs/nitro/execution/gethexec"
)

func prepareL1AndL2(t *testing.T, ctx context.Context, withL1 bool) (*NodeBuilder, []*types.Receipt) {
	builder := NewNodeBuilder(ctx).DefaultConfig(t, withL1)
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

	return builder, receipts
}

func TestGetConsensusParsedInitMsgNoParentChain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	nodeConfig := config.NodeConfigDefault
	nodeConfig.Node.ParentChainReader.Enable = false
	initMessage, err := nitroinit.GetConsensusParsedInitMsg(ctx, nodeConfig.Node.ParentChainReader.Enable, builder.chainConfig.ChainID, nil, chaininfo.RollupAddresses{}, builder.chainConfig)
	Require(t, err)

	reflect.DeepEqual(initMessage, builder.initMessage)
}

func TestGetConsensusParsedInitMsg(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	nodeConfig := config.NodeConfigDefault
	nodeConfig.Node.ParentChainReader.Enable = true
	initMessage, err := nitroinit.GetConsensusParsedInitMsg(ctx, nodeConfig.Node.ParentChainReader.Enable, builder.chainConfig.ChainID, builder.L1.Client, *builder.addresses, builder.chainConfig)
	Require(t, err)

	reflect.DeepEqual(initMessage, builder.initMessage)
}

func TestOpenExistingExecutionDB(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	builder, receipts := prepareL1AndL2(t, ctx, true)

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
