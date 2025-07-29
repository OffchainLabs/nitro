package melextraction

import (
	"context"
	"math/big"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

func Test_parseDelayedMessagesFromBlock(t *testing.T) {
	ctx := context.Background()
	delayedMsgPostingAddr := common.BytesToAddress([]byte("deadbeef"))
	melState := &mel.State{
		MsgCount:                           1,
		DelayedMessagePostingTargetAddress: delayedMsgPostingAddr,
	}

	header := &types.Header{
		Number: big.NewInt(1),
	}
	txsFetcher := &mockTxsFetcher{
		txs: []*types.Transaction{},
	}
	receiptFetcher := &mockReceiptFetcher{}

	t.Run("no transactions", func(t *testing.T) {
		msgs, err := parseDelayedMessagesFromBlock(
			ctx,
			melState,
			header,
			receiptFetcher,
			txsFetcher,
		)
		require.NoError(t, err)
		require.Empty(t, msgs)
	})
	t.Run("tx with no to field set", func(t *testing.T) {
		txData := &types.DynamicFeeTx{
			To:        nil,
			Nonce:     1,
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(1),
			Gas:       1,
			Value:     big.NewInt(0),
			Data:      nil,
		}
		tx := types.NewTx(txData)
		blockBody := &types.Body{
			Transactions: []*types.Transaction{tx},
		}
		txsFetcher = &mockTxsFetcher{
			txs: []*types.Transaction{tx},
		}
		block := types.NewBlock(
			&types.Header{},
			blockBody,
			nil,
			trie.NewStackTrie(nil),
		)
		msgs, err := parseDelayedMessagesFromBlock(
			ctx,
			melState,
			block.Header(),
			receiptFetcher,
			txsFetcher,
		)
		require.NoError(t, err)
		require.Empty(t, msgs)
	})
	t.Run("no receipt logs", func(t *testing.T) {
		txData := &types.DynamicFeeTx{
			To:        &delayedMsgPostingAddr,
			Nonce:     1,
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(1),
			Gas:       1,
			Value:     big.NewInt(0),
			Data:      nil,
		}
		tx := types.NewTx(txData)
		blockBody := &types.Body{
			Transactions: []*types.Transaction{tx},
		}
		txsFetcher = &mockTxsFetcher{
			txs: []*types.Transaction{tx},
		}
		receipt := &types.Receipt{
			Logs: []*types.Log{},
		}
		receipts := []*types.Receipt{receipt}
		block := types.NewBlock(
			&types.Header{},
			blockBody,
			receipts,
			trie.NewStackTrie(nil),
		)
		receiptFetcher = &mockReceiptFetcher{
			receipts: receipts,
		}
		msgs, err := parseDelayedMessagesFromBlock(
			ctx,
			melState,
			block.Header(),
			receiptFetcher,
			txsFetcher,
		)
		require.NoError(t, err)
		require.Empty(t, msgs)
	})
	t.Run("log emitted by delayed message posting address, but no message data found", func(t *testing.T) {
		event, packedLog := setupParseDelayedMessagesTest(t)
		txData := &types.DynamicFeeTx{
			To:        &delayedMsgPostingAddr,
			Nonce:     1,
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(1),
			Gas:       1,
			Value:     big.NewInt(0),
			Data:      nil,
		}
		tx := types.NewTx(txData)
		blockBody := &types.Body{
			Transactions: []*types.Transaction{tx},
		}
		txsFetcher = &mockTxsFetcher{
			txs: []*types.Transaction{tx},
		}
		messageIndexBytes := common.BigToHash(event.MessageIndex)
		receipt := &types.Receipt{
			Logs: []*types.Log{
				{
					Address: delayedMsgPostingAddr,
					Data:    packedLog,
					Topics: []common.Hash{
						iBridgeABI.Events["MessageDelivered"].ID,
						messageIndexBytes,
						event.BeforeInboxAcc,
					},
				},
			},
		}
		receipts := []*types.Receipt{receipt}
		block := types.NewBlock(
			&types.Header{},
			blockBody,
			receipts,
			trie.NewStackTrie(nil),
		)
		receiptFetcher = &mockReceiptFetcher{
			receipts: receipts,
		}
		_, err := parseDelayedMessagesFromBlock(
			ctx,
			melState,
			block.Header(),
			receiptFetcher,
			txsFetcher,
		)
		require.ErrorContains(t, err, "message 1 data not found")
	})
	t.Run("fetching message data from inbox message delivered event, but hash mismatched", func(t *testing.T) {
		delayedMsgEvent, delayedMsgPackedLog := setupParseDelayedMessagesTest(t)
		msgDataEvent := &bridgegen.IDelayedMessageProviderInboxMessageDelivered{
			Data: []byte("foobar"),
		}
		eventABI := iDelayedMessageProviderABI.Events["InboxMessageDelivered"]
		packedMsgDataLog, err := eventABI.Inputs.NonIndexed().Pack(msgDataEvent.Data)
		require.NoError(t, err)

		txData1 := &types.DynamicFeeTx{
			To:        &delayedMsgPostingAddr,
			Nonce:     1,
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(1),
			Gas:       1,
			Value:     big.NewInt(0),
			Data:      nil,
		}
		tx1 := types.NewTx(txData1)
		txData2 := &types.DynamicFeeTx{
			To:        &delayedMsgEvent.Inbox,
			Nonce:     1,
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(1),
			Gas:       1,
			Value:     big.NewInt(0),
			Data:      nil,
		}
		tx2 := types.NewTx(txData2)
		blockBody := &types.Body{
			Transactions: []*types.Transaction{tx1, tx2},
		}
		txsFetcher := &mockTxsFetcher{
			txs: []*types.Transaction{tx1, tx2},
		}
		messageIndexBytes := common.BigToHash(delayedMsgEvent.MessageIndex)
		receipt1 := &types.Receipt{
			Logs: []*types.Log{
				{
					Address: delayedMsgPostingAddr,
					Data:    delayedMsgPackedLog,
					Topics: []common.Hash{
						iBridgeABI.Events["MessageDelivered"].ID,
						messageIndexBytes,
						delayedMsgEvent.BeforeInboxAcc,
					},
				},
			},
		}
		receipt2 := &types.Receipt{
			Logs: []*types.Log{
				{
					Address: delayedMsgEvent.Inbox,
					Data:    packedMsgDataLog,
					Topics: []common.Hash{
						iDelayedMessageProviderABI.Events["InboxMessageDelivered"].ID,
						messageIndexBytes,
					},
				},
			},
		}
		receipts := []*types.Receipt{receipt1, receipt2}
		block := types.NewBlock(
			&types.Header{},
			blockBody,
			receipts,
			trie.NewStackTrie(nil),
		)
		receiptFetcher = &mockReceiptFetcher{
			receipts: receipts,
		}
		_, err = parseDelayedMessagesFromBlock(
			ctx,
			melState,
			block.Header(),
			receiptFetcher,
			txsFetcher,
		)
		require.ErrorContains(t, err, "mismatched hash")
	})
	t.Run("fetching message data from inbox message delivered event OK", func(t *testing.T) {
		msgData := []byte("foobar")
		delayedMsgEvent := &bridgegen.IBridgeMessageDelivered{
			MessageIndex:    big.NewInt(1),
			BeforeInboxAcc:  [32]byte{},
			Inbox:           common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
			Kind:            1,
			Sender:          [20]byte{},
			MessageDataHash: crypto.Keccak256Hash(msgData),
			BaseFeeL1:       big.NewInt(2),
			Timestamp:       0,
		}
		eventABI := iBridgeABI.Events["MessageDelivered"]
		delayedMsgPackedLog, err := eventABI.Inputs.NonIndexed().Pack(
			delayedMsgEvent.Inbox,
			delayedMsgEvent.Kind,
			delayedMsgEvent.Sender,
			delayedMsgEvent.MessageDataHash,
			delayedMsgEvent.BaseFeeL1,
			delayedMsgEvent.Timestamp,
		)
		require.NoError(t, err)
		msgDataEvent := &bridgegen.IDelayedMessageProviderInboxMessageDelivered{
			Data: msgData,
		}
		eventABI = iDelayedMessageProviderABI.Events["InboxMessageDelivered"]
		packedMsgDataLog, err := eventABI.Inputs.NonIndexed().Pack(msgDataEvent.Data)
		require.NoError(t, err)

		txData1 := &types.DynamicFeeTx{
			To:        &delayedMsgPostingAddr,
			Nonce:     1,
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(1),
			Gas:       1,
			Value:     big.NewInt(0),
			Data:      nil,
		}
		tx1 := types.NewTx(txData1)
		txData2 := &types.DynamicFeeTx{
			To:        &delayedMsgEvent.Inbox,
			Nonce:     1,
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(1),
			Gas:       1,
			Value:     big.NewInt(0),
			Data:      nil,
		}
		tx2 := types.NewTx(txData2)
		blockBody := &types.Body{
			Transactions: []*types.Transaction{tx1, tx2},
		}
		txsFetcher := &mockTxsFetcher{
			txs: []*types.Transaction{tx1, tx2},
		}
		messageIndexBytes := common.BigToHash(delayedMsgEvent.MessageIndex)
		receipt1 := &types.Receipt{
			Logs: []*types.Log{
				{
					Address: delayedMsgPostingAddr,
					Data:    delayedMsgPackedLog,
					Topics: []common.Hash{
						iBridgeABI.Events["MessageDelivered"].ID,
						messageIndexBytes,
						delayedMsgEvent.BeforeInboxAcc,
					},
				},
			},
		}
		receipt2 := &types.Receipt{
			Logs: []*types.Log{
				{
					Address: delayedMsgEvent.Inbox,
					Data:    packedMsgDataLog,
					Topics: []common.Hash{
						iDelayedMessageProviderABI.Events["InboxMessageDelivered"].ID,
						messageIndexBytes,
					},
				},
			},
		}
		receipts := []*types.Receipt{receipt1, receipt2}
		block := types.NewBlock(
			&types.Header{},
			blockBody,
			receipts,
			trie.NewStackTrie(nil),
		)
		receiptFetcher = &mockReceiptFetcher{
			receipts: receipts,
		}
		delayedMessages, err := parseDelayedMessagesFromBlock(
			ctx,
			melState,
			block.Header(),
			receiptFetcher,
			txsFetcher,
		)
		require.NoError(t, err)
		require.Equal(t, 1, len(delayedMessages))
		require.Equal(t, delayedMessages[0].Message.L2msg, msgData)
	})
	t.Run("fetching message data from inbox message delivered event from origin tx data too short", func(t *testing.T) {
		msgData := []byte("foobar")
		delayedMsgEvent := &bridgegen.IBridgeMessageDelivered{
			MessageIndex:    big.NewInt(1),
			BeforeInboxAcc:  [32]byte{},
			Inbox:           common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
			Kind:            1,
			Sender:          [20]byte{},
			MessageDataHash: crypto.Keccak256Hash(msgData),
			BaseFeeL1:       big.NewInt(2),
			Timestamp:       0,
		}
		eventABI := iBridgeABI.Events["MessageDelivered"]
		delayedMsgPackedLog, err := eventABI.Inputs.NonIndexed().Pack(
			delayedMsgEvent.Inbox,
			delayedMsgEvent.Kind,
			delayedMsgEvent.Sender,
			delayedMsgEvent.MessageDataHash,
			delayedMsgEvent.BaseFeeL1,
			delayedMsgEvent.Timestamp,
		)
		require.NoError(t, err)
		txData1 := &types.DynamicFeeTx{
			To:        &delayedMsgPostingAddr,
			Nonce:     1,
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(1),
			Gas:       1,
			Value:     big.NewInt(0),
			Data:      nil,
		}
		tx1 := types.NewTx(txData1)
		txData2 := &types.DynamicFeeTx{
			To:        &delayedMsgEvent.Inbox,
			Nonce:     1,
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(1),
			Gas:       1,
			Value:     big.NewInt(0),
			Data:      nil,
		}
		tx2 := types.NewTx(txData2)
		blockBody := &types.Body{
			Transactions: []*types.Transaction{tx1, tx2},
		}
		txsFetcher := &mockTxsFetcher{
			txs: []*types.Transaction{tx1, tx2},
		}
		messageIndexBytes := common.BigToHash(delayedMsgEvent.MessageIndex)
		receipt1 := &types.Receipt{
			Logs: []*types.Log{
				{
					Address: delayedMsgPostingAddr,
					Data:    delayedMsgPackedLog,
					Topics: []common.Hash{
						iBridgeABI.Events["MessageDelivered"].ID,
						messageIndexBytes,
						delayedMsgEvent.BeforeInboxAcc,
					},
				},
			},
		}
		receipt2 := &types.Receipt{
			Logs: []*types.Log{
				{
					Address: delayedMsgEvent.Inbox,
					Topics: []common.Hash{
						iDelayedMessageProviderABI.Events["InboxMessageDeliveredFromOrigin"].ID,
						messageIndexBytes,
					},
				},
			},
		}
		receipts := []*types.Receipt{receipt1, receipt2}
		block := types.NewBlock(
			&types.Header{},
			blockBody,
			receipts,
			trie.NewStackTrie(nil),
		)
		receiptFetcher = &mockReceiptFetcher{
			receipts: receipts,
		}
		_, err = parseDelayedMessagesFromBlock(
			ctx,
			melState,
			block.Header(),
			receiptFetcher,
			txsFetcher,
		)
		require.ErrorContains(t, err, "too short")
	})
	t.Run("fetching message data from inbox message delivered event from origin OK", func(t *testing.T) {
		msgData := []byte("foobar")
		delayedMsgEvent := &bridgegen.IBridgeMessageDelivered{
			MessageIndex:    big.NewInt(1),
			BeforeInboxAcc:  [32]byte{},
			Inbox:           common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
			Kind:            1,
			Sender:          [20]byte{},
			MessageDataHash: crypto.Keccak256Hash(msgData),
			BaseFeeL1:       big.NewInt(2),
			Timestamp:       0,
		}
		eventABI := iBridgeABI.Events["MessageDelivered"]
		delayedMsgPackedLog, err := eventABI.Inputs.NonIndexed().Pack(
			delayedMsgEvent.Inbox,
			delayedMsgEvent.Kind,
			delayedMsgEvent.Sender,
			delayedMsgEvent.MessageDataHash,
			delayedMsgEvent.BaseFeeL1,
			delayedMsgEvent.Timestamp,
		)
		require.NoError(t, err)
		eventABI = iDelayedMessageProviderABI.Events["InboxMessageDeliveredFromOrigin"]
		l2MessageFromOriginCallABI := iInboxABI.Methods["sendL2MessageFromOrigin"]
		originTxData, err := l2MessageFromOriginCallABI.Inputs.Pack(msgData)
		require.NoError(t, err)
		fullTxData := append(l2MessageFromOriginCallABI.ID, originTxData...) //nolint:gocritic

		txData1 := &types.DynamicFeeTx{
			To:        &delayedMsgPostingAddr,
			Nonce:     1,
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(1),
			Gas:       1,
			Value:     big.NewInt(0),
			Data:      nil,
		}
		tx1 := types.NewTx(txData1)
		txData2 := &types.DynamicFeeTx{
			To:        &delayedMsgEvent.Inbox,
			Nonce:     1,
			GasFeeCap: big.NewInt(1),
			GasTipCap: big.NewInt(1),
			Gas:       1,
			Value:     big.NewInt(0),
			Data:      fullTxData,
		}
		tx2 := types.NewTx(txData2)
		blockBody := &types.Body{
			Transactions: []*types.Transaction{tx1, tx2},
		}
		txsFetcher := &mockTxsFetcher{
			txs: []*types.Transaction{tx1, tx2},
		}
		messageIndexBytes := common.BigToHash(delayedMsgEvent.MessageIndex)
		receipt1 := &types.Receipt{
			Logs: []*types.Log{
				{
					Address: delayedMsgPostingAddr,
					Data:    delayedMsgPackedLog,
					Topics: []common.Hash{
						iBridgeABI.Events["MessageDelivered"].ID,
						messageIndexBytes,
						delayedMsgEvent.BeforeInboxAcc,
					},
				},
			},
		}
		receipt2 := &types.Receipt{
			Logs: []*types.Log{
				{
					Address: delayedMsgEvent.Inbox,
					Topics: []common.Hash{
						iDelayedMessageProviderABI.Events["InboxMessageDeliveredFromOrigin"].ID,
						messageIndexBytes,
					},
				},
			},
		}
		receipts := []*types.Receipt{receipt1, receipt2}
		block := types.NewBlock(
			&types.Header{},
			blockBody,
			receipts,
			trie.NewStackTrie(nil),
		)
		receiptFetcher = &mockReceiptFetcher{
			receipts: receipts,
		}
		delayedMessages, err := parseDelayedMessagesFromBlock(
			ctx,
			melState,
			block.Header(),
			receiptFetcher,
			txsFetcher,
		)
		require.NoError(t, err)
		require.Equal(t, 1, len(delayedMessages))
		require.Equal(t, delayedMessages[0].Message.L2msg, msgData)
	})
}

func Test_parseMessageScaffoldsFromLogs(t *testing.T) {
	t.Run("empty logs", func(t *testing.T) {
		delayedMsgs, events, err := delayedMessageScaffoldsFromLogs(nil, nil)
		require.NoError(t, err)
		require.Empty(t, delayedMsgs)
		require.Empty(t, events)
	})
	t.Run("nil logs", func(t *testing.T) {
		delayedMsgs, events, err := delayedMessageScaffoldsFromLogs(nil, []*types.Log{nil, nil})
		require.NoError(t, err)
		require.Empty(t, delayedMsgs)
		require.Empty(t, events)
	})
}

func Test_sortableMessageList(t *testing.T) {
	hash1 := common.BigToHash(big.NewInt(1))
	hash2 := common.BigToHash(big.NewInt(2))
	messages := []*mel.DelayedInboxMessage{
		{
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{
					RequestId: &hash2,
				},
			},
		},
		{
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{
					RequestId: &hash1,
				},
			},
		},
	}
	sort.Sort(sortableMessageList(messages))
	require.Equal(t, hash1, *messages[0].Message.Header.RequestId)
	require.Equal(t, hash2, *messages[1].Message.Header.RequestId)
}

func setupParseDelayedMessagesTest(t *testing.T) (*bridgegen.IBridgeMessageDelivered, []byte) {
	event := &bridgegen.IBridgeMessageDelivered{
		MessageIndex:    big.NewInt(1),
		BeforeInboxAcc:  [32]byte{},
		Inbox:           common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
		Kind:            1,
		Sender:          [20]byte{},
		MessageDataHash: [32]byte{},
		BaseFeeL1:       big.NewInt(2),
		Timestamp:       0,
	}
	eventABI := iBridgeABI.Events["MessageDelivered"]
	packedLog, err := eventABI.Inputs.NonIndexed().Pack(
		event.Inbox,
		event.Kind,
		event.Sender,
		event.MessageDataHash,
		event.BaseFeeL1,
		event.Timestamp,
	)
	require.NoError(t, err)
	return event, packedLog
}
