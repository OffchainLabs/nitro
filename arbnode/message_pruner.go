// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package arbnode

import (
	"bytes"
	"context"
	"encoding/binary"
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
	Enable:               true,
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
		log.Error("error getting latest confirmed node", "err", err)
		return m.config().MessagePruneInterval
	}
	nodeInfo, err := m.staker.Rollup().LookupNode(ctx, latestConfirmedNode)
	if err != nil {
		log.Error("error getting latest confirmed node info", "node", latestConfirmedNode, "err", err)
		return m.config().MessagePruneInterval
	}
	endBatchCount := nodeInfo.Assertion.AfterState.GlobalState.Batch
	if endBatchCount == 0 {
		return m.config().MessagePruneInterval
	}
	endBatchMetadata, err := m.inboxTracker.GetBatchMetadata(endBatchCount - 1)
	if err != nil {
		log.Error("error getting last batch metadata", "batch", endBatchCount-1, "err", err)
		return m.config().MessagePruneInterval
	}
	deleteOldMessageFromDB(endBatchCount, endBatchMetadata, m.inboxTracker.db, m.transactionStreamer.db)
	return m.config().MessagePruneInterval
}

func deleteOldMessageFromDB(endBatchCount uint64, endBatchMetadata BatchMetadata, inboxTrackerDb ethdb.Database, transactionStreamerDb ethdb.Database) {
	prunedKeysRange, err := deleteFromLastPrunedUptoEndKey(inboxTrackerDb, sequencerBatchMetaPrefix, endBatchCount)
	if err != nil {
		log.Error("error deleting batch metadata", "err", err)
		return
	}
	if len(prunedKeysRange) > 0 {
		log.Info("Pruned batches:", "first pruned key", prunedKeysRange[0], "last pruned key", prunedKeysRange[len(prunedKeysRange)-1])
	}

	prunedKeysRange, err = deleteFromLastPrunedUptoEndKey(transactionStreamerDb, messagePrefix, uint64(endBatchMetadata.MessageCount))
	if err != nil {
		log.Error("error deleting last batch messages", "err", err)
		return
	}
	if len(prunedKeysRange) > 0 {
		log.Info("Pruned last batch messages:", "first pruned key", prunedKeysRange[0], "last pruned key", prunedKeysRange[len(prunedKeysRange)-1])
	}

	prunedKeysRange, err = deleteFromLastPrunedUptoEndKey(inboxTrackerDb, rlpDelayedMessagePrefix, endBatchMetadata.DelayedMessageCount)
	if err != nil {
		log.Error("error deleting last batch delayed messages", "err", err)
		return
	}
	if len(prunedKeysRange) > 0 {
		log.Info("Pruned last batch delayed messages:", "first pruned key", prunedKeysRange[0], "last pruned key", prunedKeysRange[len(prunedKeysRange)-1])
	}
}

func deleteFromLastPrunedUptoEndKey(db ethdb.Database, prefix []byte, endMinKey uint64) ([][]byte, error) {
	startIter := db.NewIterator(prefix, uint64ToKey(1))
	if !startIter.Next() {
		return nil, nil
	}
	startMinKey := binary.BigEndian.Uint64(bytes.TrimPrefix(startIter.Key(), prefix))
	startIter.Release()
	if endMinKey > startMinKey {
		return deleteFromRange(db, prefix, startMinKey, endMinKey-1)
	}
	return nil, nil
}
