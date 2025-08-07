// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	melextraction "github.com/offchainlabs/nitro/arbnode/mel/extraction"
)

var _ = melextraction.MELDataProvider(&melDataProvider{})

type melDataProvider struct {
	*delayedMessageDatabase
	resolver preimageResolver
}

func (md *melDataProvider) LogsForBlockHash(
	ctx context.Context,
	parentChainBlockHash common.Hash,
) ([]*types.Log, error) {
	// TODO: Simply fetch all the receipts like we do with transactions
	// and extract logs from them into a flat slice.
	return nil, errors.New("unimplemented")
}

func (md *melDataProvider) LogsForTxIndex(
	ctx context.Context,
	parentChainBlockHash common.Hash,
	txIndex uint,
) ([]*types.Log, error) {
	header := getBlockHeaderByHash(parentChainBlockHash)
	recFetcher := &receiptFetcherForBlock{
		header:           header,
		preimageResolver: md.resolver,
	}
	receipt, err := recFetcher.ReceiptForTransactionIndex(ctx, txIndex)
	if err != nil {
		return nil, err
	}
	return receipt.Logs, nil
}

func (md *melDataProvider) TransactionByLog(
	ctx context.Context,
	log *types.Log,
) (*types.Transaction, error) {
	header := getBlockHeaderByHash(log.BlockHash)
	txFetcher := &txsFetcherForBlock{
		header:           header,
		preimageResolver: md.resolver,
	}
	// TODO: Inefficient, instead walk the txs trie like we do with receipts to find a specific index.
	txs, err := txFetcher.TransactionsByHeader(ctx, log.BlockHash)
	if err != nil {
		return nil, err
	}
	for _, tx := range txs {
		if tx.Hash() == log.TxHash {
			return tx, nil
		}
	}
	return nil, errors.New("transaction not found for log")
}
