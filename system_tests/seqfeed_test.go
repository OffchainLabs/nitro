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
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/broadcastclient"
	boostpolicies "github.com/offchainlabs/nitro/execution/gethexec/boost-policies"
	"github.com/offchainlabs/nitro/relay"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
	"golang.org/x/exp/slices"
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

func TestSequencerFeed_TxBoost(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builderSeq := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builderSeq.nodeConfig.Feed.Output = *newBroadcasterConfigTest()
	builderSeq.chainConfig.ArbitrumChainParams.InitialArbOSVersion = arbmath.MaxInt(
		arbostypes.ArbosVersion_ArbitrumTippingTx,
		builderSeq.chainConfig.ArbitrumChainParams.InitialArbOSVersion,
	)
	builderSeq.execConfig.Sequencer.TxBoost = true
	builderSeq.execConfig.Sequencer.TxBoostScoringPolicy = "express-lane"
	cleanupSeq := builderSeq.Build(t)
	defer cleanupSeq()
	seqInfo, seqNode, seqClient := builderSeq.L2Info, builderSeq.L2.ConsensusNode, builderSeq.L2.Client

	port := seqNode.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.chainConfig.ArbitrumChainParams.InitialArbOSVersion = arbmath.MaxInt(
		arbostypes.ArbosVersion_ArbitrumTippingTx,
		builder.chainConfig.ArbitrumChainParams.InitialArbOSVersion,
	)
	builder.nodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)
	builder.takeOwnership = false
	cleanup := builder.Build(t)
	defer cleanup()
	client := builder.L2.Client

	seqInfo.GenerateAccount("User2")

	tipCap := new(big.Int).SetUint64(1)
	tx := seqInfo.PrepareTippingTx("Owner", "User2", seqInfo.TransferGas, tipCap, big.NewInt(1e12), nil)

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

func TestTxBoost_BidPolicy_BidsPaid(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builderSeq := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builderSeq.chainConfig.ArbitrumChainParams.InitialArbOSVersion = arbmath.MaxInt(
		arbostypes.ArbosVersion_ArbitrumTippingTx,
		builderSeq.chainConfig.ArbitrumChainParams.InitialArbOSVersion,
	)

	builderSeq.nodeConfig.Feed.Output = *newBroadcasterConfigTest()
	builderSeq.execConfig.Sequencer.Enable = true
	builderSeq.execConfig.Sequencer.TxBoost = true
	builderSeq.execConfig.Sequencer.TxBoostScoringPolicy = "bid"
	cleanupSeq := builderSeq.Build(t)
	defer cleanupSeq()
	seqInfo, seqNode, seqClient := builderSeq.L2Info, builderSeq.L2.ConsensusNode, builderSeq.L2.Client

	port := seqNode.BroadcastServer.ListenerAddr().(*net.TCPAddr).Port
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.chainConfig.ArbitrumChainParams.InitialArbOSVersion = arbmath.MaxInt(
		arbostypes.ArbosVersion_ArbitrumTippingTx,
		builder.chainConfig.ArbitrumChainParams.InitialArbOSVersion,
	)
	builder.nodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)
	builder.takeOwnership = false
	cleanup := builder.Build(t)
	defer cleanup()

	// Get the arbowner and the current base fee for txs.
	baseFee := GetBaseFee(t, seqClient, ctx)
	seqInfo.GasPrice = baseFee
	callOpts := seqInfo.GetDefaultCallOpts("Owner", ctx)
	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(common.HexToAddress("0x6b"), seqClient)
	Require(t, err, "failed to deploy contract")

	// Get the network fee account.
	networkFeeAccount, err := arbOwnerPublic.GetNetworkFeeAccount(callOpts)
	Require(t, err, "could not get the network fee account")

	// Seed N different accounts with value.
	numAccounts := 10
	users := make([]common.Address, numAccounts)
	for i := 0; i < numAccounts; i++ {
		userName := fmt.Sprintf("User%d", i)
		seqInfo.GenerateAccount(userName)
		tx := seqInfo.PrepareTx("Owner", userName, seqInfo.TransferGas, big.NewInt(1e18), nil)
		Require(t, seqClient.SendTransaction(ctx, tx))
		_, err := EnsureTxSucceeded(ctx, seqClient, tx)
		Require(t, err)
		users[i] = seqInfo.GetAddress(userName)
	}

	// Prepare a bunch of boostable transactions to send to the sequencer
	// with tx boost enabled to ensure that they are ordered according to the score provided by the
	// express lane ordering policy.
	type txsForUser struct {
		txs []common.Hash
		sync.RWMutex
	}

	userTxs := &txsForUser{
		txs: make([]common.Hash, len(users)),
	}
	ensureTxPaysNetworkTip := func(userIdx uint64) {
		baseFee := GetBaseFee(t, seqClient, ctx)
		seqInfo.GasPrice = baseFee

		// We prepare a tipping tx type with a gas tip cap (bid).
		// This means the tx will go through the "express lane" with tx boost enabled
		// in the sequencer code and get sorted in descending order by bid,
		// breaking ties by first arrival timestamp.
		tip := uint64(userIdx + 1)
		tipCap := arbmath.BigMulByUint(baseFee, tip)
		gasPrice := arbmath.BigAdd(baseFee, tipCap)
		value := big.NewInt(1)
		var data []byte
		userName := fmt.Sprintf("User%d", userIdx)
		tx := seqInfo.PrepareTippingTx(userName, "Owner", gasPrice.Uint64(), tipCap, value, data)

		// Ensure the tx goes through and record the network fee account before and after.
		networkBefore := GetBalance(t, ctx, seqClient, networkFeeAccount)
		Require(t, seqClient.SendTransaction(ctx, tx))
		receipt, err := EnsureTxSucceeded(ctx, seqClient, tx)
		Require(t, err)

		// the network should receive
		//     1. compute costs
		//     2. tip on the compute costs
		//     3. tip on the data costs
		networkAfter := GetBalance(t, ctx, seqClient, networkFeeAccount)
		networkRevenue := arbmath.BigSub(networkAfter, networkBefore)
		gasUsedForL2 := receipt.GasUsed - receipt.GasUsedForL1
		feePaidForL2 := arbmath.BigMulByUint(gasPrice, gasUsedForL2)
		tipPaidToNet := arbmath.BigMulByUint(tipCap, receipt.GasUsedForL1)
		gotTip := networkRevenue.Cmp(arbmath.BigAdd(feePaidForL2, tipPaidToNet)) > 0
		if !gotTip {
			Fatal(t, "network didn't receive expected payment", networkRevenue, feePaidForL2, tipPaidToNet)
		}

		userTxs.Lock()
		userTxs.txs[userIdx] = tx.Hash()
		userTxs.Unlock()
	}

	sendNonBoostedTx := func(userIdx uint64) {
		userName := fmt.Sprintf("User%d", userIdx)
		tx := seqInfo.PrepareTx(userName, "Owner", seqInfo.TransferGas, big.NewInt(1e12), nil)
		err := seqClient.SendTransaction(ctx, tx)
		Require(t, err)
		_, err = builderSeq.L2.EnsureTxSucceeded(tx)
		Require(t, err)

		userTxs.Lock()
		userTxs.txs[userIdx] = tx.Hash()
		userTxs.Unlock()
	}

	// Send out 10 boosted and 10 non-boosted transactions concurrently.
	time.Sleep(time.Millisecond * 50)
	var wg sync.WaitGroup

	numBoostedTxs := 5
	numNonBoostedTxs := 5
	numTxs := numBoostedTxs + numNonBoostedTxs
	wg.Add(numTxs)
	for i := 0; i < numBoostedTxs; i++ {
		userIdx := i
		go func(ii uint64, w *sync.WaitGroup) {
			defer w.Done()
			ensureTxPaysNetworkTip(ii)
		}(uint64(userIdx), &wg)
	}
	for i := 0; i < numNonBoostedTxs; i++ {
		userIdx := numBoostedTxs + i
		go func(ii uint64, w *sync.WaitGroup) {
			defer w.Done()
			sendNonBoostedTx(ii)
		}(uint64(userIdx), &wg)
	}
	wg.Wait()

	// Group txs by block number.
	blockNumSet := make(map[uint64]struct{})
	for _, tx := range userTxs.txs {
		receipt, err := seqClient.TransactionReceipt(ctx, tx)
		Require(t, err)
		blockNum := receipt.BlockNumber.Uint64()
		blockNumSet[blockNum] = struct{}{}
	}
	blockNums := make([]uint64, 0, len(blockNumSet))
	for n := range blockNumSet {
		blockNums = append(blockNums, n)
	}
	slices.Sort[[]uint64](blockNums)
	t.Log(blockNums)

	// For a slice of "scores" assigned to txs by a bid policy, we check
	// that tx scores are sorted in descending order before any 0. That is, the policy
	// should sequence txs by score like this:
	// [4, 3, 2, 1, 0, 0, 0].
	checkExpressLaneScoresComeFirst := func(slice []uint64) bool {
		foundZero := false
		for _, value := range slice {
			if value == 0 {
				foundZero = true
			} else if value == 1 && foundZero {
				// If a 1 is found after a 0, return false
				return false
			}
		}
		return true
	}

	policyScorer := boostpolicies.BidScorer{}
	// For the txs within each block, we check those with a scoring function of 1 come first and always before
	// those with a score of 0. These "express lane" txs are those that are using the tipping tx type.
	for _, blockNum := range blockNums {
		block, err := seqClient.BlockByNumber(ctx, new(big.Int).SetUint64(blockNum))
		Require(t, err)

		blockTxs := block.Transactions()

		// Strip the first tx (sequencer message not relevant).
		blockTxs = blockTxs[1:]
		txScores := make([]uint64, len(blockTxs))
		for i, tx := range blockTxs {
			txScores[i] = policyScorer.ScoreTx(tx)
		}
		if !checkExpressLaneScoresComeFirst(txScores) {
			t.Fatal("Expected express lane txs to be ordered first in a block")
		}
	}
}

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
