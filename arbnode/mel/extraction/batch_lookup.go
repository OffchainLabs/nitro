package melextraction

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

type EventUnpacker interface {
	UnpackLogTo(event any, abi *abi.ABI, eventName string, log types.Log) error
}

func ParseBatchesFromBlock(
	ctx context.Context,
	parentChainHeader *types.Header,
	txFetcher TransactionFetcher,
	logsFetcher LogsFetcher,
	eventUnpacker EventUnpacker,
) ([]*mel.SequencerInboxBatch, []*types.Transaction, error) {
	batches := make([]*mel.SequencerInboxBatch, 0)
	batchTxs := make([]*types.Transaction, 0)
	logs, err := logsFetcher.LogsForBlockHash(ctx, parentChainHeader.Hash())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch logs from parent chain block %v: %w", parentChainHeader.Hash(), err)
	}
	var lastSeqNum *uint64
	for _, log := range logs {
		if log == nil || log.Topics[0] != BatchDeliveredID {
			continue
		}
		event := new(bridgegen.SequencerInboxSequencerBatchDelivered)
		if err := eventUnpacker.UnpackLogTo(event, SeqInboxABI, "SequencerBatchDelivered", *log); err != nil {
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

		tx, err := txFetcher.TransactionByLog(ctx, log)
		if err != nil {
			return nil, nil, fmt.Errorf("error fetching tx by hash: %v in ParseBatchesFromBlock: %w ", log.TxHash, err)
		}

		batch := &mel.SequencerInboxBatch{
			BlockHash:              log.BlockHash,
			ParentChainBlockNumber: log.BlockNumber,
			SequenceNumber:         seqNum,
			BeforeInboxAcc:         event.BeforeAcc,
			AfterInboxAcc:          event.AfterAcc,
			AfterDelayedAcc:        event.DelayedAcc,
			AfterDelayedCount:      event.AfterDelayedMessagesRead.Uint64(),
			RawLog:                 *log,
			TimeBounds:             event.TimeBounds,
			DataLocation:           mel.BatchDataLocation(event.DataLocation),
			BridgeAddress:          log.Address,
		}
		batches = append(batches, batch)
		batchTxs = append(batchTxs, tx)
	}
	return batches, batchTxs, nil
}
