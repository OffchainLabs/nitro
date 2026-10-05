// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package mel

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

// These tests pin down the serialization contract the JSON-RPC layer relies on
// (arbnode/mel/rpc_runner): the exported fields of the shared MEL types survive a
// JSON round-trip, and a State received over the wire (private preimage cache dropped)
// hashes identically and can have its cache rebuilt locally — exactly what the RPC
// query client does for the MEL validator.

func TestStateJSONRoundTripPreservesHashes(t *testing.T) {
	t.Parallel()
	original := &State{
		Version:                            3,
		ParentChainId:                      42161,
		ParentChainBlockNumber:             123456,
		BatchPostingTargetAddress:          common.HexToAddress("0x00000000000000000000000000000000000000a1"),
		DelayedMessagePostingTargetAddress: common.HexToAddress("0x00000000000000000000000000000000000000b2"),
		ParentChainBlockHash:               common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
		ParentChainPreviousBlockHash:       common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"),
		BatchCount:                         7,
		MsgCount:                           99,
		LocalMsgAccumulator:                common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333"),
		DelayedMessagesRead:                5,
		DelayedMessagesSeen:                8,
		DelayedMessageInboxAcc:             common.HexToHash("0x4444444444444444444444444444444444444444444444444444444444444444"),
		DelayedMessageOutboxAcc:            common.HexToHash("0x5555555555555555555555555555555555555555555555555555555555555555"),
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded State
	require.NoError(t, json.Unmarshal(data, &decoded))

	require.Equal(t, *original, decoded, "all exported fields must survive the JSON round-trip")
	require.Equal(t, original.Hash(), decoded.Hash(), "state hash must match across the wire")
}

func TestDelayedInboxMessageJSONRoundTrip(t *testing.T) {
	t.Parallel()
	msg := createTestDelayedMessages(1)[0]
	msg.BlockHash = common.HexToHash("0xabc")
	msg.BeforeInboxAcc = common.HexToHash("0xdef")
	msg.ParentChainBlockNumber = 777

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded DelayedInboxMessage
	require.NoError(t, json.Unmarshal(data, &decoded))

	require.Equal(t, msg.Hash(), decoded.Hash())
	require.Equal(t, msg.AfterInboxAcc(), decoded.AfterInboxAcc())
}

func TestBatchMetadataJSONRoundTrip(t *testing.T) {
	t.Parallel()
	bm := BatchMetadata{
		Accumulator:         common.HexToHash("0x99"),
		MessageCount:        12345,
		DelayedMessageCount: 42,
		ParentChainBlock:    9000,
	}
	data, err := json.Marshal(bm)
	require.NoError(t, err)
	var decoded BatchMetadata
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, bm, decoded)
}
