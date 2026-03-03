// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbnode

import (
	"context"
	"sync"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type MessageSyncProgressFetcher interface {
	GetFinalizedMsgCount(ctx context.Context) (arbutil.MessageIndex, error)
	GetMsgCount(ctx context.Context) (arbutil.MessageIndex, error)
	GetSyncProgress(ctx context.Context) (mel.MessageSyncProgress, error)
	GetL1Reader() *headerreader.HeaderReader
}

type SyncMonitor struct {
	stopwaiter.StopWaiter
	config              func() *SyncMonitorConfig
	txStreamer          *TransactionStreamer
	coordinator         *SeqCoordinator
	initialized         bool
	syncProgressFetcher MessageSyncProgressFetcher

	syncTargetLock sync.Mutex
	nextSyncTarget arbutil.MessageIndex
	syncTarget     arbutil.MessageIndex
}

func NewSyncMonitor(config func() *SyncMonitorConfig) *SyncMonitor {
	return &SyncMonitor{
		config: config,
	}
}

type SyncMonitorConfig struct {
	MsgLag time.Duration `koanf:"msg-lag"`
}

var DefaultSyncMonitorConfig = SyncMonitorConfig{
	MsgLag: time.Second,
}

var TestSyncMonitorConfig = SyncMonitorConfig{
	MsgLag: time.Millisecond * 10,
}

func SyncMonitorConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Duration(prefix+".msg-lag", DefaultSyncMonitorConfig.MsgLag, "allowed msg lag while still considered in sync")
}

func (s *SyncMonitor) Initialize(syncProgressFetcher MessageSyncProgressFetcher, txStreamer *TransactionStreamer, coordinator *SeqCoordinator) {
	s.syncProgressFetcher = syncProgressFetcher
	s.txStreamer = txStreamer
	s.coordinator = coordinator
	s.initialized = true
}

func (s *SyncMonitor) updateSyncTarget(ctx context.Context) time.Duration {
	nextSyncTarget, err := s.maxMessageCount()
	s.syncTargetLock.Lock()
	defer s.syncTargetLock.Unlock()
	if err == nil {
		s.syncTarget = s.nextSyncTarget
		s.nextSyncTarget = nextSyncTarget
	} else {
		log.Warn("failed reading max msg count", "err", err)
		s.nextSyncTarget = 0
		s.syncTarget = 0
	}
	return s.config().MsgLag
}

// note: if this returns 0 - node is not synced (init message is 1)
func (s *SyncMonitor) SyncTargetMessageCount() arbutil.MessageIndex {
	s.syncTargetLock.Lock()
	defer s.syncTargetLock.Unlock()
	return s.syncTarget
}

func (s *SyncMonitor) GetFinalizedMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	return s.syncProgressFetcher.GetFinalizedMsgCount(ctx)
}

func (s *SyncMonitor) GetMaxMessageCount() (arbutil.MessageIndex, error) {
	return s.maxMessageCount()
}

func (s *SyncMonitor) maxMessageCount() (arbutil.MessageIndex, error) {
	msgCount, err := s.txStreamer.GetMessageCount()
	if err != nil {
		return 0, err
	}

	pending := s.txStreamer.FeedPendingMessageCount()
	if pending > msgCount {
		msgCount = pending
	}

	if s.syncProgressFetcher != nil {
		fetched, err := s.syncProgressFetcher.GetMsgCount(s.GetContext())
		if err != nil {
			return msgCount, err
		}
		msgCount = max(msgCount, fetched)
	}

	if s.coordinator != nil {
		coordinatorMessageCount, err := s.coordinator.GetRemoteMsgCount() //NOTE: this creates a remote call
		if err != nil {
			return msgCount, err
		}
		if coordinatorMessageCount > msgCount {
			msgCount = coordinatorMessageCount
		}
	}

	return msgCount, nil
}

func (s *SyncMonitor) FullSyncProgressMap() map[string]interface{} {
	res := make(map[string]interface{})

	if !s.Started() {
		res["err"] = "notStarted"
		return res
	}

	if !s.initialized {
		res["err"] = "uninitialized"
		return res
	}

	syncTarget := s.SyncTargetMessageCount()
	res["consensusSyncTargetMsgCount"] = syncTarget

	maxMsgCount, err := s.maxMessageCount()
	if err != nil {
		res["maxMessageCountError"] = err.Error()
		return res
	}
	res["maxMessageCount"] = maxMsgCount

	msgCount, err := s.txStreamer.GetMessageCount()
	if err != nil {
		res["msgCountError"] = err.Error()
		return res
	}
	res["msgCount"] = msgCount

	res["feedPendingMessageCount"] = s.txStreamer.FeedPendingMessageCount()

	progress, err := s.syncProgressFetcher.GetSyncProgress(s.GetContext())
	if err != nil {
		log.Error("Error getting sync progress", "err", err)
		res["batchMetadataError"] = err.Error()
	} else {
		res["batchSeen"] = progress.BatchSeen
		res["batchProcessed"] = progress.BatchProcessed
		if progress.BatchProcessed > 0 {
			res["messageOfProcessedBatch"] = progress.MsgCount
		}
	}

	l1reader := s.syncProgressFetcher.GetL1Reader()
	if l1reader != nil {
		header, err := l1reader.LastHeaderWithError()
		if err != nil {
			res["lastL1HeaderErr"] = err
		}
		if header != nil {
			res["lastL1BlockNum"] = header.Number
			res["lastl1BlockHash"] = header.Hash()
		}
	}

	if s.coordinator != nil {
		coordinatorMessageCount, err := s.coordinator.GetRemoteMsgCount() //NOTE: this creates a remote call
		if err != nil {
			res["coordinatorMsgCountError"] = err.Error()
		} else {
			res["coordinatorMessageCount"] = coordinatorMessageCount
		}
	}

	return res
}

func (s *SyncMonitor) SyncProgressMap() map[string]interface{} {
	if s.Synced() {
		return make(map[string]interface{})
	}

	return s.FullSyncProgressMap()
}

func (s *SyncMonitor) Start(ctx_in context.Context) {
	s.StopWaiter.Start(ctx_in, s)
	s.CallIteratively(s.updateSyncTarget)
}

func (s *SyncMonitor) Synced() bool {
	if !s.Started() {
		return false
	}
	if !s.initialized {
		return false
	}
	syncTarget := s.SyncTargetMessageCount()
	if syncTarget == 0 {
		return false
	}

	msgCount, err := s.txStreamer.GetMessageCount()
	if err != nil {
		return false
	}

	if syncTarget > msgCount {
		return false
	}

	if s.syncProgressFetcher != nil {
		progress, err := s.syncProgressFetcher.GetSyncProgress(s.GetContext())
		if err != nil {
			log.Error("Error getting sync progress", "err", err)
			return false
		}
		if progress.BatchSeen == 0 {
			return false
		}
		if progress.BatchProcessed < progress.BatchSeen {
			return false
		}
	}
	return true
}
