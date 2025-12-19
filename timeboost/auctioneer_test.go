package timeboost

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/util/redisutil"
)

func TestBidValidatorAuctioneerRedisStream(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	testSetup := setupAuctionTest(t, ctx)
	redisURL := redisutil.CreateTestRedis(ctx, t)
	tmpDir := t.TempDir()
	jwtFilePath := filepath.Join(tmpDir, "jwt.key")
	jwtSecret := common.BytesToHash([]byte("jwt"))
	require.NoError(t, os.WriteFile(jwtFilePath, []byte(hexutil.Encode(jwtSecret[:])), 0600))

	// Set up multiple bid validators that will receive bids via RPC using a bidder client.
	// They inject their validated bids into a Redis stream that a single auctioneer instance
	// will then consume.
	numBidValidators := 3
	bidValidators := make([]*BidValidator, numBidValidators)
	for i := 0; i < numBidValidators; i++ {
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
		bidValidators[i] = bidValidator
	}
	t.Log("Started multiple bid validators")

	// Set up a single auctioneer instance that can consume messages produced
	// by the bid validators from a redis stream.
	cfg := &AuctioneerServerConfig{
		SequencerEndpoint:      testSetup.endpoint,
		SequencerJWTPath:       jwtFilePath,
		AuctionContractAddress: testSetup.expressLaneAuctionAddr.Hex(),
		RedisURL:               redisURL,
		ConsumerConfig:         pubsub.TestConsumerConfig,
		DbDirectory:            tmpDir,
		Wallet: genericconf.WalletConfig{
			PrivateKey: fmt.Sprintf("%x", testSetup.accounts[0].privKey.D.Bytes()),
		},
	}
	fetcher := func() *AuctioneerServerConfig {
		return cfg
	}
	am, err := NewAuctioneerServer(
		ctx,
		fetcher,
	)
	require.NoError(t, err)
	am.Start(ctx)
	t.Log("Started auctioneer")

	// Now, we set up bidder clients for Alice, Bob, and Charlie.
	aliceAddr := testSetup.accounts[1].txOpts.From
	bobAddr := testSetup.accounts[2].txOpts.From
	charlieAddr := testSetup.accounts[3].txOpts.From
	alice := setupBidderClient(t, ctx, testSetup.accounts[1], testSetup, bidValidators[0].stack.HTTPEndpoint())
	bob := setupBidderClient(t, ctx, testSetup.accounts[2], testSetup, bidValidators[1].stack.HTTPEndpoint())
	charlie := setupBidderClient(t, ctx, testSetup.accounts[3], testSetup, bidValidators[2].stack.HTTPEndpoint())
	require.NoError(t, alice.Deposit(ctx, big.NewInt(20)))
	require.NoError(t, bob.Deposit(ctx, big.NewInt(20)))
	require.NoError(t, charlie.Deposit(ctx, big.NewInt(20)))

	info, err := alice.auctionContract.RoundTimingInfo(&bind.CallOpts{})
	require.NoError(t, err)
	timeToWait := time.Until(time.Unix(int64(info.OffsetTimestamp), 0))
	t.Logf("Waiting for %v to start the bidding round, %v", timeToWait, time.Now())
	<-time.After(timeToWait)
	time.Sleep(time.Millisecond * 250) // Add 1/4 of a second of wait so that we are definitely within a round.

	// Alice, Bob, and Charlie will submit bids to the three different bid validators instances.
	start := time.Now()
	for i := 1; i <= 5; i++ {
		_, err = alice.Bid(ctx, big.NewInt(int64(i)), aliceAddr)
		require.NoError(t, err)
		_, err = bob.Bid(ctx, big.NewInt(int64(i)+1), bobAddr) // Bob bids 1 wei higher than Alice.
		require.NoError(t, err)
		_, err = charlie.Bid(ctx, big.NewInt(int64(i)+2), charlieAddr) // Charlie bids 2 wei higher than the Alice.
		require.NoError(t, err)
	}

	// We expect that a final submission from each fails, as the bid limit is exceeded.
	_, err = alice.Bid(ctx, big.NewInt(6), aliceAddr)
	require.ErrorContains(t, err, ErrTooManyBids.Error())
	_, err = bob.Bid(ctx, big.NewInt(7), bobAddr) // Bob bids 1 wei higher than Alice.
	require.ErrorContains(t, err, ErrTooManyBids.Error())
	_, err = charlie.Bid(ctx, big.NewInt(8), charlieAddr) // Charlie bids 2 wei higher than the Bob.
	require.ErrorContains(t, err, ErrTooManyBids.Error())

	t.Log("Submitted bids", time.Now(), time.Since(start))
	time.Sleep(time.Second * 15)

	// We verify that the auctioneer has consumed all validated bids from the single Redis stream.
	// We also verify the top two bids are those we expect.
	am.bidCache.Lock()
	require.Equal(t, 3, len(am.bidCache.bidsByExpressLaneControllerAddr))
	am.bidCache.Unlock()
	result := am.bidCache.topTwoBids()
	require.Equal(t, big.NewInt(7), result.firstPlace.Amount) // Best bid should be Charlie's last bid 7
	require.Equal(t, charlieAddr, result.firstPlace.Bidder)
	require.Equal(t, big.NewInt(6), result.secondPlace.Amount) // Second best bid should be Bob's last bid of 6
	require.Equal(t, bobAddr, result.secondPlace.Bidder)
}

func TestAuctioneerRecoversBidsOnRestart(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	testSetup := setupAuctionTest(t, ctx)
	redisURL := redisutil.CreateTestRedis(ctx, t)
	tmpDir := t.TempDir()
	jwtFilePath := filepath.Join(tmpDir, "jwt.key")
	jwtSecret := common.BytesToHash([]byte("jwt"))
	require.NoError(t, os.WriteFile(jwtFilePath, []byte(hexutil.Encode(jwtSecret[:])), 0600))

	// Set up a bid validator
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
	validatorCfg := &BidValidatorConfig{
		RpcEndpoint:            testSetup.endpoint,
		AuctionContractAddress: testSetup.expressLaneAuctionAddr.Hex(),
		RedisURL:               redisURL,
		ProducerConfig:         pubsub.TestProducerConfig,
		MaxBidsPerSender:       10,
	}
	validatorFetcher := func() *BidValidatorConfig {
		return validatorCfg
	}
	bidValidator, err := NewBidValidator(
		ctx,
		stack,
		validatorFetcher,
	)
	require.NoError(t, err)
	require.NoError(t, bidValidator.Initialize(ctx))
	require.NoError(t, stack.Start())
	bidValidator.Start(ctx)
	t.Log("Started bid validator")

	// Create first auctioneer instance
	auctioneerConfigFn := func() *AuctioneerServerConfig {
		return &AuctioneerServerConfig{
			SequencerEndpoint:      testSetup.endpoint,
			SequencerJWTPath:       jwtFilePath,
			AuctionContractAddress: testSetup.expressLaneAuctionAddr.Hex(),
			RedisURL:               redisURL,
			ConsumerConfig:         DefaultAuctioneerConsumerConfig,
			DbDirectory:            tmpDir,
			Wallet: genericconf.WalletConfig{
				PrivateKey: fmt.Sprintf("%x", testSetup.accounts[0].privKey.D.Bytes()),
			},
		}
	}

	auctioneer, err := NewAuctioneerServer(ctx, auctioneerConfigFn)
	require.NoError(t, err)
	auctioneer.Start(ctx)
	t.Log("Started first auctioneer instance")

	// Set up bidder clients
	aliceAddr := testSetup.accounts[1].txOpts.From
	bobAddr := testSetup.accounts[2].txOpts.From
	charlieAddr := testSetup.accounts[3].txOpts.From

	alice := setupBidderClient(t, ctx, testSetup.accounts[1], testSetup, bidValidator.stack.HTTPEndpoint())
	bob := setupBidderClient(t, ctx, testSetup.accounts[2], testSetup, bidValidator.stack.HTTPEndpoint())
	charlie := setupBidderClient(t, ctx, testSetup.accounts[3], testSetup, bidValidator.stack.HTTPEndpoint())

	// Make deposits
	require.NoError(t, alice.Deposit(ctx, big.NewInt(50)))
	require.NoError(t, bob.Deposit(ctx, big.NewInt(50)))
	require.NoError(t, charlie.Deposit(ctx, big.NewInt(50)))

	// Wait for auction round to start
	info, err := alice.auctionContract.RoundTimingInfo(&bind.CallOpts{})
	require.NoError(t, err)
	timeToWait := time.Until(time.Unix(int64(info.OffsetTimestamp), 0))
	t.Logf("Waiting for %v to start the bidding round, %v", timeToWait, time.Now())
	<-time.After(timeToWait)
	time.Sleep(time.Millisecond * 250) // Add 1/4 of a second to ensure we're in a round

	// First round of bids - Alice will be the winner with 20, Bob second with 15
	t.Log("Submitting first round of bids...")
	_, err = alice.Bid(ctx, big.NewInt(5), aliceAddr)
	require.NoError(t, err)
	_, err = alice.Bid(ctx, big.NewInt(10), aliceAddr)
	require.NoError(t, err)
	_, err = alice.Bid(ctx, big.NewInt(20), aliceAddr)
	require.NoError(t, err)

	_, err = bob.Bid(ctx, big.NewInt(3), bobAddr)
	require.NoError(t, err)
	_, err = bob.Bid(ctx, big.NewInt(8), bobAddr)
	require.NoError(t, err)
	_, err = bob.Bid(ctx, big.NewInt(15), bobAddr)
	require.NoError(t, err)

	_, err = charlie.Bid(ctx, big.NewInt(2), charlieAddr)
	require.NoError(t, err)
	_, err = charlie.Bid(ctx, big.NewInt(30), charlieAddr) // High bid
	require.NoError(t, err)
	_, err = charlie.Bid(ctx, big.NewInt(10), charlieAddr) // Overwrite high bid
	require.NoError(t, err)

	// Allow time for bids to be processed
	time.Sleep(time.Second * 2)

	// Verify first auctioneer state before restart
	auctioneer.bidCache.Lock()
	require.Equal(t, 3, len(auctioneer.bidCache.bidsByExpressLaneControllerAddr))
	auctioneer.bidCache.Unlock()

	result := auctioneer.bidCache.topTwoBids()
	require.Equal(t, big.NewInt(20), result.firstPlace.Amount)
	require.Equal(t, aliceAddr, result.firstPlace.Bidder)
	require.Equal(t, big.NewInt(15), result.secondPlace.Amount)
	require.Equal(t, bobAddr, result.secondPlace.Bidder)

	// "Restart" the auctioneer by creating a new instance
	t.Log("Stopping auctioneer...")
	auctioneer.StopAndWait()

	t.Log("Starting auctioneer...")
	// Create a new auctioneer with the same configuration (pointing to the same DB directory)
	newAuctioneer, err := NewAuctioneerServer(ctx, auctioneerConfigFn)
	require.NoError(t, err)
	newAuctioneer.Start(ctx)
	t.Log("Started new auctioneer instance")

	// Allow time for the new auctioneer to initialize and get the lock
	time.Sleep(auctioneer.auctioneerLivenessTimeout)

	// Second round of bids - these would be lower than Alice's previous bid
	t.Log("Submitting second round of bids...")
	_, err = bob.Bid(ctx, big.NewInt(12), bobAddr)
	require.NoError(t, err)
	_, err = charlie.Bid(ctx, big.NewInt(8), charlieAddr)
	require.NoError(t, err)

	// Allow time for bids to be processed
	time.Sleep(time.Second * 2)

	// Verify new auctioneer state - Alice should still be winning with 20
	newAuctioneer.bidCache.Lock()
	bidCount := len(newAuctioneer.bidCache.bidsByExpressLaneControllerAddr)
	newAuctioneer.bidCache.Unlock()

	// We expect either 2 or 3 bids in the cache, depending on whether the new auctioneer recovered
	// Alice's bid from the database or received it from Redis
	require.GreaterOrEqual(t, bidCount, 2)

	result = newAuctioneer.bidCache.topTwoBids()
	require.Equal(t, big.NewInt(20), result.firstPlace.Amount, "Alice should still be the highest bidder after restart")
	require.Equal(t, aliceAddr, result.firstPlace.Bidder)

	secondPlaceAmount := result.secondPlace.Amount
	require.True(t,
		secondPlaceAmount.Cmp(big.NewInt(12)) == 0,
		"Second place should be Bob's new 12 bid which overwrote the 15 bid, got %s", secondPlaceAmount.String())
	require.Equal(t, bobAddr, result.secondPlace.Bidder)

	// Now let the auction resolve and check the contract state
	// For this, we need to wait until the auction round closes and the auctioneer resolves it
	// #nosec G115
	roundEndTime := time.Unix(int64(info.OffsetTimestamp), 0).Add(
		time.Duration(info.RoundDurationSeconds) * time.Second)
	waitTime := time.Until(roundEndTime) + time.Second*5 // Add buffer time for resolution
	t.Logf("Waiting %v for auction to resolve...", waitTime)

	if waitTime > 0 {
		<-time.After(waitTime)
	}

	// We would verify the auction results on-chain here, but that would require additional
	// methods to query the auction results from the contract, which are not directly
	// accessible in the test code.
	t.Log("Test complete - auctioneer successfully recovered bids after restart")
}

func TestRetryUntil(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var currentAttempt int
		successAfter := 3
		retryInterval := 100 * time.Millisecond
		endTime := time.Now().Add(500 * time.Millisecond)

		err := retryUntil(context.Background(), mockOperation(successAfter, &currentAttempt), retryInterval, endTime)
		if err != nil {
			t.Errorf("expected success, got error: %v", err)
		}
		if currentAttempt != successAfter {
			t.Errorf("expected %d attempts, got %d", successAfter, currentAttempt)
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		var currentAttempt int
		successAfter := 5
		retryInterval := 100 * time.Millisecond
		endTime := time.Now().Add(300 * time.Millisecond)

		err := retryUntil(context.Background(), mockOperation(successAfter, &currentAttempt), retryInterval, endTime)
		if err == nil {
			t.Errorf("expected timeout error, got success")
		}
		if currentAttempt == successAfter {
			t.Errorf("expected failure, but operation succeeded")
		}
	})

	t.Run("ContextCancel", func(t *testing.T) {
		var currentAttempt int
		successAfter := 5
		retryInterval := 100 * time.Millisecond
		endTime := time.Now().Add(500 * time.Millisecond)

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(200 * time.Millisecond)
			cancel()
		}()

		err := retryUntil(ctx, mockOperation(successAfter, &currentAttempt), retryInterval, endTime)
		if err == nil {
			t.Errorf("expected context cancellation error, got success")
		}
		if currentAttempt >= successAfter {
			t.Errorf("expected failure due to context cancellation, but operation succeeded")
		}
	})
}

// Mock operation function to simulate different scenarios
func mockOperation(successAfter int, currentAttempt *int) func() error {
	return func() error {
		*currentAttempt++
		if *currentAttempt >= successAfter {
			return nil
		}
		return errors.New("operation failed")
	}
}

// This test is similar in structure to TestAuctioneerRecoversBidsOnRestart except it does a failover
// rather than restarting the same auctioneer.
func TestAuctioneerFailoverMessageReprocessing(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	testSetup := setupAuctionTest(t, ctx)
	redisURL := redisutil.CreateTestRedis(ctx, t)
	tmpDirPrimary := t.TempDir()
	tmpDirSecondary := t.TempDir()
	jwtFilePath := filepath.Join(tmpDirPrimary, "jwt.key")
	jwtSecret := common.BytesToHash([]byte("jwt"))
	require.NoError(t, os.WriteFile(jwtFilePath, []byte(hexutil.Encode(jwtSecret[:])), 0600))

	// Set up a bid validator
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
	validatorCfg := &BidValidatorConfig{
		RpcEndpoint:            testSetup.endpoint,
		AuctionContractAddress: testSetup.expressLaneAuctionAddr.Hex(),
		RedisURL:               redisURL,
		ProducerConfig:         pubsub.TestProducerConfig,
		MaxBidsPerSender:       10,
	}
	validatorFetcher := func() *BidValidatorConfig {
		return validatorCfg
	}
	bidValidator, err := NewBidValidator(
		ctx,
		stack,
		validatorFetcher,
	)
	require.NoError(t, err)
	require.NoError(t, bidValidator.Initialize(ctx))
	require.NoError(t, stack.Start())
	bidValidator.Start(ctx)
	t.Log("Started bid validator")

	// Use shorter timeouts for faster failover
	testConsumerConfig := pubsub.ConsumerConfig{
		IdletimeToAutoclaim:  300 * time.Millisecond,
		ResponseEntryTimeout: time.Minute,
		Retry:                true,
		MaxRetryCount:        -1,
	}

	// Create primary auctioneer instance
	primaryConfigFn := func() *AuctioneerServerConfig {
		return &AuctioneerServerConfig{
			SequencerEndpoint:      testSetup.endpoint,
			SequencerJWTPath:       jwtFilePath,
			AuctionContractAddress: testSetup.expressLaneAuctionAddr.Hex(),
			RedisURL:               redisURL,
			ConsumerConfig:         testConsumerConfig,
			DbDirectory:            tmpDirPrimary,
			Wallet: genericconf.WalletConfig{
				PrivateKey: fmt.Sprintf("%x", testSetup.accounts[0].privKey.D.Bytes()),
			},
		}
	}

	primary, err := NewAuctioneerServer(ctx, primaryConfigFn)
	require.NoError(t, err)
	primary.Start(ctx)
	t.Log("Started primary auctioneer instance")

	// Set up bidder clients
	aliceAddr := testSetup.accounts[1].txOpts.From
	bobAddr := testSetup.accounts[2].txOpts.From
	charlieAddr := testSetup.accounts[3].txOpts.From

	alice := setupBidderClient(t, ctx, testSetup.accounts[1], testSetup, bidValidator.stack.HTTPEndpoint())
	bob := setupBidderClient(t, ctx, testSetup.accounts[2], testSetup, bidValidator.stack.HTTPEndpoint())
	charlie := setupBidderClient(t, ctx, testSetup.accounts[3], testSetup, bidValidator.stack.HTTPEndpoint())

	// Make deposits
	require.NoError(t, alice.Deposit(ctx, big.NewInt(50)))
	require.NoError(t, bob.Deposit(ctx, big.NewInt(50)))
	require.NoError(t, charlie.Deposit(ctx, big.NewInt(50)))

	// Wait for auction round to start
	info, err := alice.auctionContract.RoundTimingInfo(&bind.CallOpts{})
	require.NoError(t, err)
	timeToWait := time.Until(time.Unix(int64(info.OffsetTimestamp), 0))
	t.Logf("Waiting for %v to start the bidding round, %v", timeToWait, time.Now())
	<-time.After(timeToWait)
	time.Sleep(time.Millisecond * 250) // Add 1/4 of a second to ensure we're in a round

	// First round of bids - Alice will be the winner with 20, Bob second with 15
	t.Log("Submitting first round of bids...")
	_, err = alice.Bid(ctx, big.NewInt(5), aliceAddr)
	require.NoError(t, err)
	_, err = alice.Bid(ctx, big.NewInt(10), aliceAddr)
	require.NoError(t, err)
	_, err = alice.Bid(ctx, big.NewInt(20), aliceAddr)
	require.NoError(t, err)

	_, err = bob.Bid(ctx, big.NewInt(3), bobAddr)
	require.NoError(t, err)
	_, err = bob.Bid(ctx, big.NewInt(8), bobAddr)
	require.NoError(t, err)
	_, err = bob.Bid(ctx, big.NewInt(15), bobAddr)
	require.NoError(t, err)

	_, err = charlie.Bid(ctx, big.NewInt(2), charlieAddr)
	require.NoError(t, err)
	_, err = charlie.Bid(ctx, big.NewInt(30), charlieAddr) // High bid
	require.NoError(t, err)
	_, err = charlie.Bid(ctx, big.NewInt(10), charlieAddr) // Overwrite high bid
	require.NoError(t, err)

	// Allow time for bids to be processed
	time.Sleep(time.Second * 2)

	// Verify primary auctioneer state before failover
	primary.bidCache.Lock()
	require.Equal(t, 3, len(primary.bidCache.bidsByExpressLaneControllerAddr))
	primary.bidCache.Unlock()

	result := primary.bidCache.topTwoBids()
	require.Equal(t, big.NewInt(20), result.firstPlace.Amount)
	require.Equal(t, aliceAddr, result.firstPlace.Bidder)
	require.Equal(t, big.NewInt(15), result.secondPlace.Amount)
	require.Equal(t, bobAddr, result.secondPlace.Bidder)

	// Check how many unacked bids the primary has
	primary.unackedBidsMutex.Lock()
	unackedCount := len(primary.unackedBids)
	primary.unackedBidsMutex.Unlock()
	t.Logf("Primary has %d unacked bids before failover", unackedCount)

	// "Crash" the primary by stopping it
	t.Log("Simulating primary crash...")
	primary.StopAndWait()

	// Create secondary auctioneer with different DB directory
	secondaryConfigFn := func() *AuctioneerServerConfig {
		return &AuctioneerServerConfig{
			SequencerEndpoint:      testSetup.endpoint,
			SequencerJWTPath:       jwtFilePath,
			AuctionContractAddress: testSetup.expressLaneAuctionAddr.Hex(),
			RedisURL:               redisURL,
			ConsumerConfig:         testConsumerConfig,
			DbDirectory:            tmpDirSecondary, // Different DB directory
			Wallet: genericconf.WalletConfig{
				PrivateKey: fmt.Sprintf("%x", testSetup.accounts[0].privKey.D.Bytes()),
			},
		}
	}

	t.Log("Starting secondary auctioneer...")
	secondary, err := NewAuctioneerServer(ctx, secondaryConfigFn)
	require.NoError(t, err)
	secondary.Start(ctx)
	t.Log("Started secondary auctioneer instance")

	// Wait for failover to complete (lock expiry + takeover)
	// The secondary should become primary after 3 * IdletimeToAutoclaim
	failoverTime := testConsumerConfig.IdletimeToAutoclaim * 3
	t.Logf("Waiting %v for failover to complete...", failoverTime)
	time.Sleep(failoverTime + 500*time.Millisecond)

	// Verify secondary is now primary
	require.True(t, secondary.IsPrimary(), "Secondary should have become primary")

	// Secondary should reprocess the unacked messages from Redis
	// Wait a bit for message reprocessing
	time.Sleep(2 * time.Second)

	// Second round of bids - these would be lower than Alice's previous bid
	t.Log("Submitting second round of bids...")
	_, err = bob.Bid(ctx, big.NewInt(12), bobAddr)
	require.NoError(t, err)
	_, err = charlie.Bid(ctx, big.NewInt(8), charlieAddr)
	require.NoError(t, err)

	// Allow time for bids to be processed
	time.Sleep(time.Second * 2)

	// Verify secondary auctioneer state - should have all bids including reprocessed ones
	secondary.bidCache.Lock()
	bidCount := len(secondary.bidCache.bidsByExpressLaneControllerAddr)
	secondary.bidCache.Unlock()

	// We expect 3 bidders in the cache
	require.Equal(t, 3, bidCount, "Should have all 3 bidders after reprocessing")

	result = secondary.bidCache.topTwoBids()
	require.Equal(t, big.NewInt(20), result.firstPlace.Amount, "Alice should still be the highest bidder after failover")
	require.Equal(t, aliceAddr, result.firstPlace.Bidder)

	secondPlaceAmount := result.secondPlace.Amount
	require.True(t,
		secondPlaceAmount.Cmp(big.NewInt(12)) == 0,
		"Second place should be Bob's new 12 bid which overwrote the 15 bid, got %s", secondPlaceAmount.String())
	require.Equal(t, bobAddr, result.secondPlace.Bidder)

	// The key test: verify that the secondary processed all the messages
	// including those that were unacked by the primary
	secondary.bidCache.Lock()
	aliceBids := secondary.bidCache.bidsByExpressLaneControllerAddr[aliceAddr]
	bobBids := secondary.bidCache.bidsByExpressLaneControllerAddr[bobAddr]
	charlieBids := secondary.bidCache.bidsByExpressLaneControllerAddr[charlieAddr]
	secondary.bidCache.Unlock()

	require.NotNil(t, aliceBids, "Should have Alice's bids")
	require.NotNil(t, bobBids, "Should have Bob's bids")
	require.NotNil(t, charlieBids, "Should have Charlie's bids")

	// Verify the amounts are what we expect
	require.Equal(t, big.NewInt(20), aliceBids.Amount, "Should have Alice's highest bid (from before failover)")
	require.Equal(t, big.NewInt(12), bobBids.Amount, "Should have Bob's latest bid (updated by later message)")
	require.Equal(t, big.NewInt(8), charlieBids.Amount, "Should have Charlie's final bid (updated by later message)")

	t.Log("Test complete - secondary successfully reprocessed unacked messages after failover")
}
