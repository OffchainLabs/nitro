package execution

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/containers"
)

type MaintenanceStatus struct {
	IsRunning bool `json:"isRunning"`
}

type MessageResult struct {
	BlockHash common.Hash
	SendRoot  common.Hash
}

type RecordResult struct {
	Pos       arbutil.MessageIndex
	BlockHash common.Hash
	Preimages map[common.Hash][]byte
	UserWasms state.UserWasms
}

type InboxBatch struct {
	BatchNum uint64
	Found    bool
}

// ConsensusSyncData contains sync status information pushed from consensus to execution
type ConsensusSyncData struct {
	Synced          bool
	MaxMessageCount arbutil.MessageIndex
	SyncProgressMap map[string]interface{} // Only populated when !Synced for debugging
	UpdatedAt       time.Time
}

var ErrRetrySequencer = errors.New("please retry transaction")
var ErrSequencerInsertLockTaken = errors.New("insert lock taken")

// always needed
type ExecutionClient interface {
	ArbOSVersionGetter

	DigestMessage(msgIdx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*MessageResult]
	Reorg(msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[[]*MessageResult]
	HeadMessageIndex() containers.PromiseInterface[arbutil.MessageIndex]
	ResultAtMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[*MessageResult]
	MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) containers.PromiseInterface[uint64]
	BlockNumberToMessageIndex(blockNum uint64) containers.PromiseInterface[arbutil.MessageIndex]
	SetFinalityData(safeFinalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) containers.PromiseInterface[struct{}]
	SetConsensusSyncData(syncData *ConsensusSyncData) containers.PromiseInterface[struct{}]
	MarkFeedStart(to arbutil.MessageIndex) containers.PromiseInterface[struct{}]

	TriggerMaintenance() containers.PromiseInterface[struct{}]
	ShouldTriggerMaintenance() containers.PromiseInterface[bool]
	MaintenanceStatus() containers.PromiseInterface[*MaintenanceStatus]

	Start(ctx context.Context) error
	StopAndWait()
}

// needed for validators / stakers
type ExecutionRecorder interface {
	RecordBlockCreation(
		ctx context.Context,
		pos arbutil.MessageIndex,
		msg *arbostypes.MessageWithMetadata,
		wasmTargets []rawdb.WasmTarget,
	) (*RecordResult, error)
	MarkValid(pos arbutil.MessageIndex, resultHash common.Hash)
	PrepareForRecord(ctx context.Context, start, end arbutil.MessageIndex) error
}

// needed for sequencer
type ExecutionSequencer interface {
	ExecutionClient
	Pause()
	Activate()
	ForwardTo(url string) error
	SequenceDelayedMessage(message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) error
	NextDelayedMessageNumber() (uint64, error)
	Synced(ctx context.Context) bool
	FullSyncProgressMap(ctx context.Context) map[string]interface{}
}

// needed for batch poster
type ArbOSVersionGetter interface {
	ArbOSVersionForMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[uint64]
}

// not implemented in execution, used as input
// BatchFetcher is required for any execution node
type BatchFetcher interface {
	FindInboxBatchContainingMessage(message arbutil.MessageIndex) containers.PromiseInterface[InboxBatch]
	GetBatchParentChainBlock(seqNum uint64) containers.PromiseInterface[uint64]
}

type ConsensusInfo interface {
	BlockMetadataAtMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[common.BlockMetadata]
}

type ConsensusSequencer interface {
	WriteMessageFromSequencer(msgIdx arbutil.MessageIndex, msgWithMeta arbostypes.MessageWithMetadata, msgResult MessageResult, blockMetadata common.BlockMetadata) containers.PromiseInterface[struct{}]
	ExpectChosenSequencer() containers.PromiseInterface[struct{}]
}

type FullConsensusClient interface {
	BatchFetcher
	ConsensusInfo
	ConsensusSequencer
}
