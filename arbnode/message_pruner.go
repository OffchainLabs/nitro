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
	consensusDB                         ethdb.Database
	transactionStreamer                 *TransactionStreamer
	batchMetaFetcher                    BatchMetadataFetcher
	config                              MessagePrunerConfigFetcher
	pruningLock                         sync.Mutex
	lastPruneDone                       time.Time
	cachedPrunedMessages                uint64
	cachedPrunedDelayedMessages         uint64
	cachedPrunedLegacyDelayedMessages   uint64
	cachedPrunedMelDelayedMessages      uint64
	cachedPrunedParentChainBlockNumbers uint64
	// legacyDelayedBound is the MEL migration boundary's delayed message count.
	// When set (>0), the pruner will not prune legacy delayed message prefixes
	// ("d", "e", "p") at or above this index, since the MEL boundary dispatch
	// still routes reads for those indices to legacy keys.
	legacyDelayedBound uint64
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

// SetLegacyDelayedBound sets the MEL migration boundary's delayed message
// count. The pruner will not prune legacy delayed message keys at or above
// this index, since the MEL boundary dispatch still routes reads to them.
// Must be called before Start.
func (m *MessagePruner) SetLegacyDelayedBound(bound uint64) {
	m.legacyDelayedBound = bound
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
		// keep an extra delayed message for the inbox reader to use
		delayedCount--
	}

	return m.deleteOldMessagesFromDB(ctx, msgCount, delayedCount)
}

func (m *MessagePruner) deleteOldMessagesFromDB(ctx context.Context, messageCount arbutil.MessageIndex, delayedMessageCount uint64) error {
	if m.cachedPrunedMessages == 0 {
		val, err := fetchLastPrunedKey(m.transactionStreamer.db, schema.LastPrunedMessageKey)
		if err != nil {
			return fmt.Errorf("fetching last pruned message key: %w", err)
		}
		m.cachedPrunedMessages = val
	}
	if m.cachedPrunedDelayedMessages == 0 {
		val, err := fetchLastPrunedKey(m.consensusDB, schema.LastPrunedDelayedMessageKey)
		if err != nil {
			return fmt.Errorf("fetching last pruned delayed message key: %w", err)
		}
		m.cachedPrunedDelayedMessages = val
	}
	if m.cachedPrunedLegacyDelayedMessages == 0 {
		val, err := fetchLastPrunedKey(m.consensusDB, schema.LastPrunedLegacyDelayedMessageKey)
		if err != nil {
			return fmt.Errorf("fetching last pruned legacy delayed message key: %w", err)
		}
		m.cachedPrunedLegacyDelayedMessages = val
	}
	if m.cachedPrunedMelDelayedMessages == 0 {
		val, err := fetchLastPrunedKey(m.consensusDB, schema.LastPrunedMelDelayedMessageKey)
		if err != nil {
			return fmt.Errorf("fetching last pruned MEL delayed message key: %w", err)
		}
		m.cachedPrunedMelDelayedMessages = val
	}
	if m.cachedPrunedParentChainBlockNumbers == 0 {
		val, err := fetchLastPrunedKey(m.consensusDB, schema.LastPrunedParentChainBlockNumberKey)
		if err != nil {
			return fmt.Errorf("fetching last pruned parent chain block number key: %w", err)
		}
		m.cachedPrunedParentChainBlockNumbers = val
	}

	// Cap the delayed prune target for legacy prefixes to avoid pruning entries
	// that the MEL boundary dispatch still routes reads to.
	legacyDelayedPruneLimit := delayedMessageCount
	if m.legacyDelayedBound > 0 && legacyDelayedPruneLimit > m.legacyDelayedBound {
		legacyDelayedPruneLimit = m.legacyDelayedBound
	}

	prunedKeysRange, _, err := deleteFromLastPrunedUptoEndKey(ctx, m.transactionStreamer.db, schema.MessageResultPrefix, m.cachedPrunedMessages, uint64(messageCount))
	if err != nil {
		return fmt.Errorf("error deleting message results: %w", err)
	}
	if len(prunedKeysRange) > 0 {
		log.Info("Pruned message results:", "first pruned key", prunedKeysRange[0], "last pruned key", prunedKeysRange[len(prunedKeysRange)-1])
	}

	prunedKeysRange, _, err = deleteFromLastPrunedUptoEndKey(ctx, m.transactionStreamer.db, schema.BlockHashInputFeedPrefix, m.cachedPrunedMessages, uint64(messageCount))
	if err != nil {
		return fmt.Errorf("error deleting expected block hashes: %w", err)
	}
	if len(prunedKeysRange) > 0 {
		log.Info("Pruned expected block hashes:", "first pruned key", prunedKeysRange[0], "last pruned key", prunedKeysRange[len(prunedKeysRange)-1])
	}

	prunedKeysRange, lastPrunedMessage, err := deleteFromLastPrunedUptoEndKey(ctx, m.transactionStreamer.db, schema.MessagePrefix, m.cachedPrunedMessages, uint64(messageCount))
	if err != nil {
		return fmt.Errorf("error deleting last batch messages: %w", err)
	}
	if len(prunedKeysRange) > 0 {
		log.Info("Pruned last batch messages:", "first pruned key", prunedKeysRange[0], "last pruned key", prunedKeysRange[len(prunedKeysRange)-1])
	}
	if err := insertLastPrunedKey(m.transactionStreamer.db, schema.LastPrunedMessageKey, lastPrunedMessage); err != nil {
		return fmt.Errorf("persisting last pruned message key: %w", err)
	}
	m.cachedPrunedMessages = lastPrunedMessage

	prunedKeysRange, lastPrunedDelayedMessage, err := deleteFromLastPrunedUptoEndKey(ctx, m.consensusDB, schema.RlpDelayedMessagePrefix, m.cachedPrunedDelayedMessages, legacyDelayedPruneLimit)
	if err != nil {
		return fmt.Errorf("error deleting last batch delayed messages: %w", err)
	}
	if len(prunedKeysRange) > 0 {
		log.Info("Pruned last batch delayed messages:", "first pruned key", prunedKeysRange[0], "last pruned key", prunedKeysRange[len(prunedKeysRange)-1])
	}
	if err := insertLastPrunedKey(m.consensusDB, schema.LastPrunedDelayedMessageKey, lastPrunedDelayedMessage); err != nil {
		return fmt.Errorf("persisting last pruned delayed message key: %w", err)
	}
	m.cachedPrunedDelayedMessages = lastPrunedDelayedMessage

	// Prune legacy "d"-prefixed delayed messages (oldest format, pre-RLP).
	prunedKeysRange, lastPrunedLegacyDelayed, err := deleteFromLastPrunedUptoEndKey(ctx, m.consensusDB, schema.LegacyDelayedMessagePrefix, m.cachedPrunedLegacyDelayedMessages, legacyDelayedPruneLimit)
	if err != nil {
		return fmt.Errorf("error deleting legacy delayed messages: %w", err)
	}
	if len(prunedKeysRange) > 0 {
		log.Info("Pruned legacy delayed messages:", "first pruned key", prunedKeysRange[0], "last pruned key", prunedKeysRange[len(prunedKeysRange)-1])
	}
	if err := insertLastPrunedKey(m.consensusDB, schema.LastPrunedLegacyDelayedMessageKey, lastPrunedLegacyDelayed); err != nil {
		return fmt.Errorf("persisting last pruned legacy delayed message key: %w", err)
	}
	m.cachedPrunedLegacyDelayedMessages = lastPrunedLegacyDelayed

	// Prune MEL-prefixed delayed messages (written by message extraction layer).
	prunedKeysRange, lastPrunedMelDelayed, err := deleteFromLastPrunedUptoEndKey(ctx, m.consensusDB, schema.MelDelayedMessagePrefix, m.cachedPrunedMelDelayedMessages, delayedMessageCount)
	if err != nil {
		return fmt.Errorf("error deleting MEL delayed messages: %w", err)
	}
	if len(prunedKeysRange) > 0 {
		log.Info("Pruned MEL delayed messages:", "first pruned key", prunedKeysRange[0], "last pruned key", prunedKeysRange[len(prunedKeysRange)-1])
	}
	if err := insertLastPrunedKey(m.consensusDB, schema.LastPrunedMelDelayedMessageKey, lastPrunedMelDelayed); err != nil {
		return fmt.Errorf("persisting last pruned MEL delayed message key: %w", err)
	}
	m.cachedPrunedMelDelayedMessages = lastPrunedMelDelayed

	// Prune parent chain block number entries (legacy "p" prefix, keyed by delayed message index).
	prunedKeysRange, lastPrunedPCBN, err := deleteFromLastPrunedUptoEndKey(ctx, m.consensusDB, schema.ParentChainBlockNumberPrefix, m.cachedPrunedParentChainBlockNumbers, legacyDelayedPruneLimit)
	if err != nil {
		return fmt.Errorf("error deleting parent chain block numbers: %w", err)
	}
	if len(prunedKeysRange) > 0 {
		log.Info("Pruned parent chain block numbers:", "first pruned key", prunedKeysRange[0], "last pruned key", prunedKeysRange[len(prunedKeysRange)-1])
	}
	if err := insertLastPrunedKey(m.consensusDB, schema.LastPrunedParentChainBlockNumberKey, lastPrunedPCBN); err != nil {
		return fmt.Errorf("persisting last pruned parent chain block number key: %w", err)
	}
	m.cachedPrunedParentChainBlockNumbers = lastPrunedPCBN
	return nil
}

// deleteFromLastPrunedUptoEndKey is similar to deleteFromRange but automatically populates the start key if it's not set.
// It's returns the new start key (i.e. last pruned key) at the end of this function if successful.
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
		return fmt.Errorf("encoding last pruned value: %w", err)
	}
	if err := db.Put(lastPrunedKey, lastPrunedValueByte); err != nil {
		return fmt.Errorf("saving last pruned value: %w", err)
	}
	return nil
}

func fetchLastPrunedKey(db ethdb.Database, lastPrunedKey []byte) (uint64, error) {
	hasKey, err := db.Has(lastPrunedKey)
	if err != nil {
		return 0, fmt.Errorf("checking for last pruned key: %w", err)
	}
	if !hasKey {
		return 0, nil
	}
	lastPrunedValueByte, err := db.Get(lastPrunedKey)
	if err != nil {
		return 0, fmt.Errorf("fetching last pruned key: %w", err)
	}
	var lastPrunedValue uint64
	if err := rlp.DecodeBytes(lastPrunedValueByte, &lastPrunedValue); err != nil {
		return 0, fmt.Errorf("decoding last pruned value: %w", err)
	}
	return lastPrunedValue, nil
}
