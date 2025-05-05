package extractionfunction

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbnode"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

type BatchLookupParams struct {
	BatchDeliveredEventID common.Hash
	SequencerInboxABI     *abi.ABI
}

func parseBatchesFromBlock(
	ctx context.Context,
	melState *meltypes.State,
	parentChainBlock *types.Block,
	receiptFetcher ReceiptFetcher,
	params *BatchLookupParams,
) ([]*arbnode.SequencerInboxBatch, []*types.Transaction, error) {
	allBatches := make([]*arbnode.SequencerInboxBatch, 0)
	allBatchTxs := make([]*types.Transaction, 0)
	for i, tx := range parentChainBlock.Transactions() {
		if tx.To() == nil {
			continue
		}
		if *tx.To() != melState.BatchPostingTargetAddress {
			continue
		}
		// Fetch the receipts for the transaction to get the logs.
		txIndex := uint64(i)
		receipt, err := receiptFetcher.ReceiptForTransactionIndex(ctx, parentChainBlock, txIndex)
		if err != nil {
			return nil, nil, err
		}
		if len(receipt.Logs) == 0 {
			continue
		}
		batches := make([]*arbnode.SequencerInboxBatch, 0, len(receipt.Logs))
		txs := make([]*types.Transaction, 0, len(receipt.Logs))
		var lastSeqNum *uint64
		for _, log := range receipt.Logs {
			if log == nil {
				continue
			}
			if log.Topics[0] != params.BatchDeliveredEventID {
				continue
			}
			event := new(bridgegen.SequencerInboxSequencerBatchDelivered)
			if err := unpackLogTo(event, params.SequencerInboxABI, "SequencerBatchDelivered", *log); err != nil {
				return nil, nil, err
			}
			if !event.BatchSequenceNumber.IsUint64() {
				return nil, nil, errors.New("sequencer inbox event has non-uint64 sequence number")
			}
			if !event.AfterDelayedMessagesRead.IsUint64() {
				return nil, nil, errors.New("sequencer inbox event has non-uint64 delayed messages read")
			}

			seqNum := event.BatchSequenceNumber.Uint64()
			if lastSeqNum != nil {
				if seqNum != *lastSeqNum+1 {
					return nil, nil, fmt.Errorf("sequencer batches out of order; after batch %v got batch %v", *lastSeqNum, seqNum)
				}
			}
			lastSeqNum = &seqNum
			batch := &arbnode.SequencerInboxBatch{
				BlockHash:              log.BlockHash,
				ParentChainBlockNumber: log.BlockNumber,
				SequenceNumber:         seqNum,
				BeforeInboxAcc:         event.BeforeAcc,
				AfterInboxAcc:          event.AfterAcc,
				AfterDelayedAcc:        event.DelayedAcc,
				AfterDelayedCount:      event.AfterDelayedMessagesRead.Uint64(),
				RawLog:                 *log,
				TimeBounds:             event.TimeBounds,
				DataLocation:           arbnode.BatchDataLocation(event.DataLocation),
				BridgeAddress:          log.Address,
			}
			batches = append(batches, batch)
			txs = append(txs, tx)
		}
		allBatches = append(allBatches, batches...)
		allBatchTxs = append(allBatchTxs, txs...)
	}
	return allBatches, allBatchTxs, nil
}
