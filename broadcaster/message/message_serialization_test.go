// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package message

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

func ExampleBroadcastMessage_broadcastfeedmessage() {
	var requestId common.Hash
	msg := BroadcastMessage{
		Version: 1,
		Messages: []*BroadcastFeedMessage{
			{
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
				Signature: nil,
			},
		},
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	_ = encoder.Encode(msg)
	fmt.Println(buf.String())
	// Output: {"version":1,"messages":[{"sequenceNumber":12345,"message":{"message":{"header":{"kind":0,"sender":"0x0000000000000000000000000000000000000000","blockNumber":0,"timestamp":0,"requestId":"0x0000000000000000000000000000000000000000000000000000000000000000","baseFeeL1":0},"l2Msg":"3q2+7w=="},"delayedMessagesRead":3333},"signature":null}]}
}

func ExampleBroadcastMessage_emptymessage() {
	msg := BroadcastMessage{
		Version: 1,
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	_ = encoder.Encode(msg)
	fmt.Println(buf.String())
	// Output: {"version":1}
}

func ExampleBroadcastMessage_confirmedseqnum() {
	msg := BroadcastMessage{
		Version: 1,
		ConfirmedSequenceNumberMessage: &ConfirmedSequenceNumberMessage{
			SequenceNumber: 1234,
		},
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	_ = encoder.Encode(msg)
	fmt.Println(buf.String())
	// Output: {"version":1,"confirmedSequenceNumberMessage":{"sequenceNumber":1234}}
}
