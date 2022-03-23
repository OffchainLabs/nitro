// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/validator"
)

type BlockValidatorAPI struct {
	val        *validator.BlockValidator
	blockchain *core.BlockChain
}

func (a *BlockValidatorAPI) RevalidateBlock(ctx context.Context, blockNum rpc.BlockNumberOrHash) (bool, error) {
	header, err := arbitrum.HeaderByNumberOrHash(a.blockchain, blockNum)
	if err != nil {
		return false, err
	}
	return a.val.ValidateBlock(ctx, header)
}
