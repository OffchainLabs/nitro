// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbutil

type MessageIndex uint64

func BlockNumberToMessageCount(blockNumber uint64, genesisBlockNumber uint64) MessageIndex {
	return MessageIndex(blockNumber + 1 - genesisBlockNumber)
}

// Block number must correspond to a message count, meaning it may not be less than -1
func SignedBlockNumberToMessageCount(blockNumber int64, genesisBlockNumber uint64) MessageIndex {
	return MessageIndex(uint64(blockNumber+1) - genesisBlockNumber)
}

func MessageCountToBlockNumber(messageCount MessageIndex, genesisBlockNumber uint64) int64 {
	return int64(uint64(messageCount)+genesisBlockNumber) - 1
}
