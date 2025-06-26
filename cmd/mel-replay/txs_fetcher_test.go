package main

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestFetchTransactionsForBlockHeader_DynamicFeeTxs(t *testing.T) {
	ctx := context.Background()
	total := uint64(42)
	txes := make([]*types.Transaction, total)
	for i := uint64(0); i < total; i++ {
		txData := types.DynamicFeeTx{
			Nonce:     i,
			To:        nil,
			Gas:       21000,
			GasTipCap: big.NewInt(1),
			GasFeeCap: big.NewInt(1),
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
	require.True(t, uint64(len(fetched)) == total) // #nosec G115
	for i, tx := range fetched {
		require.Equal(t, txes[i].Hash(), tx.Hash())
		require.Equal(t, uint64(i), tx.Nonce()) // #nosec G115
	}
}

func TestFetchTransactionsForBlockHeader_LegacyTxs(t *testing.T) {
	ctx := context.Background()
	total := uint64(42)
	txes := make([]*types.Transaction, total)
	for i := uint64(0); i < total; i++ {
		txes[i] = types.NewTransaction(i, common.Address{}, big.NewInt(0), 21000, big.NewInt(1), nil)
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
	require.True(t, uint64(len(fetched)) == total) // #nosec G115
	for i, tx := range fetched {
		require.Equal(t, txes[i].Hash(), tx.Hash())
		require.Equal(t, uint64(i), tx.Nonce()) // #nosec G115
	}
}
