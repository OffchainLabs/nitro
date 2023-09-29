// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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

	seqNodeConfig := arbnode.ConfigDefaultL2Test()
	seqNodeConfig.Feed.Output = *newBroadcasterConfigTest()
	l2info1, nodeA, client1 := CreateTestL2WithConfig(t, ctx, nil, seqNodeConfig, true)
	defer nodeA.StopAndWait()
	clientNodeConfig := arbnode.ConfigDefaultL2Test()
	port := nodeA.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
	clientNodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)

	_, nodeB, client2 := CreateTestL2WithConfig(t, ctx, nil, clientNodeConfig, false)
	defer nodeB.StopAndWait()

	l2info1.GenerateAccount("User2")

	tx := l2info1.PrepareTx("Owner", "User2", l2info1.TransferGas, big.NewInt(1e12), nil)

	err := client1.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, client1, tx)
	Require(t, err)

	_, err = WaitForTx(ctx, client2, tx.Hash(), time.Second*5)
	Require(t, err)
	l2balance, err := client2.BalanceAt(ctx, l2info1.GetAddress("User2"), nil)
	Require(t, err)
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}
}

func TestSequencerFeed_TimeBoost(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seqNodeConfig := arbnode.ConfigDefaultL2Test()
	seqNodeConfig.Sequencer.TimeBoost = false

	seqNodeConfig.Feed.Output = *newBroadcasterConfigTest()

	l2info1, sequencerNode, sequencerClient := CreateTestL2WithConfig(t, ctx, nil, seqNodeConfig, true)
	defer sequencerNode.StopAndWait()
	clientNodeConfig := arbnode.ConfigDefaultL2Test()
	port := sequencerNode.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
	clientNodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)

	_, listenerNode, _ := CreateTestL2WithConfig(t, ctx, nil, clientNodeConfig, false)
	defer listenerNode.StopAndWait()

	// Seed 10 different accounts with value and prepare 10 transactions to send to the sequencer
	// with timeboost enabled to ensure that they are ordered according to their priority fee.
	numTxs := 10
	txs := make([]*types.Transaction, numTxs)
	for i := 0; i < numTxs; i++ {
		userName := fmt.Sprintf("User%d", i)
		l2info1.GenerateAccount(userName)
		tx := l2info1.PrepareTx("Owner", userName, l2info1.TransferGas, big.NewInt(1e18), nil)
		Require(t, sequencerClient.SendTransaction(ctx, tx))
		_, err := EnsureTxSucceeded(ctx, sequencerClient, tx)
		Require(t, err)

		// Prepare a tx from the user back to the owner with a priority fee for time boost.
		// In this example, lower index users will have lower priority fees. That is, if we send txs
		// 0, 1, 2,... we expect the ordering to be ..., 2, 1, 0.
		priorityFee := new(big.Int).SetUint64(uint64((i + 1) * 50))
		tx = l2info1.PrepareBoostableTx(userName, "Owner", l2info1.TransferGas, big.NewInt(1e12+int64(i)), nil, priorityFee)
		txs[i] = tx
	}

	// Send out 10 boosted transactions concurrently.
	// TODO: Normalize to the same time boost round with some timer magic. Hacky for now.
	time.Sleep(time.Millisecond * 50)
	var wg sync.WaitGroup
	wg.Add(numTxs)
	for i := range txs {
		go func(ii int, w *sync.WaitGroup) {
			defer w.Done()
			Require(t, sequencerClient.SendTransaction(ctx, txs[ii]))
			_, err := EnsureTxSucceeded(ctx, sequencerClient, txs[ii])
			Require(t, err)
		}(i, &wg)
	}
	wg.Wait()

	// Group txs by block number.
	txIndexByBlockNum := make(map[uint64][]int, numTxs) // Change the value type to a slice of ints to store multiple tx indices per block
	for i := range txs {
		receipt, err := sequencerClient.TransactionReceipt(ctx, txs[i].Hash())
		Require(t, err)
		blockNum := receipt.BlockNumber.Uint64()
		txIndexByBlockNum[blockNum] = append(txIndexByBlockNum[blockNum], i)
	}
	txIndexByHash := make(map[common.Hash]int, numTxs)
	for i := range txs {
		txIndexByHash[txs[i].Hash()] = i
	}

	// For the txs within each block, we check that the txs are ordered by priority fee, and that
	// txs indices follow the relationship i < j => txs[i].PriorityFee > txs[j].PriorityFee.
	for blockNum, txIndices := range txIndexByBlockNum {
		block, err := sequencerClient.BlockByNumber(ctx, new(big.Int).SetUint64(blockNum))
		Require(t, err)

		blockTxs := block.Transactions()

		// Check this block contains all tx indices we care about.
		blockTxsByHash := make(map[common.Hash]struct{}, len(blockTxs))
		for _, blockTx := range blockTxs {
			blockTxsByHash[blockTx.Hash()] = struct{}{}
		}
		for _, txIndex := range txIndices {
			txHash := txs[txIndex].Hash()
			if _, ok := blockTxsByHash[txHash]; !ok {
				t.Fatal("Block", blockNum, "does not contain tx", txHash.Hex())
			}
		}

		// Assuming PriorityFee is a field in your tx struct and can be accessed as tx.PriorityFee
		// Check the order of priority fees for transactions within the block
		// TODO: Skip tx 0 seems to be irrelevant?
		for i := 1; i < len(blockTxs)-1; i++ {
			txA := blockTxs[i]
			txB := blockTxs[i+1]
			txAIndex := txIndexByHash[txA.Hash()]
			txBIndex := txIndexByHash[txB.Hash()]
			t.Logf("Creation_idx=%d, idx_in_block=%d has fee %d, creation_idx=%d, idx_in_block=%d has fee %d", txAIndex, i, txA.GasTipCap().Uint64(), txBIndex, i+1, txB.GasTipCap().Uint64())
			if txA.GasTipCap().Uint64() < txB.GasTipCap().Uint64() {
				t.Fatalf("Transactions in block are not ordered by priority fee, tx=%d has fee %d, tx=%d has fee %d", i, txA.GasTipCap().Uint64(), i+1, txB.GasTipCap().Uint64())
			}
			if txAIndex < txBIndex {
				t.Fatalf("Transaction at index %d should be greater than index %d due to time boost", txAIndex, txBIndex)
			}
		}
	}
}

func TestRelayedSequencerFeed(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seqNodeConfig := arbnode.ConfigDefaultL2Test()
	seqNodeConfig.Feed.Output = *newBroadcasterConfigTest()
	l2info1, nodeA, client1 := CreateTestL2WithConfig(t, ctx, nil, seqNodeConfig, true)
	defer nodeA.StopAndWait()

	bigChainId, err := client1.ChainID(ctx)
	Require(t, err)

	config := relay.ConfigDefault
	port := nodeA.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
	config.Node.Feed.Input = *newBroadcastClientConfigTest(port)
	config.Node.Feed.Output = *newBroadcasterConfigTest()
	config.Chain.ID = bigChainId.Uint64()

	feedErrChan := make(chan error, 10)
	currentRelay, err := relay.NewRelay(&config, feedErrChan)
	Require(t, err)
	err = currentRelay.Start(ctx)
	Require(t, err)
	defer currentRelay.StopAndWait()

	clientNodeConfig := arbnode.ConfigDefaultL2Test()
	port = currentRelay.GetListenerAddr().(*net.TCPAddr).Port
	clientNodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)
	_, nodeC, client3 := CreateTestL2WithConfig(t, ctx, nil, clientNodeConfig, false)
	defer nodeC.StopAndWait()
	StartWatchChanErr(t, ctx, feedErrChan, nodeC)

	l2info1.GenerateAccount("User2")

	tx := l2info1.PrepareTx("Owner", "User2", l2info1.TransferGas, big.NewInt(1e12), nil)

	err = client1.SendTransaction(ctx, tx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, client1, tx)
	Require(t, err)

	_, err = WaitForTx(ctx, client3, tx.Hash(), time.Second*5)
	Require(t, err)
	l2balance, err := client3.BalanceAt(ctx, l2info1.GetAddress("User2"), nil)
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
	l2infoA, nodeA, l2clientA, l1info, _, l1client, l1stack := createTestNodeOnL1WithConfig(t, ctx, true, nodeConfigA, chainConfig, nil)
	defer requireClose(t, l1stack, "unable to close l1stack")
	defer nodeA.StopAndWait()

	authorizeDASKeyset(t, ctx, dasSignerKey, l1info, l1client)

	// The lying sequencer
	nodeConfigC := arbnode.ConfigDefaultL1Test()
	nodeConfigC.BatchPoster.Enable = false
	nodeConfigC.DataAvailability = nodeConfigA.DataAvailability
	nodeConfigC.DataAvailability.RPCAggregator.Enable = false
	nodeConfigC.Feed.Output = *newBroadcasterConfigTest()
	l2clientC, nodeC := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, l1info, &l2infoA.ArbInitData, nodeConfigC, nil)
	defer nodeC.StopAndWait()

	port := nodeC.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port

	// The client node, connects to lying sequencer's feed
	nodeConfigB := arbnode.ConfigDefaultL1NonSequencerTest()
	nodeConfigB.Feed.Output.Enable = false
	nodeConfigB.Feed.Input = *newBroadcastClientConfigTest(port)
	nodeConfigB.DataAvailability = nodeConfigA.DataAvailability
	nodeConfigB.DataAvailability.RPCAggregator.Enable = false
	l2clientB, nodeB := Create2ndNodeWithConfig(t, ctx, nodeA, l1stack, l1info, &l2infoA.ArbInitData, nodeConfigB, nil)
	defer nodeB.StopAndWait()

	l2infoA.GenerateAccount("FraudUser")
	l2infoA.GenerateAccount("RealUser")

	fraudTx := l2infoA.PrepareTx("Owner", "FraudUser", l2infoA.TransferGas, big.NewInt(1e12), nil)
	l2infoA.GetInfoWithPrivKey("Owner").Nonce -= 1 // Use same l2info object for different l2s
	realTx := l2infoA.PrepareTx("Owner", "RealUser", l2infoA.TransferGas, big.NewInt(1e12), nil)

	err := l2clientC.SendTransaction(ctx, fraudTx)
	if err != nil {
		t.Fatal("error sending fraud transaction:", err)
	}

	_, err = EnsureTxSucceeded(ctx, l2clientC, fraudTx)
	if err != nil {
		t.Fatal("error ensuring fraud transaction succeeded:", err)
	}

	// Node B should get the transaction immediately from the sequencer feed
	_, err = WaitForTx(ctx, l2clientB, fraudTx.Hash(), time.Second*15)
	if err != nil {
		t.Fatal("error waiting for tx:", err)
	}
	l2balance, err := l2clientB.BalanceAt(ctx, l2infoA.GetAddress("FraudUser"), nil)
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

	_, err = EnsureTxSucceeded(ctx, l2clientA, realTx)
	if err != nil {
		t.Fatal("error ensuring real transaction succeeded:", err)
	}

	// Node B should get the transaction after NodeC posts a batch.
	_, err = WaitForTx(ctx, l2clientB, realTx.Hash(), time.Second*5)
	if err != nil {
		t.Fatal("error waiting for transaction to get to node b:", err)
	}
	l2balanceFraudAcct, err := l2clientB.BalanceAt(ctx, l2infoA.GetAddress("FraudUser"), nil)
	if err != nil {
		t.Fatal("error getting fraud balance:", err)
	}
	if l2balanceFraudAcct.Cmp(big.NewInt(0)) != 0 {
		t.Fatal("Unexpected balance (fraud acct should be empty) was:", l2balanceFraudAcct)
	}

	l2balanceRealAcct, err := l2clientB.BalanceAt(ctx, l2infoA.GetAddress("RealUser"), nil)
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
