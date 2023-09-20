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
	"strings"
	"testing"

	m "github.com/offchain/com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func TestGetEmptyCacheMessages(t *testing.T) {
	buffer := SequenceNumberCatchupBuffer{
		messages:     nil,
		messageCount: 0,
		limitCatchup: func() bool { return false },
		maxCatchup:   func() int { return -1 },
	}

	// Get everything
	bm := buffer.getCacheMessages(0)
	if bm != nil {
		t.Error("shouldn't have returned anything")
	}
}

func TestGetCacheMessages(t *testing.T) {
	indexes := []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46}
	buffer := SequenceNumberCatchupBuffer{
		messages:     m.CreateDummyBroadcastMessages(indexes),
		messageCount: int32(len(indexes)),
		limitCatchup: func() bool { return false },
		maxCatchup:   func() int { return -1 },
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
	bm = buffer.getCacheMessages(46)
	if bm == nil {
		t.Fatal("nothing returned")
	}
	if len(bm.Messages) != 1 {
		t.Errorf("expected 1 message, got %d messages", len(bm.Messages))
	}
	if bm.Messages[0].SequenceNumber != 46 {
		t.Errorf("expected sequence number 46, got %d", bm.Messages[0].SequenceNumber)
	}

	// Test extremes
	bm = buffer.getCacheMessages(arbutil.MessageIndex(^uint64(0)))
	if bm != nil {
		t.Fatal("should not have returned anything")
	}
}

func TestGetCacheMessagesBefore(t *testing.T) {
	indexes := []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46}
	buffer := SequenceNumberCatchupBuffer{
		messages:     m.CreateDummyBroadcastMessages(indexes),
		messageCount: int32(len(indexes)),
		limitCatchup: func() bool { return false },
	}

	// Get 0 messages
	bm, err := buffer.getCacheMessagesBefore(0)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if len(bm.Messages) != 0 {
		t.Error("should have returned 0 messages")
	}

	// Get 0 messages
	bm, err = buffer.getCacheMessagesBefore(1)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if len(bm.Messages) != 0 {
		t.Error("should have returned 0 messages")
	}

	// Get first cached message
	bm, err = buffer.getCacheMessagesBefore(40)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if len(bm.Messages) != 1 {
		t.Error("should have returned 1 message")
	}
	if bm.Messages[0].SequenceNumber != 40 {
		t.Error("returned message should have had SequenceNumber 40")
	}

	// Get error for requesting a message not stored in the buffer
	bm, err = buffer.getCacheMessagesBefore(100)
	if err == nil {
		t.Error("expected an error")
	}
	if bm != nil {
		t.Error("expected no BroadcastMessage object to be returned upon error")
	}

	// Get all 7 messages
	bm, err = buffer.getCacheMessagesBefore(46)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if bm == nil {
		t.Fatal("nothing returned")
	}
	if len(bm.Messages) != 7 {
		t.Errorf("expected 7 messages, got %d messages", len(bm.Messages))
	}
	for i, msg := range bm.Messages {
		expectedSeqNum := arbutil.MessageIndex(40 + i)
		if msg.SequenceNumber != expectedSeqNum {
			t.Errorf("expected sequenceNumber %d from message %d, got %d", expectedSeqNum, i+1, msg.SequenceNumber)
		}
	}

	// Test extremes
	bm, err = buffer.getCacheMessagesBefore(arbutil.MessageIndex(^uint64(0)))
	if err == nil {
		t.Error("expected an error")
	}
	if bm != nil {
		t.Error("expected no BroadcastMessage object to be returned upon error")
	}
}

func TestDeleteConfirmedNil(t *testing.T) {
	buffer := SequenceNumberCatchupBuffer{
		messages:     nil,
		messageCount: 0,
		limitCatchup: func() bool { return false },
		maxCatchup:   func() int { return -1 },
	}

	buffer.deleteConfirmed(0)
	if len(buffer.messages) != 0 {
		t.Error("nothing should be present")
	}
}

func TestDeleteConfirmInvalidOrder(t *testing.T) {
	indexes := []arbutil.MessageIndex{40, 42}
	buffer := SequenceNumberCatchupBuffer{
		messages:     m.CreateDummyBroadcastMessages(indexes),
		messageCount: int32(len(indexes)),
		limitCatchup: func() bool { return false },
		maxCatchup:   func() int { return -1 },
	}

	// Confirm before cache
	buffer.deleteConfirmed(41)
	if len(buffer.messages) != 0 {
		t.Error("cache not in contiguous order should have caused everything to be deleted")
	}
}

func TestDeleteConfirmed(t *testing.T) {
	indexes := []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46}
	buffer := SequenceNumberCatchupBuffer{
		messages:     m.CreateDummyBroadcastMessages(indexes),
		messageCount: int32(len(indexes)),
		limitCatchup: func() bool { return false },
		maxCatchup:   func() int { return -1 },
	}

	// Confirm older than cache
	buffer.deleteConfirmed(39)
	if len(buffer.messages) != 7 {
		t.Error("nothing should have been deleted")
	}

}
func TestDeleteFreeMem(t *testing.T) {
	indexes := []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51}
	buffer := SequenceNumberCatchupBuffer{
		messages:     m.CreateDummyBroadcastMessagesImpl(indexes, len(indexes)*10+1),
		messageCount: int32(len(indexes)),
		limitCatchup: func() bool { return false },
		maxCatchup:   func() int { return -1 },
	}

	// Confirm older than cache
	buffer.deleteConfirmed(40)
	if cap(buffer.messages) > 20 {
		t.Error("extra memory was not freed, cap: ", cap(buffer.messages))
	}

}

func TestBroadcastBadMessage(t *testing.T) {
	buffer := SequenceNumberCatchupBuffer{
		messages:     nil,
		messageCount: 0,
		limitCatchup: func() bool { return false },
		maxCatchup:   func() int { return -1 },
	}

	var foo int
	err := buffer.OnDoBroadcast(foo)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "unknown type") {
		t.Error("unexpected type")
	}
}

func TestBroadcastPastSeqNum(t *testing.T) {
	indexes := []arbutil.MessageIndex{40}
	buffer := SequenceNumberCatchupBuffer{
		messages:     m.CreateDummyBroadcastMessagesImpl(indexes, len(indexes)*10+1),
		messageCount: int32(len(indexes)),
		limitCatchup: func() bool { return false },
		maxCatchup:   func() int { return -1 },
	}

	bm := m.BroadcastMessage{
		Messages: []*m.BroadcastFeedMessage{
			{
				SequenceNumber: 39,
			},
		},
	}
	err := buffer.OnDoBroadcast(bm)
	if err != nil {
		t.Error("expected error")
	}

}

func TestBroadcastFutureSeqNum(t *testing.T) {
	indexes := []arbutil.MessageIndex{40}
	buffer := SequenceNumberCatchupBuffer{
		messages:     m.CreateDummyBroadcastMessagesImpl(indexes, len(indexes)*10+1),
		messageCount: int32(len(indexes)),
		limitCatchup: func() bool { return false },
		maxCatchup:   func() int { return -1 },
	}

	bm := m.BroadcastMessage{
		Messages: []*m.BroadcastFeedMessage{
			{
				SequenceNumber: 42,
			},
		},
	}
	err := buffer.OnDoBroadcast(bm)
	if err != nil {
		t.Error("expected error")
	}

}

func TestMaxCatchupBufferSize(t *testing.T) {
	limit := 5
	buffer := SequenceNumberCatchupBuffer{
		messages:     nil,
		messageCount: 0,
		limitCatchup: func() bool { return false },
		maxCatchup:   func() int { return limit },
	}

	firstMessage := 10
	for i := firstMessage; i <= 20; i += 2 {
		bm := BroadcastMessage{
			Messages: []*BroadcastFeedMessage{
				{
					SequenceNumber: arbutil.MessageIndex(i),
				},
				{
					SequenceNumber: arbutil.MessageIndex(i + 1),
				},
			},
		}
		err := buffer.OnDoBroadcast(bm)
		Require(t, err)
		haveMessages := buffer.getCacheMessages(0)
		expectedCount := arbmath.MinInt(i+len(bm.Messages)-firstMessage, limit)
		if len(haveMessages.Messages) != expectedCount {
			t.Errorf("after broadcasting messages %v and %v, expected to have %v messages but got %v", i, i+1, expectedCount, len(haveMessages.Messages))
		}
		expectedFirstMessage := arbutil.MessageIndex(arbmath.MaxInt(firstMessage, i+len(bm.Messages)-limit))
		if haveMessages.Messages[0].SequenceNumber != expectedFirstMessage {
			t.Errorf("after broadcasting messages %v and %v, expected the first message to be %v but got %v", i, i+1, expectedFirstMessage, haveMessages.Messages[0].SequenceNumber)
		}
	}
}
