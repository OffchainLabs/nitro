package melrunner

import (
	"context"
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
	state.SetDelayedMessageBacklog(&mel.DelayedMessageBacklog{})
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
	genesis.SetDelayedMessageBacklog(&mel.DelayedMessageBacklog{})
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
