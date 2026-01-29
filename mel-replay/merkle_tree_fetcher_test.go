package melreplay_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/mel-replay"
)

var _ melreplay.PreimageResolver = (*mockPreimageResolver)(nil)

type mockPreimageResolver struct {
	preimages map[common.Hash][]byte
}

func (m *mockPreimageResolver) ResolveTypedPreimage(preimageType arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	if preimage, exists := m.preimages[hash]; exists {
		return preimage, nil
	}
	return nil, fmt.Errorf("preimage not found for hash: %s", hash.Hex())
}

func TestFetchObjectFromMerkleTree(t *testing.T) {
	// Tests implementation of fetchObjectFromMerkleTree method
	ctx := context.Background()
	t.Run("message index out of range", func(t *testing.T) {
		db := melreplay.NewDelayedMessageDatabase(nil)
		state := &mel.State{
			DelayedMessagesSeen: 5,
		}
		_, err := db.ReadDelayedMessage(ctx, state, 5)
		require.ErrorContains(t, err, "index 5 out of range, total delayed messages seen: 5")
	})
	t.Run("single message in Merkle tree", func(t *testing.T) {
		// If there is only a single delayed message in the
		// Merkle tree, then it should be easy to retrieve as a preimage
		// lookup of the root itself.
		messages := []*mel.DelayedInboxMessage{
			buildDelayedMessage(t, 100, []byte("foobar")),
		}

		preimages, root := buildMerkleTree(t, messages)

		resolver := &mockPreimageResolver{preimages: preimages}
		db := melreplay.NewDelayedMessageDatabase(resolver)
		state := &mel.State{
			DelayedMessagesSeen:     1,
			DelayedMessagesSeenRoot: root,
		}

		msg, err := db.ReadDelayedMessage(ctx, state, uint64(0)) // #nosec G115
		require.NoError(t, err)
		require.Equal(t, []byte("foobar"), msg.Message.L2msg)
	})
	t.Run("Merkle tree with 2 levels can fetch left or right delayed message", func(t *testing.T) {
		// We have a Merkle tree for delayed messages that looks like this:
		//
		//   hash(A++B)
		//   /       \
		//  A         B
		//
		// Where A and B are delayed messages hashes.
		// If we want to fetch delayed message at index 0, we should get A,
		// and if we want to fetch delayed message at index 1, we should get B
		// through our algorithm.
		messages := []*mel.DelayedInboxMessage{
			buildDelayedMessage(t, 1, []byte("a")),
			buildDelayedMessage(t, 2, []byte("b")),
		}

		preimages, root := buildMerkleTree(t, messages)

		resolver := &mockPreimageResolver{preimages: preimages}
		db := melreplay.NewDelayedMessageDatabase(resolver)
		state := &mel.State{
			DelayedMessagesSeen:     2,
			DelayedMessagesSeenRoot: root,
		}

		// Test each message
		expectedData := [][]byte{[]byte("a"), []byte("b")}
		for i, expected := range expectedData {
			msg, err := db.ReadDelayedMessage(ctx, state, uint64(i)) // #nosec G115
			require.NoError(t, err)
			require.Equal(t, expected, msg.Message.L2msg)
		}
	})
	t.Run("Merkle tree with 3 levels can fetch specific delayed messages", func(t *testing.T) {
		// We have a Merkle tree for delayed messages that looks like this:
		//
		//     hash(hash(A++B)++hash(C++D))
		//       /                \
		//   hash(A++B)        hash(C++D)
		//   /       \          /       \
		//  A         B        C        EMPTY
		//
		// We should be able to fetch A, B, C.
		messages := []*mel.DelayedInboxMessage{
			buildDelayedMessage(t, 1, []byte("a")),
			buildDelayedMessage(t, 2, []byte("b")),
			buildDelayedMessage(t, 3, []byte("c")),
		}

		preimages, root := buildMerkleTree(t, messages)

		resolver := &mockPreimageResolver{preimages: preimages}
		db := melreplay.NewDelayedMessageDatabase(resolver)
		state := &mel.State{
			DelayedMessagesSeen:     3,
			DelayedMessagesSeenRoot: root,
		}

		// Test each message
		expectedData := [][]byte{[]byte("a"), []byte("b"), []byte("c")}
		for i, expected := range expectedData {
			msg, err := db.ReadDelayedMessage(ctx, state, uint64(i)) // #nosec G115
			require.NoError(t, err)
			require.Equal(t, expected, msg.Message.L2msg)
		}
	})
}

func TestNextPowerOfTwo(t *testing.T) {
	testCases := []struct {
		input    uint64
		expected uint64
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 8},
		{8, 8},
		{9, 16},
		{16, 16},
		{17, 32},
	}

	for _, tc := range testCases {
		result := melreplay.NextPowerOfTwo(tc.input)
		if result != tc.expected {
			t.Errorf("nextPowerOfTwo(%d) = %d, expected %d", tc.input, result, tc.expected)
		}
	}
}

func buildDelayedMessage(
	_ *testing.T,
	blockNumber uint64,
	msgData []byte,
) *mel.DelayedInboxMessage {
	msg := &mel.DelayedInboxMessage{
		ParentChainBlockNumber: blockNumber,
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				Kind: arbostypes.L1MessageType_L2Message,
			},
			L2msg: msgData,
		},
	}
	return msg
}

func buildMerkleTree(t *testing.T, messages []*mel.DelayedInboxMessage) (map[common.Hash][]byte, common.Hash) {
	preimages := make(map[common.Hash][]byte)
	leafHashes := make([]common.Hash, len(messages))
	for i, msg := range messages {
		encoded, err := rlp.EncodeToBytes(msg)
		require.NoError(t, err)
		hash := crypto.Keccak256Hash(encoded)
		preimages[hash] = encoded
		leafHashes[i] = hash
	}

	currentLevel := leafHashes
	for len(currentLevel) > 1 {
		nextLevel := make([]common.Hash, 0, (len(currentLevel)+1)/2)

		for i := 0; i < len(currentLevel); i += 2 {
			left := currentLevel[i]
			var right common.Hash

			if i+1 < len(currentLevel) {
				right = currentLevel[i+1]
			} else {
				right = common.Hash{}
			}

			preimage := make([]byte, 0)
			preimage = append(preimage, left[:]...)
			preimage = append(preimage, right[:]...)
			parent := crypto.Keccak256Hash(left[:], right[:])
			preimages[parent] = preimage
			nextLevel = append(nextLevel, parent)
		}
		currentLevel = nextLevel
	}
	return preimages, currentLevel[0]
}
