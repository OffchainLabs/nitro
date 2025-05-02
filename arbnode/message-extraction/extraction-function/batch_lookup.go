package extractionfunction

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbnode"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
)

func parseBatchesFromBlock(
	ctx context.Context,
	melState *meltypes.State,
	block *types.Block,
	eventParser BatchEventParser,
	receiptFetcher ReceiptFetcher,
	batchDeliveredEventID common.Hash,
) ([]*arbnode.SequencerInboxBatch, error) {
	allBatches := make([]*arbnode.SequencerInboxBatch, 0)
	for _, tx := range block.Transactions() {
		if tx.To() == nil {
			continue
		}
		if *tx.To() != melState.BatchPostingTargetAddress {
			continue
		}
		// Fetch the receipts for the transaction to get the logs.
		receipt, err := receiptFetcher.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			return nil, err
		}
		if len(receipt.Logs) == 0 {
			continue
		}
		batches := make([]*arbnode.SequencerInboxBatch, 0, len(receipt.Logs))
		var lastSeqNum *uint64
		for _, log := range receipt.Logs {
			if log.Topics[0] != batchDeliveredEventID {
				continue
			}
			parsedLog, err := eventParser.ParseSequencerBatchDelivered(*log)
			if err != nil {
				return nil, err
			}
			if !parsedLog.BatchSequenceNumber.IsUint64() {
				return nil, errors.New("sequencer inbox event has non-uint64 sequence number")
			}
			if !parsedLog.AfterDelayedMessagesRead.IsUint64() {
				return nil, errors.New("sequencer inbox event has non-uint64 delayed messages read")
			}

			seqNum := parsedLog.BatchSequenceNumber.Uint64()
			if lastSeqNum != nil {
				if seqNum != *lastSeqNum+1 {
					return nil, fmt.Errorf("sequencer batches out of order; after batch %v got batch %v", *lastSeqNum, seqNum)
				}
			}
			lastSeqNum = &seqNum
			batch := &arbnode.SequencerInboxBatch{
				BlockHash:              log.BlockHash,
				ParentChainBlockNumber: log.BlockNumber,
				SequenceNumber:         seqNum,
				BeforeInboxAcc:         parsedLog.BeforeAcc,
				AfterInboxAcc:          parsedLog.AfterAcc,
				AfterDelayedAcc:        parsedLog.DelayedAcc,
				AfterDelayedCount:      parsedLog.AfterDelayedMessagesRead.Uint64(),
				RawLog:                 *log,
				TimeBounds:             parsedLog.TimeBounds,
				DataLocation:           arbnode.BatchDataLocation(parsedLog.DataLocation),
				BridgeAddress:          log.Address,
			}
			batches = append(batches, batch)
		}
		allBatches = append(allBatches, batches...)
	}
	return allBatches, nil
}
