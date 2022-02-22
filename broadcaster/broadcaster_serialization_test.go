//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package broadcaster

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
)

func ExampleBroadcastMessage_broadcastfeedmessage() {
	msg := BroadcastMessage{
		Version: 1,
		Messages: []*BroadcastFeedMessage{
			{
				SequenceNumber: 12345,
				Message: arbstate.MessageWithMetadata{
					Message: &arbos.L1IncomingMessage{
						Header: &arbos.L1IncomingMessageHeader{
							Kind:        0,
							Poster:      [20]byte{},
							BlockNumber: [32]byte{},
							Timestamp:   [32]byte{},
							RequestId:   [32]byte{},
							BaseFeeL1:   [32]byte{},
						},
						L2msg: []byte{0xde, 0xad, 0xbe, 0xef},
					},
					DelayedMessagesRead: 3333,
				},
			},
		},
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	_ = encoder.Encode(msg)
	fmt.Println(buf.String())
	// Output: {"version":1,"messages":[{"sequenceNumber":12345,"message":{"message":{"header":{"kind":0,"sender":"0x0000000000000000000000000000000000000000","blockNumber":"0x0000000000000000000000000000000000000000000000000000000000000000","timestamp":"0x0000000000000000000000000000000000000000000000000000000000000000","requestId":"0x0000000000000000000000000000000000000000000000000000000000000000","baseFeeL1":"0x0000000000000000000000000000000000000000000000000000000000000000"},"l2Msg":"3q2+7w=="},"delayedMessagesRead":3333}}]}
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
