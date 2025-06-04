package extractionfunction

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"

	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

type eventUnpacker interface {
	unpackLogTo(event any, abi *abi.ABI, eventName string, log types.Log) error
}

func parseBatchesFromBlock(
	ctx context.Context,
	melState *meltypes.State,
	parentChainBlock *types.Block,
	receiptFetcher ReceiptFetcher,
	eventUnpacker eventUnpacker,
) ([]*meltypes.SequencerInboxBatch, []*types.Transaction, []uint, error) {
	allBatches := make([]*meltypes.SequencerInboxBatch, 0)
	allBatchTxs := make([]*types.Transaction, 0)
	allBatchTxIndices := make([]uint, 0)
	for i, tx := range parentChainBlock.Transactions() {
		if tx.To() == nil {
			continue
		}
		if *tx.To() != melState.BatchPostingTargetAddress {
			continue
		}
		// Fetch the receipts for the transaction to get the logs.
		txIndex := uint(i) // #nosec G115
		receipt, err := receiptFetcher.ReceiptForTransactionIndex(ctx, txIndex)
		if err != nil {
			return nil, nil, nil, err
		}
		if len(receipt.Logs) == 0 {
			continue
		}
		batches := make([]*meltypes.SequencerInboxBatch, 0, len(receipt.Logs))
		txs := make([]*types.Transaction, 0, len(receipt.Logs))
		txIndices := make([]uint, 0, len(receipt.Logs))
		var lastSeqNum *uint64
		for _, log := range receipt.Logs {
			if log == nil || log.Topics[0] != batchDeliveredID {
				continue
			}
			event := new(bridgegen.SequencerInboxSequencerBatchDelivered)
			if err := eventUnpacker.unpackLogTo(event, seqInboxABI, "SequencerBatchDelivered", *log); err != nil {
				return nil, nil, nil, err
			}
			if !event.BatchSequenceNumber.IsUint64() {
				return nil, nil, nil, errors.New("sequencer inbox event has non-uint64 sequence number")
			}
			if !event.AfterDelayedMessagesRead.IsUint64() {
				return nil, nil, nil, errors.New("sequencer inbox event has non-uint64 delayed messages read")
			}

			seqNum := event.BatchSequenceNumber.Uint64()
			if lastSeqNum != nil {
				if seqNum != *lastSeqNum+1 {
					return nil, nil, nil, fmt.Errorf("sequencer batches out of order; after batch %v got batch %v", *lastSeqNum, seqNum)
				}
			}
			lastSeqNum = &seqNum
			batch := &meltypes.SequencerInboxBatch{
				BlockHash:              log.BlockHash,
				ParentChainBlockNumber: log.BlockNumber,
				SequenceNumber:         seqNum,
				BeforeInboxAcc:         event.BeforeAcc,
				AfterInboxAcc:          event.AfterAcc,
				AfterDelayedAcc:        event.DelayedAcc,
				AfterDelayedCount:      event.AfterDelayedMessagesRead.Uint64(),
				RawLog:                 *log,
				TimeBounds:             event.TimeBounds,
				DataLocation:           meltypes.BatchDataLocation(event.DataLocation),
				BridgeAddress:          log.Address,
			}
			batches = append(batches, batch)
			txs = append(txs, tx)
			txIndices = append(txIndices, uint(i)) // #nosec G115
		}
		allBatches = append(allBatches, batches...)
		allBatchTxs = append(allBatchTxs, txs...)
		allBatchTxIndices = append(allBatchTxIndices, txIndices...)
	}
	return allBatches, allBatchTxs, allBatchTxIndices, nil
}
