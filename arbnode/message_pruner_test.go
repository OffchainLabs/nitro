// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/offchainlabs/nitro/arbutil"
)

func TestMessagePrunerWithPruningEligibleMessagePresent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messagesCount := uint64(2 * 100 * 1024)
	inboxTrackerDb, transactionStreamerDb, pruner := setupDatabase(t, 2*100*1024, 2*100*1024)
	err := pruner.deleteOldMessagesFromDB(ctx, arbutil.MessageIndex(messagesCount), messagesCount)
	Require(t, err)

	checkDbKeys(t, messagesCount, transactionStreamerDb, messagePrefix)
	checkDbKeys(t, messagesCount, transactionStreamerDb, blockHashInputFeedPrefix)
	checkDbKeys(t, messagesCount, transactionStreamerDb, messageResultPrefix)
	checkDbKeys(t, messagesCount, inboxTrackerDb, rlpDelayedMessagePrefix)
}

func TestMessagePrunerTwoHalves(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messagesCount := uint64(10)
	_, transactionStreamerDb, pruner := setupDatabase(t, messagesCount, messagesCount)
	// In first iteration message till messagesCount/2 are tried to be deleted.
	err := pruner.deleteOldMessagesFromDB(ctx, arbutil.MessageIndex(messagesCount/2), messagesCount/2)
	Require(t, err)
	// In first iteration all the message till messagesCount/2 are deleted.
	checkDbKeys(t, messagesCount/2, transactionStreamerDb, messagePrefix)
	// In second iteration message till messagesCount are tried to be deleted.
	err = pruner.deleteOldMessagesFromDB(ctx, arbutil.MessageIndex(messagesCount), messagesCount)
	Require(t, err)
	// In second iteration all the message till messagesCount are deleted.
	checkDbKeys(t, messagesCount, transactionStreamerDb, messagePrefix)
}

func TestMessagePrunerPruneTillLessThenEqualTo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messagesCount := uint64(10)
	inboxTrackerDb, transactionStreamerDb, pruner := setupDatabase(t, 2*messagesCount, 20)
	err := inboxTrackerDb.Delete(dbKey(messagePrefix, 9))
	Require(t, err)
	err = pruner.deleteOldMessagesFromDB(ctx, arbutil.MessageIndex(messagesCount), messagesCount)
	Require(t, err)
	hasKey, err := transactionStreamerDb.Has(dbKey(messagePrefix, messagesCount))
	Require(t, err)
	if !hasKey {
		Fail(t, "Key", 10, "with prefix", string(messagePrefix), "should be present after pruning")
	}
}

func TestMessagePrunerWithNoPruningEligibleMessagePresent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messagesCount := uint64(10)
	inboxTrackerDb, transactionStreamerDb, pruner := setupDatabase(t, messagesCount, messagesCount)
	err := pruner.deleteOldMessagesFromDB(ctx, arbutil.MessageIndex(messagesCount), messagesCount)
	Require(t, err)

	checkDbKeys(t, uint64(messagesCount), transactionStreamerDb, messagePrefix)
	checkDbKeys(t, uint64(messagesCount), transactionStreamerDb, blockHashInputFeedPrefix)
	checkDbKeys(t, uint64(messagesCount), transactionStreamerDb, messageResultPrefix)
	checkDbKeys(t, messagesCount, inboxTrackerDb, rlpDelayedMessagePrefix)
}

func setupDatabase(t *testing.T, messageCount, delayedMessageCount uint64) (ethdb.Database, ethdb.Database, *MessagePruner) {
	transactionStreamerDb := rawdb.NewMemoryDatabase()
	for i := uint64(0); i < uint64(messageCount); i++ {
		err := transactionStreamerDb.Put(dbKey(messagePrefix, i), []byte{})
		Require(t, err)
		err = transactionStreamerDb.Put(dbKey(blockHashInputFeedPrefix, i), []byte{})
		Require(t, err)
		err = transactionStreamerDb.Put(dbKey(messageResultPrefix, i), []byte{})
		Require(t, err)
	}

	inboxTrackerDb := rawdb.NewMemoryDatabase()
	for i := uint64(0); i < delayedMessageCount; i++ {
		err := inboxTrackerDb.Put(dbKey(rlpDelayedMessagePrefix, i), []byte{})
		Require(t, err)
	}

	return inboxTrackerDb, transactionStreamerDb, &MessagePruner{
		transactionStreamer: &TransactionStreamer{db: transactionStreamerDb},
		inboxTracker:        &InboxTracker{db: inboxTrackerDb},
	}
}

func checkDbKeys(t *testing.T, endCount uint64, db ethdb.Database, prefix []byte) {
	t.Helper()
	for i := uint64(0); i < endCount; i++ {
		hasKey, err := db.Has(dbKey(prefix, i))
		Require(t, err)
		if i == 0 || i == endCount-1 {
			if !hasKey {
				Fail(t, "Key", i, "with prefix", string(prefix), "should be present after pruning")
			}
		} else {
			if hasKey {
				Fail(t, "Key", i, "with prefix", string(prefix), "should not be present after pruning")
			}
		}
	}
}
