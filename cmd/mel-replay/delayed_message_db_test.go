package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbnode"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/stretchr/testify/require"
)

var _ preimageResolver = (*mockPreimageResolver)(nil)
var _ meltypes.DelayedMessageDatabase = (*delayedMessageDatabase)(nil)

func TestReadDelayedMessage(t *testing.T) {
	ctx := context.Background()
	t.Run("no delayed messages", func(t *testing.T) {
		db := &delayedMessageDatabase{}
		state := &meltypes.State{
			DelayedMessagedSeen: 0,
		}
		_, err := db.ReadDelayedMessage(ctx, state, 0)
		require.ErrorContains(t, err, "no delayed messages available")
	})
	t.Run("message index out of range", func(t *testing.T) {
		db := &delayedMessageDatabase{}
		state := &meltypes.State{
			DelayedMessagedSeen: 5,
		}
		_, err := db.ReadDelayedMessage(ctx, state, 5)
		require.ErrorContains(t, err, "index 5 out of range, total delayed messages seen: 5")
	})
	t.Run("single message in Merkle tree", func(t *testing.T) {
		// If there is only a single delayed message in the
		// Merkle tree, then it should be easy to retrieve as a preimage
		// lookup of the root itself.
		msg, msgHash := buildDelayedMessage(t, 100, []byte("foobar"))
		encodedMsg, err := rlp.EncodeToBytes(msg)
		require.NoError(t, err)
		resolver := &mockPreimageResolver{
			preimages: map[common.Hash][]byte{
				msgHash: encodedMsg,
			},
		}
		db := &delayedMessageDatabase{
			preimageResolver: resolver,
		}
		state := &meltypes.State{
			DelayedMessagedSeen:     1,
			DelayedMessagesSeenRoot: msgHash,
		}
		retrievedMsg, err := db.ReadDelayedMessage(ctx, state, 0)
		require.NoError(t, err)
		require.Equal(t, []byte("foobar"), retrievedMsg.Message.L2msg)
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
		msgA, msgAHash := buildDelayedMessage(t, 1, []byte("a"))
		msgB, msgBHash := buildDelayedMessage(t, 2, []byte("b"))

		encodedMsgA, err := rlp.EncodeToBytes(msgA)
		require.NoError(t, err)
		encodedMsgB, err := rlp.EncodeToBytes(msgB)
		require.NoError(t, err)

		rootPreimage := append(msgAHash[:], msgBHash[:]...)
		root := crypto.Keccak256Hash(msgAHash[:], msgBHash[:])

		resolver := &mockPreimageResolver{
			preimages: map[common.Hash][]byte{
				msgAHash: encodedMsgA,
				msgBHash: encodedMsgB,
				root:     rootPreimage,
			},
		}
		db := &delayedMessageDatabase{
			preimageResolver: resolver,
		}
		state := &meltypes.State{
			DelayedMessagedSeen:     2,
			DelayedMessagesSeenRoot: root,
		}
		retrievedMsg, err := db.ReadDelayedMessage(ctx, state, 0)
		require.NoError(t, err)
		require.Equal(t, []byte("a"), retrievedMsg.Message.L2msg)

		retrievedMsg, err = db.ReadDelayedMessage(ctx, state, 1)
		require.NoError(t, err)
		require.Equal(t, []byte("b"), retrievedMsg.Message.L2msg)
	})
	t.Run("Merkle tree with 3 levels can fetch specific delayed messages", func(t *testing.T) {
		// We have a Merkle tree for delayed messages that looks like this:
		//
		//     hash(hash(A++B)++hash(C++D))
		//       /                \
		//   hash(A++B)        hash(C++D)
		//   /       \          /       \
		//  A         B        C         D
		//
		// We should be able to fetch A, B, C, or D.
		msgA, msgAHash := buildDelayedMessage(t, 1, []byte("a"))
		msgB, msgBHash := buildDelayedMessage(t, 2, []byte("b"))
		msgC, msgCHash := buildDelayedMessage(t, 3, []byte("c"))
		msgD, msgDHash := buildDelayedMessage(t, 4, []byte("d"))

		encodedMsgA, err := rlp.EncodeToBytes(msgA)
		require.NoError(t, err)
		encodedMsgB, err := rlp.EncodeToBytes(msgB)
		require.NoError(t, err)
		encodedMsgC, err := rlp.EncodeToBytes(msgC)
		require.NoError(t, err)
		encodedMsgD, err := rlp.EncodeToBytes(msgD)
		require.NoError(t, err)

		middleLeftPreimage := append(msgAHash[:], msgBHash[:]...)
		middleRightPreimage := append(msgCHash[:], msgDHash[:]...)
		middleLeftRoot := crypto.Keccak256Hash(msgAHash[:], msgBHash[:])
		middleRightRoot := crypto.Keccak256Hash(msgCHash[:], msgDHash[:])

		rootPreimage := append(middleLeftRoot[:], middleRightRoot[:]...)
		root := crypto.Keccak256Hash(middleLeftRoot[:], middleRightRoot[:])

		resolver := &mockPreimageResolver{
			preimages: map[common.Hash][]byte{
				msgAHash:        encodedMsgA,
				msgBHash:        encodedMsgB,
				msgCHash:        encodedMsgC,
				msgDHash:        encodedMsgD,
				middleLeftRoot:  middleLeftPreimage,
				middleRightRoot: middleRightPreimage,
				root:            rootPreimage,
			},
		}
		db := &delayedMessageDatabase{
			preimageResolver: resolver,
		}
		state := &meltypes.State{
			DelayedMessagedSeen:     4,
			DelayedMessagesSeenRoot: root,
		}
		retrievedMsg, err := db.ReadDelayedMessage(ctx, state, 0)
		require.NoError(t, err)
		require.Equal(t, []byte("a"), retrievedMsg.Message.L2msg)

		retrievedMsg, err = db.ReadDelayedMessage(ctx, state, 1)
		require.NoError(t, err)
		require.Equal(t, []byte("b"), retrievedMsg.Message.L2msg)

		retrievedMsg, err = db.ReadDelayedMessage(ctx, state, 2)
		require.NoError(t, err)
		require.Equal(t, []byte("c"), retrievedMsg.Message.L2msg)

		retrievedMsg, err = db.ReadDelayedMessage(ctx, state, 3)
		require.NoError(t, err)
		require.Equal(t, []byte("d"), retrievedMsg.Message.L2msg)
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
		result := nextPowerOfTwo(tc.input)
		if result != tc.expected {
			t.Errorf("nextPowerOfTwo(%d) = %d, expected %d", tc.input, result, tc.expected)
		}
	}
}

type mockPreimageResolver struct {
	preimages map[common.Hash][]byte
}

func (m *mockPreimageResolver) ResolveTypedPreimage(preimageType arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	if preimage, exists := m.preimages[hash]; exists {
		return preimage, nil
	}
	return nil, fmt.Errorf("preimage not found for hash: %s", hash.Hex())
}

func buildDelayedMessage(
	t *testing.T,
	blockNumber uint64,
	msgData []byte,
) (*arbnode.DelayedInboxMessage, common.Hash) {
	msg := &arbnode.DelayedInboxMessage{
		ParentChainBlockNumber: blockNumber,
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				Kind: arbostypes.L1MessageType_L2Message,
			},
			L2msg: msgData,
		},
	}
	encoded, err := rlp.EncodeToBytes(msg)
	require.NoError(t, err)
	return msg, crypto.Keccak256Hash(encoded)
}
