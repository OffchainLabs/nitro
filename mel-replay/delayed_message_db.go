// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melreplay

import (
	"fmt"
	"math/bits"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbnode/mel/extraction"
)

type delayedMessageDatabase struct {
	preimageResolver PreimageResolver
}

func NewDelayedMessageDatabase(preimageResolver PreimageResolver) melextraction.DelayedMessageDatabase {
	return &delayedMessageDatabase{preimageResolver}
}

func (d *delayedMessageDatabase) ReadDelayedMessage(
	state *mel.State,
	msgIndex uint64,
) (*mel.DelayedInboxMessage, error) {
	if msgIndex >= state.DelayedMessagesSeen {
		return nil, fmt.Errorf("index %d out of range, total delayed messages seen: %d", msgIndex, state.DelayedMessagesSeen)
	}
	treeSize := NextPowerOfTwo(state.DelayedMessagesSeen)
	merkleDepth := bits.TrailingZeros64(treeSize)
	return fetchObjectFromMerkleTree[mel.DelayedInboxMessage](state.DelayedMessagesSeenRoot, merkleDepth, msgIndex, d.preimageResolver)
}
