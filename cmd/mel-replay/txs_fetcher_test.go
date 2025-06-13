package main

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestFetchTransactionsForBlockHeader_LegacyTxs(t *testing.T) {
	ctx := context.Background()
	total := 1
	txes := make([]*types.Transaction, total)
	for i := 0; i < total; i++ {
		txes[i] = types.NewTransaction(uint64(i), common.Address{}, big.NewInt(0), 21000, big.NewInt(1), nil)
	}
	hasher := newRecordingHasher()
	txsRoot := types.DeriveSha(types.Transactions(txes), hasher)
	header := &types.Header{
		TxHash: txsRoot,
	}
	preimages := hasher.GetPreimages()
	mockPreimageResolver := &mockPreimageResolver{
		preimages: preimages,
	}
	txsFetcher := &txsFetcherForBlock{
		header:           header,
		preimageResolver: mockPreimageResolver,
	}
	fetched, err := txsFetcher.TransactionsByHeader(ctx, header.Hash())
	require.NoError(t, err)
	require.Len(t, fetched, total)
	for i, tx := range fetched {
		require.Equal(t, txes[i].Hash(), tx.Hash())
		require.Equal(t, uint64(i), tx.Nonce())
	}
}

func TestFetchTransactionsForBlockHeader(t *testing.T) {
	ctx := context.Background()
	total := 1
	txes := make([]*types.Transaction, total)
	for i := 0; i < total; i++ {
		txData := types.DynamicFeeTx{
			Nonce:     uint64(i),
			To:        nil,
			Gas:       21000,
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(0),
			Value:     big.NewInt(0),
			Data:      nil,
		}
		txes[i] = types.NewTx(&txData)
	}
	hasher := newRecordingHasher()
	txsRoot := types.DeriveSha(types.Transactions(txes), hasher)
	header := &types.Header{
		TxHash: txsRoot,
	}
	preimages := hasher.GetPreimages()
	mockPreimageResolver := &mockPreimageResolver{
		preimages: preimages,
	}
	txsFetcher := &txsFetcherForBlock{
		header:           header,
		preimageResolver: mockPreimageResolver,
	}
	fetched, err := txsFetcher.TransactionsByHeader(ctx, header.Hash())
	require.NoError(t, err)
	require.Len(t, fetched, total)
	for i, tx := range fetched {
		require.Equal(t, txes[i].Hash(), tx.Hash())
		require.Equal(t, uint64(i), tx.Nonce())
	}
}
