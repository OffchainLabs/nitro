package melextraction

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
)

func Test_extractMessagesInBatch_delayedMessages(t *testing.T) {
	ctx := context.Background()
	melState := &mel.State{
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

	tests := []struct {
		name            string
		setupSegments   func() [][]byte
		setupMelState   func() *mel.State
		setupSeqMsg     func(segments [][]byte) *arbstate.SequencerMessage
		setupMockDB     func() *mockDelayedMessageDB
		wantErr         bool
		wantErrContains string
		validateResult  func(t *testing.T, msg *arbostypes.MessageWithMetadata)
	}{
		{
			name: "segment advances timestamp",
			setupSegments: func() [][]byte {
				compressedData, _ := arbcompress.CompressWell([]byte("foobar"))
				encodedTimestampAdvance, _ := rlp.EncodeToBytes(uint64(50))
				return [][]byte{
					append([]byte{arbstate.BatchSegmentKindAdvanceTimestamp}, encodedTimestampAdvance...),
					append([]byte{arbstate.BatchSegmentKindL2MessageBrotli}, compressedData...),
				}
			},
			setupMelState: func() *mel.State {
				return &mel.State{}
			},
			setupSeqMsg: func(segments [][]byte) *arbstate.SequencerMessage {
				return &arbstate.SequencerMessage{
					Segments:     segments,
					MinTimestamp: 0,
					MaxTimestamp: 1_000_000,
				}
			},
			setupMockDB: func() *mockDelayedMessageDB {
				return nil
			},
			wantErr: false,
			validateResult: func(t *testing.T, msg *arbostypes.MessageWithMetadata) {
				require.Equal(t, []byte("foobar"), msg.Message.L2msg)
				require.Equal(t, uint64(50), msg.Message.Header.Timestamp)
			},
		},
		{
			name: "segment advances blocknum",
			setupSegments: func() [][]byte {
				compressedData, _ := arbcompress.CompressWell([]byte("foobar"))
				encodedBlockNum, _ := rlp.EncodeToBytes(uint64(20))
				return [][]byte{
					append([]byte{arbstate.BatchSegmentKindAdvanceL1BlockNumber}, encodedBlockNum...),
					append([]byte{arbstate.BatchSegmentKindL2MessageBrotli}, compressedData...),
				}
			},
			setupMelState: func() *mel.State {
				return &mel.State{}
			},
			setupSeqMsg: func(segments [][]byte) *arbstate.SequencerMessage {
				return &arbstate.SequencerMessage{
					Segments:     segments,
					MinTimestamp: 0,
					MaxTimestamp: 1_000_000,
					MaxL1Block:   1_000_000,
				}
			},
			setupMockDB: func() *mockDelayedMessageDB {
				return nil
			},
			wantErr: false,
			validateResult: func(t *testing.T, msg *arbostypes.MessageWithMetadata) {
				require.Equal(t, []byte("foobar"), msg.Message.L2msg)
				require.Equal(t, uint64(20), msg.Message.Header.BlockNumber)
			},
		},
		{
			name: "brotli compressed message",
			setupSegments: func() [][]byte {
				compressedData, _ := arbcompress.CompressWell([]byte("foobar"))
				encodedTimestampAdvance := make([]byte, 8)
				binary.BigEndian.PutUint64(encodedTimestampAdvance, 50)
				return [][]byte{
					append([]byte{arbstate.BatchSegmentKindAdvanceTimestamp}, encodedTimestampAdvance...),
					append([]byte{arbstate.BatchSegmentKindL2MessageBrotli}, compressedData...),
				}
			},
			setupMelState: func() *mel.State {
				return &mel.State{}
			},
			setupSeqMsg: func(segments [][]byte) *arbstate.SequencerMessage {
				return &arbstate.SequencerMessage{
					Segments:     segments,
					MinTimestamp: 0,
					MaxTimestamp: 1_000_000,
				}
			},
			setupMockDB: func() *mockDelayedMessageDB {
				return nil
			},
			wantErr: false,
			validateResult: func(t *testing.T, msg *arbostypes.MessageWithMetadata) {
				require.Equal(t, []byte("foobar"), msg.Message.L2msg)
			},
		},
		{
			name: "delayed message segment greater than what has been read",
			setupSegments: func() [][]byte {
				return [][]byte{{}}
			},
			setupMelState: func() *mel.State {
				return &mel.State{
					DelayedMessagesRead: 1,
				}
			},
			setupSeqMsg: func(segments [][]byte) *arbstate.SequencerMessage {
				return &arbstate.SequencerMessage{
					AfterDelayedMessages: 1,
					Segments:             segments,
				}
			},
			setupMockDB: func() *mockDelayedMessageDB {
				return nil
			},
			wantErr: false,
			validateResult: func(t *testing.T, msg *arbostypes.MessageWithMetadata) {
				require.Equal(t, arbostypes.InvalidL1Message, msg.Message)
			},
		},
		{
			name: "gets error fetching delayed message from database",
			setupSegments: func() [][]byte {
				return [][]byte{{}}
			},
			setupMelState: func() *mel.State {
				return &mel.State{
					DelayedMessagesRead: 0,
				}
			},
			setupSeqMsg: func(segments [][]byte) *arbstate.SequencerMessage {
				return &arbstate.SequencerMessage{
					AfterDelayedMessages: 1,
					Segments:             segments,
				}
			},
			setupMockDB: func() *mockDelayedMessageDB {
				return &mockDelayedMessageDB{
					err: errors.New("oops"),
				}
			},
			wantErr:         true,
			wantErrContains: "oops",
		},
		{
			name: "gets nil delayed message from database",
			setupSegments: func() [][]byte {
				return [][]byte{{}}
			},
			setupMelState: func() *mel.State {
				return &mel.State{
					DelayedMessagesRead: 0,
				}
			},
			setupSeqMsg: func(segments [][]byte) *arbstate.SequencerMessage {
				return &arbstate.SequencerMessage{
					AfterDelayedMessages: 1,
					Segments:             segments,
				}
			},
			setupMockDB: func() *mockDelayedMessageDB {
				return &mockDelayedMessageDB{
					DelayedMessages: map[uint64]*arbnode.DelayedInboxMessage{},
				}
			},
			wantErr:         true,
			wantErrContains: "no more delayed messages in db",
		},
		{
			name: "reading delayed message OK",
			setupSegments: func() [][]byte {
				return [][]byte{{}}
			},
			setupMelState: func() *mel.State {
				return &mel.State{
					DelayedMessagesRead: 0,
				}
			},
			setupSeqMsg: func(segments [][]byte) *arbstate.SequencerMessage {
				return &arbstate.SequencerMessage{
					AfterDelayedMessages: 1,
					Segments:             segments,
				}
			},
			setupMockDB: func() *mockDelayedMessageDB {
				return &mockDelayedMessageDB{
					DelayedMessages: map[uint64]*arbnode.DelayedInboxMessage{
						0: {
							Message: &arbostypes.L1IncomingMessage{
								L2msg: []byte("foobar"),
							},
						},
					},
				}
			},
			wantErr: false,
			validateResult: func(t *testing.T, msg *arbostypes.MessageWithMetadata) {
				require.Equal(t, []byte("foobar"), msg.Message.L2msg)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments := tt.setupSegments()
			melState := tt.setupMelState()
			seqMsg := tt.setupSeqMsg(segments)
			mockDB := tt.setupMockDB()

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

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrContains != "" {
					require.ErrorContains(t, err, tt.wantErrContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.validateResult != nil {
				tt.validateResult(t, msg)
			}
		})
	}
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
				melState: &mel.State{
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
				melState: &mel.State{
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
				melState: &mel.State{
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
				melState: &mel.State{
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
	_ *mel.State,
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
