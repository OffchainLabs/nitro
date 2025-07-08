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
	"github.com/ethereum/go-ethereum/rlp"

	dbschema "github.com/offchainlabs/nitro/arbnode/db-schema"
	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
)

func TestMelDatabase(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create database
	arbDb := rawdb.NewMemoryDatabase()
	melDb := NewDatabase(arbDb)

	headMelState := &mel.State{
		ParentChainBlockNumber: 2,
		ParentChainBlockHash:   common.MaxHash,
	}
	require.NoError(t, melDb.SaveState(ctx, headMelState))

	headMelStateBlockNum, err := melDb.GetHeadMelStateBlockNum()
	require.NoError(t, err)
	require.True(t, headMelStateBlockNum == headMelState.ParentChainBlockNumber)

	var melState *mel.State
	checkMelState := func() {
		require.NoError(t, err)
		if !reflect.DeepEqual(melState, headMelState) {
			t.Fatal("unexpected melState retrieved via GetState using parentChainBlockHash")
		}
	}
	melState, err = melDb.FetchInitialState(ctx, headMelState.ParentChainBlockHash)
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
	// Create database
	arbDb := rawdb.NewMemoryDatabase()
	melDb := NewDatabase(arbDb)

	delayedRequestId := common.BigToHash(common.Big1)
	delayedMsg := &mel.DelayedInboxMessage{
		BlockHash: [32]byte{},
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
	state := &mel.State{}
	state.SetDelayedMessageBacklog(&DelayedMessageBacklog{})
	state.SetReadDelayedMsgsAcc(merkleAccumulator.NewNonpersistentMerkleAccumulator())
	require.NoError(t, state.AccumulateDelayedMessage(delayedMsg)) // Initialize delayedMessageBacklog
	state.DelayedMessagedSeen++

	require.NoError(t, melDb.SaveDelayedMessages(ctx, state, []*mel.DelayedInboxMessage{delayedMsg}))
	have, err := melDb.ReadDelayedMessage(ctx, state, 0)
	require.NoError(t, err)

	if !reflect.DeepEqual(have, delayedMsg) {
		t.Fatal("delayed message mismatch")
	}
}

func TestMelDelayedMessagesAccumulation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create database
	arbDb := rawdb.NewMemoryDatabase()
	melDb := NewDatabase(arbDb)

	// Add genesis melState
	var err error
	genesis := &mel.State{
		ParentChainBlockNumber: 1,
	}
	require.NoError(t, melDb.SaveState(ctx, genesis))

	numDelayed := 5
	var delayedMsgs []*mel.DelayedInboxMessage
	for i := int64(1); i <= int64(numDelayed); i++ {
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

	// Initializes delayedMessageBacklog
	genesis.SetDelayedMessageBacklog(&DelayedMessageBacklog{})
	require.NoError(t, err)
	state := genesis.Clone() // Should clone empty initialized delayedMessageBacklog
	state.ParentChainBlockNumber++

	// See 3 delayed messages and accumulate them
	for i := 0; i < numDelayed; i++ {
		require.NoError(t, state.AccumulateDelayedMessage(delayedMsgs[i]))
		state.DelayedMessagedSeen++
	}
	require.NoError(t, melDb.SaveDelayedMessages(ctx, state, delayedMsgs[:numDelayed]))
	// We can read all of these and prove that they are correct, by checking that ReadDelayedMessage doesnt error
	// #nosec G115
	for i := uint64(0); i < uint64(numDelayed); i++ {
		have, err := melDb.ReadDelayedMessage(ctx, state, i)
		require.NoError(t, err)
		require.True(t, reflect.DeepEqual(have, delayedMsgs[i]))
	}
	// If the database were to corrupt a delayed message then check that the state would detect this corruption
	corruptIndex := uint64(3)
	corruptDelayed := delayedMsgs[corruptIndex]
	corruptDelayed.Message.L2msg = []byte("corrupt")
	key := dbKey(dbschema.MelDelayedMessagePrefix, corruptIndex) // #nosec G115
	delayedBytes, err := rlp.EncodeToBytes(*corruptDelayed)
	require.NoError(t, err)
	require.NoError(t, arbDb.Put(key, delayedBytes))
	// ReadDelayedMessage should fail with not part of accumulator error
	_, err = melDb.ReadDelayedMessage(ctx, state, corruptIndex)
	require.True(t, err.Error() == "delayed message message not part of the mel state accumulator")
}

func TestMelFetchInitialStateAndDelayedMessageBacklog(t *testing.T) {
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
	var melStates []*mel.State
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
		melStates = append(melStates, state)
		head = state
	}
	headState := melStates[numMelStates-1]
	state, err := melDb.FetchInitialState(ctx, headState.ParentChainBlockHash)
	require.NoError(t, err)

	require.True(t, state.DelayedMessagedSeen == uint64(numMelStates)*5+1) // #nosec G115
	require.True(t, state.DelayedMessagesRead == 1)
	delayedMessageBacklog := NewDelayedMessageBacklog(00, nil)
	require.NoError(t, delayedMessageBacklog.Initialize(ctx, melDb, state))
	require.True(t, len(delayedMessageBacklog.entries) == 25)
	state.SetDelayedMessageBacklog(delayedMessageBacklog)

	// Lets read the delayed messages and verify their correctness against accumulator and that they match with what we stored
	// we read against the latest melState
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
	// We provide FetchInitialState the current finalized block as 7 and verify that the fetched state has delayedMessageBacklog that will hold
	// delayedMessageBacklogEntry for indexes below the DelayedMessagesRead as those have not been finalized yet!
	newState, err := melDb.FetchInitialState(ctx, newHeadState.ParentChainBlockHash)
	require.NoError(t, err)
	newDelayedMessageBacklog := NewDelayedMessageBacklog(0, func(context.Context) (uint64, error) { return 7, nil })
	require.NoError(t, newDelayedMessageBacklog.Initialize(ctx, melDb, newState))
	// Notice that instead of having seenUnread list from delayed index 13 to 25 inclusive we will have it from 7 to 25 as only till block=7 the chain has finalized and that block has DelayedMessagesRead=7
	require.True(t, len(newDelayedMessageBacklog.entries) == 19)
	newState.SetDelayedMessageBacklog(newDelayedMessageBacklog)

	for i := uint64(7); i < newHeadState.DelayedMessagedSeen; i++ {
		_, melStateParentChainBlockNum, err := newDelayedMessageBacklog.Get(i)
		require.NoError(t, err)                                                           // sanity check
		require.True(t, melStateParentChainBlockNum == uint64(math.Ceil(float64(i)/5))+1) // sanity check
	}

	// Now lets verify that advancing the finalized block number will trim the read but not finalized delayedMessageBacklogEntry while keeping the unread ones
	newDelayedMessageBacklog.finalizedAndReadIndexFetcher = func(context.Context) (uint64, error) { return newHeadState.DelayedMessagesRead, nil }
	newDelayedMessageBacklog.capacity = 0
	require.NoError(t, newDelayedMessageBacklog.clear())
	require.True(t, len(newDelayedMessageBacklog.entries) == int(newHeadState.DelayedMessagedSeen-newHeadState.DelayedMessagesRead)) // #nosec G115
	require.True(t, newDelayedMessageBacklog.entries[0].Index == newHeadState.DelayedMessagesRead)

	// Verify that Reorg handling works as expected
	// Move DelayedMessagesRead manually ahead in newDelayedMessageBacklog by marking the meta's as `Read`
	newSeen := newHeadState.DelayedMessagedSeen - 5 // move back seen by a certain value too
	require.NoError(t, newDelayedMessageBacklog.Reorg(newSeen))
	// as newDelayedMessageBacklog hasnt updated with new finalized info, its starting elements remain unchanged, just that the right parts are trimmed till (newSeen-1) delayed index
	require.True(t, len(newDelayedMessageBacklog.entries) == int(newSeen-newHeadState.DelayedMessagesRead)) // #nosec G115
}
