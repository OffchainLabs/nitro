package melextraction

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

func parseDelayedMessagesFromBlock(
	ctx context.Context,
	melState *mel.State,
	parentChainHeader *types.Header,
	receiptFetcher ReceiptFetcher,
	txsFetcher TransactionsFetcher,
) ([]*mel.DelayedInboxMessage, error) {
	msgScaffolds := make([]*mel.DelayedInboxMessage, 0)
	messageDeliveredEvents := make([]*bridgegen.IBridgeMessageDelivered, 0)
	parentChainBlockTxs, err := txsFetcher.TransactionsByHeader(
		ctx,
		parentChainHeader.Hash(),
	)
	if err != nil {
		return nil, err
	}
	for i, tx := range parentChainBlockTxs {
		if tx.To() == nil {
			continue
		}
		// Fetch the receipts for the transaction to get the logs.
		txIndex := uint(i) // #nosec G115
		receipt, err := receiptFetcher.ReceiptForTransactionIndex(ctx, txIndex)
		if err != nil {
			return nil, err
		}
		relevantLogs := make([]*types.Log, 0, len(receipt.Logs))
		// Check all logs in the receipt.
		for _, log := range receipt.Logs {
			// Check if the log was emitted by the delayed message posting address.
			// On Arbitrum One, this is the bridge contract which emits a MessageDelivered event.
			if log.Address == melState.DelayedMessagePostingTargetAddress {
				relevantLogs = append(relevantLogs, log)
			}
		}
		if len(relevantLogs) == 0 {
			continue
		}
		delayedMessageScaffolds, parsedLogs, err := delayedMessageScaffoldsFromLogs(
			parentChainHeader.Number,
			relevantLogs,
		)
		if err != nil {
			return nil, err
		}
		msgScaffolds = append(msgScaffolds, delayedMessageScaffolds...)
		messageDeliveredEvents = append(messageDeliveredEvents, parsedLogs...)
	}
	messageIds := make([]common.Hash, 0, len(messageDeliveredEvents))
	inboxAddressSet := make(map[common.Address]struct{})
	for _, event := range messageDeliveredEvents {
		inboxAddressSet[event.Inbox] = struct{}{}
		messageIds = append(messageIds, common.BigToHash(event.MessageIndex))
	}
	inboxAddressList := make([]common.Address, 0, len(inboxAddressSet))
	for addr := range inboxAddressSet {
		inboxAddressList = append(inboxAddressList, addr)
	}
	messageData := make(map[common.Hash][]byte)
	for i, tx := range parentChainBlockTxs {
		// TODO: remove this temporary work around for handling init message, i.e skipping the check when msgCount==0
		if melState.MsgCount != 0 {
			if tx.To() == nil {
				continue
			}
			_, ok := inboxAddressSet[*tx.To()]
			if !ok {
				continue
			}
		}
		txIndex := uint(i) // #nosec G115
		receipt, err := receiptFetcher.ReceiptForTransactionIndex(ctx, txIndex)
		if err != nil {
			return nil, err
		}
		if len(receipt.Logs) == 0 {
			continue
		}
		topics := [][]common.Hash{
			{inboxMessageDeliveredID, inboxMessageFromOriginID}, // matches either of these IDs.
			messageIds, // matches any of the message IDs.
		}
		filteredInboxMessageLogs := types.FilterLogs(receipt.Logs, nil, nil, inboxAddressList, topics)
		for _, inboxMsgLog := range filteredInboxMessageLogs {
			msgNum, msg, err := parseDelayedMessage(
				inboxMsgLog,
				tx,
			)
			if err != nil {
				return nil, err
			}
			messageData[common.BigToHash(msgNum)] = msg
		}
	}
	for i, parsedLog := range messageDeliveredEvents {
		msgKey := common.BigToHash(parsedLog.MessageIndex)
		data, ok := messageData[msgKey]
		if !ok {
			return nil, fmt.Errorf("message %v data not found", parsedLog.MessageIndex)
		}
		if crypto.Keccak256Hash(data) != parsedLog.MessageDataHash {
			return nil, fmt.Errorf("found message %v data with mismatched hash", parsedLog.MessageIndex)
		}
		// Fill in the message data for the delayed message scaffolds.
		msgScaffolds[i].Message.L2msg = data
	}
	// Finally, we sort the messages by their request id.
	sort.Sort(sortableMessageList(msgScaffolds))
	return msgScaffolds, nil
}

func delayedMessageScaffoldsFromLogs(
	parentChainBlockNum *big.Int, logs []*types.Log,
) ([]*mel.DelayedInboxMessage, []*bridgegen.IBridgeMessageDelivered, error) {
	if len(logs) == 0 {
		return nil, nil, nil
	}
	parsedLogs := make([]*bridgegen.IBridgeMessageDelivered, 0, len(logs))

	// First, do a pass over the logs to extract message delivered events, which
	// contain an inbox address and a message index.
	for _, ethLog := range logs {
		if ethLog == nil || len(ethLog.Topics) == 0 || ethLog.Topics[0] != iBridgeABI.Events["MessageDelivered"].ID {
			continue
		}
		event := new(bridgegen.IBridgeMessageDelivered)
		if err := unpackLogTo(event, iBridgeABI, "MessageDelivered", *ethLog); err != nil {
			return nil, nil, err
		}
		parsedLogs = append(parsedLogs, event)
	}

	// A list of delayed messages that do not have nil L2msg data within, which
	// will be filled in later after another pass over logs.
	delayedMessageScaffolds := make([]*mel.DelayedInboxMessage, 0, len(parsedLogs))

	// Next, we construct the messages themselves from the parsed logs.
	for _, parsedLog := range parsedLogs {
		msgKey := common.BigToHash(parsedLog.MessageIndex)
		_ = msgKey
		requestId := common.BigToHash(parsedLog.MessageIndex)
		msg := &mel.DelayedInboxMessage{
			BlockHash:      parsedLog.Raw.BlockHash,
			BeforeInboxAcc: parsedLog.BeforeInboxAcc,
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{
					Kind:        parsedLog.Kind,
					Poster:      parsedLog.Sender,
					BlockNumber: parentChainBlockNum.Uint64(),
					Timestamp:   parsedLog.Timestamp,
					RequestId:   &requestId,
					L1BaseFee:   parsedLog.BaseFeeL1,
				},
				L2msg: nil, // Fill in later, once we loop over the block's logs to extract message data.
			},
			ParentChainBlockNumber: parsedLog.Raw.BlockNumber,
		}
		delayedMessageScaffolds = append(delayedMessageScaffolds, msg)
	}
	return delayedMessageScaffolds, parsedLogs, nil
}

func parseDelayedMessage(
	ethLog *types.Log,
	tx *types.Transaction,
) (*big.Int, []byte, error) {
	if ethLog == nil {
		return nil, nil, nil
	}
	switch ethLog.Topics[0] {
	case inboxMessageDeliveredID:
		event := new(bridgegen.IDelayedMessageProviderInboxMessageDelivered)
		if err := unpackLogTo(event, iDelayedMessageProviderABI, "InboxMessageDelivered", *ethLog); err != nil {
			return nil, nil, err
		}
		return event.MessageNum, event.Data, nil
	case inboxMessageFromOriginID:
		event := new(bridgegen.IDelayedMessageProviderInboxMessageDeliveredFromOrigin)
		if err := unpackLogTo(event, iDelayedMessageProviderABI, "InboxMessageDeliveredFromOrigin", *ethLog); err != nil {
			return nil, nil, err
		}
		args := make(map[string]any)
		data := tx.Data()
		if len(data) < 4 {
			return nil, nil, fmt.Errorf("tx data %#x too short", data)
		}
		l2MessageFromOriginCallABI := iInboxABI.Methods["sendL2MessageFromOrigin"]
		if err := l2MessageFromOriginCallABI.Inputs.UnpackIntoMap(args, data[4:]); err != nil {
			return nil, nil, err
		}
		dataBytes, ok := args["messageData"].([]byte)
		if !ok {
			return nil, nil, errors.New("messageData not a byte array")
		}
		return event.MessageNum, dataBytes, nil
	default:
		return nil, nil, errors.New("unexpected log type")
	}
}

type sortableMessageList []*mel.DelayedInboxMessage

func (l sortableMessageList) Len() int {
	return len(l)
}

func (l sortableMessageList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l sortableMessageList) Less(i, j int) bool {
	return bytes.Compare(l[i].Message.Header.RequestId.Bytes(), l[j].Message.Header.RequestId.Bytes()) < 0
}

// Unpacks a log into the given struct with an event name string that is
// present in the specified ABI.
func unpackLogTo(out any, contractABI *abi.ABI, event string, log types.Log) error {
	if len(log.Topics) == 0 {
		return errors.New("no event signature")
	}
	if log.Topics[0] != contractABI.Events[event].ID {
		return fmt.Errorf("event signature mismatch: expected %s, got %s", contractABI.Events[event].ID.Hex(), log.Topics[0].Hex())
	}
	if len(log.Data) > 0 {
		if err := contractABI.UnpackIntoInterface(out, event, log.Data); err != nil {
			return err
		}
	}
	var indexed abi.Arguments
	for _, arg := range contractABI.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	return abi.ParseTopics(out, indexed, log.Topics[1:])
}
