package melrunner

import (
	"context"
	"math"
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

func TestDelayedMessageBacklogInitialization(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create database
	arbDb := rawdb.NewMemoryDatabase()
	melDb := NewDatabase(arbDb)

	// Add genesis melState
	genesis := &mel.State{
		ParentChainBlockNumber: 1,
		DelayedMessagedSeen:    1,
		DelayedMessagesRead:    1,
	}
	require.NoError(t, melDb.SaveState(ctx, genesis))

	numMelStates := 5
	var delayedMsgs []*mel.DelayedInboxMessage
	for i := int64(1); i <= int64(numMelStates)*5; i++ {
		requestID := common.BigToHash(big.NewInt(i))
		delayedMsgs = append(delayedMsgs, &mel.DelayedInboxMessage{
			BlockHash: [32]byte{},
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{
					Kind:        arbostypes.L1MessageType_EndOfBlock,
					Poster:      [20]byte{},
					BlockNumber: 0,
					Timestamp:   0,
					RequestId:   &requestID,
					L1BaseFee:   common.Big0,
				},
			},
		})
	}

	// Simulate a node seeing 25 delayed messages but reading none
	head := genesis
	// #nosec G115
	for i := uint64(0); i < uint64(numMelStates); i++ {
		state := &mel.State{
			ParentChainBlockNumber: i + 2,
			ParentChainBlockHash:   common.BigToHash(new(big.Int).SetUint64(i + 2)),
			DelayedMessagedSeen:    head.DelayedMessagedSeen + 5,
			DelayedMessagesRead:    1,
		}
		require.NoError(t, melDb.SaveDelayedMessages(ctx, state, delayedMsgs[(i)*5:(i+1)*5]))
		require.NoError(t, melDb.SaveState(ctx, state))
		head = state
	}
	state, err := melDb.GetHeadMelState(ctx)
	require.NoError(t, err)

	require.True(t, state.DelayedMessagedSeen == uint64(numMelStates)*5+1) // #nosec G115
	require.True(t, state.DelayedMessagesRead == 1)
	delayedMessageBacklog, err := mel.NewDelayedMessageBacklog(ctx, 1, func(ctx context.Context) (uint64, error) { return 0, nil }, mel.WithUnboundedCapacity)
	require.NoError(t, err)
	require.NoError(t, InitializeDelayedMessageBacklog(ctx, delayedMessageBacklog, melDb, state, nil))
	require.True(t, delayedMessageBacklog.Len() == 25)
	state.SetDelayedMessageBacklog(delayedMessageBacklog)
	state.SetReadCountFromBacklog(state.DelayedMessagedSeen) // skip checking against accumulator- not the purpose of this test

	// Lets read the delayed messages and verify that they match with what we stored
	for i, wantDelayed := range delayedMsgs {
		haveDelayed, err := melDb.ReadDelayedMessage(ctx, state, uint64(i+1)) // #nosec G115
		require.NoError(t, err)
		require.True(t, reflect.DeepEqual(haveDelayed, wantDelayed))
	}

	// Intermediary melState to verify that finalized read delayed messages are added to delayedMessageBacklog
	state = &mel.State{
		ParentChainBlockNumber: 7,
		ParentChainBlockHash:   common.BigToHash(new(big.Int).SetUint64(7)),
		DelayedMessagedSeen:    26,
		DelayedMessagesRead:    7,
	}
	require.NoError(t, melDb.SaveState(ctx, state))

	// Advance head state indicating that we have read 10 delayed messages
	newHeadState := &mel.State{
		ParentChainBlockNumber: 8,
		ParentChainBlockHash:   common.BigToHash(new(big.Int).SetUint64(8)),
		DelayedMessagedSeen:    26,
		DelayedMessagesRead:    13,
	}
	require.NoError(t, melDb.SaveState(ctx, newHeadState))
	// We provide InitializeDelayedMessageBacklog the current finalized block as 7 and verify that the delayedMessageBacklog has
	// delayedMessageBacklogEntry for indexes below the DelayedMessagesRead as those have not been finalized yet!
	newState, err := melDb.GetHeadMelState(ctx)
	require.NoError(t, err)
	newDelayedMessageBacklog, err := mel.NewDelayedMessageBacklog(ctx, 1, func(ctx context.Context) (uint64, error) { return 0, nil }, mel.WithUnboundedCapacity)
	require.NoError(t, err)
	require.NoError(t, InitializeDelayedMessageBacklog(ctx, newDelayedMessageBacklog, melDb, newState, func(context.Context) (uint64, error) { return 7, nil }))
	// Notice that instead of having seenUnread list from delayed index 13 to 25 inclusive we will have it from 7 to 25 as only till block=7 the chain has finalized and that block has DelayedMessagesRead=7
	require.True(t, newDelayedMessageBacklog.Len() == 19)
	newState.SetDelayedMessageBacklog(newDelayedMessageBacklog)

	for i := uint64(7); i < newHeadState.DelayedMessagedSeen; i++ {
		delayedMeta, err := newDelayedMessageBacklog.Get(i)
		require.NoError(t, err)                                                                       // sanity check
		require.True(t, delayedMeta.MelStateParentChainBlockNum == uint64(math.Ceil(float64(i)/5))+1) // sanity check
	}
}
