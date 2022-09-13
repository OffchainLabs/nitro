package arbnode

import (
	"sync/atomic"

	"github.com/offchainlabs/nitro/arbutil"
	flag "github.com/spf13/pflag"
)

type SyncMonitor struct {
	config      *SyncMonitorConfig
	inboxReader *InboxReader
	txStreamer  *TransactionStreamer
	coordinator *SeqCoordinator
	initialized bool
}

func NewSyncMonitor(config *SyncMonitorConfig) *SyncMonitor {
	return &SyncMonitor{
		config: config,
	}
}

type SyncMonitorConfig struct {
	BlockBuildLag               uint64 `koanf:"block-build-lag"`
	BlockBuildSequencerInboxLag uint64 `koanf:"block-build-sequencer-inbox-lag"`
	CoordinatorMsgLag           uint64 `koanf:"coordinator-msg-lag"`
}

var DefaultSyncMonitorConfig = SyncMonitorConfig{
	BlockBuildLag:               20,
	BlockBuildSequencerInboxLag: 0,
	CoordinatorMsgLag:           15,
}

func SyncMonitorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".block-build-lag", DefaultSyncMonitorConfig.BlockBuildLag, "allowed lag between messages read and blocks built")
	f.Uint64(prefix+".block-build-sequencer-inbox-lag", DefaultSyncMonitorConfig.BlockBuildSequencerInboxLag, "allowed lag between messages read from sequencer inbox and blocks built")
	f.Uint64(prefix+".coordinator-msg-lag", DefaultSyncMonitorConfig.CoordinatorMsgLag, "allowed lag between local and remote messages")
}

func (s *SyncMonitor) Initialize(inboxReader *InboxReader, txStreamer *TransactionStreamer, coordinator *SeqCoordinator) {
	s.inboxReader = inboxReader
	s.txStreamer = txStreamer
	s.coordinator = coordinator
	s.initialized = true
}

func (s *SyncMonitor) SyncProgressMap() map[string]interface{} {
	syncing := false
	res := make(map[string]interface{})

	if !s.initialized {
		res["err"] = "uninitialized"
		return res
	}

	broadcasterQueuedMessagesPos := atomic.LoadUint64(&(s.txStreamer.broadcasterQueuedMessagesPos))

	if broadcasterQueuedMessagesPos != 0 { // unprocessed feed
		syncing = true
	}
	res["broadcasterQueuedMessagesPos"] = broadcasterQueuedMessagesPos

	lastBlockNum := s.txStreamer.bc.CurrentHeader().Number.Uint64()
	lastBuiltMessage, err := s.txStreamer.BlockNumberToMessageCount(lastBlockNum)
	if err != nil {
		res["blockMessageToMessageCountError"] = err.Error()
		syncing = true
	} else {
		res["blockNum"] = lastBlockNum
	}
	res["messageOfLastBlock"] = lastBuiltMessage

	msgCount, err := s.txStreamer.GetMessageCount()
	if err != nil {
		res["msgCountError"] = err.Error()
		syncing = true
	} else {
		res["msgCount"] = msgCount
		if lastBuiltMessage+arbutil.MessageIndex(s.config.BlockBuildLag) < msgCount {
			syncing = true
		}
	}

	if s.inboxReader != nil {
		batchSeen := s.inboxReader.GetLastSeenBatchCount()
		_, batchProcessed := s.inboxReader.GetLastReadBlockAndBatchCount()

		if (batchSeen == 0) || // error or not yet read inbox
			(batchProcessed < batchSeen) { // unprocessed inbox messages
			syncing = true
		}
		res["batchSeen"] = batchSeen
		res["batchProcessed"] = batchProcessed

		processedMetadata, err := s.inboxReader.Tracker().GetBatchMetadata(batchProcessed - 1)
		if err != nil {
			res["batchMetadataError"] = err.Error()
			syncing = true
		} else {
			res["messageOfProcessedBatch"] = processedMetadata.MessageCount
			if lastBuiltMessage+arbutil.MessageIndex(s.config.BlockBuildSequencerInboxLag) < processedMetadata.MessageCount {
				syncing = true
			}
		}
	}

	if s.coordinator != nil {
		coordinatorMessageCount, err := s.coordinator.GetRemoteMsgCount() //NOTE: this creates a remote call
		if err != nil {
			res["coordinatorMsgCountError"] = err.Error()
			syncing = true
		} else {
			res["coordinatorMessageCount"] = coordinatorMessageCount
			if msgCount+arbutil.MessageIndex(s.config.CoordinatorMsgLag) < coordinatorMessageCount {
				syncing = true
			}
		}
	}

	if !syncing {
		return make(map[string]interface{})
	}

	return res
}

func (s *SyncMonitor) Synced() bool {
	return len(s.SyncProgressMap()) == 0
}
