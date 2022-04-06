// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"math/rand"
	"testing"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbstate"
)

func TestBatchSegmentsMaxSize(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	config := &TestBatchPosterConfig
	maxSize := 1000
	config.MaxBatchSize = maxSize + l1HeaderSize
	for i := 0; i < 1000; i++ {
		var delayedMessages uint64
		segments := newBatchSegments(delayedMessages, config)
		for {
			msg := make([]byte, rng.Int()%(maxSize/2))
			if rng.Int()%5 == 0 {
				delayedMessages++
			} else {
				divisor := byte(rng.Uint32()) / 4 // limits entropy
				if divisor == 0 {
					divisor = 1
				}
				for j := range msg {
					msg[j] = byte(rng.Uint32()) / divisor
				}
			}
			message := &arbstate.MessageWithMetadata{
				Message: &arbos.L1IncomingMessage{
					Header: &arbos.L1IncomingMessageHeader{
						BlockNumber: 0,
						Timestamp:   0,
					},
					L2msg: msg,
				},
				DelayedMessagesRead: delayedMessages,
			}
			success, err := segments.AddMessage(message)
			Require(t, err)
			if !success {
				break
			}
		}
		data, err := segments.CloseAndGetBytes()
		Require(t, err)
		if len(data) > maxSize {
			t.Fatal("created data of length", len(data), "longer than maximum size", maxSize)
		}
	}
}
