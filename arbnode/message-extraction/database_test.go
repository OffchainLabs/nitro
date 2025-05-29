package mel

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/arbnode"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

func TestMelDatabase(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create database
	arbDb := rawdb.NewMemoryDatabase()
	melDb := NewDatabase(arbDb)

	headMelState := &meltypes.State{
		ParentChainBlockNumber: 2,
		ParentChainBlockHash:   common.MaxHash,
	}
	require.NoError(t, melDb.SaveState(ctx, headMelState))

	headMelStateBlockNum, err := melDb.GetHeadMelStateBlockNum()
	require.NoError(t, err)
	require.True(t, headMelStateBlockNum == headMelState.ParentChainBlockNumber)

	var melState *meltypes.State
	checkMelState := func() {
		require.NoError(t, err)
		if !reflect.DeepEqual(melState, headMelState) {
			t.Fatal("unexpected melState retrieved via GetState using parentChainBlockHash")
		}
	}
	melState, err = melDb.GetState(ctx, headMelState.ParentChainBlockHash)
	checkMelState()
	melState, err = melDb.State(ctx, headMelState.ParentChainBlockNumber)
	checkMelState()

}

func TestMelDatabaseReadAndWriteDelayedMessages(t *testing.T) {
	// Simple test for writing and reading of delayed messages.
	// TODO: write a separate detailed test after delayed messages accumulation logic is implemented
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Init
	exec, streamer, arbDb, _ := arbnode.NewTransactionStreamerForTest(t, ctx, common.Address{})
	err := streamer.Start(ctx)
	arbnode.Require(t, err)
	exec.Start(ctx)
	melDb := NewDatabase(arbDb)

	init, err := streamer.GetMessage(0)
	require.NoError(t, err)
	initMsgDelayed := &arbnode.DelayedInboxMessage{
		BlockHash:      [32]byte{},
		BeforeInboxAcc: [32]byte{},
		Message:        init.Message,
	}
	delayedRequestId := common.BigToHash(common.Big1)
	delayedMsg := &arbnode.DelayedInboxMessage{
		BlockHash:      [32]byte{},
		BeforeInboxAcc: initMsgDelayed.AfterInboxAcc(),
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				Kind:        arbostypes.L1MessageType_EndOfBlock,
				Poster:      [20]byte{},
				BlockNumber: 0,
				Timestamp:   0,
				RequestId:   &delayedRequestId,
				L1BaseFee:   common.Big0,
			},
		},
	}
	require.NoError(t, melDb.SaveDelayedMessages(ctx, &meltypes.State{DelayedMessagedSeen: 1}, []*arbnode.DelayedInboxMessage{delayedMsg}))
	have, err := melDb.ReadDelayedMessage(ctx, nil, 0)
	require.NoError(t, err)

	if !reflect.DeepEqual(have, delayedMsg) {
		t.Fatal("delayed message mismatch")
	}
}
