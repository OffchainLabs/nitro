//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

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

	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
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
	parsedIBridgeABI, err := bridgegen.IBridgeMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	messageDeliveredID = parsedIBridgeABI.Events["MessageDelivered"].ID

	parsedIMessageProviderABI, err := bridgegen.IMessageProviderMetaData.GetAbi()
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

func (b *DelayedBridge) FirstBlock() *big.Int {
	return big.NewInt(b.fromBlock)
}

func (b *DelayedBridge) GetMessageCount(ctx context.Context, blockNumber *big.Int) (uint64, error) {
	if (blockNumber != nil) && blockNumber.Cmp(big.NewInt(b.fromBlock)) < 0 {
		return 0, nil
	}
	opts := &bind.CallOpts{
		Context:     ctx,
		BlockNumber: blockNumber,
	}
	bigRes, err := b.con.MessageCount(opts)
	if err != nil {
		return 0, err
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
	return b.con.InboxAccs(opts, new(big.Int).SetUint64(sequenceNumber))
}

// finds the block where messageNumber was added
// search is between first configured block and top of the chain minus 5
// startFromBlock is a semi-educated guess, doesnt have to be correct, can be nil
func (b *DelayedBridge) FindBlockForMessage(ctx context.Context, messageNumber uint64, startFromBlock *big.Int) (*big.Int, error) {
	var blockNr *big.Int
	if startFromBlock == nil {
		blockNr = big.NewInt(b.fromBlock)
	} else {
		blockNr = startFromBlock
	}
	msgCountLion := messageNumber + 1
	messageAtBlock, err := b.GetMessageCount(ctx, blockNr)
	if err != nil {
		return nil, err
	}
	var firstDir int64
	if msgCountLion < messageAtBlock {
		firstDir = 1
	} else {
		firstDir = -1
	}
	direction := big.NewInt(firstDir)
	lastknownHeadr, err := b.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}
	lastBlockNr := lastknownHeadr.Number
	firstBlockNr := big.NewInt(b.fromBlock)
	// exponentially increasing distance, find blocks before and after required message was added
	var lowBlockNr, highBlockNr *big.Int
	for {
		newBlockNr := &big.Int{}
		newBlockNr.Add(blockNr, direction)
		if newBlockNr.Cmp(lastBlockNr) > 0 {
			newBlockNr = lastBlockNr
		}
		if newBlockNr.Cmp(firstBlockNr) < 0 {
			newBlockNr = firstBlockNr
		}
		if newBlockNr.Cmp(blockNr) == 0 {
			return nil, errors.New("Not Found")
		}
		messageAtBlock, err := b.GetMessageCount(ctx, blockNr)
		if err != nil {
			return nil, err
		}
		if (messageAtBlock < msgCountLion) && firstDir < 0 {
			lowBlockNr = newBlockNr
			highBlockNr = blockNr
			break
		}
		if (messageAtBlock >= msgCountLion) && firstDir > 0 {
			lowBlockNr = blockNr
			highBlockNr = newBlockNr
			break
		}
		blockNr = newBlockNr
		direction.Mul(direction, big.NewInt(2))
	}
	//exponentially reducing distance, find the exact block required message was added
	for (&big.Int{}).Sub(highBlockNr, lowBlockNr).Cmp(big.NewInt(1)) > 0 {
		newBlockNr := &big.Int{}
		newBlockNr.Add(highBlockNr, lowBlockNr)
		newBlockNr.Div(newBlockNr, big.NewInt(2))
		messageAtBlock, err = b.GetMessageCount(ctx, blockNr)
		if err != nil {
			return nil, err
		}
		if messageAtBlock < msgCountLion {
			lowBlockNr = newBlockNr
		} else {
			highBlockNr = newBlockNr
		}
	}
	return highBlockNr, nil
}

type DelayedInboxMessage struct {
	BlockHash      common.Hash
	BeforeInboxAcc common.Hash
	Message        *arbos.L1IncomingMessage
}

func (m *DelayedInboxMessage) AfterInboxAcc() common.Hash {
	hash := crypto.Keccak256(
		[]byte{m.Message.Header.Kind},
		m.Message.Header.Sender.Bytes(),
		m.Message.Header.BlockNumber.Bytes(),
		m.Message.Header.Timestamp.Bytes(),
		m.Message.Header.RequestId.Bytes(),
		m.Message.Header.GasPriceL1.Bytes(),
		crypto.Keccak256(m.Message.L2msg),
	)
	return crypto.Keccak256Hash(m.BeforeInboxAcc[:], hash)
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
		if crypto.Keccak256Hash(data) != parsedLog.MessageDataHash {
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
