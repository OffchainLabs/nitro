// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util/stopwaiter"

	flag "github.com/spf13/pflag"
)

type MessagePruner struct {
	stopwaiter.StopWaiter
	transactionStreamer *TransactionStreamer
	inboxTracker        *InboxTracker
	staker              *staker.Staker
	config              MessagePrunerConfigFetcher
}

type MessagePrunerConfig struct {
	Enable               bool          `koanf:"enable"`
	MessagePruneInterval time.Duration `koanf:"prune-interval" reload:"hot"`
}

type MessagePrunerConfigFetcher func() *MessagePrunerConfig

var DefaultMessagePrunerConfig = MessagePrunerConfig{
	Enable:               false,
	MessagePruneInterval: time.Minute,
}

func MessagePrunerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultMessagePrunerConfig.Enable, "enable message pruning")
	f.Duration(prefix+".prune-interval", DefaultMessagePrunerConfig.MessagePruneInterval, "interval for running message pruner")
}

func NewMessagePruner(transactionStreamer *TransactionStreamer, inboxTracker *InboxTracker, staker *staker.Staker, config MessagePrunerConfigFetcher) *MessagePruner {
	return &MessagePruner{
		transactionStreamer: transactionStreamer,
		inboxTracker:        inboxTracker,
		staker:              staker,
		config:              config,
	}
}

func (m *MessagePruner) Start(ctxIn context.Context) {
	m.StopWaiter.Start(ctxIn, m)
	m.CallIteratively(m.prune)
}

func (m *MessagePruner) prune(ctx context.Context) time.Duration {
	latestConfirmedNode, err := m.staker.Rollup().LatestConfirmed(
		&bind.CallOpts{
			Context:     ctx,
			BlockNumber: big.NewInt(int64(rpc.FinalizedBlockNumber)),
		})
	if err != nil {
		log.Error("error getting latest confirmed node: %w", err)
		return m.config().MessagePruneInterval
	}
	nodeInfo, err := m.staker.Rollup().LookupNode(ctx, latestConfirmedNode)
	if err != nil {
		log.Error("error getting latest confirmed node info: %w", err)
		return m.config().MessagePruneInterval
	}
	endBatchCount := nodeInfo.Assertion.AfterState.GlobalState.Batch
	if endBatchCount == 0 {
		return m.config().MessagePruneInterval
	}
	endBatchMetadata, err := m.inboxTracker.GetBatchMetadata(endBatchCount - 1)
	if err != nil {
		log.Error("error getting last batch metadata: %w", err)
		return m.config().MessagePruneInterval
	}
	deleteOldMessageFromDB(endBatchCount, endBatchMetadata, m.inboxTracker.db, m.transactionStreamer.db)
	return m.config().MessagePruneInterval
}

func deleteOldMessageFromDB(endBatchCount uint64, endBatchMetadata BatchMetadata, inboxTrackerDb ethdb.Database, transactionStreamerDb ethdb.Database) {
	startBatchCount := uint64(1)
	hasKey, err := inboxTrackerDb.Has(lastPrunedSequencerBatchMetaKey)
	if err != nil {
		log.Warn("error checking last pruned batch metadata: %w", err)
	} else if hasKey {
		data, err := inboxTrackerDb.Get(lastPrunedSequencerBatchMetaKey)
		if err != nil {
			log.Warn("error fetching last pruned batch metadata: %w", err)
		} else {
			err = rlp.DecodeBytes(data, &startBatchCount)
			if err != nil {
				log.Warn("error decoding last pruned batch metadata: %w", err)
			}
		}
	}
	if endBatchCount > startBatchCount {
		err := deleteFromRange(inboxTrackerDb, sequencerBatchMetaPrefix, startBatchCount, endBatchCount-1)
		if err != nil {
			log.Error("error deleting batch metadata: %w", err)
			return
		}
		endBatchCountValue, err := rlp.EncodeToBytes(endBatchCount - 1)
		if err != nil {
			log.Error("error encoding end batch count: %w", err)
			return
		}
		err = inboxTrackerDb.Put(lastPrunedSequencerBatchMetaKey, endBatchCountValue)
		if err != nil {
			log.Error("error storing last pruned batch metadata: %w", err)
			return
		}
	}

	startMessageCount := uint64(1)
	hasKey, err = transactionStreamerDb.Has(lastPrunedMessageKey)
	if err != nil {
		log.Warn("error checking last pruned message: %w", err)
	} else if hasKey {
		data, err := transactionStreamerDb.Get(lastPrunedMessageKey)
		if err != nil {
			log.Warn("error fetching last pruned message: %w", err)
		} else {
			err = rlp.DecodeBytes(data, &startMessageCount)
			if err != nil {
				log.Warn("error decoding last pruned message: %w", err)
			}
		}
	}
	if endBatchMetadata.MessageCount > 1 {
		err := deleteFromRange(transactionStreamerDb, messagePrefix, startMessageCount, uint64(endBatchMetadata.MessageCount)-1)
		if err != nil {
			log.Error("error deleting last batch messages: %w", err)
		}
		endMessageCountValue, err := rlp.EncodeToBytes(uint64(endBatchMetadata.MessageCount) - 1)
		if err != nil {
			log.Error("error encoding end message count: %w", err)
			return
		}
		err = transactionStreamerDb.Put(lastPrunedMessageKey, endMessageCountValue)
		if err != nil {
			log.Error("error storing last pruned message: %w", err)
			return
		}
	}

	startDelayedMessageCount := uint64(1)
	hasKey, err = inboxTrackerDb.Has(lastPrunedDelayedMessageKey)
	if err != nil {
		log.Warn("error checking last pruned delayed message: %w", err)
	} else if hasKey {
		data, err := inboxTrackerDb.Get(lastPrunedDelayedMessageKey)
		if err != nil {
			log.Warn("error fetching last pruned delayed message: %w", err)
		} else {
			err = rlp.DecodeBytes(data, &startDelayedMessageCount)
			if err != nil {
				log.Warn("error decoding last pruned delayed message: %w", err)
			}
		}
	}
	if endBatchMetadata.DelayedMessageCount > 1 {
		err := deleteFromRange(inboxTrackerDb, rlpDelayedMessagePrefix, startDelayedMessageCount, endBatchMetadata.DelayedMessageCount-1)
		if err != nil {
			log.Error("error deleting last batch delayed messages: %w", err)
		}
		endDelayedMessageCountValue, err := rlp.EncodeToBytes(uint64(endBatchMetadata.DelayedMessageCount) - 1)
		if err != nil {
			log.Error("error encoding end delayed message count: %w", err)
			return
		}
		err = inboxTrackerDb.Put(lastPrunedDelayedMessageKey, endDelayedMessageCountValue)
		if err != nil {
			log.Error("error storing last pruned delayed message: %w", err)
			return
		}
	}
}
