// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"

	flag "github.com/spf13/pflag"
)

type MessagePruner struct {
	stopwaiter.StopWaiter
	transactionStreamer *TransactionStreamer
	inboxTracker        *InboxTracker
	config              MessagePrunerConfigFetcher
	pruningLock         sync.Mutex
	lastPruneDone       time.Time
}

type MessagePrunerConfig struct {
	Enable                 bool          `koanf:"enable"`
	MessagePruneInterval   time.Duration `koanf:"prune-interval" reload:"hot"`
	SearchBatchReportLimit int64         `koanf:"search-batch-report" reload:"hot"`
	MinBatchesLeft         uint64        `koanf:"min-batches-left" reload:"hot"`
}

type MessagePrunerConfigFetcher func() *MessagePrunerConfig

var DefaultMessagePrunerConfig = MessagePrunerConfig{
	Enable:                 true,
	MessagePruneInterval:   time.Minute,
	SearchBatchReportLimit: 100000,
	MinBatchesLeft:         2,
}

func MessagePrunerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultMessagePrunerConfig.Enable, "enable message pruning")
	f.Duration(prefix+".prune-interval", DefaultMessagePrunerConfig.MessagePruneInterval, "interval for running message pruner")
	f.Int64(prefix+".search-batch-report", DefaultMessagePrunerConfig.SearchBatchReportLimit, "limit for searching for a batch report when pruning (negative disables)")
	f.Uint64(prefix+".min-batches-left", DefaultMessagePrunerConfig.MinBatchesLeft, "min number of batches not pruned")
}

func NewMessagePruner(transactionStreamer *TransactionStreamer, inboxTracker *InboxTracker, config MessagePrunerConfigFetcher) *MessagePruner {
	return &MessagePruner{
		transactionStreamer: transactionStreamer,
		inboxTracker:        inboxTracker,
		config:              config,
	}
}

func (m *MessagePruner) Start(ctxIn context.Context) {
	m.StopWaiter.Start(ctxIn, m)
}

func (m *MessagePruner) UpdateLatestStaked(count arbutil.MessageIndex, globalState validator.GoGlobalState) {
	locked := m.pruningLock.TryLock()
	if !locked {
		return
	}

	if m.lastPruneDone.Add(m.config().MessagePruneInterval).After(time.Now()) {
		m.pruningLock.Unlock()
		return
	}
	err := m.LaunchThreadSafe(func(ctx context.Context) {
		defer m.pruningLock.Unlock()
		err := m.prune(ctx, count, globalState)
		if err != nil {
			log.Error("error while pruning", "err", err)
		}
	})
	if err != nil {
		log.Info("failed launching prune thread", "err", err)
		m.pruningLock.Unlock()
	}
}

// looks for batch posting report starting from delayed message delayedMsgStart
// returns number of batch for which report was found (meaning - it should not be pruned)
// if not found - returns maxUint64 (no limit on pruning)
func (m *MessagePruner) findBatchReport(ctx context.Context, delayedMsgStart uint64) (uint64, error) {
	searchLimit := m.config().SearchBatchReportLimit
	if searchLimit < 0 {
		return math.MaxUint64, nil
	}
	delayedCount, err := m.inboxTracker.GetDelayedCount()
	if err != nil {
		return 0, err
	}
	if delayedCount <= delayedMsgStart {
		return 0, errors.New("delayedCount behind pruning target")
	}
	searchUpTil := delayedCount
	searchUpLimit := delayedMsgStart + uint64(searchLimit)
	if searchLimit > 0 && searchUpLimit < searchUpTil {
		searchUpTil = searchUpLimit
	}
	for delayed := delayedMsgStart; delayed < searchUpTil; delayed++ {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		msg, err := m.inboxTracker.GetDelayedMessage(delayed)
		if err != nil {
			return 0, err
		}
		if msg.Header.Kind == arbostypes.L1MessageType_BatchPostingReport {
			_, _, _, batchNum, _, _, err := arbostypes.ParseBatchPostingReportMessageFields(bytes.NewReader(msg.L2msg))
			if err != nil {
				return 0, fmt.Errorf("trying to parse batch-posting report: %w", err)
			}
			return batchNum, nil
		}
	}
	searchDownLimit := uint64(0)
	if searchLimit > 0 {
		searchedUp := searchUpTil - delayedMsgStart
		limitRemaining := uint64(searchLimit) - searchedUp
		if limitRemaining < delayedMsgStart {
			searchDownLimit = delayedMsgStart - limitRemaining
		}
	}
	for delayed := delayedMsgStart - 1; delayed >= searchDownLimit; delayed-- {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		msg, err := m.inboxTracker.GetDelayedMessage(delayed)
		if errors.Is(err, AccumulatorNotFoundErr) {
			// older delayed probably pruned - assume we won't find a report
			return math.MaxUint64, nil
		}
		if err != nil {
			return 0, err
		}
		if msg.Header.Kind == arbostypes.L1MessageType_BatchPostingReport {
			_, _, _, batchNum, _, _, err := arbostypes.ParseBatchPostingReportMessageFields(bytes.NewReader(msg.L2msg))
			if err != nil {
				return 0, fmt.Errorf("trying to parse batch-posting report: %w", err)
			}
			// found below delayedMessage - so batchnum can be pruned but above it cannot
			return batchNum + 1, nil
		}
	}
	return math.MaxUint64, nil
}

func (m *MessagePruner) prune(ctx context.Context, count arbutil.MessageIndex, globalState validator.GoGlobalState) error {
	trimBatchCount := globalState.Batch
	minBatchesLeft := m.config().MinBatchesLeft
	if trimBatchCount < minBatchesLeft {
		return nil
	}
	batchCount, err := m.inboxTracker.GetBatchCount()
	if err != nil {
		return err
	}
	if trimBatchCount+minBatchesLeft > batchCount {
		if batchCount < minBatchesLeft {
			return nil
		}
		trimBatchCount = batchCount - minBatchesLeft
	}
	endBatchMetadata, err := m.inboxTracker.GetBatchMetadata(trimBatchCount - 1)
	if err != nil {
		return err
	}
	msgCount := endBatchMetadata.MessageCount
	delayedCount := endBatchMetadata.DelayedMessageCount

	batchPruneLimit, err := m.findBatchReport(ctx, delayedCount)
	if err != nil {
		return fmt.Errorf("failed finding batch report: %w", err)
	}
	if batchPruneLimit < trimBatchCount {
		trimBatchCount = batchPruneLimit
	}
	return deleteOldMessageFromDB(ctx, trimBatchCount, msgCount, delayedCount, m.inboxTracker.db, m.transactionStreamer.db)
}

func deleteOldMessageFromDB(ctx context.Context, endBatchCount uint64, messageCount arbutil.MessageIndex, delayedMessageCount uint64, inboxTrackerDb ethdb.Database, transactionStreamerDb ethdb.Database) error {
	prunedKeysRange, err := deleteFromLastPrunedUptoEndKey(ctx, inboxTrackerDb, sequencerBatchMetaPrefix, endBatchCount)
	if err != nil {
		return fmt.Errorf("error deleting batch metadata: %w", err)
	}
	if len(prunedKeysRange) > 0 {
		log.Info("Pruned batches:", "first pruned key", prunedKeysRange[0], "last pruned key", prunedKeysRange[len(prunedKeysRange)-1])
	}

	prunedKeysRange, err = deleteFromLastPrunedUptoEndKey(ctx, transactionStreamerDb, messagePrefix, uint64(messageCount))
	if err != nil {
		return fmt.Errorf("error deleting last batch messages: %w", err)
	}
	if len(prunedKeysRange) > 0 {
		log.Info("Pruned last batch messages:", "first pruned key", prunedKeysRange[0], "last pruned key", prunedKeysRange[len(prunedKeysRange)-1])
	}

	prunedKeysRange, err = deleteFromLastPrunedUptoEndKey(ctx, inboxTrackerDb, rlpDelayedMessagePrefix, delayedMessageCount)
	if err != nil {
		return fmt.Errorf("error deleting last batch delayed messages: %w", err)
	}
	if len(prunedKeysRange) > 0 {
		log.Info("Pruned last batch delayed messages:", "first pruned key", prunedKeysRange[0], "last pruned key", prunedKeysRange[len(prunedKeysRange)-1])
	}
	return nil
}

func deleteFromLastPrunedUptoEndKey(ctx context.Context, db ethdb.Database, prefix []byte, endMinKey uint64) ([]uint64, error) {
	startIter := db.NewIterator(prefix, uint64ToKey(1))
	if !startIter.Next() {
		return nil, nil
	}
	startMinKey := binary.BigEndian.Uint64(bytes.TrimPrefix(startIter.Key(), prefix))
	startIter.Release()
	if endMinKey > startMinKey {
		return deleteFromRange(ctx, db, prefix, startMinKey, endMinKey-1)
	}
	return nil, nil
}
