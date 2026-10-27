// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

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

// liveIteratorsDB counts the iterators that were created and not yet released.
type liveIteratorsDB struct {
	ethdb.Database
	live int
}

func (db *liveIteratorsDB) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	db.live++
	return &liveIterator{Iterator: db.Database.NewIterator(prefix, start), db: db}
}

type liveIterator struct {
	ethdb.Iterator
	db       *liveIteratorsDB
	released bool
}

func (it *liveIterator) Release() {
	if !it.released {
		it.released = true
		it.db.live--
	}
	it.Iterator.Release()
}

// A leaked iterator keeps pebble from closing the consensus database.
func TestDeleteFromLastPrunedUptoEndKeyReleasesIterators(t *testing.T) {
	for _, tc := range []struct {
		name           string
		delayedCount   uint64
		wantLastPruned uint64
	}{
		{name: "only delayed message zero", delayedCount: 1, wantLastPruned: 0},
		{name: "delayed messages to prune", delayedCount: 10, wantLastPruned: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := &liveIteratorsDB{Database: rawdb.NewMemoryDatabase()}
			for i := uint64(0); i < tc.delayedCount; i++ {
				require.NoError(t, db.Put(dbKey(schema.RlpDelayedMessagePrefix, i), []byte{}))
			}
			_, lastPruned, err := deleteFromLastPrunedUptoEndKey(context.Background(), db, schema.RlpDelayedMessagePrefix, 0, 5)
			require.NoError(t, err)
			require.Equal(t, tc.wantLastPruned, lastPruned)
			require.Zero(t, db.live, "unreleased iterators")
		})
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
