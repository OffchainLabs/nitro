package extractionfunction

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
	"github.com/offchainlabs/nitro/arbnode"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

type DelayedMessageLookupParams struct {
	MessageDeliveredID         common.Hash
	InboxMessageDeliveredID    common.Hash
	InboxMessageFromOriginID   common.Hash
	IDelayedMessageProviderABI *abi.ABI
	IBridgeABI                 *abi.ABI
	IInboxABI                  *abi.ABI
}

func parseDelayedMessagesFromBlock(
	ctx context.Context,
	melState *meltypes.State,
	parentChainBlock *types.Block,
	receiptFetcher ReceiptFetcher,
	params *DelayedMessageLookupParams,
) ([]*arbnode.DelayedInboxMessage, error) {
	msgScaffolds := make([]*arbnode.DelayedInboxMessage, 0)
	messageDeliveredEvents := make([]*bridgegen.IBridgeMessageDelivered, 0)
	txes := parentChainBlock.Transactions()
	for i, tx := range txes {
		if tx.To() == nil {
			continue
		}
		// TODO: We can exit early if the tx.To() is not the inbox address.
		// However, the inbox address is not the event emitter – the bridge is.

		// Fetch the receipts for the transaction to get the logs.
		txIndex := uint(i)
		receipt, err := receiptFetcher.ReceiptForTransactionIndex(ctx, parentChainBlock, txIndex)
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
			parentChainBlock,
			relevantLogs,
			params,
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
	for i, tx := range txes {
		if tx.To() == nil {
			continue
		}
		_, ok := inboxAddressSet[*tx.To()]
		if !ok {
			continue
		}
		txIndex := uint(i)
		receipt, err := receiptFetcher.ReceiptForTransactionIndex(ctx, parentChainBlock, txIndex)
		if err != nil {
			return nil, err
		}
		if len(receipt.Logs) == 0 {
			continue
		}
		topics := [][]common.Hash{
			{params.InboxMessageDeliveredID, params.InboxMessageFromOriginID}, // matches either of these IDs.
			messageIds, // matches any of the message IDs.
		}
		filteredInboxMessageLogs := filterLogs(receipt.Logs, inboxAddressList, topics)
		for _, inboxMsgLog := range filteredInboxMessageLogs {
			msgNum, msg, err := parseDelayedMessage(
				inboxMsgLog,
				tx,
				params,
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
	parentChainBlock *types.Block, logs []*types.Log, params *DelayedMessageLookupParams,
) ([]*arbnode.DelayedInboxMessage, []*bridgegen.IBridgeMessageDelivered, error) {
	if len(logs) == 0 {
		return nil, nil, nil
	}
	parsedLogs := make([]*bridgegen.IBridgeMessageDelivered, 0, len(logs))

	// First, do a pass over the logs to extract message delivered events, which
	// contain an inbox address and a message index.
	for _, ethLog := range logs {
		if ethLog == nil {
			continue
		}
		event := new(bridgegen.IBridgeMessageDelivered)
		if err := unpackLogTo(event, params.IBridgeABI, "MessageDelivered", *ethLog); err != nil {
			return nil, nil, err
		}
		parsedLogs = append(parsedLogs, event)
	}

	// A list of delayed messages that do not have nil L2msg data within, which
	// will be filled in later after another pass over logs.
	delayedMessageScaffolds := make([]*arbnode.DelayedInboxMessage, 0, len(parsedLogs))

	// Next, we construct the messages themselves from the parsed logs.
	for _, parsedLog := range parsedLogs {
		msgKey := common.BigToHash(parsedLog.MessageIndex)
		_ = msgKey
		requestId := common.BigToHash(parsedLog.MessageIndex)
		msg := &arbnode.DelayedInboxMessage{
			BlockHash:      parsedLog.Raw.BlockHash,
			BeforeInboxAcc: parsedLog.BeforeInboxAcc,
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{
					Kind:        parsedLog.Kind,
					Poster:      parsedLog.Sender,
					BlockNumber: parentChainBlock.NumberU64(),
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
	params *DelayedMessageLookupParams,
) (*big.Int, []byte, error) {
	if ethLog == nil {
		return nil, nil, nil
	}
	switch {
	case ethLog.Topics[0] == params.InboxMessageDeliveredID:
		event := new(bridgegen.IDelayedMessageProviderInboxMessageDelivered)
		if err := unpackLogTo(event, params.IDelayedMessageProviderABI, "InboxMessageDelivered", *ethLog); err != nil {
			return nil, nil, err
		}
		return event.MessageNum, event.Data, nil
	case ethLog.Topics[0] == params.InboxMessageFromOriginID:
		event := new(bridgegen.IDelayedMessageProviderInboxMessageDeliveredFromOrigin)
		if err := unpackLogTo(event, params.IDelayedMessageProviderABI, "InboxMessageDeliveredFromOrigin", *ethLog); err != nil {
			return nil, nil, err
		}
		args := make(map[string]interface{})
		data := tx.Data()
		if len(data) < 4 {
			return nil, nil, errors.New("tx data too short") // TODO: Add a hash of the tx that was too short.
		}
		l2MessageFromOriginCallABI := params.IInboxABI.Methods["sendL2MessageFromOrigin"]
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

func filterLogs(logs []*types.Log, addresses []common.Address, topics [][]common.Hash) []*types.Log {
	var filteredLogs []*types.Log
	for _, log := range logs {
		// Filter by address if addresses are specified.
		if len(addresses) > 0 {
			addressMatch := false
			for _, address := range addresses {
				if log.Address == address {
					addressMatch = true
					break
				}
			}
			if !addressMatch {
				continue
			}
		}
		// Filter by topics.
		if len(topics) > 0 {
			topicMatch := true
			// We can only match as many topics as the log has.
			maxTopics := len(log.Topics)
			if maxTopics > len(topics) {
				maxTopics = len(topics)
			}
			for i := 0; i < maxTopics; i++ {
				// Empty topic list (nil or {}) matches anything.
				if len(topics[i]) == 0 {
					continue
				}
				// Check if current topic matches any of the options for this position.
				positionMatch := false
				for _, topic := range topics[i] {
					if log.Topics[i] == topic {
						positionMatch = true
						break
					}
				}
				if !positionMatch {
					topicMatch = false
					break
				}
			}
			if !topicMatch {
				continue
			}
		}

		filteredLogs = append(filteredLogs, log)
	}
	return filteredLogs
}

type sortableMessageList []*arbnode.DelayedInboxMessage

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
