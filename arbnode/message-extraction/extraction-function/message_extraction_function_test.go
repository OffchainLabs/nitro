package extractionfunction

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
	"github.com/ethereum/go-ethereum/trie"

	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/daprovider"
)

func TestExtractMessages(t *testing.T) {
	ctx := context.Background()
	requestId := common.MaxHash
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
		txsFetcher := &mockTxsFetcher{}
		melState := &meltypes.State{
			ParentChainBlockHash: common.HexToHash("0x5678"),
		}
		melState.SetSeenUnreadDelayedMetaDeque(&meltypes.DelayedMetaDeque{})
		_, _, _, err := ExtractMessages(
			ctx,
			melState,
			block.Header(),
			nil,
			nil,
			nil,
			txsFetcher,
		)
		require.ErrorContains(t, err, "parent chain block hash in MEL state does not match")
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
		melState.SetSeenUnreadDelayedMetaDeque(&meltypes.DelayedMetaDeque{})
		lookupBatches := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Header,
			txsFetcher TransactionsFetcher,
			receiptFetcher ReceiptFetcher,
			eventUnpacker eventUnpacker,
		) ([]*meltypes.SequencerInboxBatch, []*types.Transaction, []uint, error) {
			return nil, nil, nil, errors.New("failed to lookup batches")
		}
		txsFetcher := &mockTxsFetcher{}
		_, _, _, err := extractMessagesImpl(
			ctx,
			melState,
			block.Header(),
			nil,
			nil,
			txsFetcher,
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
		melState.SetSeenUnreadDelayedMetaDeque(&meltypes.DelayedMetaDeque{})
		lookupBatches := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Header,
			txsFetcher TransactionsFetcher,
			receiptFetcher ReceiptFetcher,
			eventUnpacker eventUnpacker,
		) ([]*meltypes.SequencerInboxBatch, []*types.Transaction, []uint, error) {
			return nil, nil, nil, nil
		}
		lookupDelayedMsgs := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Header,
			receiptFetcher ReceiptFetcher,
			txsFetcher TransactionsFetcher,
		) ([]*meltypes.DelayedInboxMessage, error) {
			return nil, errors.New("failed to lookup delayed messages")
		}

		txsFetcher := &mockTxsFetcher{}
		_, _, _, err := extractMessagesImpl(
			ctx,
			melState,
			block.Header(),
			nil,
			nil,
			txsFetcher,
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
		melState.SetSeenUnreadDelayedMetaDeque(&meltypes.DelayedMetaDeque{})
		lookupBatches := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Header,
			txFetcher TransactionsFetcher,
			receiptFetcher ReceiptFetcher,
			eventUnpacker eventUnpacker,
		) ([]*meltypes.SequencerInboxBatch, []*types.Transaction, []uint, error) {
			return nil, nil, nil, nil
		}
		lookupDelayedMsgs := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Header,
			receiptFetcher ReceiptFetcher,
			txsFetcher TransactionsFetcher,
		) ([]*meltypes.DelayedInboxMessage, error) {
			delayedMsgs := []*meltypes.DelayedInboxMessage{
				{
					Message: &arbostypes.L1IncomingMessage{
						L2msg: []byte("foobar"),
						Header: &arbostypes.L1IncomingMessageHeader{
							Kind:      arbostypes.L1MessageType_BatchPostingReport,
							RequestId: &requestId,
							L1BaseFee: common.Big0,
						},
					},
				},
			}
			return delayedMsgs, nil
		}

		txsFetcher := &mockTxsFetcher{}
		_, _, _, err := extractMessagesImpl(
			ctx,
			melState,
			block.Header(),
			nil,
			nil,
			txsFetcher,
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
		melState.SetSeenUnreadDelayedMetaDeque(&meltypes.DelayedMetaDeque{})
		lookupBatches := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Header,
			txsFetcher TransactionsFetcher,
			receiptFetcher ReceiptFetcher,
			eventUnpacker eventUnpacker,
		) ([]*meltypes.SequencerInboxBatch, []*types.Transaction, []uint, error) {
			batches := []*meltypes.SequencerInboxBatch{
				{},
			}
			txs := []*types.Transaction{{}}
			txIndices := []uint{0}
			return batches, txs, txIndices, nil
		}
		lookupDelayedMsgs := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Header,
			receiptFetcher ReceiptFetcher,
			txsFetcher TransactionsFetcher,
		) ([]*meltypes.DelayedInboxMessage, error) {
			delayedMsgs := []*meltypes.DelayedInboxMessage{
				{
					Message: &arbostypes.L1IncomingMessage{
						L2msg: []byte("foobar"),
						Header: &arbostypes.L1IncomingMessageHeader{
							Kind:      arbostypes.L1MessageType_BatchPostingReport,
							RequestId: &requestId,
							L1BaseFee: common.Big0,
						},
					},
				},
			}
			return delayedMsgs, nil
		}
		serializer := func(ctx context.Context,
			batch *meltypes.SequencerInboxBatch,
			tx *types.Transaction,
			txIndex uint,
			receiptFetcher ReceiptFetcher,
		) ([]byte, error) {
			return nil, errors.New("serialization error")
		}
		_, _, _, err := extractMessagesImpl(
			ctx,
			melState,
			block.Header(),
			nil,
			nil,
			&mockTxsFetcher{},
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
		melState.SetSeenUnreadDelayedMetaDeque(&meltypes.DelayedMetaDeque{})
		lookupBatches := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Header,
			txsFetcher TransactionsFetcher,
			receiptFetcher ReceiptFetcher,
			eventUnpacker eventUnpacker,
		) ([]*meltypes.SequencerInboxBatch, []*types.Transaction, []uint, error) {
			batches := []*meltypes.SequencerInboxBatch{
				{},
			}
			txs := []*types.Transaction{{}}
			txIndices := []uint{0}
			return batches, txs, txIndices, nil
		}
		lookupDelayedMsgs := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Header,
			receiptFetcher ReceiptFetcher,
			txsFetcher TransactionsFetcher,
		) ([]*meltypes.DelayedInboxMessage, error) {
			delayedMsgs := []*meltypes.DelayedInboxMessage{
				{
					Message: &arbostypes.L1IncomingMessage{
						L2msg: []byte("foobar"),
						Header: &arbostypes.L1IncomingMessageHeader{
							Kind:      arbostypes.L1MessageType_BatchPostingReport,
							RequestId: &requestId,
							L1BaseFee: common.Big0,
						},
					},
				},
			}
			return delayedMsgs, nil
		}
		serializer := func(ctx context.Context,
			batch *meltypes.SequencerInboxBatch,
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
			block.Header(),
			nil,
			nil,
			&mockTxsFetcher{},
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
		melState.SetSeenUnreadDelayedMetaDeque(&meltypes.DelayedMetaDeque{})
		lookupBatches := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Header,
			txFetcher TransactionsFetcher,
			receiptFetcher ReceiptFetcher,
			eventUnpacker eventUnpacker,
		) ([]*meltypes.SequencerInboxBatch, []*types.Transaction, []uint, error) {
			batches := []*meltypes.SequencerInboxBatch{
				{},
			}
			txs := []*types.Transaction{{}}
			txIndices := []uint{0}
			return batches, txs, txIndices, nil
		}
		lookupDelayedMsgs := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Header,
			receiptFetcher ReceiptFetcher,
			txsFetcher TransactionsFetcher,
		) ([]*meltypes.DelayedInboxMessage, error) {
			delayedMsgs := []*meltypes.DelayedInboxMessage{
				{
					Message: &arbostypes.L1IncomingMessage{
						L2msg: []byte("foobar"),
						Header: &arbostypes.L1IncomingMessageHeader{
							Kind:      arbostypes.L1MessageType_BatchPostingReport,
							RequestId: &requestId,
							L1BaseFee: common.Big0,
						},
					},
				},
			}
			return delayedMsgs, nil
		}
		serializer := func(ctx context.Context,
			batch *meltypes.SequencerInboxBatch,
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
			block.Header(),
			nil,
			nil,
			&mockTxsFetcher{},
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
		melState.SetSeenUnreadDelayedMetaDeque(&meltypes.DelayedMetaDeque{})
		lookupBatches := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Header,
			txsFetcher TransactionsFetcher,
			receiptFetcher ReceiptFetcher,
			eventUnpacker eventUnpacker,
		) ([]*meltypes.SequencerInboxBatch, []*types.Transaction, []uint, error) {
			batches := []*meltypes.SequencerInboxBatch{
				{},
			}
			txs := []*types.Transaction{{}}
			txIndices := []uint{0}
			return batches, txs, txIndices, nil
		}
		lookupDelayedMsgs := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Header,
			receiptFetcher ReceiptFetcher,
			txsFetcher TransactionsFetcher,
		) ([]*meltypes.DelayedInboxMessage, error) {
			delayedMsgs := []*meltypes.DelayedInboxMessage{
				{
					Message: &arbostypes.L1IncomingMessage{
						L2msg: []byte("foobar"),
						Header: &arbostypes.L1IncomingMessageHeader{
							Kind:      arbostypes.L1MessageType_BatchPostingReport,
							RequestId: &requestId,
							L1BaseFee: common.Big0,
						},
					},
				},
			}
			return delayedMsgs, nil
		}
		serializer := func(ctx context.Context,
			batch *meltypes.SequencerInboxBatch,
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
			block.Header(),
			nil,
			nil,
			&mockTxsFetcher{},
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
		melState.SetSeenUnreadDelayedMetaDeque(&meltypes.DelayedMetaDeque{})
		lookupBatches := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Header,
			txsFetcher TransactionsFetcher,
			receiptFetcher ReceiptFetcher,
			eventUnpacker eventUnpacker,
		) ([]*meltypes.SequencerInboxBatch, []*types.Transaction, []uint, error) {
			batches := []*meltypes.SequencerInboxBatch{
				{},
			}
			txs := []*types.Transaction{{}}
			txIndices := []uint{0}
			return batches, txs, txIndices, nil
		}
		lookupDelayedMsgs := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Header,
			receiptFetcher ReceiptFetcher,
			txsFetcher TransactionsFetcher,
		) ([]*meltypes.DelayedInboxMessage, error) {
			delayedMsgs := []*meltypes.DelayedInboxMessage{
				{
					Message: &arbostypes.L1IncomingMessage{
						L2msg: []byte("foobar"),
						Header: &arbostypes.L1IncomingMessageHeader{
							Kind:      arbostypes.L1MessageType_BatchPostingReport,
							RequestId: &requestId,
							L1BaseFee: common.Big0,
						},
					},
				},
			}
			return delayedMsgs, nil
		}
		serializer := func(ctx context.Context,
			batch *meltypes.SequencerInboxBatch,
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
			block.Header(),
			nil,
			nil,
			&mockTxsFetcher{},
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
		melState.SetSeenUnreadDelayedMetaDeque(&meltypes.DelayedMetaDeque{})
		lookupBatches := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Header,
			txsFetcher TransactionsFetcher,
			receiptFetcher ReceiptFetcher,
			eventUnpacker eventUnpacker,
		) ([]*meltypes.SequencerInboxBatch, []*types.Transaction, []uint, error) {
			batches := []*meltypes.SequencerInboxBatch{
				{},
			}
			txs := []*types.Transaction{{}}
			txIndices := []uint{0}
			return batches, txs, txIndices, nil
		}
		lookupDelayedMsgs := func(
			ctx context.Context,
			melState *meltypes.State,
			parentChainBlock *types.Header,
			receiptFetcher ReceiptFetcher,
			txsFetcher TransactionsFetcher,
		) ([]*meltypes.DelayedInboxMessage, error) {
			delayedMsgs := []*meltypes.DelayedInboxMessage{
				{
					Message: &arbostypes.L1IncomingMessage{
						L2msg: []byte("foobar"),
						Header: &arbostypes.L1IncomingMessageHeader{
							Kind:      arbostypes.L1MessageType_BatchPostingReport,
							RequestId: &requestId,
							L1BaseFee: common.Big0,
						},
					},
				},
			}
			return delayedMsgs, nil
		}
		serializer := func(ctx context.Context,
			batch *meltypes.SequencerInboxBatch,
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
			block.Header(),
			nil,
			nil,
			&mockTxsFetcher{},
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
