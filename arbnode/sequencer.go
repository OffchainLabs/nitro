//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"context"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbos/arbosState"
	"github.com/offchainlabs/arbstate/arbos/l1pricing"
	"github.com/pkg/errors"
)

// TODO: make these configurable
const minBlockInterval time.Duration = time.Millisecond * 100
const maxRevertGasReject uint64 = params.TxGas + 10000

// 95% of the SequencerInbox limit, leaving ~5KB for headers and such
const maxTxDataSize uint64 = 112065

type txQueueItem struct {
	tx         *types.Transaction
	resultChan chan<- error
	ctx        context.Context
}

type Sequencer struct {
	txStreamer    *TransactionStreamer
	txQueue       chan txQueueItem
	l1Client      L1Interface
	l1BlockNumber uint64
}

func NewSequencer(txStreamer *TransactionStreamer, l1Client L1Interface) (*Sequencer, error) {
	return &Sequencer{
		txStreamer:    txStreamer,
		txQueue:       make(chan txQueueItem, 128),
		l1Client:      l1Client,
		l1BlockNumber: 0,
	}, nil
}

func (s *Sequencer) PublishTransaction(ctx context.Context, tx *types.Transaction) error {
	resultChan := make(chan error, 1)
	s.txQueue <- txQueueItem{
		tx,
		resultChan,
		ctx,
	}
	return <-resultChan
}

func (s *Sequencer) Initialize(ctx context.Context) error {
	if s.l1Client == nil {
		return nil
	}

	block, err := s.l1Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}
	atomic.StoreUint64(&s.l1BlockNumber, block.Number.Uint64())
	return nil
}

func preTxFilter(state *arbosState.ArbosState, tx *types.Transaction, sender common.Address) error {
	agg, _, err := state.L1PricingState().PreferredAggregator(sender)
	if err != nil {
		return err
	}
	if agg != l1pricing.SequencerAddress {
		return errors.New("transaction sender's preferred aggregator is not the sequencer")
	}
	return nil
}

func postTxFilter(state *arbosState.ArbosState, tx *types.Transaction, sender common.Address, dataGas uint64, receipt *types.Receipt) error {
	if receipt.Status == types.ReceiptStatusFailed && receipt.GasUsed > dataGas && receipt.GasUsed-dataGas <= maxRevertGasReject {
		return vm.ErrExecutionReverted
	}
	return nil
}

func (s *Sequencer) sequenceTransactions() {
	timestamp := common.BigToHash(new(big.Int).SetInt64(time.Now().Unix()))
	l1Block := atomic.LoadUint64(&s.l1BlockNumber)
	for s.l1Client != nil && l1Block == 0 {
		log.Error("cannot sequence: unknown L1 block")
		time.Sleep(time.Second)
		l1Block = atomic.LoadUint64(&s.l1BlockNumber)
	}

	var txes types.Transactions
	var resultChans []chan<- error
	var totalBatchSize int
	for {
		var queueItem txQueueItem
		if len(txes) == 0 {
			queueItem = <-s.txQueue
		} else {
			done := false
			select {
			case queueItem = <-s.txQueue:
			default:
				done = true
			}
			if done {
				break
			}
		}
		err := queueItem.ctx.Err()
		if err != nil {
			queueItem.resultChan <- err
			continue
		}
		txBytes, err := queueItem.tx.MarshalBinary()
		if err != nil {
			queueItem.resultChan <- err
			continue
		}
		if len(txBytes) > int(maxTxDataSize) {
			// This tx is too large
			queueItem.resultChan <- core.ErrOversizedData
			continue
		}
		if totalBatchSize+len(txBytes) > int(maxTxDataSize) {
			// This tx would be too large to add to this batch.
			// Attempt to put it back in the queue, but error if the queue is full.
			// Then, end the batch here.
			select {
			case s.txQueue <- queueItem:
			default:
				queueItem.resultChan <- core.ErrOversizedData
			}
			break
		}
		totalBatchSize += len(txBytes)
		txes = append(txes, queueItem.tx)
		resultChans = append(resultChans, queueItem.resultChan)
	}

	header := &arbos.L1IncomingMessageHeader{
		Kind:        arbos.L1MessageType_L2Message,
		Poster:      l1pricing.SequencerAddress,
		BlockNumber: common.BigToHash(new(big.Int).SetUint64(l1Block)),
		Timestamp:   timestamp,
		RequestId:   common.Hash{},
		GasPriceL1:  common.Hash{},
	}

	hooks := &arbos.SequencingHooks{
		PreTxFilter:    preTxFilter,
		PostTxFilter:   postTxFilter,
		RequireDataGas: true,
		TxErrors:       []error{},
	}
	err := s.txStreamer.SequenceTransactions(header, txes, hooks)
	if err == nil && len(hooks.TxErrors) != len(txes) {
		err = fmt.Errorf("unexpected number of error results: %v vs number of txes %v", len(hooks.TxErrors), len(txes))
	}
	if err != nil {
		log.Error("error sequencing transactions", "err", err)
		for _, resultChan := range resultChans {
			resultChan <- err
		}
		return
	}

	for i, err := range hooks.TxErrors {
		resultChans[i] <- err
	}
}

func (s *Sequencer) Start(ctx context.Context) error {
	if s.l1Client != nil {
		initialBlockNr := atomic.LoadUint64(&s.l1BlockNumber)
		if initialBlockNr == 0 {
			return errors.New("sequencer not initialized")
		}

		headerChan, cancel := HeaderSubscribeWithRetry(ctx, s.l1Client)
		defer cancel()

		go (func() {
			for {
				select {
				case header, ok := <-headerChan:
					if !ok {
						return
					}
					atomic.StoreUint64(&s.l1BlockNumber, header.Number.Uint64())
				case <-ctx.Done():
					return
				}
			}
		})()
	}

	go (func() {
		for {
			s.sequenceTransactions()
			time.Sleep(minBlockInterval)
		}
	})()

	return nil
}
