package melreplay

import (
	"context"
	"fmt"
	"math/bits"

	"github.com/offchainlabs/nitro/arbnode/mel"
)

type delayedMessageDatabase struct {
	preimageResolver PreimageResolver
}

func NewDelayedMessageDatabase(preimageResolver PreimageResolver) mel.DelayedMessageDatabase {
	return &delayedMessageDatabase{preimageResolver}
}

func (d *delayedMessageDatabase) ReadDelayedMessage(
	ctx context.Context,
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
