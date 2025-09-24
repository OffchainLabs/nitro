// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package melextraction

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
)

func Test_messagesFromBatchSegments_expectedFieldBounds_simple(t *testing.T) {
	compressedData, _ := arbcompress.CompressWell([]byte("foobar"))
	adv1, _ := rlp.EncodeToBytes(uint64(7))
	adv2, _ := rlp.EncodeToBytes(uint64(3))
	segments := [][]byte{
		append([]byte{arbstate.BatchSegmentKindAdvanceTimestamp}, adv1...),
		append([]byte{arbstate.BatchSegmentKindAdvanceTimestamp}, adv2...),
		append([]byte{arbstate.BatchSegmentKindL2MessageBrotli}, compressedData...),
	}
	ctx := context.Background()
	melState := &mel.State{
		DelayedMessagesRead: 0,
	}
	seqMsg := sequencerMessageWithTimestampRange(segments, 10, 20)
	mockDB := &mockDelayedMessageDB{}
	msgs, err := messagesFromBatchSegments(
		ctx,
		melState,
		seqMsg,
		mockDB,
	)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, msgs[0].Message.L2msg, []byte("foobar"))
	require.Equal(t, msgs[0].Message.Header.Timestamp, uint64(10))
}

func Test_messagesFromBatchSegments_expectedFieldBounds_complex(t *testing.T) {
	compressedData, _ := arbcompress.CompressWell([]byte("foobar"))
	adv1, _ := rlp.EncodeToBytes(uint64(7))
	adv2, _ := rlp.EncodeToBytes(uint64(3))
	segments := [][]byte{
		append([]byte{arbstate.BatchSegmentKindAdvanceTimestamp}, adv1...),
		append([]byte{arbstate.BatchSegmentKindL2MessageBrotli}, compressedData...),
		append([]byte{arbstate.BatchSegmentKindAdvanceTimestamp}, adv2...),
		append([]byte{arbstate.BatchSegmentKindL2MessageBrotli}, compressedData...),
	}
	ctx := context.Background()
	melState := &mel.State{
		DelayedMessagesRead: 0,
	}
	seqMsg := sequencerMessageWithTimestampRange(segments, 10, 20)
	mockDB := &mockDelayedMessageDB{}
	msgs, err := messagesFromBatchSegments(
		ctx,
		melState,
		seqMsg,
		mockDB,
	)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	require.Equal(t, msgs[0].Message.L2msg, []byte("foobar"))
	require.Equal(t, msgs[0].Message.Header.Timestamp, uint64(10))
	require.Equal(t, msgs[1].Message.L2msg, []byte("foobar"))
	require.Equal(t, msgs[1].Message.Header.Timestamp, uint64(10))
}

func Test_messagesFromBatchSegments_delayedMessages(t *testing.T) {
	ctx := context.Background()
	melState := &mel.State{
		DelayedMessagesRead: 0,
	}
	// No segments, but the sequencer message says that we must read 2 delayed messages.
	seqMsg := sequencerMessageWithSegments(2, [][]byte{})
	mockDB := &mockDelayedMessageDB{
		DelayedMessages: map[uint64]*mel.DelayedInboxMessage{
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
	msgs, err := messagesFromBatchSegments(
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

func Test_messagesFromBatchSegments(t *testing.T) {
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
				return sequencerMessageWithTimestampRange(segments, 0, 1_000_000)
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
					MinTimestamp:         0,
					MaxTimestamp:         1_000_000,
					MinL1Block:           0,
					MaxL1Block:           1_000_000,
					AfterDelayedMessages: 0,
					Segments:             segments,
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
				return sequencerMessageWithTimestampRange(segments, 0, 1_000_000)
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
				return [][]byte{
					{arbstate.BatchSegmentKindDelayedMessages},
				}
			},
			setupMelState: func() *mel.State {
				return &mel.State{
					DelayedMessagesRead: 1,
				}
			},
			setupSeqMsg: func(segments [][]byte) *arbstate.SequencerMessage {
				return sequencerMessageWithSegments(1, segments)
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
				return sequencerMessageWithSegments(1, segments)
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
				return sequencerMessageWithSegments(1, segments)
			},
			setupMockDB: func() *mockDelayedMessageDB {
				return &mockDelayedMessageDB{
					DelayedMessages: map[uint64]*mel.DelayedInboxMessage{},
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
				return sequencerMessageWithSegments(1, segments)
			},
			setupMockDB: func() *mockDelayedMessageDB {
				return &mockDelayedMessageDB{
					DelayedMessages: map[uint64]*mel.DelayedInboxMessage{
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

			msgs, err := messagesFromBatchSegments(
				context.Background(),
				melState,
				seqMsg,
				mockDB,
			)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrContains != "" {
					require.ErrorContains(t, err, tt.wantErrContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.validateResult != nil {
				require.Equal(t, 1, len(msgs), "Expected exactly one message for this test")
				tt.validateResult(t, msgs[0])
			}
		})
	}
}

type mockDelayedMessageDB struct {
	DelayedMessagesRead uint64
	DelayedMessages     map[uint64]*mel.DelayedInboxMessage
	err                 error
}

func (m *mockDelayedMessageDB) ReadDelayedMessage(
	_ context.Context,
	_ *mel.State,
	delayedMsgsRead uint64,
) (*mel.DelayedInboxMessage, error) {
	if m.err != nil {
		return nil, m.err
	}
	if delayedMsg, ok := m.DelayedMessages[delayedMsgsRead]; ok {
		return delayedMsg, nil
	}
	return nil, nil
}

func sequencerMessageWithSegments(afterDelayedMessages uint64, segments [][]byte) *arbstate.SequencerMessage {
	return &arbstate.SequencerMessage{
		MinTimestamp:         0,
		MaxTimestamp:         0,
		MinL1Block:           0,
		MaxL1Block:           0,
		AfterDelayedMessages: afterDelayedMessages,
		Segments:             segments,
	}
}

func sequencerMessageWithTimestampRange(segments [][]byte, minTimestamp, maxTimestamp uint64) *arbstate.SequencerMessage {
	return &arbstate.SequencerMessage{
		MinTimestamp:         minTimestamp,
		MaxTimestamp:         maxTimestamp,
		MinL1Block:           0,
		MaxL1Block:           0,
		AfterDelayedMessages: 0,
		Segments:             segments,
	}
}
