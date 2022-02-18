//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbutil

func BlockNumberToMessageCount(blockNumber uint64, genesisBlockNumber uint64) uint64 {
	return blockNumber + 1 - genesisBlockNumber
}

// Block number must correspond to a message count, meaning it may not be less than -1
func SignedBlockNumberToMessageCount(blockNumber int64, genesisBlockNumber uint64) uint64 {
	return uint64(blockNumber+1) - genesisBlockNumber
}

func MessageCountToBlockNumber(messageCount uint64, genesisBlockNumber uint64) int64 {
	return int64(messageCount+genesisBlockNumber) - 1
}
