package extractionfunction

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/filters"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

func serializeBatch(
	ctx context.Context,
	batch *arbnode.SequencerInboxBatch,
	parentChainBlock *types.Block,
	tx *types.Transaction,
	txIndex uint,
	seqInboxAbi *abi.ABI,
	receiptFetcher ReceiptFetcher,
) ([]byte, error) {
	if batch.Serialized != nil {
		return batch.Serialized, nil
	}

	var fullData []byte

	// Serialize the header
	headerVals := []uint64{
		batch.TimeBounds.MinTimestamp,
		batch.TimeBounds.MaxTimestamp,
		batch.TimeBounds.MinBlockNumber,
		batch.TimeBounds.MaxBlockNumber,
		batch.AfterDelayedCount,
	}
	for _, bound := range headerVals {
		var intData [8]byte
		binary.BigEndian.PutUint64(intData[:], bound)
		fullData = append(fullData, intData[:]...)
	}

	// Append the batch data
	data, err := getSequencerBatchData(
		ctx,
		batch,
		parentChainBlock,
		tx,
		txIndex,
		seqInboxAbi,
		receiptFetcher,
	)
	if err != nil {
		return nil, err
	}
	fullData = append(fullData, data...)

	batch.Serialized = fullData
	return fullData, nil
}

func getSequencerBatchData(
	ctx context.Context,
	batch *arbnode.SequencerInboxBatch,
	parentChainBlock *types.Block,
	tx *types.Transaction,
	txIndex uint,
	seqInboxAbi *abi.ABI,
	receiptFetcher ReceiptFetcher,
) ([]byte, error) {
	addSequencerL2BatchFromOriginCallABI := seqInboxAbi.Methods["addSequencerL2BatchFromOrigin0"]
	switch batch.DataLocation {
	case arbnode.BatchDataTxInput:
		data := tx.Data()
		if len(data) < 4 {
			return nil, errors.New("transaction data too short")
		}
		args := make(map[string]interface{})
		if err := addSequencerL2BatchFromOriginCallABI.Inputs.UnpackIntoMap(args, data[4:]); err != nil {
			return nil, err
		}
		dataBytes, ok := args["data"].([]byte)
		if !ok {
			return nil, errors.New("args[\"data\"] not a byte array")
		}
		return dataBytes, nil
	case arbnode.BatchDataSeparateEvent:
		sequencerBatchDataABI := seqInboxAbi.Events["SequencerBatchData"].ID
		var numberAsHash common.Hash
		binary.BigEndian.PutUint64(numberAsHash[(32-8):], batch.SequenceNumber)
		receipt, err := receiptFetcher.ReceiptForTransactionIndex(ctx, parentChainBlock, txIndex)
		if err != nil {
			return nil, err
		}
		if len(receipt.Logs) == 0 {
			return nil, errors.New("no logs found in transaction receipt")
		}
		topics := [][]common.Hash{{sequencerBatchDataABI}, {numberAsHash}}
		filteredLogs := filters.FilterLogs(receipt.Logs, nil, nil, []common.Address{batch.BridgeAddress}, topics)
		if len(filteredLogs) == 0 {
			return nil, errors.New("expected to find sequencer batch data")
		}
		if len(filteredLogs) > 1 {
			return nil, errors.New("expected to find only one matching sequencer batch data")
		}
		event := new(bridgegen.SequencerInboxSequencerBatchData)
		err = seqInboxAbi.UnpackIntoInterface(event, "SequencerBatchData", filteredLogs[0].Data)
		if err != nil {
			return nil, err
		}
		return event.Data, nil
	case arbnode.BatchDataNone:
		// No data when in a force inclusion batch
		return nil, nil
	case arbnode.BatchDataBlobHashes:
		if len(tx.BlobHashes()) == 0 {
			return nil, fmt.Errorf("blob batch transaction %v has no blobs", tx.Hash())
		}
		data := []byte{daprovider.BlobHashesHeaderFlag}
		for _, h := range tx.BlobHashes() {
			data = append(data, h[:]...)
		}
		return data, nil
	default:
		return nil, fmt.Errorf("batch has invalid data location %v", batch.DataLocation)
	}
}
