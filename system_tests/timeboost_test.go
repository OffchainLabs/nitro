package arbtest

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/seq-coordinator-manager/rediscoordinator"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/timeboost/bindings"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestTimeboostTxsTimeoutByBlock(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "*")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	})

	numTxs, blockBasedTimeout := uint64(10), uint64(5)
	auctionContractAddr, aliceBidderClient, bobBidderClient, roundDuration, builderSeq, cleanupSeq, forwarder, cleanupForwarder := setupExpressLaneAuction(t, tmpDir, ctx, withForwardingSeq, blockBasedTimeout)
	seqClient, seqInfo := builderSeq.L2.Client, builderSeq.L2Info
	defer cleanupSeq()
	defer cleanupForwarder()
	seqInfo.GenerateAccount("User2")

	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, seqClient)
	Require(t, err)
	rawRoundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	Require(t, err)
	roundTimingInfo, err := timeboost.NewRoundTimingInfo(rawRoundTimingInfo)
	Require(t, err)

	placeBidsAndDecideWinner(t, ctx, seqClient, seqInfo, auctionContract, "Bob", "Alice", bobBidderClient, aliceBidderClient, roundDuration)
	time.Sleep(roundTimingInfo.TimeTilNextRound())

	chainId, err := seqClient.ChainID(ctx)
	Require(t, err)

	// Prepare a client that can submit txs to the sequencer via the express lane.
	bobPriv := seqInfo.Accounts["Bob"].PrivateKey
	forwardingSeqDial, err := rpc.Dial(forwarder.ConsensusNode.Stack.HTTPEndpoint())
	Require(t, err)
	expressLaneClient := newExpressLaneClient(
		bobPriv,
		chainId,
		*roundTimingInfo,
		auctionContractAddr,
		forwardingSeqDial,
	)
	expressLaneClient.Start(ctx)

	size := 80 * 1024 // 80 KB
	data := make([]byte, size)
	_, err = rand.Read(data)
	Require(t, err)

	var txs types.Transactions
	for i := uint64(0); i < numTxs; i++ {
		txs = append(txs, seqInfo.PrepareTx("Owner", "User2", 700000000, big.NewInt(1e8), data)) // this tx should consume one block
	}
	// Buffer future sequence numbered txs
	for seq := uint64(1); seq < numTxs; seq++ {
		Require(t, expressLaneClient.QueueTransactionWithSequence(ctx, txs[seq], seq))
	}
	// Send tx with seq=0 that releases all the buffered txs and wait for the block to be produced
	Require(t, expressLaneClient.QueueTransactionWithSequence(ctx, txs[0], 0))
	rec, err := builderSeq.L2.EnsureTxSucceeded(txs[0])
	Require(t, err)
	firstBlockNum := rec.BlockNumber.Uint64()
	t.Logf("tx: 0 was sequenced in block: %d", firstBlockNum)

	// Verify that QueueTimeoutInBlocks config option is respected
	for i := uint64(1); i < numTxs; i++ {
		rec, err := builderSeq.L2.EnsureTxSucceeded(txs[i])
		if err == nil {
			if i >= blockBasedTimeout {
				t.Fatalf("more txs sequenced than allowed. sequencedCount: %d, allowed: %d", i+1, blockBasedTimeout)
			}
			t.Logf("tx: %d was sequenced in block: %d", i, rec.BlockNumber)
			if rec.BlockNumber.Uint64() != firstBlockNum+i {
				t.Fatalf("tx: %d sequenced in unexpected block: %d, expected to be sequenced in block: %d", i, rec.BlockNumber, firstBlockNum+i)
			}
		} else {
			// There's a possibility that all the EL txs might get stale blockStamp by 1, in that case we conservatively
			// check if at least blockBasedTimeout-1 txs have been sequenced successfully
			if i < blockBasedTimeout-1 {
				t.Fatalf("lesser than expected txs were sequenced. sequencedCount: %d, minimumRequired: %d", i, blockBasedTimeout-1)
			}
			t.Logf("tx: %d was not sequenced into a block", i)
		}
	}
}

func TestTimeboostAuctionResolutionDuringATie(t *testing.T) {
	testAuctionResolutionDuringATie(t, false)
}

func TestTimeboostAuctionResolutionDuringATieMultipleRuns(t *testing.T) {
	t.Skip("This test is skipped in CI as it might probably take too long to complete")
	testAuctionResolutionDuringATie(t, true)
}

func testAuctionResolutionDuringATie(t *testing.T, multiRuns bool) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "*")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	})

	auctionContractAddr, aliceBidderClient, bobBidderClient, _, builderSeq, cleanupSeq, _, _ := setupExpressLaneAuction(t, tmpDir, ctx, 0, 0)
	_, seqClient, seqInfo := builderSeq.L2.ConsensusNode, builderSeq.L2.Client, builderSeq.L2Info
	defer cleanupSeq()

	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, seqClient)
	Require(t, err)
	rawRoundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	Require(t, err)
	roundTimingInfo, err := timeboost.NewRoundTimingInfo(rawRoundTimingInfo)
	Require(t, err)
	domainSeparator, err := auctionContract.DomainSeparator(&bind.CallOpts{Context: ctx})
	Require(t, err)

	aliceAddr := seqInfo.GetAddress("Alice")
	bobAddr := seqInfo.GetAddress("Bob")
	var aliceHasWon, bobHasWon bool

	for {
		// For the next round, we will send equal bids and verify we get the correct winner
		t.Logf("Alice and Bob now submitting their equal bids at %v", time.Now())
		aliceBid, err := aliceBidderClient.Bid(ctx, big.NewInt(1), aliceAddr)
		Require(t, err)
		bobBid, err := bobBidderClient.Bid(ctx, big.NewInt(1), bobAddr)
		Require(t, err)
		t.Logf("Alice bid %+v", aliceBid)
		t.Logf("Bob bid %+v", bobBid)

		// Check if bidHash from ToEIP712Hash matches with the calculation in auction contract
		matchBidHash := func(bid *timeboost.Bid) {
			expectedBidHash, err := auctionContract.GetBidHash(&bind.CallOpts{}, bid.Round, bid.ExpressLaneController, bid.Amount)
			Require(t, err)
			bidHash, err := bid.ToEIP712Hash(domainSeparator)
			Require(t, err)
			if !bytes.Equal(expectedBidHash[:], bidHash.Bytes()) {
				t.Fatalf("bid hash mismatch with contract. Want: %v, Got: %v", expectedBidHash, bidHash.Bytes())
			}
		}
		matchBidHash(aliceBid)
		matchBidHash(bobBid)

		// Subscribe to auction resolutions and wait for a winner
		winnerAddr, _ := awaitAuctionResolved(t, ctx, seqClient, auctionContract)

		// Get expected Winner on the GO side
		toValidatedBid := func(bidder common.Address, bid *timeboost.Bid) *timeboost.ValidatedBid {
			return &timeboost.ValidatedBid{
				ExpressLaneController:  bid.ExpressLaneController,
				Amount:                 bid.Amount,
				Signature:              bid.Signature,
				ChainId:                bid.ChainId,
				AuctionContractAddress: bid.AuctionContractAddress,
				Round:                  bid.Round,
				Bidder:                 bidder,
			}

		}

		var expectedWinner common.Address
		aliceBigIntHash := toValidatedBid(aliceAddr, aliceBid).BigIntHash(domainSeparator)
		BobBigIntHash := toValidatedBid(bobAddr, bobBid).BigIntHash(domainSeparator)
		if aliceBigIntHash.Cmp(BobBigIntHash) > 0 {
			expectedWinner = aliceAddr
		} else if aliceBigIntHash.Cmp(BobBigIntHash) < 0 {
			expectedWinner = bobAddr
		}

		// If tie can't be broken by BigIntHash, then whoever is picked first is the winner- auction contract will agree with that as well
		if (expectedWinner != common.Address{}) {
			// Verify that the winner on the GO side is the same on the contract side
			if expectedWinner != winnerAddr {
				t.Fatalf("Unexpected auction winner in case of a tie. Want: %s, Got: %s", expectedWinner, winnerAddr)
			}
		}

		if !multiRuns {
			break
		}

		if winnerAddr == aliceAddr {
			aliceHasWon = true
		} else if winnerAddr == bobAddr {
			bobHasWon = true
		} else {
			t.Fatalf("Unexpected winner of the auction round: %s", winnerAddr)
		}

		// Both bidders winning a tie has been tested
		if aliceHasWon && bobHasWon {
			break
		}
		time.Sleep(roundTimingInfo.TimeTilNextRound())
	}
}

func TestTimeboostExpressLaneTxsHandlingDuringSequencerSwapDueToPriorities(t *testing.T) {
	testTxsHandlingDuringSequencerSwap(t, false)
}

func TestTimeboostExpressLaneTxsHandlingDuringSequencerSwapDueToActiveSequencerCrashing(t *testing.T) {
	testTxsHandlingDuringSequencerSwap(t, true)
}

func testTxsHandlingDuringSequencerSwap(t *testing.T, dueToCrash bool) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "*")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	})

	auctionContractAddr, aliceBidderClient, bobBidderClient, roundDuration, builderSeq, cleanupSeq, forwarder, cleanupForwarder := setupExpressLaneAuction(t, tmpDir, ctx, withForwardingSeq, 0)
	seqB, seqClientB, seqInfo := builderSeq.L2.ConsensusNode, builderSeq.L2.Client, builderSeq.L2Info
	seqA := forwarder.ConsensusNode
	if !dueToCrash {
		defer cleanupSeq()
	}
	defer cleanupForwarder()

	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, seqClientB)
	Require(t, err)
	rawRoundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	Require(t, err)
	roundTimingInfo, err := timeboost.NewRoundTimingInfo(rawRoundTimingInfo)
	Require(t, err)

	placeBidsAndDecideWinner(t, ctx, seqClientB, seqInfo, auctionContract, "Bob", "Alice", bobBidderClient, aliceBidderClient, roundDuration)
	time.Sleep(roundTimingInfo.TimeTilNextRound())

	// Prepare a client that can submit txs to the sequencer via the express lane.
	chainId, err := seqClientB.ChainID(ctx)
	Require(t, err)
	bobPriv := seqInfo.Accounts["Bob"].PrivateKey
	createExpressLaneClientFor := func(url string) *expressLaneClient {
		forwardingSeqDial, err := rpc.Dial(url)
		Require(t, err)
		expressLaneClient := newExpressLaneClient(
			bobPriv,
			chainId,
			*roundTimingInfo,
			auctionContractAddr,
			forwardingSeqDial,
		)
		expressLaneClient.Start(ctx)
		return expressLaneClient
	}
	expressLaneClientB := createExpressLaneClientFor(seqB.Stack.HTTPEndpoint())
	expressLaneClientA := createExpressLaneClientFor(seqA.Stack.HTTPEndpoint())

	verifyControllerAdvantage(t, ctx, seqClientB, expressLaneClientB, seqInfo, "Bob", "Alice")

	currNonce, err := seqClientB.PendingNonceAt(ctx, seqInfo.GetAddress("Alice"))
	Require(t, err)
	seqInfo.GetInfoWithPrivKey("Alice").Nonce.Store(currNonce)

	// Send txs out of order
	var txs types.Transactions
	txs = append(txs, seqInfo.PrepareTx("Alice", "Owner", seqInfo.TransferGas, big.NewInt(1), nil)) // currNonce
	txs = append(txs, seqInfo.PrepareTx("Alice", "Owner", seqInfo.TransferGas, big.NewInt(1), nil)) // currNonce + 1
	txs = append(txs, seqInfo.PrepareTx("Alice", "Owner", seqInfo.TransferGas, big.NewInt(1), nil)) // currNonce + 2
	txs = append(txs, seqInfo.PrepareTx("Alice", "Owner", seqInfo.TransferGas, big.NewInt(1), nil)) // currNonce + 3

	// We send three txs- 0,2 and 3 to the current active sequencer=B
	go func() {
		_ = expressLaneClientB.SendTransactionWithSequence(ctx, txs[3], 4)
	}()
	go func() {
		_ = expressLaneClientB.SendTransactionWithSequence(ctx, txs[2], 3)
	}()
	time.Sleep(time.Second) // Wait for txs to be submitted
	err = expressLaneClientB.SendTransactionWithSequence(ctx, txs[0], 1)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, seqClientB, txs[0])
	Require(t, err)

	// Set reader and writer coordinators for redis
	redisCoordinatorGetter, err := redisutil.NewRedisCoordinator(builderSeq.nodeConfig.SeqCoordinator.RedisUrl)
	Require(t, err)
	currentChosen, err := redisCoordinatorGetter.CurrentChosenSequencer(ctx)
	Require(t, err)
	if currentChosen != seqB.Stack.HTTPEndpoint() {
		t.Fatalf("unexepcted current chosen sequencer. Want: %s, Got: %s", seqB.Stack.HTTPEndpoint(), currentChosen)
	}
	redisCoordinatorSetter := &rediscoordinator.RedisCoordinator{RedisCoordinator: redisCoordinatorGetter}

	if dueToCrash {
		// Shutdown the current active sequencer
		t.Log("Attempting to stop current chosen sequencer")
		seqB.StopAndWait()
	} else {
		// Change priorities to make sequencer=A the chosen and verify that the update went through
		t.Log("Change coordinator priorities to switch active sequencer")
		Require(t, redisCoordinatorSetter.UpdatePriorities(ctx, []string{seqA.Stack.HTTPEndpoint(), seqB.Stack.HTTPEndpoint()}))
	}

	// Wait for chosen sequencer to change on redis
	for {
		currentChosen, err := redisCoordinatorGetter.CurrentChosenSequencer(ctx)
		Require(t, err)
		if currentChosen == seqA.Stack.HTTPEndpoint() {
			break
		}
		t.Logf("waiting for chosen sequencer to change to: %s, currently: %s", seqA.Stack.HTTPEndpoint(), currentChosen)
		time.Sleep(time.Second)
	}

	// Send the tx=1 that should be sequenced by the new active sequencer along with the future seq num txs=2,3 synced from redis
	err = expressLaneClientA.SendTransactionWithSequence(ctx, txs[1], 2)
	Require(t, err)

	var txReceipts types.Receipts
	for _, tx := range txs[1:] {
		receipt, err := EnsureTxSucceeded(ctx, forwarder.Client, tx)
		Require(t, err)
		txReceipts = append(txReceipts, receipt)
	}

	if !(txReceipts[0].BlockNumber.Cmp(txReceipts[1].BlockNumber) <= 0 &&
		txReceipts[1].BlockNumber.Cmp(txReceipts[2].BlockNumber) <= 0) {
		t.Fatal("incorrect ordering of txs acceptance, lower sequence number txs should appear in earlier block")
	}

	if txReceipts[0].BlockNumber.Cmp(txReceipts[1].BlockNumber) == 0 &&
		txReceipts[0].TransactionIndex > txReceipts[1].TransactionIndex {
		t.Fatal("incorrect ordering of txs in a block, lower sequence number txs should appear earlier")
	}

	if txReceipts[1].BlockNumber.Cmp(txReceipts[2].BlockNumber) == 0 &&
		txReceipts[1].TransactionIndex > txReceipts[2].TransactionIndex {
		t.Fatal("incorrect ordering of txs in a block, lower sequence number txs should appear earlier")
	}
}

func TestTimeboostForwardingExpressLaneTxs(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "*")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	})

	auctionContractAddr, aliceBidderClient, bobBidderClient, roundDuration, builderSeq, cleanupSeq, forwarder, cleanupForwarder := setupExpressLaneAuction(t, tmpDir, ctx, withForwardingSeq, 0)
	seqClient, seqInfo := builderSeq.L2.Client, builderSeq.L2Info
	defer cleanupSeq()
	defer cleanupForwarder()

	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, seqClient)
	Require(t, err)
	rawRoundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	Require(t, err)
	roundTimingInfo, err := timeboost.NewRoundTimingInfo(rawRoundTimingInfo)
	Require(t, err)

	placeBidsAndDecideWinner(t, ctx, seqClient, seqInfo, auctionContract, "Bob", "Alice", bobBidderClient, aliceBidderClient, roundDuration)
	time.Sleep(roundTimingInfo.TimeTilNextRound())

	chainId, err := seqClient.ChainID(ctx)
	Require(t, err)

	// Prepare a client that can submit txs to the sequencer via the express lane.
	bobPriv := seqInfo.Accounts["Bob"].PrivateKey
	forwardingSeqDial, err := rpc.Dial(forwarder.ConsensusNode.Stack.HTTPEndpoint())
	Require(t, err)
	expressLaneClient := newExpressLaneClient(
		bobPriv,
		chainId,
		*roundTimingInfo,
		auctionContractAddr,
		forwardingSeqDial,
	)
	expressLaneClient.Start(ctx)

	verifyControllerAdvantage(t, ctx, seqClient, expressLaneClient, seqInfo, "Bob", "Alice")
}

func TestTimeboostExpressLaneTransactionHandlingComplex(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "*")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	})

	auctionContractAddr, aliceBidderClient, bobBidderClient, roundDuration, builderSeq, cleanupSeq, _, _ := setupExpressLaneAuction(t, tmpDir, ctx, 0, 0)
	seq, seqClient, seqInfo := builderSeq.L2.ConsensusNode, builderSeq.L2.Client, builderSeq.L2Info
	defer cleanupSeq()

	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, seqClient)
	Require(t, err)
	rawRoundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	Require(t, err)
	roundTimingInfo, err := timeboost.NewRoundTimingInfo(rawRoundTimingInfo)
	Require(t, err)

	// Prepare clients that can submit txs to the sequencer via the express lane.
	chainId, err := seqClient.ChainID(ctx)
	Require(t, err)
	seqDial, err := rpc.Dial(seq.Stack.HTTPEndpoint())
	Require(t, err)
	createExpressLaneClientFor := func(name string) (*expressLaneClient, bind.TransactOpts) {
		priv := seqInfo.Accounts[name].PrivateKey
		expressLaneClient := newExpressLaneClient(
			priv,
			chainId,
			*roundTimingInfo,
			auctionContractAddr,
			seqDial,
		)
		expressLaneClient.Start(ctx)
		transacOpts := seqInfo.GetDefaultTransactOpts(name, ctx)
		transacOpts.NoSend = true
		return expressLaneClient, transacOpts
	}
	bobExpressLaneClient, _ := createExpressLaneClientFor("Bob")
	aliceExpressLaneClient, _ := createExpressLaneClientFor("Alice")

	// Bob will win the auction and become controller for next round = x
	placeBidsAndDecideWinner(t, ctx, seqClient, seqInfo, auctionContract, "Bob", "Alice", bobBidderClient, aliceBidderClient, roundDuration)
	time.Sleep(roundTimingInfo.TimeTilNextRound())

	// Check that Bob's tx gets priority since he's the controller
	verifyControllerAdvantage(t, ctx, seqClient, bobExpressLaneClient, seqInfo, "Bob", "Alice")

	currNonce, err := seqClient.PendingNonceAt(ctx, seqInfo.GetAddress("Alice"))
	Require(t, err)
	seqInfo.GetInfoWithPrivKey("Alice").Nonce.Store(currNonce)
	unblockingTx := seqInfo.PrepareTx("Alice", "Owner", seqInfo.TransferGas, big.NewInt(1), nil)

	bobExpressLaneClient.Lock()
	currSeq := bobExpressLaneClient.sequence
	bobExpressLaneClient.Unlock()

	// Send bunch of future txs so that they are queued up waiting for the unblocking seq num tx
	var bobExpressLaneTxs types.Transactions
	for i := currSeq + 1; i < 1000; i++ {
		futureSeqTx := seqInfo.PrepareTx("Alice", "Owner", seqInfo.TransferGas, big.NewInt(1), nil)
		bobExpressLaneTxs = append(bobExpressLaneTxs, futureSeqTx)
		go func(tx *types.Transaction, seqNum uint64) {
			err := bobExpressLaneClient.SendTransactionWithSequence(ctx, tx, seqNum)
			t.Logf("got error for tx: hash-%s, seqNum-%d, err-%s", tx.Hash(), seqNum, err.Error())
		}(futureSeqTx, i)
	}

	// Alice will win the auction for next round = x+1
	placeBidsAndDecideWinner(t, ctx, seqClient, seqInfo, auctionContract, "Alice", "Bob", aliceBidderClient, bobBidderClient, roundDuration)

	time.Sleep(roundTimingInfo.TimeTilNextRound() - 500*time.Millisecond) // we'll wait till the 1/2 second mark to the next round and then send the unblocking tx

	Require(t, bobExpressLaneClient.SendTransactionWithSequence(ctx, unblockingTx, currSeq)) // the unblockingTx itself should ideally pass, but the released 1000 txs shouldn't affect the round for which alice has won the bid for

	time.Sleep(500 * time.Millisecond) // Wait for controller change after the current round's end

	// Check that Alice's tx gets priority since she's the controller
	verifyControllerAdvantage(t, ctx, seqClient, aliceExpressLaneClient, seqInfo, "Alice", "Bob")

	// Binary search and find how many of bob's futureSeqTxs were able to go through
	s, f := 0, len(bobExpressLaneTxs)-1
	for s < f {
		m := (s + f + 1) / 2
		_, err := seqClient.TransactionReceipt(ctx, bobExpressLaneTxs[m].Hash())
		if err != nil {
			f = m - 1
		} else {
			s = m
		}
	}
	t.Logf("%d of the total %d bob's pending txs were accepted", s+1, len(bobExpressLaneTxs))
}

func TestTimeboostExpressLaneTransactionHandling(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "*")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	})

	auctionContractAddr, aliceBidderClient, bobBidderClient, roundDuration, builderSeq, cleanupSeq, _, _ := setupExpressLaneAuction(t, tmpDir, ctx, 0, 0)
	seq, seqClient, seqInfo := builderSeq.L2.ConsensusNode, builderSeq.L2.Client, builderSeq.L2Info
	defer cleanupSeq()

	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, seqClient)
	Require(t, err)
	rawRoundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	Require(t, err)
	roundTimingInfo, err := timeboost.NewRoundTimingInfo(rawRoundTimingInfo)
	Require(t, err)

	placeBidsAndDecideWinner(t, ctx, seqClient, seqInfo, auctionContract, "Bob", "Alice", bobBidderClient, aliceBidderClient, roundDuration)
	time.Sleep(roundTimingInfo.TimeTilNextRound())

	chainId, err := seqClient.ChainID(ctx)
	Require(t, err)

	// Prepare a client that can submit txs to the sequencer via the express lane.
	bobPriv := seqInfo.Accounts["Bob"].PrivateKey
	seqDial, err := rpc.Dial(seq.Stack.HTTPEndpoint())
	Require(t, err)
	expressLaneClient := newExpressLaneClient(
		bobPriv,
		chainId,
		*roundTimingInfo,
		auctionContractAddr,
		seqDial,
	)
	expressLaneClient.Start(ctx)

	currNonce, err := seqClient.PendingNonceAt(ctx, seqInfo.GetAddress("Alice"))
	Require(t, err)
	seqInfo.GetInfoWithPrivKey("Alice").Nonce.Store(currNonce)

	// Send txs out of order
	var txs types.Transactions
	txs = append(txs, seqInfo.PrepareTx("Alice", "Owner", seqInfo.TransferGas, big.NewInt(1), nil)) // currNonce
	txs = append(txs, seqInfo.PrepareTx("Alice", "Owner", seqInfo.TransferGas, big.NewInt(1), nil)) // currNonce + 1
	txs = append(txs, seqInfo.PrepareTx("Alice", "Owner", seqInfo.TransferGas, big.NewInt(1), nil)) // currNonce + 2

	var wg sync.WaitGroup
	wg.Add(2) // We send two txs in out of order
	for i := uint64(2); i > 0; i-- {
		go func(w *sync.WaitGroup) {
			err := expressLaneClient.SendTransactionWithSequence(ctx, txs[i], i)
			Require(t, err)
			w.Done()
		}(&wg)
	}

	time.Sleep(time.Second) // Wait for both txs to be submitted

	// Send the first transaction which will unblock the future ones
	err = expressLaneClient.SendTransactionWithSequence(ctx, txs[0], 0) // we'll wait for the result
	Require(t, err)

	wg.Wait() // Make sure future sequence number txs that were sent earlier did not error

	var txReceipts types.Receipts
	for _, tx := range txs {
		receipt, err := seqClient.TransactionReceipt(ctx, tx.Hash())
		Require(t, err)
		txReceipts = append(txReceipts, receipt)
	}

	if !(txReceipts[0].BlockNumber.Cmp(txReceipts[1].BlockNumber) <= 0 &&
		txReceipts[1].BlockNumber.Cmp(txReceipts[2].BlockNumber) <= 0) {
		t.Fatal("incorrect ordering of txs acceptance, lower sequence number txs should appear in earlier block")
	}

	if txReceipts[0].BlockNumber.Cmp(txReceipts[1].BlockNumber) == 0 &&
		txReceipts[0].TransactionIndex > txReceipts[1].TransactionIndex {
		t.Fatal("incorrect ordering of txs in a block, lower sequence number txs should appear earlier")
	}

	if txReceipts[1].BlockNumber.Cmp(txReceipts[2].BlockNumber) == 0 &&
		txReceipts[1].TransactionIndex > txReceipts[2].TransactionIndex {
		t.Fatal("incorrect ordering of txs in a block, lower sequence number txs should appear earlier")
	}

	// Test that failed txs are given responses
	passTx := seqInfo.PrepareTx("Alice", "Owner", seqInfo.TransferGas, big.NewInt(1), nil)  // currNonce + 3
	passTx2 := seqInfo.PrepareTx("Alice", "Owner", seqInfo.TransferGas, big.NewInt(1), nil) // currNonce + 4

	seqInfo.GetInfoWithPrivKey("Alice").Nonce.Store(20)
	failTx := seqInfo.PrepareTx("Alice", "Owner", seqInfo.TransferGas, big.NewInt(1), nil)
	failTxDueToTimeout := seqInfo.PrepareTx("Alice", "Owner", seqInfo.TransferGas, big.NewInt(1), nil)

	currSeqNumber := uint64(3)
	wg.Add(2) // We send a failing and a passing tx with cummulative future seq numbers, followed by a unblocking seq num tx
	var failErr error
	go func(w *sync.WaitGroup) {
		failErr = expressLaneClient.SendTransactionWithSequence(ctx, failTx, currSeqNumber+1) // Should give out nonce too high error
		w.Done()
	}(&wg)

	time.Sleep(time.Second)

	go func(w *sync.WaitGroup) {
		err := expressLaneClient.SendTransactionWithSequence(ctx, passTx2, currSeqNumber+2)
		Require(t, err)
		w.Done()
	}(&wg)

	err = expressLaneClient.SendTransactionWithSequence(ctx, passTx, currSeqNumber)
	Require(t, err)

	wg.Wait()

	checkFailErr := func(reason string) {
		if failErr == nil {
			t.Fatal("incorrect express lane tx didn't fail upon submission")
		}
		if !strings.Contains(failErr.Error(), reason) {
			t.Fatalf("unexpected error string returned: %s", failErr.Error())
		}
	}
	checkFailErr("context deadline exceeded") // tx will be rejected with nonce too high error so wont appear in a block

	wg.Add(1)
	go func(w *sync.WaitGroup) {
		failErr = expressLaneClient.SendTransactionWithSequence(ctx, failTxDueToTimeout, currSeqNumber+4) // Should give out a tx aborted error as this tx is never processed
		w.Done()
	}(&wg)
	wg.Wait()

	checkFailErr("context deadline exceeded")
}

func dbKey(prefix []byte, pos uint64) []byte {
	var key []byte
	key = append(key, prefix...)
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, pos)
	key = append(key, data...)
	return key
}

func TestTimeboostBulkBlockMetadataFetcher(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.TransactionStreamer.TrackBlockMetadataFrom = 1
	httpConfig := genericconf.HTTPConfigDefault
	httpConfig.Addr = "127.0.0.1"
	httpConfig.Apply(builder.l2StackConfig)
	builder.execConfig.BlockMetadataApiCacheSize = 0 // Caching is disabled
	cleanupSeq := builder.Build(t)
	defer cleanupSeq()

	blockMetadataInputFeedPrefix := []byte("t")
	missingBlockMetadataInputFeedPrefix := []byte("x")

	// Generate blocks until current block is > 20
	arbDb := builder.L2.ConsensusNode.ArbDB
	builder.L2Info.GenerateAccount("User")
	user := builder.L2Info.GetDefaultTransactOpts("User", ctx)
	var latestL2 uint64
	var err error
	var lastTx *types.Transaction
	for i := 0; ; i++ {
		lastTx, _ = builder.L2.TransferBalanceTo(t, "Owner", util.RemapL1Address(user.From), big.NewInt(1e18), builder.L2Info)
		latestL2, err = builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
		if latestL2 > uint64(20) {
			break
		}
	}
	var sampleBulkData []common.BlockMetadata
	for i := 1; i <= int(latestL2); i++ {
		// #nosec G115
		blockMetadata := []byte{0, uint8(i)}
		sampleBulkData = append(sampleBulkData, blockMetadata)
		// #nosec G115
		Require(t, arbDb.Put(dbKey(blockMetadataInputFeedPrefix, uint64(i)), blockMetadata))
	}

	nodecfg := arbnode.ConfigDefaultL1NonSequencerTest()
	trackBlockMetadataFrom := uint64(3)
	nodecfg.TransactionStreamer.TrackBlockMetadataFrom = trackBlockMetadataFrom
	newNode, cleanupNewNode := builder.Build2ndNode(t, &SecondNodeParams{
		nodeConfig:  nodecfg,
		stackConfig: testhelpers.CreateStackConfigForTest(t.TempDir()),
	})
	defer cleanupNewNode()

	// Wait for second node to catchup via L1, since L1 doesn't have the blockMetadata, we ensure that messages are tracked with missingBlockMetadataInputFeedPrefix prefix
	_, err = WaitForTx(ctx, newNode.Client, lastTx.Hash(), time.Second*5)
	Require(t, err)

	arbDb = newNode.ConsensusNode.ArbDB

	// Introduce fragmentation
	blocksWithBlockMetadata := []uint64{8, 9, 10, 14, 16}
	for _, key := range blocksWithBlockMetadata {
		Require(t, arbDb.Put(dbKey(blockMetadataInputFeedPrefix, key), sampleBulkData[key-1]))
		Require(t, arbDb.Delete(dbKey(missingBlockMetadataInputFeedPrefix, key)))
	}

	// Check if all block numbers with missingBlockMetadataInputFeedPrefix are present as keys in arbDB and that no keys with blockMetadataInputFeedPrefix
	iter := arbDb.NewIterator(blockMetadataInputFeedPrefix, nil)
	pos := uint64(0)
	for iter.Next() {
		keyBytes := bytes.TrimPrefix(iter.Key(), blockMetadataInputFeedPrefix)
		if binary.BigEndian.Uint64(keyBytes) != blocksWithBlockMetadata[pos] {
			t.Fatalf("unexpected presence of blockMetadata, when blocks are synced via L1. msgSeqNum: %d, expectedMsgSeqNum: %d", binary.BigEndian.Uint64(keyBytes), blocksWithBlockMetadata[pos])
		}
		pos++
	}
	iter.Release()
	iter = arbDb.NewIterator(missingBlockMetadataInputFeedPrefix, nil)
	pos = trackBlockMetadataFrom
	i := 0
	for iter.Next() {
		// Blocks with blockMetadata present shouldn't have the missingBlockMetadataInputFeedPrefix keys present in arbDB
		for i < len(blocksWithBlockMetadata) && blocksWithBlockMetadata[i] == pos {
			i++
			pos++
		}
		keyBytes := bytes.TrimPrefix(iter.Key(), missingBlockMetadataInputFeedPrefix)
		if binary.BigEndian.Uint64(keyBytes) != pos {
			t.Fatalf("unexpected msgSeqNum with missingBlockMetadataInputFeedPrefix for blockMetadata. Want: %d, Got: %d", pos, binary.BigEndian.Uint64(keyBytes))
		}
		pos++
	}
	if pos-1 != latestL2 {
		t.Fatalf("number of keys with missingBlockMetadataInputFeedPrefix doesn't match expected value. Want: %d, Got: %d", latestL2, pos-1)
	}
	iter.Release()

	// Rebuild blockMetadata and cleanup trackers from ArbDB
	rebuildStartPos := uint64(5)
	blockMetadataFetcher, err := arbnode.NewBlockMetadataFetcher(ctx, arbnode.BlockMetadataFetcherConfig{Source: rpcclient.ClientConfig{URL: builder.L2.Stack.HTTPEndpoint()}}, arbDb, newNode.ExecNode, rebuildStartPos)
	Require(t, err)
	blockMetadataFetcher.Update(ctx)

	// Check if all blockMetadata starting from rebuildStartPos was synced from bulk BlockMetadata API via the blockMetadataFetcher and that trackers for missing blockMetadata were cleared
	// Note that trackers for missing blockMetadata below rebuildStartPos won't be cleared and that is expected since we give user choice to only sync from a certain target instead of syncing
	// all the missing blockMetadata. Currently this target is set by node to the same value as TrackBlockMetadataFrom flag
	iter = arbDb.NewIterator(blockMetadataInputFeedPrefix, nil)
	pos = rebuildStartPos
	for iter.Next() {
		keyBytes := bytes.TrimPrefix(iter.Key(), blockMetadataInputFeedPrefix)
		if binary.BigEndian.Uint64(keyBytes) != pos {
			t.Fatalf("unexpected msgSeqNum with blockMetadataInputFeedPrefix for blockMetadata. Want: %d, Got: %d", pos, binary.BigEndian.Uint64(keyBytes))
		}
		if !bytes.Equal(sampleBulkData[pos-1], iter.Value()) {
			t.Fatalf("blockMetadata mismatch for blockNumber: %d. Want: %v, Got: %v", pos, sampleBulkData[pos-1], iter.Value())
		}
		pos++
	}
	if pos-1 != latestL2 {
		t.Fatalf("number of keys with blockMetadataInputFeedPrefix doesn't match expected value. Want: %d, Got: %d", latestL2, pos-1)
	}
	iter.Release()
	iter = arbDb.NewIterator(missingBlockMetadataInputFeedPrefix, nil)
	pos = trackBlockMetadataFrom
	for iter.Next() {
		keyBytes := bytes.TrimPrefix(iter.Key(), missingBlockMetadataInputFeedPrefix)
		if binary.BigEndian.Uint64(keyBytes) != pos {
			t.Fatalf("unexpected msgSeqNum with missingBlockMetadataInputFeedPrefix for blockMetadata. Want: %d, Got: %d", pos, binary.BigEndian.Uint64(keyBytes))
		}
		pos++
	}
	if pos != rebuildStartPos {
		t.Fatalf("number of keys with missingBlockMetadataInputFeedPrefix doesn't match expected value. Want: %d, Got: %d", rebuildStartPos-trackBlockMetadataFrom, pos-trackBlockMetadataFrom)
	}
	iter.Release()
}

func TestTimeboostedFieldInReceiptsObject(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.nodeConfig.TransactionStreamer.TrackBlockMetadataFrom = 1
	builder.execConfig.BlockMetadataApiCacheSize = 0 // Caching is disabled
	cleanup := builder.Build(t)
	defer cleanup()

	// Generate blocks until current block is totalBlocks
	arbDb := builder.L2.ConsensusNode.ArbDB
	blockNum := big.NewInt(2)
	builder.L2Info.GenerateAccount("User")
	user := builder.L2Info.GetDefaultTransactOpts("User", ctx)
	var latestL2 uint64
	var err error
	for i := 0; ; i++ {
		builder.L2.TransferBalanceTo(t, "Owner", util.RemapL1Address(user.From), big.NewInt(1e18), builder.L2Info)
		latestL2, err = builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
		if latestL2 >= blockNum.Uint64() {
			break
		}
	}

	for i := uint64(1); i < latestL2; i++ {
		// Clean BlockMetadata from arbDB so that we can modify it at will
		Require(t, arbDb.Delete(dbKey([]byte("t"), i)))
	}

	block, err := builder.L2.Client.BlockByNumber(ctx, blockNum)
	Require(t, err)
	if len(block.Transactions()) != 2 {
		t.Fatalf("expecting two txs in the second block, but found: %d txs", len(block.Transactions()))
	}

	// Set first tx (internal tx anyway) to not timeboosted and Second one to timeboosted- BlockMetadata (in bits)-> 00000000 00000010
	Require(t, arbDb.Put(dbKey([]byte("t"), blockNum.Uint64()), []byte{0, 2}))
	l2rpc := builder.L2.Stack.Attach()
	// Extra timeboosted field in pointer form to check for its existence
	type timeboostedFromReceipt struct {
		Timeboosted *bool `json:"timeboosted"`
	}
	var receiptResult []timeboostedFromReceipt
	err = l2rpc.CallContext(ctx, &receiptResult, "eth_getBlockReceipts", rpc.BlockNumber(blockNum.Int64()))
	Require(t, err)
	if receiptResult[0].Timeboosted == nil || receiptResult[1].Timeboosted == nil {
		t.Fatal("timeboosted field should exist in the receipt object of both- first and second txs")
	}
	if *receiptResult[0].Timeboosted != false {
		t.Fatal("first tx was not timeboosted, but the field indicates otherwise")
	}
	if *receiptResult[1].Timeboosted != true {
		t.Fatal("second tx was timeboosted, but the field indicates otherwise")
	}

	// Check that timeboosted is accurate for eth_getTransactionReceipt as well
	var txReceipt timeboostedFromReceipt
	err = l2rpc.CallContext(ctx, &txReceipt, "eth_getTransactionReceipt", block.Transactions()[0].Hash())
	Require(t, err)
	if txReceipt.Timeboosted == nil {
		t.Fatal("timeboosted field should exist in the receipt object of first tx")
	}
	if *txReceipt.Timeboosted != false {
		t.Fatal("first tx was not timeboosted, but the field indicates otherwise")
	}
	err = l2rpc.CallContext(ctx, &txReceipt, "eth_getTransactionReceipt", block.Transactions()[1].Hash())
	Require(t, err)
	if txReceipt.Timeboosted == nil {
		t.Fatal("timeboosted field should exist in the receipt object of second tx")
	}
	if *txReceipt.Timeboosted != true {
		t.Fatal("second tx was timeboosted, but the field indicates otherwise")
	}

	// Check that timeboosted field shouldn't exist for any txs of block=1, as this block doesn't have blockMetadata
	block, err = builder.L2.Client.BlockByNumber(ctx, common.Big1)
	Require(t, err)
	if len(block.Transactions()) != 2 {
		t.Fatalf("expecting two txs in the first block, but found: %d txs", len(block.Transactions()))
	}
	var receiptResult2 []timeboostedFromReceipt
	err = l2rpc.CallContext(ctx, &receiptResult2, "eth_getBlockReceipts", rpc.BlockNumber(1))
	Require(t, err)
	if receiptResult2[0].Timeboosted != nil || receiptResult2[1].Timeboosted != nil {
		t.Fatal("timeboosted field shouldn't exist in the receipt object of all the txs")
	}
	var txReceipt2 timeboostedFromReceipt
	err = l2rpc.CallContext(ctx, &txReceipt2, "eth_getTransactionReceipt", block.Transactions()[0].Hash())
	Require(t, err)
	if txReceipt2.Timeboosted != nil {
		t.Fatal("timeboosted field shouldn't exist in the receipt object of all the txs")
	}
	var txReceipt3 timeboostedFromReceipt
	err = l2rpc.CallContext(ctx, &txReceipt3, "eth_getTransactionReceipt", block.Transactions()[1].Hash())
	Require(t, err)
	if txReceipt3.Timeboosted != nil {
		t.Fatal("timeboosted field shouldn't exist in the receipt object of all the txs")
	}

	// Print the receipt object for reference
	var receiptResultRaw json.RawMessage
	err = l2rpc.CallContext(ctx, &receiptResultRaw, "eth_getBlockReceipts", rpc.BlockNumber(blockNum.Int64()))
	Require(t, err)
	colors.PrintGrey("receipt object- ", string(receiptResultRaw))

	builder.L2.TransferBalanceTo(t, "Owner", util.RemapL1Address(user.From), big.NewInt(1e18), builder.L2Info)
	latestL2, err = builder.L2.Client.BlockNumber(ctx)
	Require(t, err)
	var receiptWithoutTimeboostEnabled []timeboostedFromReceipt
	// #nosec G115
	err = l2rpc.CallContext(ctx, &receiptWithoutTimeboostEnabled, "eth_getBlockReceipts", rpc.BlockNumber(latestL2))
	Require(t, err)
	if len(receiptWithoutTimeboostEnabled) != 2 {
		t.Fatalf("expecting two tx receipts got: %d", len(receiptWithoutTimeboostEnabled))
	}
	if receiptWithoutTimeboostEnabled[0].Timeboosted == nil || *receiptWithoutTimeboostEnabled[0].Timeboosted {
		t.Fatal("timeboosted field should exist in the receipt object of all the txs and it should be false")
	}
	if receiptWithoutTimeboostEnabled[1].Timeboosted == nil || *receiptWithoutTimeboostEnabled[1].Timeboosted {
		t.Fatal("timeboosted field should exist in the receipt object of all the txs and it should be false")
	}
}

func TestTimeboostBulkBlockMetadataAPI(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.nodeConfig.TransactionStreamer.TrackBlockMetadataFrom = 1
	builder.execConfig.BlockMetadataApiCacheSize = 0 // Caching is disabled
	cleanup := builder.Build(t)
	defer cleanup()

	arbDb := builder.L2.ConsensusNode.ArbDB

	// Generate blocks until current block is end
	start := 1
	end := 20
	builder.L2Info.GenerateAccount("User")
	user := builder.L2Info.GetDefaultTransactOpts("User", ctx)
	for i := 0; ; i++ {
		builder.L2.TransferBalanceTo(t, "Owner", util.RemapL1Address(user.From), big.NewInt(1e18), builder.L2Info)
		latestL2, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
		// Clean BlockMetadata from arbDB so that we can modify it at will
		Require(t, arbDb.Delete(dbKey([]byte("t"), latestL2)))
		// #nosec G115
		if latestL2 > uint64(end)+10 {
			break
		}
	}
	var sampleBulkData []gethexec.NumberAndBlockMetadata
	for i := start; i <= end; i += 2 {
		sampleData := gethexec.NumberAndBlockMetadata{
			// #nosec G115
			BlockNumber: uint64(i),
			// #nosec G115
			RawMetadata: []byte{0, uint8(i)},
		}
		sampleBulkData = append(sampleBulkData, sampleData)
		Require(t, arbDb.Put(dbKey([]byte("t"), sampleData.BlockNumber), sampleData.RawMetadata))
	}

	l2rpc := builder.L2.Stack.Attach()
	var result []gethexec.NumberAndBlockMetadata
	err := l2rpc.CallContext(ctx, &result, "arb_getRawBlockMetadata", rpc.BlockNumber(start), "latest") // Test rpc.BlockNumber feature, send "latest" as an arg instead of blockNumber
	Require(t, err)

	if len(result) != len(sampleBulkData) {
		t.Fatalf("number of entries in arb_getRawBlockMetadata is incorrect. Got: %d, Want: %d", len(result), len(sampleBulkData))
	}
	for i, data := range result {
		if data.BlockNumber != sampleBulkData[i].BlockNumber {
			t.Fatalf("BlockNumber mismatch. Got: %d, Want: %d", data.BlockNumber, sampleBulkData[i].BlockNumber)
		}
		if !bytes.Equal(data.RawMetadata, sampleBulkData[i].RawMetadata) {
			t.Fatalf("RawMetadata. Got: %s, Want: %s", data.RawMetadata, sampleBulkData[i].RawMetadata)
		}
	}

	// Test that without cache the result returned is always in sync with ArbDB
	sampleBulkData[0].RawMetadata = []byte{1, 11}
	Require(t, arbDb.Put(dbKey([]byte("t"), 1), sampleBulkData[0].RawMetadata))

	err = l2rpc.CallContext(ctx, &result, "arb_getRawBlockMetadata", rpc.BlockNumber(1), rpc.BlockNumber(1))
	Require(t, err)
	if len(result) != 1 {
		t.Fatal("result returned with more than one entry")
	}
	if !bytes.Equal(sampleBulkData[0].RawMetadata, result[0].RawMetadata) {
		t.Fatal("BlockMetadata gotten from API doesn't match the latest entry in ArbDB")
	}

	// Test that LRU caching works
	builder.execConfig.BlockMetadataApiCacheSize = 1000
	builder.execConfig.BlockMetadataApiBlocksLimit = 25
	builder.RestartL2Node(t)
	l2rpc = builder.L2.Stack.Attach()
	err = l2rpc.CallContext(ctx, &result, "arb_getRawBlockMetadata", rpc.BlockNumber(start), rpc.BlockNumber(end))
	Require(t, err)

	arbDb = builder.L2.ConsensusNode.ArbDB
	updatedBlockMetadata := []byte{2, 12}
	Require(t, arbDb.Put(dbKey([]byte("t"), 1), updatedBlockMetadata))

	err = l2rpc.CallContext(ctx, &result, "arb_getRawBlockMetadata", rpc.BlockNumber(1), rpc.BlockNumber(1))
	Require(t, err)
	if len(result) != 1 {
		t.Fatal("result returned with more than one entry")
	}
	if bytes.Equal(updatedBlockMetadata, result[0].RawMetadata) {
		t.Fatal("BlockMetadata should've been fetched from cache and not the db")
	}
	if !bytes.Equal(sampleBulkData[0].RawMetadata, result[0].RawMetadata) {
		t.Fatal("incorrect caching of BlockMetadata")
	}

	// Test that ErrBlockMetadataApiBlocksLimitExceeded is thrown when query range exceeds the limit
	err = l2rpc.CallContext(ctx, &result, "arb_getRawBlockMetadata", rpc.BlockNumber(start), rpc.BlockNumber(26))
	if !strings.Contains(err.Error(), gethexec.ErrBlockMetadataApiBlocksLimitExceeded.Error()) {
		t.Fatalf("expecting ErrBlockMetadataApiBlocksLimitExceeded error, got: %v", err)
	}

	// A Reorg event should clear the cache, hence the data fetched now should be accurate
	Require(t, builder.L2.ConsensusNode.TxStreamer.ReorgTo(10))
	err = l2rpc.CallContext(ctx, &result, "arb_getRawBlockMetadata", rpc.BlockNumber(start), rpc.BlockNumber(end))
	Require(t, err)
	if !bytes.Equal(updatedBlockMetadata, result[0].RawMetadata) {
		t.Fatal("BlockMetadata should've been fetched from db and not the cache")
	}
}

// func TestExpressLaneControlTransfer(t *testing.T) {
// 	t.Parallel()
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	tmpDir, err := os.MkdirTemp("", "*")
// 	require.NoError(t, err)
// 	t.Cleanup(func() {
// 		require.NoError(t, os.RemoveAll(tmpDir))
// 	})
// 	jwtSecretPath := filepath.Join(tmpDir, "sequencer.jwt")

// 	auctionContractAddr, aliceBidderClient, bobBidderClient, roundDuration, builderSeq, cleanupSeq, _, _ := setupExpressLaneAuction(t, tmpDir, ctx, jwtSecretPath, 0)
// 	seq, seqClient, seqInfo := builderSeq.L2.ConsensusNode, builderSeq.L2.Client, builderSeq.L2Info
// 	defer cleanupSeq()

// 	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, seqClient)
// 	Require(t, err)
// 	rawRoundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
// 	Require(t, err)
// 	roundTimingInfo, err := timeboost.NewRoundTimingInfo(rawRoundTimingInfo)
// 	Require(t, err)

// 	// Prepare clients that can submit txs to the sequencer via the express lane.
// 	chainId, err := seqClient.ChainID(ctx)
// 	Require(t, err)
// 	seqDial, err := rpc.Dial(seq.Stack.HTTPEndpoint())
// 	Require(t, err)
// 	createExpressLaneClientFor := func(name string) (*expressLaneClient, bind.TransactOpts) {
// 		priv := seqInfo.Accounts[name].PrivateKey
// 		expressLaneClient := newExpressLaneClient(
// 			priv,
// 			chainId,
// 			*roundTimingInfo,
// 			auctionContractAddr,
// 			seqDial,
// 		)
// 		expressLaneClient.Start(ctx)
// 		transacOpts := seqInfo.GetDefaultTransactOpts(name, ctx)
// 		transacOpts.NoSend = true
// 		return expressLaneClient, transacOpts
// 	}
// 	bobExpressLaneClient, bobOpts := createExpressLaneClientFor("Bob")
// 	aliceExpressLaneClient, aliceOpts := createExpressLaneClientFor("Alice")

// 	// Bob will win the auction and become controller for next round
// 	placeBidsAndDecideWinner(t, ctx, seqClient, seqInfo, auctionContract, "Bob", "Alice", bobBidderClient, aliceBidderClient, roundDuration)
// 	time.Sleep(roundTimingInfo.TimeTilNextRound())

// 	// Check that Bob's tx gets priority since he's the controller
// 	verifyControllerAdvantage(t, ctx, seqClient, bobExpressLaneClient, seqInfo, "Bob", "Alice")

// 	// Transfer express lane control from Bob to Alice
// 	currRound := roundTimingInfo.RoundNumber()
// 	duringRoundTransferTx, err := auctionContract.ExpressLaneAuctionTransactor.TransferExpressLaneController(&bobOpts, currRound, seqInfo.Accounts["Alice"].Address)
// 	Require(t, err)
// 	err = bobExpressLaneClient.SendTransaction(ctx, duringRoundTransferTx)
// 	Require(t, err)

// 	time.Sleep(time.Second) // Wait for controller to change on the sequencer side
// 	// Check that now Alice's tx gets priority since she's the controller after bob transfered it
// 	verifyControllerAdvantage(t, ctx, seqClient, aliceExpressLaneClient, seqInfo, "Alice", "Bob")

// 	// Alice and Bob submit bids and Alice wins for the next round
// 	placeBidsAndDecideWinner(t, ctx, seqClient, seqInfo, auctionContract, "Alice", "Bob", aliceBidderClient, bobBidderClient, roundDuration)
// 	t.Log("Alice won the express lane auction for upcoming round, now try to transfer control before the next round begins...")

// 	// Alice now transfers control to bob before her round begins
// 	winnerRound := currRound + 1
// 	currRound = roundTimingInfo.RoundNumber()
// 	if currRound >= winnerRound {
// 		t.Fatalf("next round already began, try running the test again. Current round: %d, Winner Round: %d", currRound, winnerRound)
// 	}

// 	beforeRoundTransferTx, err := auctionContract.ExpressLaneAuctionTransactor.TransferExpressLaneController(&aliceOpts, winnerRound, seqInfo.Accounts["Bob"].Address)
// 	Require(t, err)
// 	err = aliceExpressLaneClient.SendTransaction(ctx, beforeRoundTransferTx)
// 	Require(t, err)

// 	setExpressLaneIterator, err := auctionContract.FilterSetExpressLaneController(&bind.FilterOpts{Context: ctx}, nil, nil, nil)
// 	Require(t, err)
// 	verifyControllerChange := func(round uint64, prev, new common.Address) {
// 		setExpressLaneIterator.Next()
// 		if setExpressLaneIterator.Event.Round != round {
// 			t.Fatalf("unexpected round number. Want: %d, Got: %d", round, setExpressLaneIterator.Event.Round)
// 		}
// 		if setExpressLaneIterator.Event.PreviousExpressLaneController != prev {
// 			t.Fatalf("unexpected previous express lane controller. Want: %v, Got: %v", prev, setExpressLaneIterator.Event.PreviousExpressLaneController)
// 		}
// 		if setExpressLaneIterator.Event.NewExpressLaneController != new {
// 			t.Fatalf("unexpected new express lane controller. Want: %v, Got: %v", new, setExpressLaneIterator.Event.NewExpressLaneController)
// 		}
// 	}
// 	// Verify during round control change
// 	verifyControllerChange(currRound, common.Address{}, bobOpts.From) // Bob wins auction
// 	verifyControllerChange(currRound, bobOpts.From, aliceOpts.From)   // Bob transfers control to Alice
// 	// Verify before round control change
// 	verifyControllerChange(winnerRound, common.Address{}, aliceOpts.From) // Alice wins auction
// 	verifyControllerChange(winnerRound, aliceOpts.From, bobOpts.From)     // Alice transfers control to Bob before the round begins
// }

func TestTimeboostSequencerFeed_ExpressLaneAuction_ExpressLaneTxsHaveAdvantage(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "*")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	})

	auctionContractAddr, aliceBidderClient, bobBidderClient, roundDuration, builderSeq, cleanupSeq, _, _ := setupExpressLaneAuction(t, tmpDir, ctx, 0, 0)
	seq, seqClient, seqInfo := builderSeq.L2.ConsensusNode, builderSeq.L2.Client, builderSeq.L2Info
	defer cleanupSeq()

	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, seqClient)
	Require(t, err)
	rawRoundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	Require(t, err)
	roundTimingInfo, err := timeboost.NewRoundTimingInfo(rawRoundTimingInfo)
	Require(t, err)

	placeBidsAndDecideWinner(t, ctx, seqClient, seqInfo, auctionContract, "Bob", "Alice", bobBidderClient, aliceBidderClient, roundDuration)
	time.Sleep(roundTimingInfo.TimeTilNextRound())

	chainId, err := seqClient.ChainID(ctx)
	Require(t, err)

	// Prepare a client that can submit txs to the sequencer via the express lane.
	bobPriv := seqInfo.Accounts["Bob"].PrivateKey
	seqDial, err := rpc.Dial(seq.Stack.HTTPEndpoint())
	Require(t, err)
	expressLaneClient := newExpressLaneClient(
		bobPriv,
		chainId,
		*roundTimingInfo,
		auctionContractAddr,
		seqDial,
	)
	expressLaneClient.Start(ctx)

	verifyControllerAdvantage(t, ctx, seqClient, expressLaneClient, seqInfo, "Bob", "Alice")
}

func TestTimeboostSequencerFeed_ExpressLaneAuction_InnerPayloadNoncesAreRespected_TimeboostedFieldIsCorrect(t *testing.T) {
	t.Parallel()

	logHandler := testhelpers.InitTestLog(t, log.LevelInfo)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "*")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	})
	auctionContractAddr, aliceBidderClient, bobBidderClient, roundDuration, builderSeq, cleanupSeq, feedListener, cleanupFeedListener := setupExpressLaneAuction(t, tmpDir, ctx, withFeedListener, 0)
	seq, seqClient, seqInfo := builderSeq.L2.ConsensusNode, builderSeq.L2.Client, builderSeq.L2Info
	defer cleanupSeq()
	defer cleanupFeedListener()

	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, seqClient)
	Require(t, err)
	rawRoundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	Require(t, err)
	roundTimingInfo, err := timeboost.NewRoundTimingInfo(rawRoundTimingInfo)
	Require(t, err)

	Require(t, err)

	placeBidsAndDecideWinner(t, ctx, seqClient, seqInfo, auctionContract, "Bob", "Alice", bobBidderClient, aliceBidderClient, roundDuration)
	time.Sleep(roundTimingInfo.TimeTilNextRound())

	// Prepare a client that can submit txs to the sequencer via the express lane.
	bobPriv := seqInfo.Accounts["Bob"].PrivateKey
	chainId, err := seqClient.ChainID(ctx)
	Require(t, err)
	seqDial, err := rpc.Dial(seq.Stack.HTTPEndpoint())
	Require(t, err)
	expressLaneClient := newExpressLaneClient(
		bobPriv,
		chainId,
		*roundTimingInfo,
		auctionContractAddr,
		seqDial,
	)
	expressLaneClient.Start(ctx)

	// We first generate an account for Charlie and transfer some balance to him.
	seqInfo.GenerateAccount("Charlie")
	TransferBalance(t, "Owner", "Charlie", arbmath.BigMulByUint(oneEth, 500), seqInfo, seqClient, ctx)

	// During the express lane, Bob sends txs that do not belong to him, but he is the express lane controller so they
	// will go through the express lane.
	// These tx payloads are sent with nonces out of order, and those with nonces too high should fail.
	var wg sync.WaitGroup
	wg.Add(2)
	ownerAddr := seqInfo.GetAddress("Owner")
	aliceNonce, err := seqClient.PendingNonceAt(ctx, seqInfo.GetAddress("Alice"))
	Require(t, err)
	aliceData := &types.DynamicFeeTx{
		To:        &ownerAddr,
		Gas:       seqInfo.TransferGas,
		GasFeeCap: new(big.Int).Set(seqInfo.GasPrice),
		Value:     big.NewInt(1e12),
		Nonce:     aliceNonce,
		Data:      nil,
	}
	aliceTx := seqInfo.SignTxAs("Alice", aliceData)
	go func(w *sync.WaitGroup) {
		defer w.Done()
		err = seqClient.SendTransaction(ctx, aliceTx)
		Require(t, err)
	}(&wg)

	txData := &types.DynamicFeeTx{
		To:        &ownerAddr,
		Gas:       seqInfo.TransferGas,
		Value:     big.NewInt(1e12),
		Nonce:     2,
		GasFeeCap: aliceTx.GasFeeCap(),
		Data:      nil,
	}
	charlie2 := seqInfo.SignTxAs("Charlie", txData)
	txData = &types.DynamicFeeTx{
		To:        &ownerAddr,
		Gas:       seqInfo.TransferGas,
		Value:     big.NewInt(1e12),
		Nonce:     0,
		GasFeeCap: aliceTx.GasFeeCap(),
		Data:      nil,
	}
	charlie0 := seqInfo.SignTxAs("Charlie", txData)

	// Send the express lane txs with nonces out of order, 0 and 2 so that nonce reordering logic in sequencer doesn't resequence them correctly
	var err2 error
	go func(w *sync.WaitGroup) {
		defer w.Done()
		time.Sleep(time.Millisecond * 10)
		err2 = expressLaneClient.SendTransactionWithSequence(ctx, charlie2, 0)
	}(&wg)
	time.Sleep(time.Millisecond * 50)
	err = expressLaneClient.SendTransactionWithSequence(ctx, charlie0, 1)
	Require(t, err)
	wg.Wait()
	if err2 == nil {
		t.Fatal("Charlie should not be able to send tx with nonce 2")
	}
	if !strings.Contains(err2.Error(), "context deadline exceeded") {
		t.Fatal("Charlie's first tx should've consumed a sequence number and rejected thus not appear in a block leading to context deadline exceeded from EnsureTxSucceeded")
	}
	// After round is done, verify that Charlie beats Alice in the final sequence, and that the emitted txs
	// for Charlie are correct.
	aliceReceipt, err := seqClient.TransactionReceipt(ctx, aliceTx.Hash())
	Require(t, err)
	aliceBlock := aliceReceipt.BlockNumber.Uint64()
	charlieReceipt, err := seqClient.TransactionReceipt(ctx, charlie0.Hash())
	Require(t, err)
	charlieBlock := charlieReceipt.BlockNumber.Uint64()

	if aliceBlock < charlieBlock {
		t.Fatal("Alice's tx should not have been sequenced before Charlie's in different blocks")
	} else if aliceBlock == charlieBlock {
		if aliceReceipt.TransactionIndex < charlieReceipt.TransactionIndex {
			t.Fatal("Charlie should have been sequenced before Alice with express lane")
		}
	}

	// First test that timeboosted byte array is correct on sequencer side
	verifyTimeboostedCorrectness(t, ctx, "Alice", seq, seqClient, false, aliceTx, aliceBlock)
	verifyTimeboostedCorrectness(t, ctx, "Charlie", seq, seqClient, true, charlie0, charlieBlock)

	// Verify that timeboosted byte array receieved via sequencer feed is correct
	_, err = WaitForTx(ctx, feedListener.Client, charlie0.Hash(), time.Second*5)
	Require(t, err)
	_, err = WaitForTx(ctx, feedListener.Client, aliceTx.Hash(), time.Second*5)
	Require(t, err)
	verifyTimeboostedCorrectness(t, ctx, "Alice", feedListener.ConsensusNode, feedListener.Client, false, aliceTx, aliceBlock)
	verifyTimeboostedCorrectness(t, ctx, "Charlie", feedListener.ConsensusNode, feedListener.Client, true, charlie0, charlieBlock)

	if logHandler.WasLogged(arbnode.BlockHashMismatchLogMsg) {
		t.Fatal("BlockHashMismatchLogMsg was logged unexpectedly")
	}
}

// verifyTimeboostedCorrectness is used to check if the timeboosted byte array in both the sequencer's tx streamer and the client node's tx streamer (which is connected
// to the sequencer feed) is accurate, i.e it represents correctly whether a tx is timeboosted or not
func verifyTimeboostedCorrectness(t *testing.T, ctx context.Context, user string, tNode *arbnode.Node, tClient *ethclient.Client, isTimeboosted bool, userTx *types.Transaction, userTxBlockNum uint64) {
	blockMetadataOfBlock, err := tNode.TxStreamer.BlockMetadataAtCount(arbutil.MessageIndex(userTxBlockNum) + 1)
	Require(t, err)
	if len(blockMetadataOfBlock) == 0 {
		t.Fatal("got empty blockMetadata byte array")
	}
	if blockMetadataOfBlock[0] != message.TimeboostedVersion {
		t.Fatalf("blockMetadata byte array has invalid version. Want: %d, Got: %d", message.TimeboostedVersion, blockMetadataOfBlock[0])
	}
	userTxBlock, err := tClient.BlockByNumber(ctx, new(big.Int).SetUint64(userTxBlockNum))
	Require(t, err)
	var foundUserTx bool
	for txIndex, tx := range userTxBlock.Transactions() {
		got, err := blockMetadataOfBlock.IsTxTimeboosted(txIndex)
		Require(t, err)
		if tx.Hash() == userTx.Hash() {
			foundUserTx = true
			if !isTimeboosted && got {
				t.Fatalf("incorrect timeboosted bit for %s's tx, it shouldn't be timeboosted", user)
			} else if isTimeboosted && !got {
				t.Fatalf("incorrect timeboosted bit for %s's tx, it should be timeboosted", user)
			}
		} else if got {
			// Other tx's right now shouln't be timeboosted
			t.Fatalf("incorrect timeboosted bit for nonspecified tx with index: %d, it shouldn't be timeboosted", txIndex)
		}
	}
	if !foundUserTx {
		t.Fatalf("%s's tx wasn't found in the block with blockNum retrieved from its receipt", user)
	}
}

func placeBidsAndDecideWinner(t *testing.T, ctx context.Context, seqClient *ethclient.Client, seqInfo *BlockchainTestInfo, auctionContract *express_lane_auctiongen.ExpressLaneAuction, winner, loser string, winnerBidderClient, loserBidderClient *timeboost.BidderClient, roundDuration time.Duration) {
	t.Helper()

	rawRoundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	Require(t, err)
	roundTimingInfo, err := timeboost.NewRoundTimingInfo(rawRoundTimingInfo)
	Require(t, err)
	currRound := roundTimingInfo.RoundNumber()

	// We are now in the bidding round, both issue their bids. winner will win
	t.Logf("%s and %s now submitting their bids at %v", winner, loser, time.Now())
	winnerBid, err := winnerBidderClient.Bid(ctx, big.NewInt(2), seqInfo.GetAddress(winner))
	Require(t, err)
	loserBid, err := loserBidderClient.Bid(ctx, big.NewInt(1), seqInfo.GetAddress(loser))
	Require(t, err)
	t.Logf("%s bid %+v", winner, winnerBid)
	t.Logf("%s bid %+v", loser, loserBid)

	// Subscribe to auction resolutions and wait for a winner
	winnerAddr, winnerRound := awaitAuctionResolved(t, ctx, seqClient, auctionContract)

	// Verify winner wins the auction
	if winnerAddr != seqInfo.GetAddress(winner) {
		t.Fatalf("%s should have won the express lane auction", winner)
	}
	t.Logf("%s won the auction for the round: %d", winner, winnerRound)
	if winnerRound != currRound+1 {
		t.Fatalf("unexpected winner round: Want:%d Got:%d", currRound+1, winnerRound)
	}

	it, err := auctionContract.FilterAuctionResolved(&bind.FilterOpts{Context: ctx}, nil, nil, nil)
	Require(t, err)
	winnerWon := false
	for it.Next() {
		if it.Event.FirstPriceBidder == seqInfo.GetAddress(winner) && it.Event.Round == winnerRound {
			winnerWon = true
		}
	}
	if !winnerWon {
		t.Fatalf("%s should have won the auction", winner)
	}
}

func verifyControllerAdvantage(t *testing.T, ctx context.Context, seqClient *ethclient.Client, controllerClient *expressLaneClient, seqInfo *BlockchainTestInfo, controller, otherUser string) {
	t.Helper()

	// During the express lane around, controller sends txs always 150ms later than otherUser, but otherUser's
	// txs end up getting delayed by 200ms as they are not the express lane controller.
	// In the end, controller's txs should be ordered before otherUser's during the round.
	var wg sync.WaitGroup
	wg.Add(2)
	ownerAddr := seqInfo.GetAddress("Owner")

	otherUserNonce, err := seqClient.PendingNonceAt(ctx, seqInfo.GetAddress(otherUser))
	Require(t, err)
	otherUserData := &types.DynamicFeeTx{
		To:        &ownerAddr,
		Gas:       seqInfo.TransferGas,
		GasFeeCap: new(big.Int).Set(seqInfo.GasPrice),
		Value:     big.NewInt(1e12),
		Nonce:     otherUserNonce,
		Data:      nil,
	}
	otherUserTx := seqInfo.SignTxAs(otherUser, otherUserData)
	go func(w *sync.WaitGroup) {
		defer w.Done()
		Require(t, seqClient.SendTransaction(ctx, otherUserTx))
	}(&wg)

	controllerNonce, err := seqClient.PendingNonceAt(ctx, seqInfo.GetAddress(controller))
	Require(t, err)
	controllerData := &types.DynamicFeeTx{
		To:        &ownerAddr,
		Gas:       seqInfo.TransferGas,
		GasFeeCap: new(big.Int).Set(seqInfo.GasPrice),
		Value:     big.NewInt(1e12),
		Nonce:     controllerNonce,
		Data:      nil,
	}
	controllerBoostableTx := seqInfo.SignTxAs(controller, controllerData)
	go func(w *sync.WaitGroup) {
		defer w.Done()
		time.Sleep(time.Millisecond * 10)
		Require(t, controllerClient.SendTransaction(ctx, controllerBoostableTx))
	}(&wg)
	wg.Wait()

	// After round is done, verify that controller beats otherUser in the final sequence.
	otherUserTxReceipt, err := seqClient.TransactionReceipt(ctx, otherUserTx.Hash())
	Require(t, err)
	otherUserBlock := otherUserTxReceipt.BlockNumber.Uint64()
	controllerBoostableTxReceipt, err := seqClient.TransactionReceipt(ctx, controllerBoostableTx.Hash())
	Require(t, err)
	controllerBlock := controllerBoostableTxReceipt.BlockNumber.Uint64()

	if otherUserBlock < controllerBlock {
		t.Fatalf("%s's tx should not have been sequenced before %s's in different blocks", otherUser, controller)
	} else if otherUserBlock == controllerBlock {
		if otherUserTxReceipt.TransactionIndex < controllerBoostableTxReceipt.TransactionIndex {
			t.Fatalf("%s should have been sequenced before %s with express lane", controller, otherUser)
		}
	}
}

type extraNodeType int

const (
	withForwardingSeq extraNodeType = iota + 1
	withFeedListener
)

func setupExpressLaneAuction(
	t *testing.T,
	dbDirPath string,
	ctx context.Context,
	extraNodeTy extraNodeType,
	queueTimeoutInBlocks uint64,
) (common.Address, *timeboost.BidderClient, *timeboost.BidderClient, time.Duration, *NodeBuilder, func(), *TestClient, func()) {
	seqPort := getRandomPort(t)
	forwarderPort := getRandomPort(t)

	nodeNames := []string{fmt.Sprintf("http://127.0.0.1:%d", seqPort), fmt.Sprintf("http://127.0.0.1:%d", forwarderPort)}
	expressLaneRedisURL := redisutil.CreateTestRedis(ctx, t)
	initRedisForTest(t, ctx, expressLaneRedisURL, nodeNames)

	builderSeq := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builderSeq.l2StackConfig.HTTPHost = "localhost"
	builderSeq.l2StackConfig.HTTPPort = seqPort
	builderSeq.l2StackConfig.HTTPModules = []string{"eth", "arb", "debug", "timeboost", "auctioneer"}
	builderSeq.nodeConfig.Feed.Output = *newBroadcasterConfigTest()
	builderSeq.nodeConfig.Dangerous.NoSequencerCoordinator = false
	builderSeq.nodeConfig.SeqCoordinator.Enable = true
	builderSeq.nodeConfig.SeqCoordinator.RedisUrl = expressLaneRedisURL
	builderSeq.nodeConfig.SeqCoordinator.MyUrl = nodeNames[0]
	builderSeq.nodeConfig.SeqCoordinator.DeleteFinalizedMsgs = false
	builderSeq.execConfig.Sequencer.Enable = true
	builderSeq.execConfig.Sequencer.Timeboost = gethexec.TimeboostConfig{
		Enable:                       false, // We need to start without timeboost initially to create the auction contract
		ExpressLaneAdvantage:         time.Second * 5,
		RedisUrl:                     expressLaneRedisURL,
		MaxFutureSequenceDistance:    1500, // Required for TestExpressLaneTransactionHandlingComplex
		RedisUpdateEventsChannelSize: 50,
		QueueTimeoutInBlocks:         queueTimeoutInBlocks,
	}
	builderSeq.nodeConfig.TransactionStreamer.TrackBlockMetadataFrom = 1
	cleanupSeq := builderSeq.Build(t)
	seqInfo, seqNode, seqClient := builderSeq.L2Info, builderSeq.L2.ConsensusNode, builderSeq.L2.Client

	var extraNode *TestClient
	var cleanupExtraNode func()
	switch extraNodeTy {
	case withForwardingSeq:
		forwarderNodeCfg := arbnode.ConfigDefaultL1Test()
		forwarderNodeCfg.BatchPoster.Enable = false
		forwarderNodeCfg.Dangerous.NoSequencerCoordinator = false
		forwarderNodeCfg.SeqCoordinator.Enable = true
		forwarderNodeCfg.SeqCoordinator.RedisUrl = expressLaneRedisURL
		forwarderNodeCfg.SeqCoordinator.MyUrl = nodeNames[1]
		forwarderNodeCfg.SeqCoordinator.DeleteFinalizedMsgs = false
		builderSeq.l2StackConfig.HTTPPort = forwarderPort
		extraNode, cleanupExtraNode = builderSeq.Build2ndNode(t, &SecondNodeParams{nodeConfig: forwarderNodeCfg})
	case withFeedListener:
		tcpAddr, ok := seqNode.BroadcastServer.ListenerAddr().(*net.TCPAddr)
		if !ok {
			t.Fatalf("failed to cast listener address to *net.TCPAddr")
		}
		port := tcpAddr.Port
		nodeConfig := arbnode.ConfigDefaultL1NonSequencerTest()
		nodeConfig.Feed.Input = *newBroadcastClientConfigTest(port)
		nodeConfig.Feed.Input.Timeout = broadcastclient.DefaultConfig.Timeout
		nodeConfig.TransactionStreamer.TrackBlockMetadataFrom = 1
		extraNode, cleanupExtraNode = builderSeq.Build2ndNode(t, &SecondNodeParams{nodeConfig: nodeConfig, stackConfig: testhelpers.CreateStackConfigForTest(t.TempDir())})
	}

	// Send an L2 tx in the background every two seconds to keep the chain moving.
	go func() {
		tick := time.NewTicker(time.Second * 2)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				tx := seqInfo.PrepareTx("Owner", "Owner", seqInfo.TransferGas, big.NewInt(1), nil)
				err := seqClient.SendTransaction(ctx, tx)
				t.Log("Failed to send test tx", err)
			}
		}
	}()

	// Set up the auction contracts on L2.
	// Deploy the express lane auction contract and erc20 to the parent chain.
	ownerOpts := seqInfo.GetDefaultTransactOpts("Owner", ctx)
	erc20Addr, tx, erc20, err := bindings.DeployMockERC20(&ownerOpts, seqClient)
	Require(t, err)
	if _, err = bind.WaitMined(ctx, seqClient, tx); err != nil {
		t.Fatal(err)
	}
	tx, err = erc20.Initialize(&ownerOpts, "LANE", "LNE", 18)
	Require(t, err)
	if _, err = bind.WaitMined(ctx, seqClient, tx); err != nil {
		t.Fatal(err)
	}

	// Fund the auction contract.
	seqInfo.GenerateAccount("AuctionContract")
	TransferBalance(t, "Owner", "AuctionContract", arbmath.BigMulByUint(oneEth, 500), seqInfo, seqClient, ctx)

	// Mint some tokens to Alice and Bob.
	seqInfo.GenerateAccount("Alice")
	seqInfo.GenerateAccount("Bob")
	TransferBalance(t, "Faucet", "Alice", arbmath.BigMulByUint(oneEth, 500), seqInfo, seqClient, ctx)
	TransferBalance(t, "Faucet", "Bob", arbmath.BigMulByUint(oneEth, 500), seqInfo, seqClient, ctx)
	aliceOpts := seqInfo.GetDefaultTransactOpts("Alice", ctx)
	bobOpts := seqInfo.GetDefaultTransactOpts("Bob", ctx)
	tx, err = erc20.Mint(&ownerOpts, aliceOpts.From, big.NewInt(100))
	Require(t, err)
	if _, err = bind.WaitMined(ctx, seqClient, tx); err != nil {
		t.Fatal(err)
	}
	tx, err = erc20.Mint(&ownerOpts, bobOpts.From, big.NewInt(100))
	Require(t, err)
	if _, err = bind.WaitMined(ctx, seqClient, tx); err != nil {
		t.Fatal(err)
	}

	// Calculate the number of seconds until the next minute
	// and the next timestamp that is a multiple of a minute.
	now := time.Now()
	roundDuration := time.Minute
	// Correctly calculate the remaining time until the next minute
	waitTime := roundDuration - time.Duration(now.Second())*time.Second - time.Duration(now.Nanosecond())*time.Nanosecond
	// Get the current Unix timestamp at the start of the minute
	initialTimestamp := big.NewInt(now.Add(waitTime).Unix())
	initialTimestampUnix := time.Unix(initialTimestamp.Int64(), 0)

	// Deploy the auction manager contract.
	auctionContractAddr, tx, _, err := express_lane_auctiongen.DeployExpressLaneAuction(&ownerOpts, seqClient)
	Require(t, err)
	if _, err = bind.WaitMined(ctx, seqClient, tx); err != nil {
		t.Fatal(err)
	}

	proxyAddr, tx, _, err := mocksgen.DeploySimpleProxy(&ownerOpts, seqClient, auctionContractAddr)
	Require(t, err)
	if _, err = bind.WaitMined(ctx, seqClient, tx); err != nil {
		t.Fatal(err)
	}
	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(proxyAddr, seqClient)
	Require(t, err)

	auctioneerAddr := seqInfo.GetDefaultTransactOpts("AuctionContract", ctx).From
	beneficiary := auctioneerAddr
	biddingToken := erc20Addr
	bidRoundSeconds := uint64(60)
	auctionClosingSeconds := uint64(15)
	reserveSubmissionSeconds := uint64(15)
	minReservePrice := big.NewInt(1) // 1 wei.
	roleAdmin := auctioneerAddr
	tx, err = auctionContract.Initialize(
		&ownerOpts,
		express_lane_auctiongen.InitArgs{
			Auctioneer:   auctioneerAddr,
			BiddingToken: biddingToken,
			Beneficiary:  beneficiary,
			RoundTimingInfo: express_lane_auctiongen.RoundTimingInfo{
				OffsetTimestamp:          initialTimestamp.Int64(),
				RoundDurationSeconds:     bidRoundSeconds,
				AuctionClosingSeconds:    auctionClosingSeconds,
				ReserveSubmissionSeconds: reserveSubmissionSeconds,
			},
			MinReservePrice:       minReservePrice,
			AuctioneerAdmin:       roleAdmin,
			MinReservePriceSetter: roleAdmin,
			ReservePriceSetter:    roleAdmin,
			BeneficiarySetter:     roleAdmin,
			RoundTimingSetter:     roleAdmin,
			MasterAdmin:           roleAdmin,
		},
	)
	Require(t, err)
	if _, err = bind.WaitMined(ctx, seqClient, tx); err != nil {
		t.Fatal(err)
	}
	t.Log("Deployed all the auction manager stuff", auctionContractAddr)
	// We approve the spending of the erc20 for the autonomous auction contract and bid receiver
	// for both Alice and Bob.
	bidReceiverAddr := common.HexToAddress("0x2424242424242424242424242424242424242424")
	maxUint256 := big.NewInt(1)
	maxUint256.Lsh(maxUint256, 256).Sub(maxUint256, big.NewInt(1))

	tx, err = erc20.Approve(
		&aliceOpts, proxyAddr, maxUint256,
	)
	Require(t, err)
	if _, err = bind.WaitMined(ctx, seqClient, tx); err != nil {
		t.Fatal(err)
	}
	tx, err = erc20.Approve(
		&aliceOpts, bidReceiverAddr, maxUint256,
	)
	Require(t, err)
	if _, err = bind.WaitMined(ctx, seqClient, tx); err != nil {
		t.Fatal(err)
	}
	tx, err = erc20.Approve(
		&bobOpts, proxyAddr, maxUint256,
	)
	Require(t, err)
	if _, err = bind.WaitMined(ctx, seqClient, tx); err != nil {
		t.Fatal(err)
	}
	tx, err = erc20.Approve(
		&bobOpts, bidReceiverAddr, maxUint256,
	)
	Require(t, err)
	if _, err = bind.WaitMined(ctx, seqClient, tx); err != nil {
		t.Fatal(err)
	}

	// This is hacky- we are manually starting the ExpressLaneService here instead of letting it be started
	// by the sequencer. This is due to needing to deploy the auction contract first.
	builderSeq.execConfig.Sequencer.Timeboost.Enable = true
	roundTimingInfo, err := gethexec.GetRoundTimingInfo(auctionContract)
	Require(t, err)

	expressLaneTracker := gethexec.NewExpressLaneTracker(
		*roundTimingInfo,
		builderSeq.execConfig.Sequencer.MaxBlockSpeed,
		builderSeq.L2.ExecNode.Backend.APIBackend(),
		auctionContract,
		proxyAddr,
		builderSeq.chainConfig,
		builderSeq.execConfig.Sequencer.Timeboost.EarlySubmissionGrace,
	)

	err = builderSeq.L2.ExecNode.Sequencer.InitializeExpressLaneService(
		auctioneerAddr,
		roundTimingInfo,
		expressLaneTracker)
	Require(t, err)
	builderSeq.L2.ExecNode.TxPreChecker.SetExpressLaneTracker(expressLaneTracker)
	builderSeq.L2.ExecNode.Sequencer.StartExpressLaneService(ctx)
	t.Log("Started express lane service in sequencer")

	if extraNodeTy == withForwardingSeq {
		err = extraNode.ExecNode.Sequencer.InitializeExpressLaneService(
			auctioneerAddr,
			roundTimingInfo,
			expressLaneTracker)
		Require(t, err)
		extraNode.ExecNode.TxPreChecker.SetExpressLaneTracker(expressLaneTracker)
		extraNode.ExecNode.Sequencer.StartExpressLaneService(ctx)
		t.Log("Started express lane service in forwarder sequencer")
	}

	expressLaneTracker.Start(ctx)

	// Set up an autonomous auction contract service that runs in the background in this test.
	redisURL := redisutil.CreateTestRedis(ctx, t)

	// Set up the auctioneer RPC service.
	bidValidatorPort := getRandomPort(t)
	bidValidatorWsPort := getRandomPort(t)
	stackConf := node.Config{
		DataDir:             "", // ephemeral.
		HTTPPort:            bidValidatorPort,
		HTTPHost:            "localhost",
		HTTPModules:         []string{timeboost.AuctioneerNamespace},
		HTTPVirtualHosts:    []string{"localhost"},
		HTTPTimeouts:        rpc.DefaultHTTPTimeouts,
		WSHost:              "localhost",
		WSPort:              bidValidatorWsPort,
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
	cfg := &timeboost.BidValidatorConfig{
		SequencerEndpoint:      fmt.Sprintf("http://localhost:%d", seqPort),
		AuctionContractAddress: proxyAddr.Hex(),
		RedisURL:               redisURL,
		ProducerConfig:         pubsub.TestProducerConfig,
	}
	fetcher := func() *timeboost.BidValidatorConfig {
		return cfg
	}
	bidValidator, err := timeboost.NewBidValidator(
		ctx, stack, fetcher,
	)
	Require(t, err)
	Require(t, stack.Start())
	Require(t, bidValidator.Initialize(ctx))
	bidValidator.Start(ctx)

	auctioneerCfg := &timeboost.AuctioneerServerConfig{
		SequencerEndpoint:      fmt.Sprintf("http://localhost:%d", seqPort),
		AuctionContractAddress: proxyAddr.Hex(),
		RedisURL:               redisURL,
		ConsumerConfig:         pubsub.TestConsumerConfig,
		DbDirectory:            dbDirPath,
		Wallet: genericconf.WalletConfig{
			PrivateKey: fmt.Sprintf("00%x", seqInfo.Accounts["AuctionContract"].PrivateKey.D.Bytes()),
		},
	}
	auctioneerFetcher := func() *timeboost.AuctioneerServerConfig {
		return auctioneerCfg
	}
	am, err := timeboost.NewAuctioneerServer(
		ctx,
		auctioneerFetcher,
	)
	Require(t, err)
	am.Start(ctx)

	// Set up a bidder client for Alice and Bob.
	alicePriv := seqInfo.Accounts["Alice"].PrivateKey
	cfgFetcherAlice := func() *timeboost.BidderClientConfig {
		return &timeboost.BidderClientConfig{
			AuctionContractAddress: proxyAddr.Hex(),
			BidValidatorEndpoint:   fmt.Sprintf("http://localhost:%d", bidValidatorPort),
			ArbitrumNodeEndpoint:   fmt.Sprintf("http://localhost:%d", seqPort),
			Wallet: genericconf.WalletConfig{
				PrivateKey: fmt.Sprintf("00%x", alicePriv.D.Bytes()),
			},
		}
	}
	alice, err := timeboost.NewBidderClient(
		ctx,
		cfgFetcherAlice,
	)
	Require(t, err)

	bobPriv := seqInfo.Accounts["Bob"].PrivateKey
	cfgFetcherBob := func() *timeboost.BidderClientConfig {
		return &timeboost.BidderClientConfig{
			AuctionContractAddress: proxyAddr.Hex(),
			BidValidatorEndpoint:   fmt.Sprintf("http://localhost:%d", bidValidatorPort),
			ArbitrumNodeEndpoint:   fmt.Sprintf("http://localhost:%d", seqPort),
			Wallet: genericconf.WalletConfig{
				PrivateKey: fmt.Sprintf("00%x", bobPriv.D.Bytes()),
			},
		}
	}
	bob, err := timeboost.NewBidderClient(
		ctx,
		cfgFetcherBob,
	)
	Require(t, err)

	alice.Start(ctx)
	bob.Start(ctx)

	// Wait until the initial round.
	timeToWait := time.Until(initialTimestampUnix)
	t.Logf("Waiting until the initial round %v and %v, current time %v", timeToWait, initialTimestampUnix, time.Now())
	<-time.After(timeToWait)

	t.Log("Started auction master stack and bid clients")
	Require(t, alice.Deposit(ctx, big.NewInt(30)))
	Require(t, bob.Deposit(ctx, big.NewInt(30)))

	// Wait until the next timeboost round + a few milliseconds.
	t.Logf("Alice and Bob are now deposited into the autonomous auction contract, waiting %v for bidding round..., timestamp %v", waitTime, time.Now())
	rawRoundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	Require(t, err)
	roundTimingInfo2, err := timeboost.NewRoundTimingInfo(rawRoundTimingInfo)
	Require(t, err)
	time.Sleep(roundTimingInfo2.TimeTilNextRound())
	t.Logf("Reached the bidding round at %v", time.Now())
	time.Sleep(time.Second * 5)
	return proxyAddr, alice, bob, roundDuration, builderSeq, cleanupSeq, extraNode, cleanupExtraNode
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

type expressLaneClient struct {
	stopwaiter.StopWaiter
	sync.Mutex
	privKey             *ecdsa.PrivateKey
	chainId             *big.Int
	roundTimingInfo     timeboost.RoundTimingInfo
	auctionContractAddr common.Address
	client              *rpc.Client
	ethClient           *ethclient.Client
	sequence            uint64
}

func newExpressLaneClient(
	privKey *ecdsa.PrivateKey,
	chainId *big.Int,
	roundTimingInfo timeboost.RoundTimingInfo,
	auctionContractAddr common.Address,
	client *rpc.Client,
) *expressLaneClient {
	return &expressLaneClient{
		privKey:             privKey,
		chainId:             chainId,
		roundTimingInfo:     roundTimingInfo,
		auctionContractAddr: auctionContractAddr,
		client:              client,
		ethClient:           ethclient.NewClient(client),
		sequence:            0,
	}
}

func (elc *expressLaneClient) Start(ctxIn context.Context) {
	elc.StopWaiter.Start(ctxIn, elc)
}

func (elc *expressLaneClient) QueueTransactionWithSequence(ctx context.Context, transaction *types.Transaction, seq uint64) error {
	encodedTx, err := transaction.MarshalBinary()
	if err != nil {
		return err
	}
	msg := &timeboost.JsonExpressLaneSubmission{
		ChainId:                (*hexutil.Big)(elc.chainId),
		Round:                  hexutil.Uint64(elc.roundTimingInfo.RoundNumber()),
		AuctionContractAddress: elc.auctionContractAddr,
		Transaction:            encodedTx,
		SequenceNumber:         hexutil.Uint64(seq),
		Signature:              hexutil.Bytes{},
	}
	msgGo, err := timeboost.JsonSubmissionToGo(msg)
	if err != nil {
		return err
	}
	signingMsg, err := msgGo.ToMessageBytes()
	if err != nil {
		return err
	}
	signature, err := signSubmission(signingMsg, elc.privKey)
	if err != nil {
		return err
	}
	msg.Signature = signature
	promise := elc.sendExpressLaneRPC(msg)
	if _, err := promise.Await(ctx); err != nil {
		return err
	}
	return nil
}

func (elc *expressLaneClient) SendTransactionWithSequence(ctx context.Context, transaction *types.Transaction, seq uint64) error {
	if err := elc.QueueTransactionWithSequence(ctx, transaction, seq); err != nil {
		return err
	}
	if _, err := EnsureTxSucceeded(ctx, elc.ethClient, transaction); err != nil {
		return err
	}
	return nil
}

func (elc *expressLaneClient) SendTransaction(ctx context.Context, transaction *types.Transaction) error {
	elc.Lock()
	defer elc.Unlock()
	err := elc.SendTransactionWithSequence(ctx, transaction, elc.sequence)
	if err == nil {
		elc.sequence += 1
	}
	return err
}

func (elc *expressLaneClient) sendExpressLaneRPC(msg *timeboost.JsonExpressLaneSubmission) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(elc, func(ctx context.Context) (struct{}, error) {
		err := elc.client.CallContext(ctx, nil, "timeboost_sendExpressLaneTransaction", msg)
		return struct{}{}, err
	})
}

func signSubmission(message []byte, key *ecdsa.PrivateKey) ([]byte, error) {
	prefixed := crypto.Keccak256(append([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(message))), message...))
	sig, err := secp256k1.Sign(prefixed, math.PaddedBigBytes(key.D, 32))
	if err != nil {
		return nil, err
	}
	sig[64] += 27
	return sig, nil
}

func getRandomPort(t testing.TB) int {
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()
	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("failed to cast listener address to *net.TCPAddr")
	}
	return tcpAddr.Port
}
