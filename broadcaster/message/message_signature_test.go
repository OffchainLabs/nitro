// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package message

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

func TestBroadcastFeedMessageSignature(t *testing.T) {
	var requestId common.Hash
	msg := BroadcastFeedMessage{
		SequenceNumber: 12345,
		Message: arbostypes.MessageWithMetadata{
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{
					Kind:        0,
					Poster:      [20]byte{},
					BlockNumber: 0,
					Timestamp:   0,
					RequestId:   &requestId,
					L1BaseFee:   big.NewInt(0),
				},
				L2msg: []byte{0xde, 0xad, 0xbe, 0xef},
			},
			DelayedMessagesRead: 3333,
		},
		Signature:     nil,
		BlockMetadata: []byte{0xde, 0xad, 0xbe, 0xaf},
	}

	const chainId = 0xa4b1
	preimage := msg.SignaturePreimage(chainId)
	// Compare against hard-coded preimage to ensure it won't break in the future
	expected := common.HexToHash("0x42a956aed0c4b4b1e5167461cfcbebf88f526f01828e7e78ff0283f6d60302ef")
	require.Equal(t, expected, preimage)
}
