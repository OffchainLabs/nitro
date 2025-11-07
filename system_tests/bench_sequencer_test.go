//go:build benchsequencer

package arbtest

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

func TestExperimentalBenchSequencer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	// we don't want any txes sent during NodeBuilder.Build as they will hang and timeout due to no blocks beeing created automatically
	builder = builder.DontSendL2SetupTxes()
	builder.execConfig.Dangerous.BenchSequencer.Enable = true

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
		var txQueueLen int
		err := rpcClient.CallContext(ctx, &txQueueLen, "benchseq_txQueueLength", false)
		Require(t, err)
		timeout := time.After(5 * time.Second)
		for txQueueLen < i+1 {
			err := rpcClient.CallContext(ctx, &txQueueLen, "benchseq_txQueueLength", false)
			Require(t, err)
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
		Fatal(t, "block have been created even though benchseq_createBlock hasn't been called")
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
		Fatal(t, "benchseq_txQueueLenght reported non empty queue, want: 0, have:", txQueueLen)
	}

	txSendersWg.Wait()
	for _, tx := range txes {
		builder.L2.EnsureTxSucceeded(tx)
	}

	block, err = builder.L2.Client.BlockNumber(ctx)
	Require(t, err)
	timeout := time.After(5 * time.Second)
	for block != startBlock+1 {
		select {
		case <-timeout:
			Fatal(t, "timeout exceeded while waiting for new block")
		case <-time.After(20 * time.Millisecond):
		}
		block, err = builder.L2.Client.BlockNumber(ctx)
		Require(t, err)
	}
}
