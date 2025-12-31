package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/node"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/nitro/config"
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
	lastBlockNumber := receipts[len(receipts)-1].BlockNumber.Uint64()
	block, err := builder.L2.Client.BlockByNumber(ctx, nil)
	Require(t, err)
	deadline := time.After(5 * time.Second)
	// make sure we get the last block in case API has a delayed view
	for block.NumberU64() < lastBlockNumber {
		select {
		case <-time.After(20 * time.Millisecond):
			block, err = builder.L2.Client.BlockByNumber(ctx, nil)
			Require(t, err)
		case <-deadline:
			t.Fatal("deadline exceeded while waiting for last block")
		}
	}

	return builder, receipts
}

func TestGetConsensusParsedInitMsgNoParentChain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	builder, _ := prepareL1AndL2(t, ctx, false)

	defer func() {
		cancel()
		builder.L2.cleanup()
		builder.ctxCancel()
	}()

	nodeConfig := config.NodeConfigDefault
	nodeConfig.Node.ParentChainReader.Enable = false
	initMessage, err := config.GetConsensusParsedInitMsg(ctx, &nodeConfig, builder.chainConfig.ChainID, nil, chaininfo.RollupAddresses{}, builder.chainConfig)
	Require(t, err)

	if initMessage.InitialL1BaseFee.Cmp(arbostypes.DefaultInitialL1BaseFee) != 0 {
		t.Fatalf("initMessage InitialL1BaseFee: %d, does not match expected DefaultInitialL1BaseFee: %d", initMessage.InitialL1BaseFee, arbostypes.DefaultInitialL1BaseFee)
	}
}

func TestGetConsensusParsedInitMsg(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	builder, _ := prepareL1AndL2(t, ctx, true)

	defer func() {
		cancel()
		builder.L2.cleanup()
		builder.L1.cleanup()
		builder.ctxCancel()
	}()

	nodeConfig := config.NodeConfigDefault
	nodeConfig.Node.ParentChainReader.Enable = true
	initMessage, err := config.GetConsensusParsedInitMsg(ctx, &nodeConfig, builder.chainConfig.ChainID, builder.L1.Client, *builder.addresses, builder.chainConfig)
	Require(t, err)

	if initMessage.InitialL1BaseFee.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("initMessage InitialL1BaseFee: %d, does not match expected DefaultInitialL1BaseFee: %d", initMessage.InitialL1BaseFee, arbostypes.DefaultInitialL1BaseFee)
	}
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
	nodeConfig.Init.DevInit = true
	nodeConfig.Init.DevInitAddress = "0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E"
	nodeConfig.Init.ValidateGenesisAssertion = false

	executionDB, _, _, _, err := config.OpenExistingExecutionDB(
		stack,
		&nodeConfig,
		new(big.Int).SetUint64(nodeConfig.Chain.ID),
		gethexec.DefaultCacheConfigFor(&nodeConfig.Execution.Caching),
		nil,
		&nodeConfig.Persistent,
	)
	Require(t, err)

	// Get a receipt from a random transaction to make sure executionDB contains the correct information
	targetReceipt := receipts[40]
	blockHash := rawdb.ReadCanonicalHash(executionDB, targetReceipt.BlockNumber.Uint64())
	if blockHash != targetReceipt.BlockHash {
		t.Fatalf("Expected block hash: %s does not match canonical rawdb block hash: %s", targetReceipt.BlockHash.Hex(), blockHash.Hex())
	}
}
