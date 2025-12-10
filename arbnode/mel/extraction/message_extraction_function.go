package melextraction

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/daprovider"
)

// Defines a method that can read a delayed message from an external database.
type DelayedMessageDatabase interface {
	ReadDelayedMessage(
		ctx context.Context,
		state *mel.State,
		index uint64,
	) (*mel.DelayedInboxMessage, error)
}

// Defines a method that can fetch the receipt for a specific
// transaction index in a parent chain block.
type ReceiptFetcher interface {
	ReceiptForTransactionIndex(
		ctx context.Context,
		txIndex uint,
	) (*types.Receipt, error)
}

// Defines a method that can fetch transactions for
// a parent chain block by its header hash.
type TransactionsFetcher interface {
	TransactionsByHeader(
		ctx context.Context,
		parentChainHeaderHash common.Hash,
	) (types.Transactions, error)
}

// ExtractMessages is a pure function that can read a parent chain block and
// and input MEL state to run a specific algorithm that extracts Arbitrum messages and
// delayed messages observed from transactions in the block. This function can be proven
// through a replay binary, and should also compile to WAVM in addition to running in native mode.
func ExtractMessages(
	ctx context.Context,
	inputState *mel.State,
	parentChainHeader *types.Header,
	dataProviders *daprovider.DAProviderRegistry,
	delayedMsgDatabase DelayedMessageDatabase,
	receiptFetcher ReceiptFetcher,
	txsFetcher TransactionsFetcher,
) (*mel.State, []*arbostypes.MessageWithMetadata, []*mel.DelayedInboxMessage, error) {
	return extractMessagesImpl(
		ctx,
		inputState,
		parentChainHeader,
		dataProviders,
		delayedMsgDatabase,
		txsFetcher,
		receiptFetcher,
		&logUnpacker{},
		parseBatchesFromBlock,
		parseDelayedMessagesFromBlock,
		serializeBatch,
		messagesFromBatchSegments,
		arbstate.ParseSequencerMessage,
		arbostypes.ParseBatchPostingReportMessageFields,
	)
}

// Defines an internal implementation of the ExtractMessages function where many internal details
// can be mocked out for testing purposes, while the public function is clear about what dependencies it
// needs from callers.
func extractMessagesImpl(
	ctx context.Context,
	inputState *mel.State,
	parentChainHeader *types.Header,
	dataProviders *daprovider.DAProviderRegistry,
	delayedMsgDatabase DelayedMessageDatabase,
	txsFetcher TransactionsFetcher,
	receiptFetcher ReceiptFetcher,
	eventUnpacker eventUnpacker,
	lookupBatches batchLookupFunc,
	lookupDelayedMsgs delayedMsgLookupFunc,
	serialize batchSerializingFunc,
	extractBatchMessages batchMsgExtractionFunc,
	parseSequencerMessage sequencerMessageParserFunc,
	parseBatchPostingReport batchPostingReportParserFunc,
) (*mel.State, []*arbostypes.MessageWithMetadata, []*mel.DelayedInboxMessage, error) {

	state := inputState.Clone()
	// Clones the state to avoid mutating the input pointer in case of errors.
	// Check parent chain block hash linkage.
	if state.ParentChainBlockHash != parentChainHeader.ParentHash {
		return nil, nil, nil, fmt.Errorf(
			"parent chain block hash in MEL state does not match incoming block's parent hash: expected %s, got %s",
			state.ParentChainBlockHash.Hex(),
			parentChainHeader.ParentHash.Hex(),
		)
	}
	// Updates the fields in the state to corresponding to the
	// incoming parent chain block.
	state.ParentChainBlockHash = parentChainHeader.Hash()
	state.ParentChainBlockNumber = parentChainHeader.Number.Uint64()
	state.ParentChainPreviousBlockHash = parentChainHeader.ParentHash
	// Now, check for any logs emitted by the sequencer inbox by txs
	// included in the parent chain block.
	batches, batchTxs, batchTxIndices, err := lookupBatches(
		ctx,
		state,
		parentChainHeader,
		txsFetcher,
		receiptFetcher,
		eventUnpacker,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	delayedMessages, err := lookupDelayedMsgs(
		ctx,
		state,
		parentChainHeader,
		receiptFetcher,
		txsFetcher,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	// Update the delayed message accumulator in the MEL state.
	batchPostingReports := make([]*mel.DelayedInboxMessage, 0)
	for _, delayed := range delayedMessages {
		// If this message is a batch posting report, we save it for later
		// use in this function once we extract messages from batches and
		// need to fill in their batch posting report.
		if delayed.Message.Header.Kind == arbostypes.L1MessageType_BatchPostingReport || delayed.Message.Header.Kind == arbostypes.L1MessageType_Initialize { // Let's consider the init message as a batch posting report, since it is seen as a batch as well, we can later ignore filling its batchGasCost anyway
			batchPostingReports = append(batchPostingReports, delayed)
		}
		if err = state.AccumulateDelayedMessage(delayed); err != nil {
			return nil, nil, nil, err
		}
		state.DelayedMessagesSeen += 1
	}

	// Batch posting reports are included in the same transaction as a batch, so there should
	// always be the same number of reports as there are batches.
	if len(batchPostingReports) != len(batches) {
		return nil, nil, nil, fmt.Errorf(
			"batch posting reports %d do not match the number of batches %d",
			len(batchPostingReports),
			len(batches),
		)
	}

	var messages []*arbostypes.MessageWithMetadata
	for i, batch := range batches {
		batchTx := batchTxs[i]
		txIndex := batchTxIndices[i]
		serialized, err := serialize(
			ctx,
			batch,
			batchTx,
			txIndex,
			receiptFetcher,
		)
		if err != nil {
			return nil, nil, nil, err
		}

		batchPostReport := batchPostingReports[i]
		if batchPostReport.Message.Header.Kind != arbostypes.L1MessageType_Initialize {
			_, _, batchHash, _, _, _, err := parseBatchPostingReport(bytes.NewReader(batchPostReport.Message.L2msg))
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to parse batch posting report: %w", err)
			}
			gotHash := crypto.Keccak256Hash(serialized)
			if gotHash != batchHash {
				return nil, nil, nil, fmt.Errorf(
					"batch data hash incorrect %v (wanted %v for batch %v)",
					gotHash,
					batchHash,
					batch.SequenceNumber,
				)
			}
			// Fill in the batch gas stats into the batch posting report.
			batchPostReport.Message.BatchDataStats = arbostypes.GetDataStats(serialized)
		} else if !(inputState.DelayedMessagesSeen == 0 && i == 0 && delayedMessages[i] == batchPostReport) {
			return nil, nil, nil, errors.New("encountered initialize message that is not the first delayed message and the first batch ")
		}

		rawSequencerMsg, err := parseSequencerMessage(
			ctx,
			batch.SequenceNumber,
			batch.BlockHash,
			serialized,
			dataProviders,
			daprovider.KeysetValidate,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		messagesInBatch, err := extractBatchMessages(
			ctx,
			state,
			rawSequencerMsg,
			delayedMsgDatabase,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		for _, msg := range messagesInBatch {
			messages = append(messages, msg)
			state.MsgCount += 1
			if err = state.AccumulateMessage(msg); err != nil {
				return nil, nil, nil, fmt.Errorf("failed to accumulate message: %w", err)
			}
		}
		state.BatchCount += 1
	}
	return state, messages, delayedMessages, nil
}
