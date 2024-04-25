package execution

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
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
	DigestMessage(num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) error
	Reorg(count arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadata, oldMessages []*arbostypes.MessageWithMetadata) error
	HeadMessageNumber() (arbutil.MessageIndex, error)
	HeadMessageNumberSync(t *testing.T) (arbutil.MessageIndex, error)
	ResultAtPos(pos arbutil.MessageIndex) (*MessageResult, error)
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
	GetL1GasPriceEstimate() (uint64, error)
}

type FullExecutionClient interface {
	ExecutionClient
	ExecutionRecorder
	ExecutionSequencer

	Start(ctx context.Context) error
	StopAndWait()

	Maintenance() error

	ArbOSVersionForMessageNumber(messageNum arbutil.MessageIndex) (uint64, error)
}

// not implemented in execution, used as input
// BatchFetcher is required for any execution node
type BatchFetcher interface {
	FetchBatch(ctx context.Context, batchNum uint64) ([]byte, common.Hash, error)
	FindInboxBatchContainingMessage(message arbutil.MessageIndex) (uint64, bool, error)
	GetBatchParentChainBlock(seqNum uint64) (uint64, error)
}

type ConsensusInfo interface {
	Synced() bool
	FullSyncProgressMap() map[string]interface{}
	SyncTargetMessageCount() arbutil.MessageIndex

	// TODO: switch from pulling to pushing safe/finalized
	GetSafeMsgCount(ctx context.Context) (arbutil.MessageIndex, error)
	GetFinalizedMsgCount(ctx context.Context) (arbutil.MessageIndex, error)
	ValidatedMessageCount() (arbutil.MessageIndex, error)
}

type ConsensusSequencer interface {
	BroadcastMessage(msg arbostypes.MessageWithMetadata, pos arbutil.MessageIndex)
	WriteMessageFromSequencer(pos arbutil.MessageIndex, msgWithMeta arbostypes.MessageWithMetadata) error
	ExpectChosenSequencer() error
	CacheL1PriceDataOfMsg(pos arbutil.MessageIndex, callDataUnits uint64, l1GasCharged uint64)
	BacklogL1GasCharged() uint64
	BacklogCallDataUnits() uint64
}

type FullConsensusClient interface {
	BatchFetcher
	ConsensusInfo
	ConsensusSequencer
}
