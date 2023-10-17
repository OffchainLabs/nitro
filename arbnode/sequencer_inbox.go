// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbutil"

	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

var sequencerBridgeABI *abi.ABI
var batchDeliveredID common.Hash
var addSequencerL2BatchFromOriginCallABI abi.Method
var sequencerBatchDataABI abi.Event

const sequencerBatchDataEvent = "SequencerBatchData"

type batchDataLocation uint8

const (
	batchDataTxInput batchDataLocation = iota
	batchDataSeparateEvent
	batchDataNone
)

func init() {
	var err error
	sequencerBridgeABI, err = bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	batchDeliveredID = sequencerBridgeABI.Events["SequencerBatchDelivered"].ID
	sequencerBatchDataABI = sequencerBridgeABI.Events[sequencerBatchDataEvent]
	addSequencerL2BatchFromOriginCallABI = sequencerBridgeABI.Methods["addSequencerL2BatchFromOrigin"]
}

type SequencerInbox struct {
	con       *bridgegen.SequencerInbox
	address   common.Address
	fromBlock int64
	client    arbutil.L1Interface
}

func NewSequencerInbox(client arbutil.L1Interface, addr common.Address, fromBlock int64) (*SequencerInbox, error) {
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
	if blockNumber.IsInt64() && blockNumber.Int64() < i.fromBlock {
		return 0, nil
	}
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
	acc, err := i.con.InboxAccs(opts, new(big.Int).SetUint64(sequenceNumber))
	return acc, err
}

type SequencerInboxBatch struct {
	BlockHash              common.Hash
	ParentChainBlockNumber uint64
	SequenceNumber         uint64
	BeforeInboxAcc         common.Hash
	AfterInboxAcc          common.Hash
	AfterDelayedAcc        common.Hash
	AfterDelayedCount      uint64
	TimeBounds             bridgegen.ISequencerInboxTimeBounds
	rawLog                 types.Log
	dataLocation           batchDataLocation
	bridgeAddress          common.Address
	serialized             []byte // nil if serialization isn't cached yet
}

func (m *SequencerInboxBatch) getSequencerData(ctx context.Context, client arbutil.L1Interface) ([]byte, error) {
	switch m.dataLocation {
	case batchDataTxInput:
		data, err := arbutil.GetLogEmitterTxData(ctx, client, m.rawLog)
		if err != nil {
			return nil, err
		}
		args := make(map[string]interface{})
		err = addSequencerL2BatchFromOriginCallABI.Inputs.UnpackIntoMap(args, data[4:])
		if err != nil {
			return nil, err
		}
		return args["data"].([]byte), nil
	case batchDataSeparateEvent:
		var numberAsHash common.Hash
		binary.BigEndian.PutUint64(numberAsHash[(32-8):], m.SequenceNumber)
		query := ethereum.FilterQuery{
			BlockHash: &m.BlockHash,
			Addresses: []common.Address{m.bridgeAddress},
			Topics:    [][]common.Hash{{sequencerBatchDataABI.ID}, {numberAsHash}},
		}
		logs, err := client.FilterLogs(ctx, query)
		if err != nil {
			return nil, err
		}
		if len(logs) == 0 {
			return nil, errors.New("expected to find sequencer batch data")
		}
		if len(logs) > 1 {
			return nil, errors.New("expected to find only one matching sequencer batch data")
		}
		event := new(bridgegen.SequencerInboxSequencerBatchData)
		err = sequencerBridgeABI.UnpackIntoInterface(event, sequencerBatchDataEvent, logs[0].Data)
		if err != nil {
			return nil, err
		}
		return event.Data, nil
	case batchDataNone:
		// No data when in a force inclusion batch
		return nil, nil
	default:
		return nil, fmt.Errorf("batch has invalid data location %v", m.dataLocation)
	}
}

func (m *SequencerInboxBatch) Serialize(ctx context.Context, client arbutil.L1Interface) ([]byte, error) {
	if m.serialized != nil {
		return m.serialized, nil
	}

	var fullData []byte

	// Serialize the header
	headerVals := []uint64{
		m.TimeBounds.MinTimestamp,
		m.TimeBounds.MaxTimestamp,
		m.TimeBounds.MinBlockNumber,
		m.TimeBounds.MaxBlockNumber,
		m.AfterDelayedCount,
	}
	for _, bound := range headerVals {
		var intData [8]byte
		binary.BigEndian.PutUint64(intData[:], bound)
		fullData = append(fullData, intData[:]...)
	}

	// Append the batch data
	data, err := m.getSequencerData(ctx, client)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Full data: %#x\n", data)
	fullData = append(fullData, data...)

	m.serialized = fullData
	return fullData, nil
}

func (i *SequencerInbox) LookupBatchesInRange(ctx context.Context, from, to *big.Int) ([]*SequencerInboxBatch, error) {
	query := ethereum.FilterQuery{
		FromBlock: from,
		ToBlock:   to,
		Addresses: []common.Address{i.address},
		Topics:    [][]common.Hash{{batchDeliveredID}},
	}
	logs, err := i.client.FilterLogs(ctx, query)
	if err != nil {
		return nil, err
	}
	messages := make([]*SequencerInboxBatch, 0, len(logs))
	var lastSeqNum *uint64
	for _, log := range logs {
		if log.Topics[0] != batchDeliveredID {
			return nil, errors.New("unexpected log selector")
		}
		parsedLog, err := i.con.ParseSequencerBatchDelivered(log)
		if err != nil {
			return nil, err
		}
		if !parsedLog.BatchSequenceNumber.IsUint64() {
			return nil, errors.New("sequencer inbox event has non-uint64 sequence number")
		}
		if !parsedLog.AfterDelayedMessagesRead.IsUint64() {
			return nil, errors.New("sequencer inbox event has non-uint64 delayed messages read")
		}

		seqNum := parsedLog.BatchSequenceNumber.Uint64()
		if lastSeqNum != nil {
			if seqNum != *lastSeqNum+1 {
				return nil, fmt.Errorf("sequencer batches out of order; after batch %v got batch %v", lastSeqNum, seqNum)
			}
		}
		lastSeqNum = &seqNum
		batch := &SequencerInboxBatch{
			BlockHash:              log.BlockHash,
			ParentChainBlockNumber: log.BlockNumber,
			SequenceNumber:         seqNum,
			BeforeInboxAcc:         parsedLog.BeforeAcc,
			AfterInboxAcc:          parsedLog.AfterAcc,
			AfterDelayedAcc:        parsedLog.DelayedAcc,
			AfterDelayedCount:      parsedLog.AfterDelayedMessagesRead.Uint64(),
			rawLog:                 log,
			TimeBounds:             parsedLog.TimeBounds,
			dataLocation:           batchDataLocation(parsedLog.DataLocation),
			bridgeAddress:          log.Address,
		}
		messages = append(messages, batch)
	}
	return messages, nil
}
