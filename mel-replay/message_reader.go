package melreplay

import (
	"context"
	"fmt"
	"math/bits"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

type MessageReader struct {
	preimageResolver PreimageResolver
}

func NewMessageReader(preimageResolver PreimageResolver) *MessageReader {
	return &MessageReader{preimageResolver}
}

func (m *MessageReader) Read(
	ctx context.Context,
	state *mel.State,
	msgIndex uint64,
) (*arbostypes.MessageWithMetadata, error) {
	if msgIndex >= state.MsgCount {
		return nil, fmt.Errorf("index %d out of range, total messages: %d", msgIndex, state.MsgCount)
	}
	treeSize := NextPowerOfTwo(state.MsgCount)
	merkleDepth := bits.TrailingZeros64(treeSize)
	return fetchObjectFromMerkleTree[arbostypes.MessageWithMetadata](state.MsgRoot, merkleDepth, msgIndex, m.preimageResolver)
}
