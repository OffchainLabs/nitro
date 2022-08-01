/*
 * Copyright 2020-2021, Offchain Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package broadcaster

import (
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"testing"
)

func TestGetEmptyCacheMessages(t *testing.T) {
	buffer := SequenceNumberCatchupBuffer{
		messages:     []*BroadcastFeedMessage{},
		messageCount: 0,
	}

	// Get everything
	bm := buffer.getCacheMessages(0)
	if bm != nil {
		t.Error("shouldn't have returned anything")
	}
}

func createDummyBroadcastMessages(seqNums []arbutil.MessageIndex) []*BroadcastFeedMessage {
	broadcastMessages := make([]*BroadcastFeedMessage, 0, len(seqNums))
	for _, seqNum := range seqNums {
		broadcastMessage := &BroadcastFeedMessage{
			SequenceNumber: seqNum,
			Message:        arbstate.MessageWithMetadata{},
		}
		broadcastMessages = append(broadcastMessages, broadcastMessage)
	}

	return broadcastMessages
}

func TestGetCacheMessages(t *testing.T) {
	indexes := []arbutil.MessageIndex{40, 40, 41, 45, 46, 47, 48}
	buffer := SequenceNumberCatchupBuffer{
		messages:     createDummyBroadcastMessages(indexes),
		messageCount: int32(len(indexes)),
	}

	// Get everything
	bm := buffer.getCacheMessages(0)
	if len(bm.Messages) != 7 {
		t.Error("didn't return all messages")
	}

	// Get everything
	bm = buffer.getCacheMessages(1)
	if len(bm.Messages) != 7 {
		t.Error("didn't return all messages")
	}

	// Get everything
	bm = buffer.getCacheMessages(40)
	if len(bm.Messages) != 7 {
		t.Error("didn't return all messages")
	}

	// Get nothing
	bm = buffer.getCacheMessages(100)
	if bm != nil {
		t.Error("should not have returned anything")
	}

	// Test single
	bm = buffer.getCacheMessages(48)
	if bm == nil {
		t.Fatal("nothing returned")
	}
	if len(bm.Messages) != 1 {
		t.Errorf("expected 1 message, got %d messages", len(bm.Messages))
	}
	if bm.Messages[0].SequenceNumber != 48 {
		t.Errorf("expected sequence number 48, got %d", bm.Messages[0].SequenceNumber)
	}

	// Test when messages missing
	bm = buffer.getCacheMessages(45)
	if bm == nil {
		t.Fatal("nothing returned")
	}
	if len(bm.Messages) != 4 {
		t.Errorf("expected 4 messages, got %d messages", len(bm.Messages))
	}
	if bm.Messages[0].SequenceNumber != 45 {
		t.Errorf("expected sequence number 45, got %d", bm.Messages[0].SequenceNumber)
	}

	// Test when duplicate message
	bm = buffer.getCacheMessages(41)
	if bm == nil {
		t.Fatal("nothing returned")
	}
	if len(bm.Messages) != 5 {
		t.Errorf("expected only 5 messages, got %d messages", len(bm.Messages))
	}
	if bm.Messages[0].SequenceNumber != 41 {
		t.Errorf("expected sequence number 41, got %d", bm.Messages[0].SequenceNumber)
	}

}
