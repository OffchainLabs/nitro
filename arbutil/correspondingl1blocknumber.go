// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbutil

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

func CorrespondingL1BlockNumber(ctx context.Context, client L1Interface, blockNumber uint64) (uint64, error) {
	header, err := client.HeaderByNumber(ctx, big.NewInt(int64(blockNumber)))
	if err != nil {
		return 0, fmt.Errorf("error getting L1 block number %d header : %w", blockNumber, err)
	}
	headerInfo := types.DeserializeHeaderExtraInformation(header)
	if headerInfo.ArbOSFormatVersion > 0 {
		return headerInfo.L1BlockNumber, nil
	}
	return blockNumber, nil
}
