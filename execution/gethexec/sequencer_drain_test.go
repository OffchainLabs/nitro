// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	testBlockNumber = 100
	testBaseFee     = 100
)

func makeTestQueueItem(t *testing.T, nonce uint64, gasFeeCap int64) (txQueueItem, chan error) {
	t.Helper()
	tx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		GasFeeCap: big.NewInt(gasFeeCap),
		GasTipCap: big.NewInt(0),
		Gas:       21000,
	})
	resultChan := make(chan error, 1)
	item := newRegularTxQueueItem(context.Background(), tx, nil, resultChan, false, 0)
	return item, resultChan
}

// makeTestSequencerQueues returns a minimal Sequencer with buffered queues for drain tests.
func makeTestSequencerQueues(queueSize int) *Sequencer {
	return &Sequencer{
		txQueue:                           make(chan txQueueItem, queueSize),
		timeboostAuctionResolutionTxQueue: make(chan txQueueItem, queueSize),
	}
}

func TestDrainQueueItemsEmptyQueues(t *testing.T) {
	s := makeTestSequencerQueues(1)
	items := s.drainQueueItems()
	if len(items) != 0 {
		t.Errorf("drained %d items from empty queues, want 0", len(items))
	}
}

func TestDrainQueueItemsPriorityOrder(t *testing.T) {
	s := makeTestSequencerQueues(2)

	const feeCap = 0 // fee validation is not exercised in this test

	// Nonces identify the expected drain order: auction resolution, then retry, then submitted.
	for _, nonce := range []uint64{4, 5} {
		item, _ := makeTestQueueItem(t, nonce, feeCap)
		s.txQueue <- item
	}
	for _, nonce := range []uint64{2, 3} {
		item, _ := makeTestQueueItem(t, nonce, feeCap)
		s.txRetryQueue.Push(item)
	}
	for _, nonce := range []uint64{0, 1} {
		item, _ := makeTestQueueItem(t, nonce, feeCap)
		s.timeboostAuctionResolutionTxQueue <- item
	}

	items := s.drainQueueItems()
	if len(items) != 6 {
		t.Fatalf("drained %d items, want 6", len(items))
	}
	for i, item := range items {
		if item.tx.Nonce() != uint64(i) { // #nosec G115
			t.Errorf("items[%d] has nonce %d, want %d", i, item.tx.Nonce(), i)
		}
	}
	if len(s.txQueue) != 0 || s.txRetryQueue.Len() != 0 || len(s.timeboostAuctionResolutionTxQueue) != 0 {
		t.Error("queues should be empty after draining")
	}
}

func TestDrainQueueItemsConcurrent(t *testing.T) {
	const itemsPerQueue = 100
	s := makeTestSequencerQueues(itemsPerQueue)

	for nonce := range uint64(itemsPerQueue) {
		item, _ := makeTestQueueItem(t, nonce, 0)
		s.txQueue <- item
		s.txRetryQueue.Push(item)
		s.timeboostAuctionResolutionTxQueue <- item
	}

	const drainers = 4
	results := make(chan []txQueueItem, drainers)
	var wg sync.WaitGroup
	for range drainers {
		wg.Go(func() {
			results <- s.drainQueueItems()
		})
	}
	wg.Wait()
	close(results)

	total := 0
	for items := range results {
		for _, item := range items {
			if item.tx == nil {
				t.Error("drained an empty queue item")
			}
		}
		total += len(items)
	}
	if total != 3*itemsPerQueue {
		t.Errorf("drained %d items total, want %d", total, 3*itemsPerQueue)
	}
	if len(s.txQueue) != 0 || s.txRetryQueue.Len() != 0 || len(s.timeboostAuctionResolutionTxQueue) != 0 {
		t.Error("queues should be empty after draining")
	}
}

func TestValidateQueueItemAcceptsValidItem(t *testing.T) {
	config := DefaultSequencerConfig
	item, _ := makeTestQueueItem(t, 0, testBaseFee)
	header := &types.Header{Number: big.NewInt(testBlockNumber), BaseFee: big.NewInt(testBaseFee)}

	if err := validateQueueItem(&config, header, item); err != nil {
		t.Errorf("validateQueueItem() = %v, want nil", err)
	}
}

func TestValidateQueueItemRejectsCanceledContext(t *testing.T) {
	config := DefaultSequencerConfig
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	item, _ := makeTestQueueItem(t, 0, testBaseFee)
	item.ctx = canceledCtx
	header := &types.Header{Number: big.NewInt(testBlockNumber), BaseFee: big.NewInt(testBaseFee)}

	if err := validateQueueItem(&config, header, item); !errors.Is(err, context.Canceled) {
		t.Errorf("validateQueueItem() = %v, want %v", err, context.Canceled)
	}
}

func TestValidateQueueItemRejectsOversizedTx(t *testing.T) {
	config := DefaultSequencerConfig
	item, _ := makeTestQueueItem(t, 0, testBaseFee)
	item.txSize = config.MaxTxDataSize + 1
	header := &types.Header{Number: big.NewInt(testBlockNumber), BaseFee: big.NewInt(testBaseFee)}

	if err := validateQueueItem(&config, header, item); !errors.Is(err, txpool.ErrOversizedData) {
		t.Errorf("validateQueueItem() = %v, want %v", err, txpool.ErrOversizedData)
	}
}

func TestValidateQueueItemRejectsExpiredTimeboostedTx(t *testing.T) {
	config := DefaultSequencerConfig
	config.Timeboost.QueueTimeoutInBlocks = 5
	item, _ := makeTestQueueItem(t, 0, testBaseFee)
	item.isTimeboosted = true
	item.blockStamp = 1 // testBlockNumber >= 1 + QueueTimeoutInBlocks
	header := &types.Header{Number: big.NewInt(testBlockNumber), BaseFee: big.NewInt(testBaseFee)}

	err := validateQueueItem(&config, header, item)
	if err == nil || !strings.Contains(err.Error(), "block based timeout") {
		t.Errorf("validateQueueItem() = %v, want block-age expiry error", err)
	}
}

func TestValidateQueueItemAcceptsUnexpiredTimeboostedTx(t *testing.T) {
	config := DefaultSequencerConfig
	config.Timeboost.QueueTimeoutInBlocks = 5
	item, _ := makeTestQueueItem(t, 0, testBaseFee)
	item.isTimeboosted = true
	item.blockStamp = testBlockNumber - 1 // testBlockNumber < blockStamp + QueueTimeoutInBlocks
	header := &types.Header{Number: big.NewInt(testBlockNumber), BaseFee: big.NewInt(testBaseFee)}

	if err := validateQueueItem(&config, header, item); err != nil {
		t.Errorf("validateQueueItem() = %v, want nil", err)
	}
}

func TestValidateQueueItemAcceptsTimeboostedTxWithoutBlockStamp(t *testing.T) {
	config := DefaultSequencerConfig
	config.Timeboost.QueueTimeoutInBlocks = 5
	item, _ := makeTestQueueItem(t, 0, testBaseFee)
	item.isTimeboosted = true // blockStamp stays 0, which exempts the item from expiry
	header := &types.Header{Number: big.NewInt(testBlockNumber), BaseFee: big.NewInt(testBaseFee)}

	if err := validateQueueItem(&config, header, item); err != nil {
		t.Errorf("validateQueueItem() = %v, want nil", err)
	}
}

func TestValidateQueueItemRejectsFeeCapBelowBasefee(t *testing.T) {
	config := DefaultSequencerConfig
	item, _ := makeTestQueueItem(t, 0, testBaseFee-1)
	header := &types.Header{Number: big.NewInt(testBlockNumber), BaseFee: big.NewInt(testBaseFee)}

	if err := validateQueueItem(&config, header, item); !errors.Is(err, core.ErrFeeCapTooLow) {
		t.Errorf("validateQueueItem() = %v, want %v", err, core.ErrFeeCapTooLow)
	}
}

func TestDrainAndValidateQueueItemsReturnsResultOnRejection(t *testing.T) {
	config := DefaultSequencerConfig
	rejectedItem, rejectedResultChan := makeTestQueueItem(t, 0, testBaseFee)
	rejectedItem.txSize = config.MaxTxDataSize + 1
	validItem, validResultChan := makeTestQueueItem(t, 1, testBaseFee)
	s := makeTestSequencerQueues(2)
	s.txQueue <- rejectedItem
	s.txQueue <- validItem
	header := &types.Header{Number: big.NewInt(testBlockNumber), BaseFee: big.NewInt(testBaseFee)}

	items := s.drainAndValidateQueueItems(&config, header)

	if len(items) != 1 || items[0].tx.Nonce() != 1 {
		t.Fatalf("drained items = %v, want only the valid item", items)
	}
	select {
	case err := <-rejectedResultChan:
		if !errors.Is(err, txpool.ErrOversizedData) {
			t.Errorf("rejected item got %v, want %v", err, txpool.ErrOversizedData)
		}
	default:
		t.Error("rejected item never received a result")
	}
	select {
	case err := <-validResultChan:
		t.Errorf("valid item got early result %v", err)
	default:
	}
}
