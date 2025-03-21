package execution

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/containers"
)

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

var ErrRetrySequencer = errors.New("please retry transaction")
var ErrSequencerInsertLockTaken = errors.New("insert lock taken")

// always needed
type ExecutionClient interface {
	DigestMessage(msgIdx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*MessageResult]
	Reorg(msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[[]*MessageResult]
	HeadMessageIndex() containers.PromiseInterface[arbutil.MessageIndex]
	ResultAtMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[*MessageResult]
	MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) containers.PromiseInterface[uint64]
	BlockNumberToMessageIndex(blockNum uint64) containers.PromiseInterface[arbutil.MessageIndex]
	SetFinalityData(ctx context.Context, finalityData *arbutil.FinalityData) containers.PromiseInterface[struct{}]
	MarkFeedStart(to arbutil.MessageIndex) containers.PromiseInterface[struct{}]

	Maintenance() containers.PromiseInterface[struct{}]

	Start(ctx context.Context) containers.PromiseInterface[struct{}]
	StopAndWait() containers.PromiseInterface[struct{}]
}

// needed for validators / stakers
type ExecutionRecorder interface {
	RecordBlockCreation(
		ctx context.Context,
		pos arbutil.MessageIndex,
		msg *arbostypes.MessageWithMetadata,
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
	Synced() bool
	FullSyncProgressMap() map[string]interface{}
}

// needed for batch poster
type ExecutionBatchPoster interface {
	ArbOSVersionForMessageIndex(msgIdx arbutil.MessageIndex) (uint64, error)
}

// not implemented in execution, used as input
// BatchFetcher is required for any execution node
type BatchFetcher interface {
	FindInboxBatchContainingMessage(message arbutil.MessageIndex) (uint64, bool, error)
	GetBatchParentChainBlock(seqNum uint64) (uint64, error)
}

type ConsensusInfo interface {
	Synced() bool
	FullSyncProgressMap() map[string]interface{}
	SyncTargetMessageCount() arbutil.MessageIndex
	BlockMetadataAtMessageIndex(msgIdx arbutil.MessageIndex) (common.BlockMetadata, error)
}

type ConsensusSequencer interface {
	WriteMessageFromSequencer(msgIdx arbutil.MessageIndex, msgWithMeta arbostypes.MessageWithMetadata, msgResult MessageResult, blockMetadata common.BlockMetadata) error
	ExpectChosenSequencer() error
}

type FullConsensusClient interface {
	BatchFetcher
	ConsensusInfo
	ConsensusSequencer
}
