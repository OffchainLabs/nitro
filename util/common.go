package util

import (
	"fmt"

	"github.com/offchainlabs/nitro/arbutil"
)

func ArrayToSet[T comparable](arr []T) map[T]struct{} {
	ret := make(map[T]struct{})
	for _, elem := range arr {
		ret[elem] = struct{}{}
	}
	return ret
}

func BlockNumberToMessageIndex(blockNum, genesis uint64) (arbutil.MessageIndex, error) {
	if blockNum < genesis {
		return 0, fmt.Errorf("blockNum %d < genesis %d", blockNum, genesis)
	}
	return arbutil.MessageIndex(blockNum - genesis), nil
}

func MessageIndexToBlockNumber(msgIdx arbutil.MessageIndex, genesis uint64) uint64 {
	return uint64(msgIdx) + genesis
}
