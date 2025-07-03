package melextraction

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

func serializeBatch(
	ctx context.Context,
	batch *mel.SequencerInboxBatch,
	tx *types.Transaction,
	txIndex uint,
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
		tx,
		txIndex,
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
	batch *mel.SequencerInboxBatch,
	tx *types.Transaction,
	txIndex uint,
	receiptFetcher ReceiptFetcher,
) ([]byte, error) {
	addSequencerL2BatchFromOriginCallABI := seqInboxABI.Methods["addSequencerL2BatchFromOrigin0"]
	switch batch.DataLocation {
	case mel.BatchDataTxInput:
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
	case mel.BatchDataSeparateEvent:
		sequencerBatchDataABI := seqInboxABI.Events["SequencerBatchData"].ID
		var numberAsHash common.Hash
		// we want to convert a batch sequencer number which is a uint64 into a big-endian byte slice of size 32,
		// so the last 8 bytes of that slice will contain the serialized batch.SequenceNumber
		binary.BigEndian.PutUint64(numberAsHash[(32-8):], batch.SequenceNumber)
		receipt, err := receiptFetcher.ReceiptForTransactionIndex(ctx, txIndex)
		if err != nil {
			return nil, err
		}
		if len(receipt.Logs) == 0 {
			return nil, errors.New("no logs found in transaction receipt")
		}
		topics := [][]common.Hash{{sequencerBatchDataABI}, {numberAsHash}}
		filteredLogs := types.FilterLogs(receipt.Logs, nil, nil, []common.Address{batch.BridgeAddress}, topics)
		if len(filteredLogs) == 0 {
			return nil, errors.New("expected to find sequencer batch data")
		}
		if len(filteredLogs) > 1 {
			return nil, errors.New("expected to find only one matching sequencer batch data")
		}
		event := new(bridgegen.SequencerInboxSequencerBatchData)
		err = seqInboxABI.UnpackIntoInterface(event, "SequencerBatchData", filteredLogs[0].Data)
		if err != nil {
			return nil, err
		}
		return event.Data, nil
	case mel.BatchDataNone:
		// No data when in a force inclusion batch
		return nil, nil
	case mel.BatchDataBlobHashes:
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
