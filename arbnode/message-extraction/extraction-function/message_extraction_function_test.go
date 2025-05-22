package extractionfunction

import (
	"testing"

	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/arbstate"
)

func Test_isLastSegment(t *testing.T) {
	tests := []struct {
		name     string
		p        *arbosExtractionParams
		expected bool
	}{
		{
			name: "less than after delayed messages",
			p: &arbosExtractionParams{
				melState: &meltypes.State{
					DelayedMessagesRead: 1,
				},
				seqMsg: &arbstate.SequencerMessage{
					AfterDelayedMessages: 2,
					Segments:             [][]byte{{0}, {0}},
				},
				segmentNum: 0,
			},
			expected: false,
		},
		{
			name: "first segment is zero and next one is brotli message",
			p: &arbosExtractionParams{
				melState: &meltypes.State{
					DelayedMessagesRead: 1,
				},
				seqMsg: &arbstate.SequencerMessage{
					AfterDelayedMessages: 1,
					Segments:             [][]byte{{}, {arbstate.BatchSegmentKindL2MessageBrotli}},
				},
				segmentNum: 0,
			},
			expected: false,
		},
		{
			name: "segment is delayed message kind",
			p: &arbosExtractionParams{
				melState: &meltypes.State{
					DelayedMessagesRead: 1,
				},
				seqMsg: &arbstate.SequencerMessage{
					AfterDelayedMessages: 1,
					Segments:             [][]byte{{}, {arbstate.BatchSegmentKindDelayedMessages}},
				},
				segmentNum: 0,
			},
			expected: false,
		},
		{
			name: "is last segment",
			p: &arbosExtractionParams{
				melState: &meltypes.State{
					DelayedMessagesRead: 1,
				},
				seqMsg: &arbstate.SequencerMessage{
					AfterDelayedMessages: 1,
					Segments:             [][]byte{{}, {}},
				},
				segmentNum: 0,
			},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := isLastSegment(test.p)
			if result != test.expected {
				t.Errorf("expected %v, got %v", test.expected, result)
			}
		})
	}
}
