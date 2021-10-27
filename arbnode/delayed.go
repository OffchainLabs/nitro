//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"bytes"
	"context"
	"math/big"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
	"github.com/offchainlabs/arbstate/utils"
)

type L1Interface interface {
	bind.ContractBackend
	ethereum.ChainReader
}

var messageDeliveredID common.Hash
var inboxMessageDeliveredID common.Hash
var inboxMessageFromOriginID common.Hash
var l2MessageFromOriginCallABI abi.Method

func init() {
	parsedIBridgeABI, err := abi.JSON(strings.NewReader(bridgegen.IBridgeABI))
	if err != nil {
		panic(err)
	}
	messageDeliveredID = parsedIBridgeABI.Events["MessageDelivered"].ID

	parsedIMessageProviderABI, err := abi.JSON(strings.NewReader(bridgegen.IMessageProviderABI))
	if err != nil {
		panic(err)
	}
	inboxMessageDeliveredID = parsedIMessageProviderABI.Events["InboxMessageDelivered"].ID
	inboxMessageFromOriginID = parsedIMessageProviderABI.Events["InboxMessageDeliveredFromOrigin"].ID

	parsedIInboxABI, err := abi.JSON(strings.NewReader(bridgegen.IInboxABI))
	if err != nil {
		panic(err)
	}
	l2MessageFromOriginCallABI = parsedIInboxABI.Methods["sendL2MessageFromOrigin"]
}

type DelayedBridge struct {
	con              *bridgegen.IBridge
	address          common.Address
	fromBlock        int64
	client           L1Interface
	messageProviders map[common.Address]*bridgegen.IMessageProvider
}

func NewDelayedBridge(client L1Interface, addr common.Address, fromBlock int64) (*DelayedBridge, error) {
	con, err := bridgegen.NewIBridge(addr, client)
	if err != nil {
		return nil, err
	}

	return &DelayedBridge{
		con:              con,
		address:          addr,
		fromBlock:        fromBlock,
		client:           client,
		messageProviders: make(map[common.Address]*bridgegen.IMessageProvider),
	}, nil
}

func (b *DelayedBridge) GetMessageCount(ctx context.Context, blockNumber *big.Int) (*big.Int, error) {
	opts := &bind.CallOpts{
		Context:     ctx,
		BlockNumber: blockNumber,
	}
	return b.con.MessageCount(opts)
}

func (b *DelayedBridge) GetAccumulator(ctx context.Context, sequenceNumber *big.Int, blockNumber *big.Int) (common.Hash, error) {
	opts := &bind.CallOpts{
		Context:     ctx,
		BlockNumber: blockNumber,
	}
	return b.con.InboxAccs(opts, sequenceNumber)
}

type DelayedInboxMessage struct {
	BlockHash      common.Hash
	BeforeInboxAcc common.Hash
	Message        *arbos.L1IncomingMessage
}

func (m *DelayedInboxMessage) AfterInboxAcc() common.Hash {
	hash := utils.Keccak256(
		[]byte{m.Message.Header.Kind},
		m.Message.Header.Sender.Bytes(),
		m.Message.Header.BlockNumber.Bytes(),
		m.Message.Header.Timestamp.Bytes(),
		m.Message.Header.RequestId.Bytes(),
		m.Message.Header.GasPriceL1.Bytes(),
		utils.Keccak256(m.Message.L2msg).Bytes(),
	)
	return utils.Keccak256(m.BeforeInboxAcc[:], hash.Bytes())
}

func (b *DelayedBridge) LookupMessagesInRange(ctx context.Context, from, to *big.Int) ([]*DelayedInboxMessage, error) {
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
	return b.logsToDeliveredMessages(ctx, logs)
}

type blockInfo struct {
	blockTime *big.Int
	baseFee   *big.Int
}

func (b *blockInfo) txGasPrice(tx *types.Transaction) *big.Int {
	if b.baseFee == nil {
		return tx.GasPrice()
	}
	fee := new(big.Int).Add(tx.GasTipCap(), b.baseFee)
	cap := tx.GasFeeCap()
	if fee.Cmp(cap) > 0 {
		fee = cap
	}
	return fee
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

func (b *DelayedBridge) logsToDeliveredMessages(ctx context.Context, logs []types.Log) ([]*DelayedInboxMessage, error) {
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
		return nil, err
	}

	messages := make([]*DelayedInboxMessage, 0, len(logs))
	for _, parsedLog := range parsedLogs {
		msgKey := common.BigToHash(parsedLog.MessageIndex)
		data, ok := messageData[msgKey]
		if !ok {
			return nil, errors.New("message not found")
		}
		if utils.Keccak256(data) != parsedLog.MessageDataHash {
			return nil, errors.New("found message data with mismatched hash")
		}

		msg := &DelayedInboxMessage{
			BlockHash:      parsedLog.Raw.BlockHash,
			BeforeInboxAcc: parsedLog.BeforeInboxAcc,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{
					Kind:        parsedLog.Kind,
					Sender:      parsedLog.Sender,
					BlockNumber: common.BigToHash(new(big.Int).SetUint64(parsedLog.Raw.BlockNumber)),
					Timestamp:   common.BigToHash(parsedLog.Timestamp),
					RequestId:   common.BigToHash(parsedLog.MessageIndex),
					GasPriceL1:  common.BigToHash(parsedLog.GasPrice),
				},
				L2msg: data,
			},
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
		con, err = bridgegen.NewIMessageProvider(ethLog.Address, b.client)
		if err != nil {
			return nil, nil, err
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
		tx, err := b.client.TransactionInBlock(ctx, ethLog.BlockHash, ethLog.TxIndex)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
		parsedLog, err := con.ParseInboxMessageDeliveredFromOrigin(ethLog)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
		args := make(map[string]interface{})
		err = l2MessageFromOriginCallABI.Inputs.UnpackIntoMap(args, tx.Data()[4:])
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
		return parsedLog.MessageNum, args["messageData"].([]byte), nil
	} else {
		return nil, nil, errors.New("unexpected log type")
	}
}
