package melextraction

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

func Test_serializeBatch(t *testing.T) {
	ctx := context.Background()
	t.Run("batch data retrieval fails", func(t *testing.T) {
		batch := &mel.SequencerInboxBatch{
			TimeBounds: bridgegen.IBridgeTimeBounds{
				MinTimestamp:   1,
				MaxTimestamp:   2,
				MinBlockNumber: 3,
				MaxBlockNumber: 4,
			},
			AfterDelayedCount: 1,
			DataLocation:      mel.BatchDataLocation(99),
		}
		_, err := serializeBatch(ctx, batch, nil, 0, nil)
		require.ErrorContains(t, err, "invalid data location")
	})
	t.Run("OK", func(t *testing.T) {
		txData := &types.BlobTx{
			To:    common.Address{},
			Nonce: 1,
			Gas:   1,
			Value: uint256.NewInt(0),
			BlobHashes: []common.Hash{
				common.BigToHash(big.NewInt(1)),
				common.BigToHash(big.NewInt(2)),
			},
		}
		tx := types.NewTx(txData)
		batch := &mel.SequencerInboxBatch{
			TimeBounds: bridgegen.IBridgeTimeBounds{
				MinTimestamp:   1,
				MaxTimestamp:   2,
				MinBlockNumber: 3,
				MaxBlockNumber: 4,
			},
			AfterDelayedCount: 1,
			DataLocation:      mel.BatchDataBlobHashes,
		}
		serialized, err := serializeBatch(ctx, batch, tx, 0, nil)
		require.NoError(t, err)
		// Serialization includes 5 uint64 values (8 bytes each) and the full batch
		// data appended at the end of the batch.
		// Our blob hashes serialization is 2 * 32 bytes + 1 byte prefix = 65 bytes
		// So the total size is 5 * 8 + 65 = 105 bytes.
		require.Equal(t, 105, len(serialized))

		// Expect some caching of serialized data.
		secondRound, err := serializeBatch(ctx, batch, tx, 0, nil)
		require.NoError(t, err)
		require.Equal(t, serialized, secondRound)
	})
}

func Test_getSequencerBatchData(t *testing.T) {
	ctx := context.Background()
	t.Run("invalid data location", func(t *testing.T) {
		_, err := getSequencerBatchData(
			ctx,
			&mel.SequencerInboxBatch{
				DataLocation: mel.BatchDataLocation(99),
			},
			nil,
			0,
			nil,
		)
		require.ErrorContains(t, err, "invalid data location")
	})
	t.Run("arbnode.BatchDataNone", func(t *testing.T) {
		data, err := getSequencerBatchData(
			ctx,
			&mel.SequencerInboxBatch{
				DataLocation: mel.BatchDataNone,
			},
			nil,
			0,
			nil,
		)
		require.NoError(t, err)
		require.Empty(t, data)
	})
	t.Run("arbnode.BatchDataBlobHashes", func(t *testing.T) {
		txData := &types.BlobTx{
			To:         common.Address{},
			Nonce:      1,
			Gas:        1,
			Value:      uint256.NewInt(0),
			BlobHashes: []common.Hash{},
		}
		tx := types.NewTx(txData)
		_, err := getSequencerBatchData(
			ctx,
			&mel.SequencerInboxBatch{
				DataLocation: mel.BatchDataBlobHashes,
			},
			tx,
			0,
			nil,
		)
		require.ErrorContains(t, err, "has no blobs")
		txData = &types.BlobTx{
			To:    common.Address{},
			Nonce: 1,
			Gas:   1,
			Value: uint256.NewInt(0),
			BlobHashes: []common.Hash{
				common.BigToHash(big.NewInt(1)),
				common.BigToHash(big.NewInt(2)),
			},
		}
		tx = types.NewTx(txData)
		data, err := getSequencerBatchData(
			ctx,
			&mel.SequencerInboxBatch{
				DataLocation: mel.BatchDataBlobHashes,
			},
			tx,
			0,
			nil,
		)
		require.NoError(t, err)
		require.Equal(t, 65, len(data)) // Includes a 1 byte prefix.
	})
	t.Run("arbnode.BatchDataTxInput", func(t *testing.T) {
		msgData := []byte("foobar")
		addSequencerL2BatchFromOriginCallABI := seqInboxABI.Methods["addSequencerL2BatchFromOrigin0"]
		seqNumber := big.NewInt(1)
		afterDelayedRead := big.NewInt(1)
		gasRefunder := common.Address{}
		prevMsgCount := big.NewInt(1)
		newMsgCount := big.NewInt(1)
		originTxData, err := addSequencerL2BatchFromOriginCallABI.Inputs.Pack(seqNumber, msgData, afterDelayedRead, gasRefunder, prevMsgCount, newMsgCount)
		require.NoError(t, err)
		fullTxData := make([]byte, 0)
		fullTxData = append(fullTxData, addSequencerL2BatchFromOriginCallABI.ID...)
		fullTxData = append(fullTxData, originTxData...)
		txData := &types.DynamicFeeTx{
			To:        nil,
			Nonce:     1,
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(1),
			Gas:       1,
			Value:     big.NewInt(0),
			Data:      nil,
		}
		tx := types.NewTx(txData)
		_, err = getSequencerBatchData(
			ctx,
			&mel.SequencerInboxBatch{
				DataLocation: mel.BatchDataTxInput,
			},
			tx,
			0,
			nil,
		)
		require.ErrorContains(t, err, "transaction data too short")
		txData = &types.DynamicFeeTx{
			To:        nil,
			Nonce:     1,
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(1),
			Gas:       1,
			Value:     big.NewInt(0),
			Data:      fullTxData,
		}
		tx = types.NewTx(txData)
		data, err := getSequencerBatchData(
			ctx,
			&mel.SequencerInboxBatch{
				DataLocation: mel.BatchDataTxInput,
			},
			tx,
			0,
			nil,
		)
		require.NoError(t, err)
		require.Equal(t, msgData, data)
	})
	t.Run("arbnode.BatchDataSeparateEvent", func(t *testing.T) {
		txData := &types.DynamicFeeTx{
			To:        nil,
			Nonce:     1,
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(1),
			Gas:       1,
			Value:     big.NewInt(0),
			Data:      nil,
		}
		tx := types.NewTx(txData)
		receipts := []*types.Receipt{
			{
				Logs: []*types.Log{},
			},
		}
		receiptFetcher := &mockReceiptFetcher{
			receipts: receipts,
			err:      errors.New("oops"),
		}
		_, err := getSequencerBatchData(
			ctx,
			&mel.SequencerInboxBatch{
				DataLocation: mel.BatchDataSeparateEvent,
			},
			tx,
			0,
			receiptFetcher,
		)
		require.ErrorContains(t, err, "oops")

		receiptFetcher = &mockReceiptFetcher{
			receipts: receipts,
			err:      nil,
		}
		_, err = getSequencerBatchData(
			ctx,
			&mel.SequencerInboxBatch{
				DataLocation: mel.BatchDataSeparateEvent,
			},
			tx,
			0,
			receiptFetcher,
		)
		require.ErrorContains(t, err, "no logs found")

		receipts = []*types.Receipt{
			{
				Logs: []*types.Log{
					{
						Topics: []common.Hash{{}},
					},
				},
			},
		}
		receiptFetcher = &mockReceiptFetcher{
			receipts: receipts,
			err:      nil,
		}
		_, err = getSequencerBatchData(
			ctx,
			&mel.SequencerInboxBatch{
				DataLocation: mel.BatchDataSeparateEvent,
			},
			tx,
			0,
			receiptFetcher,
		)
		require.ErrorContains(t, err, "expected to find sequencer batch data")

		sequencerBatchDataABI := seqInboxABI.Events["SequencerBatchData"].ID
		bridgeAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
		receipts = []*types.Receipt{
			{
				Logs: []*types.Log{
					{
						Address: bridgeAddr,
						Topics:  []common.Hash{sequencerBatchDataABI, {}},
					},
					{
						Address: bridgeAddr,
						Topics:  []common.Hash{sequencerBatchDataABI, {}},
					},
				},
			},
		}
		receiptFetcher = &mockReceiptFetcher{
			receipts: receipts,
			err:      nil,
		}
		_, err = getSequencerBatchData(
			ctx,
			&mel.SequencerInboxBatch{
				BridgeAddress: bridgeAddr,
				DataLocation:  mel.BatchDataSeparateEvent,
			},
			tx,
			0,
			receiptFetcher,
		)
		require.ErrorContains(t, err, "expected to find only one")

		event := &bridgegen.SequencerInboxSequencerBatchData{
			Data: []byte("foobar"),
		}
		eventABI := seqInboxABI.Events["SequencerBatchData"]
		packedLog, err := eventABI.Inputs.NonIndexed().Pack(
			event.Data,
		)
		require.NoError(t, err)
		receipts = []*types.Receipt{
			{
				Logs: []*types.Log{
					{
						Address: bridgeAddr,
						Topics:  []common.Hash{sequencerBatchDataABI, common.BigToHash(big.NewInt(1))},
						Data:    packedLog,
					},
				},
			},
		}
		receiptFetcher = &mockReceiptFetcher{
			receipts: receipts,
			err:      nil,
		}
		data, err := getSequencerBatchData(
			ctx,
			&mel.SequencerInboxBatch{
				SequenceNumber: 1,
				BridgeAddress:  bridgeAddr,
				DataLocation:   mel.BatchDataSeparateEvent,
			},
			tx,
			0,
			receiptFetcher,
		)
		require.NoError(t, err)
		require.Equal(t, event.Data, data)
	})
}
