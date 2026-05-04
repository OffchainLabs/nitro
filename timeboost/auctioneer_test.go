// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
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
		bidValidators[i], _ = setupBidValidator(t, ctx, redisURL, testSetup)
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
		StreamTimeout:          time.Minute,
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
	require.Equal(t, 3, am.bidCache.size())
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

	bidValidator, _ := setupBidValidator(t, ctx, redisURL, testSetup)
	t.Log("Started bid validator")

	// Create first auctioneer instance
	auctioneerConfigFn := func() *AuctioneerServerConfig {
		return &AuctioneerServerConfig{
			SequencerEndpoint:      testSetup.endpoint,
			SequencerJWTPath:       jwtFilePath,
			AuctionContractAddress: testSetup.expressLaneAuctionAddr.Hex(),
			RedisURL:               redisURL,
			ConsumerConfig:         DefaultAuctioneerConsumerConfig,
			StreamTimeout:          time.Minute,
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
	require.Equal(t, 3, auctioneer.bidCache.size())

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
	// We expect either 2 or 3 bids in the cache, depending on whether the new auctioneer recovered
	// Alice's bid from the database or received it from Redis
	require.GreaterOrEqual(t, newAuctioneer.bidCache.size(), 2)

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

	t.Run("DeadlineAlreadyPassed", func(t *testing.T) {
		var currentAttempt int
		successAfter := 1
		retryInterval := 100 * time.Millisecond
		endTime := time.Now().Add(-time.Second) // already in the past

		err := retryUntil(context.Background(), mockOperation(successAfter, &currentAttempt), retryInterval, endTime)
		require.Error(t, err)
		require.Contains(t, err.Error(), "operation not attempted")
		require.Equal(t, 0, currentAttempt, "operation should never have been called")
	})

	t.Run("ContextCancelPreservesLastError", func(t *testing.T) {
		retryInterval := 500 * time.Millisecond
		endTime := time.Now().Add(10 * time.Second)
		opErr := errors.New("specific RPC failure")

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(200 * time.Millisecond)
			cancel()
		}()

		start := time.Now()
		err := retryUntil(ctx, func() error { return opErr }, retryInterval, endTime)
		elapsed := time.Since(start)
		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)
		require.Contains(t, err.Error(), "specific RPC failure")
		// Must return promptly after cancellation, not wait for the full retry interval.
		require.Less(t, elapsed, 400*time.Millisecond, "retryUntil should return promptly on context cancellation")
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

func TestParseAuctioneerLockValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		value     string
		wantId    string
		wantTs    int64
		wantValid bool
	}{
		{"valid", "myid:1234567890", "myid", 1234567890, true},
		{"no colon", "nocolon", "", 0, false},
		{"empty", "", "", 0, false},
		{"unparseable timestamp", "id:notanumber", "", 0, false},
		{"empty timestamp", "id:", "", 0, false},
		{"float timestamp", "id:123.456", "", 0, false},
		{"negative timestamp", "id:-1000", "", 0, false},
		{"zero timestamp", "id:0", "", 0, false},
		{"colon in id", "my:id:123", "", 0, false},
		{"empty id", ":1234567890", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ts, ok := parseAuctioneerLockValue(tt.value)
			require.Equal(t, tt.wantValid, ok)
			if ok {
				require.Equal(t, tt.wantId, id)
				require.Equal(t, tt.wantTs, ts)
			}
		})
	}
}

func TestUpdateCoordination_MalformedRedisKey(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisURL := redisutil.CreateTestRedis(ctx, t)
	redisClient, err := redisutil.RedisClientFromURL(redisURL)
	require.NoError(t, err)

	livenessTimeout := 3 * time.Second
	a := &AuctioneerServer{
		redisClient:               redisClient,
		myId:                      "test-auctioneer",
		auctioneerLivenessTimeout: livenessTimeout,
	}

	expectedRetry := livenessTimeout / 6

	tests := []struct {
		name  string
		value string
	}{
		{"no colon separator", "nocolon"},
		{"empty value", ""},
		{"unparseable timestamp", "someid:notanumber"},
		{"timestamp with spaces", "someid: 123"},
		{"empty timestamp", "someid:"},
		{"float timestamp", "someid:123.456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := redisClient.Set(ctx, AUCTIONEER_CHOSEN_KEY, tt.value, 0).Err()
			require.NoError(t, err)

			result := a.updateCoordination(ctx)
			require.Equal(t, expectedRetry, result)

			// Key should be deleted after malformed value detected
			exists, err := redisClient.Exists(ctx, AUCTIONEER_CHOSEN_KEY).Result()
			require.NoError(t, err)
			require.Equal(t, int64(0), exists, "malformed key should be deleted")
		})
	}
}

func TestCoordinationInterval_BackoffOnRedisErrors(t *testing.T) {
	t.Parallel()
	livenessTimeout := 6 * time.Second
	a := &AuctioneerServer{
		auctioneerLivenessTimeout: livenessTimeout,
	}
	baseInterval := livenessTimeout / 6

	// No error: returns base interval and resets backoff.
	require.Equal(t, baseInterval, a.coordinationInterval(false))
	require.Equal(t, time.Duration(0), a.coordinationBackoff)

	// First error: returns base interval (minimum), sets backoff for next call.
	interval := a.coordinationInterval(true)
	require.Equal(t, baseInterval, interval)

	// Second error: backoff doubles.
	interval = a.coordinationInterval(true)
	require.Equal(t, baseInterval*2, interval)

	// Third error: doubles again.
	interval = a.coordinationInterval(true)
	require.Equal(t, baseInterval*4, interval)

	// Backoff is capped at livenessTimeout.
	for range 10 {
		interval = a.coordinationInterval(true)
	}
	require.LessOrEqual(t, interval, livenessTimeout)

	// Success resets backoff.
	interval = a.coordinationInterval(false)
	require.Equal(t, baseInterval, interval)
	require.Equal(t, time.Duration(0), a.coordinationBackoff)
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

	bidValidator, _ := setupBidValidator(t, ctx, redisURL, testSetup)
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
			StreamTimeout:          time.Minute,
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
	require.Equal(t, 3, primary.bidCache.size())

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
			StreamTimeout:          time.Minute,
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
	// We expect 3 bidders in the cache
	require.Equal(t, 3, secondary.bidCache.size(), "Should have all 3 bidders after reprocessing")

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
	aliceBids := secondary.bidCache.getBid(aliceAddr)
	bobBids := secondary.bidCache.getBid(bobAddr)
	charlieBids := secondary.bidCache.getBid(charlieAddr)

	require.NotNil(t, aliceBids, "Should have Alice's bids")
	require.NotNil(t, bobBids, "Should have Bob's bids")
	require.NotNil(t, charlieBids, "Should have Charlie's bids")

	// Verify the amounts are what we expect
	require.Equal(t, big.NewInt(20), aliceBids.Amount, "Should have Alice's highest bid (from before failover)")
	require.Equal(t, big.NewInt(12), bobBids.Amount, "Should have Bob's latest bid (updated by later message)")
	require.Equal(t, big.NewInt(8), charlieBids.Amount, "Should have Charlie's final bid (updated by later message)")

	t.Log("Test complete - secondary successfully reprocessed unacked messages after failover")
}

func TestPersistBidsChannelFull_BidStillAddedToCache(t *testing.T) {
	t.Parallel()
	cache := newBidCache([32]byte{})
	persistBids := make(chan *ValidatedBid, 1)

	// Fill the persistence channel.
	persistBids <- &ValidatedBid{
		Bidder:                common.HexToAddress("0x1"),
		Amount:                big.NewInt(1),
		ExpressLaneController: common.HexToAddress("0x1"),
		ChainId:               big.NewInt(1),
	}

	// Simulate the bid receiver logic: add to cache then non-blocking send to persist channel.
	bid := &ValidatedBid{
		Bidder:                common.HexToAddress("0xA"),
		Amount:                big.NewInt(100),
		ExpressLaneController: common.HexToAddress("0xA"),
		ChainId:               big.NewInt(1),
	}
	cache.add(bid)
	select {
	case persistBids <- bid:
		t.Fatal("expected persistence channel to be full")
	default:
		// Expected: channel full, bid not persisted but still in cache.
	}

	// Bid must still be in the cache for auction resolution despite persistence failure.
	require.Equal(t, 1, cache.size())
	result := cache.topTwoBids()
	require.NotNil(t, result.firstPlace)
	require.Equal(t, common.HexToAddress("0xA"), result.firstPlace.Bidder)
	require.Equal(t, 0, result.firstPlace.Amount.Cmp(big.NewInt(100)))
}

func TestAuctioneerServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name                     string
		auctionContractAddress   string
		reserveOriginatorAddress string
		s3Storage                S3StorageServiceConfig
		wantErr                  string
	}{
		{
			name:    "both addresses empty is valid",
			wantErr: "",
		},
		{
			name:                     "reserve originator zero address rejected",
			reserveOriginatorAddress: "0x0000000000000000000000000000000000000000",
			wantErr:                  "cannot be the zero address",
		},
		{
			name:                   "valid auction contract address only",
			auctionContractAddress: "0x1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:                     "valid reserve originator address only",
			reserveOriginatorAddress: "0xabcdef1234567890abcdef1234567890abcdef12",
		},
		{
			name:                     "both valid addresses",
			auctionContractAddress:   "0x1234567890abcdef1234567890abcdef12345678",
			reserveOriginatorAddress: "0xabcdef1234567890abcdef1234567890abcdef12",
		},
		{
			name:                   "invalid auction contract address",
			auctionContractAddress: "not-a-hex-address",
			wantErr:                "invalid auctioneer-server.auction-contract-address",
		},
		{
			name:                     "invalid reserve originator address",
			reserveOriginatorAddress: "not-a-hex-address",
			wantErr:                  "invalid auctioneer-server.reserve-originator-address",
		},
		{
			name:                     "valid auction contract but invalid reserve originator",
			auctionContractAddress:   "0x1234567890abcdef1234567890abcdef12345678",
			reserveOriginatorAddress: "xyz",
			wantErr:                  "invalid auctioneer-server.reserve-originator-address",
		},
		{
			name:                     "invalid auction contract checked first",
			auctionContractAddress:   "bad",
			reserveOriginatorAddress: "also-bad",
			wantErr:                  "invalid auctioneer-server.auction-contract-address",
		},
		{
			name: "s3 storage validation is delegated",
			s3Storage: S3StorageServiceConfig{
				Enable:       true,
				MaxBatchSize: -1,
			},
			wantErr: "s3-storage bucket cannot be empty when enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &AuctioneerServerConfig{
				AuctionContractAddress:   tt.auctionContractAddress,
				ReserveOriginatorAddress: tt.reserveOriginatorAddress,
				StreamTimeout:            time.Minute,
				S3Storage:                tt.s3Storage,
			}
			err := cfg.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestAuctioneerServerConfig_Validate_StreamTimeout(t *testing.T) {
	for _, tt := range []struct {
		name    string
		timeout time.Duration
	}{
		{"zero rejected", 0},
		{"negative rejected", -1 * time.Second},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &AuctioneerServerConfig{StreamTimeout: tt.timeout}
			require.ErrorContains(t, cfg.Validate(), "stream-timeout must be positive")
		})
	}
}
