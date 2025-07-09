// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"

	dbschema "github.com/offchainlabs/nitro/arbnode/db-schema"
	"github.com/offchainlabs/nitro/arbutil"
)

func TestMessagePrunerWithPruningEligibleMessagePresent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messagesCount := uint64(2 * 100 * 1024)
	inboxTrackerDb, transactionStreamerDb, pruner := setupDatabase(t, 2*100*1024, 2*100*1024)
	err := pruner.deleteOldMessagesFromDB(ctx, arbutil.MessageIndex(messagesCount), messagesCount)
	Require(t, err)

	checkDbKeys(t, messagesCount, transactionStreamerDb, dbschema.MessagePrefix)
	checkDbKeys(t, messagesCount, transactionStreamerDb, dbschema.BlockHashInputFeedPrefix)
	checkDbKeys(t, messagesCount, transactionStreamerDb, dbschema.MessageResultPrefix)
	checkDbKeys(t, messagesCount, inboxTrackerDb, dbschema.RlpDelayedMessagePrefix)
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
	checkDbKeys(t, messagesCount/2, transactionStreamerDb, dbschema.MessagePrefix)
	// In second iteration message till messagesCount are tried to be deleted.
	err = pruner.deleteOldMessagesFromDB(ctx, arbutil.MessageIndex(messagesCount), messagesCount)
	Require(t, err)
	// In second iteration all the message till messagesCount are deleted.
	checkDbKeys(t, messagesCount, transactionStreamerDb, dbschema.MessagePrefix)
}

func TestMessagePrunerPruneTillLessThenEqualTo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messagesCount := uint64(10)
	inboxTrackerDb, transactionStreamerDb, pruner := setupDatabase(t, 2*messagesCount, 20)
	err := inboxTrackerDb.Delete(dbKey(dbschema.MessagePrefix, 9))
	Require(t, err)
	err = pruner.deleteOldMessagesFromDB(ctx, arbutil.MessageIndex(messagesCount), messagesCount)
	Require(t, err)
	hasKey, err := transactionStreamerDb.Has(dbKey(dbschema.MessagePrefix, messagesCount))
	Require(t, err)
	if !hasKey {
		Fail(t, "Key", 10, "with prefix", string(dbschema.MessagePrefix), "should be present after pruning")
	}
}

func TestMessagePrunerWithNoPruningEligibleMessagePresent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messagesCount := uint64(10)
	inboxTrackerDb, transactionStreamerDb, pruner := setupDatabase(t, messagesCount, messagesCount)
	err := pruner.deleteOldMessagesFromDB(ctx, arbutil.MessageIndex(messagesCount), messagesCount)
	Require(t, err)

	checkDbKeys(t, uint64(messagesCount), transactionStreamerDb, dbschema.MessagePrefix)
	checkDbKeys(t, uint64(messagesCount), transactionStreamerDb, dbschema.BlockHashInputFeedPrefix)
	checkDbKeys(t, uint64(messagesCount), transactionStreamerDb, dbschema.MessageResultPrefix)
	checkDbKeys(t, messagesCount, inboxTrackerDb, dbschema.RlpDelayedMessagePrefix)
}

func setupDatabase(t *testing.T, messageCount, delayedMessageCount uint64) (ethdb.Database, ethdb.Database, *MessagePruner) {
	transactionStreamerDb := rawdb.NewMemoryDatabase()
	for i := uint64(0); i < uint64(messageCount); i++ {
		err := transactionStreamerDb.Put(dbKey(dbschema.MessagePrefix, i), []byte{})
		Require(t, err)
		err = transactionStreamerDb.Put(dbKey(dbschema.BlockHashInputFeedPrefix, i), []byte{})
		Require(t, err)
		err = transactionStreamerDb.Put(dbKey(dbschema.MessageResultPrefix, i), []byte{})
		Require(t, err)
	}

	inboxTrackerDb := rawdb.NewMemoryDatabase()
	for i := uint64(0); i < delayedMessageCount; i++ {
		err := inboxTrackerDb.Put(dbKey(dbschema.RlpDelayedMessagePrefix, i), []byte{})
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
