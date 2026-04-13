// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melextraction

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
)

// Defines a method that can read a delayed message from an external database.
type DelayedMessageDatabase interface {
	ReadDelayedMessage(
		state *mel.State,
		index uint64,
	) (*mel.DelayedInboxMessage, error)
}

// Defines methods that can fetch all the logs of a parent chain block
// and logs corresponding to a specific transaction in a parent chain block.
type LogsFetcher interface {
	LogsForBlockHash(
		ctx context.Context,
		parentChainBlockHash common.Hash,
	) ([]*types.Log, error)
	LogsForTxIndex(
		ctx context.Context,
		parentChainBlockHash common.Hash,
		txIndex uint,
	) ([]*types.Log, error)
}

// Defines a method that can fetch transaction of a parent chain block by hash.
type TransactionFetcher interface {
	TransactionByLog(
		ctx context.Context,
		log *types.Log,
	) (*types.Transaction, error)
}

// ExtractMessages is a pure function that can read a parent chain block and
// and input MEL state to run a specific algorithm that extracts Arbitrum messages and
// delayed messages observed from transactions in the block. This function can be proven
// through a replay binary, and should also compile to WAVM in addition to running in native mode.
func ExtractMessages(
	ctx context.Context,
	inputState *mel.State,
	parentChainHeader *types.Header,
	dapReaders arbstate.DapReaderSource,
	delayedMsgDatabase DelayedMessageDatabase,
	txFetcher TransactionFetcher,
	logsFetcher LogsFetcher,
	chainConfig *params.ChainConfig,
) (*mel.State, []*arbostypes.MessageWithMetadata, []*mel.DelayedInboxMessage, []*mel.BatchMetadata, error) {
	return extractMessagesImpl(
		ctx,
		inputState,
		parentChainHeader,
		chainConfig,
		dapReaders,
		delayedMsgDatabase,
		txFetcher,
		logsFetcher,
		&LogUnpacker{},
		ParseBatchesFromBlock,
		parseDelayedMessagesFromBlock,
		SerializeBatch,
		messagesFromBatchSegments,
		arbstate.ParseSequencerMessage,
		arbostypes.ParseBatchPostingReportMessageFields,
		ParseMELConfigFromBlock,
	)
}

// Defines an internal implementation of the ExtractMessages function where many internal details
// can be mocked out for testing purposes, while the public function is clear about what dependencies it
// needs from callers.
func extractMessagesImpl(
	ctx context.Context,
	inputState *mel.State,
	parentChainHeader *types.Header,
	chainConfig *params.ChainConfig,
	dapReaders arbstate.DapReaderSource,
	delayedMsgDatabase DelayedMessageDatabase,
	txFetcher TransactionFetcher,
	logsFetcher LogsFetcher,
	eventUnpacker EventUnpacker,
	lookupBatches batchLookupFunc,
	lookupDelayedMsgs delayedMsgLookupFunc,
	serialize batchSerializingFunc,
	extractBatchMessages batchMsgExtractionFunc,
	parseSequencerMessage sequencerMessageParserFunc,
	parseBatchPostingReport batchPostingReportParserFunc,
	lookupMELConfig melConfigLookupFunc,
) (*mel.State, []*arbostypes.MessageWithMetadata, []*mel.DelayedInboxMessage, []*mel.BatchMetadata, error) {

	state := inputState.Clone()
	// Clones the state to avoid mutating the input pointer in case of errors.
	// Check parent chain block hash linkage.
	if state.ParentChainBlockHash != parentChainHeader.ParentHash {
		return nil, nil, nil, nil, fmt.Errorf(
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
	batches, batchTxs, err := lookupBatches(
		ctx,
		parentChainHeader,
		txFetcher,
		logsFetcher,
		eventUnpacker,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	delayedMessages, err := lookupDelayedMsgs(
		ctx,
		state,
		parentChainHeader,
		txFetcher,
		logsFetcher,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// Extract batch posting reports from delayed messages
	batchPostingReports := make([]*mel.DelayedInboxMessage, 0)
	for _, delayed := range delayedMessages {
		// If this message is a batch posting report, we save it for later
		// use in this function once we extract messages from batches and
		// need to fill in their batch posting report.
		if delayed.Message.Header.Kind == arbostypes.L1MessageType_BatchPostingReport {
			batchPostingReports = append(batchPostingReports, delayed)
		}
	}

	// Batch posting reports are included in the same transaction as a batch, so there should
	// be the same or lesser number of reports as there are batches.
	if len(batchPostingReports) > len(batches) {
		return nil, nil, nil, nil, fmt.Errorf(
			"number of batch posting reports %d higher than the number of batches %d",
			len(batchPostingReports),
			len(batches),
		)
	}

	var batchMetas []*mel.BatchMetadata
	var messages []*arbostypes.MessageWithMetadata
	var serializedBatches [][]byte
	var batchPostReportIndex int
	var batchPostReportBatchHash common.Hash
	for i, batch := range batches {
		batchTx := batchTxs[i]
		serialized, err := serialize(
			ctx,
			batch,
			batchTx,
			logsFetcher,
		)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		if batchPostReportIndex < len(batchPostingReports) {
			batchPostReport := batchPostingReports[batchPostReportIndex]
			if (batchPostReportBatchHash == common.Hash{}) {
				_, _, batchPostReportBatchHash, _, _, _, err = parseBatchPostingReport(bytes.NewReader(batchPostReport.Message.L2msg))
				if err != nil {
					return nil, nil, nil, nil, fmt.Errorf("failed to parse batch posting report: %w", err)
				}
			}
			gotHash := crypto.Keccak256Hash(serialized)
			if gotHash == batchPostReportBatchHash {
				// Fill in the batch gas stats into the batch posting report.
				batchPostReport.Message.BatchDataStats = arbostypes.GetDataStats(serialized)
				legacyCost := arbostypes.LegacyCostForStats(batchPostReport.Message.BatchDataStats)
				batchPostReport.Message.LegacyBatchGasCost = &legacyCost
				// Process next report
				batchPostReportIndex++
				batchPostReportBatchHash = common.Hash{}
			}
		}
		serializedBatches = append(serializedBatches, serialized)
	}

	// Batch posting reports are included in the same transaction as a batch, so all the
	// reports should have been filled in with the batch gas stats
	if len(batchPostingReports) != batchPostReportIndex {
		return nil, nil, nil, nil, fmt.Errorf(
			"not all batch posting reports processed. Have: %d, Processed: %d",
			len(batchPostingReports),
			batchPostReportIndex,
		)
	}

	// Update the delayed message inbox accumulator in the MEL state.
	for _, delayed := range delayedMessages {
		if err = state.AccumulateDelayedMessage(delayed); err != nil {
			return nil, nil, nil, nil, err
		}
		state.DelayedMessagesSeen += 1
	}

	// Extract L2 messages from batches
	for i, batch := range batches {
		// #nosec G115
		expectedBatchSeqNum := batches[0].SequenceNumber + uint64(i)
		if batch.SequenceNumber != expectedBatchSeqNum {
			return nil, nil, nil, nil, fmt.Errorf("unexpected batch sequence number %v expected %v", batch.SequenceNumber, expectedBatchSeqNum)
		}
		serialized := serializedBatches[i]
		rawSequencerMsg, err := parseSequencerMessage(
			ctx,
			batch.SequenceNumber,
			batch.BlockHash,
			serialized,
			dapReaders,
			daprovider.KeysetValidate,
			chainConfig,
		)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		messagesInBatch, err := extractBatchMessages(
			ctx,
			state,
			rawSequencerMsg,
			delayedMsgDatabase,
		)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		for _, msg := range messagesInBatch {
			messages = append(messages, msg)
			if err = state.AccumulateMessage(msg); err != nil {
				return nil, nil, nil, nil, fmt.Errorf("failed to accumulate message: %w", err)
			}
			// Updating of MsgCount is consistent with how DelayedMessagesSeen is updated
			// i.e after the corresponding message has been accumulated
			state.MsgCount += 1
		}
		state.BatchCount += 1
		batchMetas = append(batchMetas, &mel.BatchMetadata{
			Accumulator:         batch.AfterInboxAcc,
			MessageCount:        arbutil.MessageIndex(state.MsgCount),
			DelayedMessageCount: batch.AfterDelayedCount,
			ParentChainBlock:    batch.ParentChainBlockNumber,
		})
		if batch.AfterDelayedCount != state.DelayedMessagesRead {
			return nil, nil, nil, nil, fmt.Errorf("batch AfterDelayedCount: %d and MEL state DelayedMessagesRead: %d mismatch", batch.AfterDelayedCount, state.DelayedMessagesRead)
		}
	}
	// Check for MEL config events in this block.
	melConfig, err := lookupMELConfig(
		ctx,
		parentChainHeader,
		logsFetcher,
		eventUnpacker,
	)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to lookup MEL config event: %w", err)
	}
	if melConfig != nil {
		// MEL consensus getting activated for the first time
		if state.Version == 0 {
			if err := moveUnreadDelayedMessagesToInboxAcc(state, delayedMsgDatabase); err != nil {
				return nil, nil, nil, nil, err
			}
		}
		state.Version += 1
		state.DelayedMessagePostingTargetAddress = melConfig.Inbox
		state.BatchPostingTargetAddress = melConfig.SequencerInbox
	}
	return state, messages, delayedMessages, batchMetas, nil
}

func moveUnreadDelayedMessagesToInboxAcc(state *mel.State, delayedMsgDatabase DelayedMessageDatabase) error {
	var unreadDelayedMsgs []*mel.DelayedInboxMessage
	for i := state.DelayedMessagesRead; i < state.DelayedMessagesSeen; i++ {
		delayedMsg, err := delayedMsgDatabase.ReadDelayedMessage(state, i)
		if err != nil {
			return fmt.Errorf("failed creating delayed msg accumulators during MEL consensus activation: %w", err)
		}
		unreadDelayedMsgs = append(unreadDelayedMsgs, delayedMsg)
	}
	// Both the accumulators must be now empty
	if state.DelayedMessageInboxAcc != (common.Hash{}) || state.DelayedMessageOutboxAcc != (common.Hash{}) {
		return fmt.Errorf(
			"one of DelayedMessageInboxAcc: %v and DelayedMessageOutboxAcc: %v is non zero after reading all delayed msgs for MEL activation",
			state.DelayedMessageInboxAcc,
			state.DelayedMessageOutboxAcc,
		)
	}
	for _, delayedMsg := range unreadDelayedMsgs {
		if err := state.AccumulateDelayedMessage(delayedMsg); err != nil {
			return fmt.Errorf("failed creating delayed msg accumulators during MEL consensus activation: %w", err)
		}
	}
	return nil
}
