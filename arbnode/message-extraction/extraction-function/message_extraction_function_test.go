package extractionfunction

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/stretchr/testify/require"
)

func TestExtractMessages(t *testing.T) {
	t.Run("parent chain block hash mismatch", func(t *testing.T) {
		prevParentBlockHash := common.Hex2Bytes("0x1234")
		block := types.NewBlock(
			&types.Header{
				ParentHash: common.Hash(prevParentBlockHash),
			},
			&types.Body{},
			nil,
			trie.NewStackTrie(nil),
		)
		_ = block
	})
}

func Test_extractMessagesInBatch(t *testing.T) {
	ctx := context.Background()
	melState := &meltypes.State{
		DelayedMessagesRead: 0,
	}
	seqMsg := &arbstate.SequencerMessage{
		AfterDelayedMessages: 2,
		Segments: [][]byte{
			{}, {},
		},
	}
	mockDB := &mockDelayedMessageDB{
		DelayedMessages: map[uint64]*arbnode.DelayedInboxMessage{
			0: {
				Message: &arbostypes.L1IncomingMessage{
					L2msg: []byte("foobar"),
				},
			},
			1: {
				Message: &arbostypes.L1IncomingMessage{
					L2msg: []byte("barfoo"),
				},
			},
		},
	}
	msgs, err := extractMessagesInBatch(
		ctx,
		melState,
		seqMsg,
		mockDB,
	)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	require.Equal(t, msgs[0].Message.L2msg, []byte("foobar"))
	require.Equal(t, msgs[1].Message.L2msg, []byte("barfoo"))
}

func Test_extractArbosMessage(t *testing.T) {
	ctx := context.Background()
	t.Run("segment advances timestamp", func(t *testing.T) {
		compressedData, err := arbcompress.CompressWell([]byte("foobar"))
		require.NoError(t, err)
		encodedTimestampAdvance, err := rlp.EncodeToBytes(uint64(50))
		require.NoError(t, err)
		melState := &meltypes.State{}
		seqMsg := &arbstate.SequencerMessage{
			Segments: [][]byte{
				append([]byte{arbstate.BatchSegmentKindAdvanceTimestamp}, encodedTimestampAdvance...),
				append([]byte{arbstate.BatchSegmentKindL2MessageBrotli}, compressedData...),
			},
			MinTimestamp: 0,
			MaxTimestamp: 1_000_000,
		}
		params := &arbosExtractionParams{
			melState:         melState,
			seqMsg:           seqMsg,
			delayedMsgDB:     nil,
			targetSubMessage: 0,
			segmentNum:       0,
			blockNumber:      0,
			timestamp:        0,
		}
		msg, _, err := extractArbosMessage(ctx, params)
		require.NoError(t, err)
		require.Equal(t, msg.Message.L2msg, []byte("foobar"))
		require.Equal(t, msg.Message.Header.Timestamp, uint64(50))
	})
	t.Run("segment advances blocknum", func(t *testing.T) {
		compressedData, err := arbcompress.CompressWell([]byte("foobar"))
		require.NoError(t, err)
		encodedBlockNum, err := rlp.EncodeToBytes(uint64(20))
		require.NoError(t, err)
		melState := &meltypes.State{}
		seqMsg := &arbstate.SequencerMessage{
			Segments: [][]byte{
				append([]byte{arbstate.BatchSegmentKindAdvanceL1BlockNumber}, encodedBlockNum...),
				append([]byte{arbstate.BatchSegmentKindL2MessageBrotli}, compressedData...),
			},
			MinTimestamp: 0,
			MaxTimestamp: 1_000_000,
			MaxL1Block:   1_000_000,
		}
		params := &arbosExtractionParams{
			melState:         melState,
			seqMsg:           seqMsg,
			delayedMsgDB:     nil,
			targetSubMessage: 0,
			segmentNum:       0,
			blockNumber:      0,
			timestamp:        0,
		}
		msg, _, err := extractArbosMessage(ctx, params)
		require.NoError(t, err)
		require.Equal(t, msg.Message.L2msg, []byte("foobar"))
		require.Equal(t, msg.Message.Header.BlockNumber, uint64(20))
	})
	t.Run("brotli compressed message", func(t *testing.T) {
		compressedData, err := arbcompress.CompressWell([]byte("foobar"))
		require.NoError(t, err)
		encodedTimestampAdvance := make([]byte, 8)
		binary.BigEndian.PutUint64(encodedTimestampAdvance, 50)
		melState := &meltypes.State{}
		seqMsg := &arbstate.SequencerMessage{
			Segments: [][]byte{
				append([]byte{arbstate.BatchSegmentKindAdvanceTimestamp}, encodedTimestampAdvance...),
				append([]byte{arbstate.BatchSegmentKindL2MessageBrotli}, compressedData...),
			},
			MinTimestamp: 0,
			MaxTimestamp: 1_000_000,
		}
		params := &arbosExtractionParams{
			melState:         melState,
			seqMsg:           seqMsg,
			delayedMsgDB:     nil,
			targetSubMessage: 0,
			segmentNum:       0,
			blockNumber:      0,
			timestamp:        0,
		}
		msg, _, err := extractArbosMessage(ctx, params)
		require.NoError(t, err)
		require.Equal(t, msg.Message.L2msg, []byte("foobar"))
	})
	t.Run("delayed message segment greater than what has been read", func(t *testing.T) {
		melState := &meltypes.State{
			DelayedMessagesRead: 1,
		}
		seqMsg := &arbstate.SequencerMessage{
			AfterDelayedMessages: 1,
			Segments: [][]byte{
				{},
			},
		}
		params := &arbosExtractionParams{
			melState:         melState,
			seqMsg:           seqMsg,
			delayedMsgDB:     nil,
			targetSubMessage: 0,
			segmentNum:       0,
			blockNumber:      0,
			timestamp:        0,
		}
		msg, _, err := extractArbosMessage(ctx, params)
		require.NoError(t, err)
		require.Equal(t, arbostypes.InvalidL1Message, msg.Message)
	})
	t.Run("gets error fetching delayed message from database", func(t *testing.T) {
		melState := &meltypes.State{
			DelayedMessagesRead: 0,
		}
		seqMsg := &arbstate.SequencerMessage{
			AfterDelayedMessages: 1,
			Segments: [][]byte{
				{},
			},
		}
		mockDB := &mockDelayedMessageDB{
			err: errors.New("oops"),
		}
		params := &arbosExtractionParams{
			melState:         melState,
			seqMsg:           seqMsg,
			delayedMsgDB:     mockDB,
			targetSubMessage: 0,
			segmentNum:       0,
			blockNumber:      0,
			timestamp:        0,
		}
		_, _, err := extractArbosMessage(ctx, params)
		require.ErrorContains(t, err, "oops")
	})
	t.Run("gets nil delayed message from database", func(t *testing.T) {
		melState := &meltypes.State{
			DelayedMessagesRead: 0,
		}
		seqMsg := &arbstate.SequencerMessage{
			AfterDelayedMessages: 1,
			Segments: [][]byte{
				{},
			},
		}
		mockDB := &mockDelayedMessageDB{
			DelayedMessages: map[uint64]*arbnode.DelayedInboxMessage{},
		}
		params := &arbosExtractionParams{
			melState:         melState,
			seqMsg:           seqMsg,
			delayedMsgDB:     mockDB,
			targetSubMessage: 0,
			segmentNum:       0,
			blockNumber:      0,
			timestamp:        0,
		}
		_, _, err := extractArbosMessage(ctx, params)
		require.ErrorContains(t, err, "no more delayed messages in db")
	})
	t.Run("reading delayed message OK", func(t *testing.T) {
		melState := &meltypes.State{
			DelayedMessagesRead: 0,
		}
		seqMsg := &arbstate.SequencerMessage{
			AfterDelayedMessages: 1,
			Segments: [][]byte{
				{},
			},
		}
		mockDB := &mockDelayedMessageDB{
			DelayedMessages: map[uint64]*arbnode.DelayedInboxMessage{
				0: {
					Message: &arbostypes.L1IncomingMessage{
						L2msg: []byte("foobar"),
					},
				},
			},
		}
		params := &arbosExtractionParams{
			melState:         melState,
			seqMsg:           seqMsg,
			delayedMsgDB:     mockDB,
			targetSubMessage: 0,
			segmentNum:       0,
			blockNumber:      0,
			timestamp:        0,
		}
		msg, _, err := extractArbosMessage(ctx, params)
		require.NoError(t, err)
		require.Equal(t, []byte("foobar"), msg.Message.L2msg)
	})
}

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

type mockDelayedMessageDB struct {
	DelayedMessagesRead uint64
	DelayedMessages     map[uint64]*arbnode.DelayedInboxMessage
	err                 error
}

func (m *mockDelayedMessageDB) ReadDelayedMessage(
	_ context.Context,
	_ *meltypes.State,
	delayedMsgsRead uint64,
) (*arbnode.DelayedInboxMessage, error) {
	if m.err != nil {
		return nil, m.err
	}
	if delayedMsg, ok := m.DelayedMessages[delayedMsgsRead]; ok {
		return delayedMsg, nil
	}
	return nil, nil
}
