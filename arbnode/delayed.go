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

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"

	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
	"github.com/offchainlabs/arbstate/utils"
)

var delayedIBridgeABI abi.ABI
var messageDeliveredID ethcommon.Hash
var inboxMessageDeliveredID ethcommon.Hash
var inboxMessageFromOriginID ethcommon.Hash

func init() {
	parsedIBridgeABI, err := abi.JSON(strings.NewReader(bridgegen.IBridgeABI))
	if err != nil {
		panic(err)
	}
	messageDeliveredID = parsedIBridgeABI.Events["MessageDelivered"].ID
	delayedIBridgeABI = parsedIBridgeABI

	parsedIMessageProviderABI, err := abi.JSON(strings.NewReader(bridgegen.IMessageProviderABI))
	if err != nil {
		panic(err)
	}
	inboxMessageDeliveredID = parsedIMessageProviderABI.Events["InboxMessageDelivered"].ID
	inboxMessageFromOriginID = parsedIMessageProviderABI.Events["InboxMessageDeliveredFromOrigin"].ID
}

type DelayedBridge struct {
	con       *bridgegen.IBridge
	address   ethcommon.Address
	fromBlock int64
	client    *ethclient.Client
}

func NewDelayedBridge(client *ethclient.Client, addr ethcommon.Address, fromBlock int64) (*DelayedBridge, error) {
	con, err := bridgegen.NewIBridge(addr, client)
	if err != nil {
		return nil, err
	}

	return &DelayedBridge{
		con:       con,
		address:   addr,
		fromBlock: fromBlock,
		client:    client,
	}, nil
}

func (b *DelayedBridge) GetAccumulator(ctx context.Context, sequenceNumber *big.Int) (common.Hash, error) {
	return b.con.InboxAccs(&bind.CallOpts{Context: ctx}, sequenceNumber)
}

type DelayedInboxMessage struct {
	BlockHash      common.Hash
	BeforeInboxAcc common.Hash
	Message        *arbos.L1IncomingMessage
}

func (b *DelayedBridge) LookupMessagesInRange(ctx context.Context, from, to *big.Int) ([]*DelayedInboxMessage, error) {
	query := ethereum.FilterQuery{
		BlockHash: nil,
		FromBlock: from,
		ToBlock:   to,
		Addresses: []ethcommon.Address{b.address},
		Topics:    [][]ethcommon.Hash{{messageDeliveredID}},
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
	rawTransactions := make(map[common.Hash]*types.Transaction)
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

		txData, err := b.client.TransactionInBlock(ctx, ethLog.BlockHash, ethLog.TxIndex)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		rawTransactions[messageKey] = txData
		inboxAddresses[parsedLog.Inbox] = struct{}{}
		messageIds = append(messageIds, messageKey)
	}

	messageData := make(map[common.Hash][]byte)
	if err := b.fillMessageData(ctx, inboxAddresses, messageIds, rawTransactions, messageData, minBlockNum, maxBlockNum); err != nil {
		return nil, err
	}

	blockInfos := make(map[ethcommon.Hash]*blockInfo)

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

		info, ok := blockInfos[parsedLog.Raw.BlockHash]
		if !ok {
			header, err := b.client.HeaderByHash(ctx, parsedLog.Raw.BlockHash)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			info = &blockInfo{
				blockTime: new(big.Int).SetUint64(header.Time),
				baseFee:   header.BaseFee,
			}
			blockInfos[parsedLog.Raw.BlockHash] = info
		}

		tx := rawTransactions[msgKey]
		msg := &DelayedInboxMessage{
			BlockHash:      parsedLog.Raw.BlockHash,
			BeforeInboxAcc: parsedLog.BeforeInboxAcc,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{
					Kind:        parsedLog.Kind,
					Sender:      parsedLog.Sender,
					BlockNumber: common.BigToHash(new(big.Int).SetUint64(parsedLog.Raw.BlockNumber)),
					Timestamp:   common.BigToHash(info.blockTime),
					RequestId:   common.BigToHash(parsedLog.MessageIndex),
					GasPriceL1:  common.BigToHash(info.txGasPrice(tx)),
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
	txData map[common.Hash]*types.Transaction,
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
		Topics:    [][]ethcommon.Hash{{inboxMessageDeliveredID, inboxMessageFromOriginID}, messageIds},
	}
	logs, err := b.client.FilterLogs(ctx, query)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, ethLog := range logs {
		msgNum, msg, err := b.parseMessage(txData, ethLog)
		if err != nil {
			return err
		}
		messageData[common.BigToHash(msgNum)] = msg
	}
	return nil
}
