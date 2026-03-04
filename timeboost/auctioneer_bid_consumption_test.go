// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package timeboost

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/redisutil"
)

type auctioneerTestHelper struct {
	auctioneer *AuctioneerServer
	producer   *pubsub.Producer[*JsonValidatedBid, error]
	testSetup  *auctionSetup
	ctx        context.Context
}

func newAuctioneerTestHelper(ctx context.Context, auctioneer *AuctioneerServer, producer *pubsub.Producer[*JsonValidatedBid, error], testSetup *auctionSetup) *auctioneerTestHelper {
	return &auctioneerTestHelper{
		auctioneer: auctioneer,
		producer:   producer,
		testSetup:  testSetup,
		ctx:        ctx,
	}
}

func (h *auctioneerTestHelper) resetState() {
	h.auctioneer.unackedBidsMutex.Lock()
	h.auctioneer.unackedBids = make(map[string]*pubsub.Message[*JsonValidatedBid])
	h.auctioneer.unackedBidsMutex.Unlock()

	h.auctioneer.isPrimary.Store(true)

	for {
		select {
		case <-h.auctioneer.bidsReceiver:
		default:
			return
		}
	}
}

func (h *auctioneerTestHelper) getUnackedBidsCount() int {
	h.auctioneer.unackedBidsMutex.Lock()
	defer h.auctioneer.unackedBidsMutex.Unlock()
	return len(h.auctioneer.unackedBids)
}

func (h *auctioneerTestHelper) getFirstUnackedMessageID() string {
	h.auctioneer.unackedBidsMutex.Lock()
	defer h.auctioneer.unackedBidsMutex.Unlock()

	for id := range h.auctioneer.unackedBids {
		return id
	}
	return ""
}

func (h *auctioneerTestHelper) waitForBid(timeout time.Duration) *JsonValidatedBid {
	select {
	case bid := <-h.auctioneer.bidsReceiver:
		return bid
	case <-time.After(timeout):
		return nil
	}
}

func (h *auctioneerTestHelper) assertNoBid(t *testing.T, timeout time.Duration) {
	select {
	case bid := <-h.auctioneer.bidsReceiver:
		t.Errorf("Unexpected bid received: %+v", bid)
	case <-time.After(timeout):
	}
}

func (h *auctioneerTestHelper) createValidBid(amount int64, account int) *JsonValidatedBid {
	nextRound := h.auctioneer.roundTimingInfo.RoundNumber() + 1
	return h.createBid(nextRound, amount, account)
}

func (h *auctioneerTestHelper) createPastRoundBid(amount int64, account int) *JsonValidatedBid {
	// The current round is a past round, for bidding purposes, avoids issue with zero underflow.
	pastRound := h.auctioneer.roundTimingInfo.RoundNumber()
	return h.createBid(pastRound, amount, account)
}

func (h *auctioneerTestHelper) createBid(round uint64, amount int64, account int) *JsonValidatedBid {
	bidder := h.testSetup.accounts[account].accountAddr
	return &JsonValidatedBid{
		ChainId:                (*hexutil.Big)(h.testSetup.chainId),
		Round:                  hexutil.Uint64(round),
		AuctionContractAddress: h.testSetup.expressLaneAuctionAddr,
		Bidder:                 bidder,
		ExpressLaneController:  bidder,
		Amount:                 (*hexutil.Big)(big.NewInt(amount)),
		Signature:              make([]byte, 65),
	}
}

func (h *auctioneerTestHelper) produceBid(bid *JsonValidatedBid) (*containers.Promise[error], error) {
	promise, err := h.producer.Produce(h.ctx, bid)
	if err != nil {
		return nil, err
	}
	time.Sleep(200 * time.Millisecond)
	return promise, nil
}

func (h *auctioneerTestHelper) consumeAndVerifyBid(t *testing.T, expectedBid *JsonValidatedBid) time.Duration {
	initialCount := h.getUnackedBidsCount()

	waitDuration := h.auctioneer.consumeNextBid(h.ctx)

	newCount := h.getUnackedBidsCount()
	assert.Equal(t, initialCount+1, newCount, "Bid should be added to unackedBids")

	receivedBid := h.waitForBid(time.Second)
	require.NotNil(t, receivedBid, "Bid should be forwarded to bidsReceiver")

	assert.Equal(t, expectedBid.Round, receivedBid.Round, "Round mismatch")
	assert.Equal(t, expectedBid.Bidder, receivedBid.Bidder, "Bidder mismatch")
	assert.Equal(t, expectedBid.Amount, receivedBid.Amount, "Amount mismatch")

	return waitDuration
}

func (h *auctioneerTestHelper) consumeAndVerifyRejectedBid(t *testing.T, promise *containers.Promise[error]) time.Duration {
	initialCount := h.getUnackedBidsCount()

	waitDuration := h.auctioneer.consumeNextBid(h.ctx)

	newCount := h.getUnackedBidsCount()
	assert.Equal(t, initialCount, newCount, "Invalid bid should not be added to unackedBids")

	h.assertNoBid(t, 100*time.Millisecond)

	_, err := promise.Current()
	require.NotNil(t, err, "Promise should have an error")
	assert.Contains(t, err.Error(), "BAD_ROUND_NUMBER", "Error should contain BAD_ROUND_NUMBER")

	return waitDuration
}

func (h *auctioneerTestHelper) acknowledgeAllBids() {
	nextRound := h.auctioneer.roundTimingInfo.RoundNumber() + 1
	h.auctioneer.acknowledgeAllBids(h.ctx, nextRound)
	time.Sleep(100 * time.Millisecond)
}

func TestConsumeNextBid_Direct(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisURL, testSetup, auctioneer := setupAuctioneerServer(t, ctx, pubsub.TestConsumerConfig, "")

	redisClient, err := redisutil.RedisClientFromURL(redisURL)
	require.NoError(t, err)
	err = pubsub.CreateStream(ctx, validatedBidsRedisStream, redisClient)
	require.NoError(t, err)

	auctioneer.consumer.Start(ctx)
	defer auctioneer.consumer.StopAndWait()

	producer, err := pubsub.NewProducer[*JsonValidatedBid, error](
		redisClient, validatedBidsRedisStream, &pubsub.TestProducerConfig,
	)
	require.NoError(t, err)
	producer.Start(ctx)
	defer producer.StopAndWait()

	helper := newAuctioneerTestHelper(ctx, auctioneer, producer, testSetup)

	t.Run("EmptyQueue", func(t *testing.T) {
		helper.resetState()

		waitDuration := auctioneer.consumeNextBid(ctx)
		assert.Equal(t, time.Millisecond*250, waitDuration, "Expected 250ms wait for empty queue")

		assert.Equal(t, 0, auctioneer.bidCache.size(), "Bid cache should be empty")
		assert.Equal(t, 0, helper.getUnackedBidsCount(), "Unacked bids should be empty")
	})

	t.Run("ValidBid", func(t *testing.T) {
		helper.resetState()

		validBid := helper.createValidBid(100, 1)

		promise, err := helper.produceBid(validBid)
		require.NoError(t, err)

		waitDuration := helper.consumeAndVerifyBid(t, validBid)
		assert.Equal(t, time.Duration(0), waitDuration, "Expected 0 wait for valid bid")

		helper.acknowledgeAllBids()

		result, err := promise.Await(ctx)
		require.Nil(t, err, "No error should be set in the promise")
		require.Nil(t, result, "Promise result should be nil for successful processing")
	})

	t.Run("InvalidBid_PastRound", func(t *testing.T) {
		helper.resetState()

		invalidBid := helper.createPastRoundBid(200, 2)

		promise, err := helper.produceBid(invalidBid)
		require.NoError(t, err)

		waitDuration := helper.consumeAndVerifyRejectedBid(t, promise)
		assert.Equal(t, time.Duration(0), waitDuration, "Expected 0 wait for invalid bid")
	})
}

func TestConsumeNextBid_DuplicateHandling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Configure a very short idle time to allow for fast reclaiming
	shortIdleTimeConfig := pubsub.TestConsumerConfig
	shortIdleTimeConfig.IdletimeToAutoclaim = 50 * time.Millisecond

	redisURL, testSetup, auctioneer := setupAuctioneerServer(t, ctx, shortIdleTimeConfig, "")

	auctioneer.consumer.Start(ctx)
	defer auctioneer.consumer.StopAndWait()

	redisClient, err := redisutil.RedisClientFromURL(redisURL)
	require.NoError(t, err)
	err = pubsub.CreateStream(ctx, validatedBidsRedisStream, redisClient)
	require.NoError(t, err)

	producer, err := pubsub.NewProducer[*JsonValidatedBid, error](
		redisClient, validatedBidsRedisStream, &pubsub.TestProducerConfig,
	)
	require.NoError(t, err)
	producer.Start(ctx)
	defer producer.StopAndWait()

	helper := newAuctioneerTestHelper(ctx, auctioneer, producer, testSetup)
	helper.resetState()

	validBid := helper.createValidBid(100, 1)

	_, err = helper.produceBid(validBid)
	require.NoError(t, err)

	waitDuration := auctioneer.consumeNextBid(ctx)
	assert.Equal(t, time.Duration(0), waitDuration)

	assert.Equal(t, 1, helper.getUnackedBidsCount(), "Message should be in unackedBids")
	receivedBid := helper.waitForBid(time.Second)
	require.NotNil(t, receivedBid, "Bid should be forwarded to bidsReceiver")

	msgID := helper.getFirstUnackedMessageID()
	require.NotEmpty(t, msgID, "Should have found a message ID")
	auctioneer.unackedBidsMutex.Lock()
	msg := auctioneer.unackedBids[msgID]
	auctioneer.unackedBidsMutex.Unlock()
	require.NotNil(t, msg, "Message should exist")

	// Call Ack() to stop the heartbeat, which will allow the message to be auto-claimed
	msg.Ack()
	// Wait for the IdletimeToAutoclaim period to pass so the message can be reclaimed
	time.Sleep(100 * time.Millisecond)

	// Clear the bidsReceiver channel to ensure we can detect if a message is forwarded again
	select {
	case <-auctioneer.bidsReceiver:
		// Drain any pending messages
	default:
		// No messages, which is good
	}

	// Try to consume again - the same message should be available for auto-claim
	waitDuration = auctioneer.consumeNextBid(ctx)
	assert.Equal(t, time.Duration(0), waitDuration)

	select {
	case bid := <-auctioneer.bidsReceiver:
		t.Errorf("Duplicate bid was incorrectly forwarded to bidsReceiver: %v", bid)
	case <-time.After(100 * time.Millisecond):
		// This is the expected behavior - no bid should be forwarded for duplicates
	}
}

func TestConsumeNextBid_MultipleValidBids(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisURL, testSetup, auctioneer := setupAuctioneerServer(t, ctx, pubsub.TestConsumerConfig, "")

	auctioneer.consumer.Start(ctx)
	defer auctioneer.consumer.StopAndWait()

	redisClient, err := redisutil.RedisClientFromURL(redisURL)
	require.NoError(t, err)
	err = pubsub.CreateStream(ctx, validatedBidsRedisStream, redisClient)
	require.NoError(t, err)

	producer, err := pubsub.NewProducer[*JsonValidatedBid, error](
		redisClient, validatedBidsRedisStream, &pubsub.TestProducerConfig,
	)
	require.NoError(t, err)
	producer.Start(ctx)
	defer producer.StopAndWait()

	helper := newAuctioneerTestHelper(ctx, auctioneer, producer, testSetup)
	helper.resetState()

	numBids := 5
	bids := make([]*JsonValidatedBid, numBids)
	promises := make([]*containers.Promise[error], numBids)

	for i := 0; i < numBids; i++ {
		bids[i] = helper.createValidBid(int64(100*(i+1)), i%len(testSetup.accounts))
		promise, err := helper.produceBid(bids[i])
		require.NoError(t, err)
		promises[i] = promise
	}

	for i := 0; i < numBids; i++ {
		waitDuration := auctioneer.consumeNextBid(ctx)
		assert.Equal(t, time.Duration(0), waitDuration, "Expected 0 wait for valid bid")

		receivedBid := helper.waitForBid(time.Second)
		require.NotNil(t, receivedBid, "Bid should be forwarded to bidsReceiver")

		assert.Equal(t, bids[i].Round, receivedBid.Round, "Round mismatch")
		assert.Equal(t, bids[i].Amount, receivedBid.Amount, "Amount mismatch")
	}

	assert.Equal(t, numBids, helper.getUnackedBidsCount(), "All bids should be in unackedBids")

	helper.acknowledgeAllBids()

	for _, promise := range promises {
		result, err := promise.Current()
		require.Nil(t, err, "Promise should be ready")
		require.Nil(t, result, "Promise should have nil result for successful processing")
	}

	assert.Equal(t, 0, helper.getUnackedBidsCount(), "unackedBids should be empty, all bids were acknowledged")
}

func setupAuctioneerServer(t *testing.T, ctx context.Context, consumerConfig pubsub.ConsumerConfig, bidFloorAgentAddr string) (string, *auctionSetup, *AuctioneerServer) {
	redisURL := redisutil.CreateTestRedis(ctx, t)

	tmpDir := t.TempDir()
	jwtFilePath := filepath.Join(tmpDir, "jwt.key")
	require.NoError(t, os.WriteFile(jwtFilePath, []byte(hexutil.Encode(common.BytesToHash([]byte("jwt")).Bytes())), 0600))

	testSetup := setupAuctionTest(t, ctx)

	auctioneerConfig := func() *AuctioneerServerConfig {
		return &AuctioneerServerConfig{
			RedisURL:               redisURL,
			SequencerEndpoint:      testSetup.endpoint,
			SequencerJWTPath:       jwtFilePath,
			AuctionContractAddress: testSetup.expressLaneAuctionAddr.Hex(),
			DbDirectory:            tmpDir,
			ConsumerConfig:         consumerConfig,
			BidFloorAgentAddress:   bidFloorAgentAddr,
			Wallet: genericconf.WalletConfig{
				PrivateKey: fmt.Sprintf("%x", testSetup.accounts[0].privKey.D.Bytes()),
			},
		}
	}

	auctioneer, err := NewAuctioneerServer(ctx, auctioneerConfig)
	require.NoError(t, err)
	return redisURL, testSetup, auctioneer
}

func TestResolveAuction_BidFloorAgent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Account index 1's address will be the BidFloorAgent
	testSetup := setupAuctionTest(t, ctx)
	bidFloorAddr := testSetup.accounts[1].accountAddr

	_, _, auctioneer := setupAuctioneerServer(
		t, ctx, pubsub.TestConsumerConfig, bidFloorAddr.Hex(),
	)

	t.Run("BidFloorAgent wins alone - skip resolution", func(t *testing.T) {
		// Only BidFloorAgent bid in cache
		auctioneer.bidCache = newBidCache(auctioneer.bidCache.auctionContractDomainSeparator)
		auctioneer.bidCache.add(&ValidatedBid{
			Bidder:                bidFloorAddr,
			ExpressLaneController: bidFloorAddr,
			Amount:                big.NewInt(100),
		})

		err := auctioneer.resolveAuction(ctx)
		require.NoError(t, err)
		// No on-chain call made, no error → BidFloorAgent path hit
	})

	t.Run("Spoofed ExpressLaneController does not skip resolution", func(t *testing.T) {
		auctioneer.bidCache = newBidCache(auctioneer.bidCache.auctionContractDomainSeparator)
		otherAddr := testSetup.accounts[3].accountAddr
		auctioneer.bidCache.add(&ValidatedBid{
			Bidder:                otherAddr,
			ExpressLaneController: bidFloorAddr,
			Amount:                big.NewInt(200),
		})

		err := auctioneer.resolveAuction(ctx)
		require.Error(t, err)
	})

	t.Run("BidFloorAgent wins with second place - skip resolution", func(t *testing.T) {
		auctioneer.bidCache = newBidCache(auctioneer.bidCache.auctionContractDomainSeparator)
		otherAddr := testSetup.accounts[2].accountAddr
		auctioneer.bidCache.add(&ValidatedBid{
			Bidder:                bidFloorAddr,
			ExpressLaneController: bidFloorAddr,
			Amount:                big.NewInt(200),
		})
		auctioneer.bidCache.add(&ValidatedBid{
			Bidder:                otherAddr,
			ExpressLaneController: otherAddr,
			Amount:                big.NewInt(100),
		})

		err := auctioneer.resolveAuction(ctx)
		require.NoError(t, err)
		// BidFloorAgent is first place → skips on-chain, returns nil
	})

	t.Run("BidFloorAgent loses - normal resolution attempted", func(t *testing.T) {
		auctioneer.bidCache = newBidCache(auctioneer.bidCache.auctionContractDomainSeparator)
		otherAddr := testSetup.accounts[2].accountAddr
		auctioneer.bidCache.add(&ValidatedBid{
			Bidder:                bidFloorAddr,
			ExpressLaneController: bidFloorAddr,
			Amount:                big.NewInt(50),
		})
		auctioneer.bidCache.add(&ValidatedBid{
			Bidder:                otherAddr,
			ExpressLaneController: otherAddr,
			Amount:                big.NewInt(200),
		})

		err := auctioneer.resolveAuction(ctx)
		// This will attempt on-chain resolution (may error in test env
		// since there's no real sequencer — the important thing is it
		// did NOT return nil early via the BidFloorAgent path)
		require.Error(t, err) // Expect error from sequencer RPC call
	})

	t.Run("No BidFloorAgent configured - normal resolution", func(t *testing.T) {
		// Create auctioneer WITHOUT BidFloorAgent
		_, _, auctioneerNoBFA := setupAuctioneerServer(
			t, ctx, pubsub.TestConsumerConfig, "",
		)
		otherAddr := testSetup.accounts[2].accountAddr
		auctioneerNoBFA.bidCache.add(&ValidatedBid{
			Bidder:                bidFloorAddr,
			ExpressLaneController: bidFloorAddr,
			Amount:                big.NewInt(200),
		})
		auctioneerNoBFA.bidCache.add(&ValidatedBid{
			Bidder:                otherAddr,
			ExpressLaneController: otherAddr,
			Amount:                big.NewInt(100),
		})

		err := auctioneerNoBFA.resolveAuction(ctx)
		// Without BidFloorAgent configured, even if same address wins,
		// it proceeds to on-chain resolution (which errors in test env)
		require.Error(t, err)
	})
}
