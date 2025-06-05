package main

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type txsFetcherForBlock struct {
	header           *types.Header
	preimageResolver preimageResolver
}

func (tf *txsFetcherForBlock) TransactionsByHeader(
	ctx context.Context,
	parentChainHeaderHash common.Hash,
) (types.Transactions, error) {
	return nil, nil
}
