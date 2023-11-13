// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
)

func callFindBatchContainig(t *testing.T, ctx context.Context, client *ethclient.Client, nodeAbi *abi.ABI, blockNum uint64) uint64 {
	findBatch := nodeAbi.Methods["findBatchContainingBlock"]
	callData := append([]byte{}, findBatch.ID...)
	packed, err := findBatch.Inputs.Pack(blockNum)
	Require(t, err)
	callData = append(callData, packed...)
	msg := ethereum.CallMsg{
		To:   &types.NodeInterfaceAddress,
		Data: callData,
	}
	returnData, err := client.CallContract(ctx, msg, nil)
	Require(t, err)
	outputs, err := findBatch.Outputs.Unpack(returnData)
	Require(t, err)
	if len(outputs) != 1 {
		Fatal(t, "expected 1 output from findBatchContainingBlock, got", len(outputs))
	}
	gotBatchNum, ok := outputs[0].(uint64)
	if !ok {
		Fatal(t, "bad output from findBatchContainingBlock")
	}
	return gotBatchNum
}

func callGetL1Confirmations(t *testing.T, ctx context.Context, client *ethclient.Client, nodeAbi *abi.ABI, blockHash common.Hash) uint64 {
	getConfirmations := nodeAbi.Methods["getL1Confirmations"]
	callData := append([]byte{}, getConfirmations.ID...)
	packed, err := getConfirmations.Inputs.Pack(blockHash)
	Require(t, err)
	callData = append(callData, packed...)
	msg := ethereum.CallMsg{
		To:   &types.NodeInterfaceAddress,
		Data: callData,
	}
	returnData, err := client.CallContract(ctx, msg, nil)
	Require(t, err)
	outputs, err := getConfirmations.Outputs.Unpack(returnData)
	Require(t, err)
	if len(outputs) != 1 {
		Fatal(t, "expected 1 output from findBatchContainingBlock, got", len(outputs))
	}
	confirmations, ok := outputs[0].(uint64)
	if !ok {
		Fatal(t, "bad output from findBatchContainingBlock")
	}
	return confirmations
}

func TestFindBatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l1Info := NewL1TestInfo(t)
	initialBalance := new(big.Int).Lsh(big.NewInt(1), 200)
	l1Info.GenerateGenesisAccount("deployer", initialBalance)
	l1Info.GenerateGenesisAccount("asserter", initialBalance)
	l1Info.GenerateGenesisAccount("challenger", initialBalance)
	l1Info.GenerateGenesisAccount("sequencer", initialBalance)

	l1Info, l1Backend, _, _ := createTestL1BlockChain(t, l1Info)
	conf := arbnode.ConfigDefaultL1Test()
	conf.BlockValidator.Enable = false
	conf.BatchPoster.Enable = false

	chainConfig := params.ArbitrumDevTestChainConfig()
	fatalErrChan := make(chan error, 10)
	rollupAddresses, initMsg := DeployOnTestL1(t, ctx, l1Info, l1Backend, chainConfig)

	bridgeAddr, seqInbox, seqInboxAddr := setupSequencerInboxStub(ctx, t, l1Info, l1Backend, chainConfig)

	rollupAddresses.Bridge = bridgeAddr
	rollupAddresses.SequencerInbox = seqInboxAddr
	l2Info := NewArbTestInfo(t, chainConfig.ChainID)
	consensus, exec := createL2Nodes(t, ctx, conf, chainConfig, l1Backend, l2Info, rollupAddresses, initMsg, nil, nil, fatalErrChan)
	Require(t, exec.Initialize(ctx))
	err := consensus.Start(ctx)
	Require(t, err)
	Require(t, exec.Start(ctx))

	l2Client := ClientForStack(t, consensus.Stack)
	nodeAbi, err := node_interfacegen.NodeInterfaceMetaData.GetAbi()
	Require(t, err)
	sequencerTxOpts := l1Info.GetDefaultTransactOpts("sequencer", ctx)

	l2Info.GenerateAccount("Destination")
	makeBatch(t, consensus, l2Info, l1Backend, &sequencerTxOpts, seqInbox, seqInboxAddr, -1)
	makeBatch(t, consensus, l2Info, l1Backend, &sequencerTxOpts, seqInbox, seqInboxAddr, -1)
	makeBatch(t, consensus, l2Info, l1Backend, &sequencerTxOpts, seqInbox, seqInboxAddr, -1)

	for blockNum := uint64(0); blockNum < uint64(makeBatch_MsgsPerBatch)*3; blockNum++ {
		gotBatchNum := callFindBatchContainig(t, ctx, l2Client, nodeAbi, blockNum)
		expBatchNum := uint64(0)
		if blockNum > 0 {
			expBatchNum = 1 + (blockNum-1)/uint64(makeBatch_MsgsPerBatch)
		}
		if expBatchNum != gotBatchNum {
			Fatal(t, "wrong result from findBatchContainingBlock. blocknum ", blockNum, " expected ", expBatchNum, " got ", gotBatchNum)
		}
		batchL1Block, err := consensus.InboxTracker.GetBatchParentChainBlock(gotBatchNum).Await(ctx)
		Require(t, err)
		blockHeader, err := l2Client.HeaderByNumber(ctx, new(big.Int).SetUint64(blockNum))
		Require(t, err)
		blockHash := blockHeader.Hash()

		minCurrentL1Block, err := l1Backend.BlockNumber(ctx)
		Require(t, err)
		gotConfirmations := callGetL1Confirmations(t, ctx, l2Client, nodeAbi, blockHash)
		maxCurrentL1Block, err := l1Backend.BlockNumber(ctx)
		Require(t, err)

		if gotConfirmations > (maxCurrentL1Block-batchL1Block) || gotConfirmations < (minCurrentL1Block-batchL1Block) {
			Fatal(t, "wrong number of confirmations. got ", gotConfirmations)
		}
	}
}

func TestL2BlockRangeForL1(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()
	user := builder.L1Info.GetDefaultTransactOpts("User", ctx)

	numTransactions := 200
	for i := 0; i < numTransactions; i++ {
		builder.L2.TransferBalanceTo(t, "Owner", util.RemapL1Address(user.From), big.NewInt(1e18), builder.L2Info)
	}

	nodeInterface, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, builder.L2.Client)
	if err != nil {
		t.Fatalf("Error creating node interface: %v", err)
	}

	l1BlockNums := map[uint64]*[2]uint64{}
	latestL2, err := builder.L2.Client.BlockNumber(ctx)
	if err != nil {
		t.Fatalf("Error querying most recent l2 block: %v", err)
	}
	for l2BlockNum := uint64(0); l2BlockNum <= latestL2; l2BlockNum++ {
		l1BlockNum, err := nodeInterface.BlockL1Num(&bind.CallOpts{}, l2BlockNum)
		if err != nil {
			t.Fatalf("Error quering l1 block number for l2 block: %d, error: %v", l2BlockNum, err)
		}
		if _, ok := l1BlockNums[l1BlockNum]; !ok {
			l1BlockNums[l1BlockNum] = &[2]uint64{l2BlockNum, l2BlockNum}
		}
		l1BlockNums[l1BlockNum][1] = l2BlockNum
	}

	// Test success.
	for l1BlockNum := range l1BlockNums {
		rng, err := nodeInterface.L2BlockRangeForL1(&bind.CallOpts{}, l1BlockNum)
		if err != nil {
			t.Fatalf("Error getting l2 block range for l1 block: %d, error: %v", l1BlockNum, err)
		}
		expected := l1BlockNums[l1BlockNum]
		if rng.FirstBlock != expected[0] || rng.LastBlock != expected[1] {
			unexpectedL1BlockNum, err := nodeInterface.BlockL1Num(&bind.CallOpts{}, rng.LastBlock)
			if err != nil {
				t.Fatalf("Error quering l1 block number for l2 block: %d, error: %v", rng.LastBlock, err)
			}
			// Handle the edge case when new l2 blocks are produced between latestL2 was last calculated and now.
			if unexpectedL1BlockNum != l1BlockNum || rng.LastBlock < expected[1] || rng.FirstBlock != expected[0] {
				t.Errorf("L2BlockRangeForL1(%d) = (%d %d) want (%d %d)", l1BlockNum, rng.FirstBlock, rng.LastBlock, expected[0], expected[1])
			}
		}
	}
	// Test invalid case.
	if _, err := nodeInterface.L2BlockRangeForL1(&bind.CallOpts{}, 1e5); err == nil {
		t.Fatalf("GetL2BlockRangeForL1 didn't fail for an invalid input")
	}
}
