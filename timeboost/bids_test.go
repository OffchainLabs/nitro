package timeboost

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"
)

func TestResolveAuction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testSetup := setupAuctionTest(t, ctx)

	// Set up a new auction master instance that can validate bids.
	am, err := NewAuctioneer(
		testSetup.accounts[0].txOpts, testSetup.chainId, testSetup.backend.Client(), testSetup.expressLaneAuctionAddr, testSetup.expressLaneAuction,
	)
	require.NoError(t, err)

	// Set up two different bidders.
	alice := setupBidderClient(t, ctx, "alice", testSetup.accounts[0], testSetup, am)
	bob := setupBidderClient(t, ctx, "bob", testSetup.accounts[1], testSetup, am)
	require.NoError(t, alice.Deposit(ctx, big.NewInt(5)))
	require.NoError(t, bob.Deposit(ctx, big.NewInt(5)))

	// Form two new bids for the round, with Alice being the bigger one.
	aliceBid, err := alice.Bid(ctx, big.NewInt(2), testSetup.accounts[0].txOpts.From)
	require.NoError(t, err)
	_, err = bob.Bid(ctx, big.NewInt(1), testSetup.accounts[1].txOpts.From)
	require.NoError(t, err)

	// Check the encoded bid bytes are as expected.
	bidBytes, err := alice.auctionContract.GetBidBytes(&bind.CallOpts{}, aliceBid.round, aliceBid.amount, aliceBid.expressLaneController)
	require.NoError(t, err)
	encoded, err := encodeBidValues(new(big.Int).SetUint64(alice.chainId), alice.auctionContractAddress, aliceBid.round, aliceBid.amount, aliceBid.expressLaneController)
	require.NoError(t, err)
	require.Equal(t, bidBytes, encoded)

	// Attempt to resolve the auction before it is closed and receive an error.
	require.ErrorContains(t, am.resolveAuction(ctx), "AuctionNotClosed")

	// // Await resolution.
	// t.Log(time.Now())
	// ticker := newAuctionCloseTicker(am.roundDuration, am.auctionClosingDuration)
	// go ticker.start()
	// <-ticker.c
	// t.Log(time.Now())

	// require.NoError(t, am.resolveAuction(ctx))
	t.Fatal(1)
	// // Expect Alice to have become the next express lane controller.
	// upcomingRound := CurrentRound(am.initialRoundTimestamp, am.roundDuration) + 1
	// controller, err := testSetup.auctionContract.ExpressLaneControllerByRound(&bind.CallOpts{}, big.NewInt(int64(upcomingRound)))
	// require.NoError(t, err)
	// require.Equal(t, alice.txOpts.From, controller)
}

func TestReceiveBid_OK(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testSetup := setupAuctionTest(t, ctx)

	// Set up a new auction master instance that can validate bids.
	am, err := NewAuctioneer(
		testSetup.accounts[1].txOpts, testSetup.chainId, testSetup.backend.Client(), testSetup.expressLaneAuctionAddr, testSetup.expressLaneAuction,
	)
	require.NoError(t, err)

	// Make a deposit as a bidder into the contract.
	bc := setupBidderClient(t, ctx, "alice", testSetup.accounts[0], testSetup, am)
	require.NoError(t, bc.Deposit(ctx, big.NewInt(5)))

	// Form a new bid with an amount.
	newBid, err := bc.Bid(ctx, big.NewInt(5), testSetup.accounts[0].txOpts.From)
	require.NoError(t, err)

	// Check the bid passes validation.
	_, err = am.newValidatedBid(newBid)
	require.NoError(t, err)
}
