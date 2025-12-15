package melextraction

import (
	"context"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/daprovider"
)

// Satisfies an eventUnpacker interface that can unpack logs
// to a specific event type using the provided ABI and event name.
type logUnpacker struct{}

func (*logUnpacker) unpackLogTo(
	event any, abi *abi.ABI, eventName string, log types.Log) error {
	return unpackLogTo(event, abi, eventName, log)
}

// Defines a function that can lookup batches for a given parent chain block.
// See: parseBatchesFromBlock.
type batchLookupFunc func(
	ctx context.Context,
	melState *mel.State,
	parentChainHeader *types.Header,
	txsFetcher TransactionsFetcher,
	receiptFetcher ReceiptFetcher,
	eventUnpacker eventUnpacker,
) ([]*mel.SequencerInboxBatch, []*types.Transaction, []uint, error)

// Defines a function that can lookup delayed messages for a given parent chain block.
// See: parseDelayedMessagesFromBlock.
type delayedMsgLookupFunc func(
	ctx context.Context,
	melState *mel.State,
	parentChainHeader *types.Header,
	receiptFetcher ReceiptFetcher,
	txsFetcher TransactionsFetcher,
) ([]*mel.DelayedInboxMessage, error)

// Defines a function that can serialize a batch.
// See: serializeBatch.
type batchSerializingFunc func(
	ctx context.Context,
	batch *mel.SequencerInboxBatch,
	tx *types.Transaction,
	txIndex uint,
	receiptFetcher ReceiptFetcher,
) ([]byte, error)

// Defines a function that can parse a sequencer message from a batch.
// See: arbstate.ParseSequencerMessage.
type sequencerMessageParserFunc func(
	ctx context.Context,
	batchNum uint64,
	batchBlockHash common.Hash,
	data []byte,
	dapReaders arbstate.DapReaderSource,
	keysetValidationMode daprovider.KeysetValidationMode,
) (*arbstate.SequencerMessage, error)

// Defines a function that can extract messages from a batch.
// See: extractMessagesFromBatch.
type batchMsgExtractionFunc func(
	ctx context.Context,
	melState *mel.State,
	seqMsg *arbstate.SequencerMessage,
	delayedMsgDB DelayedMessageDatabase,
) ([]*arbostypes.MessageWithMetadata, error)

// Defines a function that can parse a batch posting report.
// See: arbostypes.ParseBatchPostingReportMessageFields.
type batchPostingReportParserFunc func(
	rd io.Reader,
) (*big.Int, common.Address, common.Hash, uint64, *big.Int, uint64, error)
