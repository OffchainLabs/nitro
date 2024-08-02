package timeboost

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestResolveAuction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testSetup := setupAuctionTest(t, ctx)
	am, endpoint := setupAuctioneer(t, ctx, testSetup)

	// Set up two different bidders.
	alice := setupBidderClient(t, ctx, "alice", testSetup.accounts[0], testSetup, endpoint)
	bob := setupBidderClient(t, ctx, "bob", testSetup.accounts[1], testSetup, endpoint)
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

	filterOpts := &bind.FilterOpts{
		Context: ctx,
		Start:   0,
		End:     nil,
	}
	it, err := am.auctionContract.FilterAuctionResolved(filterOpts, nil, nil, nil)
	require.NoError(t, err)
	aliceWon := false
	for it.Next() {
		// Expect Alice to have become the next express lane controller.
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
	am, endpoint := setupAuctioneer(t, ctx, testSetup)
	bc := setupBidderClient(t, ctx, "alice", testSetup.accounts[0], testSetup, endpoint)
	require.NoError(t, bc.Deposit(ctx, big.NewInt(5)))

	// Form a new bid with an amount.
	newBid, err := bc.Bid(ctx, big.NewInt(5), testSetup.accounts[0].txOpts.From)
	require.NoError(t, err)

	// Check the bid passes validation.
	_, err = am.validateBid(newBid, am.auctionContract.BalanceOf, am.fetchReservePrice)
	require.NoError(t, err)

	topTwoBids := am.bidCache.topTwoBids()
	require.True(t, topTwoBids.secondPlace == nil)
	require.True(t, topTwoBids.firstPlace.expressLaneController == newBid.ExpressLaneController)
}

func TestTopTwoBids(t *testing.T) {
	tests := []struct {
		name     string
		bids     map[common.Address]*validatedBid
		expected *auctionResult
	}{
		{
			name: "Single Bid",
			bids: map[common.Address]*validatedBid{
				common.HexToAddress("0x1"): {amount: big.NewInt(100), chainId: big.NewInt(1), expressLaneController: common.HexToAddress("0x1")},
			},
			expected: &auctionResult{
				firstPlace:  &validatedBid{amount: big.NewInt(100), chainId: big.NewInt(1), expressLaneController: common.HexToAddress("0x1")},
				secondPlace: nil,
			},
		},
		{
			name: "Two Bids with Different Amounts",
			bids: map[common.Address]*validatedBid{
				common.HexToAddress("0x1"): {amount: big.NewInt(100), chainId: big.NewInt(1), expressLaneController: common.HexToAddress("0x1")},
				common.HexToAddress("0x2"): {amount: big.NewInt(200), chainId: big.NewInt(1), expressLaneController: common.HexToAddress("0x2")},
			},
			expected: &auctionResult{
				firstPlace:  &validatedBid{amount: big.NewInt(200), chainId: big.NewInt(1), expressLaneController: common.HexToAddress("0x2")},
				secondPlace: &validatedBid{amount: big.NewInt(100), chainId: big.NewInt(1), expressLaneController: common.HexToAddress("0x1")},
			},
		},
		{
			name: "Two Bids with Same Amount and Different Hashes",
			bids: map[common.Address]*validatedBid{
				common.HexToAddress("0x1"): {amount: big.NewInt(100), chainId: big.NewInt(1), bidder: common.HexToAddress("0x1"), expressLaneController: common.HexToAddress("0x1")},
				common.HexToAddress("0x2"): {amount: big.NewInt(100), chainId: big.NewInt(2), bidder: common.HexToAddress("0x2"), expressLaneController: common.HexToAddress("0x2")},
			},
			expected: &auctionResult{
				firstPlace:  &validatedBid{amount: big.NewInt(100), chainId: big.NewInt(2), bidder: common.HexToAddress("0x2"), expressLaneController: common.HexToAddress("0x2")},
				secondPlace: &validatedBid{amount: big.NewInt(100), chainId: big.NewInt(1), bidder: common.HexToAddress("0x1"), expressLaneController: common.HexToAddress("0x1")},
			},
		},
		{
			name: "More Than Two Bids, All Unique Amounts",
			bids: map[common.Address]*validatedBid{
				common.HexToAddress("0x1"): {amount: big.NewInt(300), chainId: big.NewInt(1), expressLaneController: common.HexToAddress("0x1")},
				common.HexToAddress("0x2"): {amount: big.NewInt(100), chainId: big.NewInt(1), expressLaneController: common.HexToAddress("0x2")},
				common.HexToAddress("0x3"): {amount: big.NewInt(200), chainId: big.NewInt(1), expressLaneController: common.HexToAddress("0x3")},
			},
			expected: &auctionResult{
				firstPlace:  &validatedBid{amount: big.NewInt(300), chainId: big.NewInt(1), expressLaneController: common.HexToAddress("0x1")},
				secondPlace: &validatedBid{amount: big.NewInt(200), chainId: big.NewInt(1), expressLaneController: common.HexToAddress("0x3")},
			},
		},
		{
			name: "More Than Two Bids, Some with Same Amounts",
			bids: map[common.Address]*validatedBid{
				common.HexToAddress("0x1"): {amount: big.NewInt(300), chainId: big.NewInt(1), expressLaneController: common.HexToAddress("0x1")},
				common.HexToAddress("0x2"): {amount: big.NewInt(100), chainId: big.NewInt(1), expressLaneController: common.HexToAddress("0x2")},
				common.HexToAddress("0x3"): {amount: big.NewInt(200), chainId: big.NewInt(1), expressLaneController: common.HexToAddress("0x3")},
				common.HexToAddress("0x4"): {amount: big.NewInt(200), chainId: big.NewInt(1), bidder: common.HexToAddress("0x1"), expressLaneController: common.HexToAddress("0x4")},
			},
			expected: &auctionResult{
				firstPlace:  &validatedBid{amount: big.NewInt(300), chainId: big.NewInt(1), expressLaneController: common.HexToAddress("0x1")},
				secondPlace: &validatedBid{amount: big.NewInt(200), chainId: big.NewInt(1), bidder: common.HexToAddress("0x1"), expressLaneController: common.HexToAddress("0x4")},
			},
		},
		{
			name: "More Than Two Bids, Tied for Second Place",
			bids: map[common.Address]*validatedBid{
				common.HexToAddress("0x1"): {amount: big.NewInt(300), chainId: big.NewInt(1), expressLaneController: common.HexToAddress("0x1")},
				common.HexToAddress("0x2"): {amount: big.NewInt(200), chainId: big.NewInt(1), expressLaneController: common.HexToAddress("0x2")},
				common.HexToAddress("0x3"): {amount: big.NewInt(200), chainId: big.NewInt(1), bidder: common.HexToAddress("0x1"), expressLaneController: common.HexToAddress("0x3")},
			},
			expected: &auctionResult{
				firstPlace:  &validatedBid{amount: big.NewInt(300), chainId: big.NewInt(1), expressLaneController: common.HexToAddress("0x1")},
				secondPlace: &validatedBid{amount: big.NewInt(200), chainId: big.NewInt(1), bidder: common.HexToAddress("0x1"), expressLaneController: common.HexToAddress("0x3")},
			},
		},
		{
			name: "All Bids with the Same Amount",
			bids: map[common.Address]*validatedBid{
				common.HexToAddress("0x1"): {amount: big.NewInt(100), chainId: big.NewInt(1), bidder: common.HexToAddress("0x1"), expressLaneController: common.HexToAddress("0x1")},
				common.HexToAddress("0x2"): {amount: big.NewInt(100), chainId: big.NewInt(2), bidder: common.HexToAddress("0x2"), expressLaneController: common.HexToAddress("0x2")},
				common.HexToAddress("0x3"): {amount: big.NewInt(100), chainId: big.NewInt(3), bidder: common.HexToAddress("0x3"), expressLaneController: common.HexToAddress("0x3")},
			},
			expected: &auctionResult{
				firstPlace:  &validatedBid{amount: big.NewInt(100), chainId: big.NewInt(3), bidder: common.HexToAddress("0x3"), expressLaneController: common.HexToAddress("0x3")},
				secondPlace: &validatedBid{amount: big.NewInt(100), chainId: big.NewInt(2), bidder: common.HexToAddress("0x2"), expressLaneController: common.HexToAddress("0x2")},
			},
		},
		{
			name:     "No Bids",
			bids:     nil,
			expected: &auctionResult{firstPlace: nil, secondPlace: nil},
		},
		{
			name: "Identical Bids",
			bids: map[common.Address]*validatedBid{
				common.HexToAddress("0x1"): {amount: big.NewInt(100), chainId: big.NewInt(1), bidder: common.HexToAddress("0x1"), expressLaneController: common.HexToAddress("0x1")},
				common.HexToAddress("0x2"): {amount: big.NewInt(100), chainId: big.NewInt(1), bidder: common.HexToAddress("0x1"), expressLaneController: common.HexToAddress("0x2")},
			},
			expected: &auctionResult{
				firstPlace:  &validatedBid{amount: big.NewInt(100), chainId: big.NewInt(1), bidder: common.HexToAddress("0x1"), expressLaneController: common.HexToAddress("0x1")},
				secondPlace: &validatedBid{amount: big.NewInt(100), chainId: big.NewInt(1), bidder: common.HexToAddress("0x1"), expressLaneController: common.HexToAddress("0x2")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := &bidCache{
				bidsByExpressLaneControllerAddr: tt.bids,
			}
			result := bc.topTwoBids()
			if (result.firstPlace == nil) != (tt.expected.firstPlace == nil) || (result.secondPlace == nil) != (tt.expected.secondPlace == nil) {
				t.Fatalf("expected firstPlace: %v, secondPlace: %v, got firstPlace: %v, secondPlace: %v", tt.expected.firstPlace, tt.expected.secondPlace, result.firstPlace, result.secondPlace)
			}
			if result.firstPlace != nil && result.firstPlace.amount.Cmp(tt.expected.firstPlace.amount) != 0 {
				t.Errorf("expected firstPlace amount: %v, got: %v", tt.expected.firstPlace.amount, result.firstPlace.amount)
			}
			if result.secondPlace != nil && result.secondPlace.amount.Cmp(tt.expected.secondPlace.amount) != 0 {
				t.Errorf("expected secondPlace amount: %v, got: %v", tt.expected.secondPlace.amount, result.secondPlace.amount)
			}
		})
	}
}

func BenchmarkBidValidation(b *testing.B) {
	b.StopTimer()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testSetup := setupAuctionTest(b, ctx)
	am, endpoint := setupAuctioneer(b, ctx, testSetup)
	bc := setupBidderClient(b, ctx, "alice", testSetup.accounts[0], testSetup, endpoint)
	require.NoError(b, bc.Deposit(ctx, big.NewInt(5)))

	// Form a valid bid.
	newBid, err := bc.Bid(ctx, big.NewInt(5), testSetup.accounts[0].txOpts.From)
	require.NoError(b, err)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		am.validateBid(newBid, am.auctionContract.BalanceOf, am.fetchReservePrice)
	}
}

func setupAuctioneer(t testing.TB, ctx context.Context, testSetup *auctionSetup) (*Auctioneer, string) {
	// Set up a new auction master instance that can validate bids.
	// Set up the auctioneer RPC service.
	randHttp := getRandomPort(t)
	stackConf := node.Config{
		DataDir:             "", // ephemeral.
		HTTPPort:            randHttp,
		HTTPModules:         []string{AuctioneerNamespace},
		HTTPHost:            "localhost",
		HTTPVirtualHosts:    []string{"localhost"},
		HTTPTimeouts:        rpc.DefaultHTTPTimeouts,
		WSPort:              getRandomPort(t),
		WSModules:           []string{AuctioneerNamespace},
		WSHost:              "localhost",
		GraphQLVirtualHosts: []string{"localhost"},
		P2P: p2p.Config{
			ListenAddr:  "",
			NoDial:      true,
			NoDiscovery: true,
		},
	}
	stack, err := node.New(&stackConf)
	require.NoError(t, err)
	am, err := NewAuctioneer(
		testSetup.accounts[0].txOpts, []*big.Int{testSetup.chainId}, stack, testSetup.backend.Client(), testSetup.expressLaneAuctionAddr,
	)
	require.NoError(t, err)
	go am.Start(ctx)
	require.NoError(t, stack.Start())
	return am, fmt.Sprintf("http://localhost:%d", randHttp)
}

func getRandomPort(t testing.TB) int {
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}
