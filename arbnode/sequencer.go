//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"context"
	"fmt"
	"math/big"
	"sync"
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
	"github.com/offchainlabs/arbstate/arbutil"
	"github.com/offchainlabs/arbstate/util"
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

func (i *txQueueItem) returnResult(err error) {
	i.resultChan <- err
	close(i.resultChan)
}

type Sequencer struct {
	util.StopWaiter

	txStreamer    *TransactionStreamer
	txQueue       chan txQueueItem
	l1Client      arbutil.L1Interface
	l1BlockNumber uint64

	forwarderMutex sync.Mutex
	forwarder      *TxForwarder
}

func NewSequencer(txStreamer *TransactionStreamer, l1Client arbutil.L1Interface) (*Sequencer, error) {
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
	select {
	case res := <-resultChan:
		return res
	case <-ctx.Done():
		return ctx.Err()
	}
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
	agg, err := state.L1PricingState().ReimbursableAggregatorForSender(sender)
	if err != nil {
		return err
	}
	if agg == nil || *agg != l1pricing.SequencerAddress {
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

func (s *Sequencer) ForwardTarget() string {
	s.forwarderMutex.Lock()
	defer s.forwarderMutex.Unlock()
	if s.forwarder == nil {
		return ""
	}
	return s.forwarder.target
}

func (s *Sequencer) ForwardTo(url string) {
	s.forwarderMutex.Lock()
	defer s.forwarderMutex.Unlock()
	s.forwarder, _ = NewForwarder(url)
	s.forwarder.Initialize(s.GetContext())
}

func (s *Sequencer) DontForward() {
	s.forwarderMutex.Lock()
	defer s.forwarderMutex.Unlock()
	s.forwarder = nil
}

func (s *Sequencer) forwardIfSet(queueItems []txQueueItem) bool {
	s.forwarderMutex.Lock()
	defer s.forwarderMutex.Unlock()
	if s.forwarder == nil {
		return false
	}
	for _, item := range queueItems {
		item.resultChan <- s.forwarder.PublishTransaction(item.ctx, item.tx)
	}
	return true
}

func (s *Sequencer) sequenceTransactions(ctx context.Context) {
	timestamp := common.BigToHash(new(big.Int).SetInt64(time.Now().Unix()))
	l1Block := atomic.LoadUint64(&s.l1BlockNumber)
	for s.l1Client != nil && l1Block == 0 {
		log.Error("cannot sequence: unknown L1 block")
		time.Sleep(time.Second)
		l1Block = atomic.LoadUint64(&s.l1BlockNumber)
	}

	var txes types.Transactions
	var queueItems []txQueueItem
	var totalBatchSize int
	for {
		var queueItem txQueueItem
		if len(txes) == 0 {
			select {
			case queueItem = <-s.txQueue:
			case <-ctx.Done():
				return
			}
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
			queueItem.returnResult(err)
			continue
		}
		txBytes, err := queueItem.tx.MarshalBinary()
		if err != nil {
			queueItem.returnResult(err)
			continue
		}
		if len(txBytes) > int(maxTxDataSize) {
			// This tx is too large
			queueItem.returnResult(core.ErrOversizedData)
			continue
		}
		if totalBatchSize+len(txBytes) > int(maxTxDataSize) {
			// This tx would be too large to add to this batch.
			// Attempt to put it back in the queue, but error if the queue is full.
			// Then, end the batch here.
			select {
			case s.txQueue <- queueItem:
			default:
				queueItem.returnResult(core.ErrOversizedData)
			}
			break
		}
		totalBatchSize += len(txBytes)
		txes = append(txes, queueItem.tx)
		queueItems = append(queueItems, queueItem)
	}

	if s.forwardIfSet(queueItems) {
		return
	}

	header := &arbos.L1IncomingMessageHeader{
		Kind:        arbos.L1MessageType_L2Message,
		Poster:      l1pricing.SequencerAddress,
		BlockNumber: common.BigToHash(new(big.Int).SetUint64(l1Block)),
		Timestamp:   timestamp,
		RequestId:   common.Hash{},
		BaseFeeL1:   common.Hash{},
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
	if errors.Is(err, ErrNotMainSequencer) {
		// we changed roles
		// forward if we have where to
		if s.forwardIfSet(queueItems) {
			return
		}
		// try to add back to queue otherwise
		for _, item := range queueItems {
			select {
			case s.txQueue <- item:
			default:
				item.resultChan <- errors.New("queue full")
			}
		}
		return
	}
	if err != nil {
		log.Error("error sequencing transactions", "err", err)
		for _, queueItem := range queueItems {
			queueItem.returnResult(err)
		}
		return
	}

	for i, err := range hooks.TxErrors {
		queueItem := queueItems[i]
		if errors.Is(err, core.ErrGasLimit) {
			// There's not enough gas left in the block for this tx.
			// Attempt to re-queue the transaction.
			// If the queue is full, fall through to returning an error.
			select {
			case s.txQueue <- queueItem:
				continue
			default:
			}
		}
		queueItem.returnResult(err)
	}
}

func (s *Sequencer) Start(ctxIn context.Context) error {
	s.StopWaiter.Start(ctxIn)
	if s.l1Client != nil {
		initialBlockNr := atomic.LoadUint64(&s.l1BlockNumber)
		if initialBlockNr == 0 {
			return errors.New("sequencer not initialized")
		}

		headerChan, cancel := arbutil.HeaderSubscribeWithRetry(s.GetContext(), s.l1Client)

		s.LaunchThread(func(ctx context.Context) {
			defer cancel()
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
		})
	}

	s.CallIteratively(func(ctx context.Context) time.Duration {
		s.sequenceTransactions(ctx)
		return minBlockInterval
	})

	return nil
}
