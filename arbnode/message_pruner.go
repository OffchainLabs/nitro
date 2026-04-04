// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/db/schema"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
)

type MessagePruner struct {
	stopwaiter.StopWaiter
	consensusDB                       ethdb.Database
	transactionStreamer               *TransactionStreamer
	batchMetaFetcher                  BatchMetadataFetcher
	config                            MessagePrunerConfigFetcher
	pruningLock                       sync.Mutex
	lastPruneDone                     time.Time
	cachedPrunedMessages              uint64
	cachedPrunedDelayedMessages       uint64
	cachedPrunedLegacyDelayedMessages uint64
	cachedPrunedMelDelayedMessages    uint64
}

type MessagePrunerConfig struct {
	Enable bool `koanf:"enable"`
	// Message pruning interval.
	PruneInterval  time.Duration `koanf:"prune-interval" reload:"hot"`
	MinBatchesLeft uint64        `koanf:"min-batches-left" reload:"hot"`
}

type MessagePrunerConfigFetcher func() *MessagePrunerConfig

var DefaultMessagePrunerConfig = MessagePrunerConfig{
	Enable:         true,
	PruneInterval:  time.Minute,
	MinBatchesLeft: 1000,
}

func MessagePrunerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultMessagePrunerConfig.Enable, "enable message pruning")
	f.Duration(prefix+".prune-interval", DefaultMessagePrunerConfig.PruneInterval, "interval for running message pruner")
	f.Uint64(prefix+".min-batches-left", DefaultMessagePrunerConfig.MinBatchesLeft, "min number of batches not pruned")
}

func NewMessagePruner(consensusDB ethdb.Database, transactionStreamer *TransactionStreamer, batchMetaFetcher BatchMetadataFetcher, config MessagePrunerConfigFetcher) *MessagePruner {
	return &MessagePruner{
		consensusDB:         consensusDB,
		transactionStreamer: transactionStreamer,
		batchMetaFetcher:    batchMetaFetcher,
		config:              config,
	}
}

func (m *MessagePruner) Start(ctxIn context.Context) {
	m.StopWaiter.Start(ctxIn, m)
}

func (m *MessagePruner) UpdateLatestConfirmed(count arbutil.MessageIndex, globalState validator.GoGlobalState) {
	locked := m.pruningLock.TryLock()
	if !locked {
		return
	}

	if m.lastPruneDone.Add(m.config().PruneInterval).After(time.Now()) {
		m.pruningLock.Unlock()
		return
	}
	err := m.LaunchThreadSafe(func(ctx context.Context) {
		defer m.pruningLock.Unlock()
		err := m.prune(ctx, count, globalState)
		if err != nil && ctx.Err() == nil {
			log.Error("error while pruning", "err", err)
		}
	})
	if err != nil {
		log.Info("failed launching prune thread", "err", err)
		m.pruningLock.Unlock()
	}
}

func (m *MessagePruner) prune(ctx context.Context, count arbutil.MessageIndex, globalState validator.GoGlobalState) error {
	trimBatchCount := globalState.Batch
	minBatchesLeft := m.config().MinBatchesLeft
	batchCount, err := m.batchMetaFetcher.GetBatchCount()
	if err != nil {
		return err
	}
	if batchCount < trimBatchCount+minBatchesLeft {
		if batchCount < minBatchesLeft {
			return nil
		}
		trimBatchCount = batchCount - minBatchesLeft
	}
	if trimBatchCount < 1 {
		return nil
	}
	endBatchMetadata, err := m.batchMetaFetcher.GetBatchMetadata(trimBatchCount - 1)
	if err != nil {
		return err
	}
	msgCount := endBatchMetadata.MessageCount
	delayedCount := endBatchMetadata.DelayedMessageCount
	if delayedCount > 0 {
		// keep an extra delayed message for the inbox reader or message extractor to use
		delayedCount--
	}

	return m.deleteOldMessagesFromDB(ctx, msgCount, delayedCount)
}

func (m *MessagePruner) deleteOldMessagesFromDB(ctx context.Context, messageCount arbutil.MessageIndex, delayedMessageCount uint64) error {
	// Lazy-init the message bookmark for the two auxiliary deletes below.
	if m.cachedPrunedMessages == 0 {
		val, err := fetchLastPrunedKey(m.transactionStreamer.db, schema.LastPrunedMessageKey)
		if err != nil {
			return err
		}
		m.cachedPrunedMessages = val
	}

	// Auxiliary prefixes that share the message count cursor but don't track their own bookmark.
	if _, err := deleteAndLog(ctx, m.transactionStreamer.db, schema.MessageResultPrefix, m.cachedPrunedMessages, uint64(messageCount), "message results"); err != nil {
		return err
	}
	if _, err := deleteAndLog(ctx, m.transactionStreamer.db, schema.BlockHashInputFeedPrefix, m.cachedPrunedMessages, uint64(messageCount), "expected block hashes"); err != nil {
		return err
	}

	// Main prune targets, each with its own bookmark and cache.
	if err := m.prunePrefix(ctx, m.transactionStreamer.db, schema.MessagePrefix, schema.LastPrunedMessageKey, &m.cachedPrunedMessages, uint64(messageCount), "messages"); err != nil {
		return err
	}
	if err := m.prunePrefix(ctx, m.consensusDB, schema.LegacyDelayedMessagePrefix, schema.LastPrunedLegacyDelayedMessageKey, &m.cachedPrunedLegacyDelayedMessages, delayedMessageCount, "legacy delayed messages"); err != nil {
		return err
	}
	// Lazy-init cachedPrunedDelayedMessages before capturing the start position,
	// so that the ParentChainBlockNumberPrefix delete uses the correct bookmark.
	if m.cachedPrunedDelayedMessages == 0 {
		val, err := fetchLastPrunedKey(m.consensusDB, schema.LastPrunedDelayedMessageKey)
		if err != nil {
			return err
		}
		m.cachedPrunedDelayedMessages = val
	}
	parentChainPruneStart := m.cachedPrunedDelayedMessages
	if err := m.prunePrefix(ctx, m.consensusDB, schema.RlpDelayedMessagePrefix, schema.LastPrunedDelayedMessageKey, &m.cachedPrunedDelayedMessages, delayedMessageCount, "delayed messages"); err != nil {
		return err
	}
	// ParentChainBlockNumberPrefix ("p") shares the delayed message index space with RLP delayed messages.
	if _, err := deleteAndLog(ctx, m.consensusDB, schema.ParentChainBlockNumberPrefix, parentChainPruneStart, delayedMessageCount, "parent chain block numbers"); err != nil {
		return err
	}
	return m.prunePrefix(ctx, m.consensusDB, schema.MelDelayedMessagePrefix, schema.LastPrunedMelDelayedMessageKey, &m.cachedPrunedMelDelayedMessages, delayedMessageCount, "MEL delayed messages")
}

// deleteAndLog deletes keys in [startKey, endKey) under prefix and logs the pruned range.
func deleteAndLog(ctx context.Context, db ethdb.Database, prefix []byte, startKey, endKey uint64, label string) (uint64, error) {
	prunedKeysRange, lastPruned, err := deleteFromLastPrunedUptoEndKey(ctx, db, prefix, startKey, endKey)
	if err != nil {
		return 0, fmt.Errorf("error deleting %s: %w", label, err)
	}
	if len(prunedKeysRange) > 0 {
		log.Info("Pruned "+label, "first pruned key", prunedKeysRange[0], "last pruned key", prunedKeysRange[len(prunedKeysRange)-1])
	}
	return lastPruned, nil
}

// prunePrefix deletes keys under the given prefix, persists the bookmark, and
// updates the cached start position. Lazy-inits the cache from the DB on first call.
func (m *MessagePruner) prunePrefix(ctx context.Context, db ethdb.Database, prefix, bookmarkKey []byte, cachedStart *uint64, endKey uint64, label string) error {
	if *cachedStart == 0 {
		val, err := fetchLastPrunedKey(db, bookmarkKey)
		if err != nil {
			return err
		}
		*cachedStart = val
	}
	lastPruned, err := deleteAndLog(ctx, db, prefix, *cachedStart, endKey, label)
	if err != nil {
		return err
	}
	if err = insertLastPrunedKey(db, bookmarkKey, lastPruned); err != nil {
		return fmt.Errorf("error persisting last pruned %s key: %w", label, err)
	}
	*cachedStart = lastPruned
	return nil
}

// deleteFromLastPrunedUptoEndKey is similar to deleteFromRange but automatically populates the start key if it's not set.
// It returns the pruned key range, the new bookmark value (endMinKey-1 on success), and any error.
func deleteFromLastPrunedUptoEndKey(ctx context.Context, db ethdb.Database, prefix []byte, startMinKey uint64, endMinKey uint64) ([]uint64, uint64, error) {
	if startMinKey == 0 {
		startIter := db.NewIterator(prefix, uint64ToKey(1))
		if !startIter.Next() {
			return nil, 0, nil
		}
		startMinKey = binary.BigEndian.Uint64(bytes.TrimPrefix(startIter.Key(), prefix))
		startIter.Release()
	}
	if endMinKey <= startMinKey {
		return nil, startMinKey, nil
	}
	keys, err := deleteFromRange(ctx, db, prefix, startMinKey, endMinKey-1)
	return keys, endMinKey - 1, err
}

func insertLastPrunedKey(db ethdb.Database, lastPrunedKey []byte, lastPrunedValue uint64) error {
	lastPrunedValueByte, err := rlp.EncodeToBytes(lastPrunedValue)
	if err != nil {
		return fmt.Errorf("error encoding last pruned value: %w", err)
	}
	if err = db.Put(lastPrunedKey, lastPrunedValueByte); err != nil {
		return fmt.Errorf("error saving last pruned value: %w", err)
	}
	return nil
}

func fetchLastPrunedKey(db ethdb.Database, lastPrunedKey []byte) (uint64, error) {
	hasKey, err := db.Has(lastPrunedKey)
	if err != nil {
		return 0, fmt.Errorf("error checking for last pruned key: %w", err)
	}
	if !hasKey {
		return 0, nil
	}
	lastPrunedValueByte, err := db.Get(lastPrunedKey)
	if err != nil {
		return 0, fmt.Errorf("error fetching last pruned key: %w", err)
	}
	var lastPrunedValue uint64
	err = rlp.DecodeBytes(lastPrunedValueByte, &lastPrunedValue)
	if err != nil {
		return 0, fmt.Errorf("error decoding last pruned value: %w", err)
	}
	return lastPrunedValue, nil
}
