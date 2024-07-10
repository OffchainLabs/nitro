package arbnode

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	flag "github.com/spf13/pflag"
)

type SyncMonitor struct {
	stopwaiter.StopWaiter
	config      func() *SyncMonitorConfig
	inboxReader *InboxReader
	txStreamer  *TransactionStreamer
	coordinator *SeqCoordinator
	initialized bool

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

func SyncMonitorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Duration(prefix+".msg-lag", DefaultSyncMonitorConfig.MsgLag, "allowed msg lag while still considered in sync")
}

func (s *SyncMonitor) Initialize(inboxReader *InboxReader, txStreamer *TransactionStreamer, coordinator *SeqCoordinator) {
	s.inboxReader = inboxReader
	s.txStreamer = txStreamer
	s.coordinator = coordinator
	s.initialized = true
}

func (s *SyncMonitor) updateSyncTarget(ctx context.Context) time.Duration {
	nextSyncTarget, err := s.maxMessageCount()
	if err != nil {
		log.Warn("failed readin max msg count", "err", err)
		return s.config().MsgLag
	}
	s.syncTargetLock.Lock()
	defer s.syncTargetLock.Unlock()
	s.syncTarget = s.nextSyncTarget
	s.nextSyncTarget = nextSyncTarget
	return s.config().MsgLag
}

func (s *SyncMonitor) SyncTargetMessageCount() arbutil.MessageIndex {
	s.syncTargetLock.Lock()
	defer s.syncTargetLock.Unlock()
	return s.syncTarget
}

func (s *SyncMonitor) GetFinalizedMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	if s.inboxReader != nil && s.inboxReader.l1Reader != nil {
		return s.inboxReader.GetFinalizedMsgCount(ctx)
	}
	return 0, errors.New("sync monitor's GetFinalizedMsgCount method is unsupported, try starting node with --parent-chain.connection.url")
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

	if s.inboxReader != nil {
		batchProcessed := s.inboxReader.GetLastReadBatchCount()

		if batchProcessed > 0 {
			batchMsgCount, err := s.inboxReader.Tracker().GetBatchMessageCount(batchProcessed - 1)
			if err != nil {
				return msgCount, err
			}
			if batchMsgCount > msgCount {
				msgCount = batchMsgCount
			}
		}
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

	if !s.initialized {
		res["err"] = "uninitialized"
		return res
	}

	syncTarget := s.SyncTargetMessageCount()
	res["syncTargetMsgCount"] = syncTarget

	msgCount, err := s.txStreamer.GetMessageCount()
	if err != nil {
		res["msgCountError"] = err.Error()
		return res
	}
	res["msgCount"] = msgCount

	res["feedPendingMessageCount"] = s.txStreamer.FeedPendingMessageCount()

	if s.inboxReader != nil {
		batchSeen := s.inboxReader.GetLastSeenBatchCount()
		res["batchSeen"] = batchSeen

		batchProcessed := s.inboxReader.GetLastReadBatchCount()
		res["batchProcessed"] = batchProcessed

		processedBatchMsgs, err := s.inboxReader.Tracker().GetBatchMessageCount(batchProcessed - 1)
		if err != nil {
			res["batchMetadataError"] = err.Error()
		} else {
			res["messageOfProcessedBatch"] = processedBatchMsgs
		}

		l1reader := s.inboxReader.l1Reader
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
	if !s.initialized {
		return false
	}
	if !s.Started() {
		return false
	}
	syncTarget := s.SyncTargetMessageCount()

	msgCount, err := s.txStreamer.GetMessageCount()
	if err != nil {
		return false
	}

	if syncTarget > msgCount {
		return false
	}

	if s.inboxReader != nil {
		batchSeen := s.inboxReader.GetLastSeenBatchCount()
		if batchSeen == 0 {
			return false
		}
		batchProcessed := s.inboxReader.GetLastReadBatchCount()

		if batchProcessed < batchSeen {
			return false
		}
	}
	return true
}
