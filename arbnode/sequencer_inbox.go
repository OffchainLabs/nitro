//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"

	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
)

var batchDeliveredID common.Hash
var batchDeliveredFromOriginID common.Hash
var addSequencerL2BatchFromOriginCallABI abi.Method

func init() {
	parsedSequencerBridgeABI, err := abi.JSON(strings.NewReader(bridgegen.SequencerInboxABI))
	if err != nil {
		panic(err)
	}
	batchDeliveredID = parsedSequencerBridgeABI.Events["SequencerBatchDelivered"].ID
	batchDeliveredFromOriginID = parsedSequencerBridgeABI.Events["SequencerBatchDeliveredFromOrigin"].ID
	addSequencerL2BatchFromOriginCallABI = parsedSequencerBridgeABI.Methods["addSequencerL2BatchFromOrigin"]
}

type SequencerInbox struct {
	con       *bridgegen.SequencerInbox
	address   common.Address
	fromBlock int64
	client    *ethclient.Client
}

func NewSequencerInbox(client *ethclient.Client, addr common.Address, fromBlock int64) (*SequencerInbox, error) {
	con, err := bridgegen.NewSequencerInbox(addr, client)
	if err != nil {
		return nil, err
	}

	return &SequencerInbox{
		con:       con,
		address:   addr,
		fromBlock: fromBlock,
		client:    client,
	}, nil
}

func (i *SequencerInbox) GetBatchCount(ctx context.Context, blockNumber *big.Int) (uint64, error) {
	opts := &bind.CallOpts{
		Context:     ctx,
		BlockNumber: blockNumber,
	}
	count, err := i.con.BatchCount(opts)
	if err != nil {
		return 0, err
	}
	if !count.IsUint64() {
		return 0, errors.New("sequencer inbox returned non-uint64 batch count")
	}
	return count.Uint64(), nil
}

func (i *SequencerInbox) GetAccumulator(ctx context.Context, sequenceNumber uint64, blockNumber *big.Int) (common.Hash, error) {
	opts := &bind.CallOpts{
		Context:     ctx,
		BlockNumber: blockNumber,
	}
	return i.con.InboxAccs(opts, new(big.Int).SetUint64(sequenceNumber))
}

type SequencerInboxBatch struct {
	BlockHash         common.Hash
	SequenceNumber    uint64
	BeforeInboxAcc    common.Hash
	AfterInboxAcc     common.Hash
	AfterDelayedAcc   common.Hash
	AfterDelayedCount uint64
	dataIfAvailable   *[]byte
	txIndexInBlock    uint
}

func (m *SequencerInboxBatch) GetData(ctx context.Context, client *ethclient.Client) ([]byte, error) {
	if m.dataIfAvailable != nil {
		return *m.dataIfAvailable, nil
	}
	tx, err := client.TransactionInBlock(ctx, m.BlockHash, m.txIndexInBlock)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	args := make(map[string]interface{})
	err = addSequencerL2BatchFromOriginCallABI.Inputs.UnpackIntoMap(args, tx.Data()[4:])
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return args["data"].([]byte), nil
}

func (i *SequencerInbox) LookupBatchesInRange(ctx context.Context, from, to *big.Int) ([]*SequencerInboxBatch, error) {
	query := ethereum.FilterQuery{
		BlockHash: nil,
		FromBlock: from,
		ToBlock:   to,
		Addresses: []common.Address{i.address},
		Topics:    [][]common.Hash{{batchDeliveredID, batchDeliveredFromOriginID}},
	}
	logs, err := i.client.FilterLogs(ctx, query)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	messages := make([]*SequencerInboxBatch, 0, len(logs))
	for _, log := range logs {
		if log.Topics[0] == batchDeliveredID {
			parsedLog, err := i.con.ParseSequencerBatchDelivered(log)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			if !parsedLog.BatchSequenceNumber.IsUint64() {
				return nil, errors.New("sequencer inbox event has non-uint64 sequence number")
			}
			if !parsedLog.AfterDelayedMessagesRead.IsUint64() {
				return nil, errors.New("sequencer inbox event has non-uint64 delayed messages read")
			}
			messages = append(messages, &SequencerInboxBatch{
				BlockHash:         log.BlockHash,
				SequenceNumber:    parsedLog.BatchSequenceNumber.Uint64(),
				BeforeInboxAcc:    parsedLog.BeforeAcc,
				AfterInboxAcc:     parsedLog.AfterAcc,
				AfterDelayedAcc:   parsedLog.DelayedAcc,
				AfterDelayedCount: parsedLog.AfterDelayedMessagesRead.Uint64(),
				dataIfAvailable:   &parsedLog.Data,
				txIndexInBlock:    log.TxIndex,
			})
		} else if log.Topics[0] == batchDeliveredFromOriginID {
			parsedLog, err := i.con.ParseSequencerBatchDeliveredFromOrigin(log)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			if !parsedLog.BatchSequenceNumber.IsUint64() {
				return nil, errors.New("sequencer inbox event has non-uint64 sequence number")
			}
			if !parsedLog.AfterDelayedMessagesRead.IsUint64() {
				return nil, errors.New("sequencer inbox event has non-uint64 delayed messages read")
			}
			messages = append(messages, &SequencerInboxBatch{
				BlockHash:         log.BlockHash,
				SequenceNumber:    parsedLog.BatchSequenceNumber.Uint64(),
				BeforeInboxAcc:    parsedLog.BeforeAcc,
				AfterInboxAcc:     parsedLog.AfterAcc,
				AfterDelayedAcc:   parsedLog.DelayedAcc,
				AfterDelayedCount: parsedLog.AfterDelayedMessagesRead.Uint64(),
				dataIfAvailable:   nil,
				txIndexInBlock:    log.TxIndex,
			})
		} else {
			return nil, errors.New("unexpected log selector")
		}
	}
	return messages, nil
}
