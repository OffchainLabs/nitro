package arbnode

import (
	"context"
	"math/big"
	"strings"
	"sync"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	melrunner "github.com/offchainlabs/nitro/arbnode/mel/runner"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type SyncMonitor struct {
	stopwaiter.StopWaiter
	config      func() *SyncMonitorConfig
	inboxReader *InboxReader
	txStreamer  *TransactionStreamer
	coordinator *SeqCoordinator
	initialized bool

	msgExtractor   *melrunner.MessageExtractor
	l1Reader       *headerreader.HeaderReader
	sequencerInbox *SequencerInbox

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

func (s *SyncMonitor) Initialize(msgExtractor *melrunner.MessageExtractor, inboxReader *InboxReader, txStreamer *TransactionStreamer, coordinator *SeqCoordinator, l1Reader *headerreader.HeaderReader, sequencerInbox *SequencerInbox) {
	s.msgExtractor = msgExtractor
	s.l1Reader = l1Reader
	s.sequencerInbox = sequencerInbox
	s.inboxReader = inboxReader
	s.txStreamer = txStreamer
	s.coordinator = coordinator
	s.initialized = true
}

func (s *SyncMonitor) updateSyncTarget(ctx context.Context) time.Duration {
	nextSyncTarget, err := s.maxMessageCount(ctx)
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
	if s.msgExtractor != nil {
		return s.msgExtractor.GetFinalizedMsgCount(ctx)
	}
	if s.inboxReader != nil && s.inboxReader.l1Reader != nil {
		return s.inboxReader.GetFinalizedMsgCount(ctx)
	}
	return 0, nil
}

func (s *SyncMonitor) maxMessageCount(ctx context.Context) (arbutil.MessageIndex, error) {
	msgCount, err := s.txStreamer.GetMessageCount()
	if err != nil {
		return 0, err
	}

	pending := s.txStreamer.FeedPendingMessageCount()
	if pending > msgCount {
		msgCount = pending
	}

	if s.msgExtractor != nil {
		melMsgCount, err := s.msgExtractor.GetMsgCount(ctx)
		if err != nil {
			return msgCount, err
		}
		msgCount = max(msgCount, melMsgCount)
	} else if s.inboxReader != nil {
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

	if !s.Started() {
		res["err"] = "notStarted"
		return res
	}

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

	if s.msgExtractor != nil {
		headMelState, err := s.msgExtractor.GetHeadState(s.GetContext())
		if err != nil {
			log.Error("Error getting head state from mel", "err", err)
			res["batchMetadataError"] = err.Error()
		} else {
			batchSeen, err := s.sequencerInbox.GetBatchCount(s.GetContext(), new(big.Int).SetUint64(headMelState.ParentChainBlockNumber+1))
			if err != nil {
				if strings.Contains(err.Error(), "header not found") {
					batchSeen = headMelState.BatchCount
				} else {
					log.Error("SequencerInbox GetBatchCount error", "err", err)
				}
			}
			res["batchSeen"] = batchSeen
			res["batchProcessed"] = headMelState.BatchCount
			res["messageOfProcessedBatch"] = headMelState.MsgCount
		}

		if s.l1Reader != nil {
			header, err := s.l1Reader.LastHeaderWithError()
			if err != nil {
				res["lastL1HeaderErr"] = err
			}
			if header != nil {
				res["lastL1BlockNum"] = header.Number
				res["lastl1BlockHash"] = header.Hash()
			}
		}
	} else if s.inboxReader != nil {
		batchSeen := s.inboxReader.GetLastSeenBatchCount()
		res["batchSeen"] = batchSeen

		batchProcessed := s.inboxReader.GetLastReadBatchCount()
		res["batchProcessed"] = batchProcessed

		if batchProcessed > 0 {
			processedBatchMsgs, err := s.inboxReader.Tracker().GetBatchMessageCount(batchProcessed - 1)
			if err != nil {
				res["batchMetadataError"] = err.Error()
			} else {
				res["messageOfProcessedBatch"] = processedBatchMsgs
			}
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

	if s.msgExtractor != nil {
		headMelState, err := s.msgExtractor.GetHeadState(s.GetContext())
		if err != nil {
			log.Error("Error getting head state from mel", "err", err)
			return false
		}
		batchSeen, err := s.sequencerInbox.GetBatchCount(s.GetContext(), new(big.Int).SetUint64(headMelState.ParentChainBlockNumber+1))
		if err != nil {
			if strings.Contains(err.Error(), "header not found") {
				batchSeen = headMelState.BatchCount
			} else {
				log.Error("SequencerInbox GetBatchCount error", "err", err)
			}
		}
		if batchSeen == 0 {
			return false
		}
		if headMelState.BatchCount < batchSeen {
			return false
		}
	} else if s.inboxReader != nil {
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
