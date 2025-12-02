package melextraction

import (
	"context"
	"errors"
	"io"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/daprovider"
)

func TestExtractMessages(t *testing.T) {
	ctx := context.Background()
	prevParentBlockHash := common.HexToHash("0x1234")

	tests := []struct {
		name                 string
		melStateParentHash   common.Hash
		useExtractMessages   bool // If true, use ExtractMessages instead of extractMessagesImpl
		lookupBatches        func(context.Context, *mel.State, *types.Header, TransactionsFetcher, ReceiptFetcher, eventUnpacker) ([]*mel.SequencerInboxBatch, []*types.Transaction, []uint, error)
		lookupDelayedMsgs    func(context.Context, *mel.State, *types.Header, ReceiptFetcher, TransactionsFetcher) ([]*mel.DelayedInboxMessage, error)
		serializer           func(context.Context, *mel.SequencerInboxBatch, *types.Transaction, uint, ReceiptFetcher) ([]byte, error)
		parseReport          func(io.Reader) (*big.Int, common.Address, common.Hash, uint64, *big.Int, uint64, error)
		parseSequencerMsg    func(context.Context, uint64, common.Hash, []byte, arbstate.DapReaderSource, daprovider.KeysetValidationMode) (*arbstate.SequencerMessage, error)
		extractBatchMessages func(context.Context, *mel.State, *arbstate.SequencerMessage, DelayedMessageDatabase) ([]*arbostypes.MessageWithMetadata, error)
		expectedError        string
		expectedMsgCount     uint64
		expectedDelayedSeen  uint64
		expectedMessages     int
		expectedDelayedMsgs  int
	}{
		{
			name:               "parent chain block hash mismatch",
			melStateParentHash: common.HexToHash("0x5678"), // Different from block's parent hash
			useExtractMessages: true,
			expectedError:      "parent chain block hash in MEL state does not match",
		},
		{
			name:               "looking up batches fails",
			melStateParentHash: prevParentBlockHash,
			lookupBatches:      failingLookupBatches,
			lookupDelayedMsgs:  successfulLookupDelayedMsgs,
			expectedError:      "failed to lookup batches",
		},
		{
			name:               "looking up delayed messages fails",
			melStateParentHash: prevParentBlockHash,
			lookupBatches:      emptyLookupBatches,
			lookupDelayedMsgs:  failingLookupDelayedMsgs,
			expectedError:      "failed to lookup delayed messages",
		},
		{
			name:               "mismatched number of batch posting reports vs batches",
			melStateParentHash: prevParentBlockHash,
			lookupBatches:      emptyLookupBatches,          // 0 batches
			lookupDelayedMsgs:  successfulLookupDelayedMsgs, // 1 batch posting report
			expectedError:      "batch posting reports 1 do not match the number of batches 0",
		},
		{
			name:               "batch serialization fails",
			melStateParentHash: prevParentBlockHash,
			lookupBatches:      successfulLookupBatches,
			lookupDelayedMsgs:  successfulLookupDelayedMsgs,
			serializer:         failingSerializer,
			expectedError:      "serialization error",
		},
		{
			name:               "parsing batch posting report fails",
			melStateParentHash: prevParentBlockHash,
			lookupBatches:      successfulLookupBatches,
			lookupDelayedMsgs:  successfulLookupDelayedMsgs,
			serializer:         emptySerializer,
			parseReport:        failingParseReport,
			expectedError:      "batch posting report parsing error",
		},
		{
			name:               "mismatched batch posting report batch hash and actual batch hash",
			melStateParentHash: prevParentBlockHash,
			lookupBatches:      successfulLookupBatches,
			lookupDelayedMsgs:  successfulLookupDelayedMsgs,
			serializer:         emptySerializer,  // Returns nil, hash will be different from parseReport
			parseReport:        emptyParseReport, // Returns empty hash
			expectedError:      "batch data hash incorrect",
		},
		{
			name:               "parse sequencer message fails",
			melStateParentHash: prevParentBlockHash,
			lookupBatches:      successfulLookupBatches,
			lookupDelayedMsgs:  successfulLookupDelayedMsgs,
			serializer:         successfulSerializer,
			parseReport:        successfulParseReport,
			parseSequencerMsg:  failingParseSequencerMsg,
			expectedError:      "failed to parse sequencer message",
		},
		{
			name:                 "extracting batch messages fails",
			melStateParentHash:   prevParentBlockHash,
			lookupBatches:        successfulLookupBatches,
			lookupDelayedMsgs:    successfulLookupDelayedMsgs,
			serializer:           successfulSerializer,
			parseReport:          successfulParseReport,
			parseSequencerMsg:    successfulParseSequencerMsg,
			extractBatchMessages: failingExtractBatchMessages,
			expectedError:        "failed to extract batch messages",
		},
		{
			name:                 "OK",
			melStateParentHash:   prevParentBlockHash,
			lookupBatches:        successfulLookupBatches,
			lookupDelayedMsgs:    successfulLookupDelayedMsgs,
			serializer:           successfulSerializer,
			parseReport:          successfulParseReport,
			parseSequencerMsg:    successfulParseSequencerMsg,
			extractBatchMessages: successfulExtractBatchMessages,
			expectedMsgCount:     2,
			expectedDelayedSeen:  1,
			expectedMessages:     2,
			expectedDelayedMsgs:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := createBlockHeader(prevParentBlockHash)
			melState := createMelState(tt.melStateParentHash)
			txsFetcher := &mockTxsFetcher{}

			var postState *mel.State
			var messages []*arbostypes.MessageWithMetadata
			var delayedMessages []*mel.DelayedInboxMessage
			var err error

			if tt.useExtractMessages {
				// Test the public ExtractMessages function
				postState, messages, delayedMessages, err = ExtractMessages(
					ctx,
					melState,
					header,
					nil,
					nil,
					nil,
					txsFetcher,
				)
			} else {
				// Test the internal extractMessagesImpl function
				postState, messages, delayedMessages, err = extractMessagesImpl(
					ctx,
					melState,
					header,
					nil,
					nil,
					txsFetcher,
					nil,
					nil,
					tt.lookupBatches,
					tt.lookupDelayedMsgs,
					tt.serializer,
					tt.extractBatchMessages,
					tt.parseSequencerMsg,
					tt.parseReport,
				)
			}

			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectedMsgCount, postState.MsgCount)
			require.Equal(t, tt.expectedDelayedSeen, postState.DelayedMessagesSeen)
			require.Len(t, messages, tt.expectedMessages)
			require.Len(t, delayedMessages, tt.expectedDelayedMsgs)
		})
	}
}

func createMelState(parentHash common.Hash) *mel.State {
	melState := &mel.State{
		ParentChainBlockHash: parentHash,
	}
	return melState
}

func createBlockHeader(parentHash common.Hash) *types.Header {
	return &types.Header{
		ParentHash: parentHash,
		Number:     big.NewInt(0),
	}
}

// Mock functions
func successfulLookupBatches(
	ctx context.Context,
	melState *mel.State,
	parentChainBlock *types.Header,
	txsFetcher TransactionsFetcher,
	receiptFetcher ReceiptFetcher,
	eventUnpacker eventUnpacker,
) ([]*mel.SequencerInboxBatch, []*types.Transaction, []uint, error) {
	batches := []*mel.SequencerInboxBatch{{}}
	txs := []*types.Transaction{{}}
	txIndices := []uint{0}
	return batches, txs, txIndices, nil
}

func emptyLookupBatches(
	ctx context.Context,
	melState *mel.State,
	parentChainBlock *types.Header,
	txsFetcher TransactionsFetcher,
	receiptFetcher ReceiptFetcher,
	eventUnpacker eventUnpacker,
) ([]*mel.SequencerInboxBatch, []*types.Transaction, []uint, error) {
	return nil, nil, nil, nil
}

func failingLookupBatches(
	ctx context.Context,
	melState *mel.State,
	parentChainBlock *types.Header,
	txsFetcher TransactionsFetcher,
	receiptFetcher ReceiptFetcher,
	eventUnpacker eventUnpacker,
) ([]*mel.SequencerInboxBatch, []*types.Transaction, []uint, error) {
	return nil, nil, nil, errors.New("failed to lookup batches")
}

func successfulLookupDelayedMsgs(
	ctx context.Context,
	melState *mel.State,
	parentChainBlock *types.Header,
	receiptFetcher ReceiptFetcher,
	txsFetcher TransactionsFetcher,
) ([]*mel.DelayedInboxMessage, error) {
	hash := common.MaxHash
	delayedMsgs := []*mel.DelayedInboxMessage{
		{
			Message: &arbostypes.L1IncomingMessage{
				L2msg: []byte("foobar"),
				Header: &arbostypes.L1IncomingMessageHeader{
					Kind:      arbostypes.L1MessageType_BatchPostingReport,
					RequestId: &hash,
					L1BaseFee: common.Big0,
				},
			},
		},
	}
	return delayedMsgs, nil
}

func failingLookupDelayedMsgs(
	ctx context.Context,
	melState *mel.State,
	parentChainBlock *types.Header,
	receiptFetcher ReceiptFetcher,
	txsFetcher TransactionsFetcher,
) ([]*mel.DelayedInboxMessage, error) {
	return nil, errors.New("failed to lookup delayed messages")
}

func successfulSerializer(ctx context.Context,
	batch *mel.SequencerInboxBatch,
	tx *types.Transaction,
	txIndex uint,
	receiptFetcher ReceiptFetcher,
) ([]byte, error) {
	return []byte("foobar"), nil
}

func emptySerializer(ctx context.Context,
	batch *mel.SequencerInboxBatch,
	tx *types.Transaction,
	txIndex uint,
	receiptFetcher ReceiptFetcher,
) ([]byte, error) {
	return nil, nil
}

func failingSerializer(ctx context.Context,
	batch *mel.SequencerInboxBatch,
	tx *types.Transaction,
	txIndex uint,
	receiptFetcher ReceiptFetcher,
) ([]byte, error) {
	return nil, errors.New("serialization error")
}

func successfulParseReport(
	rd io.Reader,
) (*big.Int, common.Address, common.Hash, uint64, *big.Int, uint64, error) {
	return nil, common.Address{}, crypto.Keccak256Hash([]byte("foobar")), 0, nil, 0, nil
}

func emptyParseReport(
	rd io.Reader,
) (*big.Int, common.Address, common.Hash, uint64, *big.Int, uint64, error) {
	return nil, common.Address{}, common.Hash{}, 0, nil, 0, nil
}

func failingParseReport(
	rd io.Reader,
) (*big.Int, common.Address, common.Hash, uint64, *big.Int, uint64, error) {
	return nil, common.Address{}, common.Hash{}, 0, nil, 0, errors.New("batch posting report parsing error")
}

func successfulParseSequencerMsg(
	ctx context.Context,
	batchNum uint64,
	batchBlockHash common.Hash,
	data []byte,
	dapReaders arbstate.DapReaderSource,
	keysetValidationMode daprovider.KeysetValidationMode,
) (*arbstate.SequencerMessage, error) {
	return nil, nil
}

func failingParseSequencerMsg(
	ctx context.Context,
	batchNum uint64,
	batchBlockHash common.Hash,
	data []byte,
	dapReaders arbstate.DapReaderSource,
	keysetValidationMode daprovider.KeysetValidationMode,
) (*arbstate.SequencerMessage, error) {
	return nil, errors.New("failed to parse sequencer message")
}

func successfulExtractBatchMessages(
	ctx context.Context,
	melState *mel.State,
	seqMsg *arbstate.SequencerMessage,
	delayedMsgDB DelayedMessageDatabase,
) ([]*arbostypes.MessageWithMetadata, error) {
	return []*arbostypes.MessageWithMetadata{
		{
			Message: &arbostypes.L1IncomingMessage{
				L2msg: []byte("foobar"),
				Header: &arbostypes.L1IncomingMessageHeader{
					Kind: arbostypes.L1MessageType_L2Message,
				},
			},
		},
		{
			Message: &arbostypes.L1IncomingMessage{
				L2msg: []byte("nyancat"),
				Header: &arbostypes.L1IncomingMessageHeader{
					Kind: arbostypes.L1MessageType_L2Message,
				},
			},
		},
	}, nil
}

func failingExtractBatchMessages(
	ctx context.Context,
	melState *mel.State,
	seqMsg *arbstate.SequencerMessage,
	delayedMsgDB DelayedMessageDatabase,
) ([]*arbostypes.MessageWithMetadata, error) {
	return nil, errors.New("failed to extract batch messages")
}
