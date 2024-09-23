// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
var delayedInboxAccsCallABI abi.Method

func init() {
	parsedIBridgeABI, err := bridgegen.IBridgeMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	messageDeliveredID = parsedIBridgeABI.Events["MessageDelivered"].ID
	delayedInboxAccsCallABI = parsedIBridgeABI.Methods["delayedInboxAccs"]

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
		return 0, err
	}
	if !bigRes.IsUint64() {
		return 0, errors.New("DelayedBridge MessageCount doesn't make sense!")
	}
	return bigRes.Uint64(), nil
}

// Uses blockHash if nonzero, otherwise uses blockNumber
func (b *DelayedBridge) GetAccumulator(ctx context.Context, sequenceNumber uint64, blockNumber *big.Int, blockHash common.Hash) (common.Hash, error) {
	calldata := append([]byte{}, delayedInboxAccsCallABI.ID...)
	inputs, err := delayedInboxAccsCallABI.Inputs.Pack(arbmath.UintToBig(sequenceNumber))
	if err != nil {
		return common.Hash{}, err
	}
	calldata = append(calldata, inputs...)
	msg := ethereum.CallMsg{
		To:   &b.address,
		Data: calldata,
	}
	var result hexutil.Bytes
	if blockHash != (common.Hash{}) {
		result, err = b.client.CallContractAtHash(ctx, msg, blockHash)
	} else {
		result, err = b.client.CallContract(ctx, msg, blockNumber)
	}
	if err != nil {
		return common.Hash{}, err
	}
	values, err := delayedInboxAccsCallABI.Outputs.Unpack(result)
	if err != nil {
		return common.Hash{}, err
	}
	if len(values) != 1 {
		return common.Hash{}, fmt.Errorf("expected 1 return value from %v, got %v", delayedInboxAccsCallABI.Name, len(values))
	}
	hash, ok := values[0].([32]byte)
	if !ok {
		return common.Hash{}, fmt.Errorf("expected [32]uint8 return value from %v, got %T", delayedInboxAccsCallABI.Name, values[0])
	}
	return hash, nil
}

type DelayedInboxMessage struct {
	BlockHash              common.Hash
	BeforeInboxAcc         common.Hash
	Message                *arbostypes.L1IncomingMessage
	ParentChainBlockNumber uint64
}

func (m *DelayedInboxMessage) AfterInboxAcc() common.Hash {
	hash := crypto.Keccak256(
		[]byte{m.Message.Header.Kind},
		m.Message.Header.Poster.Bytes(),
		arbmath.UintToBytes(m.Message.Header.BlockNumber),
		arbmath.UintToBytes(m.Message.Header.Timestamp),
		m.Message.Header.RequestId.Bytes(),
		arbmath.U256Bytes(m.Message.Header.L1BaseFee),
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
		return nil, err
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
			return nil, err
		}
		messageKey := common.BigToHash(parsedLog.MessageIndex)
		parsedLogs = append(parsedLogs, parsedLog)
		inboxAddresses[parsedLog.Inbox] = struct{}{}
		messageIds = append(messageIds, messageKey)
	}

	messageData := make(map[common.Hash][]byte)
	if err := b.fillMessageData(ctx, inboxAddresses, messageIds, messageData, minBlockNum, maxBlockNum); err != nil {
		return nil, err
	}

	messages := make([]*DelayedInboxMessage, 0, len(logs))
	var lastParentChainBlockHash common.Hash
	var lastL1BlockNumber uint64
	for _, parsedLog := range parsedLogs {
		msgKey := common.BigToHash(parsedLog.MessageIndex)
		data, ok := messageData[msgKey]
		if !ok {
			return nil, fmt.Errorf("message %v data not found", parsedLog.MessageIndex)
		}
		if crypto.Keccak256Hash(data) != parsedLog.MessageDataHash {
			return nil, fmt.Errorf("found message %v data with mismatched hash", parsedLog.MessageIndex)
		}

		requestId := common.BigToHash(parsedLog.MessageIndex)
		parentChainBlockHash := parsedLog.Raw.BlockHash
		var l1BlockNumber uint64
		if lastParentChainBlockHash == parentChainBlockHash && lastParentChainBlockHash != (common.Hash{}) {
			l1BlockNumber = lastL1BlockNumber
		} else {
			parentChainHeader, err := b.client.HeaderByHash(ctx, parentChainBlockHash)
			if err != nil {
				return nil, err
			}
			l1BlockNumber = arbutil.ParentHeaderToL1BlockNumber(parentChainHeader)
			lastParentChainBlockHash = parentChainBlockHash
			lastL1BlockNumber = l1BlockNumber
		}
		msg := &DelayedInboxMessage{
			BlockHash:      parsedLog.Raw.BlockHash,
			BeforeInboxAcc: parsedLog.BeforeInboxAcc,
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{
					Kind:        parsedLog.Kind,
					Poster:      parsedLog.Sender,
					BlockNumber: l1BlockNumber,
					Timestamp:   parsedLog.Timestamp,
					RequestId:   &requestId,
					L1BaseFee:   parsedLog.BaseFeeL1,
				},
				L2msg: data,
			},
			ParentChainBlockNumber: parsedLog.Raw.BlockNumber,
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
		return err
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
			return nil, nil, err
		}
		b.messageProviders[ethLog.Address] = con
	}
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
		return parsedLog.MessageNum, args["messageData"].([]byte), nil
	default:
		return nil, nil, errors.New("unexpected log type")
	}
}
