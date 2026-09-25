// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbnode

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/broadcaster/message"
)

func backfillFeedMessages(first arbutil.MessageIndex, count int) []*message.BroadcastFeedMessage {
	messages := make([]*message.BroadcastFeedMessage, 0, count)
	for i := range count {
		msg := arbostypes.EmptyTestMessageWithMetadata
		// Matches the init message, so the delayed count stays monotonic.
		msg.DelayedMessagesRead = 1
		messages = append(messages, &message.BroadcastFeedMessage{
			SequenceNumber: first + arbutil.MessageIndex(i), // #nosec G115
			Message:        msg,
		})
	}
	return messages
}

// divergentCopy returns copies of the messages at the same sequence numbers with other content.
func divergentCopy(messages []*message.BroadcastFeedMessage) []*message.BroadcastFeedMessage {
	copies := make([]*message.BroadcastFeedMessage, 0, len(messages))
	for _, msg := range messages {
		diverged := *msg
		diverged.Message.Message = &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{Timestamp: 1},
			L2msg:  []byte{1},
		}
		copies = append(copies, &diverged)
	}
	return copies
}

// endOf is the message count once every message in the slice is stored.
func endOf(messages []*message.BroadcastFeedMessage) arbutil.MessageIndex {
	return messages[len(messages)-1].SequenceNumber + 1
}

func messageCount(t *testing.T, streamer *TransactionStreamer) arbutil.MessageIndex {
	t.Helper()
	count, err := streamer.GetMessageCount()
	Require(t, err)
	return count
}

func expectCount(t *testing.T, streamer *TransactionStreamer, want arbutil.MessageIndex, why string) {
	t.Helper()
	if got := messageCount(t, streamer); got != want {
		t.Fatalf("message count is %v, expected %v: %s", got, want, why)
	}
}

// expectPending checks the end of the queued live feed, or 0 when nothing is queued.
func expectPending(t *testing.T, streamer *TransactionStreamer, want arbutil.MessageIndex) {
	t.Helper()
	if got := streamer.FeedPendingMessageCount(); got != want {
		t.Fatalf("feed pending count is %v, expected %v", got, want)
	}
}

// backfill writes the messages through the backfill path and checks how many it reported written.
func backfill(t *testing.T, streamer *TransactionStreamer, messages []*message.BroadcastFeedMessage, expectWritten int) {
	t.Helper()
	written, err := streamer.AddBroadcastBackfillMessages(messages)
	Require(t, err)
	if written != expectWritten {
		t.Fatalf("backfill reported %v messages written, expected %v", written, expectWritten)
	}
}

func startTransactionStreamerForTest(t *testing.T) (*TransactionStreamer, func()) {
	t.Helper()
	ownerAddress := common.HexToAddress("0x1111111111111111111111111111111111111111")
	ctx, cancel := context.WithCancel(context.Background())
	exec, streamer, _, _ := NewTransactionStreamerForTest(t, ctx, ownerAddress)

	Require(t, streamer.Start(ctx))
	Require(t, exec.Start(ctx))

	cleanup := func() {
		cancel()
		streamer.StopAndWait()
		exec.StopAndWait()
	}

	return streamer, cleanup
}

func TestAddBroadcastBackfillMessages(t *testing.T) {
	streamer, cleanup := startTransactionStreamerForTest(t)
	defer cleanup()

	// The fake init message is already stored, so the head starts just above it.
	head := messageCount(t, streamer)

	t.Run("contiguous messages are written", func(t *testing.T) {
		chunk := backfillFeedMessages(head, 3)
		backfill(t, streamer, chunk, len(chunk))
		expectCount(t, streamer, endOf(chunk), "contiguous messages should be written")
	})

	t.Run("messages already stored are skipped", func(t *testing.T) {
		// Starts below the head, as an aligned chunk usually does, and reaches one past it.
		chunk := backfillFeedMessages(head, 4)
		backfill(t, streamer, chunk, 1)
		expectCount(t, streamer, endOf(chunk), "only the message past the head should be written")
	})

	t.Run("messages beyond the head are ignored", func(t *testing.T) {
		before := messageCount(t, streamer)
		backfill(t, streamer, backfillFeedMessages(before+5, 3), 0)
		expectCount(t, streamer, before, "a chunk starting above the head must not be written")
	})

	t.Run("a non-contiguous batch is rejected", func(t *testing.T) {
		messages := backfillFeedMessages(messageCount(t, streamer), 3)
		messages[2].SequenceNumber += 5
		if _, err := streamer.AddBroadcastBackfillMessages(messages); err == nil {
			t.Fatal("expected a non-contiguous batch to be rejected")
		}
	})
}

// TestAddBroadcastBackfillMessagesFlushesQueuedFeed is the case the backfill exists for: live feed
// messages arrive above a gap and wait in the broadcaster queue, and the backfilled messages below
// them must release that queue rather than replace it.
func TestAddBroadcastBackfillMessagesFlushesQueuedFeed(t *testing.T) {
	streamer, cleanup := startTransactionStreamerForTest(t)
	defer cleanup()
	head := messageCount(t, streamer)

	all := backfillFeedMessages(head, 6)
	gap, queued := all[:4], all[4:]

	// The live feed is ahead of us, so these cannot be written yet.
	Require(t, streamer.AddBroadcastMessages(queued))
	expectCount(t, streamer, head, "the queued feed messages must not be written")

	// Filling the gap writes it and releases the queue; only the gap counts as written.
	backfill(t, streamer, gap, len(gap))
	expectCount(t, streamer, endOf(queued), "the queued feed messages should have been flushed too")
	expectPending(t, streamer, 0)
}

// TestAddBroadcastBackfillMessagesPreservesQueueAboveWrite is the shape of a multi-chunk gap: the
// live feed is queued well above the chunk being written, and each write must leave that queue in
// place until one finally reaches it.
func TestAddBroadcastBackfillMessagesPreservesQueueAboveWrite(t *testing.T) {
	streamer, cleanup := startTransactionStreamerForTest(t)
	defer cleanup()
	head := messageCount(t, streamer)

	all := backfillFeedMessages(head, 8)
	first, second, queued := all[:4], all[4:6], all[6:]

	Require(t, streamer.AddBroadcastMessages(queued))
	expectPending(t, streamer, endOf(queued))

	// The first chunk ends short of the queue.
	backfill(t, streamer, first, len(first))
	expectCount(t, streamer, endOf(first), "the first chunk should be written")
	expectPending(t, streamer, endOf(queued))

	// The second chunk reaches the queue and releases it.
	backfill(t, streamer, second, len(second))
	expectCount(t, streamer, endOf(queued), "the second chunk should release the queue")
	expectPending(t, streamer, 0)
}

// TestAddBroadcastBackfillMessagesLiveFeedWinsOverlap pins that where a chunk overlaps the queued
// live feed, the live copies are the ones written: the write stops at the queue and releases it.
func TestAddBroadcastBackfillMessagesLiveFeedWinsOverlap(t *testing.T) {
	streamer, cleanup := startTransactionStreamerForTest(t)
	defer cleanup()
	head := messageCount(t, streamer)

	chunk := backfillFeedMessages(head, 4)
	// The live feed has parked the upper half with content the archive's chunk does not carry.
	live := divergentCopy(chunk[2:])
	Require(t, streamer.AddBroadcastMessages(live))

	// Only the part of the chunk below the queue counts as written.
	backfill(t, streamer, chunk, len(chunk)-len(live))
	expectCount(t, streamer, endOf(chunk), "the chunk and the live copies should both be stored")
	expectPending(t, streamer, 0)
	for _, msg := range live {
		stored, err := streamer.GetMessage(msg.SequenceNumber)
		Require(t, err)
		if !bytes.Equal(stored.Message.L2msg, msg.Message.Message.L2msg) {
			t.Fatalf("message %v holds the archive's copy, expected the live feed's", msg.SequenceNumber)
		}
	}
}

// TestAddBroadcastBackfillMessagesRefusesDivergentChunk pins that a chunk disagreeing with a stored
// message is reported as divergent and rejected whole, leaving the database untouched.
func TestAddBroadcastBackfillMessagesRefusesDivergentChunk(t *testing.T) {
	streamer, cleanup := startTransactionStreamerForTest(t)
	defer cleanup()
	head := messageCount(t, streamer)

	stored := backfillFeedMessages(head, 4)
	backfill(t, streamer, stored, len(stored))
	original, err := streamer.GetMessage(stored[2].SequenceNumber)
	Require(t, err)

	// An aligned chunk that matches below the third stored message, differs there, and carries new
	// messages above.
	chunk := backfillFeedMessages(head, 8)
	chunk[2] = divergentCopy(chunk[2:3])[0]
	if _, err := streamer.AddBroadcastBackfillMessages(chunk); !errors.Is(err, broadcastclient.ErrBackfillDiverged) {
		t.Fatalf("got %v, expected the divergent chunk to be reported as such", err)
	}

	expectCount(t, streamer, endOf(stored), "a divergent chunk must not be written")
	after, err := streamer.GetMessage(stored[2].SequenceNumber)
	Require(t, err)
	if !reflect.DeepEqual(original, after) {
		t.Fatalf("stored message %v changed from %+v to %+v", stored[2].SequenceNumber, original, after)
	}
}

// TestAddBroadcastBackfillMessagesHoldsDuringFeedReorg pins that a backfill never writes while the
// live feed is reorging: addMessagesAndEndBatchImpl would otherwise splice the reorging queue after
// the archive's history and clear the reorg state.
func TestAddBroadcastBackfillMessagesHoldsDuringFeedReorg(t *testing.T) {
	streamer, cleanup := startTransactionStreamerForTest(t)
	defer cleanup()
	head := messageCount(t, streamer)

	stored := backfillFeedMessages(head, 6)
	backfill(t, streamer, stored, len(stored))

	// The feed reorgs from the third stored message with another history, which is parked unwritten.
	reorged := divergentCopy(backfillFeedMessages(stored[2].SequenceNumber, 8))
	Require(t, streamer.AddBroadcastMessages(reorged))
	expectCount(t, streamer, endOf(stored), "the reorging feed must not be written")
	if !streamer.broadcasterQueuedMessagesActiveReorg {
		t.Fatal("expected the feed reorg to be pending")
	}
	pending := streamer.FeedPendingMessageCount()

	// An archive chunk aligned below the reorg point still carries the old history.
	if _, err := streamer.AddBroadcastBackfillMessages(backfillFeedMessages(head, 8)); !errors.Is(err, broadcastclient.ErrBackfillFeedReorgPending) {
		t.Fatalf("got %v, expected the pending feed reorg to be reported", err)
	}

	expectCount(t, streamer, endOf(stored), "the backfill must not write during a feed reorg")
	if !streamer.broadcasterQueuedMessagesActiveReorg {
		t.Fatal("the backfill cleared the pending feed reorg")
	}
	expectPending(t, streamer, pending)
}
