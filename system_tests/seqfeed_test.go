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

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/broadcaster/backlog"
	"github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/relay"
	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/timeboost/bindings"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
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
	logHandler := testhelpers.InitTestLog(t, log.LvlTrace)

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

	if logHandler.WasLogged(arbnode.BlockHashMismatchLogMsg) {
		t.Fatal("BlockHashMismatchLogMsg was logged unexpectedly")
	}
}

func TestSequencerFeed_ExpressLaneAuction(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builderSeq := NewNodeBuilder(ctx).DefaultConfig(t, true)

	builderSeq.l2StackConfig.HTTPHost = "localhost"
	builderSeq.l2StackConfig.HTTPPort = 9567
	builderSeq.l2StackConfig.HTTPModules = []string{"eth", "arb", "debug", "timeboost"}
	builderSeq.nodeConfig.Feed.Output = *newBroadcasterConfigTest()
	builderSeq.execConfig.Sequencer.Enable = true
	builderSeq.execConfig.Sequencer.Timeboost = gethexec.TimeboostConfig{
		Enable:               true,
		ExpressLaneAdvantage: time.Millisecond * 200,
	}
	cleanupSeq := builderSeq.Build(t)
	defer cleanupSeq()
	seqInfo, seqNode, seqClient := builderSeq.L2Info, builderSeq.L2.ConsensusNode, builderSeq.L2.Client
	t.Logf("Sequencer endpoint %s", seqNode.Stack.HTTPEndpoint())

	auctionAddr := builderSeq.L2.ExecNode.Sequencer.ExpressLaneAuction()
	erc20Addr := builderSeq.L2.ExecNode.Sequencer.ExpressLaneERC20()

	// Seed the accounts on L2.
	seqInfo.GenerateAccount("Alice")
	tx := seqInfo.PrepareTx("Owner", "Alice", seqInfo.TransferGas, big.NewInt(1e18), nil)
	Require(t, seqClient.SendTransaction(ctx, tx))
	_, err := EnsureTxSucceeded(ctx, seqClient, tx)
	Require(t, err)
	seqInfo.GenerateAccount("Bob")
	tx = seqInfo.PrepareTx("Owner", "Bob", seqInfo.TransferGas, big.NewInt(1e18), nil)
	Require(t, seqClient.SendTransaction(ctx, tx))
	_, err = EnsureTxSucceeded(ctx, seqClient, tx)
	Require(t, err)
	t.Logf("Alice %+v and Bob %+v", seqInfo.Accounts["Alice"], seqInfo.Accounts["Bob"])

	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionAddr, builderSeq.L1.Client)
	Require(t, err)
	_ = seqInfo
	_ = seqClient
	l1client := builderSeq.L1.Client

	// We approve the spending of the erc20 for the autonomous auction contract and bid receiver
	// for both Alice and Bob.
	bidReceiverAddr := common.HexToAddress("0x3424242424242424242424242424242424242424")
	aliceOpts := builderSeq.L1Info.GetDefaultTransactOpts("Alice", ctx)
	bobOpts := builderSeq.L1Info.GetDefaultTransactOpts("Bob", ctx)

	maxUint256 := big.NewInt(1)
	maxUint256.Lsh(maxUint256, 256).Sub(maxUint256, big.NewInt(1))
	erc20, err := bindings.NewMockERC20(erc20Addr, builderSeq.L1.Client)
	Require(t, err)

	tx, err = erc20.Approve(
		&aliceOpts, auctionAddr, maxUint256,
	)
	Require(t, err)
	if _, err = bind.WaitMined(ctx, l1client, tx); err != nil {
		t.Fatal(err)
	}
	tx, err = erc20.Approve(
		&aliceOpts, bidReceiverAddr, maxUint256,
	)
	Require(t, err)
	if _, err = bind.WaitMined(ctx, l1client, tx); err != nil {
		t.Fatal(err)
	}
	tx, err = erc20.Approve(
		&bobOpts, auctionAddr, maxUint256,
	)
	Require(t, err)
	if _, err = bind.WaitMined(ctx, l1client, tx); err != nil {
		t.Fatal(err)
	}
	tx, err = erc20.Approve(
		&bobOpts, bidReceiverAddr, maxUint256,
	)
	Require(t, err)
	if _, err = bind.WaitMined(ctx, l1client, tx); err != nil {
		t.Fatal(err)
	}

	// Set up an autonomous auction contract service that runs in the background in this test.
	auctionContractOpts := builderSeq.L1Info.GetDefaultTransactOpts("AuctionContract", ctx)
	chainId, err := l1client.ChainID(ctx)
	Require(t, err)

	// Set up the auctioneer RPC service.
	stackConf := node.Config{
		DataDir:             "", // ephemeral.
		HTTPPort:            9372,
		HTTPModules:         []string{timeboost.AuctioneerNamespace},
		HTTPVirtualHosts:    []string{"localhost"},
		HTTPTimeouts:        rpc.DefaultHTTPTimeouts,
		WSPort:              9373,
		WSModules:           []string{timeboost.AuctioneerNamespace},
		GraphQLVirtualHosts: []string{"localhost"},
		P2P: p2p.Config{
			ListenAddr:  "",
			NoDial:      true,
			NoDiscovery: true,
		},
	}
	stack, err := node.New(&stackConf)
	Require(t, err)
	auctioneer, err := timeboost.NewAuctioneer(
		&auctionContractOpts, []uint64{chainId.Uint64()}, stack, builderSeq.L1.Client, auctionContract,
	)
	Require(t, err)

	go auctioneer.Start(ctx)

	// Set up a bidder client for Alice and Bob.
	alicePriv := builderSeq.L1Info.Accounts["Alice"].PrivateKey
	alice, err := timeboost.NewBidderClient(
		ctx,
		"alice",
		&timeboost.Wallet{
			TxOpts:  &aliceOpts,
			PrivKey: alicePriv,
		},
		l1client,
		auctionAddr,
		auctioneer,
	)
	Require(t, err)

	bobPriv := builderSeq.L1Info.Accounts["Bob"].PrivateKey
	bob, err := timeboost.NewBidderClient(
		ctx,
		"bob",
		&timeboost.Wallet{
			TxOpts:  &bobOpts,
			PrivKey: bobPriv,
		},
		l1client,
		auctionAddr,
		auctioneer,
	)
	Require(t, err)

	// Wait until the initial round.
	info, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	Require(t, err)
	timeToWait := time.Until(time.Unix(int64(info.OffsetTimestamp), 0))
	t.Log("Waiting until the initial round", timeToWait, time.Unix(int64(info.OffsetTimestamp), 0))
	<-time.After(timeToWait)

	t.Log("Started auction master stack and bid clients")
	Require(t, alice.Deposit(ctx, big.NewInt(5)))
	Require(t, bob.Deposit(ctx, big.NewInt(5)))

	// Wait until the next timeboost round + a few milliseconds.
	now := time.Now()
	roundDuration := time.Minute
	waitTime := roundDuration - time.Duration(now.Second())*time.Second - time.Duration(now.Nanosecond())
	t.Logf("Alice and Bob are now deposited into the autonomous auction contract, waiting %v for bidding round...", waitTime)
	time.Sleep(waitTime)
	time.Sleep(time.Second * 5)

	// We are now in the bidding round, both issue their bids. Bob will win.
	t.Log("Alice and Bob now submitting their bids")
	aliceBid, err := alice.Bid(ctx, big.NewInt(1), aliceOpts.From)
	Require(t, err)
	bobBid, err := bob.Bid(ctx, big.NewInt(2), bobOpts.From)
	Require(t, err)
	t.Logf("Alice bid %+v", aliceBid)
	t.Logf("Bob bid %+v", bobBid)

	// Subscribe to auction resolutions and wait for Bob to win the auction.
	winner, winnerRound := awaitAuctionResolved(t, ctx, l1client, auctionContract)

	// Verify Bob owns the express lane this round.
	if winner != bobOpts.From {
		t.Fatal("Bob should have won the express lane auction")
	}
	t.Log("Bob won the express lane auction for upcoming round, now waiting for that round to start...")

	// Wait until the round that Bob owns the express lane for.
	now = time.Now()
	waitTime = roundDuration - time.Duration(now.Second())*time.Second - time.Duration(now.Nanosecond())
	time.Sleep(waitTime)

	currRound := timeboost.CurrentRound(time.Unix(int64(info.OffsetTimestamp), 0), roundDuration)
	t.Log("curr round", currRound)
	if currRound != winnerRound {
		now = time.Now()
		waitTime = roundDuration - time.Duration(now.Second())*time.Second - time.Duration(now.Nanosecond())
		t.Log("Not express lane round yet, waiting for next round", waitTime)
		time.Sleep(waitTime)
	}
	filterOpts := &bind.FilterOpts{
		Context: ctx,
		Start:   0,
		End:     nil,
	}
	it, err := auctionContract.FilterAuctionResolved(filterOpts, nil, nil, nil)
	Require(t, err)
	bobWon := false
	for it.Next() {
		if it.Event.FirstPriceBidder == bobOpts.From {
			bobWon = true
		}
	}
	if !bobWon {
		t.Fatal("Bob should have won the auction")
	}

	t.Log("Now submitting txs to sequencer")

	// Prepare a client that can submit txs to the sequencer via the express lane.
	seqDial, err := rpc.Dial("http://localhost:9567")
	Require(t, err)
	expressLaneClient := timeboost.NewExpressLaneClient(
		bobPriv,
		chainId.Uint64(),
		time.Unix(int64(info.OffsetTimestamp), 0),
		roundDuration,
		auctionAddr,
		seqDial,
	)
	expressLaneClient.StopWaiter.Start(ctx, expressLaneClient)

	// During the express lane around, Bob sends txs always 150ms later than Alice, but Alice's
	// txs end up getting delayed by 200ms as she is not the express lane controller.
	// In the end, Bob's txs should be ordered before Alice's during the round.
	var wg sync.WaitGroup
	wg.Add(2)
	aliceTx := seqInfo.PrepareTx("Alice", "Owner", seqInfo.TransferGas, big.NewInt(1e12), nil)
	go func(w *sync.WaitGroup) {
		defer w.Done()
		err = seqClient.SendTransaction(ctx, aliceTx)
		Require(t, err)
	}(&wg)

	bobBoostableTx := seqInfo.PrepareTx("Bob", "Owner", seqInfo.TransferGas, big.NewInt(1e12), nil)
	go func(w *sync.WaitGroup) {
		defer w.Done()
		time.Sleep(time.Millisecond * 10)
		err = expressLaneClient.SendTransaction(ctx, bobBoostableTx)
		Require(t, err)
	}(&wg)
	wg.Wait()

	time.Sleep(time.Minute)

	// After round is done, verify that Bob beats Alice in the final sequence.
	aliceReceipt, err := seqClient.TransactionReceipt(ctx, aliceTx.Hash())
	Require(t, err)
	aliceBlock := aliceReceipt.BlockNumber.Uint64()
	bobReceipt, err := seqClient.TransactionReceipt(ctx, bobBoostableTx.Hash())
	Require(t, err)
	bobBlock := bobReceipt.BlockNumber.Uint64()

	if aliceBlock < bobBlock {
		t.Fatal("Bob should have been sequenced before Alice with express lane")
	} else if aliceBlock == bobBlock {
		t.Log("Sequenced in same output block")
		block, err := seqClient.BlockByNumber(ctx, new(big.Int).SetUint64(aliceBlock))
		Require(t, err)
		findTransactionIndex := func(transactions types.Transactions, txHash common.Hash) int {
			for index, tx := range transactions {
				if tx.Hash() == txHash {
					return index
				}
			}
			return -1
		}
		txes := block.Transactions()
		indexA := findTransactionIndex(txes, aliceTx.Hash())
		indexB := findTransactionIndex(txes, bobBoostableTx.Hash())
		if indexA == -1 || indexB == -1 {
			t.Fatal("Did not find txs in block")
		}
		if indexA < indexB {
			t.Fatal("Bob should have been sequenced before Alice with express lane")
		}
	}
	time.Sleep(time.Hour)
}

func awaitAuctionResolved(
	t *testing.T,
	ctx context.Context,
	client *ethclient.Client,
	contract *express_lane_auctiongen.ExpressLaneAuction,
) (common.Address, uint64) {
	fromBlock, err := client.BlockNumber(ctx)
	Require(t, err)
	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return common.Address{}, 0
		case <-ticker.C:
			latestBlock, err := client.HeaderByNumber(ctx, nil)
			if err != nil {
				t.Log("Could not get latest header", err)
				continue
			}
			toBlock := latestBlock.Number.Uint64()
			if fromBlock == toBlock {
				continue
			}
			filterOpts := &bind.FilterOpts{
				Context: ctx,
				Start:   fromBlock,
				End:     &toBlock,
			}
			it, err := contract.FilterAuctionResolved(filterOpts, nil, nil, nil)
			if err != nil {
				t.Log("Could not filter auction resolutions", err)
				continue
			}
			for it.Next() {
				return it.Event.FirstPriceBidder, it.Event.Round
			}
			fromBlock = toBlock
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
	builder.L2Info.GetInfoWithPrivKey("Owner").Nonce.Add(^uint64(0)) // Use same l2info object for different l2s
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

func testBlockHashComparison(t *testing.T, blockHash *common.Hash, mustMismatch bool) {
	logHandler := testhelpers.InitTestLog(t, log.LvlTrace)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	backlogConfiFetcher := func() *backlog.Config {
		return &backlog.DefaultTestConfig
	}
	bklg := backlog.NewBacklog(backlogConfiFetcher)

	wsBroadcastServer := wsbroadcastserver.NewWSBroadcastServer(
		newBroadcasterConfigTest,
		bklg,
		412346,
		nil,
	)
	err := wsBroadcastServer.Initialize()
	if err != nil {
		t.Fatal("error initializing wsBroadcastServer:", err)
	}
	err = wsBroadcastServer.Start(ctx)
	if err != nil {
		t.Fatal("error starting wsBroadcastServer:", err)
	}
	defer wsBroadcastServer.StopAndWait()

	port := wsBroadcastServer.ListenerAddr().(*net.TCPAddr).Port

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)
	cleanup := builder.Build(t)
	defer cleanup()
	testClient := builder.L2

	userAccount := "User2"
	builder.L2Info.GenerateAccount(userAccount)
	tx := builder.L2Info.PrepareTx("Owner", userAccount, builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	l1IncomingMsgHeader := arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_L2Message,
		Poster:      l1pricing.BatchPosterAddress,
		BlockNumber: 29,
		Timestamp:   1715295980,
		RequestId:   nil,
		L1BaseFee:   nil,
	}
	l1IncomingMsg, err := gethexec.MessageFromTxes(
		&l1IncomingMsgHeader,
		types.Transactions{tx},
		[]error{nil},
	)
	Require(t, err)

	broadcastMessage := message.BroadcastMessage{
		Version: 1,
		Messages: []*message.BroadcastFeedMessage{
			{
				SequenceNumber: 1,
				Message: arbostypes.MessageWithMetadata{
					Message:             l1IncomingMsg,
					DelayedMessagesRead: 1,
				},
				BlockHash: blockHash,
			},
		},
	}
	wsBroadcastServer.Broadcast(&broadcastMessage)

	// By now, even though block hash mismatch, the transaction should still be processed
	_, err = WaitForTx(ctx, testClient.Client, tx.Hash(), time.Second*15)
	if err != nil {
		t.Fatal("error waiting for tx:", err)
	}
	l2balance, err := testClient.Client.BalanceAt(ctx, builder.L2Info.GetAddress(userAccount), nil)
	if err != nil {
		t.Fatal("error getting balance:", err)
	}
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}

	mismatched := logHandler.WasLogged(arbnode.BlockHashMismatchLogMsg)
	if mustMismatch && !mismatched {
		t.Fatal("Failed to log BlockHashMismatchLogMsg")
	} else if !mustMismatch && mismatched {
		t.Fatal("BlockHashMismatchLogMsg was logged unexpectedly")
	}
}

func TestBlockHashFeedMismatch(t *testing.T) {
	blockHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	testBlockHashComparison(t, &blockHash, true)
}

func TestBlockHashFeedNil(t *testing.T) {
	testBlockHashComparison(t, nil, false)
}
