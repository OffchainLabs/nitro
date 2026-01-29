// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package melreplay_test

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/offchainlabs/nitro/arbnode/mel/recording"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/mel-replay"
)

func TestRecordingOfReceiptPreimagesAndFetchingLogsFromPreimages(t *testing.T) {
	ctx := context.Background()
	blockReader := &mockBlockReader{
		blocks:          make(map[common.Hash]*types.Block),
		receiptByTxHash: map[common.Hash]*types.Receipt{},
	}
	toAddr := common.HexToAddress("0x0000000000000000000000000000000000DeaDBeef")
	blockHeader := &types.Header{}
	receipts := []*types.Receipt{}
	txs := make([]*types.Transaction, 0)
	numTxs := uint64(50)
	for i := range numTxs {
		txData := &types.DynamicFeeTx{
			To:        &toAddr,
			Nonce:     i,
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(1),
			Gas:       1,
			Value:     big.NewInt(0),
			Data:      nil,
		}
		tx := types.NewTx(txData)
		txs = append(txs, tx)
		receipt := &types.Receipt{
			TxHash:           tx.Hash(),
			TransactionIndex: uint(i),
			Type:             types.DynamicFeeTxType,
			Logs: []*types.Log{
				{
					// Consensus fields:
					Address: common.HexToAddress(fmt.Sprintf("%d", i)),
					Topics:  []common.Hash{common.HexToHash("topic1"), common.HexToHash("topic2")},
					Data:    common.Hex2Bytes(fmt.Sprintf("%d", i)),

					// Derived Fields:
					TxIndex: uint(i),
				},
			},
		}
		receipts = append(receipts, receipt)
		blockReader.receiptByTxHash[tx.Hash()] = receipt
	}
	blockBody := &types.Body{Transactions: txs}
	block := types.NewBlock(blockHeader, blockBody, receipts, trie.NewStackTrie(nil))
	blockReader.blocks[block.Hash()] = block
	// Fill in blockHash and BlockNumber fields of the logs
	for _, receipt := range receipts {
		for _, log := range receipt.Logs {
			log.BlockHash = block.Hash()
			log.BlockNumber = block.NumberU64()
		}
	}
	preimages := make(daprovider.PreimagesMap)
	recordedLogsFetcher, err := melrecording.RecordReceipts(ctx, blockReader, block.Hash(), preimages)
	require.NoError(t, err)

	// Test recording of preimages
	for i := range numTxs {
		logs, err := recordedLogsFetcher.LogsForTxIndex(ctx, block.Hash(), uint(i))
		require.NoError(t, err)
		have, err := logs[0].MarshalJSON()
		require.NoError(t, err)
		want, err := receipts[i].Logs[0].MarshalJSON()
		require.NoError(t, err)
		require.Equal(t, want, have)
	}

	// Test reading of logs from the recorded preimages
	receiptFetcher := melreplay.NewLogsFetcher(
		block.Header(),
		melreplay.NewTypeBasedPreimageResolver(
			arbutil.Keccak256PreimageType,
			preimages,
		),
	)
	// Test LogsForBlockHash
	logs, err := receiptFetcher.LogsForBlockHash(ctx, block.Hash())
	require.NoError(t, err)
	// #nosec G115
	if len(logs) != int(numTxs) {
		t.Fatalf("number of logs from LogsForBlockHash mismatch. Want: %d, Got: %d", numTxs, len(logs))
	}
	for _, log := range logs {
		have, err := log.MarshalJSON()
		require.NoError(t, err)
		want, err := receipts[log.TxIndex].Logs[0].MarshalJSON()
		require.NoError(t, err)
		require.Equal(t, want, have)
	}
	// Test LogsForTxIndex
	for i := range numTxs {
		logs, err := receiptFetcher.LogsForTxIndex(ctx, block.Hash(), uint(i))
		require.NoError(t, err)
		have, err := logs[0].MarshalJSON()
		require.NoError(t, err)
		want, err := receipts[i].Logs[0].MarshalJSON()
		require.NoError(t, err)
		require.Equal(t, want, have)
	}
}
