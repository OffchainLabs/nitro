// Copyright 2026-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melreplay_test

import (
	"context"
	"math/big"
	"strings"
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

type mockBlockReader struct {
	blocks          map[common.Hash]*types.Block
	receiptByTxHash map[common.Hash]*types.Receipt
}

func (mbr *mockBlockReader) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	block, exists := mbr.blocks[hash]
	if !exists {
		return nil, nil
	}
	return block, nil
}

func (mbr *mockBlockReader) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	receipt, exists := mbr.receiptByTxHash[txHash]
	if !exists {
		return nil, nil
	}
	return receipt, nil
}

func TestRecordingOfTxPreimagesAndFetchingTxsFromPreimages(t *testing.T) {
	ctx := context.Background()
	toAddr := common.HexToAddress("0x0000000000000000000000000000000000DeaDBeef")
	blockHeader := &types.Header{}
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
	}
	blockBody := &types.Body{Transactions: txs}
	receipts := []*types.Receipt{}
	block := types.NewBlock(blockHeader, blockBody, receipts, trie.NewStackTrie(nil))
	blockReader := &mockBlockReader{
		blocks: map[common.Hash]*types.Block{
			block.Hash(): block,
		},
	}
	preimages := make(daprovider.PreimagesMap)
	recorder, err := melrecording.NewTransactionRecorder(blockReader, block.Hash(), preimages)
	require.NoError(t, err)
	require.NoError(t, recorder.Initialize(ctx))

	// Test recording of preimages
	recordStart := uint(9)
	recordEnd := uint(27)
	for i := recordStart; i <= recordEnd; i++ {
		tx, err := recorder.TransactionByLog(ctx, &types.Log{TxIndex: i})
		require.NoError(t, err)
		have, err := tx.MarshalJSON()
		require.NoError(t, err)
		want, err := block.Transactions()[i].MarshalJSON()
		require.NoError(t, err)
		require.Equal(t, want, have)
	}

	// Test reading of txs from the recorded preimages
	txsFetcher := melreplay.NewTransactionFetcher(
		block.Header(),
		melreplay.NewTypeBasedPreimageResolver(
			arbutil.Keccak256PreimageType,
			preimages,
		),
	)
	for i := recordStart; i <= recordEnd; i++ {
		tx, err := txsFetcher.TransactionByLog(ctx, &types.Log{TxIndex: i})
		require.NoError(t, err)
		have, err := tx.MarshalJSON()
		require.NoError(t, err)
		want, err := block.Transactions()[i].MarshalJSON()
		require.NoError(t, err)
		require.Equal(t, want, have)
	}

	// Tx fetching should fail for not recorded ones
	_, err = txsFetcher.TransactionByLog(ctx, &types.Log{TxIndex: recordStart - 1})
	if err == nil || !strings.Contains(err.Error(), "preimage not found for hash") {
		t.Fatalf("failed with unexpected error: %v", err)
	}
}
