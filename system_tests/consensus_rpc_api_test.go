// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
)

func getL1Confirmations(
	ctx context.Context,
	nodeInterface *node_interfacegen.NodeInterface,
	client *ethclient.Client,
	block *types.Block,
) (uint64, uint64, error) {
	l1ConfsNodeInterface, err := nodeInterface.GetL1Confirmations(&bind.CallOpts{}, block.Hash())
	if err != nil {
		return 0, 0, err
	}

	var l1ConfsRPC uint64
	err = client.Client().CallContext(ctx, &l1ConfsRPC, "arb_getL1Confirmations", block.Number())

	return l1ConfsNodeInterface, l1ConfsRPC, err
}

func testGetL1Confirmations(
	t *testing.T,
	ctx context.Context,
	childChainTestClient *TestClient,
	parentChainTestClient *TestClient,
	parentChainInfo info,
) {
	// Wait so ConsensusNode.L1Reader has some time to read parent chain headers,
	// which is needed for the RPC GetL1Confirmations call to work.
	time.Sleep(time.Second)

	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, childChainTestClient.Client)
	Require(t, err)

	genesisBlock, err := childChainTestClient.Client.BlockByNumber(ctx, big.NewInt(0))
	Require(t, err)

	l1ConfsNodeInterface, l1ConfsRPC, err := getL1Confirmations(ctx, nodeInterface, childChainTestClient.Client, genesisBlock)
	Require(t, err)

	numTransactions := 200

	// #nosec G115
	if l1ConfsNodeInterface >= uint64(numTransactions) || l1ConfsRPC >= uint64(numTransactions) {
		t.Fatalf("L1Confirmations for latest block %v is already l1ConfsNodeInterface=%v, l1ConfsRPC=%v, which is over %v",
			genesisBlock.Number(), l1ConfsNodeInterface, l1ConfsRPC, numTransactions)
	}

	for i := 0; i < numTransactions; i++ {
		parentChainTestClient.TransferBalance(t, "User", "User", common.Big0, parentChainInfo)
	}

	// wait a bit for the parent/grandparent chains to process the transactions
	time.Sleep(2 * time.Second)

	l1ConfsNodeInterface, l1ConfsRPC, err = getL1Confirmations(ctx, nodeInterface, childChainTestClient.Client, genesisBlock)
	Require(t, err)

	// Allow a gap of 10 for asynchronicity, just in case
	// #nosec G115
	if (l1ConfsNodeInterface+10 < uint64(numTransactions)) || (l1ConfsRPC+10 < uint64(numTransactions)) {
		t.Fatalf("L1Confirmations for latest block %v is only l1ConfsNodeInterface=%v, l1ConfsRPC=%v (did not hit expected %v)",
			genesisBlock.Number(), l1ConfsNodeInterface, l1ConfsRPC, numTransactions)
	}
}

func TestGetL1ConfirmationsForL2(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	testGetL1Confirmations(t, ctx, builder.L2, builder.L1, builder.L1Info)
}

func TestGetL1ConfirmationsForL3(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanupL1AndL2 := builder.Build(t)
	defer cleanupL1AndL2()

	cleanupL3 := builder.BuildL3OnL2(t)
	defer cleanupL3()

	testGetL1Confirmations(t, ctx, builder.L3, builder.L2, builder.L2Info)
}

func TestFindBatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).DontParalellise()
	l1Info := builder.L1Info
	initialBalance := new(big.Int).Lsh(big.NewInt(1), 200)
	l1Info.GenerateGenesisAccount("deployer", initialBalance)
	l1Info.GenerateGenesisAccount("asserter", initialBalance)
	l1Info.GenerateGenesisAccount("challenger", initialBalance)
	l1Info.GenerateGenesisAccount("sequencer", initialBalance)

	conf := builder.nodeConfig
	conf.BlockValidator.Enable = false
	conf.BatchPoster.Enable = false

	builder.BuildL1(t)

	bridgeAddr, seqInbox, seqInboxAddr := setupSequencerInboxStub(ctx, t, builder.L1Info, builder.L1.Client, builder.chainConfig)
	builder.addresses.Bridge = bridgeAddr
	builder.addresses.SequencerInbox = seqInboxAddr

	cleanup := builder.BuildL2OnL1(t)
	defer cleanup()

	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, builder.L2.Client)
	Require(t, err)
	sequencerTxOpts := builder.L1Info.GetDefaultTransactOpts("sequencer", ctx)

	builder.L2Info.GenerateAccount("Destination")
	const numBatches = 3
	for i := 0; i < numBatches; i++ {
		makeBatch(t, builder.L2.ConsensusNode, builder.L2Info, builder.L1.Client, &sequencerTxOpts, seqInbox, seqInboxAddr, -1)
	}

	for blockNum := uint64(0); blockNum < uint64(makeBatch_MsgsPerBatch)*3; blockNum++ {
		callOpts := bind.CallOpts{Context: ctx}
		gotBatchNumNodeInterface, err := nodeInterface.FindBatchContainingBlock(&callOpts, blockNum)
		Require(t, err)
		var gotBatchNumRPC uint64
		err = builder.L2.Client.Client().CallContext(ctx, &gotBatchNumRPC, "arb_findBatchContainingBlock", blockNum)
		Require(t, err)
		if gotBatchNumNodeInterface != gotBatchNumRPC {
			Fatal(t, "mismatched results from arb_findBatchContainingBlock and NodeInterface. blocknum ", blockNum, " nodeinterface ", gotBatchNumNodeInterface, " rpc ", gotBatchNumRPC)
		}
		gotBatchNum := gotBatchNumNodeInterface

		expBatchNum := uint64(0)
		if blockNum > 0 {
			expBatchNum = 1 + (blockNum-1)/uint64(makeBatch_MsgsPerBatch)
		}
		if expBatchNum != gotBatchNum {
			Fatal(t, "wrong result from findBatchContainingBlock. blocknum ", blockNum, " expected ", expBatchNum, " got ", gotBatchNum)
		}
		batchL1Block, err := builder.L2.ConsensusNode.InboxTracker.GetBatchParentChainBlock(gotBatchNum)
		Require(t, err)
		blockHeader, err := builder.L2.Client.HeaderByNumber(ctx, new(big.Int).SetUint64(blockNum))
		Require(t, err)
		blockHash := blockHeader.Hash()

		minCurrentL1Block, err := builder.L1.Client.BlockNumber(ctx)
		Require(t, err)
		gotConfirmations, err := nodeInterface.GetL1Confirmations(&callOpts, blockHash)
		Require(t, err)
		maxCurrentL1Block, err := builder.L1.Client.BlockNumber(ctx)
		Require(t, err)

		if gotConfirmations > (maxCurrentL1Block-batchL1Block) || gotConfirmations < (minCurrentL1Block-batchL1Block) {
			Fatal(t, "wrong number of confirmations. got ", gotConfirmations)
		}
	}
}
