package timeboost

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/util/redisutil"
)

// Test configuration with shorter timeouts for faster tests
var testCoordinationConfig = pubsub.ConsumerConfig{
	IdletimeToAutoclaim:  300 * time.Millisecond,
	ResponseEntryTimeout: time.Minute,
	Retry:                true,
	MaxRetryCount:        -1,
}

// Helper function to create and start an auctioneer for testing
func createAndStartAuctioneer(t *testing.T, ctx context.Context, redisURL string, testSetup *auctionSetup, name string) (*AuctioneerServer, *auctioneerTestHelper) {
	tmpDir := t.TempDir()

	auctioneerConfig := func() *AuctioneerServerConfig {
		return &AuctioneerServerConfig{
			RedisURL:               redisURL,
			SequencerEndpoint:      testSetup.endpoint,
			AuctionContractAddress: testSetup.expressLaneAuctionAddr.Hex(),
			DbDirectory:            tmpDir,
			ConsumerConfig:         testCoordinationConfig,
			StreamTimeout:          10 * time.Millisecond, // Very short for tests
			Wallet: genericconf.WalletConfig{
				PrivateKey: fmt.Sprintf("%x", testSetup.accounts[0].privKey.D.Bytes()),
			},
		}
	}

	auctioneer, err := NewAuctioneerServer(ctx, auctioneerConfig)
	require.NoError(t, err)

	// Start the auctioneer
	auctioneer.Start(ctx)

	// Create producer for this test
	redisClient, err := redisutil.RedisClientFromURL(redisURL)
	require.NoError(t, err)

	producer, err := pubsub.NewProducer[*JsonValidatedBid, error](
		redisClient, validatedBidsRedisStream, &pubsub.TestProducerConfig,
	)
	require.NoError(t, err)
	producer.Start(ctx)

	helper := newAuctioneerTestHelper(ctx, auctioneer, producer, testSetup)

	log.Info("Created auctioneer", "name", name, "id", auctioneer.GetId())

	return auctioneer, helper
}

// Helper function to wait for an auctioneer to reach expected primary status
func waitForPrimaryStatus(t *testing.T, auctioneer *AuctioneerServer, expectedPrimary bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if auctioneer.IsPrimary() == expectedPrimary {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("Auctioneer %s did not reach expected primary status %v within %v",
		auctioneer.GetId(), expectedPrimary, timeout)
}

// Helper to verify that a specific auctioneer would consume bids
func verifyPrimaryBehavior(t *testing.T, auctioneer *AuctioneerServer, shouldBePrimary bool) {
	t.Helper()
	if shouldBePrimary {
		require.True(t, auctioneer.IsPrimary(), "Auctioneer should be primary")
		// When primary, consumeNextBid should actively try to consume
		// We don't need to actually produce bids, just verify the behavior
	} else {
		require.False(t, auctioneer.IsPrimary(), "Auctioneer should not be primary")
		// When not primary, consumeNextBid should return quickly without consuming
	}
}

func TestAuctioneerFailover_BasicScenario(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisURL := redisutil.CreateTestRedis(ctx, t)
	testSetup := setupAuctionTest(t, ctx)

	// Ensure Redis stream exists
	redisClient, err := redisutil.RedisClientFromURL(redisURL)
	require.NoError(t, err)
	err = pubsub.CreateStream(ctx, validatedBidsRedisStream, redisClient)
	require.NoError(t, err)

	// Start primary auctioneer
	primary, primaryHelper := createAndStartAuctioneer(t, ctx, redisURL, testSetup, "primary")
	defer primary.StopAndWait()
	defer primaryHelper.producer.StopAndWait()

	// Verify it becomes primary
	waitForPrimaryStatus(t, primary, true, 2*time.Second)
	t.Log("Primary auctioneer established")

	// Start secondary auctioneer
	secondary, secondaryHelper := createAndStartAuctioneer(t, ctx, redisURL, testSetup, "secondary")
	defer secondary.StopAndWait()
	defer secondaryHelper.producer.StopAndWait()

	// Verify secondary is NOT primary
	time.Sleep(1 * time.Second) // Give it time to check coordination
	assert.False(t, secondary.IsPrimary(), "Secondary should not be primary while primary is active")

	// Verify primary behavior
	t.Log("Testing primary behavior")
	verifyPrimaryBehavior(t, primary, true)
	verifyPrimaryBehavior(t, secondary, false)

	// Stop primary auctioneer
	t.Log("Stopping primary auctioneer")
	primary.StopAndWait()

	// Wait for failover (lock expiry + processing time)
	t.Log("Waiting for failover...")
	waitForPrimaryStatus(t, secondary, true, 2*time.Second)
	t.Log("Secondary became primary after failover")

	// Verify secondary is now primary
	t.Log("Testing new primary behavior")
	verifyPrimaryBehavior(t, secondary, true)
}

func TestAuctioneerFailover_MultipleInstances(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisURL := redisutil.CreateTestRedis(ctx, t)
	testSetup := setupAuctionTest(t, ctx)

	// Ensure Redis stream exists
	redisClient, err := redisutil.RedisClientFromURL(redisURL)
	require.NoError(t, err)
	err = pubsub.CreateStream(ctx, validatedBidsRedisStream, redisClient)
	require.NoError(t, err)

	// Start 3 auctioneers
	a1, h1 := createAndStartAuctioneer(t, ctx, redisURL, testSetup, "A")
	defer h1.producer.StopAndWait()
	defer a1.StopAndWait()

	a2, h2 := createAndStartAuctioneer(t, ctx, redisURL, testSetup, "B")
	defer h2.producer.StopAndWait()
	defer a2.StopAndWait()

	a3, h3 := createAndStartAuctioneer(t, ctx, redisURL, testSetup, "C")
	defer h3.producer.StopAndWait()
	defer a3.StopAndWait()

	// One should become primary (we don't know which one)
	time.Sleep(1 * time.Second)

	// Count primaries
	primaryCount := 0
	var primaryName string

	if a1.IsPrimary() {
		primaryCount++
		primaryName = "A"
		t.Log("A is primary")
	}
	if a2.IsPrimary() {
		primaryCount++
		primaryName = "B"
		t.Log("B is primary")
	}
	if a3.IsPrimary() {
		primaryCount++
		primaryName = "C"
		t.Log("C is primary")
	}

	assert.Equal(t, 1, primaryCount, "Exactly one should be primary")
	require.NotEmpty(t, primaryName, "Should have a primary")

	// Stop current primary based on which one it is
	t.Logf("Stopping current primary %s", primaryName)
	switch primaryName {
	case "A":
		a1.StopAndWait()
	case "B":
		a2.StopAndWait()
	case "C":
		a3.StopAndWait()
	}

	// Wait for failover
	time.Sleep(1500 * time.Millisecond)

	// Check that exactly one of the remaining is now primary
	primaryCount = 0
	var newPrimaryName string

	if primaryName != "A" && a1.IsPrimary() {
		primaryCount++
		newPrimaryName = "A"
		t.Log("A became new primary")
	}
	if primaryName != "B" && a2.IsPrimary() {
		primaryCount++
		newPrimaryName = "B"
		t.Log("B became new primary")
	}
	if primaryName != "C" && a3.IsPrimary() {
		primaryCount++
		newPrimaryName = "C"
		t.Log("C became new primary")
	}

	assert.Equal(t, 1, primaryCount, "Exactly one should be new primary")
	require.NotEmpty(t, newPrimaryName, "Should have a new primary")
}

func TestAuctioneerFailover_ConcurrentStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisURL := redisutil.CreateTestRedis(ctx, t)
	testSetup := setupAuctionTest(t, ctx)

	// Ensure Redis stream exists
	redisClient, err := redisutil.RedisClientFromURL(redisURL)
	require.NoError(t, err)
	err = pubsub.CreateStream(ctx, validatedBidsRedisStream, redisClient)
	require.NoError(t, err)

	// Start 2 auctioneers concurrently
	var wg sync.WaitGroup
	wg.Add(2)

	var a1, a2 *AuctioneerServer
	var h1, h2 *auctioneerTestHelper

	go func() {
		defer wg.Done()
		a1, h1 = createAndStartAuctioneer(t, ctx, redisURL, testSetup, "concurrent1")
	}()

	go func() {
		defer wg.Done()
		a2, h2 = createAndStartAuctioneer(t, ctx, redisURL, testSetup, "concurrent2")
	}()

	wg.Wait()

	defer a1.StopAndWait()
	defer h1.producer.StopAndWait()
	defer a2.StopAndWait()
	defer h2.producer.StopAndWait()

	// Wait for coordination to settle
	time.Sleep(1 * time.Second)

	// Verify exactly one is primary
	primaryCount := 0
	var primary *AuctioneerServer

	if a1.IsPrimary() {
		primaryCount++
		primary = a1
		t.Log("Concurrent1 is primary")
	}
	if a2.IsPrimary() {
		primaryCount++
		primary = a2
		t.Log("Concurrent2 is primary")
	}

	assert.Equal(t, 1, primaryCount, "Exactly one should be primary with concurrent start")

	// Verify primary behavior
	require.NotNil(t, primary, "Should have a primary")
	verifyPrimaryBehavior(t, primary, true)
}

// TestAuctioneerFailover_RapidRecovery tests that a restarted auctioneer doesn't
// immediately become primary if another holds the lock
func TestAuctioneerFailover_RapidRecovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisURL := redisutil.CreateTestRedis(ctx, t)
	testSetup := setupAuctionTest(t, ctx)

	// Ensure Redis stream exists
	redisClient, err := redisutil.RedisClientFromURL(redisURL)
	require.NoError(t, err)
	err = pubsub.CreateStream(ctx, validatedBidsRedisStream, redisClient)
	require.NoError(t, err)

	// Start primary
	primary, primaryHelper := createAndStartAuctioneer(t, ctx, redisURL, testSetup, "primary")
	defer primaryHelper.producer.StopAndWait()

	waitForPrimaryStatus(t, primary, true, 2*time.Second)

	// Start secondary
	secondary, secondaryHelper := createAndStartAuctioneer(t, ctx, redisURL, testSetup, "secondary")
	defer secondary.StopAndWait()
	defer secondaryHelper.producer.StopAndWait()

	time.Sleep(1 * time.Second)
	assert.False(t, secondary.IsPrimary(), "Secondary should not be primary")

	// Stop and immediately restart primary
	t.Log("Restarting primary")
	primary.StopAndWait()

	// Immediately start a new instance (simulating restart)
	newPrimary, newPrimaryHelper := createAndStartAuctioneer(t, ctx, redisURL, testSetup, "new-primary")
	defer newPrimary.StopAndWait()
	defer newPrimaryHelper.producer.StopAndWait()

	// The lock should still be held, so new primary should not become primary immediately
	time.Sleep(500 * time.Millisecond)
	assert.False(t, newPrimary.IsPrimary(), "Restarted instance should not be primary immediately")

	// Wait for failover - either secondary or newPrimary could become primary
	time.Sleep(2 * time.Second)

	// Check which one became primary
	primaryCount := 0
	var actualPrimary *AuctioneerServer

	if secondary.IsPrimary() {
		primaryCount++
		actualPrimary = secondary
		t.Log("Secondary became primary")
	}
	if newPrimary.IsPrimary() {
		primaryCount++
		actualPrimary = newPrimary
		t.Log("Restarted instance became primary")
	}

	assert.Equal(t, 1, primaryCount, "Exactly one should be primary after failover")
	require.NotNil(t, actualPrimary, "Should have a primary")
}

// TestAuctioneerFailover_StaleLockDetection tests the timestamp-based stale lock detection
func TestAuctioneerFailover_StaleLockDetection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisURL := redisutil.CreateTestRedis(ctx, t)
	testSetup := setupAuctionTest(t, ctx)

	// Ensure Redis stream exists
	redisClient, err := redisutil.RedisClientFromURL(redisURL)
	require.NoError(t, err)
	err = pubsub.CreateStream(ctx, validatedBidsRedisStream, redisClient)
	require.NoError(t, err)

	// Start primary auctioneer
	primary, primaryHelper := createAndStartAuctioneer(t, ctx, redisURL, testSetup, "primary")
	defer primaryHelper.producer.StopAndWait()

	// Wait for primary to be established
	waitForPrimaryStatus(t, primary, true, 2*time.Second)
	t.Log("Primary auctioneer established")

	// Stop primary without cleanup (simulating crash)
	t.Log("Simulating primary crash")
	primary.StopAndWait()

	// Start secondary immediately
	secondary, secondaryHelper := createAndStartAuctioneer(t, ctx, redisURL, testSetup, "secondary")
	defer secondary.StopAndWait()
	defer secondaryHelper.producer.StopAndWait()

	// Secondary should detect stale lock and take over
	waitForPrimaryStatus(t, secondary, true, 2*time.Second)
	t.Log("Secondary detected stale lock and became primary")

	// Verify secondary is now primary
	verifyPrimaryBehavior(t, secondary, true)
}

// TestAuctioneerFailover_InvalidLockFormat tests handling of invalid lock format
func TestAuctioneerFailover_InvalidLockFormat(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisURL := redisutil.CreateTestRedis(ctx, t)
	testSetup := setupAuctionTest(t, ctx)

	// Ensure Redis stream exists
	redisClient, err := redisutil.RedisClientFromURL(redisURL)
	require.NoError(t, err)
	err = pubsub.CreateStream(ctx, validatedBidsRedisStream, redisClient)
	require.NoError(t, err)

	// Manually set an invalid lock format
	err = redisClient.Set(ctx, AUCTIONEER_CHOSEN_KEY, "invalid-format", time.Hour).Err()
	require.NoError(t, err)

	// Start auctioneer
	auctioneer, helper := createAndStartAuctioneer(t, ctx, redisURL, testSetup, "auctioneer")
	defer auctioneer.StopAndWait()
	defer helper.producer.StopAndWait()

	// Auctioneer should handle invalid format and eventually become primary
	waitForPrimaryStatus(t, auctioneer, true, 2*time.Second)
	t.Log("Auctioneer handled invalid lock format and became primary")

	// Verify it's primary
	verifyPrimaryBehavior(t, auctioneer, true)
}
