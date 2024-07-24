package timeboost

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"
)

func TestResolveAuction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testSetup := setupAuctionTest(t, ctx)

	// Set up a new auction master instance that can validate bids.
	am, err := NewAuctioneer(
		testSetup.accounts[0].txOpts, []uint64{testSetup.chainId.Uint64()}, testSetup.backend.Client(), testSetup.expressLaneAuctionAddr, testSetup.expressLaneAuction,
	)
	require.NoError(t, err)

	// Set up two different bidders.
	alice := setupBidderClient(t, ctx, "alice", testSetup.accounts[0], testSetup, am)
	bob := setupBidderClient(t, ctx, "bob", testSetup.accounts[1], testSetup, am)
	require.NoError(t, alice.Deposit(ctx, big.NewInt(5)))
	require.NoError(t, bob.Deposit(ctx, big.NewInt(5)))

	// Wait until the initial round.
	info, err := alice.auctionContract.RoundTimingInfo(&bind.CallOpts{})
	require.NoError(t, err)
	timeToWait := time.Until(time.Unix(int64(info.OffsetTimestamp), 0))
	<-time.After(timeToWait)
	time.Sleep(time.Second) // Add a second of wait so that we are within a round.

	// Form two new bids for the round, with Alice being the bigger one.
	_, err = alice.Bid(ctx, big.NewInt(2), alice.txOpts.From)
	require.NoError(t, err)
	_, err = bob.Bid(ctx, big.NewInt(1), bob.txOpts.From)
	require.NoError(t, err)

	// Attempt to resolve the auction before it is closed and receive an error.
	require.ErrorContains(t, am.resolveAuction(ctx), "AuctionNotClosed")

	// Await resolution.
	t.Log(time.Now())
	ticker := newAuctionCloseTicker(am.roundDuration, am.auctionClosingDuration)
	go ticker.start()
	<-ticker.c
	require.NoError(t, am.resolveAuction(ctx))
	// Expect Alice to have become the next express lane controller.

	filterOpts := &bind.FilterOpts{
		Context: ctx,
		Start:   0,
		End:     nil,
	}
	it, err := am.auctionContract.FilterAuctionResolved(filterOpts, nil, nil, nil)
	require.NoError(t, err)
	aliceWon := false
	for it.Next() {
		if it.Event.FirstPriceBidder == alice.txOpts.From {
			aliceWon = true
		}
	}
	require.True(t, aliceWon)
}

func TestReceiveBid_OK(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testSetup := setupAuctionTest(t, ctx)

	// Set up a new auction master instance that can validate bids.
	am, err := NewAuctioneer(
		testSetup.accounts[1].txOpts, []uint64{testSetup.chainId.Uint64()}, testSetup.backend.Client(), testSetup.expressLaneAuctionAddr, testSetup.expressLaneAuction,
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
