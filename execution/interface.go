package execution

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/validator"
)

type MessageResult struct {
	BlockHash common.Hash
	SendRoot  common.Hash
}

type RecordResult struct {
	Pos       arbutil.MessageIndex
	BlockHash common.Hash
	Preimages map[common.Hash][]byte
	BatchInfo []validator.BatchInfo
}

var ErrRetrySequencer = errors.New("please retry transaction")
var ErrSequencerInsertLockTaken = errors.New("insert lock taken")

// always needed
type ExecutionClient interface {
	DigestMessage(num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[struct{}]
	Reorg(count arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadata, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[struct{}]
	HeadMessageNumber() containers.PromiseInterface[arbutil.MessageIndex]
	ResultAtPos(pos arbutil.MessageIndex) containers.PromiseInterface[*MessageResult]
}

// needed for validators / stakers
type ExecutionRecorder interface {
	RecordBlockCreation(pos arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata) containers.PromiseInterface[*RecordResult]
	MarkValid(pos arbutil.MessageIndex, resultHash common.Hash)
	PrepareForRecord(start, end arbutil.MessageIndex) containers.PromiseInterface[struct{}]
}

// needed for sequencer
type ExecutionSequencer interface {
	ExecutionClient
	Pause() containers.PromiseInterface[struct{}]
	Activate() containers.PromiseInterface[struct{}]
	ForwardTo(url string) containers.PromiseInterface[struct{}]
	SequenceDelayedMessage(message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) containers.PromiseInterface[struct{}]
	NextDelayedMessageNumber() containers.PromiseInterface[uint64]
}

type FullExecutionClient interface {
	ExecutionClient
	ExecutionRecorder
	ExecutionSequencer

	Start(ctx context.Context) error
	StopAndWait()

	Maintenance() containers.PromiseInterface[struct{}]

	ArbOSVersionForMessageNumber(messageNum arbutil.MessageIndex) containers.PromiseInterface[uint64]
}

type FetchBatchResult struct {
	Data            []byte
	ParentBlockHash common.Hash
}

// not implemented in execution, used as input
// BatchFetcher is required for any execution node
type BatchFetcher interface {
	FetchBatch(batchNum uint64) containers.PromiseInterface[FetchBatchResult]
	FindInboxBatchContainingMessage(message arbutil.MessageIndex) containers.PromiseInterface[*uint64]
	GetBatchParentChainBlock(seqNum uint64) containers.PromiseInterface[uint64]
}

type ConsensusInfo interface {
	Synced() containers.PromiseInterface[bool]
	FullSyncProgressMap() containers.PromiseInterface[map[string]interface{}]
	SyncTargetMessageCount() containers.PromiseInterface[arbutil.MessageIndex]

	// TODO: switch from pulling to pushing safe/finalized
	GetSafeMsgCount() containers.PromiseInterface[arbutil.MessageIndex]
	GetFinalizedMsgCount() containers.PromiseInterface[arbutil.MessageIndex]
	ValidatedMessageCount() containers.PromiseInterface[arbutil.MessageIndex]
}

type ConsensusSequencer interface {
	WriteMessageFromSequencer(pos arbutil.MessageIndex, msgWithMeta arbostypes.MessageWithMetadata) containers.PromiseInterface[struct{}]
	ExpectChosenSequencer() containers.PromiseInterface[struct{}]
}

type FullConsensusClient interface {
	BatchFetcher
	ConsensusInfo
	ConsensusSequencer
}
