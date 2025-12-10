// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbutil

import "fmt"

// messages are 0-indexed
type MessageIndex uint64

func BlockNumberToMessageIndex(blockNum, genesis uint64) (MessageIndex, error) {
	if blockNum < genesis {
		return 0, fmt.Errorf("blockNum %d < genesis %d", blockNum, genesis)
	}
	return MessageIndex(blockNum - genesis), nil
}

func MessageIndexToBlockNumber(msgIdx MessageIndex, genesis uint64) uint64 {
	return uint64(msgIdx) + genesis
}
