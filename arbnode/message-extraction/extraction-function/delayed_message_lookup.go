package extractionfunction

import (
	"context"
	"errors"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbnode"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

func parseDelayedMessagesFromBlock(
	ctx context.Context,
	melState *meltypes.State,
	block *types.Block,
	eventParser BridgeEventParser,
	receiptFetcher ReceiptFetcher,
	batchDeliveredEventID common.Hash,
) ([]*arbnode.DelayedInboxMessage, error) {
	msgScaffolds := make([]*arbnode.DelayedInboxMessage, 0)
	messageDeliveredEvents := make([]*bridgegen.IBridgeMessageDelivered, 0)
	txes := block.Transactions()
	for _, tx := range txes {
		if tx.To() == nil {
			continue
		}
		if *tx.To() != melState.DelayedMessagePostingTargetAddress {
			continue
		}
		// Fetch the receipts for the transaction to get the logs.
		receipt, err := receiptFetcher.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			return nil, err
		}
		delayedMessageScaffolds, parsedLogs, err := delayedMessageScaffoldsFromLogs(
			receipt.Logs, eventParser,
		)
		if err != nil {
			return nil, err
		}
		msgScaffolds = append(msgScaffolds, delayedMessageScaffolds...)
		messageDeliveredEvents = append(messageDeliveredEvents, parsedLogs...)
	}

	inboxAddrs := make(map[common.Address]struct{})
	for _, event := range messageDeliveredEvents {
		inboxAddrs[event.Inbox] = struct{}{}
	}

	for _, tx := range txes {
		if tx.To() == nil {
			continue
		}
		_, ok := inboxAddrs[*tx.To()]
		if !ok {
			continue
		}
		receipt, err := receiptFetcher.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			return nil, err
		}
		if len(receipt.Logs) == 0 {
			continue
		}
		for _, log := range receipt.Logs {
			// TODO: Check these topics:
			// inboxMessageDeliveredID, inboxMessageFromOriginID}, messageIds
			if !slices.Contains(log.Topics, batchDeliveredEventID) {
				continue
			}

			// messageData[common.BigToHash(msgNum)] = msg
		}
	}

	return nil, nil
}

func delayedMessageScaffoldsFromLogs(
	logs []*types.Log, eventParser BridgeEventParser,
) ([]*arbnode.DelayedInboxMessage, []*bridgegen.IBridgeMessageDelivered, error) {
	if len(logs) == 0 {
		return nil, nil, nil
	}
	parsedLogs := make([]*bridgegen.IBridgeMessageDelivered, 0, len(logs))

	// First, do a pass over the logs to extract message delivered events, which
	// contain an inbox address and a message index.
	for _, ethLog := range logs {
		parsedLog, err := eventParser.ParseMessageDelivered(*ethLog)
		if err != nil {
			return nil, nil, err
		}
		parsedLogs = append(parsedLogs, parsedLog)
	}

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
					BlockNumber: parsedLog.Raw.BlockNumber,
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

func parseDelayedMessage(ctx context.Context, ethLog types.Log) (*big.Int, []byte, error) {
	// con, ok := b.messageProviders[ethLog.Address]
	// if !ok {
	// 	var err error
	// 	con, err = bridgegen.NewIDelayedMessageProvider(ethLog.Address, b.client)
	// 	if err != nil {
	// 		return nil, nil, err
	// 	}
	// 	b.messageProviders[ethLog.Address] = con
	// }
	switch {
	case ethLog.Topics[0] == inboxMessageDeliveredID:
		parsedLog, err := con.ParseInboxMessageDelivered(ethLog)
		if err != nil {
			return nil, nil, err
		}
		return parsedLog.MessageNum, parsedLog.Data, nil
	case ethLog.Topics[0] == inboxMessageFromOriginID:
		parsedLog, err := con.ParseInboxMessageDeliveredFromOrigin(ethLog)
		if err != nil {
			return nil, nil, err
		}
		data, err := arbutil.GetLogEmitterTxData(ctx, b.client, ethLog)
		if err != nil {
			return nil, nil, err
		}
		args := make(map[string]interface{})
		err = l2MessageFromOriginCallABI.Inputs.UnpackIntoMap(args, data[4:])
		if err != nil {
			return nil, nil, err
		}
		dataBytes, ok := args["messageData"].([]byte)
		if !ok {
			return nil, nil, errors.New("messageData not a byte array")
		}
		return parsedLog.MessageNum, dataBytes, nil
	default:
		return nil, nil, errors.New("unexpected log type")
	}
}
