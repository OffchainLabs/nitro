// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melreplay_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbnode/mel/recording"
	"github.com/offchainlabs/nitro/arbnode/mel/runner"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/mel-replay"
)

func TestRecordingPreimagesForReadDelayedMessage(t *testing.T) {
	var delayedMessages []*mel.DelayedInboxMessage
	numMsgs := uint64(10)
	for i := range numMsgs {
		delayedMessages = append(delayedMessages, &mel.DelayedInboxMessage{
			ParentChainBlockNumber: i,
			BlockHash:              common.HexToHash(fmt.Sprintf("msg:%d", i)),
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{
					BlockNumber: i,
					RequestId:   &common.Hash{},
					L1BaseFee:   common.Big0,
				},
			},
		})
	}
	db := rawdb.NewMemoryDatabase()
	melDB := melrunner.NewDatabase(db)
	err := melDB.SaveDelayedMessages(&mel.State{DelayedMessagesSeen: uint64(len(delayedMessages))}, delayedMessages)
	require.NoError(t, err)

	startBlockNum := uint64(3)
	state := &mel.State{
		ParentChainBlockNumber: startBlockNum,
		DelayedMessagesSeen:    startBlockNum,
		DelayedMessagesRead:    startBlockNum,
	}
	for i := range startBlockNum {
		require.NoError(t, state.AccumulateDelayedMessage(delayedMessages[i]))
	}
	require.NoError(t, state.GenerateDelayedMessagesSeenMerklePartialsAndRoot())
	require.NoError(t, melDB.SaveState(state))

	preimages := make(daprovider.PreimagesMap)
	recordingDB, err := melrecording.NewDelayedMsgDatabase(db, preimages)
	require.NoError(t, err)
	for i := startBlockNum; i < numMsgs; i++ {
		require.NoError(t, state.AccumulateDelayedMessage(delayedMessages[i]))
		state.DelayedMessagesSeen++
	}
	require.NoError(t, state.GenerateDelayedMessagesSeenMerklePartialsAndRoot())

	// Simulate reading of delayed Messages in native mode to record preimages
	numMsgsToRead := uint64(7)
	for i := startBlockNum; i < numMsgsToRead; i++ {
		delayed, err := recordingDB.ReadDelayedMessage(state, i)
		require.NoError(t, err)
		require.Equal(t, delayed.Hash(), delayedMessages[i].Hash())
	}

	// Test reading in wasm mode
	delayedDB := melreplay.NewDelayedMessageDatabase(
		melreplay.NewTypeBasedPreimageResolver(
			arbutil.Keccak256PreimageType,
			preimages,
		),
	)
	for i := startBlockNum; i < numMsgsToRead; i++ {
		msg, err := delayedDB.ReadDelayedMessage(state, i)
		require.NoError(t, err)
		require.Equal(t, msg.Hash(), delayedMessages[i].Hash())
	}
}
