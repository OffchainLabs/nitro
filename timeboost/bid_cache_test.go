// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package timeboost

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/util/redisutil"
)

func TestTopTwoBids(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		bids     map[common.Address]*ValidatedBid
		expected *auctionResult
	}{
		{
			name: "single bid",
			bids: map[common.Address]*ValidatedBid{
				common.HexToAddress("0x1"): {Amount: big.NewInt(100), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x1")},
			},
			expected: &auctionResult{
				firstPlace:  &ValidatedBid{Amount: big.NewInt(100), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x1")},
				secondPlace: nil,
			},
		},
		{
			name: "two bids with different amounts",
			bids: map[common.Address]*ValidatedBid{
				common.HexToAddress("0x1"): {Amount: big.NewInt(100), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x1")},
				common.HexToAddress("0x2"): {Amount: big.NewInt(200), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x2"), ExpressLaneController: common.HexToAddress("0x2")},
			},
			expected: &auctionResult{
				firstPlace:  &ValidatedBid{Amount: big.NewInt(200), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x2"), ExpressLaneController: common.HexToAddress("0x2")},
				secondPlace: &ValidatedBid{Amount: big.NewInt(100), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x1")},
			},
		},
		{
			name: "two bids same amount but different hashes",
			bids: map[common.Address]*ValidatedBid{
				common.HexToAddress("0x1"): {Amount: big.NewInt(100), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x1")},
				common.HexToAddress("0x2"): {Amount: big.NewInt(100), ChainId: big.NewInt(2), Bidder: common.HexToAddress("0x2"), ExpressLaneController: common.HexToAddress("0x2")},
			},
			expected: &auctionResult{
				firstPlace:  &ValidatedBid{Amount: big.NewInt(100), ChainId: big.NewInt(2), Bidder: common.HexToAddress("0x2"), ExpressLaneController: common.HexToAddress("0x2")},
				secondPlace: &ValidatedBid{Amount: big.NewInt(100), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x1")},
			},
		},
		{
			name: "many bids with different amounts",
			bids: map[common.Address]*ValidatedBid{
				common.HexToAddress("0x1"): {Amount: big.NewInt(300), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x1")},
				common.HexToAddress("0x2"): {Amount: big.NewInt(100), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x2"), ExpressLaneController: common.HexToAddress("0x2")},
				common.HexToAddress("0x3"): {Amount: big.NewInt(200), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x3"), ExpressLaneController: common.HexToAddress("0x3")},
			},
			expected: &auctionResult{
				firstPlace:  &ValidatedBid{Amount: big.NewInt(300), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x1")},
				secondPlace: &ValidatedBid{Amount: big.NewInt(200), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x3"), ExpressLaneController: common.HexToAddress("0x3")},
			},
		},
		{
			name: "many bids with some tied and others with different amounts",
			bids: map[common.Address]*ValidatedBid{
				common.HexToAddress("0x1"): {Amount: big.NewInt(300), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x1")},
				common.HexToAddress("0x2"): {Amount: big.NewInt(100), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x2"), ExpressLaneController: common.HexToAddress("0x2")},
				common.HexToAddress("0x3"): {Amount: big.NewInt(200), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x3"), ExpressLaneController: common.HexToAddress("0x3")},
				common.HexToAddress("0x4"): {Amount: big.NewInt(200), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x4"), ExpressLaneController: common.HexToAddress("0x4")},
			},
			expected: &auctionResult{
				firstPlace:  &ValidatedBid{Amount: big.NewInt(300), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x1")},
				secondPlace: &ValidatedBid{Amount: big.NewInt(200), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x4"), ExpressLaneController: common.HexToAddress("0x4")},
			},
		},
		{
			name: "many bids and tied for second place",
			bids: map[common.Address]*ValidatedBid{
				common.HexToAddress("0x1"): {Amount: big.NewInt(300), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x1")},
				common.HexToAddress("0x2"): {Amount: big.NewInt(200), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x2"), ExpressLaneController: common.HexToAddress("0x2")},
				common.HexToAddress("0x3"): {Amount: big.NewInt(200), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x3"), ExpressLaneController: common.HexToAddress("0x3")},
			},
			expected: &auctionResult{
				firstPlace:  &ValidatedBid{Amount: big.NewInt(300), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x1")},
				secondPlace: &ValidatedBid{Amount: big.NewInt(200), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x3"), ExpressLaneController: common.HexToAddress("0x3")},
			},
		},
		{
			name: "all bids with the same amount",
			bids: map[common.Address]*ValidatedBid{
				common.HexToAddress("0x1"): {Amount: big.NewInt(100), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x1")},
				common.HexToAddress("0x2"): {Amount: big.NewInt(100), ChainId: big.NewInt(2), Bidder: common.HexToAddress("0x2"), ExpressLaneController: common.HexToAddress("0x2")},
				common.HexToAddress("0x3"): {Amount: big.NewInt(100), ChainId: big.NewInt(3), Bidder: common.HexToAddress("0x3"), ExpressLaneController: common.HexToAddress("0x3")},
			},
			expected: &auctionResult{
				firstPlace:  &ValidatedBid{Amount: big.NewInt(100), ChainId: big.NewInt(3), Bidder: common.HexToAddress("0x3"), ExpressLaneController: common.HexToAddress("0x3")},
				secondPlace: &ValidatedBid{Amount: big.NewInt(100), ChainId: big.NewInt(2), Bidder: common.HexToAddress("0x2"), ExpressLaneController: common.HexToAddress("0x2")},
			},
		},
		{
			name:     "no bids",
			bids:     nil,
			expected: &auctionResult{firstPlace: nil, secondPlace: nil},
		},
		{
			name: "two bidders same controller different amounts",
			bids: map[common.Address]*ValidatedBid{
				common.HexToAddress("0x1"): {Amount: big.NewInt(100), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0xC")},
				common.HexToAddress("0x2"): {Amount: big.NewInt(100), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x2"), ExpressLaneController: common.HexToAddress("0xC")},
			},
			expected: &auctionResult{
				firstPlace:  &ValidatedBid{Amount: big.NewInt(100), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0xC")},
				secondPlace: &ValidatedBid{Amount: big.NewInt(100), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x2"), ExpressLaneController: common.HexToAddress("0xC")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := &bidCache{
				bidsByBidder: tt.bids,
			}
			result := bc.topTwoBids()
			if (result.firstPlace == nil) != (tt.expected.firstPlace == nil) || (result.secondPlace == nil) != (tt.expected.secondPlace == nil) {
				t.Fatalf("expected firstPlace: %v, secondPlace: %v, got firstPlace: %v, secondPlace: %v", tt.expected.firstPlace, tt.expected.secondPlace, result.firstPlace, result.secondPlace)
			}
			if result.firstPlace != nil && result.firstPlace.Amount.Cmp(tt.expected.firstPlace.Amount) != 0 {
				t.Errorf("expected firstPlace amount: %v, got: %v", tt.expected.firstPlace.Amount, result.firstPlace.Amount)
			}
			if result.secondPlace != nil && result.secondPlace.Amount.Cmp(tt.expected.secondPlace.Amount) != 0 {
				t.Errorf("expected secondPlace amount: %v, got: %v", tt.expected.secondPlace.Amount, result.secondPlace.Amount)
			}
		})
	}
}

func TestBidCacheOverwriteByBidder(t *testing.T) {
	t.Parallel()
	bc := newBidCache([32]byte{})

	bidderA := common.HexToAddress("0xA")
	controllerX := common.HexToAddress("0xC1")

	// Bidder A bids 5, then updates to 10
	bc.add(&ValidatedBid{Bidder: bidderA, Amount: big.NewInt(5), ExpressLaneController: controllerX, ChainId: big.NewInt(1)})
	require.Equal(t, 1, bc.size())

	bc.add(&ValidatedBid{Bidder: bidderA, Amount: big.NewInt(10), ExpressLaneController: controllerX, ChainId: big.NewInt(1)})
	require.Equal(t, 1, bc.size())

	result := bc.topTwoBids()
	require.NotNil(t, result.firstPlace)
	require.Equal(t, 0, result.firstPlace.Amount.Cmp(big.NewInt(10)))
	require.Nil(t, result.secondPlace)
}

func TestBidCacheCannotOverwriteOtherBidder(t *testing.T) {
	t.Parallel()
	bc := newBidCache([32]byte{})

	bidderA := common.HexToAddress("0xA")
	bidderB := common.HexToAddress("0xB")
	controllerA := common.HexToAddress("0xC1")

	// Honest bidder A bids 10 ETH for their controller
	bc.add(&ValidatedBid{Bidder: bidderA, Amount: big.NewInt(10), ExpressLaneController: controllerA, ChainId: big.NewInt(1)})

	// Attacker B bids 1 wei for A's controller — should NOT overwrite A's bid
	bc.add(&ValidatedBid{Bidder: bidderB, Amount: big.NewInt(1), ExpressLaneController: controllerA, ChainId: big.NewInt(1)})

	// Both bids should be in the cache (keyed by bidder, not controller)
	require.Equal(t, 2, bc.size())

	result := bc.topTwoBids()
	require.NotNil(t, result.firstPlace)
	require.NotNil(t, result.secondPlace)

	// A's 10 ETH bid wins first place
	require.Equal(t, bidderA, result.firstPlace.Bidder)
	require.Equal(t, 0, result.firstPlace.Amount.Cmp(big.NewInt(10)))

	// B's 1 wei bid is second place
	require.Equal(t, bidderB, result.secondPlace.Bidder)
	require.Equal(t, 0, result.secondPlace.Amount.Cmp(big.NewInt(1)))
}

func TestBidCacheBidderCanChangeController(t *testing.T) {
	t.Parallel()
	bc := newBidCache([32]byte{})

	bidderA := common.HexToAddress("0xA")
	controllerX := common.HexToAddress("0xC1")
	controllerY := common.HexToAddress("0xC2")

	// Bidder A bids for controller X, then changes to controller Y
	bc.add(&ValidatedBid{Bidder: bidderA, Amount: big.NewInt(5), ExpressLaneController: controllerX, ChainId: big.NewInt(1)})
	bc.add(&ValidatedBid{Bidder: bidderA, Amount: big.NewInt(5), ExpressLaneController: controllerY, ChainId: big.NewInt(1)})

	require.Equal(t, 1, bc.size())

	result := bc.topTwoBids()
	require.NotNil(t, result.firstPlace)
	require.Equal(t, controllerY, result.firstPlace.ExpressLaneController)
	require.Nil(t, result.secondPlace)
}

func TestBidCacheTwoBiddersSameController(t *testing.T) {
	t.Parallel()
	bc := newBidCache([32]byte{})

	bidderA := common.HexToAddress("0xA")
	bidderB := common.HexToAddress("0xB")
	controllerX := common.HexToAddress("0xC1")

	// Two different bidders both want the same controller
	bc.add(&ValidatedBid{Bidder: bidderA, Amount: big.NewInt(10), ExpressLaneController: controllerX, ChainId: big.NewInt(1)})
	bc.add(&ValidatedBid{Bidder: bidderB, Amount: big.NewInt(5), ExpressLaneController: controllerX, ChainId: big.NewInt(1)})

	// Both entries exist — keyed by bidder
	require.Equal(t, 2, bc.size())

	result := bc.topTwoBids()
	require.NotNil(t, result.firstPlace)
	require.NotNil(t, result.secondPlace)

	// Higher bidder wins
	require.Equal(t, bidderA, result.firstPlace.Bidder)
	require.Equal(t, 0, result.firstPlace.Amount.Cmp(big.NewInt(10)))
	require.Equal(t, controllerX, result.firstPlace.ExpressLaneController)

	require.Equal(t, bidderB, result.secondPlace.Bidder)
	require.Equal(t, 0, result.secondPlace.Amount.Cmp(big.NewInt(5)))
}

func TestBidCacheConcurrentAddAndClear(t *testing.T) {
	t.Parallel()
	cache := newBidCache([32]byte{})

	const numAdds = 1000
	var wg sync.WaitGroup

	// Writer goroutine: adds bids concurrently.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numAdds; i++ {
			addr := common.BigToAddress(big.NewInt(int64(i)))
			cache.add(&ValidatedBid{
				Bidder:                addr,
				Amount:                big.NewInt(int64(i)),
				ExpressLaneController: addr,
				ChainId:               big.NewInt(1),
			})
		}
	}()

	// Clearer goroutine: periodically clears the cache (as auction resolution does).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			cache.topTwoBidsAndClear()
			time.Sleep(10 * time.Microsecond)
		}
	}()

	wg.Wait()
	// The main assertion is that no panic or race occurred (run with -race).
	_ = cache.topTwoBids()
}

func TestTopTwoBidsAndClearReturnsCorrectResultAndEmptiesCache(t *testing.T) {
	t.Parallel()
	bc := newBidCache([32]byte{})

	bidderA := common.HexToAddress("0xA")
	bidderB := common.HexToAddress("0xB")
	bidderC := common.HexToAddress("0xC")

	bc.add(&ValidatedBid{Bidder: bidderA, Amount: big.NewInt(100), ExpressLaneController: bidderA, ChainId: big.NewInt(1)})
	bc.add(&ValidatedBid{Bidder: bidderB, Amount: big.NewInt(300), ExpressLaneController: bidderB, ChainId: big.NewInt(1)})
	bc.add(&ValidatedBid{Bidder: bidderC, Amount: big.NewInt(200), ExpressLaneController: bidderC, ChainId: big.NewInt(1)})
	require.Equal(t, 3, bc.size())

	result := bc.topTwoBidsAndClear()

	// Verify the result contains the correct top two bids.
	require.NotNil(t, result.firstPlace)
	require.NotNil(t, result.secondPlace)
	require.Equal(t, bidderB, result.firstPlace.Bidder)
	require.Equal(t, 0, result.firstPlace.Amount.Cmp(big.NewInt(300)))
	require.Equal(t, bidderC, result.secondPlace.Bidder)
	require.Equal(t, 0, result.secondPlace.Amount.Cmp(big.NewInt(200)))

	// Verify the cache is empty after the call.
	require.Equal(t, 0, bc.size())

	// Verify a subsequent topTwoBids returns nil.
	emptyResult := bc.topTwoBids()
	require.Nil(t, emptyResult.firstPlace)
	require.Nil(t, emptyResult.secondPlace)
}

func TestTopTwoBidsAndClearConcurrentAddGoesToFreshMap(t *testing.T) {
	t.Parallel()
	bc := newBidCache([32]byte{})

	// Add initial bids.
	bc.add(&ValidatedBid{Bidder: common.HexToAddress("0xA"), Amount: big.NewInt(100), ExpressLaneController: common.HexToAddress("0xA"), ChainId: big.NewInt(1)})

	// Clear and simultaneously add a new bid.
	var wg sync.WaitGroup
	var result *auctionResult

	wg.Add(2)
	go func() {
		defer wg.Done()
		result = bc.topTwoBidsAndClear()
	}()
	go func() {
		defer wg.Done()
		bc.add(&ValidatedBid{Bidder: common.HexToAddress("0xB"), Amount: big.NewInt(200), ExpressLaneController: common.HexToAddress("0xB"), ChainId: big.NewInt(1)})
	}()
	wg.Wait()

	// The bid from 0xA must appear in the result or in the cache (never lost).
	// The bid from 0xB must appear in the result or in the cache (never lost).
	resultCount := 0
	if result.firstPlace != nil {
		resultCount++
	}
	if result.secondPlace != nil {
		resultCount++
	}
	cacheCount := bc.size()

	// Total bids across result and cache must equal 2 (both bids accounted for).
	require.Equal(t, 2, resultCount+cacheCount, "every bid must appear in the result or the fresh map, never lost")
}

func BenchmarkBidValidation(b *testing.B) {
	b.StopTimer()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	redisURL := redisutil.CreateTestRedis(ctx, b)
	testSetup := setupAuctionTest(b, ctx)
	bv, endpoint := setupBidValidator(b, ctx, redisURL, testSetup)
	bc := setupBidderClient(b, ctx, testSetup.accounts[0], testSetup, endpoint)
	require.NoError(b, bc.Deposit(ctx, big.NewInt(5)))

	// Form a valid bid.
	newBid, err := bc.Bid(ctx, big.NewInt(5), testSetup.accounts[0].txOpts.From)
	require.NoError(b, err)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err = bv.validateBid(newBid, bv.auctionContract.BalanceOf)
		require.NoError(b, err)
	}
}

func setupBidValidator(t testing.TB, ctx context.Context, redisURL string, testSetup *auctionSetup) (*BidValidator, string) {
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
	cfg := &BidValidatorConfig{
		RpcEndpoint:            testSetup.endpoint,
		AuctionContractAddress: testSetup.expressLaneAuctionAddr.Hex(),
		RedisURL:               redisURL,
		ProducerConfig:         pubsub.TestProducerConfig,
		MaxBidsPerSender:       5,
	}
	fetcher := func() *BidValidatorConfig {
		return cfg
	}
	bidValidator, err := NewBidValidator(
		ctx,
		stack,
		fetcher,
	)
	require.NoError(t, err)
	require.NoError(t, bidValidator.Initialize(ctx))
	require.NoError(t, stack.Start())
	bidValidator.Start(ctx)
	return bidValidator, fmt.Sprintf("http://localhost:%d", randHttp)
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
