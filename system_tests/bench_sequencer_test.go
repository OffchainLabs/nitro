// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build benchmarking-sequencer

package arbtest

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

func TestBenchmarkingSequencer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	// We don't want any txes sent during NodeBuilder.Build as they will hang and timeout due to no blocks being created automatically.
	builder = builder.WithTakeOwnership(false)
	builder.execConfig.Dangerous.BenchmarkingSequencer.Enable = true

	cleanup := builder.Build(t)
	defer cleanup()

	startBlock, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)

	rpcClient := builder.L2.Client.Client()
	var txSendersWg sync.WaitGroup
	var txes types.Transactions
	for i := 0; i < 5; i++ {
		// send the transaction in separate thread as the rpc call will wait for it to be accepted by sequencer
		tx := builder.L2Info.PrepareTx("Owner", "Owner", builder.L2Info.TransferGas, big.NewInt(1), nil)
		txes = append(txes, tx)
		txSendersWg.Add(1)
		go func() {
			defer txSendersWg.Done()
			err := builder.L2.Client.SendTransaction(ctx, tx)
			Require(t, err)
		}()

		// wait for the transaction to be enqueued
		timeout := time.After(5 * time.Second)
		for {
			var txQueueLen int
			err := rpcClient.CallContext(ctx, &txQueueLen, "benchseq_txQueueLength", false)
			Require(t, err)
			if txQueueLen >= i+1 {
				break
			}
			select {
			case <-timeout:
				Fatal(t, "timeout exceeded while waiting for tx queue to grow")
			case <-time.After(10 * time.Millisecond):
			}
		}
	}

	block, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)
	if block != startBlock {
		Fatal(t, "block has been created even though benchseq_createBlock hasn't been called")
	}

	var blockCreated bool
	// create block with all of the transactions (they should fit)
	err = rpcClient.CallContext(ctx, &blockCreated, "benchseq_createBlock")
	Require(t, err)
	if !blockCreated {
		Fatal(t, "block should have been created")
	}
	// check that tx queue is empty
	var txQueueLen int
	err = rpcClient.CallContext(ctx, &txQueueLen, "benchseq_txQueueLength", false)
	Require(t, err)
	if txQueueLen != 0 {
		Fatal(t, "benchseq_txQueueLength reported non empty queue, want: 0, have:", txQueueLen)
	}

	txSendersWg.Wait()
	for _, tx := range txes {
		_, err := builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}

	timeout := time.After(5 * time.Second)
	for {
		block, err := builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
		if block >= startBlock+1 {
			break
		}
		select {
		case <-timeout:
			Fatal(t, "timeout exceeded while waiting for new block")
		case <-time.After(20 * time.Millisecond):
		}
	}
}
