// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"

	melrecording "github.com/offchainlabs/nitro/arbnode/mel/recording"
	"github.com/offchainlabs/nitro/arbutil"
)

type mockPreimageResolver struct {
	preimages map[common.Hash][]byte
}

func (m *mockPreimageResolver) ResolveTypedPreimage(preimageType arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	if preimage, exists := m.preimages[hash]; exists {
		return preimage, nil
	}
	return nil, fmt.Errorf("preimage not found for hash: %s", hash.Hex())
}

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
	for i := range uint64(50) {
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
	recorder := melrecording.NewReceiptRecorder(blockReader, block.Hash())
	require.NoError(t, recorder.Initialize(ctx))

	// Test recording of preimages
	recordStart := uint(6)
	recordEnd := uint(20)
	for i := recordStart; i <= recordEnd; i++ {
		logs, err := recorder.LogsForTxIndex(ctx, block.Hash(), i)
		require.NoError(t, err)
		have, err := logs[0].MarshalJSON()
		require.NoError(t, err)
		want, err := receipts[i].Logs[0].MarshalJSON()
		require.NoError(t, err)
		require.Equal(t, want, have)
	}

	// Test reading of logs from the recorded preimages
	preimages, err := recorder.GetPreimages()
	require.NoError(t, err)
	receiptFetcher := &receiptFetcherForBlock{
		header: block.Header(),
		preimageResolver: &testPreimageResolver{
			preimages: preimages[arbutil.Keccak256PreimageType],
		},
	}
	// Test LogsForBlockHash
	logs, err := receiptFetcher.LogsForBlockHash(ctx, block.Hash())
	require.NoError(t, err)
	// #nosec G115
	if len(logs) != int(recordEnd-recordStart+1) {
		t.Fatalf("number of logs from LogsForBlockHash mismatch. Want: %d, Got: %d", recordEnd-recordStart+1, len(logs))
	}
	for _, log := range logs {
		have, err := log.MarshalJSON()
		require.NoError(t, err)
		want, err := receipts[log.TxIndex].Logs[0].MarshalJSON()
		require.NoError(t, err)
		require.Equal(t, want, have)
	}
	// Test LogsForTxIndex
	for i := recordStart; i <= recordEnd; i++ {
		logs, err := receiptFetcher.LogsForTxIndex(ctx, block.Hash(), i)
		require.NoError(t, err)
		have, err := logs[0].MarshalJSON()
		require.NoError(t, err)
		want, err := receipts[i].Logs[0].MarshalJSON()
		require.NoError(t, err)
		require.Equal(t, want, have)
	}

	// Logs fetching should fail for not recorded ones
	_, err = receiptFetcher.LogsForTxIndex(ctx, block.Hash(), recordStart-1)
	if err == nil || !strings.Contains(err.Error(), "preimage not found for hash") {
		t.Fatalf("failed with unexpected error: %v", err)
	}
}
