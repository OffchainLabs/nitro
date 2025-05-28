package extractionfunction

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/stretchr/testify/require"
)

func TestExtractMessages(t *testing.T) {
	ctx := context.Background()
	t.Run("parent chain block hash mismatch", func(t *testing.T) {
		prevParentBlockHash := common.HexToHash("0x1234")
		block := types.NewBlock(
			&types.Header{
				ParentHash: prevParentBlockHash,
			},
			&types.Body{},
			nil,
			trie.NewStackTrie(nil),
		)
		melState := &meltypes.State{
			ParentChainBlockHash: common.HexToHash("0x5678"),
		}
		_, _, _, err := ExtractMessages(
			ctx,
			melState,
			block,
			nil,
			nil,
			nil,
		)
		require.ErrorIs(t, err, ErrInvalidParentChainBlock)
	})
	t.Run("looking up batches fails", func(t *testing.T) {
		prevParentBlockHash := common.HexToHash("0x1234")
		block := types.NewBlock(
			&types.Header{
				ParentHash: prevParentBlockHash,
			},
			&types.Body{},
			nil,
			trie.NewStackTrie(nil),
		)
		melState := &meltypes.State{
			ParentChainBlockHash: prevParentBlockHash,
		}
		lookupBatches := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Block,
			receiptFetcher ReceiptFetcher,
			eventUnpacker eventUnpacker,
		) ([]*arbnode.SequencerInboxBatch, []*types.Transaction, []uint, error) {
			return nil, nil, nil, errors.New("failed to lookup batches")
		}
		_, _, _, err := extractMessagesImpl(
			ctx,
			melState,
			block,
			nil,
			nil,
			nil,
			nil,
			lookupBatches,
			nil,
			nil,
			nil,
			nil,
			nil,
		)
		require.ErrorContains(t, err, "failed to lookup batches")
	})
	t.Run("looking up delayed messages fails", func(t *testing.T) {
		prevParentBlockHash := common.HexToHash("0x1234")
		block := types.NewBlock(
			&types.Header{
				ParentHash: prevParentBlockHash,
			},
			&types.Body{},
			nil,
			trie.NewStackTrie(nil),
		)
		melState := &meltypes.State{
			ParentChainBlockHash: prevParentBlockHash,
		}
		lookupBatches := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Block,
			receiptFetcher ReceiptFetcher,
			eventUnpacker eventUnpacker,
		) ([]*arbnode.SequencerInboxBatch, []*types.Transaction, []uint, error) {
			return nil, nil, nil, nil
		}
		lookupDelayedMsgs := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlockNum *big.Int,
			parentChainBlockTxs []*types.Transaction,
			receiptFetcher ReceiptFetcher,
		) ([]*arbnode.DelayedInboxMessage, error) {
			return nil, errors.New("failed to lookup delayed messages")
		}

		_, _, _, err := extractMessagesImpl(
			ctx,
			melState,
			block,
			nil,
			nil,
			nil,
			nil,
			lookupBatches,
			lookupDelayedMsgs,
			nil,
			nil,
			nil,
			nil,
		)
		require.ErrorContains(t, err, "failed to lookup delayed messages")
	})
	t.Run("mismatched number of batch posting reports vs batches", func(t *testing.T) {
		prevParentBlockHash := common.HexToHash("0x1234")
		block := types.NewBlock(
			&types.Header{
				ParentHash: prevParentBlockHash,
			},
			&types.Body{},
			nil,
			trie.NewStackTrie(nil),
		)
		melState := &meltypes.State{
			ParentChainBlockHash: prevParentBlockHash,
		}
		lookupBatches := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Block,
			receiptFetcher ReceiptFetcher,
			eventUnpacker eventUnpacker,
		) ([]*arbnode.SequencerInboxBatch, []*types.Transaction, []uint, error) {
			return nil, nil, nil, nil
		}
		lookupDelayedMsgs := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlockNum *big.Int,
			parentChainBlockTxs []*types.Transaction,
			receiptFetcher ReceiptFetcher,
		) ([]*arbnode.DelayedInboxMessage, error) {
			delayedMsgs := []*arbnode.DelayedInboxMessage{
				{
					Message: &arbostypes.L1IncomingMessage{
						L2msg: []byte("foobar"),
						Header: &arbostypes.L1IncomingMessageHeader{
							Kind: arbostypes.L1MessageType_BatchPostingReport,
						},
					},
				},
			}
			return delayedMsgs, nil
		}

		_, _, _, err := extractMessagesImpl(
			ctx,
			melState,
			block,
			nil,
			nil,
			nil,
			nil,
			lookupBatches,
			lookupDelayedMsgs,
			nil,
			nil,
			nil,
			nil,
		)
		require.ErrorContains(t, err, "batch posting reports 1 do not match the number of batches 0")
	})
	t.Run("batch serialization fails", func(t *testing.T) {
		prevParentBlockHash := common.HexToHash("0x1234")
		block := types.NewBlock(
			&types.Header{
				ParentHash: prevParentBlockHash,
			},
			&types.Body{},
			nil,
			trie.NewStackTrie(nil),
		)
		melState := &meltypes.State{
			ParentChainBlockHash: prevParentBlockHash,
		}
		lookupBatches := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Block,
			receiptFetcher ReceiptFetcher,
			eventUnpacker eventUnpacker,
		) ([]*arbnode.SequencerInboxBatch, []*types.Transaction, []uint, error) {
			batches := []*arbnode.SequencerInboxBatch{
				{},
			}
			txs := []*types.Transaction{{}}
			txIndices := []uint{0}
			return batches, txs, txIndices, nil
		}
		lookupDelayedMsgs := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlockNum *big.Int,
			parentChainBlockTxs []*types.Transaction,
			receiptFetcher ReceiptFetcher,
		) ([]*arbnode.DelayedInboxMessage, error) {
			delayedMsgs := []*arbnode.DelayedInboxMessage{
				{
					Message: &arbostypes.L1IncomingMessage{
						L2msg: []byte("foobar"),
						Header: &arbostypes.L1IncomingMessageHeader{
							Kind: arbostypes.L1MessageType_BatchPostingReport,
						},
					},
				},
			}
			return delayedMsgs, nil
		}
		serializer := func(ctx context.Context,
			batch *arbnode.SequencerInboxBatch,
			tx *types.Transaction,
			txIndex uint,
			receiptFetcher ReceiptFetcher,
		) ([]byte, error) {
			return nil, errors.New("serialization error")
		}
		_, _, _, err := extractMessagesImpl(
			ctx,
			melState,
			block,
			nil,
			nil,
			nil,
			nil,
			lookupBatches,
			lookupDelayedMsgs,
			serializer,
			nil,
			nil,
			nil,
		)
		require.ErrorContains(t, err, "serialization error")
	})
	t.Run("parsing batch posting report fails", func(t *testing.T) {
		prevParentBlockHash := common.HexToHash("0x1234")
		block := types.NewBlock(
			&types.Header{
				ParentHash: prevParentBlockHash,
			},
			&types.Body{},
			nil,
			trie.NewStackTrie(nil),
		)
		melState := &meltypes.State{
			ParentChainBlockHash: prevParentBlockHash,
		}
		lookupBatches := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Block,
			receiptFetcher ReceiptFetcher,
			eventUnpacker eventUnpacker,
		) ([]*arbnode.SequencerInboxBatch, []*types.Transaction, []uint, error) {
			batches := []*arbnode.SequencerInboxBatch{
				{},
			}
			txs := []*types.Transaction{{}}
			txIndices := []uint{0}
			return batches, txs, txIndices, nil
		}
		lookupDelayedMsgs := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlockNum *big.Int,
			parentChainBlockTxs []*types.Transaction,
			receiptFetcher ReceiptFetcher,
		) ([]*arbnode.DelayedInboxMessage, error) {
			delayedMsgs := []*arbnode.DelayedInboxMessage{
				{
					Message: &arbostypes.L1IncomingMessage{
						L2msg: []byte("foobar"),
						Header: &arbostypes.L1IncomingMessageHeader{
							Kind: arbostypes.L1MessageType_BatchPostingReport,
						},
					},
				},
			}
			return delayedMsgs, nil
		}
		serializer := func(ctx context.Context,
			batch *arbnode.SequencerInboxBatch,
			tx *types.Transaction,
			txIndex uint,
			receiptFetcher ReceiptFetcher,
		) ([]byte, error) {
			return nil, nil
		}
		parseReport := func(
			rd io.Reader,
		) (*big.Int, common.Address, common.Hash, uint64, *big.Int, uint64, error) {
			return nil, common.Address{}, common.Hash{}, 0, nil, 0, errors.New("batch posting report parsing error")
		}
		_, _, _, err := extractMessagesImpl(
			ctx,
			melState,
			block,
			nil,
			nil,
			nil,
			nil,
			lookupBatches,
			lookupDelayedMsgs,
			serializer,
			nil,
			nil,
			parseReport,
		)
		require.ErrorContains(t, err, "batch posting report parsing error")
	})
	t.Run("mismatched batch posting report batch hash and actual batch hash", func(t *testing.T) {
		prevParentBlockHash := common.HexToHash("0x1234")
		block := types.NewBlock(
			&types.Header{
				ParentHash: prevParentBlockHash,
			},
			&types.Body{},
			nil,
			trie.NewStackTrie(nil),
		)
		melState := &meltypes.State{
			ParentChainBlockHash: prevParentBlockHash,
		}
		lookupBatches := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Block,
			receiptFetcher ReceiptFetcher,
			eventUnpacker eventUnpacker,
		) ([]*arbnode.SequencerInboxBatch, []*types.Transaction, []uint, error) {
			batches := []*arbnode.SequencerInboxBatch{
				{},
			}
			txs := []*types.Transaction{{}}
			txIndices := []uint{0}
			return batches, txs, txIndices, nil
		}
		lookupDelayedMsgs := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlockNum *big.Int,
			parentChainBlockTxs []*types.Transaction,
			receiptFetcher ReceiptFetcher,
		) ([]*arbnode.DelayedInboxMessage, error) {
			delayedMsgs := []*arbnode.DelayedInboxMessage{
				{
					Message: &arbostypes.L1IncomingMessage{
						L2msg: []byte("foobar"),
						Header: &arbostypes.L1IncomingMessageHeader{
							Kind: arbostypes.L1MessageType_BatchPostingReport,
						},
					},
				},
			}
			return delayedMsgs, nil
		}
		serializer := func(ctx context.Context,
			batch *arbnode.SequencerInboxBatch,
			tx *types.Transaction,
			txIndex uint,
			receiptFetcher ReceiptFetcher,
		) ([]byte, error) {
			return []byte("foobar"), nil
		}
		parseReport := func(
			rd io.Reader,
		) (*big.Int, common.Address, common.Hash, uint64, *big.Int, uint64, error) {
			return nil, common.Address{}, common.Hash{}, 0, nil, 0, nil
		}
		_, _, _, err := extractMessagesImpl(
			ctx,
			melState,
			block,
			nil,
			nil,
			nil,
			nil,
			lookupBatches,
			lookupDelayedMsgs,
			serializer,
			nil,
			nil,
			parseReport,
		)
		require.ErrorContains(t, err, "batch data hash incorrect")
	})
	t.Run("parse sequencer message fails", func(t *testing.T) {
		prevParentBlockHash := common.HexToHash("0x1234")
		block := types.NewBlock(
			&types.Header{
				ParentHash: prevParentBlockHash,
			},
			&types.Body{},
			nil,
			trie.NewStackTrie(nil),
		)
		melState := &meltypes.State{
			ParentChainBlockHash: prevParentBlockHash,
		}
		lookupBatches := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Block,
			receiptFetcher ReceiptFetcher,
			eventUnpacker eventUnpacker,
		) ([]*arbnode.SequencerInboxBatch, []*types.Transaction, []uint, error) {
			batches := []*arbnode.SequencerInboxBatch{
				{},
			}
			txs := []*types.Transaction{{}}
			txIndices := []uint{0}
			return batches, txs, txIndices, nil
		}
		lookupDelayedMsgs := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlockNum *big.Int,
			parentChainBlockTxs []*types.Transaction,
			receiptFetcher ReceiptFetcher,
		) ([]*arbnode.DelayedInboxMessage, error) {
			delayedMsgs := []*arbnode.DelayedInboxMessage{
				{
					Message: &arbostypes.L1IncomingMessage{
						L2msg: []byte("foobar"),
						Header: &arbostypes.L1IncomingMessageHeader{
							Kind: arbostypes.L1MessageType_BatchPostingReport,
						},
					},
				},
			}
			return delayedMsgs, nil
		}
		serializer := func(ctx context.Context,
			batch *arbnode.SequencerInboxBatch,
			tx *types.Transaction,
			txIndex uint,
			receiptFetcher ReceiptFetcher,
		) ([]byte, error) {
			return []byte("foobar"), nil
		}
		parseReport := func(
			rd io.Reader,
		) (*big.Int, common.Address, common.Hash, uint64, *big.Int, uint64, error) {
			return nil, common.Address{}, crypto.Keccak256Hash([]byte("foobar")), 0, nil, 0, nil
		}
		parseSequencerMsg := func(
			ctx context.Context,
			batchNum uint64,
			batchBlockHash common.Hash,
			data []byte,
			dapReaders []daprovider.Reader,
			keysetValidationMode daprovider.KeysetValidationMode,
		) (*arbstate.SequencerMessage, error) {
			return nil, errors.New("failed to parse sequencer message")
		}
		_, _, _, err := extractMessagesImpl(
			ctx,
			melState,
			block,
			nil,
			nil,
			nil,
			nil,
			lookupBatches,
			lookupDelayedMsgs,
			serializer,
			nil,
			parseSequencerMsg,
			parseReport,
		)
		require.ErrorContains(t, err, "failed to parse sequencer message")
	})
	t.Run("extracting batch messages fails", func(t *testing.T) {
		prevParentBlockHash := common.HexToHash("0x1234")
		block := types.NewBlock(
			&types.Header{
				ParentHash: prevParentBlockHash,
			},
			&types.Body{},
			nil,
			trie.NewStackTrie(nil),
		)
		melState := &meltypes.State{
			ParentChainBlockHash: prevParentBlockHash,
		}
		lookupBatches := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Block,
			receiptFetcher ReceiptFetcher,
			eventUnpacker eventUnpacker,
		) ([]*arbnode.SequencerInboxBatch, []*types.Transaction, []uint, error) {
			batches := []*arbnode.SequencerInboxBatch{
				{},
			}
			txs := []*types.Transaction{{}}
			txIndices := []uint{0}
			return batches, txs, txIndices, nil
		}
		lookupDelayedMsgs := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlockNum *big.Int,
			parentChainBlockTxs []*types.Transaction,
			receiptFetcher ReceiptFetcher,
		) ([]*arbnode.DelayedInboxMessage, error) {
			delayedMsgs := []*arbnode.DelayedInboxMessage{
				{
					Message: &arbostypes.L1IncomingMessage{
						L2msg: []byte("foobar"),
						Header: &arbostypes.L1IncomingMessageHeader{
							Kind: arbostypes.L1MessageType_BatchPostingReport,
						},
					},
				},
			}
			return delayedMsgs, nil
		}
		serializer := func(ctx context.Context,
			batch *arbnode.SequencerInboxBatch,
			tx *types.Transaction,
			txIndex uint,
			receiptFetcher ReceiptFetcher,
		) ([]byte, error) {
			return []byte("foobar"), nil
		}
		parseReport := func(
			rd io.Reader,
		) (*big.Int, common.Address, common.Hash, uint64, *big.Int, uint64, error) {
			return nil, common.Address{}, crypto.Keccak256Hash([]byte("foobar")), 0, nil, 0, nil
		}
		parseSequencerMsg := func(
			ctx context.Context,
			batchNum uint64,
			batchBlockHash common.Hash,
			data []byte,
			dapReaders []daprovider.Reader,
			keysetValidationMode daprovider.KeysetValidationMode,
		) (*arbstate.SequencerMessage, error) {
			return nil, nil
		}
		extractBatchMessages := func(
			ctx context.Context,
			melState *meltypes.State,
			seqMsg *arbstate.SequencerMessage,
			delayedMsgDB DelayedMessageDatabase,
		) ([]*arbostypes.MessageWithMetadata, error) {
			return nil, errors.New("failed to extract batch messages")
		}
		_, _, _, err := extractMessagesImpl(
			ctx,
			melState,
			block,
			nil,
			nil,
			nil,
			nil,
			lookupBatches,
			lookupDelayedMsgs,
			serializer,
			extractBatchMessages,
			parseSequencerMsg,
			parseReport,
		)
		require.ErrorContains(t, err, "failed to extract batch messages")
	})
	t.Run("OK", func(t *testing.T) {
		prevParentBlockHash := common.HexToHash("0x1234")
		block := types.NewBlock(
			&types.Header{
				ParentHash: prevParentBlockHash,
			},
			&types.Body{},
			nil,
			trie.NewStackTrie(nil),
		)
		melState := &meltypes.State{
			ParentChainBlockHash: prevParentBlockHash,
		}
		lookupBatches := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Block,
			receiptFetcher ReceiptFetcher,
			eventUnpacker eventUnpacker,
		) ([]*arbnode.SequencerInboxBatch, []*types.Transaction, []uint, error) {
			batches := []*arbnode.SequencerInboxBatch{
				{},
			}
			txs := []*types.Transaction{{}}
			txIndices := []uint{0}
			return batches, txs, txIndices, nil
		}
		lookupDelayedMsgs := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlockNum *big.Int,
			parentChainBlockTxs []*types.Transaction,
			receiptFetcher ReceiptFetcher,
		) ([]*arbnode.DelayedInboxMessage, error) {
			delayedMsgs := []*arbnode.DelayedInboxMessage{
				{
					Message: &arbostypes.L1IncomingMessage{
						L2msg: []byte("foobar"),
						Header: &arbostypes.L1IncomingMessageHeader{
							Kind: arbostypes.L1MessageType_BatchPostingReport,
						},
					},
				},
			}
			return delayedMsgs, nil
		}
		serializer := func(ctx context.Context,
			batch *arbnode.SequencerInboxBatch,
			tx *types.Transaction,
			txIndex uint,
			receiptFetcher ReceiptFetcher,
		) ([]byte, error) {
			return []byte("foobar"), nil
		}
		parseReport := func(
			rd io.Reader,
		) (*big.Int, common.Address, common.Hash, uint64, *big.Int, uint64, error) {
			return nil, common.Address{}, crypto.Keccak256Hash([]byte("foobar")), 0, nil, 0, nil
		}
		parseSequencerMsg := func(
			ctx context.Context,
			batchNum uint64,
			batchBlockHash common.Hash,
			data []byte,
			dapReaders []daprovider.Reader,
			keysetValidationMode daprovider.KeysetValidationMode,
		) (*arbstate.SequencerMessage, error) {
			return nil, nil
		}
		extractBatchMessages := func(
			ctx context.Context,
			melState *meltypes.State,
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
		postState, messages, delayedMessages, err := extractMessagesImpl(
			ctx,
			melState,
			block,
			nil,
			nil,
			nil,
			nil,
			lookupBatches,
			lookupDelayedMsgs,
			serializer,
			extractBatchMessages,
			parseSequencerMsg,
			parseReport,
		)
		require.NoError(t, err)
		require.Equal(t, uint64(2), postState.MsgCount)
		require.Equal(t, uint64(1), postState.DelayedMessagedSeen)
		require.Len(t, messages, 2)
		require.Len(t, delayedMessages, 1)
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
