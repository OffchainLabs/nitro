package timeboost

import (
	"math/big"
	"testing"
)

type mockSequencer struct{}

// TODO: Mock sequencer subscribes to auction resolution events to
// figure out who is the upcoming express lane auction controller and allows
// sequencing of txs from that controller in their given round.

// Runs a simulation of an express lane auction between different parties,
// with some rounds randomly being canceled due to sequencer downtime.
func TestCompleteAuctionSimulation(t *testing.T) {
	// ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	// defer cancel()

	// testSetup := setupAuctionTest(t, ctx)

	// // Set up two different bidders.
	// alice := setupBidderClient(t, ctx, "alice", testSetup.accounts[0], testSetup)
	// bob := setupBidderClient(t, ctx, "bob", testSetup.accounts[1], testSetup)
	// require.NoError(t, alice.deposit(ctx, big.NewInt(5)))
	// require.NoError(t, bob.deposit(ctx, big.NewInt(5)))

	// // Set up a new auction master instance that can validate bids.
	// am, err := newAuctionMaster(
	// 	testSetup.accounts[2].txOpts, testSetup.chainId, testSetup.backend.Client(), testSetup.auctionContract,
	// )
	// require.NoError(t, err)
	// alice.auctioneer = am
	// bob.auctioneer = am

	// TODO: Start auction master and randomly bid from different bidders in a round.
	// Start the sequencer.
	// Have the winner of the express lane send txs if they detect they are the winner.
	// Auction master will log any deposits that are made to the contract.
}

func TestFilterReservePrice(t *testing.T) {
	a := &Auctioneer{
		reservePrice: big.NewInt(100),
	}
	r := &auctionResult{
		firstPlace: &validatedBid{
			Bid: Bid{amount: big.NewInt(101)},
		},
		secondPlace: &validatedBid{
			Bid: Bid{amount: big.NewInt(101)},
		},
	}
	r1 := a.filterReservePrice(r)
	if r1.firstPlace == nil {
		t.Errorf("firstPlace should not be nil")
	}
	if r1.secondPlace == nil {
		t.Errorf("secondPlace should not be nil")
	}
	r.firstPlace.Bid.amount = big.NewInt(99)
	r.secondPlace.Bid.amount = big.NewInt(99)
	r2 := a.filterReservePrice(r)
	if r2.firstPlace != nil {
		t.Errorf("firstPlace should be nil")
	}
	if r2.secondPlace != nil {
		t.Errorf("secondPlace should be nil")
	}

	r.firstPlace = &validatedBid{Bid{amount: big.NewInt(101)}}
	r.secondPlace = &validatedBid{Bid{amount: big.NewInt(99)}}
	r2 = a.filterReservePrice(r)
	if r1.firstPlace == nil {
		t.Errorf("firstPlace should not be nil")
	}
	if r2.secondPlace != nil {
		t.Errorf("secondPlace should be nil")
	}
}
