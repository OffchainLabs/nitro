// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbutil

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

func ParentHeaderToL1BlockNumber(header *types.Header) uint64 {
	headerInfo := types.DeserializeHeaderExtraInformation(header)
	if headerInfo.ArbOSFormatVersion > 0 {
		return headerInfo.L1BlockNumber
	}
	return header.Number.Uint64()
}

func CorrespondingL1BlockNumber(ctx context.Context, client L1Interface, parentBlockNumber uint64) (uint64, error) {
	// #nosec G115
	header, err := client.HeaderByNumber(ctx, big.NewInt(int64(parentBlockNumber)))
	if err != nil {
		return 0, fmt.Errorf("error getting L1 block number %d header : %w", parentBlockNumber, err)
	}
	return ParentHeaderToL1BlockNumber(header), nil
}
