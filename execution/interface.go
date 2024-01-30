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
	SetTransactionStreamer(streamer TransactionStreamer)
}

type FullExecutionClient interface {
	ExecutionClient
	ExecutionRecorder
	ExecutionSequencer

	Start(ctx context.Context) error
	StopAndWait()

	Maintenance() error

	// TODO: only used to get safe/finalized block numbers
	MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) uint64
}

// not implemented in execution, used as input
type BatchFetcher interface {
	FetchBatch(batchNum uint64) ([]byte, common.Hash, error)
}

type TransactionStreamer interface {
	BatchFetcher
	WriteMessageFromSequencer(pos arbutil.MessageIndex, msgWithMeta arbostypes.MessageWithMetadata) error
	ExpectChosenSequencer() error
}
