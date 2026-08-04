// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import "github.com/ethereum/go-ethereum/core/types"

// nextSequencerParentChainBlockNumber limits each sequenced L2 block to one
// parent-chain block of progress. HeaderReader subscribers may miss intermediate
// headers, especially when polling, but ArbOS associates the previous L2 block
// hash with each parent-chain block transition. Advancing one block at a time
// ensures every transition is recorded in order instead of leaving a stale
// ring-buffer slot behind.
func nextSequencerParentChainBlockNumber(observed uint64, lastL2Block *types.Header) uint64 {
	lastBlockInfo := types.DeserializeHeaderExtraInformation(lastL2Block)
	if lastBlockInfo.ArbOSFormatVersion == 0 {
		return observed
	}
	previous := lastBlockInfo.L1BlockNumber
	if observed > previous && observed-previous > 1 {
		return previous + 1
	}
	return observed
}
