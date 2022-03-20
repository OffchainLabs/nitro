//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"context"
	"errors"
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
	if header == nil {
		return false, errors.New("header not found")
	}
	return a.val.ValidateBlock(ctx, header)
}
