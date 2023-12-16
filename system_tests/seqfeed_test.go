// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/relay"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

func newBroadcasterConfigTest() *wsbroadcastserver.BroadcasterConfig {
	config := wsbroadcastserver.DefaultTestBroadcasterConfig
	config.Enable = true
	config.Port = "0"
	return &config
}

func newBroadcastClientConfigTest(port int) *broadcastclient.Config {
	return &broadcastclient.Config{
		URL:     []string{fmt.Sprintf("ws://localhost:%d/feed", port)},
		Timeout: 200 * time.Millisecond,
		Verify: signature.VerifierConfig{
			Dangerous: signature.DangerousVerifierConfig{
				AcceptMissing: true,
			},
		},
	}
}

func TestSequencerFeed(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builderSeq := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builderSeq.nodeConfig.Feed.Output = *newBroadcasterConfigTest()
	cleanupSeq := builderSeq.Build(t)
	defer cleanupSeq()
	seqInfo, seqNode, seqClient := builderSeq.L2Info, builderSeq.L2.ConsensusNode, builderSeq.L2.Client

	port := seqNode.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.nodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)
	builder.takeOwnership = false
	cleanup := builder.Build(t)
	defer cleanup()
	client := builder.L2.Client

	seqInfo.GenerateAccount("User2")

	tx := seqInfo.PrepareTx("Owner", "User2", seqInfo.TransferGas, big.NewInt(1e12), nil)

	err := seqClient.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = builderSeq.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	_, err = WaitForTx(ctx, client, tx.Hash(), time.Second*5)
	Require(t, err)
	l2balance, err := client.BalanceAt(ctx, seqInfo.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}
}

// func TestSequencerFeed_TimeBoost(t *testing.T) {
// 	t.Parallel()
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	builderSeq := NewNodeBuilder(ctx).DefaultConfig(t, false)
// 	builderSeq.nodeConfig.Feed.Output = *newBroadcasterConfigTest()
// 	builderSeq.execConfig.Sequencer.Enable = true
// 	builderSeq.execConfig.Sequencer.TimeBoost = true
// 	cleanupSeq := builderSeq.Build(t)
// 	defer cleanupSeq()
// 	seqInfo, seqNode, seqClient := builderSeq.L2Info, builderSeq.L2.ConsensusNode, builderSeq.L2.Client

// 	port := seqNode.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
// 	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
// 	builder.nodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)
// 	builder.takeOwnership = false
// 	cleanup := builder.Build(t)
// 	defer cleanup()
// 	client := builder.L2.Client

// 	seqInfo.GenerateAccount("User2")

// 	tx := seqInfo.PrepareTx("Owner", "User2", seqInfo.TransferGas, big.NewInt(1e12), nil)

// 	err := seqClient.SendTransaction(ctx, tx)
// 	Require(t, err)

// 	_, err = builderSeq.L2.EnsureTxSucceeded(tx)
// 	Require(t, err)

// 	_, err = WaitForTx(ctx, client, tx.Hash(), time.Second*5)
// 	Require(t, err)
// 	l2balance, err := client.BalanceAt(ctx, seqInfo.GetAddress("User2"), nil)
// 	Require(t, err)
// 	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
// 		t.Fatal("Unexpected balance:", l2balance)
// 	}
// 	t.Fatal("oops")

// baseFee := GetBaseFee(t, sequencerClient, ctx)
// l2info1.GasPrice = baseFee
// callOpts := l2info1.GetDefaultCallOpts("Owner", ctx)
// arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(common.HexToAddress("0x6b"), sequencerClient)
// Require(t, err, "failed to deploy contract")

// // Get the network fee account
// networkFeeAccount, err := arbOwnerPublic.GetNetworkFeeAccount(callOpts)
// Require(t, err, "could not get the network fee account")

// // Seed 5 different accounts with value and prepare 10 transactions to send to the sequencer
// // with timeboost enabled to ensure that they are ordered according to their priority fee.
// numTxs := 5
// users := make([]common.Address, numTxs)
// for i := 0; i < numTxs; i++ {
// 	userName := fmt.Sprintf("User%d", i)
// 	l2info1.GenerateAccount(userName)
// 	tx := l2info1.PrepareTx("Owner", userName, l2info1.TransferGas, big.NewInt(1e18), nil)
// 	Require(t, sequencerClient.SendTransaction(ctx, tx))
// 	_, err := EnsureTxSucceeded(ctx, sequencerClient, tx)
// 	Require(t, err)
// 	users[i] = l2info1.GetAddress(userName)
// }

// type txsForUser struct {
// 	txs []common.Hash
// 	sync.RWMutex
// }

// userTxs := &txsForUser{
// 	txs: make([]common.Hash, len(users)),
// }
// ensureTxPaysNetworkTip := func(userIdx uint64) {
// 	baseFee := GetBaseFee(t, sequencerClient, ctx)
// 	l2info1.GasPrice = baseFee
// 	tip := uint64(userIdx + 1)
// 	tipCap := arbmath.BigMulByUint(baseFee, tip)
// 	gasPrice := arbmath.BigAdd(baseFee, tipCap)
// 	value := big.NewInt(1)
// 	var data []byte
// 	userName := fmt.Sprintf("User%d", userIdx)
// 	tx := l2info1.PrepareTippingTx(userName, "Owner", gasPrice.Uint64(), tipCap, value, data)
// 	networkBefore := GetBalance(t, ctx, sequencerClient, networkFeeAccount)
// 	Require(t, sequencerClient.SendTransaction(ctx, tx))
// 	_, err := EnsureTxSucceeded(ctx, sequencerClient, tx)
// 	Require(t, err)
// 	userTxs.Lock()
// 	userTxs.txs[userIdx] = tx.Hash()
// 	userTxs.Unlock()

// 	networkAfter := GetBalance(t, ctx, sequencerClient, networkFeeAccount)
// 	networkRevenue := arbmath.BigSub(networkAfter, networkBefore)
// 	t.Logf("Network revenue=%d", networkRevenue.Uint64())
// }

// // Send out 10 boosted transactions concurrently.
// // TODO: Normalize to the same time boost round with some timer magic. Hacky for now.
// time.Sleep(time.Millisecond * 50)
// var wg sync.WaitGroup
// wg.Add(numTxs)
// for userIdx := range users {
// 	go func(ii uint64, w *sync.WaitGroup) {
// 		defer w.Done()
// 		ensureTxPaysNetworkTip(ii)
// 	}(uint64(userIdx), &wg)
// }
// wg.Wait()

// // Group txs by block number.
// txIndexByBlockNum := make(map[uint64][]int, numTxs) // Change the value type to a slice of ints to store multiple tx indices per block
// for i, tx := range userTxs.txs {
// 	receipt, err := sequencerClient.TransactionReceipt(ctx, tx)
// 	Require(t, err)
// 	blockNum := receipt.BlockNumber.Uint64()
// 	txIndexByBlockNum[blockNum] = append(txIndexByBlockNum[blockNum], i)
// }
// txIndexByHash := make(map[common.Hash]int, numTxs)

// for i, tx := range userTxs.txs {
// 	txIndexByHash[tx] = i
// }

// // For the txs within each block, we check that the txs are ordered by priority fee, and that
// // txs indices follow the relationship i < j => txs[i].PriorityFee > txs[j].PriorityFee.
// for blockNum, txIndices := range txIndexByBlockNum {
// 	block, err := sequencerClient.BlockByNumber(ctx, new(big.Int).SetUint64(blockNum))
// 	Require(t, err)

// 	blockTxs := block.Transactions()

// 	// Check this block contains all tx indices we care about.
// 	blockTxsByHash := make(map[common.Hash]struct{}, len(blockTxs))
// 	for _, blockTx := range blockTxs {
// 		blockTxsByHash[blockTx.Hash()] = struct{}{}
// 	}
// 	for _, txIndex := range txIndices {
// 		txHash := userTxs.txs[txIndex]
// 		if _, ok := blockTxsByHash[txHash]; !ok {
// 			t.Fatal("Block", blockNum, "does not contain tx", txHash.Hex())
// 		}
// 	}

// 	// Assuming PriorityFee is a field in your tx struct and can be accessed as tx.PriorityFee
// 	// Check the order of priority fees for transactions within the block
// 	// TODO: Skip tx 0 seems to be irrelevant?
// 	for i := 1; i < len(blockTxs)-1; i++ {
// 		txA := blockTxs[i]
// 		txB := blockTxs[i+1]
// 		txAIndex := txIndexByHash[txA.Hash()]
// 		txBIndex := txIndexByHash[txB.Hash()]
// 		t.Logf("Creation_idx=%d, idx_in_block=%d has fee %d, creation_idx=%d, idx_in_block=%d has fee %d", txAIndex, i, txA.GasTipCap().Uint64(), txBIndex, i+1, txB.GasTipCap().Uint64())
// 		if txA.GasTipCap().Uint64() < txB.GasTipCap().Uint64() {
// 			t.Fatalf("Transactions in block are not ordered by priority fee, tx=%d has fee %d, tx=%d has fee %d", i, txA.GasTipCap().Uint64(), i+1, txB.GasTipCap().Uint64())
// 		}
// 		if txAIndex < txBIndex {
// 			t.Fatalf("Transaction at index %d should be greater than index %d due to time boost", txAIndex, txBIndex)
// 		}
// 	}
// }
//}

func TestRelayedSequencerFeed(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builderSeq := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builderSeq.nodeConfig.Feed.Output = *newBroadcasterConfigTest()
	cleanupSeq := builderSeq.Build(t)
	defer cleanupSeq()
	seqInfo, seqNode, seqClient := builderSeq.L2Info, builderSeq.L2.ConsensusNode, builderSeq.L2.Client

	bigChainId, err := seqClient.ChainID(ctx)
	Require(t, err)

	config := relay.ConfigDefault
	port := seqNode.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
	config.Node.Feed.Input = *newBroadcastClientConfigTest(port)
	config.Node.Feed.Output = *newBroadcasterConfigTest()
	config.Chain.ID = bigChainId.Uint64()

	feedErrChan := make(chan error, 10)
	currentRelay, err := relay.NewRelay(&config, feedErrChan)
	Require(t, err)
	err = currentRelay.Start(ctx)
	Require(t, err)
	defer currentRelay.StopAndWait()

	port = currentRelay.GetListenerAddr().(*net.TCPAddr).Port
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.nodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)
	builder.takeOwnership = false
	cleanup := builder.Build(t)
	defer cleanup()
	node, client := builder.L2.ConsensusNode, builder.L2.Client
	StartWatchChanErr(t, ctx, feedErrChan, node)

	seqInfo.GenerateAccount("User2")

	tx := seqInfo.PrepareTx("Owner", "User2", seqInfo.TransferGas, big.NewInt(1e12), nil)

	err = seqClient.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = builderSeq.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	_, err = WaitForTx(ctx, client, tx.Hash(), time.Second*5)
	Require(t, err)
	l2balance, err := client.BalanceAt(ctx, seqInfo.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}
}

func testLyingSequencer(t *testing.T, dasModeStr string) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The truthful sequencer
	chainConfig, nodeConfigA, lifecycleManager, _, dasSignerKey := setupConfigWithDAS(t, ctx, dasModeStr)
	defer lifecycleManager.StopAndWaitUntil(time.Second)

	nodeConfigA.BatchPoster.Enable = true
	nodeConfigA.Feed.Output.Enable = false
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig = nodeConfigA
	builder.chainConfig = chainConfig
	builder.L2Info = nil
	cleanup := builder.Build(t)
	defer cleanup()

	l2clientA := builder.L2.Client

	authorizeDASKeyset(t, ctx, dasSignerKey, builder.L1Info, builder.L1.Client)

	// The lying sequencer
	nodeConfigC := arbnode.ConfigDefaultL1Test()
	nodeConfigC.BatchPoster.Enable = false
	nodeConfigC.DataAvailability = nodeConfigA.DataAvailability
	nodeConfigC.DataAvailability.RPCAggregator.Enable = false
	nodeConfigC.Feed.Output = *newBroadcasterConfigTest()
	testClientC, cleanupC := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfigC})
	defer cleanupC()
	l2clientC, nodeC := testClientC.Client, testClientC.ConsensusNode

	port := nodeC.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port

	// The client node, connects to lying sequencer's feed
	nodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfigB.Feed.Output.Enable = false
	nodeConfigB.Feed.Input = *newBroadcastClientConfigTest(port)
	nodeConfigB.DataAvailability = nodeConfigA.DataAvailability
	nodeConfigB.DataAvailability.RPCAggregator.Enable = false
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfigB})
	defer cleanupB()
	l2clientB := testClientB.Client

	builder.L2Info.GenerateAccount("FraudUser")
	builder.L2Info.GenerateAccount("RealUser")

	fraudTx := builder.L2Info.PrepareTx("Owner", "FraudUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	builder.L2Info.GetInfoWithPrivKey("Owner").Nonce -= 1 // Use same l2info object for different l2s
	realTx := builder.L2Info.PrepareTx("Owner", "RealUser", builder.L2Info.TransferGas, big.NewInt(1e12), nil)

	for i := 0; i < 10; i++ {
		err := l2clientC.SendTransaction(ctx, fraudTx)
		if err == nil {
			break
		}
		<-time.After(time.Millisecond * 10)
		if i == 9 {
			t.Fatal("error sending fraud transaction:", err)
		}
	}

	_, err := testClientC.EnsureTxSucceeded(fraudTx)
	if err != nil {
		t.Fatal("error ensuring fraud transaction succeeded:", err)
	}

	// Node B should get the transaction immediately from the sequencer feed
	_, err = WaitForTx(ctx, l2clientB, fraudTx.Hash(), time.Second*15)
	if err != nil {
		t.Fatal("error waiting for tx:", err)
	}
	l2balance, err := l2clientB.BalanceAt(ctx, builder.L2Info.GetAddress("FraudUser"), nil)
	if err != nil {
		t.Fatal("error getting balance:", err)
	}
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}

	// Send the real transaction to client A
	err = l2clientA.SendTransaction(ctx, realTx)
	if err != nil {
		t.Fatal("error sending real transaction:", err)
	}

	_, err = builder.L2.EnsureTxSucceeded(realTx)
	if err != nil {
		t.Fatal("error ensuring real transaction succeeded:", err)
	}

	// Node B should get the transaction after NodeC posts a batch.
	_, err = WaitForTx(ctx, l2clientB, realTx.Hash(), time.Second*5)
	if err != nil {
		t.Fatal("error waiting for transaction to get to node b:", err)
	}
	l2balanceFraudAcct, err := l2clientB.BalanceAt(ctx, builder.L2Info.GetAddress("FraudUser"), nil)
	if err != nil {
		t.Fatal("error getting fraud balance:", err)
	}
	if l2balanceFraudAcct.Cmp(big.NewInt(0)) != 0 {
		t.Fatal("Unexpected balance (fraud acct should be empty) was:", l2balanceFraudAcct)
	}

	l2balanceRealAcct, err := l2clientB.BalanceAt(ctx, builder.L2Info.GetAddress("RealUser"), nil)
	if err != nil {
		t.Fatal("error getting real balance:", err)
	}
	if l2balanceRealAcct.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance of real account:", l2balanceRealAcct)
	}
}

func TestLyingSequencer(t *testing.T) {
	testLyingSequencer(t, "onchain")
}

func TestLyingSequencerLocalDAS(t *testing.T) {
	testLyingSequencer(t, "files")
}
