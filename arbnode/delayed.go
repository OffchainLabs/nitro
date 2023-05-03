// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"bytes"
	"context"
	"math/big"
	"sort"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

var messageDeliveredID common.Hash
var inboxMessageDeliveredID common.Hash
var inboxMessageFromOriginID common.Hash
var l2MessageFromOriginCallABI abi.Method

func init() {
	parsedIBridgeABI, err := bridgegen.IBridgeMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	messageDeliveredID = parsedIBridgeABI.Events["MessageDelivered"].ID

	parsedIMessageProviderABI, err := bridgegen.IDelayedMessageProviderMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	inboxMessageDeliveredID = parsedIMessageProviderABI.Events["InboxMessageDelivered"].ID
	inboxMessageFromOriginID = parsedIMessageProviderABI.Events["InboxMessageDeliveredFromOrigin"].ID

	parsedIInboxABI, err := bridgegen.IInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	l2MessageFromOriginCallABI = parsedIInboxABI.Methods["sendL2MessageFromOrigin"]
}

type DelayedBridge struct {
	con              *bridgegen.IBridge
	address          common.Address
	fromBlock        uint64
	client           arbutil.L1Interface
	messageProviders map[common.Address]*bridgegen.IDelayedMessageProvider
}

func NewDelayedBridge(client arbutil.L1Interface, addr common.Address, fromBlock uint64) (*DelayedBridge, error) {
	con, err := bridgegen.NewIBridge(addr, client)
	if err != nil {
		return nil, err
	}

	return &DelayedBridge{
		con:              con,
		address:          addr,
		fromBlock:        fromBlock,
		client:           client,
		messageProviders: make(map[common.Address]*bridgegen.IDelayedMessageProvider),
	}, nil
}

func (b *DelayedBridge) FirstBlock() *big.Int {
	return new(big.Int).SetUint64(b.fromBlock)
}

func (b *DelayedBridge) GetMessageCount(ctx context.Context, blockNumber *big.Int) (uint64, error) {
	if (blockNumber != nil) && blockNumber.Cmp(new(big.Int).SetUint64(b.fromBlock)) < 0 {
		return 0, nil
	}
	opts := &bind.CallOpts{
		Context:     ctx,
		BlockNumber: blockNumber,
	}
	bigRes, err := b.con.DelayedMessageCount(opts)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	if !bigRes.IsUint64() {
		return 0, errors.New("DelayedBridge MessageCount doesn't make sense!")
	}
	return bigRes.Uint64(), nil
}

func (b *DelayedBridge) GetAccumulator(ctx context.Context, sequenceNumber uint64, blockNumber *big.Int) (common.Hash, error) {
	opts := &bind.CallOpts{
		Context:     ctx,
		BlockNumber: blockNumber,
	}
	return b.con.DelayedInboxAccs(opts, new(big.Int).SetUint64(sequenceNumber))
}

type DelayedInboxMessage struct {
	BlockHash      common.Hash
	BeforeInboxAcc common.Hash
	Message        *arbostypes.L1IncomingMessage
}

func (m *DelayedInboxMessage) AfterInboxAcc() common.Hash {
	hash := crypto.Keccak256(
		[]byte{m.Message.Header.Kind},
		m.Message.Header.Poster.Bytes(),
		arbmath.UintToBytes(m.Message.Header.BlockNumber),
		arbmath.UintToBytes(m.Message.Header.Timestamp),
		m.Message.Header.RequestId.Bytes(),
		math.U256Bytes(m.Message.Header.L1BaseFee),
		crypto.Keccak256(m.Message.L2msg),
	)
	return crypto.Keccak256Hash(m.BeforeInboxAcc[:], hash)
}

func (b *DelayedBridge) LookupMessagesInRange(ctx context.Context, from, to *big.Int, batchFetcher arbostypes.FallibleBatchFetcher) ([]*DelayedInboxMessage, error) {
	query := ethereum.FilterQuery{
		BlockHash: nil,
		FromBlock: from,
		ToBlock:   to,
		Addresses: []common.Address{b.address},
		Topics:    [][]common.Hash{{messageDeliveredID}},
	}
	logs, err := b.client.FilterLogs(ctx, query)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return b.logsToDeliveredMessages(ctx, logs, batchFetcher)
}

type sortableMessageList []*DelayedInboxMessage

func (l sortableMessageList) Len() int {
	return len(l)
}

func (l sortableMessageList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l sortableMessageList) Less(i, j int) bool {
	return bytes.Compare(l[i].Message.Header.RequestId.Bytes(), l[j].Message.Header.RequestId.Bytes()) < 0
}

func (b *DelayedBridge) logsToDeliveredMessages(ctx context.Context, logs []types.Log, batchFetcher arbostypes.FallibleBatchFetcher) ([]*DelayedInboxMessage, error) {
	if len(logs) == 0 {
		return nil, nil
	}
	parsedLogs := make([]*bridgegen.IBridgeMessageDelivered, 0, len(logs))
	messageIds := make([]common.Hash, 0, len(logs))
	inboxAddresses := make(map[common.Address]struct{})
	minBlockNum := uint64(math.MaxUint64)
	maxBlockNum := uint64(0)
	for _, ethLog := range logs {
		if ethLog.BlockNumber < minBlockNum {
			minBlockNum = ethLog.BlockNumber
		}
		if ethLog.BlockNumber > maxBlockNum {
			maxBlockNum = ethLog.BlockNumber
		}
		parsedLog, err := b.con.ParseMessageDelivered(ethLog)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		messageKey := common.BigToHash(parsedLog.MessageIndex)
		parsedLogs = append(parsedLogs, parsedLog)
		inboxAddresses[parsedLog.Inbox] = struct{}{}
		messageIds = append(messageIds, messageKey)
	}

	messageData := make(map[common.Hash][]byte)
	if err := b.fillMessageData(ctx, inboxAddresses, messageIds, messageData, minBlockNum, maxBlockNum); err != nil {
		return nil, errors.WithStack(err)
	}

	messages := make([]*DelayedInboxMessage, 0, len(logs))
	for _, parsedLog := range parsedLogs {
		msgKey := common.BigToHash(parsedLog.MessageIndex)
		data, ok := messageData[msgKey]
		if !ok {
			return nil, errors.New("message not found")
		}
		if crypto.Keccak256Hash(data) != parsedLog.MessageDataHash {
			return nil, errors.New("found message data with mismatched hash")
		}

		requestId := common.BigToHash(parsedLog.MessageIndex)
		msg := &DelayedInboxMessage{
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
				L2msg: data,
			},
		}
		err := msg.Message.FillInBatchGasCost(batchFetcher)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	sort.Sort(sortableMessageList(messages))

	return messages, nil
}

func (b *DelayedBridge) fillMessageData(
	ctx context.Context,
	inboxAddressSet map[common.Address]struct{},
	messageIds []common.Hash,
	messageData map[common.Hash][]byte,
	minBlockNum, maxBlockNum uint64,
) error {
	inboxAddressList := make([]common.Address, 0, len(inboxAddressSet))
	for addr := range inboxAddressSet {
		inboxAddressList = append(inboxAddressList, addr)
	}

	query := ethereum.FilterQuery{
		BlockHash: nil,
		FromBlock: new(big.Int).SetUint64(minBlockNum),
		ToBlock:   new(big.Int).SetUint64(maxBlockNum),
		Addresses: inboxAddressList,
		Topics:    [][]common.Hash{{inboxMessageDeliveredID, inboxMessageFromOriginID}, messageIds},
	}
	logs, err := b.client.FilterLogs(ctx, query)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, ethLog := range logs {
		msgNum, msg, err := b.parseMessage(ctx, ethLog)
		if err != nil {
			return err
		}
		messageData[common.BigToHash(msgNum)] = msg
	}
	return nil
}

func (b *DelayedBridge) parseMessage(ctx context.Context, ethLog types.Log) (*big.Int, []byte, error) {
	con, ok := b.messageProviders[ethLog.Address]
	if !ok {
		var err error
		con, err = bridgegen.NewIDelayedMessageProvider(ethLog.Address, b.client)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
		b.messageProviders[ethLog.Address] = con
	}
	if ethLog.Topics[0] == inboxMessageDeliveredID {
		parsedLog, err := con.ParseInboxMessageDelivered(ethLog)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
		return parsedLog.MessageNum, parsedLog.Data, nil
	} else if ethLog.Topics[0] == inboxMessageFromOriginID {
		parsedLog, err := con.ParseInboxMessageDeliveredFromOrigin(ethLog)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
		data, err := arbutil.GetLogEmitterTxData(ctx, b.client, ethLog)
		if err != nil {
			return nil, nil, err
		}
		args := make(map[string]interface{})
		err = l2MessageFromOriginCallABI.Inputs.UnpackIntoMap(args, data[4:])
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
		return parsedLog.MessageNum, args["messageData"].([]byte), nil
	} else {
		return nil, nil, errors.New("unexpected log type")
	}
}
