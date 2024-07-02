package timeboost

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"
)

func TestWinningBidderBecomesExpressLaneController(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testSetup := setupAuctionTest(t, ctx)

	// Set up two different bidders.
	alice := setupBidderClient(t, ctx, "alice", testSetup.accounts[0], testSetup)
	bob := setupBidderClient(t, ctx, "bob", testSetup.accounts[1], testSetup)
	require.NoError(t, alice.Deposit(ctx, big.NewInt(5)))
	require.NoError(t, bob.Deposit(ctx, big.NewInt(5)))

	// Set up a new auction master instance that can validate bids.
	am, err := NewAuctionMaster(
		testSetup.accounts[2].txOpts, testSetup.chainId, testSetup.backend.Client(), testSetup.auctionContract,
	)
	require.NoError(t, err)
	alice.auctionMaster = am
	bob.auctionMaster = am

	// Form two new bids for the round, with Alice being the bigger one.
	aliceBid, err := alice.Bid(ctx, big.NewInt(2))
	require.NoError(t, err)
	bobBid, err := bob.Bid(ctx, big.NewInt(1))
	require.NoError(t, err)
	_, _ = aliceBid, bobBid

	// Resolve the auction.
	require.NoError(t, am.resolveAuctions(ctx))

	// Expect Alice to have become the next express lane controller.
	upcomingRound := CurrentRound(am.initialRoundTimestamp, am.roundDuration) + 1
	controller, err := testSetup.auctionContract.ExpressLaneControllerByRound(&bind.CallOpts{}, big.NewInt(int64(upcomingRound)))
	require.NoError(t, err)
	require.Equal(t, alice.txOpts.From, controller)
}

func TestSubmitBid_OK(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testSetup := setupAuctionTest(t, ctx)

	// Make a deposit as a bidder into the contract.
	bc := setupBidderClient(t, ctx, "alice", testSetup.accounts[0], testSetup)
	require.NoError(t, bc.Deposit(ctx, big.NewInt(5)))

	// Set up a new auction master instance that can validate bids.
	am, err := NewAuctionMaster(
		testSetup.accounts[1].txOpts, testSetup.chainId, testSetup.backend.Client(), testSetup.auctionContract,
	)
	require.NoError(t, err)
	bc.auctionMaster = am

	// Form a new bid with an amount.
	newBid, err := bc.Bid(ctx, big.NewInt(5))
	require.NoError(t, err)

	// Check the bid passes validation.
	_, err = am.newValidatedBid(newBid)
	require.NoError(t, err)
}
