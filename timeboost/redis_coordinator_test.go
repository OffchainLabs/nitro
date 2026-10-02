// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package timeboost

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/util/redisutil"
)

func TestTimeboostRedisCoordinator(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// RedisCoordinator applies updates from its own goroutines, so every read-back below
	// polls instead of sleeping a fixed amount: on a loaded CI runner those goroutines can
	// be descheduled for far longer than any sleep we would pick.
	const (
		waitFor = 10 * time.Second
		tick    = 5 * time.Millisecond
	)

	redisUrl := redisutil.CreateTestRedis(ctx, t)
	// Round is deliberately far longer than the test can run: both update loops drop
	// updates for rounds older than RoundNumber(), so with a short round every write under
	// test would silently vanish if the test ever ran past a round boundary.
	timingInfo := &RoundTimingInfo{
		Offset: time.Now(),
		Round:  time.Hour,
	}
	redisCoordinator, err := NewRedisCoordinator(redisUrl, timingInfo, 50)
	require.NoError(t, err, "error initializing redis coordinator")
	redisCoordinator.Start(ctx)
	// Stop the update goroutines before the deferred cancel() tears down redis.
	defer redisCoordinator.StopAndWait()

	// Verify adding and retrieving global sequence count of a round.
	// The check function runs on a testify-owned goroutine, so it must report through
	// assert.CollectT rather than calling require/t.Fatal.
	requireSeqCount := func(round, expected uint64) {
		t.Helper()
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			globalSeq, err := redisCoordinator.GetSequenceCount(round)
			assert.NoError(c, err)
			assert.Equal(c, expected, globalSeq)
		}, waitFor, tick, "round %d sequence count never settled at %d", round, expected)
	}

	var round uint64
	require.NoError(t, redisCoordinator.UpdateSequenceCount(round, 3)) // should succeed
	requireSeqCount(round, 3)

	// A lower sequence count must not overwrite the stored one. trackSequenceCountUpdates
	// drains its update channel from a single goroutine, so once round 1's count is visible
	// in redis the stale round 0 update queued ahead of it has necessarily already been
	// handled - the round 0 read-back below therefore needs no wait of its own.
	require.NoError(t, redisCoordinator.UpdateSequenceCount(round, 1)) // shouldn't succeed as the sequence count is a lower value
	round = 1
	require.NoError(t, redisCoordinator.UpdateSequenceCount(round, 4)) // should succeed
	requireSeqCount(round, 4)
	requireSeqCount(0, 3)

	// Test adding and retrieval of expressLane messages
	var addedMsgs []*ExpressLaneSubmission
	emptyTx := types.NewTransaction(0, common.MaxAddress, big.NewInt(0), 0, big.NewInt(0), nil)
	for i := uint64(0); i < 5; i++ {
		msg := &ExpressLaneSubmission{ChainId: common.Big0, Round: round, SequenceNumber: i, Transaction: emptyTx}
		require.NoError(t, redisCoordinator.AddAcceptedTx(msg), "error adding expressLane msg to redis")
		addedMsgs = append(addedMsgs, msg)
	}
	// GetAcceptedTxs skips sequence numbers that aren't in redis yet, so wait for all of them.
	// #nosec G115
	lastSeqNum := uint64(len(addedMsgs) - 1)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.Len(c, redisCoordinator.GetAcceptedTxs(round, 0, lastSeqNum), len(addedMsgs))
	}, waitFor, tick, "expressLane msgs never all landed in redis")

	checkCorrectness := func(startSeqNum uint64) {
		t.Helper()
		fetchedMsgs := redisCoordinator.GetAcceptedTxs(round, startSeqNum, startSeqNum+5)
		require.Len(t, fetchedMsgs, len(addedMsgs[startSeqNum:]), "mismatch in number of fetched msgs")
		for i, msg := range fetchedMsgs {
			haveBytes, err := msg.ToMessageBytes()
			require.NoError(t, err, "error getting messageBytes")
			// #nosec G115
			wantBytes, err := addedMsgs[int(startSeqNum)+i].ToMessageBytes()
			require.NoError(t, err, "error getting messageBytes")
			require.True(t, bytes.Equal(haveBytes, wantBytes), "mismatch in message fetched from redis")
		}
	}
	checkCorrectness(0) // when all messages are fetched
	checkCorrectness(3) // when messages are filtered with startSeqNum=3
}
