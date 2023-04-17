// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
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
	m.CallIteratively(func(ctx context.Context) time.Duration {
		m.prune(ctx)
		return m.config().MessagePruneInterval
	})
}

func (m *MessagePruner) prune(ctx context.Context) {
	latestConfirmedNode, err := m.staker.Rollup().LatestConfirmed(
		&bind.CallOpts{
			Context:     ctx,
			BlockNumber: big.NewInt(int64(rpc.FinalizedBlockNumber)),
		})
	if err != nil {
		log.Error("error getting latest confirmed node: %w", err)
		return
	}
	nodeInfo, err := m.staker.Rollup().LookupNode(ctx, latestConfirmedNode)
	if err != nil {
		log.Error("error getting latest confirmed node info: %w", err)
		return
	}
	endBatchCount := nodeInfo.Assertion.AfterState.GlobalState.Batch
	if endBatchCount == 0 {
		return
	}
	endBatchMetadata, err := m.inboxTracker.GetBatchMetadata(endBatchCount - 1)
	if err != nil {
		log.Error("error getting last batch metadata: %w", err)
		return
	}
	deleteOldMessageFromDB(endBatchCount, endBatchMetadata, m.inboxTracker.db, m.transactionStreamer.db)

}

func deleteOldMessageFromDB(endBatchCount uint64, endBatchMetadata BatchMetadata, inboxTrackerDb ethdb.Database, transactionStreamerDb ethdb.Database) {
	if endBatchCount > 1 {
		err := deleteFromRange(inboxTrackerDb, sequencerBatchMetaPrefix, 1, endBatchCount-1)
		if err != nil {
			log.Error("error deleting batch metadata: %w", err)
			return
		}
	}
	if endBatchMetadata.MessageCount > 1 {
		err := deleteFromRange(transactionStreamerDb, messagePrefix, 1, uint64(endBatchMetadata.MessageCount)-1)
		if err != nil {
			log.Error("error deleting last batch messages: %w", err)
		}
	}
	if endBatchMetadata.DelayedMessageCount > 1 {
		err := deleteFromRange(inboxTrackerDb, rlpDelayedMessagePrefix, 1, endBatchMetadata.DelayedMessageCount-1)
		if err != nil {
			log.Error("error deleting last batch delayed messages: %w", err)
		}
	}
}
