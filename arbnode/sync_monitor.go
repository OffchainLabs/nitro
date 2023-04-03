package arbnode

import (
	"context"
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

	maxMsgLock          sync.Mutex
	lastMaxMessageCount arbutil.MessageIndex
	prevMaxMessageCount arbutil.MessageIndex
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

func (s *SyncMonitor) updateDelayedMaxMessageCount(ctx context.Context) time.Duration {
	maxMsg, err := s.maxMessageCount()
	if err != nil {
		log.Warn("failed readin max msg count", "err", err)
		return s.config().MsgLag
	}
	s.maxMsgLock.Lock()
	defer s.maxMsgLock.Unlock()
	s.prevMaxMessageCount = s.lastMaxMessageCount
	s.lastMaxMessageCount = maxMsg
	return s.config().MsgLag
}

func (s *SyncMonitor) GetDelayedMaxMessageCount() arbutil.MessageIndex {
	s.maxMsgLock.Lock()
	defer s.maxMsgLock.Unlock()
	return s.prevMaxMessageCount
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

func (s *SyncMonitor) SyncProgressMap() map[string]interface{} {
	res := make(map[string]interface{})

	if s.Synced() {
		return res
	}

	if !s.initialized {
		res["err"] = "uninitialized"
		return res
	}

	delayedMax := s.GetDelayedMaxMessageCount()
	res["delayedMaxMsgCount"] = delayedMax

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

func (s *SyncMonitor) Start(ctx_in context.Context) {
	s.StopWaiter.Start(ctx_in, s)
	s.CallIteratively(s.updateDelayedMaxMessageCount)
}

func (s *SyncMonitor) Synced() bool {
	if !s.initialized {
		return false
	}
	if !s.Started() {
		return false
	}
	delayedMax := s.GetDelayedMaxMessageCount()

	msgCount, err := s.txStreamer.GetMessageCount()
	if err != nil {
		return false
	}

	if delayedMax > msgCount {
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
