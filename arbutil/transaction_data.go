// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbutil

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// GetLogEmitterTxData requires that the tx's data is at least 4 bytes long
func GetLogEmitterTxData(ctx context.Context, client L1Interface, log types.Log) ([]byte, error) {
	tx, err := client.TransactionInBlock(ctx, log.BlockHash, log.TxIndex)
	if err != nil {
		return nil, err
	}
	if tx.Hash() != log.TxHash {
		return nil, fmt.Errorf("L1 client returned unexpected transaction hash %v when looking up block %v transaction %v with expected hash %v", tx.Hash(), log.BlockHash, log.TxIndex, log.TxHash)
	}
	if len(tx.Data()) < 4 {
		return nil, fmt.Errorf("log emitting transaction %v unexpectedly does not have enough data", tx.Hash())
	}
	return tx.Data(), nil
}

func BlockTransactions(ctx context.Context, l1Client L1Interface, hash common.Hash) (types.Transactions, error) {
	b, err := l1Client.BlockByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("getting block by hash: %w", err)
	}
	return b.Transactions(), nil
}
