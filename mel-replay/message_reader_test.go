package melreplay_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/mel-replay"
)

func TestRecordingMessagePreimagesAndReadingMessages(t *testing.T) {
	ctx := context.Background()
	var messages []*arbostypes.MessageWithMetadata
	numMsgs := uint64(25)
	for i := range numMsgs {
		messages = append(messages, &arbostypes.MessageWithMetadata{
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{
					BlockNumber: i + 1,
					RequestId:   &common.Hash{},
					L1BaseFee:   common.Big0,
				},
			},
			DelayedMessagesRead: i + 2,
		})
	}
	state := &mel.State{}
	// Simulate extracting of Messages in native mode to record preimages
	preimages := make(daprovider.PreimagesMap)
	require.NoError(t, state.RecordMsgPreimagesTo(preimages))
	for i := range numMsgs {
		require.NoError(t, state.AccumulateMessage(messages[i]))
		state.MsgCount++
	}
	require.NoError(t, state.GenerateMessageMerklePartialsAndRoot())

	// Test reading in wasm mode
	msgReader := melreplay.NewMessageReader(
		melreplay.NewTypeBasedPreimageResolver(
			arbutil.Keccak256PreimageType,
			preimages,
		),
	)
	for i := range numMsgs {
		msg, err := msgReader.Read(ctx, state, i)
		require.NoError(t, err)
		require.Equal(t, msg.Hash(), messages[i].Hash())
	}
}
