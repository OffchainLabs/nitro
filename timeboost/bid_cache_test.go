package timeboost

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"testing"

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
				common.HexToAddress("0x1"): {Amount: big.NewInt(100), ChainId: big.NewInt(1), ExpressLaneController: common.HexToAddress("0x1")},
			},
			expected: &auctionResult{
				firstPlace:  &ValidatedBid{Amount: big.NewInt(100), ChainId: big.NewInt(1), ExpressLaneController: common.HexToAddress("0x1")},
				secondPlace: nil,
			},
		},
		{
			name: "two bids with different amounts",
			bids: map[common.Address]*ValidatedBid{
				common.HexToAddress("0x1"): {Amount: big.NewInt(100), ChainId: big.NewInt(1), ExpressLaneController: common.HexToAddress("0x1")},
				common.HexToAddress("0x2"): {Amount: big.NewInt(200), ChainId: big.NewInt(1), ExpressLaneController: common.HexToAddress("0x2")},
			},
			expected: &auctionResult{
				firstPlace:  &ValidatedBid{Amount: big.NewInt(200), ChainId: big.NewInt(1), ExpressLaneController: common.HexToAddress("0x2")},
				secondPlace: &ValidatedBid{Amount: big.NewInt(100), ChainId: big.NewInt(1), ExpressLaneController: common.HexToAddress("0x1")},
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
			name: "many bids but all same amount",
			bids: map[common.Address]*ValidatedBid{
				common.HexToAddress("0x1"): {Amount: big.NewInt(300), ChainId: big.NewInt(1), ExpressLaneController: common.HexToAddress("0x1")},
				common.HexToAddress("0x2"): {Amount: big.NewInt(100), ChainId: big.NewInt(1), ExpressLaneController: common.HexToAddress("0x2")},
				common.HexToAddress("0x3"): {Amount: big.NewInt(200), ChainId: big.NewInt(1), ExpressLaneController: common.HexToAddress("0x3")},
			},
			expected: &auctionResult{
				firstPlace:  &ValidatedBid{Amount: big.NewInt(300), ChainId: big.NewInt(1), ExpressLaneController: common.HexToAddress("0x1")},
				secondPlace: &ValidatedBid{Amount: big.NewInt(200), ChainId: big.NewInt(1), ExpressLaneController: common.HexToAddress("0x3")},
			},
		},
		{
			name: "many bids with some tied and others with different amounts",
			bids: map[common.Address]*ValidatedBid{
				common.HexToAddress("0x1"): {Amount: big.NewInt(300), ChainId: big.NewInt(1), ExpressLaneController: common.HexToAddress("0x1")},
				common.HexToAddress("0x2"): {Amount: big.NewInt(100), ChainId: big.NewInt(1), ExpressLaneController: common.HexToAddress("0x2")},
				common.HexToAddress("0x3"): {Amount: big.NewInt(200), ChainId: big.NewInt(1), ExpressLaneController: common.HexToAddress("0x3")},
				common.HexToAddress("0x4"): {Amount: big.NewInt(200), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x4")},
			},
			expected: &auctionResult{
				firstPlace:  &ValidatedBid{Amount: big.NewInt(300), ChainId: big.NewInt(1), ExpressLaneController: common.HexToAddress("0x1")},
				secondPlace: &ValidatedBid{Amount: big.NewInt(200), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x4")},
			},
		},
		{
			name: "many bids and tied for second place",
			bids: map[common.Address]*ValidatedBid{
				common.HexToAddress("0x1"): {Amount: big.NewInt(300), ChainId: big.NewInt(1), ExpressLaneController: common.HexToAddress("0x1")},
				common.HexToAddress("0x2"): {Amount: big.NewInt(200), ChainId: big.NewInt(1), ExpressLaneController: common.HexToAddress("0x2")},
				common.HexToAddress("0x3"): {Amount: big.NewInt(200), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x3")},
			},
			expected: &auctionResult{
				firstPlace:  &ValidatedBid{Amount: big.NewInt(300), ChainId: big.NewInt(1), ExpressLaneController: common.HexToAddress("0x1")},
				secondPlace: &ValidatedBid{Amount: big.NewInt(200), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x3")},
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
			name: "identical bids",
			bids: map[common.Address]*ValidatedBid{
				common.HexToAddress("0x1"): {Amount: big.NewInt(100), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x1")},
				common.HexToAddress("0x2"): {Amount: big.NewInt(100), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x2")},
			},
			expected: &auctionResult{
				firstPlace:  &ValidatedBid{Amount: big.NewInt(100), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x1")},
				secondPlace: &ValidatedBid{Amount: big.NewInt(100), ChainId: big.NewInt(1), Bidder: common.HexToAddress("0x1"), ExpressLaneController: common.HexToAddress("0x2")},
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
			if result.firstPlace != nil && result.firstPlace.Amount.Cmp(tt.expected.firstPlace.Amount) != 0 {
				t.Errorf("expected firstPlace amount: %v, got: %v", tt.expected.firstPlace.Amount, result.firstPlace.Amount)
			}
			if result.secondPlace != nil && result.secondPlace.Amount.Cmp(tt.expected.secondPlace.Amount) != 0 {
				t.Errorf("expected secondPlace amount: %v, got: %v", tt.expected.secondPlace.Amount, result.secondPlace.Amount)
			}
		})
	}
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
		SequencerEndpoint:      testSetup.endpoint,
		AuctionContractAddress: testSetup.expressLaneAuctionAddr.Hex(),
		RedisURL:               redisURL,
		ProducerConfig:         pubsub.TestProducerConfig,
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
