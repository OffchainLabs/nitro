// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/offchainlabs/nitro/arbnode/db/schema"
	"github.com/offchainlabs/nitro/arbutil"
)

func TestMessagePrunerWithPruningEligibleMessagePresent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messagesCount := uint64(2 * 100 * 1024)
	inboxTrackerDb, transactionStreamerDb, pruner := setupDatabase(t, 2*100*1024, 2*100*1024)
	err := pruner.deleteOldMessagesFromDB(ctx, arbutil.MessageIndex(messagesCount), messagesCount)
	Require(t, err)

	checkDbKeys(t, messagesCount, transactionStreamerDb, schema.MessagePrefix)
	checkDbKeys(t, messagesCount, transactionStreamerDb, schema.BlockHashInputFeedPrefix)
	checkDbKeys(t, messagesCount, transactionStreamerDb, schema.MessageResultPrefix)
	checkDbKeys(t, messagesCount, inboxTrackerDb, schema.RlpDelayedMessagePrefix)
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
	checkDbKeys(t, messagesCount/2, transactionStreamerDb, schema.MessagePrefix)
	// In second iteration message till messagesCount are tried to be deleted.
	err = pruner.deleteOldMessagesFromDB(ctx, arbutil.MessageIndex(messagesCount), messagesCount)
	Require(t, err)
	// In second iteration all the message till messagesCount are deleted.
	checkDbKeys(t, messagesCount, transactionStreamerDb, schema.MessagePrefix)
}

func TestMessagePrunerPruneTillLessThenEqualTo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messagesCount := uint64(10)
	inboxTrackerDb, transactionStreamerDb, pruner := setupDatabase(t, 2*messagesCount, 20)
	err := inboxTrackerDb.Delete(dbKey(schema.MessagePrefix, 9))
	Require(t, err)
	err = pruner.deleteOldMessagesFromDB(ctx, arbutil.MessageIndex(messagesCount), messagesCount)
	Require(t, err)
	hasKey, err := transactionStreamerDb.Has(dbKey(schema.MessagePrefix, messagesCount))
	Require(t, err)
	if !hasKey {
		Fail(t, "Key", 10, "with prefix", string(schema.MessagePrefix), "should be present after pruning")
	}
}

func TestMessagePrunerWithNoPruningEligibleMessagePresent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messagesCount := uint64(10)
	inboxTrackerDb, transactionStreamerDb, pruner := setupDatabase(t, messagesCount, messagesCount)
	err := pruner.deleteOldMessagesFromDB(ctx, arbutil.MessageIndex(messagesCount), messagesCount)
	Require(t, err)

	checkDbKeys(t, uint64(messagesCount), transactionStreamerDb, schema.MessagePrefix)
	checkDbKeys(t, uint64(messagesCount), transactionStreamerDb, schema.BlockHashInputFeedPrefix)
	checkDbKeys(t, uint64(messagesCount), transactionStreamerDb, schema.MessageResultPrefix)
	checkDbKeys(t, messagesCount, inboxTrackerDb, schema.RlpDelayedMessagePrefix)
}

func TestMessagePrunerLegacyDelayedBound(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up 20 entries under each legacy delayed prefix ("e", "d", "p")
	// and 10 entries under the MEL delayed prefix ("y").
	count := uint64(20)
	consensusDB := rawdb.NewMemoryDatabase()
	transactionStreamerDb := rawdb.NewMemoryDatabase()
	for i := uint64(1); i <= count; i++ {
		Require(t, consensusDB.Put(dbKey(schema.RlpDelayedMessagePrefix, i), []byte{}))
		Require(t, consensusDB.Put(dbKey(schema.LegacyDelayedMessagePrefix, i), []byte{}))
		Require(t, consensusDB.Put(dbKey(schema.ParentChainBlockNumberPrefix, i), []byte{}))
	}
	for i := uint64(1); i <= 10; i++ {
		Require(t, consensusDB.Put(dbKey(schema.MelDelayedMessagePrefix, i), []byte{}))
	}

	pruner := &MessagePruner{
		transactionStreamer: &TransactionStreamer{db: transactionStreamerDb},
		consensusDB:         consensusDB,
		batchMetaFetcher:    &InboxTracker{db: consensusDB},
		legacyDelayedBound:  15, // cap legacy pruning at index 15
	}

	// Try to prune up to index 20 — legacy prefixes should be capped at 15,
	// MEL prefix should prune up to 20.
	err := pruner.deleteOldMessagesFromDB(ctx, 0, count)
	Require(t, err)

	// Legacy "e" entries at 15+ should still exist (not pruned past bound)
	for i := uint64(15); i <= count; i++ {
		has, err := consensusDB.Has(dbKey(schema.RlpDelayedMessagePrefix, i))
		Require(t, err)
		if !has {
			Fail(t, "RlpDelayedMessagePrefix key", i, "should still exist (at or above legacyDelayedBound)")
		}
	}
	// Legacy "d" entries at 15+ should still exist
	for i := uint64(15); i <= count; i++ {
		has, err := consensusDB.Has(dbKey(schema.LegacyDelayedMessagePrefix, i))
		Require(t, err)
		if !has {
			Fail(t, "LegacyDelayedMessagePrefix key", i, "should still exist (at or above legacyDelayedBound)")
		}
	}
	// Legacy "p" entries at 15+ should still exist
	for i := uint64(15); i <= count; i++ {
		has, err := consensusDB.Has(dbKey(schema.ParentChainBlockNumberPrefix, i))
		Require(t, err)
		if !has {
			Fail(t, "ParentChainBlockNumberPrefix key", i, "should still exist (at or above legacyDelayedBound)")
		}
	}

	// Legacy entries below bound should be pruned (except boundary keys)
	for i := uint64(2); i < 14; i++ {
		for _, tc := range []struct {
			prefix []byte
			name   string
		}{
			{schema.RlpDelayedMessagePrefix, "RlpDelayedMessagePrefix"},
			{schema.LegacyDelayedMessagePrefix, "LegacyDelayedMessagePrefix"},
			{schema.ParentChainBlockNumberPrefix, "ParentChainBlockNumberPrefix"},
		} {
			has, err := consensusDB.Has(dbKey(tc.prefix, i))
			Require(t, err)
			if has {
				Fail(t, tc.name, "key", i, "should be pruned (below legacyDelayedBound)")
			}
		}
	}

	// MEL "y" entries should be pruned up to count (not capped by bound)
	for i := uint64(2); i < 10; i++ {
		has, err := consensusDB.Has(dbKey(schema.MelDelayedMessagePrefix, i))
		Require(t, err)
		if has {
			Fail(t, "MelDelayedMessagePrefix key", i, "should be pruned (not limited by legacyDelayedBound)")
		}
	}
}

func TestMessagePrunerNewPrefixes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	count := uint64(2 * 100 * 1024)
	consensusDB := rawdb.NewMemoryDatabase()
	transactionStreamerDb := rawdb.NewMemoryDatabase()

	// Populate all delayed-related prefixes
	for i := uint64(0); i < count; i++ {
		Require(t, consensusDB.Put(dbKey(schema.RlpDelayedMessagePrefix, i), []byte{}))
		Require(t, consensusDB.Put(dbKey(schema.LegacyDelayedMessagePrefix, i), []byte{}))
		Require(t, consensusDB.Put(dbKey(schema.ParentChainBlockNumberPrefix, i), []byte{}))
		Require(t, consensusDB.Put(dbKey(schema.MelDelayedMessagePrefix, i), []byte{}))
	}

	pruner := &MessagePruner{
		transactionStreamer: &TransactionStreamer{db: transactionStreamerDb},
		consensusDB:         consensusDB,
		batchMetaFetcher:    &InboxTracker{db: consensusDB},
		// No legacyDelayedBound — all prefixes should prune to the same target
	}

	err := pruner.deleteOldMessagesFromDB(ctx, 0, count)
	Require(t, err)

	// All prefixes should be pruned up to count
	for _, prefix := range [][]byte{
		schema.RlpDelayedMessagePrefix,
		schema.LegacyDelayedMessagePrefix,
		schema.ParentChainBlockNumberPrefix,
		schema.MelDelayedMessagePrefix,
	} {
		checkDbKeys(t, count, consensusDB, prefix)
	}
}

func setupDatabase(t *testing.T, messageCount, delayedMessageCount uint64) (ethdb.Database, ethdb.Database, *MessagePruner) {
	transactionStreamerDb := rawdb.NewMemoryDatabase()
	for i := uint64(0); i < uint64(messageCount); i++ {
		err := transactionStreamerDb.Put(dbKey(schema.MessagePrefix, i), []byte{})
		Require(t, err)
		err = transactionStreamerDb.Put(dbKey(schema.BlockHashInputFeedPrefix, i), []byte{})
		Require(t, err)
		err = transactionStreamerDb.Put(dbKey(schema.MessageResultPrefix, i), []byte{})
		Require(t, err)
	}

	inboxTrackerDb := rawdb.NewMemoryDatabase()
	for i := uint64(0); i < delayedMessageCount; i++ {
		err := inboxTrackerDb.Put(dbKey(schema.RlpDelayedMessagePrefix, i), []byte{})
		Require(t, err)
	}

	return inboxTrackerDb, transactionStreamerDb, &MessagePruner{
		transactionStreamer: &TransactionStreamer{db: transactionStreamerDb},
		consensusDB:         inboxTrackerDb,
		batchMetaFetcher:    &InboxTracker{db: inboxTrackerDb},
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
